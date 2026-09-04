package cmap

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/fontbox/resources"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

const (
	markEndOfDictionary = ">>"
	markEndOfArray      = "]"
)

// The token types parseNextToken hands back. Java returns a bare Object and
// switches on instanceof; these are the types it can be.
type (
	// literalName is a /Name token.
	literalName struct{ name string }
	// operator is a bare keyword token.
	operator struct{ op string }
)

// Parser parses a CMap stream.
//
// Port of org.apache.fontbox.cmap.CMapParser.
type Parser struct {
	tokenParserByteBuffer [512]byte

	strictMode bool
}

// NewParser creates a new instance of Parser.
func NewParser() *Parser {
	return &Parser{}
}

// NewParserStrict creates a new instance of Parser, strictMode activating the
// strict mode used for inline CMaps.
func NewParserStrict(strictMode bool) *Parser {
	return &Parser{strictMode: strictMode}
}

// ParsePredefined parses a predefined CMap by name. The CMap it returns is
// never nil unless the error is.
func (p *Parser) ParsePredefined(name string) (*CMap, error) {
	randomAccessRead, err := p.getExternalCMap(name)
	if err != nil {
		return nil, err
	}
	defer pdfio.CloseQuietly(randomAccessRead)
	// deactivate strict mode
	p.strictMode = false
	return p.Parse(randomAccessRead)
}

// Parse parses the source and creates a cmap object. The CMap it returns is
// never nil unless the error is.
func (p *Parser) Parse(randomAccessRead pdfio.RandomAccessRead) (*CMap, error) {
	result := newCMap()
	var previousToken any
	token, err := p.parseNextToken(randomAccessRead)
	if err != nil {
		return nil, err
	}
	for token != nil {
		if op, ok := token.(operator); ok {
			if op.op == "endcmap" {
				// end of CMap reached, stop reading as there isn't any
				// interesting info anymore
				break
			}

			previousLiteral, previousIsLiteral := previousToken.(literalName)
			if op.op == "usecmap" && previousIsLiteral {
				if err := p.parseUsecmap(previousLiteral, result); err != nil {
					return nil, err
				}
			} else if count, ok := asNumber(previousToken); ok {
				switch op.op {
				case "begincodespacerange":
					err = p.parseBegincodespacerange(count, randomAccessRead, result)
				case "beginbfchar":
					err = p.parseBeginbfchar(count, randomAccessRead, result)
				case "beginbfrange":
					err = p.parseBeginbfrange(count, randomAccessRead, result)
				case "begincidchar":
					err = p.parseBegincidchar(count, randomAccessRead, result)
				case "begincidrange":
					// Java tests previousToken instanceof Integer here, so a
					// real number does not open a cidrange.
					if numberOfLines, isInt := previousToken.(int); isInt {
						err = p.parseBegincidrange(numberOfLines, randomAccessRead, result)
					}
				}
				if err != nil {
					return nil, err
				}
			}
		} else if literal, ok := token.(literalName); ok {
			if err := p.parseLiteralName(literal, randomAccessRead, result); err != nil {
				return nil, err
			}
		}
		previousToken = token
		token, err = p.parseNextToken(randomAccessRead)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// asNumber reports whether the token is one of the two number types the parser
// produces, and gives its int value -- Java's Number.intValue.
func asNumber(token any) (int, bool) {
	switch v := token.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	}
	return 0, false
}

func (p *Parser) parseUsecmap(useCmapName literalName, result *CMap) error {
	randomAccessRead, err := p.getExternalCMap(useCmapName.name)
	if err != nil {
		return err
	}
	defer pdfio.CloseQuietly(randomAccessRead)
	useCMap, err := p.Parse(randomAccessRead)
	if err != nil {
		return err
	}
	result.useCmap(useCMap)
	return nil
}

func (p *Parser) parseLiteralName(literal literalName, randomAccessRead pdfio.RandomAccessRead,
	result *CMap) error {
	next, err := p.parseNextTokenFor(literal.name, randomAccessRead)
	if err != nil {
		return err
	}
	switch literal.name {
	case "WMode":
		if wmode, ok := next.(int); ok {
			result.SetWMode(wmode)
		}
	case "CMapName":
		if name, ok := next.(literalName); ok {
			result.SetName(name.name)
		}
	case "CMapVersion":
		if number, ok := next.(int); ok {
			result.SetVersion(strconv.Itoa(number))
		} else if number, ok := next.(float64); ok {
			result.SetVersion(javaDoubleString(number))
		} else if version, ok := next.(string); ok {
			result.SetVersion(version)
		}
	case "CMapType":
		if cmapType, ok := next.(int); ok {
			result.SetType(cmapType)
		}
	case "Registry":
		if registry, ok := next.(string); ok {
			result.SetRegistry(registry)
		}
	case "Ordering":
		if ordering, ok := next.(string); ok {
			result.SetOrdering(ordering)
		}
	case "Supplement":
		if supplement, ok := next.(int); ok {
			result.SetSupplement(supplement)
		}
	}
	return nil
}

// parseNextTokenFor reads the value belonging to one of the literal names the
// switch above handles, and nothing at all for any other name -- Java's switch
// reads the next token inside each case, so its default arm consumes nothing.
func (p *Parser) parseNextTokenFor(name string, randomAccessRead pdfio.RandomAccessRead) (any, error) {
	switch name {
	case "WMode", "CMapName", "CMapVersion", "CMapType", "Registry", "Ordering", "Supplement":
		return p.parseNextToken(randomAccessRead)
	}
	return nil, nil
}

// javaDoubleString renders a float64 the way Java's Double.toString does, which
// is what CMapVersion records for a real number.
func javaDoubleString(value float64) string {
	s := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

// checkExpectedOperator returns an error if expectedOperatorName does not equal
// op.op, rangeName naming the range in which the operator is expected (without
// a tilde character), to be used in the message.
func checkExpectedOperator(op operator, expectedOperatorName, rangeName string) error {
	if op.op != expectedOperatorName {
		return fmt.Errorf("Error : ~%s contains an unexpected operator : %s",
			rangeName, op.op)
	}
	return nil
}

func (p *Parser) parseBegincodespacerange(cosCount int,
	randomAccessRead pdfio.RandomAccessRead, result *CMap) error {
	for j := 0; j < cosCount; j++ {
		nextToken, err := p.parseNextToken(randomAccessRead)
		if err != nil {
			return err
		}
		if op, ok := nextToken.(operator); ok {
			return checkExpectedOperator(op, "endcodespacerange", "codespacerange")
		}
		startRange, ok := nextToken.([]byte)
		if !ok {
			return errors.New("start range missing")
		}
		endRange, err := p.parseByteArray(randomAccessRead)
		if err != nil {
			return err
		}
		codespaceRange, err := NewCodespaceRange(startRange, endRange)
		if err != nil {
			return err
		}
		result.addCodespaceRange(codespaceRange)
	}
	return nil
}

func (p *Parser) parseBeginbfchar(cosCount int, randomAccessRead pdfio.RandomAccessRead,
	result *CMap) error {
	for j := 0; j < cosCount; j++ {
		nextToken, err := p.parseNextToken(randomAccessRead)
		if err != nil {
			return err
		}
		if op, ok := nextToken.(operator); ok {
			return checkExpectedOperator(op, "endbfchar", "bfchar")
		}
		inputCode, ok := nextToken.([]byte)
		if !ok {
			return errors.New("input code missing")
		}
		nextToken, err = p.parseNextToken(randomAccessRead)
		if err != nil {
			return err
		}
		switch value := nextToken.(type) {
		case []byte:
			result.addCharMapping(inputCode, createStringFromBytes(value))
		case literalName:
			result.addCharMapping(inputCode, value.name)
		default:
			return fmt.Errorf("Error parsing CMap beginbfchar, expected"+
				"{COSString or COSName} and not %v", nextToken)
		}
	}
	return nil
}

func (p *Parser) parseBegincidrange(numberOfLines int,
	randomAccessRead pdfio.RandomAccessRead, result *CMap) error {
	for n := 0; n < numberOfLines; n++ {
		nextToken, err := p.parseNextToken(randomAccessRead)
		if err != nil {
			return err
		}
		if op, ok := nextToken.(operator); ok {
			return checkExpectedOperator(op, "endcidrange", "cidrange")
		}
		startCode, ok := nextToken.([]byte)
		if !ok {
			return errors.New("start code missing")
		}
		endCode, err := p.parseByteArray(randomAccessRead)
		if err != nil {
			return err
		}
		mappedCode, err := p.parseInteger(randomAccessRead)
		if err != nil {
			return err
		}
		if len(startCode) != len(endCode) {
			return errors.New("Error : ~cidrange values must not have different byte lengths")
		}
		// some CMaps are using CID ranges to map single values
		if bytes.Equal(startCode, endCode) {
			result.addCIDMapping(startCode, mappedCode)
		} else {
			result.addCIDRange(startCode, endCode, mappedCode)
		}
	}
	return nil
}

func (p *Parser) parseBegincidchar(cosCount int, randomAccessRead pdfio.RandomAccessRead,
	result *CMap) error {
	for j := 0; j < cosCount; j++ {
		nextToken, err := p.parseNextToken(randomAccessRead)
		if err != nil {
			return err
		}
		if op, ok := nextToken.(operator); ok {
			return checkExpectedOperator(op, "endcidchar", "cidchar")
		}
		inputCode, ok := nextToken.([]byte)
		if !ok {
			return errors.New("input code missing")
		}
		mappedCID, err := p.parseInteger(randomAccessRead)
		if err != nil {
			return err
		}
		result.addCIDMapping(inputCode, mappedCID)
	}
	return nil
}

func (p *Parser) parseBeginbfrange(cosCount int, randomAccessRead pdfio.RandomAccessRead,
	result *CMap) error {
	for j := 0; j < cosCount; j++ {
		nextToken, err := p.parseNextToken(randomAccessRead)
		if err != nil {
			return err
		}
		if op, ok := nextToken.(operator); ok {
			return checkExpectedOperator(op, "endbfrange", "bfrange")
		}
		startCode, ok := nextToken.([]byte)
		if !ok {
			return errors.New("start code missing")
		}
		nextToken, err = p.parseNextToken(randomAccessRead)
		if err != nil {
			return err
		}
		if op, ok := nextToken.(operator); ok {
			return checkExpectedOperator(op, "endbfrange", "bfrange")
		}
		endCode, ok := nextToken.([]byte)
		if !ok {
			return errors.New("end code missing")
		}
		start := ToInt(startCode)
		end := ToInt(endCode)
		// end has to be bigger than start or equal
		if end < start {
			// PDFBOX-4550: likely corrupt stream
			break
		}
		nextToken, err = p.parseNextToken(randomAccessRead)
		if err != nil {
			return err
		}
		if array, ok := nextToken.([]any); ok {
			// ignore empty and malformed arrays
			if len(array) != 0 && len(array) >= end-start {
				p.addMappingFrombfrangeList(result, startCode, array)
			}
			continue
		}
		// PDFBOX-3807: ignore null
		tokenBytes, ok := nextToken.([]byte)
		if !ok {
			continue
		}
		// PDFBOX-3450: ignore <>
		if len(tokenBytes) == 0 {
			continue
		}
		// PDFBOX-4720:
		// some pdfs use the malformed bfrange <0000> <FFFF> <0000>. Add support
		// by adding a identity mapping for the whole range instead of cutting
		// it after 255 entries
		// TODO find a more efficient method to represent all values for a
		// identity mapping
		if len(tokenBytes) == 2 && start == 0 && end == 0xffff &&
			tokenBytes[0] == 0 && tokenBytes[1] == 0 {
			for i := 0; i < 256; i++ {
				startCode[0] = byte(i)
				startCode[1] = 0
				tokenBytes[0] = byte(i)
				tokenBytes[1] = 0
				p.addMappingFrombfrange(result, startCode, 256, tokenBytes)
			}
		} else {
			p.addMappingFrombfrange(result, startCode, end-start+1, tokenBytes)
		}
	}
	return nil
}

func (p *Parser) addMappingFrombfrangeList(cmap *CMap, startCode []byte, tokenBytesList []any) {
	for _, token := range tokenBytesList {
		// Java declares the list as List<byte[]> and blows up on anything else;
		// the port skips what it cannot cast rather than panicking, because the
		// array is only reached when it is long enough to cover the range.
		tokenBytes, ok := token.([]byte)
		if !ok {
			continue
		}
		cmap.addCharMapping(startCode, createStringFromBytes(tokenBytes))
		increment(startCode, len(startCode)-1, false)
	}
}

func (p *Parser) addMappingFrombfrange(cmap *CMap, startCode []byte, values int,
	tokenBytes []byte) {
	for i := 0; i < values; i++ {
		cmap.addCharMapping(startCode, createStringFromBytes(tokenBytes))
		if !increment(tokenBytes, len(tokenBytes)-1, p.strictMode) {
			// overflow detected -> stop adding further mappings
			break
		}
		increment(startCode, len(startCode)-1, false)
	}
}

// getExternalCMap returns a RandomAccessRead containing the given "use" CMap.
func (p *Parser) getExternalCMap(name string) (pdfio.RandomAccessRead, error) {
	// Validate name to point to the (predefined) resources in the classpath.
	if name == "" || strings.ContainsAny(name, "/\\") || name[0] == '.' {
		return nil, fmt.Errorf("Error: Invalid CMap name %s", name)
	}

	data, err := resources.Read("cmap/" + name)
	if err != nil {
		return nil, fmt.Errorf("Error: Could not find referenced cmap stream %s", name)
	}
	return pdfio.NewReadBufferBytes(data), nil
}

// readByte reads one byte, returning -1 at the end of the source, as Java's
// RandomAccessRead.read does.
func readByte(randomAccessRead pdfio.RandomAccessRead) (int, error) {
	b, err := randomAccessRead.ReadByte()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return -1, nil
		}
		return -1, err
	}
	return int(b), nil
}

// rewind steps the cursor back one byte, as Java's rewind(1) does.
func rewind(randomAccessRead pdfio.RandomAccessRead) error {
	_, err := randomAccessRead.Seek(-1, io.SeekCurrent)
	return err
}

func (p *Parser) parseNextToken(randomAccessRead pdfio.RandomAccessRead) (any, error) {
	nextByte, err := readByte(randomAccessRead)
	if err != nil {
		return nil, err
	}
	// skip whitespace
	for nextByte == 0x09 || nextByte == 0x20 || nextByte == 0x0D || nextByte == 0x0A {
		nextByte, err = readByte(randomAccessRead)
		if err != nil {
			return nil, err
		}
	}
	switch nextByte {
	case '%':
		return p.readLine(randomAccessRead, nextByte)
	case '(':
		return p.readString(randomAccessRead)
	case '>':
		next, err := readByte(randomAccessRead)
		if err != nil {
			return nil, err
		}
		if next == '>' {
			return markEndOfDictionary, nil
		}
		return nil, errors.New("Error: expected the end of a dictionary.")
	case ']':
		return markEndOfArray, nil
	case '[':
		return p.readArray(randomAccessRead)
	case '<':
		return p.readDictionary(randomAccessRead)
	case '/':
		return p.readLiteralName(randomAccessRead)
	case -1:
		// EOF returning null
		return nil, nil
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.readNumber(randomAccessRead, nextByte)
	default:
		return p.readOperator(randomAccessRead, nextByte)
	}
}

func (p *Parser) parseInteger(randomAccessRead pdfio.RandomAccessRead) (int, error) {
	nextToken, err := p.parseNextToken(randomAccessRead)
	if err != nil {
		return 0, err
	}
	if nextToken == nil {
		return 0, errors.New("expected integer value is missing")
	}
	if value, ok := nextToken.(int); ok {
		return value, nil
	}
	return 0, errors.New("invalid type for next token")
}

func (p *Parser) parseByteArray(randomAccessRead pdfio.RandomAccessRead) ([]byte, error) {
	nextToken, err := p.parseNextToken(randomAccessRead)
	if err != nil {
		return nil, err
	}
	if nextToken == nil {
		return nil, errors.New("expected byte[] value is missing")
	}
	if value, ok := nextToken.([]byte); ok {
		return value, nil
	}
	return nil, errors.New("invalid type for next token")
}

func (p *Parser) readArray(randomAccessRead pdfio.RandomAccessRead) ([]any, error) {
	list := []any{}
	nextToken, err := p.parseNextToken(randomAccessRead)
	if err != nil {
		return nil, err
	}
	for nextToken != nil && nextToken != any(markEndOfArray) {
		list = append(list, nextToken)
		nextToken, err = p.parseNextToken(randomAccessRead)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (p *Parser) readString(randomAccessRead pdfio.RandomAccessRead) (string, error) {
	var buffer strings.Builder
	stringByte, err := readByte(randomAccessRead)
	if err != nil {
		return "", err
	}
	for stringByte != -1 && stringByte != ')' {
		buffer.WriteRune(rune(stringByte))
		stringByte, err = readByte(randomAccessRead)
		if err != nil {
			return "", err
		}
	}
	return buffer.String(), nil
}

func (p *Parser) readLine(randomAccessRead pdfio.RandomAccessRead, firstByte int) (string, error) {
	// header operations, for now return the entire line
	// may need to smarter in the future
	var buffer strings.Builder
	buffer.WriteRune(rune(firstByte))
	if err := readUntilEndOfLine(randomAccessRead, &buffer); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (p *Parser) readLiteralName(randomAccessRead pdfio.RandomAccessRead) (literalName, error) {
	var buffer strings.Builder
	stringByte, err := readByte(randomAccessRead)
	if err != nil {
		return literalName{}, err
	}
	for !isWhitespaceOrEOF(stringByte) && !isDelimiter(stringByte) {
		buffer.WriteRune(rune(stringByte))
		stringByte, err = readByte(randomAccessRead)
		if err != nil {
			return literalName{}, err
		}
	}
	if isDelimiter(stringByte) {
		if err := rewind(randomAccessRead); err != nil {
			return literalName{}, err
		}
	}
	return literalName{buffer.String()}, nil
}

func (p *Parser) readOperator(randomAccessRead pdfio.RandomAccessRead, firstByte int) (operator, error) {
	var buffer strings.Builder
	buffer.WriteRune(rune(firstByte))
	nextByte, err := readByte(randomAccessRead)
	if err != nil {
		return operator{}, err
	}

	// newline separator may be missing in malformed CMap files
	// see PDFBOX-2035
	for !isWhitespaceOrEOF(nextByte) && !isDelimiter(nextByte) && !isDigit(nextByte) {
		buffer.WriteRune(rune(nextByte))
		nextByte, err = readByte(randomAccessRead)
		if err != nil {
			return operator{}, err
		}
	}
	if isDelimiter(nextByte) || isDigit(nextByte) {
		if err := rewind(randomAccessRead); err != nil {
			return operator{}, err
		}
	}
	return operator{buffer.String()}, nil
}

func (p *Parser) readNumber(randomAccessRead pdfio.RandomAccessRead, firstByte int) (any, error) {
	var buffer strings.Builder
	buffer.WriteRune(rune(firstByte))
	nextByte, err := readByte(randomAccessRead)
	if err != nil {
		return nil, err
	}

	for !isWhitespaceOrEOF(nextByte) && (isDigit(nextByte) || nextByte == '.') {
		buffer.WriteRune(rune(nextByte))
		nextByte, err = readByte(randomAccessRead)
		if err != nil {
			return nil, err
		}
	}
	if nextByte != -1 {
		if err := rewind(randomAccessRead); err != nil {
			return nil, err
		}
	}
	value := buffer.String()
	if strings.ContainsRune(value, '.') {
		number, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid number '%s': %w", value, err)
		}
		return number, nil
	}
	// Java's Integer.valueOf rejects anything outside the 32-bit range.
	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("Invalid number '%s': %w", value, err)
	}
	return int(number), nil
}

func (p *Parser) readDictionary(randomAccessRead pdfio.RandomAccessRead) (any, error) {
	theNextByte, err := readByte(randomAccessRead)
	if err != nil {
		return nil, err
	}
	if theNextByte == '<' {
		result := map[string]any{}
		// we are reading a dictionary
		key, err := p.parseNextToken(randomAccessRead)
		if err != nil {
			return nil, err
		}
		for {
			name, ok := key.(literalName)
			if !ok || name.name == markEndOfDictionary {
				break
			}
			value, err := p.parseNextToken(randomAccessRead)
			if err != nil {
				return nil, err
			}
			result[name.name] = value
			key, err = p.parseNextToken(randomAccessRead)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	// won't read more than 512 bytes
	multiplyer := 16
	bufferIndex := -1
	for theNextByte != -1 && theNextByte != '>' {
		// all kind of whitespaces may occur in malformed CMap files
		// see PDFBOX-2035
		if isWhitespaceOrEOF(theNextByte) {
			// skipping whitespaces
			theNextByte, err = readByte(randomAccessRead)
			if err != nil {
				return nil, err
			}
			continue
		}
		intValue := 0
		switch {
		case theNextByte >= '0' && theNextByte <= '9':
			intValue = theNextByte - '0'
		case theNextByte >= 'A' && theNextByte <= 'F':
			intValue = 10 + theNextByte - 'A'
		case theNextByte >= 'a' && theNextByte <= 'f':
			intValue = 10 + theNextByte - 'a'
		default:
			return nil, fmt.Errorf("Error: expected hex character and not %c:%d",
				rune(theNextByte), theNextByte)
		}
		intValue *= multiplyer
		if multiplyer == 16 {
			bufferIndex++
			if bufferIndex >= len(p.tokenParserByteBuffer) {
				return nil, fmt.Errorf("cmap token ist larger than buffer size %d",
					len(p.tokenParserByteBuffer))
			}
			p.tokenParserByteBuffer[bufferIndex] = 0
			multiplyer = 1
		} else {
			multiplyer = 16
		}
		p.tokenParserByteBuffer[bufferIndex] += byte(intValue)
		theNextByte, err = readByte(randomAccessRead)
		if err != nil {
			return nil, err
		}
	}
	finalResult := make([]byte, bufferIndex+1)
	copy(finalResult, p.tokenParserByteBuffer[:bufferIndex+1])
	return finalResult, nil
}

func readUntilEndOfLine(randomAccessRead pdfio.RandomAccessRead, buf *strings.Builder) error {
	nextByte, err := readByte(randomAccessRead)
	if err != nil {
		return err
	}
	for nextByte != -1 && nextByte != 0x0D && nextByte != 0x0A {
		buf.WriteRune(rune(nextByte))
		nextByte, err = readByte(randomAccessRead)
		if err != nil {
			return err
		}
	}
	return nil
}

func isWhitespaceOrEOF(aByte int) bool {
	switch aByte {
	case -1, 0x20, 0x0D, 0x0A:
		return true
	default:
		return false
	}
}

// isDelimiter says whether this is a standard PDF delimiter character.
func isDelimiter(aByte int) bool {
	switch aByte {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	default:
		return false
	}
}

// isDigit is Java's Character.isDigit for the byte values the parser reads,
// which for a value below 256 is the ASCII digits alone.
func isDigit(aByte int) bool {
	return aByte >= '0' && aByte <= '9'
}

func increment(data []byte, position int, useStrictMode bool) bool {
	if position < 0 {
		return false
	}
	if position > 0 && data[position] == 255 {
		// PDFBOX-4661: avoid overflow of the last byte, all following values
		// are undefined
		// PDFBOX-5090: strict mode has to be used for CMaps within pdfs
		if useStrictMode {
			return false
		}
		data[position] = 0
		increment(data, position-1, useStrictMode)
	} else {
		data[position] = data[position] + 1
	}
	return true
}

func createStringFromBytes(b []byte) string {
	if len(b) <= 2 {
		value, _ := GetMapping(b)
		return value
	}
	return decodeUTF16BE(b)
}
