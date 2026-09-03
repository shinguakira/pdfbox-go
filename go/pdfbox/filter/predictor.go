package filter

import (
	"bufio"
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

// decodePredictorRow undoes the prediction on one row, in place.
//
// Port of Predictor.decodePredictorRow. actline is the row being decoded and
// lastline the row above it, already decoded.
func decodePredictorRow(predictor, colors, bitsPerComponent, columns int, actline, lastline []byte) {
	if predictor == 1 {
		// no prediction
		return
	}

	bitsPerPixel := colors * bitsPerComponent
	bytesPerPixel := (bitsPerPixel + 7) / 8
	rowlength := len(actline)

	switch predictor {
	case 2:
		decodeTIFFSub(colors, bitsPerComponent, columns, bytesPerPixel, actline)

	case 10:
		// PNG None

	case 11:
		// PNG Sub: add the pixel to the left
		for p := bytesPerPixel; p < rowlength; p++ {
			actline[p] += actline[p-bytesPerPixel]
		}

	case 12:
		// PNG Up: add the pixel above
		for p := 0; p < rowlength; p++ {
			actline[p] += lastline[p]
		}

	case 13:
		// PNG Average: add the mean of left and above
		for p := 0; p < rowlength; p++ {
			var left int
			if p-bytesPerPixel >= 0 {
				left = int(actline[p-bytesPerPixel])
			}
			up := int(lastline[p])
			actline[p] = byte(int(actline[p]) + (left+up)/2)
		}

	case 14:
		// PNG Paeth
		for p := 0; p < rowlength; p++ {
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
}

// decodeTIFFSub undoes the TIFF horizontal-difference predictor.
func decodeTIFFSub(colors, bitsPerComponent, columns, bytesPerPixel int, actline []byte) {
	rowlength := len(actline)

	if bitsPerComponent == 8 {
		// same algorithm as the PNG Sub predictor
		for p := bytesPerPixel; p < rowlength; p++ {
			actline[p] += actline[p-bytesPerPixel]
		}
		return
	}

	if bitsPerComponent == 16 {
		for p := bytesPerPixel; p < rowlength-1; p += 2 {
			sub := int(actline[p])<<8 + int(actline[p+1])
			left := int(actline[p-bytesPerPixel])<<8 + int(actline[p-bytesPerPixel+1])
			actline[p] = byte((sub + left) >> 8)
			actline[p+1] = byte(sub + left)
		}
		return
	}

	if bitsPerComponent == 1 && colors == 1 {
		// bytesPerPixel cannot be used here: a row occupies a whole number of
		// bytes, and samples are packed high-order bit first.
		for p := 0; p < rowlength; p++ {
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
		return
	}

	// everything else, i.e. 2 and 4 bits per component
	elements := columns * colors
	for p := colors; p < elements; p++ {
		bytePosSub := p * bitsPerComponent / 8
		bitPosSub := 8 - p*bitsPerComponent%8 - bitsPerComponent
		bytePosLeft := (p - colors) * bitsPerComponent / 8
		bitPosLeft := 8 - (p-colors)*bitsPerComponent%8 - bitsPerComponent

		sub := getBitSeq(int(actline[bytePosSub]), bitPosSub, bitsPerComponent)
		left := getBitSeq(int(actline[bytePosLeft]), bitPosLeft, bitsPerComponent)
		actline[bytePosSub] = byte(calcSetBitSeq(int(actline[bytePosSub]), bitPosSub, bitsPerComponent, sub+left))
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// calculateRowLength returns the number of bytes one row occupies, rounded up
// to a whole byte.
func calculateRowLength(colors, bitsPerComponent, columns int) int {
	bitsPerPixel := colors * bitsPerComponent
	return (columns*bitsPerPixel + 7) / 8
}

// getBitSeq reads a bit field out of a byte.
func getBitSeq(by, startBit, bitSize int) int {
	mask := (1 << uint(bitSize)) - 1
	// Java uses >>> here; by is a byte value so it is never negative and a
	// signed shift is equivalent.
	return (by >> uint(startBit)) & mask
}

// calcSetBitSeq writes a bit field into a byte and returns the result. The
// value is truncated to bitSize bits.
func calcSetBitSeq(by, startBit, bitSize, val int) int {
	mask := (1 << uint(bitSize)) - 1
	truncated := val & mask
	mask = ^(mask << uint(startBit))
	return (by & mask) | (truncated << uint(startBit))
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
func readPredictorParams(decodeParams cos.ReadOnlyDictionary) predictorParams {
	p := predictorParams{predictor: 1, colors: 1, bitsPerComponent: 8, columns: 1}
	if decodeParams == nil {
		return p
	}
	p.predictor = decodeParams.GetIntDefault(cos.Predictor, 1)
	p.colors = decodeParams.GetIntDefault(cos.Colors, 1)
	p.bitsPerComponent = decodeParams.GetIntDefault(cos.BitsPerComponent, 8)
	p.columns = decodeParams.GetIntDefault(cos.Columns, 1)
	return p
}

// decodePredictor reads predicted rows from r and writes the decoded data to w.
//
// Java wraps the destination in a PredictorOutputStream and pushes bytes
// through it. The port pulls instead: a decoder here is a reader-to-writer
// copy, and pulling avoids reproducing the partial-row buffering that the Java
// stream needs in order to accept arbitrary write sizes.
func decodePredictor(w io.Writer, r io.Reader, params predictorParams) error {
	if params.predictor == 1 {
		_, err := io.Copy(w, r)
		return err
	}
	if params.colors <= 0 || params.bitsPerComponent <= 0 || params.columns <= 0 {
		return fmt.Errorf("filter: invalid predictor parameters: colors=%d bpc=%d columns=%d",
			params.colors, params.bitsPerComponent, params.columns)
	}

	rowLength := calculateRowLength(params.colors, params.bitsPerComponent, params.columns)
	if rowLength <= 0 {
		return fmt.Errorf("filter: invalid predictor row length %d", rowLength)
	}

	br := bufio.NewReader(r)
	actline := make([]byte, rowLength)
	lastline := make([]byte, rowLength)

	// A PNG predictor writes the algorithm as a leading byte on every row; a
	// TIFF predictor applies one algorithm to the whole stream.
	perRow := params.predictor >= 10

	for {
		predictor := params.predictor
		if perRow {
			b, err := br.ReadByte()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			// The row algorithm is the leading byte plus 10, so that it lands
			// in the same 10..14 range the parameter uses.
			predictor = int(b) + 10
		}

		n, err := io.ReadFull(br, actline)
		if n == 0 && (err == io.EOF || err == io.ErrUnexpectedEOF) {
			return nil
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return err
		}

		decodePredictorRow(predictor, params.colors, params.bitsPerComponent, params.columns,
			actline[:n], lastline)

		if _, err := w.Write(actline[:n]); err != nil {
			return err
		}
		if err == io.ErrUnexpectedEOF {
			// a short final row
			return nil
		}
		copy(lastline, actline)
	}
}
