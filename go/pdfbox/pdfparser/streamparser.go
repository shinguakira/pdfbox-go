package pdfparser

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// strmBufLen is the scan buffer size used when looking for the endstream
// keyword.
const strmBufLen = 2048

var (
	endstreamKeyword = []byte("endstream")
	endobjKeyword    = []byte("endobj")
)

// StreamParser reads stream objects.
//
// Port of the stream-reading half of org.apache.pdfbox.pdfparser.COSParser,
// together with EndstreamFilterStream.
type StreamParser struct {
	*ObjectParser

	fileLen   int64
	isLenient bool
	strmBuf   []byte
}

// NewStreamParser returns a parser reading from source, together with the
// document it fills.
//
// The parser creates the document rather than being handed one, because the
// document needs the parser as its cos.Parser and the parser needs the document
// as its object pool. Java resolves the same circularity the same way: the
// COSParser constructor runs new COSDocument(streamCache, this).
func NewStreamParser(source pdfio.RandomAccessRead, cache pdfio.StreamCache, codecs cos.CodecProvider) (*StreamParser, error) {
	length, err := source.Length()
	if err != nil {
		return nil, err
	}
	p := &StreamParser{
		fileLen:   length,
		isLenient: true,
		strmBuf:   make([]byte, strmBufLen),
	}
	document := cos.NewDocumentWithCache(cache, codecs, p)
	p.ObjectParser = NewObjectParser(source, document)
	return p, nil
}

// CreateRandomAccessReadView returns a view onto the file being parsed.
//
// Part of the cos.Parser interface, the port of ICOSParser.
func (p *StreamParser) CreateRandomAccessReadView(start, length int64) (pdfio.RandomAccessRead, error) {
	return p.source.CreateView(start, length)
}

// DereferenceObject resolves an indirect reference.
//
// Part of the cos.Parser interface. Resolving a reference needs the
// cross-reference table, which the file-level parser owns; a StreamParser used
// on its own has none, so it reports that rather than returning a wrong answer.
func (p *StreamParser) DereferenceObject(obj *cos.Object) (cos.Base, error) {
	return nil, fmt.Errorf("pdfparser: cannot dereference %v without a cross-reference table", obj.Key())
}

// IsLenient reports whether malformed input is worked around rather than
// rejected.
func (p *StreamParser) IsLenient() bool { return p.isLenient }

// SetLenient sets that flag. Java defaults it to true.
func (p *StreamParser) SetLenient(lenient bool) { p.isLenient = lenient }

// ParseCOSStream reads the stream that follows a stream dictionary.
//
// Port of parseCOSStream. The "stream" keyword has already been seen by the
// caller, so it is consumed here without checking.
func (p *StreamParser) ParseCOSStream(dic *cos.Dictionary) (*cos.Stream, error) {
	// consume the 'stream' keyword
	if _, err := p.ReadString(); err != nil {
		return nil, err
	}
	if err := p.SkipWhiteSpaces(); err != nil {
		return nil, err
	}

	lengthObj, err := p.streamLength(dic.GetItem(cos.Length))
	if err != nil {
		return nil, err
	}
	if lengthObj == nil {
		pos, _ := p.source.Position()
		if !p.isLenient {
			return nil, fmt.Errorf("pdfparser: missing length for stream at offset %d", pos)
		}
		slog.Warn("pdfparser: the stream does not give a length, scanning for endstream instead",
			"offset", pos)
	}

	streamStart, err := p.source.Position()
	if err != nil {
		return nil, err
	}

	var streamLength int64
	usable := false
	if lengthObj != nil {
		usable, err = p.validateStreamLength(lengthObj.LongValue())
		if err != nil {
			return nil, err
		}
	}

	if usable {
		streamLength = lengthObj.LongValue()
		if err := pdfio.SeekTo(p.source, streamStart+streamLength); err != nil {
			return nil, err
		}
	} else {
		streamLength, err = p.readUntilEndStream()
		if err != nil {
			return nil, err
		}
		if lengthObj == nil || lengthObj.LongValue() != streamLength {
			dic.SetLong(cos.Length, streamLength)
		}
	}

	endStream, err := p.ReadString()
	if err != nil {
		return nil, err
	}
	pos, _ := p.source.Position()

	switch {
	case endStream == string(endobjKeyword) && p.isLenient:
		slog.Warn("pdfparser: stream ends with endobj instead of endstream", "offset", pos)
		// give the keyword back, so the caller does not also warn about a
		// missing endobj
		if err := p.rewind(int64(len(endobjKeyword))); err != nil {
			return nil, err
		}

	case len(endStream) > 9 && p.isLenient && strings.HasPrefix(endStream, string(endstreamKeyword)):
		slog.Warn("pdfparser: stream ends with unexpected trailing bytes",
			"value", endStream, "offset", pos)
		// unread the extra bytes
		if err := p.rewind(int64(len(endStream[9:]))); err != nil {
			return nil, err
		}

	case endStream != string(endstreamKeyword):
		return nil, fmt.Errorf(
			"pdfparser: error reading stream, expected 'endstream' actual %q at offset %d",
			endStream, pos)
	}

	if p.document == nil {
		return nil, fmt.Errorf("pdfparser: cannot create a stream without a document")
	}
	return p.document.CreateStreamFromFile(dic, streamStart, streamLength)
}

// streamLength resolves the /Length entry, which may be an indirect reference.
//
// Port of the private getLength.
func (p *StreamParser) streamLength(lengthBase cos.Base) (cos.Number, error) {
	if lengthBase == nil {
		return nil, nil
	}
	if n, ok := lengthBase.(cos.Number); ok {
		return n, nil
	}
	ref, ok := lengthBase.(*cos.Object)
	if !ok {
		return nil, fmt.Errorf("pdfparser: wrong type of length object: %T", lengthBase)
	}

	length := ref.Object()
	if length == nil {
		return nil, fmt.Errorf("pdfparser: length object content was not read")
	}
	if length == cos.Base(cos.NullObject) {
		slog.Warn("pdfparser: length object not found", "key", ref.Key())
		return nil, nil
	}
	if n, ok := length.(cos.Number); ok {
		return n, nil
	}
	return nil, fmt.Errorf("pdfparser: wrong type of referenced length object %v: %T", ref, length)
}

// validateStreamLength reports whether a declared length can be trusted, by
// checking that the endstream keyword really is where it says.
//
// Port of the private validateStreamLength.
func (p *StreamParser) validateStreamLength(streamLength int64) (bool, error) {
	originOffset, err := p.source.Position()
	if err != nil {
		return false, err
	}

	if streamLength == 0 {
		// may be valid (PDFBOX-5954) or not (PDFBOX-5880)
		slog.Debug("pdfparser: suspicious stream length 0", "startPosition", originOffset)
		return false, nil
	}
	if streamLength < 0 {
		slog.Warn("pdfparser: invalid stream length",
			"length", streamLength, "startPosition", originOffset)
		return false, nil
	}

	expectedEnd := originOffset + streamLength
	if expectedEnd > p.fileLen {
		slog.Warn("pdfparser: the end of the stream is out of range, using the fallback scan",
			"startPosition", originOffset, "length", streamLength, "expectedEnd", expectedEnd)
		return false, nil
	}

	if err := pdfio.SeekTo(p.source, expectedEnd); err != nil {
		return false, err
	}
	if err := p.SkipSpaces(); err != nil {
		return false, err
	}
	found, err := p.isString(endstreamKeyword)
	if err != nil {
		return false, err
	}
	if err := pdfio.SeekTo(p.source, originOffset); err != nil {
		return false, err
	}

	if !found {
		slog.Warn("pdfparser: the stream length does not point at endstream, using the fallback scan",
			"startPosition", originOffset, "length", streamLength, "expectedEnd", expectedEnd)
		return false, nil
	}
	return true, nil
}

// isString reports whether the given bytes are at the cursor, leaving the
// cursor where it was.
//
// Port of the protected isString(byte[]).
func (p *StreamParser) isString(want []byte) (bool, error) {
	origin, err := p.source.Position()
	if err != nil {
		return false, err
	}
	match := true
	for _, b := range want {
		c, err := p.readByte()
		if err != nil {
			return false, err
		}
		if c != int(b) {
			match = false
			break
		}
	}
	if err := pdfio.SeekTo(p.source, origin); err != nil {
		return false, err
	}
	return match, nil
}

// readUntilEndStream scans forward for the endstream keyword and returns the
// length of the data before it.
//
// Port of the private readUntilEndStream. The scan carries a partial keyword
// match across buffer boundaries, and uses the look-ahead shortcut the Java
// comment credits to Boyer-Moore, which it says saves roughly 20% of parsing
// time. If endstream is missing it accepts endobj instead, which is what makes
// a truncated file readable.
func (p *StreamParser) readUntilEndStream() (int64, error) {
	out := &endstreamFilter{}
	charMatchCount := 0
	keyw := endstreamKeyword
	// last character position of the shortest keyword, endobj
	const quickTestOffset = 5

	for {
		n, err := p.source.Read(p.strmBuf[charMatchCount : strmBufLen-charMatchCount+charMatchCount])
		if err != nil && !isEOFError(err) {
			return 0, err
		}
		if n <= 0 {
			break
		}

		bufSize := n + charMatchCount
		bIdx := charMatchCount
		maxQuickTestIdx := bufSize - quickTestOffset

		for ; bIdx < bufSize; bIdx++ {
			// Reduce comparisons by first testing the last character that would
			// have to match; if it is not a character from either keyword, jump
			// past it.
			quickTestIdx := bIdx + quickTestOffset
			if charMatchCount == 0 && quickTestIdx < maxQuickTestIdx {
				ch := p.strmBuf[quickTestIdx]
				if ch > 't' || ch < 'a' {
					bIdx = quickTestIdx
					continue
				}
			}

			ch := p.strmBuf[bIdx]
			if ch == keyw[charMatchCount] {
				charMatchCount++
				if charMatchCount == len(keyw) {
					bIdx++
					break
				}
				continue
			}

			if charMatchCount == 3 && ch == endobjKeyword[charMatchCount] {
				// endstream may be missing where endobj is present
				keyw = endobjKeyword
				charMatchCount++
				continue
			}

			// No match. Incrementing the start by one would discard what is
			// already known: an 'e' begins a new match, and an 'n' at match
			// position 7 means two characters are already matched.
			switch {
			case ch == 'e':
				charMatchCount = 1
			case ch == 'n' && charMatchCount == 7:
				charMatchCount = 2
			default:
				charMatchCount = 0
			}
			keyw = endstreamKeyword
		}

		contentBytes := bIdx - charMatchCount
		if contentBytes < 0 {
			contentBytes = 0
		}
		if contentBytes > 0 {
			out.filter(p.strmBuf, 0, contentBytes)
		}

		if charMatchCount == len(keyw) {
			// Give back the matched keyword and everything buffered after it.
			if err := p.rewind(int64(bufSize - contentBytes)); err != nil {
				return 0, err
			}
			break
		}
		// carry the partial match to the start of the next buffer
		copy(p.strmBuf, keyw[:charMatchCount])
	}

	return out.calculateLength(), nil
}

// endstreamFilter measures stream data while dropping the end-of-line that
// precedes the endstream keyword, which is not part of the stream.
//
// Port of org.apache.pdfbox.pdfparser.EndstreamFilterStream. It only measures;
// the data itself is read again later through a view onto the file.
type endstreamFilter struct {
	hasCR      bool
	hasLF      bool
	pos        int
	mustFilter bool
	length     int64
}

// filter accounts for one buffer of stream data.
func (f *endstreamFilter) filter(b []byte, off, length int) {
	if f.pos == 0 {
		// The first buffer decides whether to trim at all. PDFBOX-2120: do not
		// trim ASCII data, so that a final CR LF or LF is kept. The heuristic
		// looks at ten bytes and is taken from PDFStreamParser, PDFBOX-1164.
		f.mustFilter = true
		if length > 10 {
			f.mustFilter = false
			for i := 0; i < 10; i++ {
				if b[i] < 0x09 || (b[i] > 0x0a && b[i] < 0x20 && b[i] != 0x0d) {
					// a control character, or above 0x7f, means binary data
					f.mustFilter = true
					break
				}
			}
		}
	}

	if f.mustFilter {
		// first account for what was held back last time
		if f.hasCR {
			f.hasCR = false
			if !f.hasLF && length == 1 && b[off] == '\n' {
				// this buffer holds only the LF of a split CR LF, so it is the
				// last one and neither byte counts
				return
			}
			f.length++
		}
		if f.hasLF {
			f.length++
			f.hasLF = false
		}

		// hold back a CR, LF or CR LF at the end of the buffer
		if length > 0 {
			switch b[off+length-1] {
			case '\r':
				f.hasCR = true
				length--
			case '\n':
				f.hasLF = true
				length--
				if length > 0 && b[off+length-1] == '\r' {
					f.hasCR = true
					length--
				}
			}
		}
	}

	f.length += int64(length)
	f.pos += length
}

// calculateLength returns the stream length, keeping a lone CR.
func (f *endstreamFilter) calculateLength() int64 {
	if f.hasCR && !f.hasLF {
		f.length++
		f.pos++
	}
	f.hasCR = false
	f.hasLF = false
	return f.length
}
