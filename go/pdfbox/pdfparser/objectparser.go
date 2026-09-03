package pdfparser

import (
	"bytes"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Limits carried over from COSParser. All three exist because the values they
// bound are attacker-controlled.
const (
	// maxRecursionDepth bounds nesting in parseDirObject and its callees.
	maxRecursionDepth = 500
	// objectNumberThreshold rejects an object number of more than ten digits.
	objectNumberThreshold = 10000000000
	// generationNumberThreshold rejects a generation above 65535.
	generationNumberThreshold = 65535
)

const (
	endobjString    = "endobj"
	endstreamString = "endstream"
)

// ObjectParser reads COS objects out of a source.
//
// Port of the object-parsing half of org.apache.pdfbox.pdfparser.COSParser.
// Java has one class extending BaseParser; the port embeds BaseParser here and
// leaves the file-level concerns — the header, the trailer, the cross-reference
// sections — to the parsers that build on this one.
type ObjectParser struct {
	*BaseParser

	// document supplies the object pool that indirect references resolve
	// through. It may be nil when parsing a content stream, where a reference
	// is an error.
	document *cos.Document

	recursionDepth int

	// keyCache maps a packed object key to the key instance the
	// cross-reference table already holds, so that the pool is not filled with
	// duplicates. Java notes that iterating the xref table per lookup gets slow
	// on large files, which is what the cache is for.
	keyCache map[int64]*cos.ObjectKey
}

// NewObjectParser returns a parser reading from source. document may be nil.
func NewObjectParser(source pdfio.RandomAccessRead, document *cos.Document) *ObjectParser {
	return &ObjectParser{
		BaseParser: NewBaseParser(source),
		document:   document,
		keyCache:   make(map[int64]*cos.ObjectKey),
	}
}

// Document returns the document this parser fills, which may be nil.
func (p *ObjectParser) Document() *cos.Document { return p.document }

// peek returns the next byte without consuming it, or eof.
func (p *ObjectParser) peek() (int, error) {
	b, err := pdfio.Peek(p.source)
	if err != nil {
		if isEOFError(err) {
			return eof, nil
		}
		return eof, err
	}
	return int(b), nil
}

func isEOFError(err error) bool {
	return err != nil && err.Error() == "EOF"
}

// enter increments the recursion depth, failing past the limit.
func (p *ObjectParser) enter() error {
	p.recursionDepth++
	if p.recursionDepth > maxRecursionDepth {
		return fmt.Errorf("pdfparser: maximum allowed object recursion depth of %d exceeded",
			maxRecursionDepth)
	}
	return nil
}

func (p *ObjectParser) leave() { p.recursionDepth-- }

// ParseDirObject reads whatever object begins at the cursor.
//
// Port of parseDirObject.
func (p *ObjectParser) ParseDirObject() (cos.Base, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()

	if err := p.SkipSpaces(); err != nil {
		return nil, err
	}
	c, err := p.peek()
	if err != nil {
		return nil, err
	}

	switch c {
	case '<':
		// consume the first bracket and look for a second
		if _, err := p.readByte(); err != nil {
			return nil, err
		}
		next, err := p.peek()
		if err != nil {
			return nil, err
		}
		if next == '<' {
			if err := p.rewind(1); err != nil {
				return nil, err
			}
			return p.ParseCOSDictionary(true)
		}
		return p.ParseCOSHexString()

	case '[':
		return p.ParseCOSArray()

	case '(':
		return p.ParseCOSLiteralString()

	case '/':
		return p.ParseCOSName()

	case 'n':
		if err := p.ReadExpectedString("null", false); err != nil {
			return nil, err
		}
		return cos.NullObject, nil

	case 't':
		if err := p.ReadExpectedString("true", false); err != nil {
			return nil, err
		}
		return cos.True, nil

	case 'f':
		if err := p.ReadExpectedString("false", false); err != nil {
			return nil, err
		}
		return cos.False, nil

	case 'R':
		// A bare R, with the two numbers already consumed by the caller. Java
		// returns an empty COSObject as a marker, which the array and
		// dictionary parsers recognise and replace.
		if _, err := p.readByte(); err != nil {
			return nil, err
		}
		return cos.NewObject(nil), nil

	case eof:
		return nil, nil
	}

	if IsDigit(c) || c == '-' || c == '+' || c == '.' {
		return p.parseCOSNumber()
	}

	// Not supposed to happen, but allowed for, to be more compatible with
	// producers that do not follow the specification.
	startOffset, _ := p.source.Position()
	bad, err := p.ReadString()
	if err != nil {
		return nil, err
	}
	if bad == "" {
		peek, _ := p.peek()
		pos, _ := p.source.Position()
		return nil, fmt.Errorf(
			"pdfparser: unknown dir object c=%q cInt=%d peek=%q peekInt=%d at offset %d (start offset: %d)",
			rune(c), c, rune(peek), peek, pos, startOffset)
	}
	if bad == endobjString || bad == endstreamString {
		// put it back so the caller sees it
		if err := p.rewind(int64(len(bad))); err != nil {
			return nil, err
		}
		return nil, nil
	}
	pos, _ := p.source.Position()
	slog.Warn("pdfparser: skipped unexpected dir object",
		"value", bad, "offset", pos, "startOffset", startOffset)
	return cos.NullObject, nil
}

// ParseCOSName reads a name, resolving its #XX escapes.
//
// Port of parseCOSName. The '#' is treated as an escape only when two valid hex
// digits follow it: before PDF 1.2 it was an ordinary character, and tools that
// claim a later version still emit it that way.
func (p *ObjectParser) ParseCOSName() (*cos.Name, error) {
	if err := p.ReadExpectedChar('/'); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	c, err := p.readByte()
	if err != nil {
		return nil, err
	}

	for !IsEndOfName(c) {
		if c != '#' {
			buf.WriteByte(byte(c))
			c, err = p.readByte()
			if err != nil {
				return nil, err
			}
			continue
		}

		ch1, err := p.readByte()
		if err != nil {
			return nil, err
		}
		ch2, err := p.readByte()
		if err != nil {
			return nil, err
		}

		if isHexDigit(ch1) && isHexDigit(ch2) {
			value, convErr := strconv.ParseInt(string([]byte{byte(ch1), byte(ch2)}), 16, 32)
			if convErr != nil {
				return nil, fmt.Errorf("pdfparser: expected a hex digit, got %q: %w",
					string([]byte{byte(ch1), byte(ch2)}), convErr)
			}
			buf.WriteByte(byte(value))
			c, err = p.readByte()
			if err != nil {
				return nil, err
			}
			continue
		}

		if ch1 == eof || ch2 == eof {
			slog.Error("pdfparser: premature EOF in parseCOSName")
			c = eof
			break
		}
		if err := p.rewind(1); err != nil {
			return nil, err
		}
		c = ch1
		buf.WriteByte('#')
	}

	if c != eof {
		if err := p.rewind(1); err != nil {
			return nil, err
		}
	}
	return cos.GetPDFNameBytes(buf.Bytes()), nil
}

func isHexDigit(c int) bool {
	return IsDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// ParseCOSLiteralString reads a parenthesised string.
func (p *ObjectParser) ParseCOSLiteralString() (*cos.StringObj, error) {
	b, err := p.ReadLiteralString()
	if err != nil {
		return nil, err
	}
	return cos.NewStringObjBytes(b), nil
}

// ParseCOSHexString reads an angle-bracketed hex string.
//
// Port of parseCOSHexString. On an invalid character it discards a trailing
// unpaired digit and runs to the closing bracket, so that a malformed string
// does not derail the rest of the file.
func (p *ObjectParser) ParseCOSHexString() (*cos.StringObj, error) {
	var buf strings.Builder

	for {
		c, err := p.readByte()
		if err != nil {
			return nil, err
		}

		switch {
		case isHexDigit(c):
			buf.WriteByte(byte(c))
			continue

		case c == '>':
			return cos.ParseHexString(buf.String())

		case c == eof:
			return nil, fmt.Errorf("pdfparser: missing closing bracket for hex string, reached end of source")

		case c == ' ', c == '\n', c == '\t', c == '\r', c == '\b', c == '\f':
			continue
		}

		// An invalid character: discard the last digit if it has no pair, then
		// read to the closing bracket.
		s := buf.String()
		if len(s)%2 != 0 {
			s = s[:len(s)-1]
		}
		for {
			c, err = p.readByte()
			if err != nil {
				return nil, err
			}
			if c == '>' || c == eof {
				break
			}
		}
		if c == eof {
			return nil, fmt.Errorf("pdfparser: missing closing bracket for hex string, reached end of source")
		}
		return cos.ParseHexString(s)
	}
}

// parseCOSNumber reads a numeric literal.
//
// Port of the private parseCOSNumber. PDFBOX-5025: a number run together with a
// keyword, as in "74191endobj", takes the 'e' as the start of an exponent, so a
// trailing one is given back.
func (p *ObjectParser) parseCOSNumber() (cos.Number, error) {
	var buf bytes.Buffer
	c, err := p.readByte()
	if err != nil {
		return nil, err
	}
	for IsDigit(c) || c == '-' || c == '+' || c == '.' || c == 'E' || c == 'e' {
		buf.WriteByte(byte(c))
		c, err = p.readByte()
		if err != nil {
			return nil, err
		}
	}
	if c != eof {
		if err := p.rewind(1); err != nil {
			return nil, err
		}
	}

	s := buf.String()
	if s == "" {
		return nil, fmt.Errorf("pdfparser: expected a number but read nothing")
	}
	if last := s[len(s)-1]; last == 'e' || last == 'E' {
		s = s[:len(s)-1]
		if err := p.rewind(1); err != nil {
			return nil, err
		}
	}
	return cos.GetNumber(s)
}

// ParseCOSArray reads a bracketed array.
//
// Port of parseCOSArray, including the walk-back that turns a trailing
// "<num> <gen> R" into an indirect reference — PDFBOX-385 — and the recovery
// for a corrupt element.
func (p *ObjectParser) ParseCOSArray() (*cos.Array, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()

	startPosition, _ := p.source.Position()
	if err := p.ReadExpectedChar('['); err != nil {
		return nil, err
	}
	out := cos.NewArray()
	if err := p.SkipSpaces(); err != nil {
		return nil, err
	}

	for {
		c, err := p.peek()
		if err != nil {
			return nil, err
		}
		if c <= 0 || c == ']' {
			break
		}

		element, err := p.ParseDirObject()
		if err != nil {
			return nil, err
		}

		if _, isRef := element.(*cos.Object); isRef {
			// The two integers already added are the object and generation
			// number; take them back and build a reference.
			element = nil
			if out.Size() > 1 {
				if gen, ok := out.Get(out.Size() - 1).(*cos.Integer); ok {
					out.RemoveAt(out.Size() - 1)
					if num, ok := out.Get(out.Size() - 1).(*cos.Integer); ok && out.Size() > 0 {
						out.RemoveAt(out.Size() - 1)
						if num.LongValue() >= 0 && gen.IntValue() >= 0 {
							key, err := p.ObjectKey(num.LongValue(), gen.IntValue())
							if err != nil {
								return nil, err
							}
							element, err = p.objectFromPool(key)
							if err != nil {
								return nil, err
							}
						} else {
							slog.Warn("pdfparser: invalid values for an object key",
								"number", num.LongValue(), "generation", gen.IntValue())
						}
					}
				}
			}
		}

		if element == nil {
			pos, _ := p.source.Position()
			slog.Warn("pdfparser: corrupt array element",
				"offset", pos, "startOffset", startPosition)

			end, err := p.ReadString()
			if err != nil {
				return nil, err
			}
			// A corrupt element followed by another array most likely means the
			// whole array is corrupt; return now rather than recursing.
			if end == "" {
				if next, _ := p.peek(); next == '[' {
					return out, nil
				}
			}
			if err := p.rewind(int64(len(end))); err != nil {
				return nil, err
			}
			if end == endobjString || end == endstreamString {
				return out, nil
			}
		} else {
			out.Add(element)
		}

		if err := p.SkipSpaces(); err != nil {
			return nil, err
		}
	}

	// consume the ']'
	if _, err := p.readByte(); err != nil {
		return nil, err
	}
	if err := p.SkipSpaces(); err != nil {
		return nil, err
	}
	return out, nil
}

// ParseCOSDictionary reads a << >> dictionary.
//
// Port of parseCOSDictionary. A key that is not a name is a corrupt dictionary;
// Java reads on until it can recover and returns what it has rather than
// failing the whole object.
func (p *ObjectParser) ParseCOSDictionary(isDirect bool) (*cos.Dictionary, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()

	if err := p.ReadExpectedChar('<'); err != nil {
		return nil, err
	}
	if err := p.ReadExpectedChar('<'); err != nil {
		return nil, err
	}
	if err := p.SkipSpaces(); err != nil {
		return nil, err
	}

	out := cos.NewDictionary()
	out.SetDirect(isDirect)

	for {
		if err := p.SkipSpaces(); err != nil {
			return nil, err
		}
		c, err := p.peek()
		if err != nil {
			return nil, err
		}

		if c == '>' {
			break
		}
		if c == '/' {
			ok, err := p.parseNameValuePair(out)
			if err != nil {
				return nil, err
			}
			if !ok {
				return out, nil
			}
			continue
		}
		if c == eof {
			return out, nil
		}

		pos, _ := p.source.Position()
		slog.Warn("pdfparser: invalid dictionary, expected '/'",
			"found", string(rune(c)), "offset", pos)
		done, err := p.readUntilEndOfDictionary()
		if err != nil {
			return nil, err
		}
		if done {
			return out, nil
		}
	}

	// Java logs rather than failing when the closing brackets are missing.
	if err := p.ReadExpectedChar('>'); err != nil {
		pos, _ := p.source.Position()
		slog.Warn("pdfparser: invalid dictionary, cannot find its end", "offset", pos)
		return out, nil
	}
	if err := p.ReadExpectedChar('>'); err != nil {
		pos, _ := p.source.Position()
		slog.Warn("pdfparser: invalid dictionary, cannot find its end", "offset", pos)
	}
	return out, nil
}

// parseNameValuePair reads one entry, reporting whether parsing should go on.
func (p *ObjectParser) parseNameValuePair(out *cos.Dictionary) (bool, error) {
	key, err := p.ParseCOSName()
	if err != nil {
		return false, err
	}
	if key == nil || key.IsEmpty() {
		pos, _ := p.source.Position()
		slog.Warn("pdfparser: empty name used as a dictionary key", "offset", pos)
	}

	value, err := p.parseDictionaryValue()
	if err != nil {
		return false, err
	}
	if err := p.SkipSpaces(); err != nil {
		return false, err
	}

	switch {
	case value == nil:
		pos, _ := p.source.Position()
		slog.Warn("pdfparser: bad dictionary declaration", "offset", pos)
		return false, nil

	case isInvalidInteger(value):
		pos, _ := p.source.Position()
		slog.Warn("pdfparser: skipped an out of range number value", "offset", pos)

	default:
		// marked direct to avoid signature problems
		value.SetDirect(true)
		out.SetItem(key, value)
	}
	return true, nil
}

func isInvalidInteger(value cos.Base) bool {
	n, ok := value.(*cos.Integer)
	return ok && !n.IsValid()
}

// parseDictionaryValue reads a value, joining "<num> <gen> R" into a reference.
func (p *ObjectParser) parseDictionaryValue() (cos.Base, error) {
	numOffset, _ := p.source.Position()
	value, err := p.ParseDirObject()
	if err != nil {
		return nil, err
	}
	if err := p.SkipSpaces(); err != nil {
		return nil, err
	}

	if _, isNumber := value.(cos.Number); !isNumber {
		return value, nil
	}
	next, err := p.peek()
	if err != nil {
		return nil, err
	}
	if !IsDigit(next) {
		return value, nil
	}

	genOffset, _ := p.source.Position()
	generation, err := p.ParseDirObject()
	if err != nil {
		return nil, err
	}
	if err := p.SkipSpaces(); err != nil {
		return nil, err
	}
	if err := p.ReadExpectedChar('R'); err != nil {
		return nil, err
	}

	number, ok := value.(*cos.Integer)
	if !ok {
		slog.Error("pdfparser: expected a number", "actual", value, "offset", numOffset)
		return cos.NullObject, nil
	}
	genNumber, ok := generation.(*cos.Integer)
	if !ok {
		slog.Error("pdfparser: expected a number", "actual", generation, "offset", genOffset)
		return cos.NullObject, nil
	}
	if number.LongValue() <= 0 {
		slog.Warn("pdfparser: invalid object number", "value", number.LongValue(), "offset", numOffset)
		return cos.NullObject, nil
	}
	if genNumber.IntValue() < 0 {
		slog.Error("pdfparser: invalid generation number", "value", genNumber.IntValue(), "offset", numOffset)
		return cos.NullObject, nil
	}

	key, err := p.ObjectKey(number.LongValue(), genNumber.IntValue())
	if err != nil {
		return nil, err
	}
	return p.objectFromPool(key)
}

// readUntilEndOfDictionary skips damage, stopping at a name, a closing bracket,
// or an endstream/endobj keyword. It reports whether the object ended.
//
// Port of the private readUntilEndOfCOSDictionary.
func (p *ObjectParser) readUntilEndOfDictionary() (bool, error) {
	c, err := p.readByte()
	if err != nil {
		return false, err
	}
	for c != eof && c != '/' && c != '>' {
		if c == 'e' {
			matched, next, err := p.matchEndKeyword()
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
			c = next
			continue
		}
		c, err = p.readByte()
		if err != nil {
			return false, err
		}
	}
	if c == eof {
		return true, nil
	}
	return false, p.rewind(1)
}

// matchEndKeyword tries to read "ndstream" or "ndobj" after an 'e'.
func (p *ObjectParser) matchEndKeyword() (bool, int, error) {
	c, err := p.readByte()
	if err != nil {
		return false, eof, err
	}
	if c != 'n' {
		return false, c, nil
	}
	c, err = p.readByte()
	if err != nil {
		return false, eof, err
	}
	if c != 'd' {
		return false, c, nil
	}
	c, err = p.readByte()
	if err != nil {
		return false, eof, err
	}

	if c == 's' {
		ok, err := p.matchRest("tream")
		if err != nil {
			return false, eof, err
		}
		if ok {
			return true, eof, nil
		}
		return false, c, nil
	}
	if c == 'o' {
		ok, err := p.matchRest("bj")
		if err != nil {
			return false, eof, err
		}
		if ok {
			return true, eof, nil
		}
	}
	return false, c, nil
}

func (p *ObjectParser) matchRest(rest string) (bool, error) {
	for i := 0; i < len(rest); i++ {
		c, err := p.readByte()
		if err != nil {
			return false, err
		}
		if c != int(rest[i]) {
			return false, nil
		}
	}
	return true, nil
}

// objectFromPool returns the proxy for a key from the document object pool.
func (p *ObjectParser) objectFromPool(key *cos.ObjectKey) (cos.Base, error) {
	if p.document == nil {
		pos, _ := p.source.Position()
		return nil, fmt.Errorf("pdfparser: object reference %v at offset %d in content stream",
			key, pos)
	}
	return p.document.ObjectFromPool(key), nil
}

// ObjectKey returns the key for an object and generation number, reusing the
// instance the cross-reference table already holds where there is one.
//
// Port of getObjectKey. Java notes that iterating the xref table on every
// lookup gets slow on large files, which is what the cache is for.
func (p *ObjectParser) ObjectKey(num int64, gen int) (*cos.ObjectKey, error) {
	if p.document == nil {
		return cos.NewObjectKey(num, gen)
	}
	xrefTable := p.document.XRefTable()
	if len(xrefTable) == 0 {
		return cos.NewObjectKey(num, gen)
	}
	if len(xrefTable) > len(p.keyCache) {
		for key := range xrefTable {
			if key == nil {
				continue
			}
			if _, exists := p.keyCache[key.InternalHash()]; !exists {
				p.keyCache[key.InternalHash()] = key
			}
		}
	}
	if found, ok := p.keyCache[cos.ComputeInternalHash(num, gen)]; ok {
		return found, nil
	}
	return cos.NewObjectKey(num, gen)
}

// ReadObjectMarker consumes the "obj" keyword.
func (p *ObjectParser) ReadObjectMarker() error {
	return p.ReadExpectedString("obj", true)
}

// ReadObjectNumber reads an object number, rejecting one out of range.
func (p *ObjectParser) ReadObjectNumber() (int64, error) {
	value, err := p.ReadLong()
	if err != nil {
		return 0, err
	}
	if value < 0 || value >= objectNumberThreshold {
		return 0, fmt.Errorf("pdfparser: object number %d has more than 10 digits or is negative",
			value)
	}
	return value, nil
}

// ReadGenerationNumber reads a generation number, rejecting one out of range.
func (p *ObjectParser) ReadGenerationNumber() (int, error) {
	value, err := p.ReadInt()
	if err != nil {
		return 0, err
	}
	if value < 0 || value > generationNumberThreshold {
		return 0, fmt.Errorf("pdfparser: generation number %d has more than 5 digits or is negative",
			value)
	}
	return value, nil
}

// ReadLine reads to the next end of line, consuming a CRLF pair as one break.
func (p *ObjectParser) ReadLine() (string, error) {
	eofNow, err := p.source.IsEOF()
	if err != nil {
		return "", err
	}
	if eofNow {
		pos, _ := p.source.Position()
		return "", fmt.Errorf("pdfparser: end of file, expected a line at offset %d", pos)
	}

	var buf bytes.Buffer
	c := eof
	for {
		c, err = p.readByte()
		if err != nil {
			return "", err
		}
		if c == eof || IsEOL(c) {
			break
		}
		buf.WriteByte(byte(c))
	}

	if IsCR(c) {
		next, err := p.peek()
		if err != nil {
			return "", err
		}
		if IsLF(next) {
			if _, err := p.readByte(); err != nil {
				return "", err
			}
		}
	}
	return buf.String(), nil
}
