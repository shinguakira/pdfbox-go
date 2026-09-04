package filter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// SysPropCCITTFaxMaxBytes names the environment variable that raises the cap on
// how large a CCITT bitmap may be.
//
// Port of Filter.SYSPROP_CCITTFAX_MAXBYTES, which Java reads with
// System.getProperty; Go has no system properties, so the port reads the
// environment under the same name.
const SysPropCCITTFaxMaxBytes = "org.apache.pdfbox.filter.ccittmaxbytes"

// CCITTFax is the CCITTFaxDecode filter, Group 3 and Group 4 fax images.
//
// Port of org.apache.pdfbox.filter.CCITTFaxFilter.
type CCITTFax struct{}

var _ Filter = CCITTFax{}

// Decode expands the fax image into one bit per pixel.
func (CCITTFax) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int) (DecodeResult, error) {
	result := DecodeResult{Parameters: parameters}

	// get decode parameters
	decodeParms := decodeParamsOrEmpty(parameters, index)

	// parse dimensions
	cols := decodeParms.GetIntDefault(cos.Columns, 1728)
	rows := decodeParms.GetIntDefault(cos.Rows, 0)
	height := parameters.GetInt2(cos.Height, cos.H, 0)
	if rows > 0 && height > 0 {
		// PDFBOX-771, PDFBOX-3727: rows in DecodeParms sometimes contains an incorrect value
		rows = height
	} else {
		// at least one of the values has to have a valid value
		if height > rows {
			rows = height
		}
	}

	// decompress data
	k := decodeParms.GetIntDefault(cos.K, 0)
	encodedByteAlign := decodeParms.GetBoolean(cos.EncodedByteAlign, false)
	if cols <= 0 || rows <= 0 {
		return result, fmt.Errorf("Invalid CCITT image dimensions: cols=%d, rows=%d", cols, rows)
	}

	arraySizeLong := (int64(cols) + 7) / 8 * int64(rows)

	// PDFBOX-6243: CCITTFaxDecoderStream allocates two int arrays:
	// changesReferenceRow and changesCurrentRow
	changesSize := (int64(cols) + 2) * 4 * 2

	maxBytes := int64(256 * 1024 * 1024)
	if sysProp, ok := os.LookupEnv(SysPropCCITTFaxMaxBytes); ok {
		if parsed, err := strconv.ParseInt(sysProp, 10, 64); err == nil {
			if parsed > 0 {
				maxBytes = parsed
			}
			// else ignore zero/negative values
		}
		// ignore invalid value, keep default
	}

	if arraySizeLong+changesSize > maxBytes {
		return result, fmt.Errorf(
			"CCITT decode buffer too large (bitmapSize: %d, changesSize: %d) for cols=%d, "+
				"rows=%d; max allowed=%d; increase %s to override",
			arraySizeLong, changesSize, cols, rows, maxBytes, SysPropCCITTFaxMaxBytes)
	}

	arraySize := int(arraySizeLong)
	decompressed := make([]byte, arraySize)

	encoded := bufio.NewReader(r)
	var kind int
	var tiffOptions int64
	switch {
	case k == 0:
		if decodeParms.ContainsKey(cos.EndOfLine) {
			// PDFBOX-6080: respect the parameter if it exists
			if decodeParms.GetBoolean(cos.EndOfLine, false) {
				kind = compressionCCITTT4
			} else {
				kind = compressionCCITTModifiedHuffmanRLE
			}
			break
		}
		// In twelvemonkeys, this part is found in
		// CCITTFaxDecoderStream.findCompressionType()
		// needed for 015315-p8-ccitt.pdf, PDFBOX-2123-1bit.pdf, PDFBOX-2778.pdf
		kind = compressionCCITTT4 // Group 3 1D
		streamData := make([]byte, 20)
		bytesRead, err := readSome(encoded, streamData)
		if err != nil || bytesRead == 0 {
			return result, fmt.Errorf("EOF while reading CCITT header")
		}
		// Java pushes the bytes it sniffed back; the port peeks instead, so
		// nothing has to be put back.
		if streamData[0] != 0 || (int8(streamData[1])>>4 != 1 && streamData[1] != 1) {
			// leading EOL (0b000000000001) not found, search further and try RLE if not found
			kind = compressionCCITTModifiedHuffmanRLE
			b := int16((int32(int8(streamData[0]))<<8 + int32(streamData[1])) >> 4)
			for i := 12; i < bytesRead*8; i++ {
				b = int16(int32(b)<<1 + int32((int8(streamData[i/8])>>uint(7-i%8))&0x01))
				if b&0xFFF == 1 {
					kind = compressionCCITTT4
					break
				}
			}
		}

	case k > 0:
		// Group 3 2D
		kind = compressionCCITTT4
		tiffOptions = group3Opt2DEncoding

	default:
		// Group 4
		kind = compressionCCITTT6
	}

	s := newCCITTFaxDecoderStream(encoded, cols, kind, tiffOptions, encodedByteAlign)
	if err := readFromDecoderStream(s, decompressed); err != nil {
		return result, err
	}

	// invert bitmap
	blackIsOne := decodeParms.GetBoolean(cos.BlackIs1, false)
	if !blackIsOne {
		// Inverting the bitmap
		// Note the previous approach with starting from an IndexColorModel didn't work
		// reliably. In some cases the image wouldn't be painted for some reason.
		// So a safe but slower approach was taken.
		invertBitmap(decompressed)
	}

	if _, err := w.Write(decompressed); err != nil {
		return result, err
	}
	return result, nil
}

// readSome reads up to len(buf) bytes without consuming them, which is what
// Java's read-then-unread through a PushbackInputStream comes to.
func readSome(encoded *bufio.Reader, buf []byte) (int, error) {
	peeked, err := encoded.Peek(len(buf))
	n := copy(buf, peeked)
	if n > 0 {
		return n, nil
	}
	return n, err
}

func readFromDecoderStream(decoderStream *ccittFaxDecoderStream, result []byte) error {
	pos := 0
	for pos < len(result) {
		read, err := decoderStream.Read(result[pos:])
		if err != nil {
			return err
		}
		if read <= 0 {
			break
		}
		pos += read
	}
	return nil
}

func invertBitmap(bufferData []byte) {
	for i := range bufferData {
		bufferData[i] = ^bufferData[i] & 0xFF
	}
}

// Encode compresses the bitmap into a Group 3 one dimensional fax stream.
func (CCITTFax) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	cols := parameters.GetInt(cos.Columns)
	rows := parameters.GetInt(cos.Rows)
	out := newCCITTFaxEncoderStream(w, cols, rows, fillLeftToRight)
	if _, err := io.Copy(out, r); err != nil {
		return err
	}
	return out.Close()
}

// decodeParamsOrEmpty is Filter.getDecodeParams, which returns an empty
// dictionary rather than null where a stream has no decode parameters; the
// port's decodeParamsFor returns nil, and every caller that reads defaults out
// of the result needs the empty dictionary instead.
func decodeParamsOrEmpty(parameters *cos.Dictionary, index int) *cos.Dictionary {
	if params := decodeParamsFor(parameters, index); params != nil {
		return params
	}
	return cos.NewDictionary()
}
