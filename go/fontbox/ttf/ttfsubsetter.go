package ttf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/bits"
	"sort"
	"time"
	"unicode/utf16"
)

// padBuf is the zero padding tables and glyphs are aligned with.
var padBuf = []byte{0, 0, 0, 0}

// epoch1904 is the zero of a TrueType longDateTime.
var epoch1904 = time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)

// TTFSubsetter is a subsetter for TrueType (TTF) fonts.
//
// Originally developed by Wolfgang Glas for Sketch.
//
// Port of org.apache.fontbox.ttf.TTFSubsetter.
type TTFSubsetter struct {
	ttf         *TrueTypeFont
	unicodeCmap CmapLookup
	uniToGID    map[int]int // sorted on demand, standing for Java's TreeMap

	keepTables        []string // nil means keep every table
	glyphIds          sortedIntSet
	invisibleGlyphIds map[int]bool
	// prefix is Java's nullable String; the port takes the empty string for
	// absent, which is what setPrefix is never called with.
	prefix                     string
	hasAddedCompoundReferences bool
}

// NewTTFSubsetter creates a subsetter for the given font.
func NewTTFSubsetter(font *TrueTypeFont) (*TTFSubsetter, error) {
	return NewTTFSubsetterTables(font, nil)
}

// NewTTFSubsetterTables creates a subsetter for the given font, tables being the
// optional tables to keep if present.
func NewTTFSubsetterTables(font *TrueTypeFont, tables []string) (*TTFSubsetter, error) {
	s := &TTFSubsetter{
		ttf:               font,
		keepTables:        tables,
		uniToGID:          map[int]int{},
		invisibleGlyphIds: map[int]bool{},
	}

	// find the best Unicode cmap
	unicodeCmap, err := font.UnicodeCmapLookupStrict()
	if err != nil {
		return nil, err
	}
	s.unicodeCmap = unicodeCmap

	// always copy GID 0
	s.glyphIds.add(0)
	return s, nil
}

// SetPrefix sets the prefix to add to the font's PostScript name.
func (s *TTFSubsetter) SetPrefix(prefix string) { s.prefix = prefix }

// Add adds the given character code to the subset.
func (s *TTFSubsetter) Add(unicode int) {
	gid := s.unicodeCmap.GetGlyphID(unicode)
	if gid != 0 {
		s.uniToGID[unicode] = gid
		s.glyphIds.add(gid)
	}
}

// AddAll adds the given character codes to the subset.
func (s *TTFSubsetter) AddAll(unicodeSet []int) {
	for _, unicode := range unicodeSet {
		s.Add(unicode)
	}
}

// ForceInvisible forces the glyph for the specified character code to be
// zero-width and contour-free, regardless of what the glyph looks like in the
// original font. Note that the specified character code is not added to the
// subset unless it is also added separately.
func (s *TTFSubsetter) ForceInvisible(unicode int) {
	gid := s.unicodeCmap.GetGlyphID(unicode)
	if gid != 0 {
		s.invisibleGlyphIds[gid] = true
	}
}

// GetGIDMap returns the map of new -> old GIDs.
func (s *TTFSubsetter) GetGIDMap() (map[int]int, error) {
	if err := s.addCompoundReferences(); err != nil {
		return nil, err
	}

	newToOld := map[int]int{}
	newGID := 0
	for _, oldGID := range s.glyphIds.all() {
		newToOld[newGID] = oldGID
		newGID++
	}
	return newToOld, nil
}

// AddGlyphIds adds the given glyph ids to the subset.
func (s *TTFSubsetter) AddGlyphIds(allGlyphIds []int) {
	for _, gid := range allGlyphIds {
		s.glyphIds.add(gid)
	}
}

// writeFileHeader writes the file header of nTables tables and returns the file
// offset of the first TTF table to write.
func (s *TTFSubsetter) writeFileHeader(out *dataOutput, nTables int) int64 {
	out.writeInt(0x00010000)
	out.writeShort(nTables)

	mask := highestOneBit(nTables)
	searchRange := mask * 16
	out.writeShort(searchRange)

	entrySelector := log2(mask)

	out.writeShort(entrySelector)

	// numTables * 16 - searchRange
	last := 16*nTables - searchRange
	out.writeShort(last)

	return 0x00010000 + toUInt32Pair(nTables, searchRange) + toUInt32Pair(entrySelector, last)
}

func (s *TTFSubsetter) writeTableHeader(out *dataOutput, tag string, offset int64,
	tableBytes []byte) int64 {
	var checksum int64
	for nup, n := 0, len(tableBytes); nup < n; nup++ {
		checksum += int64(tableBytes[nup]&0xff) << (24 - nup%4*8)
	}
	checksum &= 0xffffffff

	tagbytes := []byte(tag)

	out.write(tagbytes[:4])
	out.writeInt(int32(checksum))
	out.writeInt(int32(offset))
	out.writeInt(int32(len(tableBytes)))

	// account for the checksum twice, once for the header field, once for the content itself
	return toUInt32Bytes(tagbytes) + checksum + checksum + offset + int64(len(tableBytes))
}

func (s *TTFSubsetter) writeTableBody(os *dataOutput, tableBytes []byte) {
	n := len(tableBytes)
	os.write(tableBytes)
	if n%4 != 0 {
		os.write(padBuf[:4-n%4])
	}
}

func (s *TTFSubsetter) buildHeadTable() ([]byte, error) {
	out := newDataOutput()

	h, err := s.ttf.Header()
	if err != nil {
		return nil, err
	}
	writeFixed(out, float64(h.Version()))
	writeFixed(out, float64(h.FontRevision()))
	writeUint32(out, 0) // h.getCheckSumAdjustment()
	writeUint32(out, h.MagicNumber())
	writeUint16(out, h.Flags())
	writeUint16(out, h.UnitsPerEm())
	writeLongDateTime(out, h.Created())
	writeLongDateTime(out, h.Modified())
	writeSInt16(out, h.XMin())
	writeSInt16(out, h.YMin())
	writeSInt16(out, h.XMax())
	writeSInt16(out, h.YMax())
	writeUint16(out, h.MacStyle())
	writeUint16(out, h.LowestRecPPEM())
	writeSInt16(out, h.FontDirectionHint())
	// force long format of 'loca' table
	writeSInt16(out, 1) // h.getIndexToLocFormat()
	writeSInt16(out, h.GlyphDataFormat())

	return out.bytes(), nil
}

func (s *TTFSubsetter) buildHheaTable() ([]byte, error) {
	out := newDataOutput()

	h, err := s.ttf.HorizontalHeader()
	if err != nil {
		return nil, err
	}
	writeFixed(out, float64(h.Version()))
	writeSInt16(out, h.Ascender())
	writeSInt16(out, h.Descender())
	writeSInt16(out, h.LineGap())
	writeUint16(out, h.AdvanceWidthMax())
	writeSInt16(out, h.MinLeftSideBearing())
	writeSInt16(out, h.MinRightSideBearing())
	writeSInt16(out, h.XMaxExtent())
	writeSInt16(out, h.CaretSlopeRise())
	writeSInt16(out, h.CaretSlopeRun())
	writeSInt16(out, h.Reserved1()) // caretOffset
	writeSInt16(out, h.Reserved2())
	writeSInt16(out, h.Reserved3())
	writeSInt16(out, h.Reserved4())
	writeSInt16(out, h.Reserved5())
	writeSInt16(out, h.MetricDataFormat())

	// is there a GID >= numberOfHMetrics ? Then keep the last entry of original hmtx table,
	// (add if it isn't in our set of GIDs), see also in buildHmtxTable()
	hmetrics := s.glyphIds.subSetSize(0, h.NumberOfHMetrics())
	if s.glyphIds.last() >= h.NumberOfHMetrics() && !s.glyphIds.contains(h.NumberOfHMetrics()-1) {
		hmetrics++
	}
	writeUint16(out, hmetrics)

	return out.bytes(), nil
}

func shouldCopyNameRecord(nr *NameRecord) bool {
	return nr.PlatformID() == PlatformWindows &&
		nr.PlatformEncodingID() == EncodingWindowsUnicodeBMP &&
		nr.LanguageID() == LanguageWindowsENUS &&
		nr.NameID() >= 0 && nr.NameID() < 7
}

func (s *TTFSubsetter) buildNameTable() ([]byte, error) {
	out := newDataOutput()

	name, err := s.ttf.Naming()
	if err != nil {
		return nil, err
	}
	if name == nil || s.keepTables != nil && !containsTable(s.keepTables, NamingTag) {
		return nil, nil
	}

	nameRecords := name.NameRecords()
	numRecords := 0
	for _, nr := range nameRecords {
		if shouldCopyNameRecord(nr) {
			numRecords++
		}
	}
	writeUint16(out, 0)
	writeUint16(out, numRecords)
	writeUint16(out, 2*3+2*6*numRecords)

	if numRecords == 0 {
		return nil, nil
	}

	names := make([][]byte, numRecords)
	j := 0
	for _, nameRecord := range nameRecords {
		if shouldCopyNameRecord(nameRecord) {
			platform := nameRecord.PlatformID()
			encoding := nameRecord.PlatformEncodingID()
			charset := charsetISO88591

			if platform == CmapPlatformWindows && encoding == EncodingWinUnicodeBMP {
				charset = charsetUTF16BE
			} else if platform == 2 { // ISO [deprecated]=
				if encoding == 0 { // 7-bit ASCII
					charset = charsetUSASCII
				} else if encoding == 1 { // ISO 10646=
					// not sure is this is correct??
					charset = charsetUTF16BE
				}
			}
			value := nameRecord.String()
			if nameRecord.NameID() == 6 && s.prefix != "" {
				value = s.prefix + value
			}
			names[j] = encodeString(value, charset)
			j++
		}
	}

	offset := 0
	j = 0
	for _, nr := range nameRecords {
		if shouldCopyNameRecord(nr) {
			writeUint16(out, nr.PlatformID())
			writeUint16(out, nr.PlatformEncodingID())
			writeUint16(out, nr.LanguageID())
			writeUint16(out, nr.NameID())
			writeUint16(out, len(names[j]))
			writeUint16(out, offset)
			offset += len(names[j])
			j++
		}
	}

	for i := 0; i < numRecords; i++ {
		out.write(names[i])
	}

	return out.bytes(), nil
}

func (s *TTFSubsetter) buildMaxpTable() ([]byte, error) {
	out := newDataOutput()

	p, err := s.ttf.MaximumProfile()
	if err != nil {
		return nil, err
	}
	writeFixed(out, float64(p.Version()))
	writeUint16(out, s.glyphIds.size())
	if p.Version() >= 1.0 {
		writeUint16(out, p.MaxPoints())
		writeUint16(out, p.MaxContours())
		writeUint16(out, p.MaxCompositePoints())
		writeUint16(out, p.MaxCompositeContours())
		writeUint16(out, p.MaxZones())
		writeUint16(out, p.MaxTwilightPoints())
		writeUint16(out, p.MaxStorage())
		writeUint16(out, p.MaxFunctionDefs())
		writeUint16(out, p.MaxInstructionDefs())
		writeUint16(out, p.MaxStackElements())
		writeUint16(out, p.MaxSizeOfInstructions())
		writeUint16(out, p.MaxComponentElements())
		writeUint16(out, p.MaxComponentDepth())
	}
	return out.bytes(), nil
}

func (s *TTFSubsetter) buildOS2Table() ([]byte, error) {
	os2, err := s.ttf.OS2Windows()
	if err != nil {
		return nil, err
	}
	if os2 == nil || len(s.uniToGID) == 0 ||
		s.keepTables != nil && !containsTable(s.keepTables, OS2WindowsMetricsTag) {
		return nil, nil
	}

	out := newDataOutput()

	writeUint16(out, os2.Version())
	writeSInt16(out, os2.AverageCharWidth())
	writeUint16(out, os2.WeightClass())
	writeUint16(out, os2.WidthClass())

	writeSInt16(out, os2.FsType())

	writeSInt16(out, os2.SubscriptXSize())
	writeSInt16(out, os2.SubscriptYSize())
	writeSInt16(out, os2.SubscriptXOffset())
	writeSInt16(out, os2.SubscriptYOffset())

	writeSInt16(out, os2.SuperscriptXSize())
	writeSInt16(out, os2.SuperscriptYSize())
	writeSInt16(out, os2.SuperscriptXOffset())
	writeSInt16(out, os2.SuperscriptYOffset())

	writeSInt16(out, os2.StrikeoutSize())
	writeSInt16(out, os2.StrikeoutPosition())
	writeSInt16(out, int16(os2.FamilyClass()))
	out.write(os2.Panose())

	writeUint32(out, 0)
	writeUint32(out, 0)
	writeUint32(out, 0)
	writeUint32(out, 0)

	out.write([]byte(os2.AchVendID()))

	writeUint16(out, os2.FsSelection())
	unicodes := s.sortedUnicodes()
	writeUint16(out, unicodes[0])
	writeUint16(out, unicodes[len(unicodes)-1])
	writeUint16(out, os2.TypoAscender())
	writeUint16(out, os2.TypoDescender())
	writeUint16(out, os2.TypoLineGap())
	writeUint16(out, os2.WinAscent())
	writeUint16(out, os2.WinDescent())

	return out.bytes(), nil
}

// buildLocaTable never returns nil.
func (s *TTFSubsetter) buildLocaTable(newOffsets []int64) []byte {
	out := newDataOutput()

	for _, offset := range newOffsets {
		writeUint32(out, offset)
	}

	return out.bytes()
}

// addCompoundReferences resolves compound glyph references.
func (s *TTFSubsetter) addCompoundReferences() error {
	if s.hasAddedCompoundReferences {
		return nil
	}
	s.hasAddedCompoundReferences = true

	g, err := s.ttf.Glyph()
	if err != nil {
		return err
	}
	indexToLocation, err := s.ttf.IndexToLocation()
	if err != nil {
		return err
	}
	offsets := indexToLocation.Offsets()
	for {
		var glyphIdsToAdd []int
		is, err := s.ttf.OriginalData()
		if err != nil {
			return err
		}
		isResult, err := skipBytes(is, g.Offset())
		if err != nil {
			return err
		}
		if isResult != g.Offset() {
			slog.Debug("Tried skipping bytes but skipped fewer",
				"wanted", g.Offset(), "skipped", isResult)
		}

		var lastOff int64
		for _, gid := range s.glyphIds.all() {
			offset := offsets[gid]
			length := offsets[gid+1] - offset
			isResult, err := skipBytes(is, offset-lastOff)
			if err != nil {
				return err
			}
			if isResult != offset-lastOff {
				slog.Debug("Tried skipping bytes but skipped fewer",
					"wanted", offset-lastOff, "skipped", isResult)
			}
			buf := make([]byte, int(length))
			read, err := readSome(is, buf)
			if err != nil {
				return err
			}
			if int64(read) != length {
				slog.Debug("Tried reading bytes but read fewer",
					"wanted", length, "read", read)
			}

			// rewrite glyphIds for compound glyphs
			if len(buf) >= 2 && buf[0] == 0xff && buf[1] == 0xff {
				off := 2 * 5
				var flags int
				for {
					flags = int(buf[off])<<8 | int(buf[off+1])
					off += 2
					ogid := int(buf[off])<<8 | int(buf[off+1])
					if !s.glyphIds.contains(ogid) {
						glyphIdsToAdd = append(glyphIdsToAdd, ogid)
					}
					off += 2
					// ARG_1_AND_2_ARE_WORDS
					if flags&(1<<0) != 0 {
						off += 2 * 2
					} else {
						off += 2
					}
					// WE_HAVE_A_TWO_BY_TWO
					if flags&(1<<7) != 0 {
						off += 2 * 4
					} else if flags&(1<<6) != 0 {
						// WE_HAVE_AN_X_AND_Y_SCALE
						off += 2 * 2
					} else if flags&(1<<3) != 0 {
						// WE_HAVE_A_SCALE
						off += 2
					}
					if flags&(1<<5) == 0 { // MORE_COMPONENTS
						break
					}
				}
			}
			lastOff = offsets[gid+1]
		}
		if len(glyphIdsToAdd) == 0 {
			return nil
		}
		for _, gid := range glyphIdsToAdd {
			s.glyphIds.add(gid)
		}
	}
}

// buildGlyfTable never returns nil.
func (s *TTFSubsetter) buildGlyfTable(newOffsets []int64) ([]byte, error) {
	var bos bytes.Buffer

	g, err := s.ttf.Glyph()
	if err != nil {
		return nil, err
	}
	indexToLocation, err := s.ttf.IndexToLocation()
	if err != nil {
		return nil, err
	}
	offsets := indexToLocation.Offsets()
	is, err := s.ttf.OriginalData()
	if err != nil {
		return nil, err
	}
	isResult, err := skipBytes(is, g.Offset())
	if err != nil {
		return nil, err
	}
	if isResult != g.Offset() {
		slog.Debug("Tried skipping bytes but skipped fewer",
			"wanted", g.Offset(), "skipped", isResult)
	}

	var lastOff int64   // previously read glyph offset
	var newOffset int64 // new offset for the glyph in the subset font
	newGid := 0         // new GID in subset font

	// for each glyph in the subset
	for _, gid := range s.glyphIds.all() {
		offset := offsets[gid]
		length := offsets[gid+1] - offset

		newOffsets[newGid] = newOffset
		newGid++
		isResult, err := skipBytes(is, offset-lastOff)
		if err != nil {
			return nil, err
		}
		if isResult != offset-lastOff {
			slog.Debug("Tried skipping bytes but skipped fewer",
				"wanted", offset-lastOff, "skipped", isResult)
		}

		// glyphs with no outlines have an empty entry in the 'glyf' table, with a
		// corresponding 'loca' table entry with length = 0
		if s.invisibleGlyphIds[gid] {
			lastOff = offset
			continue
		}

		buf := make([]byte, int(length))
		read, err := readSome(is, buf)
		if err != nil {
			return nil, err
		}
		if int64(read) != length {
			slog.Debug("Tried reading bytes but read fewer", "wanted", length, "read", read)
		}

		// detect glyph type
		if len(buf) >= 2 && buf[0] == 0xff && buf[1] == 0xff {
			// compound glyph
			off := 2 * 5
			var flags int
			for {
				// flags
				flags = int(buf[off])<<8 | int(buf[off+1])
				off += 2

				// glyphIndex
				componentGid := int(buf[off])<<8 | int(buf[off+1])
				if !s.glyphIds.contains(componentGid) {
					// PDFBOX-6085
					return nil, fmt.Errorf(
						"ttf: internal error: componentGid %d not in glyphIds set", componentGid)
				}
				newComponentGid := s.getNewGlyphID(componentGid)
				buf[off] = byte(uint32(newComponentGid) >> 8)
				buf[off+1] = byte(newComponentGid)
				off += 2

				// ARG_1_AND_2_ARE_WORDS
				if flags&(1<<0) != 0 {
					off += 2 * 2
				} else {
					off += 2
				}
				// WE_HAVE_A_TWO_BY_TWO
				if flags&(1<<7) != 0 {
					off += 2 * 4
				} else if flags&(1<<6) != 0 {
					// WE_HAVE_AN_X_AND_Y_SCALE
					off += 2 * 2
				} else if flags&(1<<3) != 0 {
					// WE_HAVE_A_SCALE
					off += 2
				}
				if flags&(1<<5) == 0 { // MORE_COMPONENTS
					break
				}
			}

			// WE_HAVE_INSTRUCTIONS
			if flags&0x0100 == 0x0100 {
				// USHORT numInstr
				numInstr := int(buf[off])<<8 | int(buf[off+1])
				off += 2

				// BYTE instr[numInstr]
				off += numInstr
			}

			// write the compound glyph
			bos.Write(buf[:off])

			// offset to start next glyph
			newOffset += int64(off)
		} else if len(buf) > 0 {
			// copy the entire glyph
			bos.Write(buf)

			// offset to start next glyph
			newOffset += int64(len(buf))
		}

		// 4-byte alignment
		if newOffset%4 != 0 {
			padLen := 4 - int(newOffset%4)
			bos.Write(padBuf[:padLen])
			newOffset += int64(padLen)
		}

		lastOff = offset + length
	}
	newOffsets[newGid] = newOffset

	return bos.Bytes(), nil
}

func (s *TTFSubsetter) getNewGlyphID(oldGid int) int {
	return s.glyphIds.headSetSize(oldGid)
}

func (s *TTFSubsetter) buildCmapTable() ([]byte, error) {
	cmap, err := s.ttf.Cmap()
	if err != nil {
		return nil, err
	}
	if cmap == nil || len(s.uniToGID) == 0 ||
		s.keepTables != nil && !containsTable(s.keepTables, CmapTag) {
		return nil, nil
	}

	out := newDataOutput()

	// cmap header
	writeUint16(out, 0) // version
	writeUint16(out, 1) // numberSubtables

	// encoding record
	writeUint16(out, CmapPlatformWindows)   // platformID
	writeUint16(out, EncodingWinUnicodeBMP) // platformSpecificID
	writeUint32(out, 12)                    // offset 4 * 2 + 4

	// build Format 4 subtable (Unicode BMP)
	unicodes := s.sortedUnicodes()
	index := 0
	lastChar := unicodes[index]
	index++
	prevChar := lastChar
	lastGid := s.getNewGlyphID(s.uniToGID[lastChar])

	// +1 because .notdef is missing in uniToGID
	startCode := make([]int, len(s.uniToGID)+1)
	endCode := make([]int, len(startCode))
	idDelta := make([]int, len(startCode))
	segCount := 0
	for index < len(unicodes) {
		curChar := unicodes[index]
		index++
		curGid := s.getNewGlyphID(s.uniToGID[curChar])

		// todo: need format Format 12 for non-BMP
		if curChar > 0xFFFF {
			return nil, errors.New("ttf: non-BMP Unicode character")
		}

		if curChar != prevChar+1 || curGid-lastGid != curChar-lastChar {
			if lastGid != 0 {
				// don't emit ranges, which map to GID 0, the
				// undef glyph is emitted a the very last segment
				startCode[segCount] = lastChar
				endCode[segCount] = prevChar
				idDelta[segCount] = lastGid - lastChar
				segCount++
			} else if lastChar != prevChar {
				// shorten ranges which start with GID 0 by one
				startCode[segCount] = lastChar + 1
				endCode[segCount] = prevChar
				idDelta[segCount] = lastGid - lastChar
				segCount++
			}
			lastGid = curGid
			lastChar = curChar
		}
		prevChar = curChar
	}

	// trailing segment
	startCode[segCount] = lastChar
	endCode[segCount] = prevChar
	idDelta[segCount] = lastGid - lastChar
	segCount++

	// GID 0
	startCode[segCount] = 0xffff
	endCode[segCount] = 0xffff
	idDelta[segCount] = 1
	segCount++

	// write format 4 subtable
	searchRange := 2 * int(math.Pow(2, float64(log2(segCount))))
	writeUint16(out, 4)                      // format
	writeUint16(out, 8*2+segCount*4*2)       // length
	writeUint16(out, 0)                      // language
	writeUint16(out, segCount*2)             // segCountX2
	writeUint16(out, searchRange)            // searchRange
	writeUint16(out, log2(searchRange/2))    // entrySelector
	writeUint16(out, 2*segCount-searchRange) // rangeShift

	// endCode[segCount]
	for i := 0; i < segCount; i++ {
		writeUint16(out, endCode[i])
	}

	// reservedPad
	writeUint16(out, 0)

	// startCode[segCount]
	for i := 0; i < segCount; i++ {
		writeUint16(out, startCode[i])
	}

	// idDelta[segCount]
	for i := 0; i < segCount; i++ {
		writeUint16(out, idDelta[i])
	}

	for i := 0; i < segCount; i++ {
		writeUint16(out, 0)
	}

	return out.bytes(), nil
}

func (s *TTFSubsetter) buildPostTable() ([]byte, error) {
	post, err := s.ttf.PostScript()
	if err != nil {
		return nil, err
	}
	if post == nil || post.GlyphNames() == nil ||
		s.keepTables != nil && !containsTable(s.keepTables, PostScriptTag) {
		return nil, nil
	}

	out := newDataOutput()

	writeFixed(out, 2.0) // version
	writeFixed(out, float64(post.ItalicAngle()))
	writeSInt16(out, post.UnderlinePosition())
	writeSInt16(out, post.UnderlineThickness())
	writeUint32(out, post.IsFixedPitch())
	writeUint32(out, post.MinMemType42())
	writeUint32(out, post.MaxMemType42())
	writeUint32(out, post.MinMemType1())
	writeUint32(out, post.MaxMemType1())

	// version 2.0

	// numberOfGlyphs
	writeUint16(out, s.glyphIds.size())

	// glyphNameIndex[numGlyphs]
	//
	// Java keeps the explicit names in a LinkedHashMap, which is the insertion
	// order the names are written back out in; the port keeps that order in a
	// slice beside the map.
	names := map[string]int{}
	var nameOrder []string
	for _, gid := range s.glyphIds.all() {
		// PostScriptTable.getName returns null for a gid past the end of the
		// names array, and Java goes on to call getBytes on it; the port has
		// the empty string there and writes a zero-length name instead.
		name := post.GetName(gid)
		macID, ok := GlyphIndex(name)
		if ok {
			// the name is implicit, as it's from MacRoman
			writeUint16(out, macID)
		} else {
			// the name will be written explicitly
			ordinal, seen := names[name]
			if !seen {
				ordinal = len(names)
				names[name] = ordinal
				nameOrder = append(nameOrder, name)
			}
			writeUint16(out, 258+ordinal)
		}
	}

	// names[numberNewGlyphs]
	for _, name := range nameOrder {
		buf := encodeString(name, charsetUSASCII)
		writeUint8(out, len(buf))
		out.write(buf)
	}

	return out.bytes(), nil
}

func (s *TTFSubsetter) buildHmtxTable() ([]byte, error) {
	var bos bytes.Buffer

	h, err := s.ttf.HorizontalHeader()
	if err != nil {
		return nil, err
	}
	hm, err := s.ttf.HorizontalMetrics()
	if err != nil {
		return nil, err
	}
	is, err := s.ttf.OriginalData()
	if err != nil {
		return nil, err
	}

	// more info: https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6hmtx.html
	lastgid := h.NumberOfHMetrics() - 1
	// true if lastgid is not in the set: we'll need its width (but not its left side bearing) later
	needLastGidWidth := s.glyphIds.last() > lastgid && !s.glyphIds.contains(lastgid)

	isResult, err := skipBytes(is, hm.Offset())
	if err != nil {
		return nil, err
	}
	if isResult != hm.Offset() {
		slog.Debug("Tried skipping bytes but skipped fewer",
			"wanted", hm.Offset(), "skipped", isResult)
	}

	var lastOffset int64
	for _, gid := range s.glyphIds.all() {
		// offset in original file
		var offset int64
		if gid <= lastgid {
			if s.invisibleGlyphIds[gid] {
				// force zero width (no change to last offset)
				// 4 bytes total, 2 bytes each for: advance width = 0, left side bearing = 0
				bos.Write(padBuf[:4])
			} else {
				// copy width and lsb
				offset = int64(gid) * 4
				if lastOffset, err = copyBytes(is, &bos, offset, lastOffset, 4); err != nil {
					return nil, err
				}
			}
		} else {
			if needLastGidWidth {
				// one time only: copy width from lastgid, whose width applies
				// to all later glyphs
				needLastGidWidth = false
				offset = int64(lastgid) * 4
				if lastOffset, err = copyBytes(is, &bos, offset, lastOffset, 2); err != nil {
					return nil, err
				}

				// then go on with lsb from actual glyph (lsb are individual even in monotype fonts)
			}

			// copy lsb only, as we are beyond numOfHMetrics
			offset = int64(h.NumberOfHMetrics())*4 + int64(gid-h.NumberOfHMetrics())*2
			if lastOffset, err = copyBytes(is, &bos, offset, lastOffset, 2); err != nil {
				return nil, err
			}
		}
	}

	return bos.Bytes(), nil
}

func copyBytes(is io.Reader, os io.Writer, newOffset, lastOffset int64, count int) (int64, error) {
	// skip over from last original offset
	nskip := newOffset - lastOffset
	skipped, err := skipBytes(is, nskip)
	if err != nil {
		return 0, err
	}
	if nskip != skipped {
		return 0, errors.New("ttf: unexpected EOF exception parsing glyphId of hmtx table")
	}
	buf := make([]byte, count)
	read, err := readSome(is, buf)
	if err != nil {
		return 0, err
	}
	if count != read {
		return 0, errors.New("ttf: unexpected EOF exception parsing glyphId of hmtx table")
	}
	if _, err := os.Write(buf[:count]); err != nil {
		return 0, err
	}
	return newOffset + int64(count), nil
}

// WriteToStream writes the subfont to the given output stream, which it closes
// where the stream is a Closer -- Java's DataOutputStream wrapper closes it.
func (s *TTFSubsetter) WriteToStream(os io.Writer) error {
	if s.glyphIds.isEmpty() && len(s.uniToGID) == 0 {
		slog.Info("font subset is empty")
	}

	if err := s.addCompoundReferences(); err != nil {
		return err
	}

	if closer, ok := os.(io.Closer); ok {
		defer closer.Close()
	}

	newLoca := make([]int64, s.glyphIds.size()+1)

	// generate tables in dependency order
	head, err := s.buildHeadTable()
	if err != nil {
		return err
	}
	hhea, err := s.buildHheaTable()
	if err != nil {
		return err
	}
	maxp, err := s.buildMaxpTable()
	if err != nil {
		return err
	}
	name, err := s.buildNameTable()
	if err != nil {
		return err
	}
	os2, err := s.buildOS2Table()
	if err != nil {
		return err
	}
	glyf, err := s.buildGlyfTable(newLoca)
	if err != nil {
		return err
	}
	loca := s.buildLocaTable(newLoca)
	cmap, err := s.buildCmapTable()
	if err != nil {
		return err
	}
	hmtx, err := s.buildHmtxTable()
	if err != nil {
		return err
	}
	post, err := s.buildPostTable()
	if err != nil {
		return err
	}

	// save to TTF in optimized order
	//
	// Java uses a TreeMap, so the tables come out ordered by tag; the port
	// sorts the keys where it walks them.
	tables := map[string][]byte{}
	if os2 != nil {
		tables[OS2WindowsMetricsTag] = os2
	}
	if cmap != nil {
		tables[CmapTag] = cmap
	}
	tables[GlyphTag] = glyf
	tables[HeaderTag] = head
	tables[HorizontalHeaderTag] = hhea
	tables[HorizontalMetricsTag] = hmtx
	tables[IndexToLocationTag] = loca
	tables[MaximumProfileTag] = maxp
	if name != nil {
		tables[NamingTag] = name
	}
	if post != nil {
		tables[PostScriptTag] = post
	}

	// copy all other tables
	for tag, table := range s.ttf.TableMap() {
		if _, present := tables[tag]; !present &&
			(s.keepTables == nil || containsTable(s.keepTables, tag)) {
			tableBytes, err := s.ttf.TableBytes(table)
			if err != nil {
				return err
			}
			tables[tag] = tableBytes
		}
	}

	tags := make([]string, 0, len(tables))
	for tag := range tables {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	out := newDataOutput()

	// calculate checksum
	checksum := s.writeFileHeader(out, len(tables))
	offset := int64(12) + 16*int64(len(tables))
	for _, tag := range tags {
		checksum += s.writeTableHeader(out, tag, offset, tables[tag])
		offset += (int64(len(tables[tag])) + 3) / 4 * 4
	}
	checksum = 0xB1B0AFBA - (checksum & 0xffffffff)

	// update checksumAdjustment in 'head' table
	head[8] = byte(uint64(checksum) >> 24)
	head[9] = byte(uint64(checksum) >> 16)
	head[10] = byte(uint64(checksum) >> 8)
	head[11] = byte(checksum)
	for _, tag := range tags {
		s.writeTableBody(out, tables[tag])
	}

	_, err = os.Write(out.bytes())
	return err
}

// sortedUnicodes is the ascending key order of Java's TreeMap uniToGID.
func (s *TTFSubsetter) sortedUnicodes() []int {
	unicodes := make([]int, 0, len(s.uniToGID))
	for unicode := range s.uniToGID {
		unicodes = append(unicodes, unicode)
	}
	sort.Ints(unicodes)
	return unicodes
}

func containsTable(tables []string, tag string) bool {
	for _, table := range tables {
		if table == tag {
			return true
		}
	}
	return false
}

// dataOutput is java.io.DataOutputStream over a ByteArrayOutputStream: every
// number is written big-endian and truncated to its width.
type dataOutput struct {
	buf bytes.Buffer
}

func newDataOutput() *dataOutput { return &dataOutput{} }

func (o *dataOutput) bytes() []byte { return o.buf.Bytes() }

func (o *dataOutput) write(b []byte) { o.buf.Write(b) }

func (o *dataOutput) writeByte(i int) { o.buf.WriteByte(byte(i)) }

func (o *dataOutput) writeShort(i int) {
	o.buf.WriteByte(byte(uint32(i) >> 8))
	o.buf.WriteByte(byte(i))
}

func (o *dataOutput) writeInt(i int32) {
	o.buf.WriteByte(byte(uint32(i) >> 24))
	o.buf.WriteByte(byte(uint32(i) >> 16))
	o.buf.WriteByte(byte(uint32(i) >> 8))
	o.buf.WriteByte(byte(i))
}

func (o *dataOutput) writeLong(l int64) {
	for shift := 56; shift >= 0; shift -= 8 {
		o.buf.WriteByte(byte(uint64(l) >> shift))
	}
}

func writeFixed(out *dataOutput, f float64) {
	ip := math.Floor(f)
	fp := (f - ip) * 65536.0
	out.writeShort(int(doubleToInt32(ip)))
	out.writeShort(int(doubleToInt32(fp)))
}

func writeUint32(out *dataOutput, l int64) { out.writeInt(int32(l)) }

func writeUint16(out *dataOutput, i int) { out.writeShort(i) }

func writeSInt16(out *dataOutput, i int16) { out.writeShort(int(i)) }

func writeUint8(out *dataOutput, i int) { out.writeByte(i) }

// writeLongDateTime is the inverse operation of TTFDataStream.readInternationalDate.
func writeLongDateTime(out *dataOutput, calendar time.Time) {
	millisFor1904 := epoch1904.UnixMilli()
	secondsSince1904 := (calendar.UnixMilli() - millisFor1904) / 1000
	out.writeLong(secondsSince1904)
}

func toUInt32Pair(high, low int) int64 {
	return int64(high&0xffff)<<16 | int64(low&0xffff)
}

func toUInt32Bytes(b []byte) int64 {
	return int64(b[0])<<24 | int64(b[1])<<16 | int64(b[2])<<8 | int64(b[3])
}

func log2(num int) int {
	return int(math.Floor(math.Log(float64(num)) / math.Log(2)))
}

// highestOneBit is Integer.highestOneBit.
func highestOneBit(i int) int {
	if i <= 0 {
		return 0
	}
	return 1 << (bits.Len(uint(i)) - 1)
}

// doubleToInt32 is Java's (int) cast of a double, which saturates rather than
// wrapping and turns NaN into zero.
func doubleToInt32(d float64) int32 {
	switch {
	case math.IsNaN(d):
		return 0
	case d >= math.MaxInt32:
		return math.MaxInt32
	case d <= math.MinInt32:
		return math.MinInt32
	default:
		return int32(d)
	}
}

// skipBytes is InputStream.skip, which reports how far it got rather than
// failing at the end of the stream.
func skipBytes(r io.Reader, n int64) (int64, error) {
	if n <= 0 {
		return 0, nil
	}
	skipped, err := io.CopyN(io.Discard, r, n)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return skipped, err
}

// readSome is InputStream.read(byte[]), which reports how much it read.
//
// Java calls it once and logs a short read; the stream it gets is always over a
// byte array, where one call fills the buffer, so the port reads to the end of
// the buffer and reports the same count.
func readSome(r io.Reader, buf []byte) (int, error) {
	read, err := io.ReadFull(r, buf)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		err = nil
	}
	return read, err
}

// encodeString is String.getBytes(Charset), which replaces a character the
// charset cannot hold with '?'.
func encodeString(value string, cs int) []byte {
	units := utf16.Encode([]rune(value))
	switch cs {
	case charsetUTF16BE:
		out := make([]byte, 0, len(units)*2)
		for _, unit := range units {
			out = append(out, byte(unit>>8), byte(unit))
		}
		return out
	case charsetUSASCII:
		out := make([]byte, 0, len(units))
		for _, unit := range units {
			if unit > 0x7F {
				out = append(out, '?')
			} else {
				out = append(out, byte(unit))
			}
		}
		return out
	default: // charsetISO88591
		out := make([]byte, 0, len(units))
		for _, unit := range units {
			if unit > 0xFF {
				out = append(out, '?')
			} else {
				out = append(out, byte(unit))
			}
		}
		return out
	}
}

// sortedIntSet is java.util.TreeSet<Integer> over the operations the subsetter
// uses: an ascending walk, a membership test, and the size of a head or a sub
// set.
type sortedIntSet struct {
	values []int
}

func (s *sortedIntSet) add(v int) {
	i := sort.SearchInts(s.values, v)
	if i < len(s.values) && s.values[i] == v {
		return
	}
	s.values = append(s.values, 0)
	copy(s.values[i+1:], s.values[i:])
	s.values[i] = v
}

func (s *sortedIntSet) contains(v int) bool {
	i := sort.SearchInts(s.values, v)
	return i < len(s.values) && s.values[i] == v
}

func (s *sortedIntSet) size() int { return len(s.values) }

func (s *sortedIntSet) isEmpty() bool { return len(s.values) == 0 }

func (s *sortedIntSet) last() int { return s.values[len(s.values)-1] }

// headSetSize is headSet(v).size(), the number of members below v.
func (s *sortedIntSet) headSetSize(v int) int { return sort.SearchInts(s.values, v) }

// subSetSize is subSet(from, to).size(), the number of members in [from, to).
func (s *sortedIntSet) subSetSize(from, to int) int {
	return sort.SearchInts(s.values, to) - sort.SearchInts(s.values, from)
}

func (s *sortedIntSet) all() []int { return s.values }
