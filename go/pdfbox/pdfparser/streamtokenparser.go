package pdfparser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// numberPattern matches numbers (integers or decimals). Safe from ReDoS: no
// overlapping quantifiers or character classes that cause backtracking. The
// optional decimal group is explicit and bounded.
var numberPattern = regexp.MustCompile(`^\d*(\.\d*)?$`)

// maxBinCharTestLength is how far past an "EI" the parser looks to decide
// whether the bytes there are an operator or more image data.
const maxBinCharTestLength = 10

// StreamTokenParser reads the operators and operands of a content stream.
//
// Port of org.apache.pdfbox.pdfparser.PDFStreamParser. Java extends COSParser;
// the port embeds ObjectParser, which holds the half of COSParser this needs.
// The name differs from the Java one because StreamParser is already the port
// of COSParser's stream-parsing half, which is a different class.
type StreamTokenParser struct {
	*ObjectParser

	binCharTestArr   [maxBinCharTestLength]byte
	inlineImageDepth int
	inlineOffset     int64
}

// NewStreamTokenParser returns a parser over the given content stream bytes.
//
// Port of the PDFStreamParser(byte[]) constructor.
func NewStreamTokenParser(b []byte) (*StreamTokenParser, error) {
	return NewStreamTokenParserSource(pdfio.NewReadBufferBytes(b))
}

// NewStreamTokenParserSource returns a parser reading from source.
//
// Java has a constructor taking a PDContentStream and calling
// getContentsForStreamParsing() on it; this stands in until that type exists.
func NewStreamTokenParserSource(source pdfio.RandomAccessRead) (*StreamTokenParser, error) {
	return &StreamTokenParser{ObjectParser: NewObjectParser(source, nil)}, nil
}

// Parse reads every token in the stream, closing the source when it is done.
//
// Port of parse. A token is a cos.Base or an *operator.Operator.
func (p *StreamTokenParser) Parse() ([]any, error) {
	streamObjects := make([]any, 0, 100)
	for {
		token, err := p.ParseNextToken()
		if err != nil {
			return nil, err
		}
		if token == nil {
			return streamObjects, nil
		}
		streamObjects = append(streamObjects, token)
	}
}

// ParseNextToken reads the next token, returning nil at the end of the stream.
//
// Port of parseNextToken.
func (p *StreamTokenParser) ParseNextToken() (any, error) {
	if p.source.IsClosed() {
		return nil, nil
	}
	if err := p.SkipSpaces(); err != nil {
		return nil, err
	}
	atEOF, err := p.IsEOF()
	if err != nil {
		return nil, err
	}
	if atEOF {
		return nil, p.Close()
	}
	c, err := p.peek()
	if err != nil {
		return nil, err
	}
	switch c {
	case '<':
		// pull off first left bracket
		if _, err := p.readByte(); err != nil {
			return nil, err
		}
		// check for second left bracket
		c, err = p.peek()
		if err != nil {
			return nil, err
		}
		if c == '<' {
			// put back first bracket
			if err := p.rewind(1); err != nil {
				return nil, err
			}
			dict, dictErr := p.ParseCOSDictionary(true)
			if dictErr != nil {
				pos, _ := p.source.Position()
				slog.Warn("stop reading invalid dictionary from content stream",
					"offset", pos, "err", dictErr)
				return nil, p.Close()
			}
			return dict, nil
		}
		hex, err := p.ParseCOSHexString()
		if err != nil {
			return nil, err
		}
		return hex, nil

	case '[':
		// array
		array, arrayErr := p.ParseCOSArray()
		if arrayErr != nil {
			pos, _ := p.source.Position()
			slog.Warn("stop reading invalid array from content stream",
				"offset", pos, "err", arrayErr)
			return nil, p.Close()
		}
		return array, nil

	case '(':
		// string
		str, err := p.ParseCOSLiteralString()
		if err != nil {
			return nil, err
		}
		return str, nil

	case '/':
		// name
		name, err := p.ParseCOSName()
		if err != nil {
			return nil, err
		}
		return name, nil

	case 'n':
		// null
		nullString, err := p.ReadString()
		if err != nil {
			return nil, err
		}
		if nullString == "null" {
			return cos.NullObject, nil
		}
		return operator.GetChecked(nullString)

	case 't', 'f':
		next, err := p.ReadString()
		if err != nil {
			return nil, err
		}
		switch next {
		case "true":
			return cos.True, nil
		case "false":
			return cos.False, nil
		}
		return operator.GetChecked(next)

	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-', '+', '.':
		// We will be filling buf with the rest of the number. Only allow 1 "."
		// and "-" and "+" at start of number.
		var buf bytes.Buffer
		buf.WriteByte(byte(c))
		if _, err := p.readByte(); err != nil {
			return nil, err
		}

		// Ignore double negative (this is consistent with Adobe Reader)
		if c == '-' {
			next, err := p.peek()
			if err != nil {
				return nil, err
			}
			if next == c {
				if _, err := p.readByte(); err != nil {
					return nil, err
				}
			}
		}

		dotNotRead := c != '.'
		for {
			c, err = p.peek()
			if err != nil {
				return nil, err
			}
			if !IsDigit(c) && !(dotNotRead && c == '.') && c != '-' {
				break
			}
			if c != '-' {
				// PDFBOX-4064: ignore "-" in the middle of a number
				buf.WriteByte(byte(c))
			}
			if _, err := p.readByte(); err != nil {
				return nil, err
			}
			if dotNotRead && c == '.' {
				dotNotRead = false
			}
		}
		s := buf.String()
		if s == "+" {
			// PDFBOX-5906
			slog.Warn("isolated '+' is ignored")
			return cos.NullObject, nil
		}
		number, err := cos.GetNumber(s)
		if err != nil {
			return nil, err
		}
		return number, nil

	case 'B':
		return p.parseBeginInlineImage()

	case 'I':
		// Special case for ID operator
		return p.parseInlineImageData()

	case ']':
		// some ']' around without its previous '['
		// this means a PDF is somewhat corrupt but we will continue to parse.
		if _, err := p.readByte(); err != nil {
			return nil, err
		}
		// must be a better solution than null...
		return cos.NullObject, nil

	default:
		// we must be an operator
		name, err := p.readOperator()
		if err != nil {
			return nil, err
		}
		name = javaTrim(name)
		if name != "" {
			return operator.GetChecked(name)
		}
	}
	return nil, nil
}

// parseBeginInlineImage handles the 'B' branch of parseNextToken, which is the
// BI operator and everything up to the image data behind it.
func (p *StreamTokenParser) parseBeginInlineImage() (any, error) {
	nextOperator, err := p.ReadString()
	if err != nil {
		return nil, err
	}
	beginImageOP, err := operator.GetChecked(nextOperator)
	if err != nil {
		return nil, err
	}
	if nextOperator != operator.BeginInlineImage {
		return beginImageOP, nil
	}

	p.inlineImageDepth++
	if p.inlineImageDepth > 1 {
		// PDFBOX-6038
		pos, _ := p.source.Position()
		return nil, fmt.Errorf("pdfparser: nested %q operator not allowed at offset %d, first: %d",
			operator.BeginInlineImage, pos, p.inlineOffset)
	}
	p.inlineOffset, _ = p.source.Position()

	imageParams := cos.NewDictionary()
	beginImageOP.SetImageParameters(imageParams)
	var nextToken any
	for {
		nextToken, err = p.ParseNextToken()
		if err != nil {
			return nil, err
		}
		name, isName := nextToken.(*cos.Name)
		if !isName {
			break
		}
		value, err := p.ParseNextToken()
		if err != nil {
			return nil, err
		}
		base, isBase := value.(cos.Base)
		if !isBase {
			slog.Warn("unexpected token in inline image dictionary", "offset", p.offsetOrEOF())
			break
		}
		imageParams.SetItem(name, base)
	}
	// final token will be the image data, maybe??
	if imageData, isOperator := nextToken.(*operator.Operator); isOperator {
		if len(imageData.ImageData()) == 0 {
			slog.Warn("empty inline image", "offset", p.offsetOrEOF())
		}
		beginImageOP.SetImageData(imageData.ImageData())
		p.inlineImageDepth--
	} else {
		slog.Warn("unexpected token", "token", nextToken, "offset", p.offsetOrEOF(),
			"expected", operator.BeginInlineImageData)
	}
	return beginImageOP, nil
}

// parseInlineImageData handles the 'I' branch of parseNextToken: the ID
// operator and the raw bytes that follow it, up to the closing EI.
func (p *StreamTokenParser) parseInlineImageData() (any, error) {
	first, err := p.readByte()
	if err != nil {
		return nil, err
	}
	second, err := p.readByte()
	if err != nil {
		return nil, err
	}
	id := string([]rune{javaChar(first), javaChar(second)})
	if id != operator.BeginInlineImageData {
		currentPosition, _ := p.source.Position()
		if err := p.Close(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("pdfparser: expected operator 'ID', got %q at stream offset %d",
			id, currentPosition)
	}

	var imageData bytes.Buffer
	// skip one line break (CR, LF or CRLF) or any one-byte whitespace
	skipped, err := p.SkipLinebreak()
	if err != nil {
		return nil, err
	}
	if !skipped {
		next, err := p.peek()
		if err != nil {
			return nil, err
		}
		if IsWhitespace(next) {
			// pull off the whitespace character
			if _, err := p.readByte(); err != nil {
				return nil, err
			}
		}
	}
	lastByte, err := p.readByte()
	if err != nil {
		return nil, err
	}
	currentByte, err := p.readByte()
	if err != nil {
		return nil, err
	}
	// PDF spec is kinda unclear about this. Should a whitespace always appear
	// before EI? Not sure, so that we just read until EI<whitespace>.
	// Be aware not all kind of whitespaces are allowed here. see PDFBOX-1561
	for {
		done, err := p.atEndOfInlineImage(lastByte, currentByte)
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
		atEOF, err := p.IsEOF()
		if err != nil {
			return nil, err
		}
		if atEOF {
			break
		}
		imageData.WriteByte(byte(lastByte))
		lastByte = currentByte
		currentByte, err = p.readByte()
		if err != nil {
			return nil, err
		}
	}
	// the EI operator isn't unread, as it won't be processed anyway
	beginImageDataOP, err := operator.GetChecked(operator.BeginInlineImageData)
	if err != nil {
		return nil, err
	}
	// save the image data to the operator, so that it can be accessed later
	beginImageDataOP.SetImageData(imageData.Bytes())
	return beginImageDataOP, nil
}

// atEndOfInlineImage reports whether the two bytes just read close the image.
// It is the short-circuiting conjunction from the Java loop condition, kept in
// one place so that the more expensive tests still run only when they must.
func (p *StreamTokenParser) atEndOfInlineImage(lastByte, currentByte int) (bool, error) {
	if lastByte != 'E' || currentByte != 'I' {
		return false, nil
	}
	spaceOrReturn, err := p.hasNextSpaceOrReturn()
	if err != nil || !spaceOrReturn {
		return false, err
	}
	return p.hasNoFollowingBinData()
}

// hasNoFollowingBinData looks up an amount of bytes if they contain only ASCII
// characters (no control sequences etc.), and that these ASCII characters begin
// with a sequence of 1-3 non-blank characters between blanks.
//
// It reports true if the next bytes are probably printable ASCII characters
// starting with a PDF operator.
func (p *StreamTokenParser) hasNoFollowingBinData() (bool, error) {
	// as suggested in PDFBOX-1164
	readBytes, err := p.source.Read(p.binCharTestArr[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	noBinData := true
	startOpIdx := -1
	endOpIdx := -1
	s := ""

	if readBytes > 0 {
		for bIdx := 0; bIdx < readBytes; bIdx++ {
			// Java's byte is signed, so a byte above 0x7f is negative here and
			// the first test catches it — that is what the comment below means.
			b := int8(p.binCharTestArr[bIdx])
			if b != 0 && b < 0x09 || b > 0x0a && b < 0x20 && b != 0x0d {
				// control character or > 0x7f -> we have binary data
				noBinData = false
				break
			}
			// find the start of a PDF operator
			if startOpIdx == -1 && !(b == 0 || b == 9 || b == 0x20 || b == 0x0a || b == 0x0d) {
				startOpIdx = bIdx
			} else if startOpIdx != -1 && endOpIdx == -1 &&
				(b == 0 || b == 9 || b == 0x20 || b == 0x0a || b == 0x0d) {
				endOpIdx = bIdx
			}
		}

		// PDFBOX-3742: just assuming that 1-3 non blanks is a PDF operator isn't enough
		if noBinData && endOpIdx != -1 && startOpIdx != -1 {
			// usually, the operator here is Q, sometimes EMC (PDFBOX-2376),
			// S (PDFBOX-3784), or a number (PDFBOX-5957)
			s = string(p.binCharTestArr[startOpIdx:endOpIdx])
			if s != "Q" && s != "EMC" && s != "S" && !numberPattern.MatchString(s) {
				// operator is not Q, not EMC, not S, nor a number -> assume binary data
				noBinData = false
			}
		}

		// only if not close to EOF
		if noBinData && startOpIdx != -1 && readBytes == maxBinCharTestLength {
			if endOpIdx == -1 {
				endOpIdx = maxBinCharTestLength
				s = string(p.binCharTestArr[startOpIdx:endOpIdx])
			}
			// look for token of 3 chars max or a number
			if endOpIdx-startOpIdx > 3 && !numberPattern.MatchString(s) {
				noBinData = false // "operator" too long, assume binary data
			}
		}
		if err := p.rewind(int64(readBytes)); err != nil {
			return false, err
		}
	}
	if !noBinData {
		pos, _ := p.source.Position()
		slog.Warn("ignoring 'EI' assumed to be in the middle of inline image",
			"offset", pos, "s", s)
	}
	return noBinData, nil
}

// readOperator reads an operator from the stream.
func (p *StreamTokenParser) readOperator() (string, error) {
	if err := p.SkipSpaces(); err != nil {
		return "", err
	}

	// average string size is around 2 and the normal string buffer size is
	// about 16 so lets save some space.
	var buffer strings.Builder
	nextChar, err := p.peek()
	if err != nil {
		return "", err
	}
	for nextChar != eof && // EOF
		!IsWhitespace(nextChar) &&
		nextChar != '[' &&
		nextChar != '<' &&
		nextChar != '(' &&
		nextChar != '/' &&
		nextChar != '%' &&
		(nextChar < '0' || nextChar > '9') {
		currentChar, err := p.readByte()
		if err != nil {
			return "", err
		}
		nextChar, err = p.peek()
		if err != nil {
			return "", err
		}
		buffer.WriteByte(byte(currentChar))
		// Type3 Glyph description has operators with a number in the name
		if currentChar == 'd' && (nextChar == '0' || nextChar == '1') {
			digit, err := p.readByte()
			if err != nil {
				return "", err
			}
			buffer.WriteByte(byte(digit))
			nextChar, err = p.peek()
			if err != nil {
				return "", err
			}
		}
	}
	return buffer.String(), nil
}

// isSpaceOrReturn reports whether c is a line feed, a carriage return or a
// space. It is deliberately narrower than IsWhitespace; see PDFBOX-1561.
func isSpaceOrReturn(c int) bool {
	return c == 10 || c == 13 || c == 32
}

// hasNextSpaceOrReturn reports whether the next char is a space or a return.
func (p *StreamTokenParser) hasNextSpaceOrReturn() (bool, error) {
	c, err := p.peek()
	if err != nil {
		return false, err
	}
	return isSpaceOrReturn(c), nil
}

// Close closes the underlying source.
func (p *StreamTokenParser) Close() error {
	if p.source != nil && !p.source.IsClosed() {
		return p.source.Close()
	}
	return nil
}

// offsetOrEOF returns the position to log, or the string "EOF" once the source
// is closed and has none. Java builds the same value inline at each log call.
func (p *StreamTokenParser) offsetOrEOF() any {
	if p.source.IsClosed() {
		return "EOF"
	}
	pos, _ := p.source.Position()
	return pos
}

// javaChar widens a byte the way Java's (char) cast does, so that the -1 of a
// read past the end becomes U+FFFF rather than a replacement character.
func javaChar(c int) rune {
	return rune(uint16(c))
}

// javaTrim removes leading and trailing characters at or below U+0020, which is
// what Java's String.trim does. strings.TrimSpace is not the same: it would
// leave the control characters an operator name can pick up from a corrupt
// stream, and strip Unicode spaces Java keeps.
func javaTrim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r <= ' ' })
}
