package filter

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Predictor support for the Flate and LZW filters.
//
// Port of org.apache.pdfbox.filter.Predictor. A predictor transforms each row
// of data so that it compresses better; the decoder has to undo it. PDF uses
// the TIFF predictor (2) and the PNG predictors (10 to 14). Cross-reference
// streams almost always use 12, PNG Up, so nothing parses without this.
//
// # Java's int, all the way through
//
// Every calculation here is done the way Java does it: in 32-bit int, where a
// product wraps, a shift uses only the low five bits of its count, and a byte
// passed as an int is sign-extended. The decode parameters come straight out of
// the PDF, so a malformed file reaches all three, and the port gives PDFBox's
// answer for that file only if its arithmetic is Java's. Go's int is 64 bits and
// Go's shifts do not wrap their count, and each of those once gave a different
// answer here. conventions/java-to-go.md: "use int32 where the width is
// load-bearing". TestPredictorAnswersWhatPDFBoxAnswers holds PDFBox's own
// output for each case.

var (
	// errPredictorIndex stands for the ArrayIndexOutOfBoundsException that
	// Predictor.decodePredictorRow throws when the decode parameters place a
	// sample outside its row. /Colors -2 with /Columns -1 does it: the row is
	// two bytes long and a pixel is -1 of them.
	errPredictorIndex = errors.New("filter: predictor sample outside its row")

	// errNegativeRowLength is PredictorOutputStream's IOException, "Calculated
	// row length is negative".
	errNegativeRowLength = errors.New("filter: calculated row length is negative")

	// errZeroRowLength is where PDFBox gives no answer at all: a TIFF predictor
	// whose row length is zero, over a stream that is not empty. JAVA-BUGS 88.
	errZeroRowLength = errors.New("filter: predictor row length is zero")
)

// decodePredictorRow undoes the prediction on one row, in place.
//
// Port of Predictor.decodePredictorRow. actline is the row being decoded and
// lastline the row above it, already decoded. Where Java would index outside the
// row and throw, this returns errPredictorIndex; the row is then partly decoded,
// as Java's is, and the caller discards it, as Java's does.
func decodePredictorRow(predictor, colors, bitsPerComponent, columns int, actline, lastline []byte) error {
	if predictor == 1 {
		// no prediction
		return nil
	}

	bitsPerPixel := int32(colors) * int32(bitsPerComponent)
	bytesPerPixel := (bitsPerPixel + 7) / 8
	rowlength := int32(len(actline))

	switch predictor {
	case 2:
		return decodeTIFFSub(int32(colors), int32(bitsPerComponent), int32(columns), bytesPerPixel, actline)

	case 10:
		// PNG None

	case 11:
		// PNG Sub: add the pixel to the left. Java's first index is
		// bytesPerPixel itself, so a negative one throws straight away.
		if bytesPerPixel < 0 {
			return errPredictorIndex
		}
		for p := bytesPerPixel; p < rowlength; p++ {
			actline[p] += actline[p-bytesPerPixel]
		}

	case 12:
		// PNG Up: add the pixel above
		for p := range actline {
			actline[p] += lastline[p]
		}

	case 13:
		// PNG Average: add the mean of left and above. A negative
		// bytesPerPixel puts "left" ahead of p, and off the end of any row
		// with a byte in it.
		if bytesPerPixel < 0 && rowlength > 0 {
			return errPredictorIndex
		}
		for p := int32(0); p < rowlength; p++ {
			var left int
			if p-bytesPerPixel >= 0 {
				left = int(actline[p-bytesPerPixel])
			}
			up := int(lastline[p])
			actline[p] = byte(int(actline[p]) + (left+up)/2)
		}

	case 14:
		// PNG Paeth, which reads left the way Average does
		if bytesPerPixel < 0 && rowlength > 0 {
			return errPredictorIndex
		}
		for p := int32(0); p < rowlength; p++ {
			var a, c int // left, upper left
			if p-bytesPerPixel >= 0 {
				a = int(actline[p-bytesPerPixel])
				c = int(lastline[p-bytesPerPixel])
			}
			b := int(lastline[p]) // upper

			value := a + b - c
			absa, absb, absc := abs(value-a), abs(value-b), abs(value-c)

			switch {
			case absa <= absb && absa <= absc:
				actline[p] += byte(a)
			case absb <= absc:
				actline[p] += byte(b)
			default:
				actline[p] += byte(c)
			}
		}
	}
	return nil
}

// decodeTIFFSub undoes the TIFF horizontal-difference predictor.
func decodeTIFFSub(colors, bitsPerComponent, columns, bytesPerPixel int32, actline []byte) error {
	rowlength := int32(len(actline))

	if bitsPerComponent == 8 {
		// same algorithm as the PNG Sub predictor, and the same first index
		if bytesPerPixel < 0 {
			return errPredictorIndex
		}
		for p := bytesPerPixel; p < rowlength; p++ {
			actline[p] += actline[p-bytesPerPixel]
		}
		return nil
	}

	if bitsPerComponent == 16 {
		if bytesPerPixel < 0 && bytesPerPixel < rowlength-1 {
			return errPredictorIndex
		}
		for p := bytesPerPixel; p < rowlength-1; p += 2 {
			sub := int(actline[p])<<8 + int(actline[p+1])
			left := int(actline[p-bytesPerPixel])<<8 + int(actline[p-bytesPerPixel+1])
			actline[p] = byte((sub + left) >> 8)
			actline[p+1] = byte(sub + left)
		}
		return nil
	}

	if bitsPerComponent == 1 && colors == 1 {
		// bytesPerPixel cannot be used here: a row occupies a whole number of
		// bytes, and samples are packed high-order bit first.
		for p := int32(0); p < rowlength; p++ {
			for bit := 7; bit >= 0; bit-- {
				sub := int(actline[p]>>uint(bit)) & 1
				if p == 0 && bit == 7 {
					continue
				}
				var left int
				if bit == 7 {
					// bit 0 of the previous byte
					left = int(actline[p-1]) & 1
				} else {
					left = int(actline[p]>>uint(bit+1)) & 1
				}
				if (sub+left)&1 == 0 {
					actline[p] &^= 1 << uint(bit)
				} else {
					actline[p] |= 1 << uint(bit)
				}
			}
		}
		return nil
	}

	// everything else, i.e. 2 and 4 bits per component
	//
	// Java hands each byte to getBitSeq and calcSetBitSeq as an int, which
	// sign-extends it. At 1, 2 and 4 bits no component crosses a byte, only the
	// low eight bits are read, and the extension cannot be seen. At a depth that
	// does not divide 8 a component does cross one, its bit position goes
	// negative, and Java's shift reads the extension. The port passes the same
	// sign-extended value, so it reads the same bits.
	elements := columns * colors
	for p := colors; p < elements; p++ {
		bytePosSub := p * bitsPerComponent / 8
		bitPosSub := 8 - p*bitsPerComponent%8 - bitsPerComponent
		bytePosLeft := (p - colors) * bitsPerComponent / 8
		bitPosLeft := 8 - (p-colors)*bitsPerComponent%8 - bitsPerComponent

		if bytePosSub < 0 || bytePosSub >= rowlength || bytePosLeft < 0 || bytePosLeft >= rowlength {
			return errPredictorIndex
		}
		sub := getBitSeq(int(int8(actline[bytePosSub])), int(bitPosSub), int(bitsPerComponent))
		left := getBitSeq(int(int8(actline[bytePosLeft])), int(bitPosLeft), int(bitsPerComponent))
		actline[bytePosSub] = byte(calcSetBitSeq(int(int8(actline[bytePosSub])), int(bitPosSub),
			int(bitsPerComponent), sub+left))
	}
	return nil
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// calculateRowLength returns the number of bytes one row occupies, rounded up
// to a whole byte.
//
// Port of Predictor.calculateRowLength, in Java's int: the products wrap, and
// the division is of the wrapped value. /Colors is at most 32 by the time it
// gets here, but /BitsPerComponent and /Columns are whatever the PDF says, and a
// product that wraps to a small row in Java has to wrap to the same row here. In
// Go's int it would not wrap, and the port would allocate a row PDFBox never
// does: /Columns 1073741825 at 4 bits is a one-byte row in Java and 512 MB in
// 64 bits.
func calculateRowLength(colors, bitsPerComponent, columns int) int {
	bitsPerPixel := int32(colors) * int32(bitsPerComponent)
	return int((int32(columns)*bitsPerPixel + 7) / 8)
}

// getBitSeq reads a bit field out of a byte.
//
// In Java's int: each shift uses the low five bits of its count, and >>> fills
// from the top of 32 bits.
func getBitSeq(by, startBit, bitSize int) int {
	mask := int32(1)<<(uint(bitSize)&31) - 1
	return int(int32(uint32(int32(by))>>(uint(startBit)&31)) & mask)
}

// calcSetBitSeq writes a bit field into a byte and returns the result. The
// value is truncated to bitSize bits. In Java's int, as getBitSeq is.
func calcSetBitSeq(by, startBit, bitSize, val int) int {
	mask := int32(1)<<(uint(bitSize)&31) - 1
	truncated := int32(val) & mask
	mask = ^(mask << (uint(startBit) & 31))
	return int((int32(by) & mask) | (truncated << (uint(startBit) & 31)))
}

// predictorParams holds the decode parameters a predictor needs.
type predictorParams struct {
	predictor        int
	colors           int
	bitsPerComponent int
	columns          int
}

// readPredictorParams reads the predictor settings out of a decode parameter
// dictionary, applying the PDF defaults.
//
// Predictor.wrapPredictor reads /Colors through Math.min(..., 32), so a stream
// that declares more colours than that is decoded as if it had 32. The clamp is
// here because that is where Java has it: before the row length is worked out,
// and before any row is decoded. qpdf/issue-1688a.pdf declares 536870913.
func readPredictorParams(decodeParams cos.ReadOnlyDictionary) predictorParams {
	p := predictorParams{predictor: 1, colors: 1, bitsPerComponent: 8, columns: 1}
	if decodeParams == nil {
		return p
	}
	p.predictor = decodeParams.GetIntDefault(cos.Predictor, 1)
	p.colors = min(decodeParams.GetIntDefault(cos.Colors, 1), 32)
	p.bitsPerComponent = decodeParams.GetIntDefault(cos.BitsPerComponent, 8)
	p.columns = decodeParams.GetIntDefault(cos.Columns, 1)
	return p
}

// decodePredictor reads predicted rows from r and writes the decoded data to w.
//
// Port of Predictor.wrapPredictor and PredictorOutputStream. Java wraps the
// destination and pushes bytes through it; the port pulls instead, because a
// decoder here is a reader-to-writer copy. The rows are assembled the same way
// either way, and what comes out is Java's:
//
//   - Only a predictor above 1 wraps the stream. 0 and below pass the data
//     through untouched, as 1 does.
//   - A negative row length is refused before anything is read, as the
//     constructor refuses it.
//   - A PNG row starts with its algorithm: a signed Java byte, plus 10.
//   - The last row may be short. flush fills the rest of it with zeros and
//     decodes it whole, so the output is always a whole number of rows.
//   - A zero row length is allowed. A PNG predictor then takes every byte for a
//     row's algorithm and writes nothing. A TIFF predictor is JAVA-BUGS 88.
func decodePredictor(w io.Writer, r io.Reader, params predictorParams) error {
	if params.predictor <= 1 {
		_, err := io.Copy(w, r)
		return err
	}

	rowLength := calculateRowLength(params.colors, params.bitsPerComponent, params.columns)
	if rowLength < 0 {
		return fmt.Errorf("%w: %d", errNegativeRowLength, rowLength)
	}

	// A PNG predictor writes the algorithm as a leading byte on every row; a
	// TIFF predictor applies one algorithm to the whole stream.
	perRow := params.predictor >= 10

	br := bufio.NewReader(r)

	if rowLength == 0 && !perRow {
		// JAVA-BUGS 88. PredictorOutputStream.write copies nothing into a row
		// that holds nothing, counts it full, writes it and goes round again
		// without moving, so given a single byte PDFBox never returns. An empty
		// stream never enters that loop, and PDFBox answers it with nothing.
		if _, err := br.Peek(1); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		return errZeroRowLength
	}

	currentRow := make([]byte, rowLength)
	lastRow := make([]byte, rowLength)
	predictor := params.predictor

	for {
		if perRow {
			b, err := br.ReadByte()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			predictor = int(int8(b)) + 10
		}

		n, err := io.ReadFull(br, currentRow)
		if err == io.EOF {
			// no part of another row, so flush has nothing to complete
			return nil
		}
		short := err == io.ErrUnexpectedEOF
		if err != nil && !short {
			return err
		}
		if short {
			// flush: "The last row is allowed to be incomplete, and should be
			// completed with zeros."
			clear(currentRow[n:])
		}

		if err := decodePredictorRow(predictor, params.colors, params.bitsPerComponent, params.columns,
			currentRow, lastRow); err != nil {
			return err
		}
		if _, err := w.Write(currentRow); err != nil {
			return err
		}
		if short {
			return nil
		}

		// flipRows: the row just written is the row above the next one
		currentRow, lastRow = lastRow, currentRow
	}
}
