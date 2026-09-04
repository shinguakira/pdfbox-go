package type1

import (
	"errors"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/fontbox/encoding"
)

// constants for encryption
const (
	eexecKey      = 55665
	charstringKey = 4330
)

// type1Parser parses an Adobe Type 1 (.pfb) font. It is used exclusively by
// Type1Font.
//
// The Type 1 font format is a free-text format which is somewhat difficult
// to parse. This is made worse by the fact that many Type 1 font files do
// not conform to the specification, especially those embedded in PDFs. This
// parser therefore tries to be as forgiving as possible.
//
// See "Adobe Type 1 Font Format, Adobe Systems (1999)".
//
// Port of org.apache.fontbox.type1.Type1Parser.
type type1Parser struct {
	// state
	lexer *type1Lexer
	font  *Type1Font
}

func newType1Parser() *type1Parser { return &type1Parser{} }

// parse parses a Type 1 font and returns a Type1Font which represents it,
// segment1 being the ASCII segment and segment2 the binary one.
func (p *type1Parser) parse(segment1, segment2 []byte) (*Type1Font, error) {
	p.font = newType1Font(segment1, segment2)
	// Java catches NumberFormatException here and rethrows it as an
	// IOException; the port's Token.FloatValue panics on one, and this is
	// where that panic becomes an error.
	if err := p.parseASCIICatchingNumberFormat(segment1); err != nil {
		return nil, err
	}
	if len(segment2) > 0 {
		if err := p.parseBinary(segment2); err != nil {
			return nil, err
		}
	}
	return p.font, nil
}

func (p *type1Parser) parseASCIICatchingNumberFormat(bytes []byte) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			numberFormat, ok := recovered.(*numberFormatError)
			if !ok {
				panic(recovered)
			}
			err = numberFormat
		}
	}()
	return p.parseASCII(bytes)
}

// parseASCII parses the ASCII portion of a Type 1 font.
func (p *type1Parser) parseASCII(bytes []byte) error {
	if len(bytes) == 0 {
		return errors.New("ASCII segment of type 1 font is empty")
	}

	// %!FontType1-1.0
	// %!PS-AdobeFont-1.0
	if len(bytes) < 2 || (bytes[0] != '%' && bytes[1] != '!') {
		return errors.New("Invalid start of ASCII segment of type 1 font")
	}

	lexer, err := newType1Lexer(bytes)
	if err != nil {
		return err
	}
	p.lexer = lexer

	// (corrupt?) synthetic font
	if p.peekText() == "FontDirectory" {
		if err := p.readNamed(kindName, "FontDirectory"); err != nil {
			return err
		}
		if _, err := p.read(kindLiteral); err != nil { // font name
			return err
		}
		if err := p.readNamed(kindName, "known"); err != nil {
			return err
		}
		if _, err := p.read(kindStartProc); err != nil {
			return err
		}
		if err := p.readProcVoid(); err != nil {
			return err
		}
		if _, err := p.read(kindStartProc); err != nil {
			return err
		}
		if err := p.readProcVoid(); err != nil {
			return err
		}
		if err := p.readNamed(kindName, "ifelse"); err != nil {
			return err
		}
	}

	// font dict
	lengthToken, err := p.read(kindInteger)
	if err != nil {
		return err
	}
	length := lengthToken.IntValue()
	if err := p.readNamed(kindName, "dict"); err != nil {
		return err
	}
	// found in some TeX fonts
	if _, err := p.readMaybe(kindName, "dup"); err != nil {
		return err
	}
	// if present, the "currentdict" is not required
	if err := p.readNamed(kindName, "begin"); err != nil {
		return err
	}

	for i := 0; i < length; i++ {
		// premature end
		token := p.lexer.PeekToken()
		if token == nil {
			break
		}
		if token.Kind() == kindName &&
			(token.Text() == "currentdict" || token.Text() == "end") {
			break
		}

		// key/value
		keyToken, err := p.read(kindLiteral)
		if err != nil {
			return err
		}
		switch key := keyToken.Text(); key {
		case "FontInfo", "Fontinfo":
			fontInfo, order, err := p.readSimpleDict()
			if err != nil {
				return err
			}
			p.readFontInfo(fontInfo, order)
		case "Metrics":
			if _, _, err := p.readSimpleDict(); err != nil {
				return err
			}
		case "Encoding":
			if err := p.readEncoding(); err != nil {
				return err
			}
		default:
			if err := p.readSimpleValue(key); err != nil {
				return err
			}
		}
	}

	if _, err := p.readMaybe(kindName, "currentdict"); err != nil {
		return err
	}
	if err := p.readNamed(kindName, "end"); err != nil {
		return err
	}

	if err := p.readNamed(kindName, "currentfile"); err != nil {
		return err
	}
	return p.readNamed(kindName, "eexec")
}

func (p *type1Parser) readSimpleValue(key string) error {
	value, err := p.readDictValue()
	if err != nil {
		return err
	}

	switch key {
	case "FontName":
		p.font.fontName = value[0].Text()
	case "PaintType":
		p.font.paintType = value[0].IntValue()
	case "FontType":
		p.font.fontType = value[0].IntValue()
	case "FontMatrix":
		if p.font.fontMatrix, err = arrayToNumbers(value); err != nil {
			return err
		}
	case "FontBBox":
		if p.font.fontBBox, err = arrayToNumbers(value); err != nil {
			return err
		}
	case "UniqueID":
		p.font.uniqueID = value[0].IntValue()
	case "StrokeWidth":
		p.font.strokeWidth = value[0].FloatValue()
	case "FID":
		p.font.fontID = value[0].Text()
	}
	return nil
}

func (p *type1Parser) readEncoding() error {
	if p.lexer.PeekKind(kindName) {
		token, err := p.lexer.NextToken()
		if err != nil {
			return err
		}
		name := token.Text()

		if name != "StandardEncoding" {
			return fmt.Errorf("Unknown encoding: %s", name)
		}
		p.font.encoding = encoding.StandardEncoding
		if _, err := p.readMaybe(kindName, "readonly"); err != nil {
			return err
		}
		return p.readNamed(kindName, "def")
	}

	if _, err := p.read(kindInteger); err != nil {
		return err
	}
	if _, err := p.readMaybe(kindName, "array"); err != nil {
		return err
	}

	// 0 1 255 {1 index exch /.notdef put } for
	// we have to check "readonly" and "def" too
	// as some fonts don't provide any dup-values, see PDFBOX-2134
	for !(p.lexer.PeekKind(kindName) &&
		(p.peekText() == "dup" || p.peekText() == "readonly" || p.peekText() == "def")) {
		next, err := p.lexer.NextToken()
		if err != nil {
			return err
		}
		if next == nil {
			return errors.New("Incomplete data while reading encoding of type 1 font")
		}
	}

	codeToName := map[int]string{}
	for p.lexer.PeekKind(kindName) && p.peekText() == "dup" {
		if err := p.readNamed(kindName, "dup"); err != nil {
			return err
		}
		codeToken, err := p.read(kindInteger)
		if err != nil {
			return err
		}
		nameToken, err := p.read(kindLiteral)
		if err != nil {
			return err
		}
		if err := p.readNamed(kindName, "put"); err != nil {
			return err
		}
		codeToName[codeToken.IntValue()] = nameToken.Text()
	}
	p.font.encoding = encoding.NewBuiltInEncoding(codeToName)
	if _, err := p.readMaybe(kindName, "readonly"); err != nil {
		return err
	}
	return p.readNamed(kindName, "def")
}

// arrayToNumbers extracts values from an array as numbers.
func arrayToNumbers(value []*token) ([]any, error) {
	numbers := make([]any, 0, len(value))
	for i, size := 1, len(value)-1; i < size; i++ {
		t := value[i]
		switch t.Kind() {
		case kindReal:
			numbers = append(numbers, t.FloatValue())
		case kindInteger:
			numbers = append(numbers, t.IntValue())
		default:
			return nil, fmt.Errorf("Expected INTEGER or REAL but got %v at array position %d",
				t, i)
		}
	}
	return numbers, nil
}

// readFontInfo extracts values from the /FontInfo dictionary.
func (p *type1Parser) readFontInfo(fontInfo map[string][]*token, order []string) {
	for _, key := range order {
		value := fontInfo[key]

		switch key {
		case "version":
			p.font.version = value[0].Text()
		case "Notice":
			p.font.notice = value[0].Text()
		case "FullName":
			p.font.fullName = value[0].Text()
		case "FamilyName":
			p.font.familyName = value[0].Text()
		case "Weight":
			p.font.weight = value[0].Text()
		case "ItalicAngle":
			p.font.italicAngle = value[0].FloatValue()
		case "isFixedPitch":
			p.font.isFixedPitch = value[0].BooleanValue()
		case "UnderlinePosition":
			p.font.underlinePosition = value[0].FloatValue()
		case "UnderlineThickness":
			p.font.underlineThickness = value[0].FloatValue()
		}
	}
}

// readSimpleDict reads a dictionary whose values are simple, i.e., do not
// contain nested dictionaries. The second result is the order the keys were
// read in, which Java does not need because it iterates a HashMap and the
// switch over the keys does not care about order.
func (p *type1Parser) readSimpleDict() (map[string][]*token, []string, error) {
	dict := map[string][]*token{}
	var order []string

	lengthToken, err := p.read(kindInteger)
	if err != nil {
		return nil, nil, err
	}
	length := lengthToken.IntValue()
	if err := p.readNamed(kindName, "dict"); err != nil {
		return nil, nil, err
	}
	if _, err := p.readMaybe(kindName, "dup"); err != nil {
		return nil, nil, err
	}

	maybeDef, err := p.readMaybe(kindName, "def")
	if err != nil {
		return nil, nil, err
	}
	if maybeDef != nil {
		// PDFBOX-5942 empty dict
		return dict, order, nil
	}

	if err := p.readNamed(kindName, "begin"); err != nil {
		return nil, nil, err
	}

	for i := 0; i < length; i++ {
		if p.lexer.PeekToken() == nil {
			break
		}
		if p.lexer.PeekKind(kindName) && p.peekText() != "end" {
			if _, err := p.read(kindName); err != nil {
				return nil, nil, err
			}
		}
		// premature end
		if p.lexer.PeekToken() == nil {
			break
		}
		if p.lexer.PeekKind(kindName) && p.peekText() == "end" {
			break
		}

		// simple value
		keyToken, err := p.read(kindLiteral)
		if err != nil {
			return nil, nil, err
		}
		value, err := p.readDictValue()
		if err != nil {
			return nil, nil, err
		}
		key := keyToken.Text()
		if _, seen := dict[key]; !seen {
			order = append(order, key)
		}
		dict[key] = value
	}

	if err := p.readNamed(kindName, "end"); err != nil {
		return nil, nil, err
	}
	if _, err := p.readMaybe(kindName, "readonly"); err != nil {
		return nil, nil, err
	}
	if err := p.readNamed(kindName, "def"); err != nil {
		return nil, nil, err
	}

	return dict, order, nil
}

// readDictValue reads a simple value from a dictionary.
func (p *type1Parser) readDictValue() ([]*token, error) {
	value, err := p.readValue()
	if err != nil {
		return nil, err
	}
	if err := p.readDef(); err != nil {
		return nil, err
	}
	return value, nil
}

// readValue reads a simple value. This is either a number, a string, a name, a
// literal name, an array, a procedure, or a charstring. This method does not
// support reading nested dictionaries unless they're empty.
func (p *type1Parser) readValue() ([]*token, error) {
	var value []*token
	t, err := p.lexer.NextToken()
	if err != nil {
		return nil, err
	}
	if p.lexer.PeekToken() == nil {
		return value, nil
	}
	value = append(value, t)

	switch t.Kind() {
	case kindStartArray:
		openArray := 1
		for {
			if p.lexer.PeekToken() == nil {
				return value, nil
			}
			if p.lexer.PeekKind(kindStartArray) {
				openArray++
			}

			if t, err = p.lexer.NextToken(); err != nil {
				return nil, err
			}
			value = append(value, t)

			if t.Kind() == kindEndArray {
				openArray--
				if openArray == 0 {
					break
				}
			}
		}
	case kindStartProc:
		proc, err := p.readProc()
		if err != nil {
			return nil, err
		}
		value = append(value, proc...)
	case kindStartDict:
		// skip "/GlyphNames2HostCode << >> def"
		if _, err := p.read(kindEndDict); err != nil {
			return nil, err
		}
		return value, nil
	}

	value, err = p.readPostScriptWrapper(value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (p *type1Parser) readPostScriptWrapper(value []*token) ([]*token, error) {
	if p.lexer.PeekToken() == nil {
		return nil, errors.New("Missing start token for the system dictionary")
	}
	// postscript wrapper (not in the Type 1 spec)
	if p.peekText() != "systemdict" {
		return value, nil
	}
	if err := p.readNamed(kindName, "systemdict"); err != nil {
		return nil, err
	}
	if err := p.readNamed(kindLiteral, "internaldict"); err != nil {
		return nil, err
	}
	if err := p.readNamed(kindName, "known"); err != nil {
		return nil, err
	}

	if _, err := p.read(kindStartProc); err != nil {
		return nil, err
	}
	if err := p.readProcVoid(); err != nil {
		return nil, err
	}

	if _, err := p.read(kindStartProc); err != nil {
		return nil, err
	}
	if err := p.readProcVoid(); err != nil {
		return nil, err
	}

	if err := p.readNamed(kindName, "ifelse"); err != nil {
		return nil, err
	}

	// replace value
	if _, err := p.read(kindStartProc); err != nil {
		return nil, err
	}
	if err := p.readNamed(kindName, "pop"); err != nil {
		return nil, err
	}
	replacement, err := p.readValue()
	if err != nil {
		return nil, err
	}
	value = replacement
	if _, err := p.read(kindEndProc); err != nil {
		return nil, err
	}

	if err := p.readNamed(kindName, "if"); err != nil {
		return nil, err
	}
	return value, nil
}

// readProc reads a procedure.
func (p *type1Parser) readProc() ([]*token, error) {
	var value []*token

	openProc := 1
	for {
		if p.lexer.PeekToken() == nil {
			return nil, errors.New("Malformed procedure: missing token")
		}

		if p.lexer.PeekKind(kindStartProc) {
			openProc++
		}

		t, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}
		value = append(value, t)

		if t.Kind() == kindEndProc {
			openProc--
			if openProc == 0 {
				break
			}
		}
	}
	executeonly, err := p.readMaybe(kindName, "executeonly")
	if err != nil {
		return nil, err
	}
	if executeonly != nil {
		value = append(value, executeonly)
	}

	return value, nil
}

// readProcVoid reads a procedure but without returning anything.
func (p *type1Parser) readProcVoid() error {
	openProc := 1
	for {
		if p.lexer.PeekToken() == nil {
			return errors.New("Malformed procedure: missing token")
		}
		if p.lexer.PeekKind(kindStartProc) {
			openProc++
		}

		t, err := p.lexer.NextToken()
		if err != nil {
			return err
		}

		if t.Kind() == kindEndProc {
			openProc--
			if openProc == 0 {
				break
			}
		}
	}
	_, err := p.readMaybe(kindName, "executeonly")
	return err
}

// parseBinary parses the binary portion of a Type 1 font.
func (p *type1Parser) parseBinary(bytes []byte) error {
	var decrypted []byte
	// Sometimes, fonts use the hex format, so this needs to be converted before
	// decryption
	if isBinary(bytes) {
		decrypted = decrypt(bytes, eexecKey, 4)
	} else {
		decrypted = decrypt(hexToBinary(bytes), eexecKey, 4)
	}
	lexer, err := newType1Lexer(decrypted)
	if err != nil {
		return err
	}
	p.lexer = lexer

	// find /Private dict
	peekToken := p.lexer.PeekToken()
	for peekToken != nil && peekToken.Text() != "Private" {
		// for a more thorough validation, the presence of "begin" before Private
		// determines how code before and following charstrings should look
		// it is not currently checked anyway
		if _, err := p.lexer.NextToken(); err != nil {
			return err
		}
		peekToken = p.lexer.PeekToken()
	}
	if peekToken == nil {
		return errors.New("/Private token not found")
	}

	// Private dict
	if err := p.readNamed(kindLiteral, "Private"); err != nil {
		return err
	}
	lengthToken, err := p.read(kindInteger)
	if err != nil {
		return err
	}
	length := lengthToken.IntValue()
	if err := p.readNamed(kindName, "dict"); err != nil {
		return err
	}
	// actually could also be "/Private 10 dict def Private begin"
	// instead of the "dup"
	if _, err := p.readMaybe(kindName, "dup"); err != nil {
		return err
	}
	if err := p.readNamed(kindName, "begin"); err != nil {
		return err
	}

	lenIV := 4 // number of random bytes at start of charstring

	for i := 0; i < length; i++ {
		// premature end
		if !p.lexer.PeekKind(kindLiteral) {
			break
		}

		// key/value
		keyToken, err := p.read(kindLiteral)
		if err != nil {
			return err
		}

		switch key := keyToken.Text(); key {
		case "Subrs":
			if err := p.readSubrs(lenIV); err != nil {
				return err
			}
		case "OtherSubrs":
			if err := p.readOtherSubrs(); err != nil {
				return err
			}
		case "lenIV":
			value, err := p.readDictValue()
			if err != nil {
				return err
			}
			lenIV = value[0].IntValue()
		case "ND":
			if _, err := p.read(kindStartProc); err != nil {
				return err
			}
			// the access restrictions are not mandatory
			if _, err := p.readMaybe(kindName, "noaccess"); err != nil {
				return err
			}
			if err := p.readNamed(kindName, "def"); err != nil {
				return err
			}
			if _, err := p.read(kindEndProc); err != nil {
				return err
			}
			if _, err := p.readMaybe(kindName, "executeonly"); err != nil {
				return err
			}
			if _, err := p.readMaybe(kindName, "readonly"); err != nil {
				return err
			}
			if err := p.readNamed(kindName, "def"); err != nil {
				return err
			}
		case "NP":
			if _, err := p.read(kindStartProc); err != nil {
				return err
			}
			if _, err := p.readMaybe(kindName, "noaccess"); err != nil {
				return err
			}
			if _, err := p.read(kindName); err != nil {
				return err
			}
			if _, err := p.read(kindEndProc); err != nil {
				return err
			}
			if _, err := p.readMaybe(kindName, "executeonly"); err != nil {
				return err
			}
			if _, err := p.readMaybe(kindName, "readonly"); err != nil {
				return err
			}
			if err := p.readNamed(kindName, "def"); err != nil {
				return err
			}
		case "RD":
			// /RD {string currentfile exch readstring pop} bind executeonly def
			if _, err := p.read(kindStartProc); err != nil {
				return err
			}
			if err := p.readProcVoid(); err != nil {
				return err
			}
			if _, err := p.readMaybe(kindName, "bind"); err != nil {
				return err
			}
			if _, err := p.readMaybe(kindName, "executeonly"); err != nil {
				return err
			}
			if _, err := p.readMaybe(kindName, "readonly"); err != nil {
				return err
			}
			if err := p.readNamed(kindName, "def"); err != nil {
				return err
			}
		default:
			value, err := p.readDictValue()
			if err != nil {
				return err
			}
			if err := p.readPrivate(key, value); err != nil {
				return err
			}
		}
	}

	// some fonts have "2 index" here, others have "end noaccess put"
	// sometimes followed by "put". Either way, we just skip until
	// the /CharStrings dict is found
	for !(p.lexer.PeekKind(kindLiteral) && p.peekText() == "CharStrings") {
		next, err := p.lexer.NextToken()
		if err != nil {
			return err
		}
		if next == nil {
			return errors.New("Missing 'CharStrings' dictionary in type 1 font")
		}
	}

	// CharStrings dict
	if err := p.readNamed(kindLiteral, "CharStrings"); err != nil {
		return err
	}
	return p.readCharStrings(lenIV)
}

// readPrivate extracts values from the /Private dictionary.
func (p *type1Parser) readPrivate(key string, value []*token) error {
	var err error
	switch key {
	case "BlueValues":
		p.font.blueValues, err = arrayToNumbers(value)
	case "OtherBlues":
		p.font.otherBlues, err = arrayToNumbers(value)
	case "FamilyBlues":
		p.font.familyBlues, err = arrayToNumbers(value)
	case "FamilyOtherBlues":
		p.font.familyOtherBlues, err = arrayToNumbers(value)
	case "BlueScale":
		p.font.blueScale = value[0].FloatValue()
	case "BlueShift":
		p.font.blueShift = value[0].IntValue()
	case "BlueFuzz":
		p.font.blueFuzz = value[0].IntValue()
	case "StdHW":
		p.font.stdHW, err = arrayToNumbers(value)
	case "StdVW":
		p.font.stdVW, err = arrayToNumbers(value)
	case "StemSnapH":
		p.font.stemSnapH, err = arrayToNumbers(value)
	case "StemSnapV":
		p.font.stemSnapV, err = arrayToNumbers(value)
	case "ForceBold":
		p.font.forceBold = value[0].BooleanValue()
	case "LanguageGroup":
		p.font.languageGroup = value[0].IntValue()
	}
	return err
}

// readSubrs reads the /Subrs array, lenIV being the number of random bytes used
// in charstring encryption.
func (p *type1Parser) readSubrs(lenIV int) error {
	// allocate size (array indexes may not be in-order)
	lengthToken, err := p.read(kindInteger)
	if err != nil {
		return err
	}
	length := lengthToken.IntValue()
	for i := 0; i < length; i++ {
		p.font.subrs = append(p.font.subrs, nil)
	}
	if err := p.readNamed(kindName, "array"); err != nil {
		return err
	}

	for i := 0; i < length; i++ {
		// premature end
		if p.lexer.PeekToken() == nil {
			break
		}
		if !(p.lexer.PeekKind(kindName) && p.peekText() == "dup") {
			break
		}

		if err := p.readNamed(kindName, "dup"); err != nil {
			return err
		}
		index, err := p.read(kindInteger)
		if err != nil {
			return err
		}
		if _, err := p.read(kindInteger); err != nil {
			return err
		}

		// RD
		charstring, err := p.read(kindCharstring)
		if err != nil {
			return err
		}
		j := index.IntValue()
		if j < len(p.font.subrs) {
			p.font.subrs[j] = decrypt(charstring.Data(), charstringKey, lenIV)
		}
		if err := p.readPut(); err != nil {
			return err
		}
	}
	return p.readDef()
}

// readOtherSubrs reads the OtherSubrs, which are embedded PostScript procedures
// we can safely ignore.
func (p *type1Parser) readOtherSubrs() error {
	if p.lexer.PeekToken() == nil {
		return errors.New("Missing start token of OtherSubrs procedure")
	}
	if p.lexer.PeekKind(kindStartArray) {
		if _, err := p.readValue(); err != nil {
			return err
		}
		return p.readDef()
	}

	lengthToken, err := p.read(kindInteger)
	if err != nil {
		return err
	}
	length := lengthToken.IntValue()
	if err := p.readNamed(kindName, "array"); err != nil {
		return err
	}

	for i := 0; i < length; i++ {
		if err := p.readNamed(kindName, "dup"); err != nil {
			return err
		}
		if _, err := p.read(kindInteger); err != nil { // index
			return err
		}
		if _, err := p.readValue(); err != nil { // PostScript
			return err
		}
		if err := p.readPut(); err != nil {
			return err
		}
	}
	return p.readDef()
}

// readCharStrings reads the /CharStrings dictionary, lenIV being the number of
// random bytes used in charstring encryption.
func (p *type1Parser) readCharStrings(lenIV int) error {
	lengthToken, err := p.read(kindInteger)
	if err != nil {
		return err
	}
	length := lengthToken.IntValue()
	if err := p.readNamed(kindName, "dict"); err != nil {
		return err
	}
	// could actually be a sequence ending in "CharStrings begin", too
	// instead of the "dup begin"
	if err := p.readNamed(kindName, "dup"); err != nil {
		return err
	}
	if err := p.readNamed(kindName, "begin"); err != nil {
		return err
	}

	for i := 0; i < length; i++ {
		// premature end
		if p.lexer.PeekToken() == nil {
			break
		}
		if p.lexer.PeekKind(kindName) && p.peekText() == "end" {
			break
		}
		// key/value
		nameToken, err := p.read(kindLiteral)
		if err != nil {
			return err
		}
		name := nameToken.Text()

		// RD
		if _, err := p.read(kindInteger); err != nil {
			return err
		}
		charstring, err := p.read(kindCharstring)
		if err != nil {
			return err
		}
		if _, seen := p.font.charstrings[name]; !seen {
			p.font.charstringNames = append(p.font.charstringNames, name)
		}
		p.font.charstrings[name] = decrypt(charstring.Data(), charstringKey, lenIV)
		if err := p.readDef(); err != nil {
			return err
		}
	}

	// some fonts have one "end", others two
	return p.readNamed(kindName, "end")
	// since checking ends here, this does not matter ....
	// more thorough checking would see whether there is "begin" before /Private
	// and expect a "def" somewhere, otherwise a "put"
}

// readDef reads the sequence "noaccess def" or equivalent.
func (p *type1Parser) readDef() error {
	if _, err := p.readMaybe(kindName, "readonly"); err != nil {
		return err
	}
	// allows "noaccess ND" (not in the Type 1 spec)
	if _, err := p.readMaybe(kindName, "noaccess"); err != nil {
		return err
	}

	t, err := p.read(kindName)
	if err != nil {
		return err
	}
	switch t.Text() {
	case "ND", "|-":
		return nil
	case "noaccess":
		if t, err = p.read(kindName); err != nil {
			return err
		}
	}

	if t.Text() == "def" {
		return nil
	}
	return fmt.Errorf("Found %v but expected ND", t)
}

// readPut reads the sequence "noaccess put" or equivalent.
func (p *type1Parser) readPut() error {
	if _, err := p.readMaybe(kindName, "readonly"); err != nil {
		return err
	}

	t, err := p.read(kindName)
	if err != nil {
		return err
	}
	switch t.Text() {
	case "NP", "|":
		return nil
	case "noaccess":
		if t, err = p.read(kindName); err != nil {
			return err
		}
	}

	if t.Text() == "put" {
		return nil
	}
	return fmt.Errorf("Found %v but expected NP", t)
}

// peekText is the text of the next token, the empty string where there is none.
//
// Java calls lexer.peekToken().getText() and lets the null through to
// String.equals, which answers false; the empty string does the same here.
func (p *type1Parser) peekText() string {
	if t := p.lexer.PeekToken(); t != nil {
		return t.Text()
	}
	return ""
}

// read reads the next token and reports an error if it is not of the given
// kind. The token it returns is never nil.
func (p *type1Parser) read(kind tokenKind) (*token, error) {
	t, err := p.lexer.NextToken()
	if err != nil {
		return nil, err
	}
	if t == nil || t.Kind() != kind {
		return nil, fmt.Errorf("Found %v but expected %v", t, kind)
	}
	return t, nil
}

// readNamed reads the next token and reports an error if it is not of the given
// kind and does not have the given value.
func (p *type1Parser) readNamed(kind tokenKind, name string) error {
	t, err := p.read(kind)
	if err != nil {
		return err
	}
	if t.Text() != name {
		return fmt.Errorf("Found %v but expected %s", t, name)
	}
	return nil
}

// readMaybe reads the next token if and only if it is of the given kind and has
// the given value, and gives nil otherwise.
func (p *type1Parser) readMaybe(kind tokenKind, name string) (*token, error) {
	if p.lexer.PeekKind(kind) && p.peekText() == name {
		return p.lexer.NextToken()
	}
	return nil, nil
}

// decrypt is Type 1 decryption (eexec, charstring), cipherBytes being the
// cipher text, r the key and n the number of random bytes (lenIV).
func decrypt(cipherBytes []byte, r, n int) []byte {
	// lenIV of -1 means no encryption (not documented)
	if n == -1 {
		return cipherBytes
	}
	// empty charstrings and charstrings of insufficient length
	if len(cipherBytes) == 0 || n > len(cipherBytes) {
		return []byte{}
	}
	// decrypt
	const c1 = 52845
	const c2 = 22719
	plainBytes := make([]byte, len(cipherBytes)-n)
	for i := 0; i < len(cipherBytes); i++ {
		cipher := int(cipherBytes[i]) & 0xFF
		plain := cipher ^ r>>8
		if i >= n {
			plainBytes[i-n] = byte(plain)
		}
		// Java's & binds looser than +, so the whole sum is masked; Go's binds
		// tighter, and the brackets are what keeps the two the same.
		r = ((cipher+r)*c1 + c2) & 0xffff
	}
	return plainBytes
}

// isBinary checks whether the segment is binary or hex encoded. See Adobe Type
// 1 Font Format specification 7.2 eexec encryption.
func isBinary(bytes []byte) bool {
	if len(bytes) < 4 {
		return true
	}
	// "At least one of the first 4 ciphertext bytes must not be one of
	// the ASCII hexadecimal character codes (a code for 0-9, A-F, or a-f)."
	for i := 0; i < 4; i++ {
		by := bytes[i]
		if by != 0x0a && by != 0x0d && by != 0x20 && by != '\t' && hexDigit(by) == -1 {
			return true
		}
	}
	return false
}

func hexToBinary(bytes []byte) []byte {
	// calculate needed length
	length := 0
	for _, by := range bytes {
		if hexDigit(by) != -1 {
			length++
		}
	}
	res := make([]byte, length/2)
	r := 0
	prev := -1
	for _, by := range bytes {
		digit := hexDigit(by)
		if digit == -1 {
			continue
		}
		if prev == -1 {
			prev = digit
		} else {
			res[r] = byte(prev*16 + digit)
			r++
			prev = -1
		}
	}
	return res
}

// hexDigit is Java's Character.digit(c, 16) for the byte values the parser
// reads, giving -1 for anything that is not a hex digit.
func hexDigit(by byte) int {
	switch {
	case by >= '0' && by <= '9':
		return int(by - '0')
	case by >= 'a' && by <= 'f':
		return int(by-'a') + 10
	case by >= 'A' && by <= 'F':
		return int(by-'A') + 10
	}
	return -1
}
