package xml

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

// The encoding a document is read in.
//
// Java's DocumentBuilder is the JDK's Xerces, and Xerces settles the encoding
// before it reads any markup. XMLEntityManager.setupCurrentEntity reads the
// first four bytes and asks getEncodingInfo what they are -- a byte order mark
// for UTF-8 or UTF-16, or the pattern "<?" makes in UTF-16 or UCS-4 without one
// (XML 1.0, appendix F) -- skips a mark it found, and starts the reader
// createReader picks. When the XML declaration names an encoding,
// XMLEntityScanner.setEncoding puts a reader for that encoding on the rest of
// the bytes, unless it is the one already in use, or the one in use is UTF-16
// and the declaration says only "UTF-16".
//
// encoding/xml reads UTF-8 and nothing else, so an entity hands it the
// document as UTF-8, one byte at a time. One byte at a time is what lets the
// declaration change the encoding after its closing "?>" and not a byte later:
// Xerces reads the declaration a character at a time for the same reason.

// The names Xerces's EncodingInfo gives the encodings it tells apart.
const (
	encodingUTF8    = "UTF-8"
	encodingUTF16   = "UTF-16"
	encodingUTF16BE = "UTF-16BE"
	encodingUTF16LE = "UTF-16LE"
	encodingUCS4    = "ISO-10646-UCS-4"
	encodingUCS2    = "ISO-10646-UCS-2"
	encodingCP037   = "CP037"
)

// encodingInfo is Xerces's EncodingInfo: what the first four bytes say.
type encodingInfo struct {
	autoDetected string
	reader       string
	bigEndian    *bool
	hasBOM       bool
}

var (
	bigEndian    = true
	littleEndian = false
)

// encodingInfoOf is XMLEntityManager.getEncodingInfo, for the four bytes
// setupCurrentEntity always reads.
func encodingInfoOf(b4 [4]byte) encodingInfo {
	b0, b1, b2, b3 := b4[0], b4[1], b4[2], b4[3]
	switch {
	// UTF-16, with BOM
	case b0 == 0xFE && b1 == 0xFF:
		return encodingInfo{encodingUTF16BE, encodingUTF16, &bigEndian, true}
	case b0 == 0xFF && b1 == 0xFE:
		return encodingInfo{encodingUTF16LE, encodingUTF16, &littleEndian, true}
	// UTF-8 with a BOM
	case b0 == 0xEF && b1 == 0xBB && b2 == 0xBF:
		return encodingInfo{encodingUTF8, encodingUTF8, nil, true}
	// UCS-4, big endian (1234)
	case b0 == 0x00 && b1 == 0x00 && b2 == 0x00 && b3 == 0x3C:
		return encodingInfo{encodingUCS4, encodingUCS4, &bigEndian, false}
	// UCS-4, little endian (4321)
	case b0 == 0x3C && b1 == 0x00 && b2 == 0x00 && b3 == 0x00:
		return encodingInfo{encodingUCS4, encodingUCS4, &littleEndian, false}
	// UCS-4, unusual octet orders (2143) and (3412)
	case b0 == 0x00 && b1 == 0x00 && b2 == 0x3C && b3 == 0x00,
		b0 == 0x00 && b1 == 0x3C && b2 == 0x00 && b3 == 0x00:
		return encodingInfo{encodingUCS4, encodingUCS4, nil, false}
	// UTF-16, big-endian, no BOM
	case b0 == 0x00 && b1 == 0x3C && b2 == 0x00 && b3 == 0x3F:
		return encodingInfo{encodingUTF16BE, encodingUTF16, &bigEndian, false}
	// UTF-16, little-endian, no BOM
	case b0 == 0x3C && b1 == 0x00 && b2 == 0x3F && b3 == 0x00:
		return encodingInfo{encodingUTF16LE, encodingUTF16, &littleEndian, false}
	// EBCDIC
	case b0 == 0x4C && b1 == 0x6F && b2 == 0xA7 && b3 == 0x94:
		return encodingInfo{encodingCP037, encodingCP037, nil, false}
	}
	// default encoding
	return encodingInfo{encodingUTF8, encodingUTF8, nil, false}
}

// entity is the document being read, handed out as UTF-8.
type entity struct {
	raw *bufio.Reader
	// encoding is what Xerces keeps as fCurrentEntity.encoding.
	encoding string
	// fill reads the next character in the encoding in force into pending.
	fill    func() error
	pending []byte
}

// newEntity is the encoding half of XMLEntityManager.setupCurrentEntity.
func newEntity(input io.Reader) (*entity, error) {
	e := &entity{raw: bufio.NewReader(input)}
	// Xerces casts each read to a byte, so the end of the input reads as 0xFF.
	b4 := [4]byte{0xFF, 0xFF, 0xFF, 0xFF}
	peeked, _ := e.raw.Peek(4)
	copy(b4[:], peeked)

	info := encodingInfoOf(b4)
	e.encoding = info.autoDetected
	if info.hasBOM {
		switch info.reader {
		case encodingUTF8:
			e.raw.Discard(3)
		case encodingUTF16:
			e.raw.Discard(2)
		}
	}
	fill, err := e.readerFor(info.reader, info.bigEndian)
	if err != nil {
		return nil, err
	}
	e.fill = fill
	return e, nil
}

// ReadByte hands out the next byte of the document as UTF-8.
func (e *entity) ReadByte() (byte, error) {
	for len(e.pending) == 0 {
		if err := e.fill(); err != nil {
			return 0, err
		}
	}
	b := e.pending[0]
	e.pending = e.pending[1:]
	return b, nil
}

// Read hands out one byte, so that nothing is read ahead of the declaration.
func (e *entity) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	b, err := e.ReadByte()
	if err != nil {
		return 0, err
	}
	p[0] = b
	return 1, nil
}

// setEncoding is XMLEntityScanner.setEncoding: the encoding an XML declaration
// names, applied to the rest of the document.
func (e *entity) setEncoding(encoding string) error {
	if e.encoding == encoding {
		return nil
	}
	// UTF-16 is a bit of a special case. If the encoding is UTF-16, and we
	// know the endian-ness, we shouldn't change readers. If it's
	// ISO-10646-UCS-(2|4), then we'll have to deduce the endian-ness from the
	// encoding we presently have.
	if strings.HasPrefix(e.encoding, encodingUTF16) {
		switch strings.ToUpper(encoding) {
		case encodingUTF16:
			return nil
		case encodingUCS4:
			e.fill = e.ucs4(e.encoding == encodingUTF16BE)
			return nil
		case encodingUCS2:
			e.fill = e.utf16(e.encoding == encodingUTF16BE)
			return nil
		}
	}
	fill, err := e.readerFor(encoding, nil)
	if err != nil {
		return err
	}
	e.fill = fill
	e.encoding = encoding
	return nil
}

// readerFor is XMLEntityManager.createReader.
//
// Where Xerces falls back on a Java reader, through its own table of IANA
// names, the port looks the name up in the IANA index of golang.org/x/text.
// Xerces's UCS-2 reader reads two bytes a character, as UTF-16 does.
func (e *entity) readerFor(encoding string, isBigEndian *bool) (func() error, error) {
	switch strings.ToUpper(encoding) {
	case encodingUTF8:
		return e.utf8, nil
	case encodingUTF16:
		if isBigEndian != nil {
			return e.utf16(*isBigEndian), nil
		}
	case encodingUTF16BE:
		return e.utf16(true), nil
	case encodingUTF16LE:
		return e.utf16(false), nil
	case encodingUCS4:
		if isBigEndian == nil {
			return nil, fmt.Errorf("Given byte order for encoding %q is not supported.", encoding)
		}
		return e.ucs4(*isBigEndian), nil
	case encodingUCS2:
		if isBigEndian == nil {
			return nil, fmt.Errorf("Given byte order for encoding %q is not supported.", encoding)
		}
		return e.utf16(*isBigEndian), nil
	}
	named, err := ianaindex.IANA.Encoding(encoding)
	if err != nil || named == nil {
		return nil, fmt.Errorf("Invalid encoding name %q.", encoding)
	}
	decoded := bufio.NewReader(transform.NewReader(e.raw, named.NewDecoder()))
	return func() error {
		b, err := decoded.ReadByte()
		if err != nil {
			return err
		}
		e.pending = append(e.pending[:0], b)
		return nil
	}, nil
}

// utf8 passes the bytes through; encoding/xml checks them.
func (e *entity) utf8() error {
	b, err := e.raw.ReadByte()
	if err != nil {
		return err
	}
	e.pending = append(e.pending[:0], b)
	return nil
}

func (e *entity) utf16(isBigEndian bool) func() error {
	unit := func() (rune, error) {
		b0, err := e.raw.ReadByte()
		if err != nil {
			return 0, err
		}
		b1, err := e.raw.ReadByte()
		if err == io.EOF {
			return 0, errors.New("Premature end of UTF-16 input: an odd number of bytes")
		}
		if err != nil {
			return 0, err
		}
		if isBigEndian {
			return rune(b0)<<8 | rune(b1), nil
		}
		return rune(b1)<<8 | rune(b0), nil
	}
	return func() error {
		r, err := unit()
		if err != nil {
			return err
		}
		if utf16.IsSurrogate(r) {
			surrogate := r
			low, err := unit()
			if err == io.EOF {
				err = nil
			}
			if err != nil {
				return err
			}
			if r = utf16.DecodeRune(surrogate, low); r == utf8.RuneError {
				return fmt.Errorf("An invalid XML character (Unicode: 0x%x) was found", surrogate)
			}
		}
		e.pending = utf8.AppendRune(e.pending[:0], r)
		return nil
	}
}

func (e *entity) ucs4(isBigEndian bool) func() error {
	return func() error {
		var b [4]byte
		if _, err := io.ReadFull(e.raw, b[:]); err != nil {
			if err == io.ErrUnexpectedEOF {
				return errors.New("Premature end of UCS-4 input")
			}
			return err
		}
		var value uint32
		if isBigEndian {
			value = uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
		} else {
			value = uint32(b[3])<<24 | uint32(b[2])<<16 | uint32(b[1])<<8 | uint32(b[0])
		}
		r := rune(value)
		if value > utf8.MaxRune || !utf8.ValidRune(r) {
			return fmt.Errorf("An invalid XML character (Unicode: 0x%x) was found", value)
		}
		e.pending = utf8.AppendRune(e.pending[:0], r)
		return nil
	}
}

// declaredEncoding reads the encoding pseudo-attribute out of an XML
// declaration, as encoding/xml does before it asks its CharsetReader.
func declaredEncoding(declaration string) string {
	at := strings.Index(declaration, "encoding")
	if at < 0 {
		return ""
	}
	rest := strings.TrimLeft(declaration[at+len("encoding"):], " \t\r\n")
	if !strings.HasPrefix(rest, "=") {
		return ""
	}
	rest = strings.TrimLeft(rest[1:], " \t\r\n")
	if rest == "" || (rest[0] != '"' && rest[0] != '\'') {
		return ""
	}
	end := strings.IndexByte(rest[1:], rest[0])
	if end < 0 {
		return ""
	}
	return rest[1 : 1+end]
}
