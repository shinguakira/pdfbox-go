package font

// Embeds a TrueType font as a CIDFontType2.
//
// Port of the package-private
// org.apache.pdfbox.pdmodel.font.PDCIDFontType2Embedder, which extends
// TrueTypeEmbedder.

import (
	"bytes"
	"fmt"
	"log/slog"
	"math"
	"sort"

	"github.com/shinguakira/pdfbox-go/go/fontbox/cff"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// embeddingDocument is what an embedder needs of a PDDocument.
//
// pdmodel/font cannot name PDDocument -- pdmodel imports font -- so the
// document arrives as the interface of what is used, which is this port's way
// with a dependency that only goes one way. PDDocument satisfies it.
type embeddingDocument interface {
	common.COSDocumentLike

	// Version is document.getVersion(), which buildToUnicodeCMap raises where
	// the ToUnicode CMap holds a surrogate pair.
	Version() float32

	// SetVersion is document.setVersion(float).
	SetVersion(newVersion float32)

	// RegisterTrueTypeFontForClosing keeps a font program open until the
	// document is closed, which a subsetting embedder needs because the subset
	// is not built until the document is saved.
	RegisterTrueTypeFontForClosing(font *ttf.TrueTypeFont)
}

// widthState is the three-state machine getWidths and getVerticalMetrics run.
//
// Port of the nested enum State.
type widthState int

const (
	stateFirst widthState = iota
	stateBracket
	stateSerial
)

// pdCIDFontType2Embedder embeds a TrueType font as a CIDFontType2.
type pdCIDFontType2Embedder struct {
	*trueTypeEmbedder

	document embeddingDocument
	dict     *cos.Dictionary
	cidFont  *cos.Dictionary
	vertical bool
}

// newPDCIDFontType2Embedder creates a new TrueType font embedder for the given
// TTF as a PDCIDFontType2.
func newPDCIDFontType2Embedder(document embeddingDocument, dict *cos.Dictionary,
	font *ttf.TrueTypeFont, embedSubset, vertical bool) (*pdCIDFontType2Embedder, error) {
	base, err := newTrueTypeEmbedder(document, dict, font, embedSubset)
	if err != nil {
		return nil, err
	}
	e := &pdCIDFontType2Embedder{
		trueTypeEmbedder: base,
		document:         document,
		dict:             dict,
		vertical:         vertical,
	}
	base.buildSubsetFromStream = e.buildSubset

	// parent Type 0 font
	dict.SetItem(cos.Subtype, cos.Type0)
	dict.SetName(cos.BaseFont, e.fontDescriptor.FontName())
	encoding := cos.IdentityH
	if vertical {
		encoding = cos.IdentityV
	}
	dict.SetItem(cos.Encoding, encoding) // CID = GID

	// descendant CIDFont
	cidFont, err := e.createCIDFont()
	if err != nil {
		return nil, err
	}
	e.cidFont = cidFont
	descendantFonts := cos.NewArray()
	descendantFonts.Add(cidFont)
	dict.SetItem(cos.DescendantFonts, descendantFonts)

	if !embedSubset {
		// build GID -> Unicode map
		if err := e.buildToUnicodeCMap(nil); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// buildSubset rebuilds a font subset.
func (e *pdCIDFontType2Embedder) buildSubset(ttfSubset []byte, tag string,
	gidToCid map[int]int) error {
	// build CID2GIDMap, because the content stream has been written with the
	// old GIDs
	cidToGid := map[int]int{}
	for newGID, oldGID := range gidToCid {
		cidToGid[oldGID] = newGID
	}

	// build unicode mapping before subsetting as the subsetted font won't have
	// a cmap
	if err := e.buildToUnicodeCMap(gidToCid); err != nil {
		return err
	}
	// build vertical metrics before subsetting as the subsetted font won't have
	// vhea, vmtx
	if e.vertical {
		if err := e.buildVerticalMetricsOfSubset(cidToGid); err != nil {
			return err
		}
	}
	// rebuild the relevant part of the font
	if err := e.buildFontFile2(bytes.NewReader(ttfSubset)); err != nil {
		return err
	}
	e.addNameTag(tag)
	if err := e.buildWidthsOfSubset(cidToGid); err != nil {
		return err
	}
	if err := e.buildCIDToGIDMap(cidToGid); err != nil {
		return err
	}
	return e.buildCIDSet(cidToGid)
}

// buildToUnicodeCMap writes the /ToUnicode CMap of the font.
func (e *pdCIDFontType2Embedder) buildToUnicodeCMap(newGIDToOldCID map[int]int) error {
	// PDFBOX-6210:
	// When several code points map to one glyph, prefer the one actually used
	// in the document (first occurrence wins) instead of
	// cmapLookup.getCharCodes(gid).get(0), which is the lowest code point and
	// often an unexpected compatibility character.
	inputCodePointByGID := map[int]int{}
	for _, codePoint := range e.SubsetCodePoints() {
		inputGid := e.cmapLookup.GetGlyphID(codePoint)
		if inputGid > 0 {
			if _, seen := inputCodePointByGID[inputGid]; !seen {
				inputCodePointByGID[inputGid] = codePoint
			}
		}
	}

	toUniWriter := newToUnicodeWriter()
	hasSurrogates := false
	profile, err := e.ttf.MaximumProfile()
	if err != nil {
		return err
	}
	for gid, max := 1, profile.NumGlyphs(); gid <= max; gid++ {
		// optional CID2GIDMap for subsetting
		var cid int
		if newGIDToOldCID != nil {
			old, ok := newGIDToOldCID[gid]
			if !ok {
				continue
			}
			cid = old
		} else {
			cid = gid
		}

		// skip composite glyph components that have no code point
		codes := e.cmapLookup.GetCharCodes(cid) // old GID -> Unicode
		// PDFBOX-6210: try to get the codepoint that is actually used in the
		// document instead of cmapLookup.getCharCodes(gid).get(0)
		inputCodePoint, haveInput := inputCodePointByGID[cid]
		if haveInput || len(codes) > 0 {
			// fall back to the cmap's first entry for glyphs with no recorded
			// input
			codePoint := inputCodePoint
			if !haveInput {
				codePoint = codes[0]
			}
			if len(codes) > 1 {
				slog.Debug("font: several codes map to one glyph",
					"codes", codes, "chosen", codePoint)
			}
			if codePoint > 0xFFFF {
				hasSurrogates = true
			}
			toUniWriter.add(cid, string(rune(codePoint)))
		}
	}

	var out bytes.Buffer
	if err := toUniWriter.writeTo(&out); err != nil {
		return err
	}

	stream, err := common.NewPDStreamOfInput(e.document, bytes.NewReader(out.Bytes()),
		cos.FlateDecode)
	if err != nil {
		return err
	}

	// surrogate code points, requires PDF 1.5
	if hasSurrogates && e.document.Version() < 1.5 {
		e.document.SetVersion(1.5)
	}

	e.dict.SetItem(cos.ToUnicode, stream.COSObject())
	return nil
}

// toCIDSystemInfo builds a /CIDSystemInfo dictionary.
func toCIDSystemInfo(registry, ordering string, supplement int) *cos.Dictionary {
	info := cos.NewDictionary()
	info.SetString(cos.Registry, registry)
	info.SetString(cos.Ordering, ordering)
	info.SetInt(cos.Supplement, supplement)
	return info
}

// createCIDFont builds the descendant CIDFont dictionary.
func (e *pdCIDFontType2Embedder) createCIDFont() (*cos.Dictionary, error) {
	cidFont := cos.NewDictionary()

	// Type, Subtype
	cidFont.SetItem(cos.Type, cos.Font)
	cidFont.SetItem(cos.Subtype, cos.CIDFontType2)

	// BaseFont
	cidFont.SetName(cos.BaseFont, e.fontDescriptor.FontName())

	// CIDSystemInfo
	cidFont.SetItem(cos.CIDSystemInfo, toCIDSystemInfo("Adobe", "Identity", 0))

	// FontDescriptor
	cidFont.SetItem(cos.FontDescriptor, e.fontDescriptor.COSObject())

	// the two builders below write into it
	e.cidFont = cidFont

	// W - widths
	if err := e.buildWidths(cidFont); err != nil {
		return nil, err
	}

	// Vertical metrics
	if e.vertical {
		if err := e.buildVerticalMetrics(cidFont); err != nil {
			return nil, err
		}
	}

	// `ttf instanceof OpenTypeFont`. A Go type assertion cannot say it:
	// OpenTypeFont embeds *TrueTypeFont rather than extending it, so the field
	// never holds one. AsOpenType is the port's standing answer, and
	// pdcidfonttype2.go asks the same question the same way.
	if otf := e.ttf.AsOpenType(); otf != nil && !e.NeedsSubset() {
		if err := e.checkForCidGidIdentity(otf); err != nil {
			return nil, err
		}
	}

	// CIDToGIDMap
	cidFont.SetItem(cos.CIDToGIDMap, cos.Identity)

	return cidFont, nil
}

// checkForCidGidIdentity is PDFBOX-6172: if somebody is using a not subsetted
// otf font, check whether cid == gid (subsetted will fail anyway).
func (e *pdCIDFontType2Embedder) checkForCidGidIdentity(otf *ttf.OpenTypeFont) error {
	// Java calls getCFF() unguarded, and it throws UnsupportedOperationException
	// -- unchecked -- for an OTF whose outlines are glyf rather than CFF. The
	// port panics there for the same reason.
	cffTable, err := otf.CFF()
	if err != nil {
		return err
	}
	if cffTable == nil {
		return nil
	}
	cidKeyed, isCIDKeyed := cffTable.Font().(*cff.CFFCIDFont)
	if !isCIDKeyed {
		return nil
	}
	charset := cidKeyed.Charset()
	if charset == nil {
		return nil
	}
	glyphCount, err := otf.NumberOfGlyphs()
	if err != nil {
		return err
	}
	for gid := 0; gid < glyphCount; gid++ {
		cid := charset.CIDForGID(gid)
		if gid != cid {
			// Java throws IllegalStateException, which is unchecked.
			panic(fmt.Sprintf("CID and GID not identical: CID %d != GID %d, "+
				"use a ttf font instead", cid, gid))
		}
	}
	return nil
}

// addNameTag prefixes the font's name with the subset tag.
func (e *pdCIDFontType2Embedder) addNameTag(tag string) {
	newName := tag + e.fontDescriptor.FontName()

	e.dict.SetName(cos.BaseFont, newName)
	e.fontDescriptor.SetFontName(newName)
	e.cidFont.SetName(cos.BaseFont, newName)
}

// sortedCIDs returns the CIDs of the map in order, which is what Java's
// TreeMap gives its callers.
func sortedCIDs(cidToGid map[int]int) []int {
	cids := make([]int, 0, len(cidToGid))
	for cid := range cidToGid {
		cids = append(cids, cid)
	}
	sort.Ints(cids)
	return cids
}

// buildCIDToGIDMap writes the /CIDToGIDMap stream.
func (e *pdCIDFontType2Embedder) buildCIDToGIDMap(cidToGid map[int]int) error {
	cids := sortedCIDs(cidToGid)
	cidMax := cids[len(cids)-1]
	buffer := make([]byte, cidMax*2+2)
	bi := 0
	for i := 0; i <= cidMax; i++ {
		if gid, ok := cidToGid[i]; ok {
			buffer[bi] = byte(gid >> 8 & 0xff)
			buffer[bi+1] = byte(gid & 0xff)
		}
		// else keep 0 initialization
		bi += 2
	}

	stream, err := common.NewPDStreamOfInput(e.document, bytes.NewReader(buffer),
		cos.FlateDecode)
	if err != nil {
		return err
	}
	e.cidFont.SetItem(cos.CIDToGIDMap, stream.COSObject())
	return nil
}

// buildCIDSet writes the /CIDSet entry PDF/A requires, which lists every CID in
// the font including those with no GID.
func (e *pdCIDFontType2Embedder) buildCIDSet(cidToGid map[int]int) error {
	cids := sortedCIDs(cidToGid)
	cidMax := cids[len(cids)-1]
	data := make([]byte, cidMax/8+1)
	for cid := 0; cid <= cidMax; cid++ {
		mask := 1 << (7 - cid%8)
		data[cid/8] |= byte(mask)
	}

	stream, err := common.NewPDStreamOfInput(e.document, bytes.NewReader(data),
		cos.FlateDecode)
	if err != nil {
		return err
	}
	e.fontDescriptor.SetCIDSet(stream)
	return nil
}

// buildWidthsOfSubset builds the widths with a custom CIDToGIDMap, for
// embedding a font subset.
//
// Port of the buildWidths(TreeMap) overload.
func (e *pdCIDFontType2Embedder) buildWidthsOfSubset(cidToGid map[int]int) error {
	scaling, err := e.unitsScaling()
	if err != nil {
		return err
	}
	hmtx, err := e.ttf.HorizontalMetrics()
	if err != nil {
		return err
	}

	widths := cos.NewArray()
	ws := cos.NewArray()
	prev := math.MinInt32
	// Use a sorted list to get an optimal width array
	for _, cid := range sortedCIDs(cidToGid) {
		gid := cidToGid[cid]
		width := int64(javaRound(float32(hmtx.AdvanceWidth(gid)) * scaling))
		if width == 1000 {
			// skip default width
			continue
		}
		// c [w1 w2 ... wn]
		if prev != cid-1 {
			ws = cos.NewArray()
			widths.Add(cos.GetInteger(int64(cid))) // c
			widths.Add(ws)
		}
		ws.Add(cos.GetInteger(width)) // wi
		prev = cid
	}
	e.cidFont.SetItem(cos.W, widths)
	return nil
}

// unitsScaling is `1000f / ttf.getHeader().getUnitsPerEm()`, which every metric
// builder starts with.
func (e *pdCIDFontType2Embedder) unitsScaling() (float32, error) {
	header, err := e.ttf.Header()
	if err != nil {
		return 0, err
	}
	return 1000 / float32(header.UnitsPerEm()), nil
}

// buildVerticalHeader writes /DW2 and reports whether the font has the vertical
// header to build the rest from.
func (e *pdCIDFontType2Embedder) buildVerticalHeader(cidFont *cos.Dictionary) (bool, error) {
	vhea, err := e.ttf.VerticalHeader()
	if err != nil {
		return false, err
	}
	if vhea == nil {
		slog.Warn("font: font to be subset is set to vertical, but has no 'vhea' table")
		return false, nil
	}

	scaling, err := e.unitsScaling()
	if err != nil {
		return false, err
	}

	v := int64(javaRound(float32(vhea.Ascender()) * scaling))
	w1 := int64(javaRound(float32(-vhea.AdvanceHeightMax()) * scaling))
	if v != 880 || w1 != -1000 {
		cosDw2 := cos.NewArray()
		cosDw2.Add(cos.GetInteger(v))
		cosDw2.Add(cos.GetInteger(w1))
		cidFont.SetItem(cos.DW2, cosDw2)
	}
	return true, nil
}

// buildVerticalMetricsOfSubset builds the vertical metrics with a custom
// CIDToGIDMap, for embedding a font subset.
//
// The "vhea" and "vmtx" tables that specify vertical metrics shall never be
// used by a conforming reader. The only way to specify vertical metrics in PDF
// shall be by means of the DW2 and W2 entries in a CIDFont dictionary.
func (e *pdCIDFontType2Embedder) buildVerticalMetricsOfSubset(cidToGid map[int]int) error {
	ok, err := e.buildVerticalHeader(e.cidFont)
	if err != nil || !ok {
		return err
	}

	scaling, err := e.unitsScaling()
	if err != nil {
		return err
	}
	vhea, err := e.ttf.VerticalHeader()
	if err != nil {
		return err
	}
	vmtx, err := e.ttf.VerticalMetrics()
	if err != nil {
		return err
	}
	glyf, err := e.ttf.Glyph()
	if err != nil {
		return err
	}
	hmtx, err := e.ttf.HorizontalMetrics()
	if err != nil {
		return err
	}

	vY := int64(javaRound(float32(vhea.Ascender()) * scaling))
	w1 := int64(javaRound(float32(-vhea.AdvanceHeightMax()) * scaling))

	heights := cos.NewArray()
	w2 := cos.NewArray()
	prev := math.MinInt32
	// Use a sorted list to get an optimal width array
	for _, cid := range sortedCIDs(cidToGid) {
		// Unlike buildWidths, we look up with cid (not gid) here because this
		// is the original TTF, not the rebuilt one.
		glyph, err := glyf.GetGlyph(cid)
		if err != nil {
			return err
		}
		if glyph == nil {
			continue
		}
		height := int64(javaRound(
			float32(int(glyph.YMaximum())+vmtx.TopSideBearing(cid)) * scaling))
		advance := int64(javaRound(float32(-vmtx.AdvanceHeight(cid)) * scaling))
		if height == vY && advance == w1 {
			// skip default metrics
			continue
		}
		// c [w1_1y v_1x v_1y w1_2y v_2x v_2y ... w1_ny v_nx v_ny]
		if prev != cid-1 {
			w2 = cos.NewArray()
			heights.Add(cos.GetInteger(int64(cid))) // c
			heights.Add(w2)
		}
		w2.Add(cos.GetInteger(advance)) // w1_iy
		width := int64(javaRound(float32(hmtx.AdvanceWidth(cid)) * scaling))
		w2.Add(cos.GetInteger(width / 2)) // v_ix
		w2.Add(cos.GetInteger(height))    // v_iy
		prev = cid
	}
	e.cidFont.SetItem(cos.W2, heights)
	return nil
}

// buildWidths builds the widths with an identity CIDToGIDMap, for embedding a
// whole font.
func (e *pdCIDFontType2Embedder) buildWidths(cidFont *cos.Dictionary) error {
	cidMax, err := e.ttf.NumberOfGlyphs()
	if err != nil {
		return err
	}
	gidwidths := make([]int, cidMax*2)
	hmtx, err := e.ttf.HorizontalMetrics()
	if err != nil {
		return err
	}
	for cid := 0; cid < cidMax; cid++ {
		gidwidths[cid*2] = cid
		gidwidths[cid*2+1] = hmtx.AdvanceWidth(cid)
	}

	widths, err := e.getWidths(gidwidths)
	if err != nil {
		return err
	}
	cidFont.SetItem(cos.W, widths)
	return nil
}

// getWidths compresses a cid/width sequence into the /W array's runs.
func (e *pdCIDFontType2Embedder) getWidths(widths []int) (*cos.Array, error) {
	if len(widths) < 2 {
		// Java throws IllegalArgumentException, which is unchecked.
		panic("length of widths must be >= 2")
	}

	scaling, err := e.unitsScaling()
	if err != nil {
		return nil, err
	}

	lastCid := int64(widths[0])
	lastValue := int64(javaRound(float32(widths[1]) * scaling))

	inner := cos.NewArray()
	outer := cos.NewArray()
	outer.Add(cos.GetInteger(lastCid))

	state := stateFirst

	for i := 2; i < len(widths)-1; i += 2 {
		cid := int64(widths[i])
		value := int64(javaRound(float32(widths[i+1]) * scaling))

		switch state {
		case stateFirst:
			if cid == lastCid+1 && value == lastValue {
				state = stateSerial
			} else if cid == lastCid+1 {
				state = stateBracket
				inner = cos.NewArray()
				inner.Add(cos.GetInteger(lastValue))
			} else {
				inner = cos.NewArray()
				inner.Add(cos.GetInteger(lastValue))
				outer.Add(inner)
				outer.Add(cos.GetInteger(cid))
			}
		case stateBracket:
			if cid == lastCid+1 && value == lastValue {
				state = stateSerial
				outer.Add(inner)
				outer.Add(cos.GetInteger(lastCid))
			} else if cid == lastCid+1 {
				inner.Add(cos.GetInteger(lastValue))
			} else {
				state = stateFirst
				inner.Add(cos.GetInteger(lastValue))
				outer.Add(inner)
				outer.Add(cos.GetInteger(cid))
			}
		case stateSerial:
			if cid != lastCid+1 || value != lastValue {
				outer.Add(cos.GetInteger(lastCid))
				outer.Add(cos.GetInteger(lastValue))
				outer.Add(cos.GetInteger(cid))
				state = stateFirst
			}
		}
		lastValue = value
		lastCid = cid
	}

	switch state {
	case stateFirst:
		inner = cos.NewArray()
		inner.Add(cos.GetInteger(lastValue))
		outer.Add(inner)
	case stateBracket:
		inner.Add(cos.GetInteger(lastValue))
		outer.Add(inner)
	case stateSerial:
		outer.Add(cos.GetInteger(lastCid))
		outer.Add(cos.GetInteger(lastValue))
	}
	return outer, nil
}

// buildVerticalMetrics builds the vertical metrics with an identity
// CIDToGIDMap, for embedding a whole font.
func (e *pdCIDFontType2Embedder) buildVerticalMetrics(cidFont *cos.Dictionary) error {
	ok, err := e.buildVerticalHeader(cidFont)
	if err != nil || !ok {
		return err
	}

	cidMax, err := e.ttf.NumberOfGlyphs()
	if err != nil {
		return err
	}
	gidMetrics := make([]int, cidMax*4)
	glyphTable, err := e.ttf.Glyph()
	if err != nil {
		return err
	}
	vmtx, err := e.ttf.VerticalMetrics()
	if err != nil {
		return err
	}
	hmtx, err := e.ttf.HorizontalMetrics()
	if err != nil {
		return err
	}
	for cid := 0; cid < cidMax; cid++ {
		glyph, err := glyphTable.GetGlyph(cid)
		if err != nil {
			return err
		}
		if glyph == nil {
			gidMetrics[cid*4] = math.MinInt32
		} else {
			gidMetrics[cid*4] = cid
			gidMetrics[cid*4+1] = vmtx.AdvanceHeight(cid)
			gidMetrics[cid*4+2] = hmtx.AdvanceWidth(cid)
			gidMetrics[cid*4+3] = int(glyph.YMaximum()) + vmtx.TopSideBearing(cid)
		}
	}

	metrics, err := e.getVerticalMetrics(gidMetrics)
	if err != nil {
		return err
	}
	cidFont.SetItem(cos.W2, metrics)
	return nil
}

// getVerticalMetrics compresses a cid/metric sequence into the /W2 array's
// runs.
func (e *pdCIDFontType2Embedder) getVerticalMetrics(values []int) (*cos.Array, error) {
	if len(values) < 4 {
		// Java throws IllegalArgumentException, which is unchecked.
		panic("length of values must be at least 4")
	}

	scaling, err := e.unitsScaling()
	if err != nil {
		return nil, err
	}

	lastCid := int64(values[0])
	lastW1Value := int64(javaRound(float32(-values[1]) * scaling))
	lastVxValue := int64(javaRound(float32(values[2]) * scaling / 2))
	lastVyValue := int64(javaRound(float32(values[3]) * scaling))

	inner := cos.NewArray()
	outer := cos.NewArray()
	outer.Add(cos.GetInteger(lastCid))

	state := stateFirst

	for i := 4; i < len(values)-3; i += 4 {
		cid := int64(values[i])
		if cid == math.MinInt32 {
			// no glyph for this cid
			continue
		}
		w1Value := int64(javaRound(float32(-values[i+1]) * scaling))
		vxValue := int64(javaRound(float32(values[i+2]) * scaling / 2))
		vyValue := int64(javaRound(float32(values[i+3]) * scaling))

		switch state {
		case stateFirst:
			if cid == lastCid+1 && w1Value == lastW1Value &&
				vxValue == lastVxValue && vyValue == lastVyValue {
				state = stateSerial
			} else if cid == lastCid+1 {
				state = stateBracket
				inner = cos.NewArray()
				inner.Add(cos.GetInteger(lastW1Value))
				inner.Add(cos.GetInteger(lastVxValue))
				inner.Add(cos.GetInteger(lastVyValue))
			} else {
				inner = cos.NewArray()
				inner.Add(cos.GetInteger(lastW1Value))
				inner.Add(cos.GetInteger(lastVxValue))
				inner.Add(cos.GetInteger(lastVyValue))
				outer.Add(inner)
				outer.Add(cos.GetInteger(cid))
			}
		case stateBracket:
			if cid == lastCid+1 && w1Value == lastW1Value &&
				vxValue == lastVxValue && vyValue == lastVyValue {
				state = stateSerial
				outer.Add(inner)
				outer.Add(cos.GetInteger(lastCid))
			} else if cid == lastCid+1 {
				inner.Add(cos.GetInteger(lastW1Value))
				inner.Add(cos.GetInteger(lastVxValue))
				inner.Add(cos.GetInteger(lastVyValue))
			} else {
				state = stateFirst
				inner.Add(cos.GetInteger(lastW1Value))
				inner.Add(cos.GetInteger(lastVxValue))
				inner.Add(cos.GetInteger(lastVyValue))
				outer.Add(inner)
				outer.Add(cos.GetInteger(cid))
			}
		case stateSerial:
			if cid != lastCid+1 || w1Value != lastW1Value ||
				vxValue != lastVxValue || vyValue != lastVyValue {
				outer.Add(cos.GetInteger(lastCid))
				outer.Add(cos.GetInteger(lastW1Value))
				outer.Add(cos.GetInteger(lastVxValue))
				outer.Add(cos.GetInteger(lastVyValue))
				outer.Add(cos.GetInteger(cid))
				state = stateFirst
			}
		}
		lastW1Value = w1Value
		lastVxValue = vxValue
		lastVyValue = vyValue
		lastCid = cid
	}

	switch state {
	case stateFirst:
		inner = cos.NewArray()
		inner.Add(cos.GetInteger(lastW1Value))
		inner.Add(cos.GetInteger(lastVxValue))
		inner.Add(cos.GetInteger(lastVyValue))
		outer.Add(inner)
	case stateBracket:
		inner.Add(cos.GetInteger(lastW1Value))
		inner.Add(cos.GetInteger(lastVxValue))
		inner.Add(cos.GetInteger(lastVyValue))
		outer.Add(inner)
	case stateSerial:
		outer.Add(cos.GetInteger(lastCid))
		outer.Add(cos.GetInteger(lastW1Value))
		outer.Add(cos.GetInteger(lastVxValue))
		outer.Add(cos.GetInteger(lastVyValue))
	}
	return outer, nil
}

// CIDFont returns the descendant CIDFont.
func (e *pdCIDFontType2Embedder) CIDFont() (PDCIDFont, error) {
	return NewPDCIDFontType2WithFont(e.cidFont, e.ttf, nil)
}
