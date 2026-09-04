package ttf

import (
	"fmt"
	"sort"
)

// The platform IDs a cmap subtable can be for.
const (
	CmapPlatformUnicode   = 0
	CmapPlatformMacintosh = 1
	CmapPlatformWindows   = 3
)

// EncodingMacRoman is the Roman encoding ID of the Macintosh platform.
const EncodingMacRoman = 0

// The encoding IDs of the Windows platform.
const (
	EncodingWinSymbol      = 0  // Unicode, non-standard character set
	EncodingWinUnicodeBMP  = 1  // Unicode BMP (UCS-2)
	EncodingWinShiftJIS    = 2  //
	EncodingWinBig5        = 3  //
	EncodingWinPRC         = 4  //
	EncodingWinWansung     = 5  //
	EncodingWinJohab       = 6  //
	EncodingWinUnicodeFull = 10 // Unicode Full (UCS-4)
)

// The encoding IDs of the Unicode platform.
const (
	CmapEncodingUnicode10     = 0
	CmapEncodingUnicode11     = 1
	CmapEncodingUnicode20BMP  = 3
	CmapEncodingUnicode20Full = 4
)

// CmapTable is the cmap table: the subtables mapping character codes to glyphs.
//
// Port of org.apache.fontbox.ttf.CmapTable.
type CmapTable struct {
	Table

	cmaps []*CmapSubtable
}

var _ TableReader = (*CmapTable)(nil)

// Read reads the cmap table and every subtable in it.
func (t *CmapTable) Read(ttf *TrueTypeFont, data DataStream) error {
	r := newReader(data)
	_ = r.unsignedShort() // version
	numberOfTables := r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	t.cmaps = make([]*CmapSubtable, numberOfTables)
	for i := 0; i < numberOfTables; i++ {
		cmap := &CmapSubtable{}
		if err := cmap.initData(data); err != nil {
			return err
		}
		t.cmaps[i] = cmap
	}
	numberOfGlyphs, err := ttf.NumberOfGlyphs()
	if err != nil {
		return err
	}
	for i := 0; i < numberOfTables; i++ {
		if err := t.cmaps[i].initSubtable(t, numberOfGlyphs, data); err != nil {
			return err
		}
	}
	t.SetInitialized(true)
	return nil
}

// Cmaps returns every subtable of the table.
func (t *CmapTable) Cmaps() []*CmapSubtable { return t.cmaps }

// SetCmaps sets the subtables of the table.
func (t *CmapTable) SetCmaps(cmaps []*CmapSubtable) { t.cmaps = cmaps }

// GetSubtable returns the subtable for the given platform and encoding, or nil
// where the table has none.
func (t *CmapTable) GetSubtable(platformID, platformEncodingID int) *CmapSubtable {
	for _, cmap := range t.cmaps {
		if cmap.PlatformID() == platformID &&
			cmap.PlatformEncodingID() == platformEncodingID {
			return cmap
		}
	}
	return nil
}

// CmapLookup is what a character code is looked up through.
//
// Port of org.apache.fontbox.ttf.CmapLookup.
type CmapLookup interface {
	// GetGlyphID returns the glyph for the given code point.
	GetGlyphID(codePointAt int) int
	// GetCharCodes returns every code point mapped to the given glyph, in
	// ascending order, or nil where none is.
	GetCharCodes(gid int) []int
}

// The two halves a supplementary code point is split into by a format 8
// subtable.
const (
	leadOffset      = 0xD800 - (0x10000 >> 10)
	surrogateOffset = 0x10000 - (0xD800 << 10) - 0xDC00
)

// maxInt32 is Java's Integer.MAX_VALUE, which several of the subtable formats
// range-check against.
const maxInt32 = 1<<31 - 1

// CmapSubtable is one subtable of the cmap table: a mapping from character
// codes to glyphs in one particular encoding.
//
// Port of org.apache.fontbox.ttf.CmapSubtable.
type CmapSubtable struct {
	platformID         int
	platformEncodingID int
	subTableOffset     int64

	// glyphIdToCharacterCode holds -1 where a glyph has no code, and the
	// smallest negative int where it has several, which then live in
	// glyphIdToCharacterCodeMultiple.
	glyphIDToCharacterCode         []int
	glyphIDToCharacterCodeMultiple map[int][]int
	characterCodeToGlyphID         map[int]int
}

var _ CmapLookup = (*CmapSubtable)(nil)

// initData reads the entry of the subtable in the cmap directory.
func (c *CmapSubtable) initData(data DataStream) error {
	r := newReader(data)
	c.platformID = r.unsignedShort()
	c.platformEncodingID = r.unsignedShort()
	c.subTableOffset = r.unsignedInt()
	return r.err
}

// initSubtable reads the subtable itself.
func (c *CmapSubtable) initSubtable(cmap *CmapTable, numGlyphs int, data DataStream) error {
	c.glyphIDToCharacterCodeMultiple = map[int][]int{}

	r := newReader(data)
	r.seek(cmap.Offset() + c.subTableOffset)
	subtableFormat := r.unsignedShort()
	if subtableFormat < 8 {
		_ = r.unsignedShort() // length
		_ = r.unsignedShort() // version
	} else {
		// read an other UnsignedShort to read a Fixed32
		_ = r.unsignedShort()
		_ = r.unsignedInt() // length
		_ = r.unsignedInt() // version
	}
	if r.err != nil {
		return r.err
	}

	switch subtableFormat {
	case 0:
		return c.processSubtype0(data)
	case 2:
		return c.processSubtype2(data, numGlyphs)
	case 4:
		return c.processSubtype4(data, numGlyphs)
	case 6:
		return c.processSubtype6(data, numGlyphs)
	case 8:
		return c.processSubtype8(data, numGlyphs)
	case 10:
		return c.processSubtype10(data, numGlyphs)
	case 12:
		return c.processSubtype12(data, numGlyphs)
	case 13:
		return c.processSubtype13(data, numGlyphs)
	case 14:
		return c.processSubtype14(data, numGlyphs)
	default:
		return fmt.Errorf("ttf: Unknown cmap format:%d", subtableFormat)
	}
}

// processSubtype8 reads a mixed 16-bit and 32-bit mapping.
func (c *CmapSubtable) processSubtype8(data DataStream, numGlyphs int) error {
	r := newReader(data)
	is32, err := readUnsignedByteArray(data, 8192)
	if err != nil {
		return err
	}
	nbGroups := r.unsignedInt()
	if r.err != nil {
		return r.err
	}
	if nbGroups > 65536 {
		return fmt.Errorf("ttf: CMap ( Subtype8 ) is invalid")
	}
	c.glyphIDToCharacterCode = newGlyphIDToCharacterCode(numGlyphs)
	c.characterCodeToGlyphID = make(map[int]int, numGlyphs)
	if numGlyphs == 0 {
		// subtable has no glyphs
		return nil
	}
	for i := int64(0); i < nbGroups; i++ {
		firstCode := r.unsignedInt()
		endCode := r.unsignedInt()
		startGlyph := r.unsignedInt()
		if r.err != nil {
			return r.err
		}
		if firstCode > endCode || 0 > firstCode {
			return fmt.Errorf("ttf: Range invalid")
		}
		for j := firstCode; j <= endCode; j++ {
			if j > maxInt32 {
				return fmt.Errorf("ttf: [Sub Format 8] Invalid character code %d", j)
			}
			if int(j)/8 >= len(is32) {
				return fmt.Errorf("ttf: [Sub Format 8] Invalid character code %d", j)
			}
			var currentCharCode int
			if is32[int(j)/8]&(1<<(int(j)%8)) == 0 {
				currentCharCode = int(j)
			} else {
				lead := int64(leadOffset) + (j >> 10)
				trail := int64(0xDC00) + (j & 0x3FF)
				codepoint := (lead << 10) + trail + surrogateOffset
				if codepoint > maxInt32 {
					return fmt.Errorf("ttf: [Sub Format 8] Invalid character code %d", codepoint)
				}
				currentCharCode = int(codepoint)
			}
			glyphIndex := startGlyph + (j - firstCode)
			if glyphIndex > int64(numGlyphs) || glyphIndex > maxInt32 {
				return fmt.Errorf("ttf: CMap contains an invalid glyph index")
			}
			c.glyphIDToCharacterCode[int(glyphIndex)] = currentCharCode
			c.characterCodeToGlyphID[currentCharCode] = int(glyphIndex)
		}
	}
	return nil
}

// processSubtype10 reads a trimmed 32-bit mapping, which Java validates and
// then does nothing with.
func (c *CmapSubtable) processSubtype10(data DataStream, numGlyphs int) error {
	r := newReader(data)
	startCode := r.unsignedInt()
	numChars := r.unsignedInt()
	if r.err != nil {
		return r.err
	}
	if numChars > maxInt32 {
		return fmt.Errorf("ttf: Invalid number of Characters")
	}
	if startCode < 0 || startCode > 0x0010FFFF || (startCode+numChars) > 0x0010FFFF ||
		((startCode+numChars) >= 0x0000D800 && (startCode+numChars) <= 0x0000DFFF) {
		return fmt.Errorf("ttf: Invalid character codes, startCode: 0x%X, numChars: %d",
			startCode, numChars)
	}
	return nil
}

// processSubtype12 reads a segmented 32-bit mapping.
func (c *CmapSubtable) processSubtype12(data DataStream, numGlyphs int) error {
	r := newReader(data)
	maxGlyphID := 0
	nbGroups := r.unsignedInt()
	if r.err != nil {
		return r.err
	}
	c.glyphIDToCharacterCode = newGlyphIDToCharacterCode(numGlyphs)
	c.characterCodeToGlyphID = make(map[int]int, numGlyphs)
	if numGlyphs == 0 {
		// subtable has no glyphs
		return nil
	}
	for i := int64(0); i < nbGroups; i++ {
		firstCode := r.unsignedInt()
		endCode := r.unsignedInt()
		startGlyph := r.unsignedInt()
		if r.err != nil {
			return r.err
		}
		if firstCode < 0 || firstCode > 0x0010FFFF ||
			firstCode >= 0x0000D800 && firstCode <= 0x0000DFFF {
			return fmt.Errorf("ttf: Invalid character code 0x%X", firstCode)
		}
		if endCode > 0 && endCode < firstCode ||
			endCode > 0x0010FFFF ||
			endCode >= 0x0000D800 && endCode <= 0x0000DFFF {
			return fmt.Errorf("ttf: Invalid character code 0x%X", endCode)
		}
		for j := int64(0); j <= endCode-firstCode; j++ {
			glyphIndex := startGlyph + j
			if glyphIndex >= int64(numGlyphs) {
				// Format 12 cmap contains an invalid glyph index
				break
			}
			// a character beyond UCS-4 is kept, as it is in Java
			maxGlyphID = max(maxGlyphID, int(glyphIndex))
			c.characterCodeToGlyphID[int(firstCode+j)] = int(glyphIndex)
		}
	}
	c.buildGlyphIDToCharacterCodeLookup(maxGlyphID)
	return nil
}

// processSubtype13 reads a many-to-one 32-bit mapping.
func (c *CmapSubtable) processSubtype13(data DataStream, numGlyphs int) error {
	r := newReader(data)
	nbGroups := r.unsignedInt()
	if r.err != nil {
		return r.err
	}
	c.glyphIDToCharacterCode = newGlyphIDToCharacterCode(numGlyphs)
	c.characterCodeToGlyphID = make(map[int]int, numGlyphs)
	if numGlyphs == 0 {
		// subtable has no glyphs
		return nil
	}
	for i := int64(0); i < nbGroups; i++ {
		firstCode := r.unsignedInt()
		endCode := r.unsignedInt()
		glyphID := r.unsignedInt()
		if r.err != nil {
			return r.err
		}
		if glyphID > int64(numGlyphs) {
			// Format 13 cmap contains an invalid glyph index
			break
		}
		if firstCode < 0 || firstCode > 0x0010FFFF ||
			(firstCode >= 0x0000D800 && firstCode <= 0x0000DFFF) {
			return fmt.Errorf("ttf: Invalid character code 0x%X", firstCode)
		}
		if (endCode > 0 && endCode < firstCode) || endCode > 0x0010FFFF ||
			(endCode >= 0x0000D800 && endCode <= 0x0000DFFF) {
			return fmt.Errorf("ttf: Invalid character code 0x%X", endCode)
		}
		for j := int64(0); j <= endCode-firstCode; j++ {
			if firstCode+j > maxInt32 {
				return fmt.Errorf("ttf: Character Code greater than Integer.MAX_VALUE")
			}
			// a character beyond UCS-4 is kept, as it is in Java
			c.glyphIDToCharacterCode[int(glyphID)] = int(firstCode + j)
			c.characterCodeToGlyphID[int(firstCode+j)] = int(glyphID)
		}
	}
	return nil
}

// processSubtype14 would read a variation sequence mapping, which Java ignores.
func (c *CmapSubtable) processSubtype14(data DataStream, numGlyphs int) error {
	// Format 14 cmap table is not supported and will be ignored
	return nil
}

// processSubtype6 reads a trimmed 16-bit mapping.
func (c *CmapSubtable) processSubtype6(data DataStream, numGlyphs int) error {
	r := newReader(data)
	firstCode := r.unsignedShort()
	entryCount := r.unsignedShort()
	if r.err != nil {
		return r.err
	}
	if entryCount == 0 {
		return nil
	}
	c.characterCodeToGlyphID = make(map[int]int, numGlyphs)
	glyphIDArray := r.unsignedShortArray(entryCount)
	if r.err != nil {
		return r.err
	}
	maxGlyphID := 0
	for i := 0; i < entryCount; i++ {
		maxGlyphID = max(maxGlyphID, glyphIDArray[i])
		c.characterCodeToGlyphID[firstCode+i] = glyphIDArray[i]
	}
	c.buildGlyphIDToCharacterCodeLookup(maxGlyphID)
	return nil
}

// processSubtype4 reads a segmented 16-bit mapping, which is the format nearly
// every font uses.
func (c *CmapSubtable) processSubtype4(data DataStream, numGlyphs int) error {
	r := newReader(data)
	segCountX2 := r.unsignedShort()
	segCount := segCountX2 / 2
	_ = r.unsignedShort() // searchRange
	_ = r.unsignedShort() // entrySelector
	_ = r.unsignedShort() // rangeShift
	endCount := r.unsignedShortArray(segCount)
	_ = r.unsignedShort() // reservedPad
	startCount := r.unsignedShortArray(segCount)
	idDelta := r.unsignedShortArray(segCount)
	idRangeOffsetPosition := r.position()
	idRangeOffset := r.unsignedShortArray(segCount)
	if r.err != nil {
		return r.err
	}

	c.characterCodeToGlyphID = make(map[int]int, numGlyphs)
	maxGlyphID := 0

	for i := 0; i < segCount; i++ {
		start := startCount[i]
		end := endCount[i]
		if start != 65535 && end != 65535 {
			delta := idDelta[i]
			rangeOffset := idRangeOffset[i]
			segmentRangeOffset := idRangeOffsetPosition + int64(i)*2 + int64(rangeOffset)
			for j := start; j <= end; j++ {
				if rangeOffset == 0 {
					glyphid := (j + delta) & 0xFFFF
					maxGlyphID = max(glyphid, maxGlyphID)
					c.characterCodeToGlyphID[j] = glyphid
				} else {
					glyphOffset := segmentRangeOffset + int64(j-start)*2
					r.seek(glyphOffset)
					glyphIndex := r.unsignedShort()
					if r.err != nil {
						return r.err
					}
					if glyphIndex != 0 {
						glyphIndex = (glyphIndex + delta) & 0xFFFF
						maxGlyphID = max(glyphIndex, maxGlyphID)
						c.characterCodeToGlyphID[j] = glyphIndex
					}
				}
			}
		}
	}

	if len(c.characterCodeToGlyphID) == 0 {
		// cmap format 4 subtable is empty
		return nil
	}
	c.buildGlyphIDToCharacterCodeLookup(maxGlyphID)
	return nil
}

// buildGlyphIDToCharacterCodeLookup turns the code-to-glyph map round.
func (c *CmapSubtable) buildGlyphIDToCharacterCodeLookup(maxGlyphID int) {
	c.glyphIDToCharacterCode = newGlyphIDToCharacterCode(maxGlyphID + 1)
	// Java walks a HashMap, whose order is unspecified; the port walks the keys
	// in order, so that a font always reads back the same way.
	keys := make([]int, 0, len(c.characterCodeToGlyphID))
	for key := range c.characterCodeToGlyphID {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	for _, key := range keys {
		value := c.characterCodeToGlyphID[key]
		if c.glyphIDToCharacterCode[value] == -1 {
			// add new value to the array
			c.glyphIDToCharacterCode[value] = key
		} else {
			// there is already a mapping for the given glyphId
			mappedValues, ok := c.glyphIDToCharacterCodeMultiple[value]
			if !ok {
				mappedValues = make([]int, 0, 2)
				mappedValues = append(mappedValues, c.glyphIDToCharacterCode[value])
				// mark value as multiple mapping
				c.glyphIDToCharacterCode[value] = minInt32
			}
			mappedValues = append(mappedValues, key)
			c.glyphIDToCharacterCodeMultiple[value] = mappedValues
		}
	}
}

// subHeader is one entry of the high-byte table of a format 2 subtable.
type subHeader struct {
	firstCode     int
	entryCount    int
	idDelta       int16
	idRangeOffset int
}

// processSubtype2 reads a high-byte mapping, which the CJK legacy encodings
// use.
func (c *CmapSubtable) processSubtype2(data DataStream, numGlyphs int) error {
	r := newReader(data)
	subHeaderKeys := make([]int, 256)
	// ---- keep the Max Index of the SubHeader array to know its length
	maxSubHeaderIndex := 0
	for i := 0; i < 256; i++ {
		subHeaderKeys[i] = r.unsignedShort()
		if r.err != nil {
			return r.err
		}
		maxSubHeaderIndex = max(maxSubHeaderIndex, subHeaderKeys[i]/8)
	}

	// ---- Read all SubHeaders to avoid useless seek on DataSource
	subHeaders := make([]subHeader, maxSubHeaderIndex+1)
	for i := 0; i <= maxSubHeaderIndex; i++ {
		firstCode := r.unsignedShort()
		entryCount := r.unsignedShort()
		idDelta := r.signedShort()
		idRangeOffset := r.unsignedShort() - (maxSubHeaderIndex+1-i-1)*8 - 2
		if r.err != nil {
			return r.err
		}
		subHeaders[i] = subHeader{
			firstCode:     firstCode,
			entryCount:    entryCount,
			idDelta:       idDelta,
			idRangeOffset: idRangeOffset,
		}
	}
	startGlyphIndexOffset := r.position()
	c.glyphIDToCharacterCode = newGlyphIDToCharacterCode(numGlyphs)
	c.characterCodeToGlyphID = make(map[int]int, numGlyphs)
	if numGlyphs == 0 {
		// subtable has no glyphs
		return nil
	}

	for i := 0; i <= maxSubHeaderIndex; i++ {
		sh := subHeaders[i]
		firstCode := sh.firstCode
		idRangeOffset := sh.idRangeOffset
		idDelta := int(sh.idDelta)
		entryCount := sh.entryCount
		r.seek(startGlyphIndexOffset + int64(idRangeOffset))
		for j := 0; j < entryCount; j++ {
			// ---- compute the Character Code
			charCode := i
			charCode = (charCode << 8) + (firstCode + j)

			// ---- Go to the CharacterCOde position in the Sub Array
			// of the glyphIndexArray
			// glyphIndexArray contains Unsigned Short so add (j * 2) bytes
			// at the index position
			p := r.unsignedShort()
			if r.err != nil {
				return r.err
			}
			// ---- compute the glyphIndex
			if p > 0 {
				p = (p + idDelta) % 65536
				if p < 0 {
					p += 65536
				}
			}

			if p >= numGlyphs {
				// a glyphId past the end of the font is ignored
				continue
			}
			c.glyphIDToCharacterCode[p] = charCode
			c.characterCodeToGlyphID[charCode] = p
		}
	}
	return nil
}

// processSubtype0 reads a byte mapping, which covers the first 256 codes.
func (c *CmapSubtable) processSubtype0(data DataStream) error {
	glyphMapping, err := readBytes(data, 256)
	if err != nil {
		return err
	}
	c.glyphIDToCharacterCode = newGlyphIDToCharacterCode(256)
	c.characterCodeToGlyphID = make(map[int]int, len(glyphMapping))
	for i := 0; i < len(glyphMapping); i++ {
		glyphIndex := int(glyphMapping[i]) & 0xFF
		c.glyphIDToCharacterCode[glyphIndex] = i
		c.characterCodeToGlyphID[i] = glyphIndex
	}
	return nil
}

// newGlyphIDToCharacterCode returns an array with no glyph mapped yet.
func newGlyphIDToCharacterCode(size int) []int {
	gidToCode := make([]int, size)
	for i := range gidToCode {
		gidToCode[i] = -1
	}
	return gidToCode
}

// PlatformEncodingID returns which encoding of the platform the subtable is
// for.
func (c *CmapSubtable) PlatformEncodingID() int { return c.platformEncodingID }

// SetPlatformEncodingID sets which encoding of the platform the subtable is
// for.
func (c *CmapSubtable) SetPlatformEncodingID(platformEncodingID int) {
	c.platformEncodingID = platformEncodingID
}

// PlatformID returns which platform the subtable is for.
func (c *CmapSubtable) PlatformID() int { return c.platformID }

// SetPlatformID sets which platform the subtable is for.
func (c *CmapSubtable) SetPlatformID(platformID int) { c.platformID = platformID }

// GetGlyphID returns the glyph the given character code maps to, or zero where
// the subtable does not map it.
func (c *CmapSubtable) GetGlyphID(characterCode int) int {
	return c.characterCodeToGlyphID[characterCode]
}

// getCharCode returns the code mapped to the given glyph, -1 where none is, and
// the smallest negative int where several are.
func (c *CmapSubtable) getCharCode(gid int) int {
	if gid < 0 || c.glyphIDToCharacterCode == nil || gid >= len(c.glyphIDToCharacterCode) {
		return -1
	}
	return c.glyphIDToCharacterCode[gid]
}

// GetCharCodes returns every code mapped to the given glyph, in ascending
// order, or nil where none is.
func (c *CmapSubtable) GetCharCodes(gid int) []int {
	code := c.getCharCode(gid)
	if code == -1 {
		return nil
	}
	var codes []int
	if code == minInt32 {
		if mappedValues, ok := c.glyphIDToCharacterCodeMultiple[gid]; ok {
			codes = make([]int, len(mappedValues))
			copy(codes, mappedValues)
			// sort the list to provide a reliable order
			sort.Ints(codes)
		}
	} else {
		codes = []int{code}
	}
	return codes
}

// String returns the platform and encoding of the subtable.
func (c *CmapSubtable) String() string {
	return fmt.Sprintf("{%d %d}", c.PlatformID(), c.PlatformEncodingID())
}
