package filter

import (
	"bufio"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// runLengthEOD is the byte that ends a run length stream.
//
// Port of RunLengthDecodeFilter.RUN_LENGTH_EOD.
const runLengthEOD = 128

// RunLength decompresses data encoded using a byte-oriented run-length encoding
// algorithm, reproducing the original text or binary data.
//
// Port of org.apache.pdfbox.filter.RunLengthDecodeFilter.
type RunLength struct{}

var _ Filter = RunLength{}

// Decode expands the runs.
func (RunLength) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int) (DecodeResult, error) {
	result := DecodeResult{Parameters: parameters}
	encoded := bufio.NewReader(r)
	decoded := bufio.NewWriter(w)

	buffer := make([]byte, 128)
	for {
		dupAmount, err := encoded.ReadByte()
		if err != nil || dupAmount == runLengthEOD {
			break
		}
		if dupAmount <= 127 {
			amountToCopy := int(dupAmount) + 1
			for amountToCopy > 0 {
				compressedRead, err := encoded.Read(buffer[:amountToCopy])
				// EOF reached?
				if compressedRead == 0 && err != nil {
					break
				}
				if _, err := decoded.Write(buffer[:compressedRead]); err != nil {
					return result, err
				}
				amountToCopy -= compressedRead
			}
		} else {
			dupByte, err := encoded.ReadByte()
			// EOF reached?
			if err != nil {
				break
			}
			for i := 0; i < 257-int(dupAmount); i++ {
				if err := decoded.WriteByte(dupByte); err != nil {
					return result, err
				}
			}
		}
	}
	return result, decoded.Flush()
}

// Encode compresses the runs.
//
// Not used in PDFBox except for testing the decoder.
func (RunLength) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	input := bufio.NewReader(r)
	encoded := bufio.NewWriter(w)

	lastVal := -1
	count := 0
	equality := false

	// buffer for "unequal" runs, size between 2 and 128
	buf := make([]byte, 128)

	for {
		b, err := input.ReadByte()
		if err != nil {
			break
		}
		byt := int(b)
		if lastVal == -1 {
			// first time
			lastVal = byt
			count = 1
			continue
		}
		if count == 128 {
			if equality {
				// max length of equals
				if err := encoded.WriteByte(129); err != nil { // = 257 - 128
					return err
				}
				if err := encoded.WriteByte(byte(lastVal)); err != nil {
					return err
				}
			} else {
				// max length of unequals
				if err := encoded.WriteByte(127); err != nil {
					return err
				}
				if _, err := encoded.Write(buf[:128]); err != nil {
					return err
				}
			}
			equality = false
			lastVal = byt
			count = 1
		} else if count == 1 {
			if byt == lastVal {
				equality = true
			} else {
				buf[0] = byte(lastVal)
				buf[1] = byte(byt)
				lastVal = byt
			}
			count = 2
		} else {
			// 1 < count < 128
			if byt == lastVal {
				if equality {
					count++
				} else {
					// write all we got except the last
					if err := encoded.WriteByte(byte(count - 2)); err != nil {
						return err
					}
					if _, err := encoded.Write(buf[:count-1]); err != nil {
						return err
					}
					count = 2
					equality = true
				}
			} else {
				if equality {
					// equality ends here
					if err := encoded.WriteByte(byte(257 - count)); err != nil {
						return err
					}
					if err := encoded.WriteByte(byte(lastVal)); err != nil {
						return err
					}
					equality = false
					count = 1
				} else {
					buf[count] = byte(byt)
					count++
				}
				lastVal = byt
			}
		}
	}
	if count > 0 {
		switch {
		case count == 1:
			if err := encoded.WriteByte(0); err != nil {
				return err
			}
			if err := encoded.WriteByte(byte(lastVal)); err != nil {
				return err
			}
		case equality:
			if err := encoded.WriteByte(byte(257 - count)); err != nil {
				return err
			}
			if err := encoded.WriteByte(byte(lastVal)); err != nil {
				return err
			}
		default:
			if err := encoded.WriteByte(byte(count - 1)); err != nil {
				return err
			}
			if _, err := encoded.Write(buf[:count]); err != nil {
				return err
			}
		}
	}
	if err := encoded.WriteByte(runLengthEOD); err != nil {
		return err
	}
	return encoded.Flush()
}
