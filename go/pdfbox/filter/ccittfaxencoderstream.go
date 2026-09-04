package filter

import "io"

// ccittFaxEncoderStream compresses a one bit per pixel bitmap into a CCITT
// Group 4 (T.6) fax stream.
//
// Port of org.apache.pdfbox.filter.CCITTFaxEncoderStream, also from
// TwelveMonkeys. Java names the row method encodeRowType6 and calls encode2D
// from it; the port keeps both, because the name says which of the three row
// kinds this encoder writes.
type ccittFaxEncoderStream struct {
	currentBufferLength int
	inputBuffer         []byte
	inputBufferLength   int
	columns             int
	rows                int

	changesCurrentRow         []int
	changesReferenceRow       []int
	currentRow                int
	changesCurrentRowLength   int
	changesReferenceRowLength int
	outputBuffer              byte
	outputBufferBitLength     byte
	fillOrder                 int
	stream                    io.Writer
}

var _ io.Writer = (*ccittFaxEncoderStream)(nil)

func newCCITTFaxEncoderStream(stream io.Writer, columns, rows, fillOrder int) *ccittFaxEncoderStream {
	inputBufferLength := (columns + 7) / 8
	return &ccittFaxEncoderStream{
		stream:              stream,
		columns:             columns,
		rows:                rows,
		fillOrder:           fillOrder,
		changesReferenceRow: make([]int, columns),
		changesCurrentRow:   make([]int, columns),
		inputBufferLength:   inputBufferLength,
		inputBuffer:         make([]byte, inputBufferLength),
	}
}

// writeByte is Java's write(int b).
func (e *ccittFaxEncoderStream) writeByte(b byte) error {
	e.inputBuffer[e.currentBufferLength] = b
	e.currentBufferLength++

	if e.currentBufferLength == e.inputBufferLength {
		if err := e.encodeRow(); err != nil {
			return err
		}
		e.currentBufferLength = 0
	}
	return nil
}

func (e *ccittFaxEncoderStream) Write(p []byte) (int, error) {
	for i, b := range p {
		if err := e.writeByte(b); err != nil {
			return i, err
		}
	}
	return len(p), nil
}

// Close is Java's close, which releases the buffers; the port has nothing to
// release and does not close the writer below, because the Go filters take an
// io.Writer they do not own.
func (e *ccittFaxEncoderStream) Close() error { return nil }

func (e *ccittFaxEncoderStream) encodeRow() error {
	e.currentRow++
	e.changesReferenceRow, e.changesCurrentRow = e.changesCurrentRow, e.changesReferenceRow
	e.changesReferenceRowLength = e.changesCurrentRowLength
	e.changesCurrentRowLength = 0

	index := 0
	white := true
	for index < e.columns {
		byteIndex := index / 8
		bit := index % 8
		if ((int8(e.inputBuffer[byteIndex])>>uint(7-bit))&1 == 1) == white {
			e.changesCurrentRow[e.changesCurrentRowLength] = index
			e.changesCurrentRowLength++
			white = !white
		}
		index++
	}

	if err := e.encodeRowType6(); err != nil {
		return err
	}

	if e.currentRow == e.rows {
		if err := e.writeEOL(); err != nil {
			return err
		}
		if err := e.writeEOL(); err != nil {
			return err
		}
		return e.fill()
	}
	return nil
}

func (e *ccittFaxEncoderStream) encodeRowType6() error { return e.encode2D() }

func (e *ccittFaxEncoderStream) getNextChanges(pos int, white bool) [2]int {
	result := [2]int{e.columns, e.columns}
	for i := 0; i < e.changesCurrentRowLength; i++ {
		if pos < e.changesCurrentRow[i] || (pos == 0 && white) {
			result[0] = e.changesCurrentRow[i]
			if i+1 < e.changesCurrentRowLength {
				result[1] = e.changesCurrentRow[i+1]
			}
			break
		}
	}
	return result
}

func (e *ccittFaxEncoderStream) writeRun(runLength int, white bool) error {
	nonterm := runLength / 64
	codes := blackNonterminatingCodes
	if white {
		codes = whiteNonterminatingCodes
	}
	for nonterm > 0 {
		if nonterm >= len(codes) {
			last := codes[len(codes)-1]
			if err := e.write(last.code, last.length); err != nil {
				return err
			}
			nonterm -= len(codes)
		} else {
			c := codes[nonterm-1]
			if err := e.write(c.code, c.length); err != nil {
				return err
			}
			nonterm = 0
		}
	}

	c := blackTerminatingCodes[runLength%64]
	if white {
		c = whiteTerminatingCodes[runLength%64]
	}
	return e.write(c.code, c.length)
}

func (e *ccittFaxEncoderStream) encode2D() error {
	white := true
	index := 0 // a0
	for index < e.columns {
		nextChanges := e.getNextChanges(index, white) // a1, a2

		nextRefs := e.getNextRefChanges(index, white) // b1, b2

		difference := nextChanges[0] - nextRefs[0]
		switch {
		case nextChanges[0] > nextRefs[1]:
			// PMODE
			if err := e.write(1, 4); err != nil {
				return err
			}
			index = nextRefs[1]

		case difference > 3 || difference < -3:
			// HMODE
			if err := e.write(1, 3); err != nil {
				return err
			}
			if err := e.writeRun(nextChanges[0]-index, white); err != nil {
				return err
			}
			if err := e.writeRun(nextChanges[1]-nextChanges[0], !white); err != nil {
				return err
			}
			index = nextChanges[1]

		default:
			// VMODE
			var err error
			switch difference {
			case 0:
				err = e.write(1, 1)
			case 1:
				err = e.write(3, 3)
			case 2:
				err = e.write(3, 6)
			case 3:
				err = e.write(3, 7)
			case -1:
				err = e.write(2, 3)
			case -2:
				err = e.write(2, 6)
			case -3:
				err = e.write(2, 7)
			}
			if err != nil {
				return err
			}
			white = !white
			index = nextRefs[0] + difference
		}
	}
	return nil
}

func (e *ccittFaxEncoderStream) getNextRefChanges(a0 int, white bool) [2]int {
	result := [2]int{e.columns, e.columns}
	start := 1
	if white {
		start = 0
	}
	for i := start; i < e.changesReferenceRowLength; i += 2 {
		if e.changesReferenceRow[i] > a0 || (a0 == 0 && i == 0) {
			result[0] = e.changesReferenceRow[i]
			if i+1 < e.changesReferenceRowLength {
				result[1] = e.changesReferenceRow[i+1]
			}
			break
		}
	}
	return result
}

func (e *ccittFaxEncoderStream) write(code, codeLength int) error {
	for i := 0; i < codeLength; i++ {
		codeBit := (code>>uint(codeLength-i-1))&1 == 1
		if codeBit {
			if e.fillOrder == fillLeftToRight {
				e.outputBuffer |= 1 << uint(7-e.outputBufferBitLength%8)
			} else {
				e.outputBuffer |= 1 << uint(e.outputBufferBitLength%8)
			}
		}
		e.outputBufferBitLength++

		if e.outputBufferBitLength == 8 {
			if _, err := e.stream.Write([]byte{e.outputBuffer}); err != nil {
				return err
			}
			e.clearOutputBuffer()
		}
	}
	return nil
}

func (e *ccittFaxEncoderStream) writeEOL() error { return e.write(1, 12) }

func (e *ccittFaxEncoderStream) fill() error {
	if e.outputBufferBitLength != 0 {
		if _, err := e.stream.Write([]byte{e.outputBuffer}); err != nil {
			return err
		}
	}
	e.clearOutputBuffer()
	return nil
}

func (e *ccittFaxEncoderStream) clearOutputBuffer() {
	e.outputBuffer = 0
	e.outputBufferBitLength = 0
}

// ccittCode is one Huffman code, its bits and how many of them there are.
//
// Port of CCITTFaxEncoderStream.Code.
type ccittCode struct {
	code   int
	length int
}

// The encoder's Huffman tables, built from the decoder's the way Java's static
// initialiser builds them. A slot no run length reaches stays zero, which is
// Java's null; nothing indexes one, because every run length under 64 and every
// multiple of 64 up to 2560 has a code.
var (
	whiteTerminatingCodes    [64]ccittCode
	whiteNonterminatingCodes [40]ccittCode
	blackTerminatingCodes    [64]ccittCode
	blackNonterminatingCodes [40]ccittCode
)

func init() {
	// Setup HUFFMAN Codes
	for i := range whiteCodes {
		bitLength := i + 4
		for j := range whiteCodes[i] {
			value := whiteRunLengths[i][j]
			code := whiteCodes[i][j]

			if value < 64 {
				whiteTerminatingCodes[value] = ccittCode{code, bitLength}
			} else {
				whiteNonterminatingCodes[value/64-1] = ccittCode{code, bitLength}
			}
		}
	}

	for i := range blackCodes {
		bitLength := i + 2
		for j := range blackCodes[i] {
			value := blackRunLengths[i][j]
			code := blackCodes[i][j]

			if value < 64 {
				blackTerminatingCodes[value] = ccittCode{code, bitLength}
			} else {
				blackNonterminatingCodes[value/64-1] = ccittCode{code, bitLength}
			}
		}
	}
}
