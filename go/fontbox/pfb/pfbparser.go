// Package pfb reads the PFB container an Adobe Type 1 font arrives in.
//
// Port of org.apache.fontbox.pfb.
package pfb

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	// HeaderLength is the pfb header length: start-marker (1 byte),
	// ascii-/binary-marker (1 byte), size (4 byte), 3*6 == 18.
	HeaderLength = 18

	// StartMarker is the start marker.
	StartMarker = 0x80

	// AsciiMarker is the ascii marker.
	AsciiMarker = 0x01

	// BinaryMarker is the binary marker.
	BinaryMarker = 0x02

	// EOFMarker is the EOF marker.
	EOFMarker = 0x03
)

// PfbParser is a parser for a pfb-file.
//
// Port of org.apache.fontbox.pfb.PfbParser.
type PfbParser struct {
	// the parsed pfb-data
	pfbdata []byte

	// the lengths of the records (ASCII, BINARY, ASCII)
	lengths [3]int
}

// sample (pfb-file)
// 00000000 80 01 8b 15  00 00 25 21  50 53 2d 41  64 6f 62 65
//          ......%!PS-Adobe

// NewPfbParserFile creates a new object from the named file.
func NewPfbParserFile(filename string) (*PfbParser, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return NewPfbParser(f)
}

// NewPfbParser creates a new object from the given input.
func NewPfbParser(in io.Reader) (*PfbParser, error) {
	p := &PfbParser{}
	if err := p.parsePfb(in); err != nil {
		return nil, err
	}
	return p, nil
}

// NewPfbParserBytes creates a new object from the given input.
func NewPfbParserBytes(data []byte) (*PfbParser, error) {
	return NewPfbParser(bytes.NewReader(data))
}

// readByte reads one byte, returning -1 at the end of the input, as Java's
// InputStream.read does.
func readByte(in io.Reader) (int, error) {
	var b [1]byte
	n, err := in.Read(b[:])
	if n == 1 {
		return int(b[0]), nil
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return -1, err
	}
	return -1, nil
}

// parsePfb parses the pfb-stream.
func (p *PfbParser) parsePfb(pfbStream io.Reader) error {
	// read into segments and keep them
	var typeList []int
	var barrList [][]byte
	var total int64
	for {
		r, err := readByte(pfbStream)
		if err != nil {
			return err
		}
		if r == -1 && total > 0 {
			break // EOF
		}
		if r != StartMarker {
			return errors.New("Start marker missing")
		}
		recordType, err := readByte(pfbStream)
		if err != nil {
			return err
		}
		if recordType == EOFMarker {
			break
		}
		if recordType != AsciiMarker && recordType != BinaryMarker {
			return fmt.Errorf("Incorrect record type: %d", recordType)
		}

		// Java accumulates into an int, so the four bytes wrap at 32 bits and a
		// top byte of 0xFF makes the size negative. The check below is what
		// catches that, so the arithmetic has to be done at the same width.
		var size int32
		for shift := 0; shift < 32; shift += 8 {
			b, err := readByte(pfbStream)
			if err != nil {
				return err
			}
			size += int32(b) << shift
		}
		slog.Debug("pfb record", "type", recordType, "segment size", size)
		if size < 0 {
			return fmt.Errorf("record size %d is negative", size)
		}
		// PDFBOX-6044: avoid potential OOM. readNBytes() grows its buffer
		// incrementally as bytes actually arrive, so a bogus/huge size can
		// never force an allocation larger than what the stream really holds.
		ar, err := io.ReadAll(io.LimitReader(pfbStream, int64(size)))
		if err != nil {
			return err
		}
		if int32(len(ar)) != size {
			return errors.New("EOF while reading PFB font")
		}
		total += int64(size)
		typeList = append(typeList, recordType)
		barrList = append(barrList, ar)
	}

	if total < HeaderLength {
		return errors.New("PFB header missing")
	}

	// We now have ASCII and binary segments. Lets arrange these so that the
	// ASCII segments come first, then the binary segments, then the last ASCII
	// segment if it is 0000... cleartomark

	p.pfbdata = make([]byte, int(total))
	var cleartomarkSegment []byte
	dstPos := 0

	// copy the ASCII segments
	for i := 0; i < len(typeList); i++ {
		if typeList[i] != AsciiMarker {
			continue
		}
		ar := barrList[i]
		if i == len(typeList)-1 && len(ar) < 600 && strings.Contains(string(ar), "cleartomark") {
			cleartomarkSegment = ar
			continue
		}
		copy(p.pfbdata[dstPos:], ar)
		dstPos += len(ar)
	}
	p.lengths[0] = dstPos

	// copy the binary segments
	for i := 0; i < len(typeList); i++ {
		if typeList[i] != BinaryMarker {
			continue
		}
		ar := barrList[i]
		copy(p.pfbdata[dstPos:], ar)
		dstPos += len(ar)
	}
	p.lengths[1] = dstPos - p.lengths[0]

	if cleartomarkSegment != nil {
		copy(p.pfbdata[dstPos:], cleartomarkSegment)
		p.lengths[2] = len(cleartomarkSegment)
	}
	return nil
}

// Lengths returns the lengths.
func (p *PfbParser) Lengths() [3]int { return p.lengths }

// Pfbdata returns the pfbdata.
func (p *PfbParser) Pfbdata() []byte { return p.pfbdata }

// InputStream returns the pfb data as stream.
func (p *PfbParser) InputStream() io.Reader { return bytes.NewReader(p.pfbdata) }

// Size returns the size of the pfb-data.
func (p *PfbParser) Size() int { return len(p.pfbdata) }

// Segment1 returns the first segment.
func (p *PfbParser) Segment1() []byte {
	out := make([]byte, p.lengths[0])
	copy(out, p.pfbdata[0:p.lengths[0]])
	return out
}

// Segment2 returns the second segment.
func (p *PfbParser) Segment2() []byte {
	out := make([]byte, p.lengths[1])
	copy(out, p.pfbdata[p.lengths[0]:p.lengths[0]+p.lengths[1]])
	return out
}
