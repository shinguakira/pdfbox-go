package cmap

import (
	"bytes"
	"io"
	"log/slog"
)

// space is the Unicode mapping whose code CMap remembers separately.
const space = " "

// CMap represents a CMap file.
//
// Port of org.apache.fontbox.cmap.CMap.
//
// Java returns null from every lookup that misses, so each of those is a
// comma-ok method here: the second result is false where Java returns null.
type CMap struct {
	wmode       int
	cmapName    string
	cmapVersion string
	cmapType    int

	registry   string
	ordering   string
	supplement int

	minCodeLength int
	maxCodeLength int

	minCidLength int
	maxCidLength int

	// code lengths
	codespaceRanges []*CodespaceRange

	// Unicode mappings
	// one byte input values
	charToUnicodeOneByte map[int]string
	// two byte input values
	charToUnicodeTwoBytes map[int]string
	// 3 / 4 byte input values
	charToUnicodeMoreBytes map[int]string

	// CID mappings
	// map with all code to cid mappings organized by the origin byte length of
	// the input value
	codeToCid       map[int]map[int]int
	codeToCidRanges []*cidRange

	// inverted map
	unicodeToByteCodes map[string][]byte

	spaceMapping int
}

// newCMap creates a new instance of CMap.
func newCMap() *CMap {
	return &CMap{
		cmapType:               -1,
		minCodeLength:          4,
		minCidLength:           4,
		charToUnicodeOneByte:   map[int]string{},
		charToUnicodeTwoBytes:  map[int]string{},
		charToUnicodeMoreBytes: map[int]string{},
		codeToCid:              map[int]map[int]int{},
		unicodeToByteCodes:     map[string][]byte{},
		spaceMapping:           -1,
	}
}

// HasCIDMappings tells if this cmap has any CID mappings.
func (c *CMap) HasCIDMappings() bool {
	return len(c.codeToCid) != 0 || len(c.codeToCidRanges) != 0
}

// HasUnicodeMappings tells if this cmap has any Unicode mappings.
func (c *CMap) HasUnicodeMappings() bool {
	return len(c.charToUnicodeOneByte) != 0 || len(c.charToUnicodeTwoBytes) != 0 ||
		len(c.charToUnicodeMoreBytes) != 0
}

// ToUnicode returns the sequence of Unicode characters for the given character
// code -- more than one, for an "fi" ligature say -- the second result being
// false where there is no mapping.
//
// This method exists for convenience. It may return false values as the origin
// byte length of the input value is unknown and the mapping for some input
// values aren't unique.
//
// Example: the two byte value 0x00, 0x65 maps to 0x20. An input value of 0x65
// always returns 0x20 even if the value has an origin byte length of 1.
func (c *CMap) ToUnicode(code int) (string, bool) {
	if code < 256 {
		if unicode, ok := c.ToUnicodeLength(code, 1); ok {
			return unicode, true
		}
	}
	if code <= 0xFFFF {
		return c.ToUnicodeLength(code, 2)
	}
	if code <= 0xFFFFFF {
		return c.ToUnicodeLength(code, 3)
	}
	return c.ToUnicodeLength(code, 4)
}

// ToUnicodeLength returns the sequence of Unicode characters for the given
// character code of the given length.
func (c *CMap) ToUnicodeLength(code, length int) (string, bool) {
	if length == 1 {
		unicode, ok := c.charToUnicodeOneByte[code]
		return unicode, ok
	}
	if length == 2 {
		unicode, ok := c.charToUnicodeTwoBytes[code]
		return unicode, ok
	}
	unicode, ok := c.charToUnicodeMoreBytes[code]
	return unicode, ok
}

// ToUnicodeBytes returns the sequence of Unicode characters for the given
// character code.
func (c *CMap) ToUnicodeBytes(code []byte) (string, bool) {
	return c.ToUnicodeLength(ToInt(code), len(code))
}

// ReadCode reads a character code from a string in the content stream.
//
// See "CMap Mapping" and "Handling Undefined Characters" in PDF32000 for more
// details.
func (c *CMap) ReadCode(in *bytes.Reader) (int, error) {
	buf := make([]byte, c.maxCodeLength)
	// Java ignores the count this returns, leaving the untouched bytes zero.
	_, _ = in.Read(buf[0:c.minCodeLength])
	// Java marks the stream here, so that the reset below comes back to just
	// after the initial read.
	mark, err := in.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	for i := c.minCodeLength - 1; i < c.maxCodeLength; i++ {
		byteCount := i + 1
		for _, r := range c.codespaceRanges {
			if r.IsFullMatch(buf, byteCount) {
				return ToIntLen(buf, byteCount), nil
			}
		}
		if byteCount < c.maxCodeLength {
			// Java reads a single byte and casts it, so EOF -- which reads as
			// -1 -- becomes 0xFF.
			next, err := in.ReadByte()
			if err != nil {
				next = 0xFF
			}
			buf[byteCount] = next
		}
	}
	slog.Warn("Invalid character code sequence in CMap",
		"code", buf, "cmap", c.cmapName)
	// PDFBOX-4811 reposition to where we were after initial read
	if _, err := in.Seek(mark, io.SeekStart); err != nil {
		return 0, err
	}
	return ToIntLen(buf, c.minCodeLength), nil // Adobe Reader behavior
}

// ToCIDBytes returns the CID for the given character code.
func (c *CMap) ToCIDBytes(code []byte) int {
	if !c.HasCIDMappings() || len(code) < c.minCidLength || len(code) > c.maxCidLength {
		return 0
	}
	if codeToCidMap, ok := c.codeToCid[len(code)]; ok {
		if cid, ok := codeToCidMap[ToInt(code)]; ok {
			return cid
		}
	}
	return c.toCIDFromRangesBytes(code)
}

// ToCID returns the CID for the given character code.
//
// This method exists for convenience. It may return false values as the origin
// byte length of the input value is unknown and the mapping for some input
// values aren't unique.
//
// Example: the two byte value 0x00, 0x65 maps to 0x20. An input value of 0x65
// always returns 0x20 even if the value has an origin byte length of 1.
func (c *CMap) ToCID(code int) int {
	if !c.HasCIDMappings() {
		return 0
	}
	cid := 0
	length := c.minCidLength
	for cid == 0 && length <= c.maxCidLength {
		cid = c.ToCIDLength(code, length)
		length++
	}
	return cid
}

// ToCIDLength returns the CID for the given character code, length being the
// origin byte length of the code.
func (c *CMap) ToCIDLength(code, length int) int {
	if !c.HasCIDMappings() || length < c.minCidLength || length > c.maxCidLength {
		return 0
	}
	if codeToCidMap, ok := c.codeToCid[length]; ok {
		if cid, ok := codeToCidMap[code]; ok {
			return cid
		}
	}
	return c.toCIDFromRanges(code, length)
}

// toCIDFromRanges returns the CID for the given character code.
func (c *CMap) toCIDFromRanges(code, length int) int {
	for _, r := range c.codeToCidRanges {
		if ch := r.Map(code, length); ch != -1 {
			return ch
		}
	}
	return 0
}

// toCIDFromRangesBytes returns the CID for the given character code.
func (c *CMap) toCIDFromRangesBytes(code []byte) int {
	for _, r := range c.codeToCidRanges {
		if ch := r.MapBytes(code); ch != -1 {
			return ch
		}
	}
	return 0
}

// addCharMapping adds a character code to Unicode character sequence mapping.
func (c *CMap) addCharMapping(codes []byte, unicode string) {
	switch len(codes) {
	case 1:
		index, _ := GetIndexValue(codes)
		c.charToUnicodeOneByte[index] = unicode
		c.unicodeToByteCodes[unicode] = GetByteValue(codes)
	case 2:
		index, _ := GetIndexValue(codes)
		c.charToUnicodeTwoBytes[index] = unicode
		c.unicodeToByteCodes[unicode] = GetByteValue(codes)
	case 3, 4:
		c.charToUnicodeMoreBytes[ToInt(codes)] = unicode
		clone := make([]byte, len(codes))
		copy(clone, codes)
		c.unicodeToByteCodes[unicode] = clone
	default:
		slog.Warn("Mappings with more than 4 bytes aren't supported yet",
			"length", len(codes))
	}
	// fixme: ugly little hack
	if unicode == space {
		c.spaceMapping = ToInt(codes)
	}
}

// GetCodesFromUnicode returns the code bytes for a unicode string, the second
// result being false if there is none.
func (c *CMap) GetCodesFromUnicode(unicode string) ([]byte, bool) {
	codes, ok := c.unicodeToByteCodes[unicode]
	return codes, ok
}

// addCIDMapping adds a CID mapping.
func (c *CMap) addCIDMapping(code []byte, cid int) {
	codeToCidMap, ok := c.codeToCid[len(code)]
	if !ok {
		codeToCidMap = map[int]int{}
		c.codeToCid[len(code)] = codeToCidMap
		c.minCidLength = min(c.minCidLength, len(code))
		c.maxCidLength = max(c.maxCidLength, len(code))
	}
	codeToCidMap[ToInt(code)] = cid
}

// addCIDRange adds a CID range, from being the starting character of the range,
// to the ending one, and cid the CID to be started with.
func (c *CMap) addCIDRange(from, to []byte, cid int) {
	c.addCIDRangeInts(ToInt(from), ToInt(to), cid, len(from))
}

func (c *CMap) addCIDRangeInts(from, to, cid, length int) {
	var lastRange *cidRange
	if len(c.codeToCidRanges) != 0 {
		lastRange = c.codeToCidRanges[len(c.codeToCidRanges)-1]
	}
	if lastRange == nil || !lastRange.Extend(from, to, cid, length) {
		c.codeToCidRanges = append(c.codeToCidRanges, newCIDRange(from, to, cid, length))
		c.minCidLength = min(c.minCidLength, length)
		c.maxCidLength = max(c.maxCidLength, length)
	}
}

// addCodespaceRange adds a single codespace range.
func (c *CMap) addCodespaceRange(r *CodespaceRange) {
	c.codespaceRanges = append(c.codespaceRanges, r)
	c.maxCodeLength = max(c.maxCodeLength, r.CodeLength())
	c.minCodeLength = min(c.minCodeLength, r.CodeLength())
}

// useCmap is the implementation of the usecmap operator. This will copy all of
// the mappings from one cmap to another.
func (c *CMap) useCmap(cmap *CMap) {
	for _, r := range cmap.codespaceRanges {
		c.addCodespaceRange(r)
	}
	for k, v := range cmap.charToUnicodeOneByte {
		c.charToUnicodeOneByte[k] = v
	}
	for k, v := range cmap.charToUnicodeTwoBytes {
		c.charToUnicodeTwoBytes[k] = v
	}
	for k, v := range cmap.charToUnicodeMoreBytes {
		c.charToUnicodeMoreBytes[k] = v
	}
	for k, v := range cmap.charToUnicodeOneByte {
		// Java writes k % 0xFF here, not k & 0xFF, so the one code 255 comes
		// back out as 0. See migration/JAVA-BUGS.md entry 16; ported as it
		// stands.
		c.unicodeToByteCodes[v] = []byte{byte(k % 0xFF)}
	}
	for k, v := range cmap.charToUnicodeTwoBytes {
		c.unicodeToByteCodes[v] = []byte{
			byte((uint32(k) >> 8) & 0xFF), byte(k & 0xFF),
		}
	}
	for k, v := range cmap.charToUnicodeMoreBytes {
		var bar []byte
		if k <= 0xFFFFFF {
			// 3 bytes
			bar = []byte{
				byte((uint32(k) >> 16) & 0xFF), byte((uint32(k) >> 8) & 0xFF),
				byte(k & 0xFF),
			}
		} else {
			// 4 bytes
			bar = []byte{
				byte((uint32(k) >> 24) & 0xFF), byte((uint32(k) >> 16) & 0xFF),
				byte((uint32(k) >> 8) & 0xFF), byte(k & 0xFF),
			}
		}
		c.unicodeToByteCodes[v] = bar
	}
	for key, value := range cmap.codeToCid {
		existingMapping, ok := c.codeToCid[key]
		if !ok {
			// Java's putIfAbsent stores the very map it was handed, so the two
			// CMaps share it from here on.
			c.codeToCid[key] = value
			continue
		}
		for k, v := range value {
			existingMapping[k] = v
		}
	}
	c.codeToCidRanges = append(c.codeToCidRanges, cmap.codeToCidRanges...)
	c.maxCodeLength = max(c.maxCodeLength, cmap.maxCodeLength)
	c.minCodeLength = min(c.minCodeLength, cmap.minCodeLength)
	c.maxCidLength = max(c.maxCidLength, cmap.maxCidLength)
	c.minCidLength = min(c.minCidLength, cmap.minCidLength)
}

// WMode returns the WMode of a CMap, 0 representing a horizontal and 1 a
// vertical orientation.
func (c *CMap) WMode() int { return c.wmode }

// SetWMode sets the WMode of a CMap.
func (c *CMap) SetWMode(newWMode int) { c.wmode = newWMode }

// Name returns the name of the CMap.
func (c *CMap) Name() string { return c.cmapName }

// SetName sets the name of the CMap.
func (c *CMap) SetName(name string) { c.cmapName = name }

// Version returns the version of the CMap.
func (c *CMap) Version() string { return c.cmapVersion }

// SetVersion sets the version of the CMap.
func (c *CMap) SetVersion(version string) { c.cmapVersion = version }

// Type returns the type of the CMap.
func (c *CMap) Type() int { return c.cmapType }

// SetType sets the type of the CMap.
func (c *CMap) SetType(cmapType int) { c.cmapType = cmapType }

// Registry returns the registry of the CIDSystemInfo.
func (c *CMap) Registry() string { return c.registry }

// SetRegistry sets the registry of the CIDSystemInfo.
func (c *CMap) SetRegistry(newRegistry string) { c.registry = newRegistry }

// Ordering returns the ordering of the CIDSystemInfo.
func (c *CMap) Ordering() string { return c.ordering }

// SetOrdering sets the ordering of the CIDSystemInfo.
func (c *CMap) SetOrdering(newOrdering string) { c.ordering = newOrdering }

// Supplement returns the supplement of the CIDSystemInfo.
func (c *CMap) Supplement() int { return c.supplement }

// SetSupplement sets the supplement of the CIDSystemInfo.
func (c *CMap) SetSupplement(newSupplement int) { c.supplement = newSupplement }

// SpaceMapping returns the mapped code for the space character.
func (c *CMap) SpaceMapping() int { return c.spaceMapping }

// String returns the name of the CMap.
func (c *CMap) String() string { return c.cmapName }
