package afm

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
)

// The keys of an AFM file.
//
// Port of the constants of org.apache.fontbox.afm.AFMParser.
const (
	Comment          = "Comment"
	StartFontMetrics = "StartFontMetrics"
	EndFontMetrics   = "EndFontMetrics"

	FontName       = "FontName"
	FullName       = "FullName"
	FamilyName     = "FamilyName"
	Weight         = "Weight"
	FontBBox       = "FontBBox"
	Version        = "Version"
	Notice         = "Notice"
	EncodingScheme = "EncodingScheme"
	MappingScheme  = "MappingScheme"
	EscChar        = "EscChar"
	CharacterSet   = "CharacterSet"
	Characters     = "Characters"
	IsBaseFont     = "IsBaseFont"
	VVector        = "VVector"
	IsFixedV       = "IsFixedV"
	CapHeight      = "CapHeight"
	XHeight        = "XHeight"
	Ascender       = "Ascender"
	Descender      = "Descender"

	UnderlinePosition  = "UnderlinePosition"
	UnderlineThickness = "UnderlineThickness"
	ItalicAngle        = "ItalicAngle"
	CharWidth          = "CharWidth"
	IsFixedPitch       = "IsFixedPitch"

	StartCharMetrics = "StartCharMetrics"
	EndCharMetrics   = "EndCharMetrics"

	CharMetricsC   = "C"
	CharMetricsCH  = "CH"
	CharMetricsWX  = "WX"
	CharMetricsW0X = "W0X"
	CharMetricsW1X = "W1X"
	CharMetricsWY  = "WY"
	CharMetricsW0Y = "W0Y"
	CharMetricsW1Y = "W1Y"
	CharMetricsW   = "W"
	CharMetricsW0  = "W0"
	CharMetricsW1  = "W1"
	CharMetricsVV  = "VV"
	CharMetricsN   = "N"
	CharMetricsB   = "B"
	CharMetricsL   = "L"

	StdHW = "StdHW"
	StdVW = "StdVW"

	StartTrackKern  = "StartTrackKern"
	EndTrackKern    = "EndTrackKern"
	StartKernData   = "StartKernData"
	EndKernData     = "EndKernData"
	StartKernPairs  = "StartKernPairs"
	EndKernPairs    = "EndKernPairs"
	StartKernPairs0 = "StartKernPairs0"
	StartKernPairs1 = "StartKernPairs1"

	StartComposites = "StartComposites"
	EndComposites   = "EndComposites"
	CC              = "CC"
	PCC             = "PCC"

	KernPairKP  = "KP"
	KernPairKPH = "KPH"
	KernPairKPX = "KPX"
	KernPairKPY = "KPY"
)

// bitsInHex is the radix a hex character code is read in.
const bitsInHex = 16

// eofRune is what Java appends when read() returns -1 partway through a token:
// (char) -1, which is U+FFFF. The parser then hands that on as an unknown key,
// which is how a truncated file is reported rather than looping.
const eofRune = '￿'

// AFMParser reads an AFM file.
//
// Port of org.apache.fontbox.afm.AFMParser. Java takes an InputStream and reads
// it one byte at a time; the port takes an io.Reader and does the same, so that
// the parser stops exactly where Java stops.
type AFMParser struct {
	input io.Reader

	// one is the one-byte buffer every read goes through, so that the parser
	// consumes no more than Java does.
	one [1]byte
}

// NewAFMParser returns a parser reading the AFM document from in.
func NewAFMParser(in io.Reader) *AFMParser {
	return &AFMParser{input: in}
}

// Parse parses the AFM document.
func (p *AFMParser) Parse() (*FontMetrics, error) {
	return p.ParseReduced(false)
}

// ParseReduced parses the AFM document, skipping the kerning and composite data
// when reducedDataset is true.
func (p *AFMParser) ParseReduced(reducedDataset bool) (*FontMetrics, error) {
	return p.parseFontMetric(reducedDataset)
}

// parseFontMetric parses a font metrics item.
func (p *AFMParser) parseFontMetric(reducedDataset bool) (*FontMetrics, error) {
	if err := p.readCommand(StartFontMetrics); err != nil {
		return nil, err
	}
	fontMetrics := NewFontMetrics()
	version, err := p.readFloat()
	if err != nil {
		return nil, err
	}
	fontMetrics.SetAFMVersion(version)

	charMetricsRead := false
	for {
		nextCommand, err := p.readString()
		if err != nil {
			return nil, err
		}
		if nextCommand == EndFontMetrics {
			return fontMetrics, nil
		}

		switch nextCommand {
		case FontName:
			err = p.readLineInto(fontMetrics.SetFontName)
		case FullName:
			err = p.readLineInto(fontMetrics.SetFullName)
		case FamilyName:
			err = p.readLineInto(fontMetrics.SetFamilyName)
		case Weight:
			err = p.readLineInto(fontMetrics.SetWeight)
		case FontBBox:
			var bBox *util.BoundingBox
			if bBox, err = p.readBoundingBox(); err == nil {
				fontMetrics.SetFontBBox(bBox)
			}
		case Version:
			err = p.readLineInto(fontMetrics.SetFontVersion)
		case Notice:
			err = p.readLineInto(fontMetrics.SetNotice)
		case EncodingScheme:
			err = p.readLineInto(fontMetrics.SetEncodingScheme)
		case MappingScheme:
			err = p.readIntInto(fontMetrics.SetMappingScheme)
		case EscChar:
			err = p.readIntInto(fontMetrics.SetEscChar)
		case CharacterSet:
			err = p.readLineInto(fontMetrics.SetCharacterSet)
		case Characters:
			err = p.readIntInto(fontMetrics.SetCharacters)
		case IsBaseFont:
			err = p.readBooleanInto(fontMetrics.SetIsBaseFont)
		case VVector:
			var vector []float32
			if vector, err = p.readFloatPair(); err == nil {
				fontMetrics.SetVVector(vector)
			}
		case IsFixedV:
			err = p.readBooleanInto(fontMetrics.SetIsFixedV)
		case CapHeight:
			err = p.readFloatInto(fontMetrics.SetCapHeight)
		case XHeight:
			err = p.readFloatInto(fontMetrics.SetXHeight)
		case Ascender:
			err = p.readFloatInto(fontMetrics.SetAscender)
		case Descender:
			err = p.readFloatInto(fontMetrics.SetDescender)
		case StdHW:
			err = p.readFloatInto(fontMetrics.SetStandardHorizontalWidth)
		case StdVW:
			err = p.readFloatInto(fontMetrics.SetStandardVerticalWidth)
		case Comment:
			err = p.readLineInto(fontMetrics.AddComment)
		case UnderlinePosition:
			err = p.readFloatInto(fontMetrics.SetUnderlinePosition)
		case UnderlineThickness:
			err = p.readFloatInto(fontMetrics.SetUnderlineThickness)
		case ItalicAngle:
			err = p.readFloatInto(fontMetrics.SetItalicAngle)
		case CharWidth:
			var widths []float32
			if widths, err = p.readFloatPair(); err == nil {
				fontMetrics.SetCharWidth(widths)
			}
		case IsFixedPitch:
			err = p.readBooleanInto(fontMetrics.SetFixedPitch)
		case StartCharMetrics:
			charMetricsRead, err = p.parseCharMetrics(fontMetrics)
		case StartKernData:
			if !reducedDataset {
				err = p.parseKernData(fontMetrics)
			}
		case StartComposites:
			if !reducedDataset {
				err = p.parseComposites(fontMetrics)
			}
		default:
			if !reducedDataset || !charMetricsRead {
				// The wording is Javas, because AFMParserTest asserts this substring.
				return nil, fmt.Errorf("afm: Unknown AFM key '%s'", nextCommand)
			}
		}
		if err != nil {
			return nil, err
		}
	}
}

// parseKernData parses the kern data.
func (p *AFMParser) parseKernData(fontMetrics *FontMetrics) error {
	for {
		nextCommand, err := p.readString()
		if err != nil {
			return err
		}
		if nextCommand == EndKernData {
			return nil
		}

		switch nextCommand {
		case StartTrackKern:
			countTrackKern, err := p.readInt()
			if err != nil {
				return err
			}
			for i := 0; i < countTrackKern; i++ {
				kern, err := p.readTrackKern()
				if err != nil {
					return err
				}
				fontMetrics.AddTrackKern(kern)
			}
			if err := p.readCommand(EndTrackKern); err != nil {
				return err
			}
		case StartKernPairs:
			err = p.parseKernPairsInto(fontMetrics.AddKernPair)
		case StartKernPairs0:
			err = p.parseKernPairsInto(fontMetrics.AddKernPair0)
		case StartKernPairs1:
			err = p.parseKernPairsInto(fontMetrics.AddKernPair1)
		default:
			return fmt.Errorf("afm: unknown kerning data type '%s'", nextCommand)
		}
		if err != nil {
			return err
		}
	}
}

func (p *AFMParser) readTrackKern() (*TrackKern, error) {
	degree, err := p.readInt()
	if err != nil {
		return nil, err
	}
	values := make([]float32, 4)
	for i := range values {
		if values[i], err = p.readFloat(); err != nil {
			return nil, err
		}
	}
	return NewTrackKern(degree, values[0], values[1], values[2], values[3]), nil
}

// parseKernPairsInto reads a counted run of kerning pairs and hands each to add.
//
// Java writes the three directions out as three near-identical methods; they
// differ only in which list the pair goes into.
func (p *AFMParser) parseKernPairsInto(add func(*KernPair)) error {
	countKernPairs, err := p.readInt()
	if err != nil {
		return err
	}
	for i := 0; i < countKernPairs; i++ {
		pair, err := p.parseKernPair()
		if err != nil {
			return err
		}
		add(pair)
	}
	return p.readCommand(EndKernPairs)
}

// parseKernPair parses a kern pair from the data stream.
func (p *AFMParser) parseKernPair() (*KernPair, error) {
	cmd, err := p.readString()
	if err != nil {
		return nil, err
	}
	switch cmd {
	case KernPairKP:
		return p.readKernPair(false, true, true)
	case KernPairKPH:
		return p.readKernPair(true, true, true)
	case KernPairKPX:
		return p.readKernPair(false, true, false)
	case KernPairKPY:
		return p.readKernPair(false, false, true)
	default:
		return nil, fmt.Errorf("afm: error expected kern pair command actual='%s'", cmd)
	}
}

// readKernPair reads the two characters of a pair and whichever of the two
// displacements the command carries, leaving the other at zero.
func (p *AFMParser) readKernPair(hex, readX, readY bool) (*KernPair, error) {
	first, err := p.readString()
	if err != nil {
		return nil, err
	}
	second, err := p.readString()
	if err != nil {
		return nil, err
	}
	if hex {
		if first, err = p.hexToString(first); err != nil {
			return nil, err
		}
		if second, err = p.hexToString(second); err != nil {
			return nil, err
		}
	}

	var x, y float32
	if readX {
		if x, err = p.readFloat(); err != nil {
			return nil, err
		}
	}
	if readY {
		if y, err = p.readFloat(); err != nil {
			return nil, err
		}
	}
	return NewKernPair(first, second, x, y), nil
}

// hexToString converts an angle bracket hex string to a string.
func (p *AFMParser) hexToString(hexToString string) (string, error) {
	if len(hexToString) < 2 {
		return "", fmt.Errorf("afm: error: expected hex string of length >= 2 not='%s", hexToString)
	}
	if hexToString[0] != '<' || hexToString[len(hexToString)-1] != '>' {
		return "", fmt.Errorf("afm: string should be enclosed by angle brackets '%s'", hexToString)
	}
	hexString := hexToString[1 : len(hexToString)-1]
	data := make([]byte, len(hexString)/2)
	for i := 0; i+1 < len(hexString); i += 2 {
		value, err := parseIntRadix(hexString[i:i+2], bitsInHex)
		if err != nil {
			return "", err
		}
		data[i/2] = byte(value)
	}
	// Java decodes as ISO-8859-1, which maps each byte to the code point of the
	// same value.
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes), nil
}

func (p *AFMParser) parseComposites(fontMetrics *FontMetrics) error {
	countComposites, err := p.readInt()
	if err != nil {
		return err
	}
	for i := 0; i < countComposites; i++ {
		composite, err := p.parseComposite()
		if err != nil {
			return err
		}
		fontMetrics.AddComposite(composite)
	}
	return p.readCommand(EndComposites)
}

// parseComposite parses a composite character from the stream.
func (p *AFMParser) parseComposite() (*Composite, error) {
	partData, err := p.readLine()
	if err != nil {
		return nil, err
	}
	// Java uses a StringTokenizer over " ;", which skips empty tokens.
	tokens := strings.FieldsFunc(partData, func(r rune) bool { return r == ' ' || r == ';' })
	next := func() (string, error) {
		if len(tokens) == 0 {
			return "", errors.New("afm: composite data ended early")
		}
		token := tokens[0]
		tokens = tokens[1:]
		return token, nil
	}

	cc, err := next()
	if err != nil {
		return nil, err
	}
	if cc != CC {
		return nil, fmt.Errorf("afm: expected '%s' actual='%s'", CC, cc)
	}
	name, err := next()
	if err != nil {
		return nil, err
	}
	composite := NewComposite(name)

	countToken, err := next()
	if err != nil {
		return nil, err
	}
	partCount, err := parseIntRadix(countToken, 10)
	if err != nil {
		return nil, err
	}
	for i := 0; i < partCount; i++ {
		pcc, err := next()
		if err != nil {
			return nil, err
		}
		if pcc != PCC {
			return nil, fmt.Errorf("afm: expected '%s' actual='%s'", PCC, pcc)
		}
		partName, err := next()
		if err != nil {
			return nil, err
		}
		x, err := nextInt(next)
		if err != nil {
			return nil, err
		}
		y, err := nextInt(next)
		if err != nil {
			return nil, err
		}
		composite.AddPart(NewCompositePart(partName, x, y))
	}
	return composite, nil
}

func nextInt(next func() (string, error)) (int, error) {
	token, err := next()
	if err != nil {
		return 0, err
	}
	return parseIntRadix(token, 10)
}

func (p *AFMParser) parseCharMetrics(fontMetrics *FontMetrics) (bool, error) {
	countMetrics, err := p.readInt()
	if err != nil {
		return false, err
	}
	for i := 0; i < countMetrics; i++ {
		charMetric, err := p.parseCharMetric()
		if err != nil {
			return false, err
		}
		fontMetrics.AddCharMetric(charMetric)
	}
	if err := p.readCommand(EndCharMetrics); err != nil {
		return false, err
	}
	return true, nil
}

// parseCharMetric parses a single character metric from the stream.
func (p *AFMParser) parseCharMetric() (*CharMetric, error) {
	charMetric := NewCharMetric()
	metrics, err := p.readLine()
	if err != nil {
		return nil, err
	}
	// Java uses a StringTokenizer with the default delimiters, which are the
	// whitespace characters, and which skips empty tokens.
	tokens := strings.Fields(metrics)
	next := func() (string, error) {
		if len(tokens) == 0 {
			return "", errors.New("afm: char metrics ended early")
		}
		token := tokens[0]
		tokens = tokens[1:]
		return token, nil
	}
	// verifySemicolon reads the token that must follow a command.
	verifySemicolon := func() error {
		if len(tokens) == 0 {
			return errors.New("afm: CharMetrics is missing a semicolon after a command")
		}
		semicolon, _ := next()
		if semicolon != ";" {
			return fmt.Errorf("afm: error: expected semicolon in stream actual='%s'", semicolon)
		}
		return nil
	}

	for len(tokens) > 0 {
		nextCommand, err := next()
		if err != nil {
			return nil, err
		}
		switch nextCommand {
		// top 5 most used first
		case CharMetricsC:
			err = withInt(next, 10, charMetric.SetCharacterCode)
		case CharMetricsWX:
			err = withFloat(next, charMetric.SetWx)
		case CharMetricsN:
			var name string
			if name, err = next(); err == nil {
				charMetric.SetName(name)
			}
		case CharMetricsB:
			var box *util.BoundingBox
			if box, err = boundingBoxFrom(next); err == nil {
				charMetric.SetBoundingBox(box)
			}
		case CharMetricsL:
			var successor, ligature string
			if successor, err = next(); err == nil {
				if ligature, err = next(); err == nil {
					charMetric.AddLigature(NewLigature(successor, ligature))
				}
			}
		case CharMetricsCH:
			// Is the hex string <FF> or FF, the spec is a little
			// unclear, wait and see if it breaks anything.
			err = withInt(next, bitsInHex, charMetric.SetCharacterCode)
		case CharMetricsW0X:
			err = withFloat(next, charMetric.SetW0x)
		case CharMetricsW1X:
			err = withFloat(next, charMetric.SetW1x)
		case CharMetricsWY:
			err = withFloat(next, charMetric.SetWy)
		case CharMetricsW0Y:
			err = withFloat(next, charMetric.SetW0y)
		case CharMetricsW1Y:
			err = withFloat(next, charMetric.SetW1y)
		case CharMetricsW:
			err = withFloatPair(next, charMetric.SetW)
		case CharMetricsW0:
			err = withFloatPair(next, charMetric.SetW0)
		case CharMetricsW1:
			err = withFloatPair(next, charMetric.SetW1)
		case CharMetricsVV:
			err = withFloatPair(next, charMetric.SetVv)
		default:
			return nil, fmt.Errorf("afm: unknown CharMetrics command '%s'", nextCommand)
		}
		if err != nil {
			return nil, err
		}
		if err := verifySemicolon(); err != nil {
			return nil, err
		}
	}
	return charMetric, nil
}

func withInt(next func() (string, error), radix int, set func(int)) error {
	token, err := next()
	if err != nil {
		return err
	}
	value, err := parseIntRadix(token, radix)
	if err != nil {
		return err
	}
	set(value)
	return nil
}

func withFloat(next func() (string, error), set func(float32)) error {
	token, err := next()
	if err != nil {
		return err
	}
	value, err := parseFloat(token)
	if err != nil {
		return err
	}
	set(value)
	return nil
}

func withFloatPair(next func() (string, error), set func([]float32)) error {
	pair := make([]float32, 2)
	for i := range pair {
		token, err := next()
		if err != nil {
			return err
		}
		if pair[i], err = parseFloat(token); err != nil {
			return err
		}
	}
	set(pair)
	return nil
}

func boundingBoxFrom(next func() (string, error)) (*util.BoundingBox, error) {
	box := util.NewBoundingBox()
	setters := []func(float32){
		box.SetLowerLeftX, box.SetLowerLeftY, box.SetUpperRightX, box.SetUpperRightY,
	}
	for _, set := range setters {
		if err := withFloat(next, set); err != nil {
			return nil, err
		}
	}
	return box, nil
}

// --- reading from the stream ---

func (p *AFMParser) readBooleanInto(set func(bool)) error {
	value, err := p.readBoolean()
	if err != nil {
		return err
	}
	set(value)
	return nil
}

func (p *AFMParser) readIntInto(set func(int)) error {
	value, err := p.readInt()
	if err != nil {
		return err
	}
	set(value)
	return nil
}

func (p *AFMParser) readFloatInto(set func(float32)) error {
	value, err := p.readFloat()
	if err != nil {
		return err
	}
	set(value)
	return nil
}

func (p *AFMParser) readLineInto(set func(string)) error {
	value, err := p.readLine()
	if err != nil {
		return err
	}
	set(value)
	return nil
}

func (p *AFMParser) readBoundingBox() (*util.BoundingBox, error) {
	box := util.NewBoundingBox()
	setters := []func(float32){
		box.SetLowerLeftX, box.SetLowerLeftY, box.SetUpperRightX, box.SetUpperRightY,
	}
	for _, set := range setters {
		if err := p.readFloatInto(set); err != nil {
			return nil, err
		}
	}
	return box, nil
}

func (p *AFMParser) readFloatPair() ([]float32, error) {
	pair := make([]float32, 2)
	for i := range pair {
		value, err := p.readFloat()
		if err != nil {
			return nil, err
		}
		pair[i] = value
	}
	return pair, nil
}

// readBoolean reads a boolean from the stream. Java's Boolean.parseBoolean
// accepts only "true", ignoring case, and calls everything else false.
func (p *AFMParser) readBoolean() (bool, error) {
	value, err := p.readString()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(value, "true"), nil
}

// readInt reads a decimal integer from the stream.
func (p *AFMParser) readInt() (int, error) {
	value, err := p.readString()
	if err != nil {
		return 0, err
	}
	return parseIntRadix(value, 10)
}

func parseIntRadix(intValue string, radix int) (int, error) {
	// Java calls Integer.parseInt, which rejects anything outside the 32-bit
	// range; a 64-bit parse here would accept a count the reference
	// implementation throws on and then loop on it.
	value, err := strconv.ParseInt(intValue, radix, 32)
	if err != nil {
		return 0, fmt.Errorf("afm: error parsing AFM document: %w", err)
	}
	return int(value), nil
}

// readFloat reads a float from the stream.
func (p *AFMParser) readFloat() (float32, error) {
	value, err := p.readString()
	if err != nil {
		return 0, err
	}
	return parseFloat(value)
}

func parseFloat(floatValue string) (float32, error) {
	value, err := strconv.ParseFloat(floatValue, 32)
	if err != nil {
		return 0, fmt.Errorf("afm: error parsing AFM document: %w", err)
	}
	return float32(value), nil
}

// readByte reads one byte, reporting eof as Java's read() does with -1.
func (p *AFMParser) readByte() (int, error) {
	n, err := p.input.Read(p.one[:])
	if n == 1 {
		return int(p.one[0]), nil
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return -1, err
	}
	return -1, nil
}

// readLine reads until the end of a line.
//
// The first byte is appended before it is tested for end of input, exactly as
// Java does, so a read past the end yields U+FFFF rather than an empty string.
func (p *AFMParser) readLine() (string, error) {
	return p.readToken(isEOL)
}

// readString reads a string from the input stream and stops at any whitespace.
func (p *AFMParser) readString() (string, error) {
	return p.readToken(isWhitespace)
}

// readToken is the body both readLine and readString have in Java, differing
// only in what ends the token.
func (p *AFMParser) readToken(ends func(int) bool) (string, error) {
	// First skip the whitespace
	var buf strings.Builder
	nextByte, err := p.readByte()
	if err != nil {
		return "", err
	}
	for isWhitespace(nextByte) {
		if nextByte, err = p.readByte(); err != nil {
			return "", err
		}
		// do nothing just skip the whitespace.
	}
	buf.WriteRune(javaChar(nextByte))

	// now read the data
	if nextByte, err = p.readByte(); err != nil {
		return "", err
	}
	for nextByte != -1 && !ends(nextByte) {
		buf.WriteRune(javaChar(nextByte))
		if nextByte, err = p.readByte(); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

// javaChar widens a byte the way Java's (char) cast does, so that the -1 of a
// read past the end becomes U+FFFF.
func javaChar(b int) rune {
	if b == -1 {
		return eofRune
	}
	return rune(b)
}

// readCommand reads the next string and reports one that differs from the
// expected command.
func (p *AFMParser) readCommand(expectedCommand string) error {
	command, err := p.readString()
	if err != nil {
		return err
	}
	if command != expectedCommand {
		return fmt.Errorf("afm: error: expected '%s' actual '%s'", expectedCommand, command)
	}
	return nil
}

// isEOL reports whether the byte ends a line.
func isEOL(character int) bool {
	return character == 0x0D || character == 0x0A
}

// isWhitespace reports whether the byte is whitespace as defined by the AFM
// specification.
func isWhitespace(character int) bool {
	switch character {
	case ' ', '\t', 0x0D, 0x0A:
		return true
	}
	return false
}
