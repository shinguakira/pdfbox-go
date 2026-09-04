package cff

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

const (
	tagOTTO    = "OTTO"
	tagTTCF    = "ttcf"
	tagTTFOnly = "\x00\x01\x00\x00"
)

// FontHeadersSink is what ParseFirstSubFontROS reports into.
//
// Java passes org.apache.fontbox.ttf.FontHeaders here, which would make cff
// import ttf while ttf's CFFTable imports cff -- a cycle Java allows and Go
// does not. The parser only ever writes to that object, so the two methods it
// calls are declared as an interface and ttf.FontHeaders satisfies it.
type FontHeadersSink interface {
	// SetError records why the headers could not be read.
	SetError(message string)

	// SetOtfROS records the Registry, Ordering and Supplement of the first CFF
	// subfont.
	SetOtfROS(registry, ordering string, supplement int)
}

// Parser represents a parser for a CFF font.
//
// Port of org.apache.fontbox.cff.CFFParser.
type Parser struct {
	stringIndex []string
	source      ByteSource

	// for debugging only
	debugFontName string
}

// NewCFFParser returns a parser for a CFF font.
func NewCFFParser() *Parser { return &Parser{} }

// ParseFirstSubFontROS extracts "Registry", "Ordering" and "Supplement"
// properties from the first CFF subfont, putting the results in outHeaders.
func (p *Parser) ParseFirstSubFontROS(randomAccessRead pdfio.RandomAccessRead,
	outHeaders FontHeadersSink) error {
	// this method is a simplified and merged version of parse(RandomAccessRead)
	// > parse(DataInput) > parseFont(...)

	// start code from parse(RandomAccessRead)
	if _, err := randomAccessRead.Seek(0, io.SeekStart); err != nil {
		return err
	}
	var input DataInput = NewDataInputRandomAccessRead(randomAccessRead)

	// start code from parse(DataInput)
	input, err := p.skipHeader(input)
	if err != nil {
		return err
	}
	nameIndex, err := readStringIndexData(input)
	if err != nil {
		return err
	}
	if len(nameIndex) == 0 {
		outHeaders.SetError("Name index missing in CFF font")
		return nil
	}
	topDictIndex, err := readIndexData(input)
	if err != nil {
		return err
	}
	if len(topDictIndex) == 0 {
		outHeaders.SetError("Top DICT INDEX missing in CFF font")
		return nil
	}

	// 'stringIndex' is required by 'parseROS() > readString()'
	if p.stringIndex, err = readStringIndexData(input); err != nil {
		return err
	}

	// start code from parseFont(...)
	topDictInput := NewDataInputByteArray(topDictIndex[0])
	topDict, err := readDictData(topDictInput)
	if err != nil {
		return err
	}

	if topDict.getEntry("SyntheticBase") != nil {
		outHeaders.SetError("Synthetic Fonts are not supported")
		return nil
	}

	cffCIDFont, err := p.parseROS(topDict)
	if err != nil {
		return err
	}
	if cffCIDFont != nil {
		outHeaders.SetOtfROS(cffCIDFont.Registry(), cffCIDFont.Ordering(),
			cffCIDFont.Supplement())
	}
	return nil
}

// ParseBytes parses a CFF font using a byte array, also passing in a byte
// source for future use.
func (p *Parser) ParseBytes(bytes []byte, source ByteSource) ([]CFFFont, error) {
	// TODO do we need to store the source data of the font? It isn't used at all
	p.source = source
	return p.parse(NewDataInputByteArray(bytes))
}

// Parse parses a CFF font using a RandomAccessRead as input.
func (p *Parser) Parse(randomAccessRead pdfio.RandomAccessRead) ([]CFFFont, error) {
	// TODO do we need to store the source data of the font? It isn't used at all
	length, err := randomAccessRead.Length()
	if err != nil {
		return nil, err
	}
	bytes := make([]byte, int(length))
	if _, err := randomAccessRead.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	remainingBytes := len(bytes)
	for remainingBytes > 0 {
		amountRead, err := randomAccessRead.Read(bytes[len(bytes)-remainingBytes:])
		if amountRead <= 0 {
			if err != nil && !errors.Is(err, io.EOF) {
				return nil, err
			}
			break
		}
		remainingBytes -= amountRead
	}
	if _, err := randomAccessRead.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	p.source = &cffByteSource{bytes: bytes}
	return p.parse(NewDataInputRandomAccessRead(randomAccessRead))
}

func (p *Parser) skipHeader(input DataInput) (DataInput, error) {
	firstTag, err := readTagName(input)
	if err != nil {
		return nil, err
	}
	// try to determine which kind of font we have
	switch firstTag {
	case tagOTTO:
		if input, err = p.createTaggedCFFDataInput(input); err != nil {
			return nil, err
		}
	case tagTTCF:
		return nil, errors.New("True Type Collection fonts are not supported.")
	case tagTTFOnly:
		return nil, errors.New("OpenType fonts containing a true type font are not supported.")
	default:
		if err := input.SetPosition(0); err != nil {
			return nil, err
		}
	}

	if _, err := readHeader(input); err != nil {
		return nil, err
	}
	return input, nil
}

// parse parses a CFF font using a DataInput as input.
func (p *Parser) parse(input DataInput) ([]CFFFont, error) {
	input, err := p.skipHeader(input)
	if err != nil {
		return nil, err
	}
	nameIndex, err := readStringIndexData(input)
	if err != nil {
		return nil, err
	}
	if len(nameIndex) == 0 {
		return nil, errors.New("Name index missing in CFF font")
	}
	topDictIndex, err := readIndexData(input)
	if err != nil {
		return nil, err
	}
	if len(topDictIndex) == 0 {
		return nil, errors.New("Top DICT INDEX missing in CFF font")
	}

	if p.stringIndex, err = readStringIndexData(input); err != nil {
		return nil, err
	}
	globalSubrIndex, err := readIndexData(input)
	if err != nil {
		return nil, err
	}

	fonts := make([]CFFFont, 0, len(nameIndex))
	for i := 0; i < len(nameIndex); i++ {
		font, err := p.parseFont(input, nameIndex[i], topDictIndex[i])
		if err != nil {
			return nil, err
		}
		setGlobalSubrIndexOf(font, globalSubrIndex)
		setDataOf(font, p.source)
		fonts = append(fonts, font)
	}
	return fonts, nil
}

// setGlobalSubrIndexOf and setDataOf reach the package-private setters through
// whichever font the parse produced.
func setGlobalSubrIndexOf(font CFFFont, globalSubrIndex [][]byte) {
	switch f := font.(type) {
	case *CFFType1Font:
		f.setGlobalSubrIndex(globalSubrIndex)
	case *CFFCIDFont:
		f.setGlobalSubrIndex(globalSubrIndex)
	}
}

func setDataOf(font CFFFont, source ByteSource) {
	switch f := font.(type) {
	case *CFFType1Font:
		f.setData(source)
	case *CFFCIDFont:
		f.setData(source)
	}
}

func (p *Parser) createTaggedCFFDataInput(input DataInput) (DataInput, error) {
	// this is OpenType font containing CFF data
	// so find CFF tag
	numTables, err := input.ReadShort()
	if err != nil {
		return nil, err
	}
	// searchRange, entrySelector and rangeShift are read and discarded
	for i := 0; i < 3; i++ {
		if _, err := input.ReadShort(); err != nil {
			return nil, err
		}
	}
	for q := 0; q < int(numTables); q++ {
		tagName, err := readTagName(input)
		if err != nil {
			return nil, err
		}
		if _, err := readLong(input); err != nil { // checksum
			return nil, err
		}
		offset, err := readLong(input)
		if err != nil {
			return nil, err
		}
		length, err := readLong(input)
		if err != nil {
			return nil, err
		}
		if tagName == "CFF " {
			if err := input.SetPosition(int(offset)); err != nil {
				return nil, err
			}
			bytes2, err := input.ReadBytes(int(length))
			if err != nil {
				return nil, err
			}
			return NewDataInputByteArray(bytes2), nil
		}
	}
	return nil, errors.New("CFF tag not found in this OpenType font.")
}

func readTagName(input DataInput) (string, error) {
	b, err := input.ReadBytes(4)
	if err != nil {
		return "", err
	}
	return decodeLatin1(b), nil
}

// decodeLatin1 decodes ISO-8859-1, where every byte is the code point of the
// same value.
func decodeLatin1(data []byte) string {
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}

func readLong(input DataInput) (int64, error) {
	high, err := input.ReadUnsignedShort()
	if err != nil {
		return 0, err
	}
	low, err := input.ReadUnsignedShort()
	if err != nil {
		return 0, err
	}
	return int64(high<<16 | low), nil
}

func readOffSize(input DataInput) (int, error) {
	offSize, err := input.ReadUnsignedByte()
	if err != nil {
		return 0, err
	}
	if offSize < 1 || offSize > 4 {
		position, err := input.Position()
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("Illegal (< 1 or > 4) offSize value %d in CFF font at position %d",
			offSize, position-1)
	}
	return offSize, nil
}

// header holds the header of a CFF font.
type header struct {
	major   int
	minor   int
	hdrSize int
	offSize int
}

func (h header) String() string {
	return fmt.Sprintf("org.apache.fontbox.cff.CFFParser$Header[major=%d, minor=%d, hdrSize=%d, offSize=%d]",
		h.major, h.minor, h.hdrSize, h.offSize)
}

func readHeader(input DataInput) (header, error) {
	major, err := input.ReadUnsignedByte()
	if err != nil {
		return header{}, err
	}
	minor, err := input.ReadUnsignedByte()
	if err != nil {
		return header{}, err
	}
	hdrSize, err := input.ReadUnsignedByte()
	if err != nil {
		return header{}, err
	}
	offSize, err := readOffSize(input)
	if err != nil {
		return header{}, err
	}
	return header{major, minor, hdrSize, offSize}, nil
}

func readIndexDataOffsets(input DataInput) ([]int, error) {
	count, err := input.ReadUnsignedShort()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	offSize, err := readOffSize(input)
	if err != nil {
		return nil, err
	}
	length, err := input.Length()
	if err != nil {
		return nil, err
	}
	offsets := make([]int, count+1)
	for i := 0; i <= count; i++ {
		offset, err := input.ReadOffset(offSize)
		if err != nil {
			return nil, err
		}
		if offset > length {
			return nil, fmt.Errorf("illegal offset value %d in CFF font", offset)
		}
		offsets[i] = offset
	}
	return offsets, nil
}

func readIndexData(input DataInput) ([][]byte, error) {
	offsets, err := readIndexDataOffsets(input)
	if err != nil {
		return nil, err
	}
	if len(offsets) == 0 {
		return nil, nil
	}
	count := len(offsets) - 1
	indexDataValues := make([][]byte, count)
	for i := 0; i < count; i++ {
		length := offsets[i+1] - offsets[i]
		if indexDataValues[i], err = input.ReadBytes(length); err != nil {
			return nil, err
		}
	}
	return indexDataValues, nil
}

func readStringIndexData(input DataInput) ([]string, error) {
	offsets, err := readIndexDataOffsets(input)
	if err != nil {
		return nil, err
	}
	if len(offsets) == 0 {
		return nil, nil
	}
	count := len(offsets) - 1
	indexDataValues := make([]string, count)
	for i := 0; i < count; i++ {
		length := offsets[i+1] - offsets[i]
		if length < 0 {
			return nil, fmt.Errorf(
				"Negative index data length + %d at %d: offsets[%d]=%d, offsets[%d]=%d",
				length, i, i+1, offsets[i+1], i, offsets[i])
		}
		bytes, err := input.ReadBytes(length)
		if err != nil {
			return nil, err
		}
		indexDataValues[i] = decodeLatin1(bytes)
	}
	return indexDataValues, nil
}

func readDictData(input DataInput) (*dictData, error) {
	dict := newDictData()
	for {
		hasRemaining, err := input.HasRemaining()
		if err != nil {
			return nil, err
		}
		if !hasRemaining {
			return dict, nil
		}
		entry, err := readEntry(input)
		if err != nil {
			return nil, err
		}
		dict.add(entry)
	}
}

func readDictDataAt(input DataInput, offset, dictSize int) (*dictData, error) {
	dict := newDictData()
	if dictSize > 0 {
		if err := input.SetPosition(offset); err != nil {
			return nil, err
		}
		endPosition := offset + dictSize
		for {
			position, err := input.Position()
			if err != nil {
				return nil, err
			}
			if position >= endPosition {
				return dict, nil
			}
			entry, err := readEntry(input)
			if err != nil {
				return nil, err
			}
			dict.add(entry)
		}
	}
	return dict, nil
}

func readEntry(input DataInput) (*dictEntry, error) {
	entry := &dictEntry{}
	for {
		b0, err := input.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}

		switch {
		case b0 >= 0 && b0 <= 21:
			if entry.operatorName, entry.hasOperator, err = readOperator(input, b0); err != nil {
				return nil, err
			}
			return entry, nil
		case b0 == 28 || b0 == 29:
			number, err := readIntegerNumber(input, b0)
			if err != nil {
				return nil, err
			}
			entry.operands = append(entry.operands, number)
		case b0 == 30:
			number, err := readRealNumber(input)
			if err != nil {
				return nil, err
			}
			entry.operands = append(entry.operands, number)
		case b0 >= 32 && b0 <= 254:
			number, err := readIntegerNumber(input, b0)
			if err != nil {
				return nil, err
			}
			entry.operands = append(entry.operands, number)
		default:
			return nil, fmt.Errorf("invalid DICT data b0 byte: %d", b0)
		}
	}
}

func readOperator(input DataInput, b0 int) (string, bool, error) {
	if b0 == 12 {
		b1, err := input.ReadUnsignedByte()
		if err != nil {
			return "", false, err
		}
		name, ok := GetOperator2(b0, b1)
		return name, ok, nil
	}
	name, ok := GetOperator(b0)
	return name, ok, nil
}

func readIntegerNumber(input DataInput, b0 int) (int, error) {
	switch {
	case b0 == 28:
		value, err := input.ReadShort()
		if err != nil {
			return 0, err
		}
		return int(value), nil
	case b0 == 29:
		value, err := input.ReadInt()
		if err != nil {
			return 0, err
		}
		return int(value), nil
	case b0 >= 32 && b0 <= 246:
		return b0 - 139, nil
	case b0 >= 247 && b0 <= 250:
		b1, err := input.ReadUnsignedByte()
		if err != nil {
			return 0, err
		}
		return (b0-247)*256 + b1 + 108, nil
	case b0 >= 251 && b0 <= 254:
		b1, err := input.ReadUnsignedByte()
		if err != nil {
			return 0, err
		}
		return -(b0-251)*256 - b1 - 108, nil
	}
	// Java throws IllegalArgumentException, which the callers above never
	// reach: every b0 they pass is inside one of the ranges.
	panic("cff: illegal integer number")
}

func readRealNumber(input DataInput) (float64, error) {
	var sb strings.Builder
	done := false
	exponentMissing := false
	hasExponent := false
	nibbles := make([]int, 2)
	for !done {
		b, err := input.ReadUnsignedByte()
		if err != nil {
			return 0, err
		}
		nibbles[0] = b / 16
		nibbles[1] = b % 16
		for _, nibble := range nibbles {
			switch nibble {
			case 0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9:
				sb.WriteString(strconv.Itoa(nibble))
				exponentMissing = false
			case 0xa:
				sb.WriteByte('.')
			case 0xb:
				if hasExponent {
					slog.Warn("duplicate 'E' ignored", "after", sb.String())
					break
				}
				sb.WriteByte('E')
				exponentMissing = true
				hasExponent = true
			case 0xc:
				if hasExponent {
					slog.Warn("duplicate 'E-' ignored", "after", sb.String())
					break
				}
				sb.WriteString("E-")
				exponentMissing = true
				hasExponent = true
			case 0xd:
			case 0xe:
				sb.WriteByte('-')
			case 0xf:
				done = true
			default:
				// can only be a programming error because a nibble is between
				// 0 and F
				panic(fmt.Sprintf("illegal nibble %d", nibble))
			}
		}
	}
	if exponentMissing {
		// the exponent is missing, just append "0" to avoid an exception
		// not sure if 0 is the correct value, but it seems to fit
		// see PDFBOX-1522
		sb.WriteByte('0')
	}
	if sb.Len() == 0 {
		return 0, nil
	}
	value, err := strconv.ParseFloat(sb.String(), 64)
	if err != nil {
		return 0, fmt.Errorf("cff: %w", err)
	}
	return value, nil
}

// parseROS extracts Registry, Ordering and Supplement from topDict["ROS"].
func (p *Parser) parseROS(topDict *dictData) (*CFFCIDFont, error) {
	// determine if this is a Type 1-equivalent font or a CIDFont
	rosEntry := topDict.getEntry("ROS")
	if rosEntry == nil {
		return nil, nil
	}
	if rosEntry.size() < 3 {
		return nil, errors.New("ROS entry must have 3 elements")
	}
	cffCIDFont := NewCFFCIDFont()
	registry, err := p.readString(numberInt(rosEntry.getNumber(0)))
	if err != nil {
		return nil, err
	}
	cffCIDFont.setRegistry(registry)
	ordering, err := p.readString(numberInt(rosEntry.getNumber(1)))
	if err != nil {
		return nil, err
	}
	cffCIDFont.setOrdering(ordering)
	cffCIDFont.setSupplement(numberInt(rosEntry.getNumber(2)))
	return cffCIDFont, nil
}

func (p *Parser) parseFont(input DataInput, name string, topDictIndex []byte) (CFFFont, error) {
	// top dict
	topDictInput := NewDataInputByteArray(topDictIndex)
	topDict, err := readDictData(topDictInput)
	if err != nil {
		return nil, err
	}

	// we don't support synthetic fonts
	if topDict.getEntry("SyntheticBase") != nil {
		return nil, errors.New("Synthetic Fonts are not supported")
	}

	// determine if this is a Type 1-equivalent font or a CIDFont
	cffCIDFont, err := p.parseROS(topDict)
	if err != nil {
		return nil, err
	}
	isCIDFont := cffCIDFont != nil
	var font CFFFont
	var base *cffFontBase
	var type1Font *CFFType1Font
	if isCIDFont {
		font = cffCIDFont
		base = &cffCIDFont.cffFontBase
	} else {
		type1Font = NewCFFType1Font()
		font = type1Font
		base = &type1Font.cffFontBase
	}

	// name
	p.debugFontName = name
	base.setName(name)

	// top dict
	for _, key := range []string{"version", "Notice", "Copyright", "FullName", "FamilyName",
		"Weight"} {
		value, err := p.getString(topDict, key)
		if err != nil {
			return nil, err
		}
		base.AddValueToTopDict(key, value)
	}
	base.AddValueToTopDict("isFixedPitch", topDict.getBoolean("isFixedPitch", false))
	base.AddValueToTopDict("ItalicAngle", topDict.getNumber("ItalicAngle", 0))
	base.AddValueToTopDict("UnderlinePosition", topDict.getNumber("UnderlinePosition", -100))
	base.AddValueToTopDict("UnderlineThickness", topDict.getNumber("UnderlineThickness", 50))
	base.AddValueToTopDict("PaintType", topDict.getNumber("PaintType", 0))
	base.AddValueToTopDict("CharstringType", topDict.getNumber("CharstringType", 2))
	base.AddValueToTopDict("FontMatrix", topDict.getArray("FontMatrix",
		[]any{0.001, 0.0, 0.0, 0.001, 0.0, 0.0}))
	base.AddValueToTopDict("UniqueID", topDict.getNumber("UniqueID", nil))
	base.AddValueToTopDict("FontBBox", topDict.getArray("FontBBox", []any{0, 0, 0, 0}))
	base.AddValueToTopDict("StrokeWidth", topDict.getNumber("StrokeWidth", 0))
	base.AddValueToTopDict("XUID", topDict.getArray("XUID", nil))

	// charstrings index
	charStringsEntry := topDict.getEntry("CharStrings")
	if charStringsEntry == nil || !charStringsEntry.hasOperands() {
		return nil, errors.New("CharStrings is missing or empty")
	}
	charStringsOffset := numberInt(charStringsEntry.getNumber(0))
	if err := input.SetPosition(charStringsOffset); err != nil {
		return nil, err
	}
	charStringsIndex, err := readIndexData(input)
	if err != nil {
		return nil, err
	}

	// charset
	charsetEntry := topDict.getEntry("charset")
	var charset CFFCharset
	if charsetEntry != nil && charsetEntry.hasOperands() {
		charsetId := numberInt(charsetEntry.getNumber(0))
		switch {
		case !isCIDFont && charsetId == 0:
			charset = CFFISOAdobeCharset
		case !isCIDFont && charsetId == 1:
			charset = CFFExpertCharset
		case !isCIDFont && charsetId == 2:
			charset = CFFExpertSubsetCharset
		case len(charStringsIndex) > 0:
			if err := input.SetPosition(charsetId); err != nil {
				return nil, err
			}
			if charset, err = p.readCharset(input, len(charStringsIndex), isCIDFont); err != nil {
				return nil, err
			}
		default:
			// that should not happen
			slog.Debug("Couldn't read CharStrings index - returning empty charset instead")
			charset = newEmptyCharsetType1()
		}
	} else if isCIDFont {
		// a CID font with no charset does not default to any predefined charset
		charset = newEmptyCharsetCID(len(charStringsIndex))
	} else {
		charset = CFFISOAdobeCharset
	}
	base.setCharset(charset)

	// charstrings dict
	base.charStrings = charStringsIndex

	// format-specific dictionaries
	if isCIDFont {
		// CharStrings index could be null if the index data couldn't be read
		numEntries := 0
		if len(charStringsIndex) == 0 {
			slog.Debug("Couldn't read CharStrings index - parsing CIDFontDicts with number " +
				"of char strings set to 0")
		} else {
			numEntries = len(charStringsIndex)
		}

		if err := p.parseCIDFontDicts(input, topDict, cffCIDFont, numEntries); err != nil {
			return nil, err
		}

		fontDicts := cffCIDFont.FontDicts()
		var privMatrix []any
		if len(fontDicts) != 0 {
			privMatrix, _ = fontDicts[0]["FontMatrix"].([]any)
		}
		// some malformed fonts have FontMatrix in their Font DICT, see PDFBOX-2495
		matrix := topDict.getArray("FontMatrix", nil)
		if matrix == nil {
			if privMatrix != nil {
				base.AddValueToTopDict("FontMatrix", privMatrix)
			} else {
				// default
				base.AddValueToTopDict("FontMatrix", topDict.getArray("FontMatrix",
					[]any{0.001, 0.0, 0.0, 0.001, 0.0, 0.0}))
			}
		} else if privMatrix != nil {
			// we have to multiply the font matrix from the top directory with
			// the font matrix from the private directory. This should be done
			// for synthetic fonts only but in case of PDFBOX-3579 it's needed
			// as well to get the right scaling
			concatenateMatrix(matrix, privMatrix)
		}
	} else if err := p.parseType1Dicts(input, topDict, type1Font, charset); err != nil {
		return nil, err
	}

	return font, nil
}

func concatenateMatrix(matrixDest, matrixConcat []any) {
	// concatenate matrices
	// (a b 0)
	// (c d 0)
	// (x y 1)
	a1 := numberDouble(matrixDest[0])
	b1 := numberDouble(matrixDest[1])
	c1 := numberDouble(matrixDest[2])
	d1 := numberDouble(matrixDest[3])
	x1 := numberDouble(matrixDest[4])
	y1 := numberDouble(matrixDest[5])

	a2 := numberDouble(matrixConcat[0])
	b2 := numberDouble(matrixConcat[1])
	c2 := numberDouble(matrixConcat[2])
	d2 := numberDouble(matrixConcat[3])
	x2 := numberDouble(matrixConcat[4])
	y2 := numberDouble(matrixConcat[5])

	matrixDest[0] = a1*a2 + b1*c2
	// Java writes b1 * d1 here, where the other five rows use the second
	// matrix throughout. See migration/JAVA-BUGS.md entry 17; ported as it
	// stands.
	matrixDest[1] = a1*b2 + b1*d1
	matrixDest[2] = c1*a2 + d1*c2
	matrixDest[3] = c1*b2 + d1*d2
	matrixDest[4] = x1*a2 + y1*c2 + x2
	matrixDest[5] = x1*b2 + y1*d2 + y2
}

// numberDouble is Java's Number.doubleValue.
func numberDouble(entry any) float64 {
	switch v := entry.(type) {
	case int:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	}
	panic("cff: not a number")
}

// parseCIDFontDicts parses dictionaries specific to a CIDFont.
func (p *Parser) parseCIDFontDicts(input DataInput, topDict *dictData, font *CFFCIDFont,
	nrOfcharStrings int) error {
	// In a CIDKeyed Font, the Private dictionary isn't in the Top Dict but in
	// the Font dict which can be accessed by a lookup using FDArray and FDSelect
	fdArrayEntry := topDict.getEntry("FDArray")
	if fdArrayEntry == nil || !fdArrayEntry.hasOperands() {
		return errors.New("FDArray is missing for a CIDKeyed Font.")
	}

	// font dict index
	fontDictOffset := numberInt(fdArrayEntry.getNumber(0))
	if err := input.SetPosition(fontDictOffset); err != nil {
		return err
	}
	fdIndex, err := readIndexData(input)
	if err != nil {
		return err
	}
	if len(fdIndex) == 0 {
		return errors.New("Font dict index is missing for a CIDKeyed Font")
	}

	var privateDictionaries []map[string]any
	var fontDictionaries []map[string]any

	privateDictPopulated := false
	for _, bytes := range fdIndex {
		fontDictInput := NewDataInputByteArray(bytes)
		fontDict, err := readDictData(fontDictInput)
		if err != nil {
			return err
		}

		// font dict
		fontName, err := p.getString(fontDict, "FontName")
		if err != nil {
			return err
		}
		fontDictMap := map[string]any{
			"FontName":   fontName,
			"FontType":   fontDict.getNumber("FontType", 0),
			"FontBBox":   fontDict.getArray("FontBBox", nil),
			"FontMatrix": fontDict.getArray("FontMatrix", nil),
		}
		// TODO OD-4 : Add here other keys
		fontDictionaries = append(fontDictionaries, fontDictMap)

		// read private dict
		privateEntry := fontDict.getEntry("Private")
		if privateEntry == nil || privateEntry.size() < 2 {
			// PDFBOX-5843 don't abort here, and don't skip empty bytes entries,
			// because getLocalSubrIndex() expects subr at a specific index
			privateDictionaries = append(privateDictionaries, map[string]any{})
			continue
		}

		privateOffset := numberInt(privateEntry.getNumber(1))
		privateSize := numberInt(privateEntry.getNumber(0))
		privateDict, err := readDictDataAt(input, privateOffset, privateSize)
		if err != nil {
			return err
		}

		// populate private dict
		privateDictPopulated = true
		privDict, _ := readPrivateDict(privateDict)
		privateDictionaries = append(privateDictionaries, privDict)

		// local subrs
		localSubrOffset := privateDict.getNumber("Subrs", 0)
		if offset, ok := localSubrOffset.(int); ok && offset > 0 {
			if err := input.SetPosition(privateOffset + offset); err != nil {
				return err
			}
			subrs, err := readIndexData(input)
			if err != nil {
				return err
			}
			privDict["Subrs"] = subrs
		}
	}

	if !privateDictPopulated {
		return errors.New(`Font DICT invalid without "Private" entry`)
	}

	// font-dict (FD) select
	fdSelectEntry := topDict.getEntry("FDSelect")
	if fdSelectEntry == nil || !fdSelectEntry.hasOperands() {
		return errors.New("FDSelect is missing or empty")
	}
	fdSelectPos := numberInt(fdSelectEntry.getNumber(0))
	if err := input.SetPosition(fdSelectPos); err != nil {
		return err
	}
	fdSelect, err := readFDSelect(input, nrOfcharStrings)
	if err != nil {
		return err
	}

	// TODO almost certainly erroneous - CIDFonts do not have a top-level private dict
	// font.addValueToPrivateDict("defaultWidthX", 1000);
	// font.addValueToPrivateDict("nominalWidthX", 0);

	font.setFontDict(fontDictionaries)
	font.setPrivDict(privateDictionaries)
	font.setFdSelect(fdSelect)
	return nil
}

// privateDictKeys is the order readPrivateDict fills the map in, which Java's
// LinkedHashMap keeps.
var privateDictKeys = []string{
	"BlueValues", "OtherBlues", "FamilyBlues", "FamilyOtherBlues", "BlueScale", "BlueShift",
	"BlueFuzz", "StdHW", "StdVW", "StemSnapH", "StemSnapV", "ForceBold", "LanguageGroup",
	"ExpansionFactor", "initialRandomSeed", "defaultWidthX", "nominalWidthX",
}

func readPrivateDict(privateDict *dictData) (map[string]any, []string) {
	privDict := map[string]any{
		"BlueValues":        privateDict.getDelta("BlueValues", nil),
		"OtherBlues":        privateDict.getDelta("OtherBlues", nil),
		"FamilyBlues":       privateDict.getDelta("FamilyBlues", nil),
		"FamilyOtherBlues":  privateDict.getDelta("FamilyOtherBlues", nil),
		"BlueScale":         privateDict.getNumber("BlueScale", 0.039625),
		"BlueShift":         privateDict.getNumber("BlueShift", 7),
		"BlueFuzz":          privateDict.getNumber("BlueFuzz", 1),
		"StdHW":             privateDict.getNumber("StdHW", nil),
		"StdVW":             privateDict.getNumber("StdVW", nil),
		"StemSnapH":         privateDict.getDelta("StemSnapH", nil),
		"StemSnapV":         privateDict.getDelta("StemSnapV", nil),
		"ForceBold":         privateDict.getBoolean("ForceBold", false),
		"LanguageGroup":     privateDict.getNumber("LanguageGroup", 0),
		"ExpansionFactor":   privateDict.getNumber("ExpansionFactor", 0.06),
		"initialRandomSeed": privateDict.getNumber("initialRandomSeed", 0),
		"defaultWidthX":     privateDict.getNumber("defaultWidthX", 0),
		"nominalWidthX":     privateDict.getNumber("nominalWidthX", 0),
	}
	return privDict, privateDictKeys
}

// parseType1Dicts parses dictionaries specific to a Type 1-equivalent font.
func (p *Parser) parseType1Dicts(input DataInput, topDict *dictData, font *CFFType1Font,
	charset CFFCharset) error {
	// encoding
	encodingEntry := topDict.getEntry("Encoding")
	encodingId := 0
	if encodingEntry != nil && encodingEntry.hasOperands() {
		encodingId = numberInt(encodingEntry.getNumber(0))
	}
	var encoding *CFFEncoding
	switch encodingId {
	case 0:
		encoding = CFFStandardEncoding
	case 1:
		encoding = CFFExpertEncoding
	default:
		if err := input.SetPosition(encodingId); err != nil {
			return err
		}
		var err error
		if encoding, err = p.readEncoding(input, charset); err != nil {
			return err
		}
	}
	font.setEncoding(encoding)

	// read private dict
	privateEntry := topDict.getEntry("Private")
	if privateEntry == nil || privateEntry.size() < 2 {
		name, _ := font.Name()
		return fmt.Errorf("Private dictionary entry missing for font %s", name)
	}
	privateOffset := numberInt(privateEntry.getNumber(1))
	privateSize := numberInt(privateEntry.getNumber(0))
	privateDict, err := readDictDataAt(input, privateOffset, privateSize)
	if err != nil {
		return err
	}

	// populate private dict
	privDict, order := readPrivateDict(privateDict)
	for _, key := range order {
		font.addToPrivateDict(key, privDict[key])
	}

	// local subrs
	localSubrOffset := privateDict.getNumber("Subrs", 0)
	if offset, ok := localSubrOffset.(int); ok && offset > 0 {
		if err := input.SetPosition(privateOffset + offset); err != nil {
			return err
		}
		subrs, err := readIndexData(input)
		if err != nil {
			return err
		}
		font.addToPrivateDict("Subrs", subrs)
	}
	return nil
}

func (p *Parser) readString(index int) (string, error) {
	if index < 0 {
		return "", errors.New("Invalid negative index when reading a string")
	}
	if index <= 390 {
		return StandardStringName(index), nil
	}
	if p.stringIndex != nil && index-391 < len(p.stringIndex) {
		return p.stringIndex[index-391], nil
	}
	// technically this maps to .notdef, but we need a unique sid name
	return "SID" + strconv.Itoa(index), nil
}

// getString returns the string the named entry points at, the empty string
// where Java returns null.
func (p *Parser) getString(dict *dictData, name string) (any, error) {
	entry := dict.getEntry(name)
	if entry == nil || !entry.hasOperands() {
		return nil, nil
	}
	value, err := p.readString(numberInt(entry.getNumber(0)))
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (p *Parser) readEncoding(dataInput DataInput, charset CFFCharset) (*CFFEncoding, error) {
	format, err := dataInput.ReadUnsignedByte()
	if err != nil {
		return nil, err
	}
	baseFormat := format & 0x7f

	switch baseFormat {
	case 0:
		return p.readFormat0Encoding(dataInput, charset, format)
	case 1:
		return p.readFormat1Encoding(dataInput, charset, format)
	default:
		return nil, fmt.Errorf("Invalid encoding base format %d", baseFormat)
	}
}

// builtInEncoding is a font's built-in CFF encoding.
//
// Port of the abstract CFFParser.CFFBuiltInEncoding. Java's two subclasses hold
// nCodes and nRanges respectively, both for their toString alone; the port
// keeps the count under one name and says which it is.
type builtInEncoding struct {
	*CFFEncoding

	format     string
	count      int
	supplement []supplement
}

type supplement struct {
	code int
	sid  int
	name string
}

func (s supplement) String() string {
	return fmt.Sprintf("org.apache.fontbox.cff.CFFParser$CFFBuiltInEncoding$Supplement[code=%d, sid=%d]",
		s.code, s.sid)
}

func (e *builtInEncoding) String() string {
	parts := make([]string, len(e.supplement))
	for i, s := range e.supplement {
		parts[i] = s.String()
	}
	label := "nCodes"
	if e.format == "Format1Encoding" {
		label = "nRanges"
	}
	return fmt.Sprintf("org.apache.fontbox.cff.CFFParser$%s[%s=%d, supplement=[%s]]",
		e.format, label, e.count, strings.Join(parts, ", "))
}

func newBuiltInEncoding(format string, count int) *builtInEncoding {
	return &builtInEncoding{CFFEncoding: NewCFFEncoding(), format: format, count: count}
}

func (p *Parser) readFormat0Encoding(dataInput DataInput, charset CFFCharset,
	format int) (*CFFEncoding, error) {
	nCodes, err := dataInput.ReadUnsignedByte()
	if err != nil {
		return nil, err
	}
	encoding := newBuiltInEncoding("Format0Encoding", nCodes)
	encoding.Add(0, 0, ".notdef")
	for gid := 1; gid <= nCodes; gid++ {
		code, err := dataInput.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}
		sid := charset.SIDForGID(gid)
		name, err := p.readString(sid)
		if err != nil {
			return nil, err
		}
		encoding.Add(code, sid, name)
	}
	if format&0x80 != 0 {
		if err := p.readSupplement(dataInput, encoding); err != nil {
			return nil, err
		}
	}
	return encoding.CFFEncoding, nil
}

func (p *Parser) readFormat1Encoding(dataInput DataInput, charset CFFCharset,
	format int) (*CFFEncoding, error) {
	nRanges, err := dataInput.ReadUnsignedByte()
	if err != nil {
		return nil, err
	}
	encoding := newBuiltInEncoding("Format1Encoding", nRanges)
	encoding.Add(0, 0, ".notdef")
	gid := 1
	for i := 0; i < nRanges; i++ {
		rangeFirst, err := dataInput.ReadUnsignedByte() // First code in range
		if err != nil {
			return nil, err
		}
		rangeLeft, err := dataInput.ReadUnsignedByte() // Codes left in range (excluding first)
		if err != nil {
			return nil, err
		}
		for j := 0; j <= rangeLeft; j++ {
			sid := charset.SIDForGID(gid)
			name, err := p.readString(sid)
			if err != nil {
				return nil, err
			}
			encoding.Add(rangeFirst+j, sid, name)
			gid++
		}
	}
	if format&0x80 != 0 {
		if err := p.readSupplement(dataInput, encoding); err != nil {
			return nil, err
		}
	}
	return encoding.CFFEncoding, nil
}

func (p *Parser) readSupplement(dataInput DataInput, encoding *builtInEncoding) error {
	nSups, err := dataInput.ReadUnsignedByte()
	if err != nil {
		return err
	}
	encoding.supplement = make([]supplement, nSups)
	for i := 0; i < nSups; i++ {
		code, err := dataInput.ReadUnsignedByte()
		if err != nil {
			return err
		}
		sid, err := dataInput.ReadUnsignedShort()
		if err != nil {
			return err
		}
		name, err := p.readString(sid)
		if err != nil {
			return err
		}
		encoding.supplement[i] = supplement{code, sid, name}
		encoding.Add(code, sid, name)
	}
	return nil
}

// readFDSelect reads the FDSelect data according to the format.
func readFDSelect(dataInput DataInput, nGlyphs int) (FDSelect, error) {
	format, err := dataInput.ReadUnsignedByte()
	if err != nil {
		return nil, err
	}
	switch format {
	case 0:
		return readFormat0FDSelect(dataInput, nGlyphs)
	case 3:
		return readFormat3FDSelect(dataInput)
	default:
		// Java throws IllegalArgumentException, which is unchecked.
		panic("cff: illegal FDSelect format")
	}
}

// readFormat0FDSelect reads the Format 0 of the FDSelect data structure.
func readFormat0FDSelect(dataInput DataInput, nGlyphs int) (*format0FDSelect, error) {
	fds := make([]int, nGlyphs)
	for i := 0; i < nGlyphs; i++ {
		value, err := dataInput.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}
		fds[i] = value
	}
	return &format0FDSelect{fds: fds}, nil
}

// readFormat3FDSelect reads the Format 3 of the FDSelect data structure.
func readFormat3FDSelect(dataInput DataInput) (*format3FDSelect, error) {
	nbRanges, err := dataInput.ReadUnsignedShort()
	if err != nil {
		return nil, err
	}

	ranges := make([]range3, nbRanges)
	for i := 0; i < nbRanges; i++ {
		first, err := dataInput.ReadUnsignedShort()
		if err != nil {
			return nil, err
		}
		fd, err := dataInput.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}
		ranges[i] = range3{first, fd}
	}
	sentinel, err := dataInput.ReadUnsignedShort()
	if err != nil {
		return nil, err
	}
	return &format3FDSelect{range3: ranges, sentinel: sentinel}, nil
}

// format3FDSelect is Format 3 FDSelect data.
type format3FDSelect struct {
	range3   []range3
	sentinel int
}

var _ FDSelect = (*format3FDSelect)(nil)

func (f *format3FDSelect) FDIndex(gid int) int {
	for i := 0; i < len(f.range3); i++ {
		if f.range3[i].first <= gid {
			if i+1 < len(f.range3) {
				if f.range3[i+1].first > gid {
					return f.range3[i].fd
				}
				// go to next range
			} else {
				// last range reach, the sentinel must be greater than gid
				if f.sentinel > gid {
					return f.range3[i].fd
				}
				return -1
			}
		}
	}
	return 0
}

func (f *format3FDSelect) String() string {
	parts := make([]string, len(f.range3))
	for i, r := range f.range3 {
		parts[i] = r.String()
	}
	return fmt.Sprintf(
		"org.apache.fontbox.cff.CFFParser$Format3FDSelect[nbRanges=%d, range3=[%s] sentinel=%d]",
		len(f.range3), strings.Join(parts, ", "), f.sentinel)
}

// range3 is the structure of a Range3 element.
type range3 struct {
	first int
	fd    int
}

func (r range3) String() string {
	return fmt.Sprintf("org.apache.fontbox.cff.CFFParser$Range3[first=%d, fd=%d]", r.first, r.fd)
}

// format0FDSelect is Format 0 FDSelect.
type format0FDSelect struct {
	fds []int
}

var _ FDSelect = (*format0FDSelect)(nil)

func (f *format0FDSelect) FDIndex(gid int) int {
	if gid < len(f.fds) {
		return f.fds[gid]
	}
	return 0
}

func (f *format0FDSelect) String() string {
	parts := make([]string, len(f.fds))
	for i, fd := range f.fds {
		parts[i] = strconv.Itoa(fd)
	}
	return fmt.Sprintf("org.apache.fontbox.cff.CFFParser$Format0FDSelect[fds=[%s]]",
		strings.Join(parts, ", "))
}

func (p *Parser) readCharset(dataInput DataInput, nGlyphs int, isCIDFont bool) (CFFCharset, error) {
	format, err := dataInput.ReadUnsignedByte()
	if err != nil {
		return nil, err
	}
	switch format {
	case 0:
		return p.readFormat0Charset(dataInput, nGlyphs, isCIDFont)
	case 1:
		return p.readFormat1Charset(dataInput, nGlyphs, isCIDFont)
	case 2:
		return p.readFormat2Charset(dataInput, nGlyphs, isCIDFont)
	default:
		// we can't return new EmptyCharset(0), because this will bring more mayhem
		return nil, fmt.Errorf("Incorrect charset format %d", format)
	}
}

func (p *Parser) readFormat0Charset(dataInput DataInput, nGlyphs int,
	isCIDFont bool) (*format0Charset, error) {
	charset := &format0Charset{EmbeddedCharset: NewEmbeddedCharset(isCIDFont)}
	if isCIDFont {
		charset.AddCID(0, 0)
		for gid := 1; gid < nGlyphs; gid++ {
			cid, err := dataInput.ReadUnsignedShort()
			if err != nil {
				return nil, err
			}
			charset.AddCID(gid, cid)
		}
		return charset, nil
	}
	charset.AddSID(0, 0, ".notdef")
	for gid := 1; gid < nGlyphs; gid++ {
		sid, err := dataInput.ReadUnsignedShort()
		if err != nil {
			return nil, err
		}
		name, err := p.readString(sid)
		if err != nil {
			return nil, err
		}
		charset.AddSID(gid, sid, name)
	}
	return charset, nil
}

func (p *Parser) readFormat1Charset(dataInput DataInput, nGlyphs int,
	isCIDFont bool) (*format1Charset, error) {
	charset := &format1Charset{EmbeddedCharset: NewEmbeddedCharset(isCIDFont)}
	if isCIDFont {
		charset.AddCID(0, 0)
		gid := 1
		for gid < nGlyphs {
			rangeFirst, err := dataInput.ReadUnsignedShort()
			if err != nil {
				return nil, err
			}
			rangeLeft, err := dataInput.ReadUnsignedByte()
			if err != nil {
				return nil, err
			}
			charset.addRangeMapping(newRangeMapping(gid, rangeFirst, rangeLeft))
			gid += rangeLeft + 1
		}
		return charset, nil
	}
	charset.AddSID(0, 0, ".notdef")
	gid := 1
	for gid < nGlyphs {
		rangeFirst, err := dataInput.ReadUnsignedShort()
		if err != nil {
			return nil, err
		}
		rangeLeftRead, err := dataInput.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}
		rangeLeft := rangeLeftRead + 1
		for j := 0; j < rangeLeft; j++ {
			sid := rangeFirst + j
			name, err := p.readString(sid)
			if err != nil {
				return nil, err
			}
			charset.AddSID(gid+j, sid, name)
		}
		gid += rangeLeft
	}
	return charset, nil
}

func (p *Parser) readFormat2Charset(dataInput DataInput, nGlyphs int,
	isCIDFont bool) (*format2Charset, error) {
	charset := &format2Charset{EmbeddedCharset: NewEmbeddedCharset(isCIDFont)}
	if isCIDFont {
		charset.AddCID(0, 0)
		gid := 1
		for gid < nGlyphs {
			first, err := dataInput.ReadUnsignedShort()
			if err != nil {
				return nil, err
			}
			nLeft, err := dataInput.ReadUnsignedShort()
			if err != nil {
				return nil, err
			}
			charset.addRangeMapping(newRangeMapping(gid, first, nLeft))
			gid += nLeft + 1
		}
		return charset, nil
	}
	charset.AddSID(0, 0, ".notdef")
	gid := 1
	for gid < nGlyphs {
		first, err := dataInput.ReadUnsignedShort()
		if err != nil {
			return nil, err
		}
		nLeftRead, err := dataInput.ReadUnsignedShort()
		if err != nil {
			return nil, err
		}
		nLeft := nLeftRead + 1
		for j := 0; j < nLeft; j++ {
			sid := first + j
			name, err := p.readString(sid)
			if err != nil {
				return nil, err
			}
			charset.AddSID(gid+j, sid, name)
		}
		gid += nLeft
	}
	return charset, nil
}

// dictData holds the DictData of a CFF font.
type dictData struct {
	entries map[string]*dictEntry
}

func newDictData() *dictData { return &dictData{entries: map[string]*dictEntry{}} }

func (d *dictData) add(entry *dictEntry) {
	if entry.hasOperator {
		d.entries[entry.operatorName] = entry
	}
}

func (d *dictData) getEntry(name string) *dictEntry { return d.entries[name] }

func (d *dictData) getBoolean(name string, defaultValue bool) bool {
	entry := d.getEntry(name)
	if entry != nil && entry.hasOperands() {
		return entry.getBoolean(0, defaultValue)
	}
	return defaultValue
}

func (d *dictData) getArray(name string, defaultValue []any) []any {
	entry := d.getEntry(name)
	if entry != nil && entry.hasOperands() {
		return entry.operands
	}
	return defaultValue
}

func (d *dictData) getNumber(name string, defaultValue any) any {
	entry := d.getEntry(name)
	if entry != nil && entry.hasOperands() {
		return entry.getNumber(0)
	}
	return defaultValue
}

func (d *dictData) getDelta(name string, defaultValue []any) []any {
	entry := d.getEntry(name)
	if entry != nil && entry.hasOperands() {
		return entry.getDelta()
	}
	return defaultValue
}

// dictEntry holds an operand of a CFF font.
type dictEntry struct {
	operands     []any
	operatorName string
	hasOperator  bool
}

func (e *dictEntry) getNumber(index int) any { return e.operands[index] }

func (e *dictEntry) size() int { return len(e.operands) }

func (e *dictEntry) getBoolean(index int, defaultValue bool) bool {
	operand := e.operands[index]
	if value, ok := operand.(int); ok {
		switch value {
		case 0:
			return false
		case 1:
			return true
		}
	}
	slog.Warn("Expected boolean", "got", operand, "returning default", defaultValue)
	return defaultValue
}

func (e *dictEntry) hasOperands() bool { return len(e.operands) != 0 }

func (e *dictEntry) getDelta() []any {
	result := make([]any, len(e.operands))
	copy(result, e.operands)
	for i := 1; i < len(result); i++ {
		previous := result[i-1]
		current := result[i]
		sum := numberInt(previous) + numberInt(current)
		result[i] = sum
	}
	return result
}

// emptyCharsetCID is an empty charset in a malformed CID font.
type emptyCharsetCID struct {
	*CFFCharsetCID
}

func newEmptyCharsetCID(numCharStrings int) *emptyCharsetCID {
	charset := &emptyCharsetCID{CFFCharsetCID: NewCFFCharsetCID()}
	charset.AddCID(0, 0) // .notdef

	// Adobe Reader treats CID as GID, PDFBOX-2571 p11.
	for i := 1; i <= numCharStrings; i++ {
		charset.AddCID(i, i)
	}
	return charset
}

func (c *emptyCharsetCID) String() string {
	return "org.apache.fontbox.cff.CFFParser$EmptyCharsetCID"
}

// emptyCharsetType1 is an empty charset in a malformed Type1 font.
type emptyCharsetType1 struct {
	*CFFCharsetType1
}

func newEmptyCharsetType1() *emptyCharsetType1 {
	charset := &emptyCharsetType1{CFFCharsetType1: NewCFFCharsetType1()}
	charset.AddSID(0, 0, ".notdef")
	return charset
}

func (c *emptyCharsetType1) String() string {
	return "org.apache.fontbox.cff.CFFParser$EmptyCharsetType1"
}

// format0Charset is a Format0 charset.
type format0Charset struct {
	*EmbeddedCharset
}

// format1Charset is a Format1 charset.
type format1Charset struct {
	*EmbeddedCharset

	rangesCID2GID []rangeMapping
}

// addRangeMapping adds the given range mapping.
func (c *format1Charset) addRangeMapping(rangeMapping rangeMapping) {
	c.rangesCID2GID = append(c.rangesCID2GID, rangeMapping)
}

func (c *format1Charset) CIDForGID(gid int) int {
	if c.IsCIDFont() {
		for _, mapping := range c.rangesCID2GID {
			if mapping.isInRange(gid) {
				return mapping.mapValue(gid)
			}
		}
	}
	return c.EmbeddedCharset.CIDForGID(gid)
}

func (c *format1Charset) GIDForCID(cid int) int {
	if c.IsCIDFont() {
		for _, mapping := range c.rangesCID2GID {
			if mapping.isInReverseRange(cid) {
				return mapping.mapReverseValue(cid)
			}
		}
	}
	return c.EmbeddedCharset.GIDForCID(cid)
}

// format2Charset is a Format2 charset.
type format2Charset struct {
	*EmbeddedCharset

	rangesCID2GID []rangeMapping
}

// addRangeMapping adds the given range mapping.
func (c *format2Charset) addRangeMapping(rangeMapping rangeMapping) {
	c.rangesCID2GID = append(c.rangesCID2GID, rangeMapping)
}

func (c *format2Charset) CIDForGID(gid int) int {
	for _, mapping := range c.rangesCID2GID {
		if mapping.isInRange(gid) {
			return mapping.mapValue(gid)
		}
	}
	return c.EmbeddedCharset.CIDForGID(gid)
}

func (c *format2Charset) GIDForCID(cid int) int {
	for _, mapping := range c.rangesCID2GID {
		if mapping.isInReverseRange(cid) {
			return mapping.mapReverseValue(cid)
		}
	}
	return c.EmbeddedCharset.GIDForCID(cid)
}

// rangeMapping is a range mapping for a CID charset.
type rangeMapping struct {
	startValue       int
	endValue         int
	startMappedValue int
	endMappedValue   int
}

func newRangeMapping(startGID, first, nLeft int) rangeMapping {
	return rangeMapping{
		startValue:       startGID,
		endValue:         startGID + nLeft,
		startMappedValue: first,
		endMappedValue:   first + nLeft,
	}
}

func (r rangeMapping) isInRange(value int) bool {
	return value >= r.startValue && value <= r.endValue
}

func (r rangeMapping) isInReverseRange(value int) bool {
	return value >= r.startMappedValue && value <= r.endMappedValue
}

func (r rangeMapping) mapValue(value int) int {
	if r.isInRange(value) {
		return r.startMappedValue + (value - r.startValue)
	}
	return 0
}

func (r rangeMapping) mapReverseValue(value int) int {
	if r.isInReverseRange(value) {
		return r.startValue + (value - r.startMappedValue)
	}
	return 0
}

// cffByteSource allows bytes to be re-read later by the parser.
type cffByteSource struct {
	bytes []byte
}

func (s *cffByteSource) Bytes() ([]byte, error) { return s.bytes, nil }

// String describes the parser.
func (p *Parser) String() string { return "CFFParser[" + p.debugFontName + "]" }
