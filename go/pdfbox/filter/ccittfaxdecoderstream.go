package filter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

// ccittFaxDecoderStream decodes a CCITT Group 3 or Group 4 fax stream into rows
// of one bit per pixel.
//
// Port of org.apache.pdfbox.filter.CCITTFaxDecoderStream, which PDFBox took
// from TwelveMonkeys. See TIFF 6.0 Specification, Section 10: "Modified Huffman
// Compression", page 43.
type ccittFaxDecoderStream struct {
	in         *bufio.Reader
	columns    int
	decodedRow []byte

	optionG32D bool
	// Leading zeros for aligning EOL
	optionG3Fill       bool
	optionUncompressed bool
	optionByteAligned  bool

	kind int

	decodedLength int
	decodedPos    int

	changesReferenceRow      []int
	changesCurrentRow        []int
	changesReferenceRowCount int
	changesCurrentRowCount   int

	lastChangingElement int

	buffer    int
	bufferPos int
}

var _ io.Reader = (*ccittFaxDecoderStream)(nil)

// errUnexpectedEndOfHuffmanRLE is Java's EOFException("Unexpected end of
// Huffman RLE stream"), which fetch catches to end the stream.
var errUnexpectedEndOfHuffmanRLE = errors.New("Unexpected end of Huffman RLE stream")

// newCCITTFaxDecoderStream builds a decoder for one image.
//
// Java throws IllegalArgumentException for a compression kind it does not know,
// which is unchecked, so the port panics.
func newCCITTFaxDecoderStream(stream io.Reader, columns, kind int, options int64,
	byteAligned bool) *ccittFaxDecoderStream {
	s := &ccittFaxDecoderStream{
		in:      bufio.NewReader(stream),
		columns: columns,
		kind:    kind,
		// We know this is only used for b/w (1 bit)
		decodedRow:          make([]byte, (columns+7)/8),
		changesReferenceRow: make([]int, columns+2),
		changesCurrentRow:   make([]int, columns+2),
		buffer:              -1,
		bufferPos:           -1,
	}

	switch kind {
	case compressionCCITTModifiedHuffmanRLE:
		s.optionByteAligned = byteAligned
	case compressionCCITTT4:
		s.optionByteAligned = byteAligned
		s.optionG32D = options&group3Opt2DEncoding != 0
		s.optionG3Fill = options&group3OptFillBits != 0
		s.optionUncompressed = options&group3OptUncompressed != 0
	case compressionCCITTT6:
		s.optionByteAligned = byteAligned
		s.optionUncompressed = options&group4OptUncompressed != 0
	default:
		panic(fmt.Sprintf("Illegal parameter: %d", kind))
	}
	return s
}

func (s *ccittFaxDecoderStream) fetch() error {
	if s.decodedPos >= s.decodedLength {
		s.decodedLength = 0

		err := s.decodeRow()
		if errors.Is(err, errUnexpectedEndOfHuffmanRLE) {
			// TODO: Rewrite to avoid throw/catch for normal flow...
			if s.decodedLength != 0 {
				return err
			}
			// ..otherwise, just let client code try to read past the
			// end of stream
			s.decodedLength = -1
		} else if err != nil {
			return err
		}

		s.decodedPos = 0
	}
	return nil
}

func (s *ccittFaxDecoderStream) decode1D() error {
	index := 0
	white := true
	s.changesCurrentRowCount = 0

	for {
		var completeRun int
		var err error
		if white {
			completeRun, err = s.decodeRun(whiteRunTree)
		} else {
			completeRun, err = s.decodeRun(blackRunTree)
		}
		if err != nil {
			return err
		}

		index += completeRun
		if err := s.appendCurrentChange(index); err != nil {
			return err
		}

		// Flip color for next run
		white = !white

		if index >= s.columns {
			return nil
		}
	}
}

// appendCurrentChange writes one changing element.
//
// Java indexes changesCurrentRow straight and lets an
// ArrayIndexOutOfBoundsException escape to decodeRow's caller, where fetch
// turns it into IOException("Malformed CCITT stream"); the port checks the
// bound and returns that error where Java's array would have thrown.
func (s *ccittFaxDecoderStream) appendCurrentChange(index int) error {
	if s.changesCurrentRowCount >= len(s.changesCurrentRow) {
		return errors.New("Malformed CCITT stream")
	}
	s.changesCurrentRow[s.changesCurrentRowCount] = index
	s.changesCurrentRowCount++
	return nil
}

func (s *ccittFaxDecoderStream) decode2D() error {
	s.changesReferenceRowCount = s.changesCurrentRowCount
	s.changesCurrentRow, s.changesReferenceRow = s.changesReferenceRow, s.changesCurrentRow

	white := true
	index := 0
	s.changesCurrentRowCount = 0

	for index < s.columns {
		// read mode
		n := codeTree.root

		for {
			bit, err := s.readBit()
			if err != nil {
				return err
			}
			n = n.walk(bit)

			if n == nil {
				break // continue mode
			}
			if !n.isLeaf {
				continue
			}

			switch n.value {
			case valueHMode:
				var runLength int
				tree := blackRunTree
				if white {
					tree = whiteRunTree
				}
				if runLength, err = s.decodeRun(tree); err != nil {
					return err
				}
				index += runLength
				if err := s.appendCurrentChange(index); err != nil {
					return err
				}

				tree = whiteRunTree
				if white {
					tree = blackRunTree
				}
				if runLength, err = s.decodeRun(tree); err != nil {
					return err
				}
				index += runLength
				if err := s.appendCurrentChange(index); err != nil {
					return err
				}

			case valuePassMode:
				pChangingElement := s.getNextChangingElement(index, white) + 1

				if pChangingElement >= s.changesReferenceRowCount {
					index = s.columns
				} else {
					index = s.changesReferenceRow[pChangingElement]
				}

			default:
				// Vertical mode (-3 to 3)
				vChangingElement := s.getNextChangingElement(index, white)

				if vChangingElement >= s.changesReferenceRowCount || vChangingElement == -1 {
					index = s.columns + n.value
				} else {
					index = s.changesReferenceRow[vChangingElement] + n.value
				}

				if err := s.appendCurrentChange(index); err != nil {
					return err
				}
				white = !white
			}

			break // continue mode
		}
	}
	return nil
}

func (s *ccittFaxDecoderStream) getNextChangingElement(a0 int, white bool) int {
	// Java masks with 0xFFFF_FFFE, which clears the low bit of a 32 bit int;
	// the port writes the same mask so that a value above 2^31 would narrow the
	// same way, though lastChangingElement never gets there.
	start := int(int32(s.lastChangingElement)&int32(-2)) + boolToInt(!white)
	if start > 2 {
		start -= 2
	}

	if a0 == 0 {
		return start
	}

	for i := start; i < s.changesReferenceRowCount; i += 2 {
		if a0 < s.changesReferenceRow[i] {
			s.lastChangingElement = i
			return i
		}
	}

	return -1
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *ccittFaxDecoderStream) decodeRowType2() error {
	if s.optionByteAligned {
		s.resetBuffer()
	}
	return s.decode1D()
}

func (s *ccittFaxDecoderStream) decodeRowType4() error {
	if s.optionByteAligned {
		s.resetBuffer()
	}
	// read till next EOL code
	for {
		n := eolOnlyTree.root
		done := false
		for {
			bit, err := s.readBit()
			if err != nil {
				return err
			}
			n = n.walk(bit)

			if n == nil {
				break // continue eof
			}
			if n.isLeaf {
				done = true
				break // break eof
			}
		}
		if done {
			break
		}
	}

	if !s.optionG32D {
		return s.decode1D()
	}
	bit, err := s.readBit()
	if err != nil {
		return err
	}
	if bit {
		return s.decode1D()
	}
	return s.decode2D()
}

func (s *ccittFaxDecoderStream) decodeRowType6() error {
	if s.optionByteAligned {
		s.resetBuffer()
	}
	return s.decode2D()
}

func (s *ccittFaxDecoderStream) decodeRow() error {
	var err error
	switch s.kind {
	case compressionCCITTModifiedHuffmanRLE:
		err = s.decodeRowType2()
	case compressionCCITTT4:
		err = s.decodeRowType4()
	case compressionCCITTT6:
		err = s.decodeRowType6()
	default:
		panic(fmt.Sprintf("Illegal parameter: %d", s.kind))
	}
	if err != nil {
		return err
	}

	index := 0
	white := true

	s.lastChangingElement = 0
	for i := 0; i <= s.changesCurrentRowCount; i++ {
		nextChange := s.columns

		if i != s.changesCurrentRowCount {
			nextChange = s.changesCurrentRow[i]
		}

		if nextChange > s.columns {
			nextChange = s.columns
		}

		byteIndex := index / 8

		for index%8 != 0 && (nextChange-index) > 0 {
			s.decodedRow[byteIndex] |= whiteOrBit(white, index)
			index++
		}

		if index%8 == 0 {
			byteIndex = index / 8
			value := byte(0x00)
			if !white {
				value = 0xff
			}

			for (nextChange - index) > 7 {
				s.decodedRow[byteIndex] = value
				index += 8
				byteIndex++
			}
		}

		for (nextChange - index) > 0 {
			if index%8 == 0 {
				s.decodedRow[byteIndex] = 0
			}

			s.decodedRow[byteIndex] |= whiteOrBit(white, index)
			index++
		}

		white = !white
	}

	if index != s.columns {
		return fmt.Errorf("Sum of run-lengths does not equal scan line width: %d > %d",
			index, s.columns)
	}

	s.decodedLength = (index + 7) / 8
	return nil
}

// whiteOrBit is Java's `(white ? 0 : 1 << (7 - (index % 8)))`.
func whiteOrBit(white bool, index int) byte {
	if white {
		return 0
	}
	return 1 << uint(7-index%8)
}

func (s *ccittFaxDecoderStream) decodeRun(tree *ccittTree) (int, error) {
	total := 0

	n := tree.root

	for {
		bit, err := s.readBit()
		if err != nil {
			return 0, err
		}
		n = n.walk(bit)

		if n == nil {
			return 0, errors.New("Unknown code in Huffman RLE stream")
		}

		if n.isLeaf {
			total += n.value
			switch {
			case n.value >= 64:
				n = tree.root
			case n.value >= 0:
				return total, nil
			default:
				return s.columns, nil
			}
		}
	}
}

func (s *ccittFaxDecoderStream) resetBuffer() { s.bufferPos = -1 }

func (s *ccittFaxDecoderStream) readBit() (bool, error) {
	if s.bufferPos < 0 || s.bufferPos > 7 {
		c, err := s.in.ReadByte()
		if err != nil {
			s.buffer = -1
			return false, errUnexpectedEndOfHuffmanRLE
		}
		s.buffer = int(c)
		s.bufferPos = 0
	}

	isSet := s.buffer&0x80 != 0
	s.buffer <<= 1
	s.bufferPos++

	return isSet, nil
}

// Read fills b with decoded rows.
//
// Port of read(byte[], int, int). Past the end of the encoded data it fills
// with zeroes and reports the full length rather than the end of the stream,
// which is what lets CCITTFaxFilter.readFromDecoderStream keep going until the
// bitmap it sized from /Columns and /Rows is full.
func (s *ccittFaxDecoderStream) Read(b []byte) (int, error) {
	if s.decodedLength < 0 {
		fillZero(b)
		return len(b), nil
	}

	if s.decodedPos >= s.decodedLength {
		if err := s.fetch(); err != nil {
			return 0, err
		}

		if s.decodedLength < 0 {
			fillZero(b)
			return len(b), nil
		}
	}

	read := s.decodedLength - s.decodedPos
	if len(b) < read {
		read = len(b)
	}
	copy(b, s.decodedRow[s.decodedPos:s.decodedPos+read])
	s.decodedPos += read

	return read, nil
}

func fillZero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ccittNode is one node of a CCITT Huffman tree.
//
// Port of CCITTFaxDecoderStream.Node. Java's canBeFill field is set by the tree
// builder and read by nothing, so the port leaves it out.
type ccittNode struct {
	left   *ccittNode
	right  *ccittNode
	value  int // > 63 non term.
	isLeaf bool
}

func (n *ccittNode) set(next bool, node *ccittNode) {
	if !next {
		n.left = node
	} else {
		n.right = node
	}
}

func (n *ccittNode) walk(next bool) *ccittNode {
	if next {
		return n.right
	}
	return n.left
}

// ccittTree is a CCITT Huffman tree.
//
// Port of CCITTFaxDecoderStream.Tree.
type ccittTree struct {
	root *ccittNode
}

func newCCITTTree() *ccittTree { return &ccittTree{root: &ccittNode{}} }

// fillValue adds a leaf for a code.
//
// Java's two fill overloads differ only in whether the leaf is made here or
// handed in; both raise IOException("node is leaf, no other following") for a
// code that is a prefix of one already there, which only a wrong table can
// cause, so the port panics where Java's static initialiser turns it into an
// AssertionError.
func (t *ccittTree) fillValue(depth, path, value int) {
	t.fill(depth, path, nil, value)
}

func (t *ccittTree) fillNode(depth, path int, node *ccittNode) {
	t.fill(depth, path, node, 0)
}

func (t *ccittTree) fill(depth, path int, node *ccittNode, value int) {
	current := t.root

	for i := 0; i < depth; i++ {
		bitPos := depth - 1 - i
		isSet := (path>>uint(bitPos))&1 == 1
		next := current.walk(isSet)

		if next == nil {
			if i == depth-1 && node != nil {
				next = node
			} else {
				next = &ccittNode{}
				if i == depth-1 {
					next.value = value
					next.isLeaf = true
				}
			}
			current.set(isSet, next)
		} else if next.isLeaf {
			panic("node is leaf, no other following")
		}

		current = next
	}
}

// The run length code tables, copied from CCITTFaxDecoderStream. The outer
// index is the code length: two bits upward for black, four upward for white.
var blackCodes = [][]int{
	{ // 2 bits
		0x2, 0x3,
	},
	{ // 3 bits
		0x2, 0x3,
	},
	{ // 4 bits
		0x2, 0x3,
	},
	{ // 5 bits
		0x3,
	},
	{ // 6 bits
		0x4, 0x5,
	},
	{ // 7 bits
		0x4, 0x5, 0x7,
	},
	{ // 8 bits
		0x4, 0x7,
	},
	{ // 9 bits
		0x18,
	},
	{ // 10 bits
		0x17, 0x18, 0x37, 0x8, 0xf,
	},
	{ // 11 bits
		0x17, 0x18, 0x28, 0x37, 0x67, 0x68, 0x6c, 0x8, 0xc, 0xd,
	},
	{ // 12 bits
		0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x1c, 0x1d, 0x1e, 0x1f, 0x24, 0x27, 0x28, 0x2b, 0x2c, 0x33,
		0x34, 0x35, 0x37, 0x38, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x5b, 0x64, 0x65,
		0x66, 0x67, 0x68, 0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0xc8, 0xc9, 0xca, 0xcb, 0xcc, 0xcd, 0xd2, 0xd3,
		0xd4, 0xd5, 0xd6, 0xd7, 0xda, 0xdb,
	},
	{ // 13 bits
		0x4a, 0x4b, 0x4c, 0x4d, 0x52, 0x53, 0x54, 0x55, 0x5a, 0x5b, 0x64, 0x65, 0x6c, 0x6d, 0x72, 0x73,
		0x74, 0x75, 0x76, 0x77,
	},
}

var blackRunLengths = [][]int{
	{ // 2 bits
		3, 2,
	},
	{ // 3 bits
		1, 4,
	},
	{ // 4 bits
		6, 5,
	},
	{ // 5 bits
		7,
	},
	{ // 6 bits
		9, 8,
	},
	{ // 7 bits
		10, 11, 12,
	},
	{ // 8 bits
		13, 14,
	},
	{ // 9 bits
		15,
	},
	{ // 10 bits
		16, 17, 0, 18, 64,
	},
	{ // 11 bits
		24, 25, 23, 22, 19, 20, 21, 1792, 1856, 1920,
	},
	{ // 12 bits
		1984, 2048, 2112, 2176, 2240, 2304, 2368, 2432, 2496, 2560, 52, 55, 56, 59, 60, 320, 384, 448, 53,
		54, 50, 51, 44, 45, 46, 47, 57, 58, 61, 256, 48, 49, 62, 63, 30, 31, 32, 33, 40, 41, 128, 192, 26,
		27, 28, 29, 34, 35, 36, 37, 38, 39, 42, 43,
	},
	{ // 13 bits
		640, 704, 768, 832, 1280, 1344, 1408, 1472, 1536, 1600, 1664, 1728, 512, 576, 896, 960, 1024, 1088,
		1152, 1216,
	},
}

var whiteCodes = [][]int{
	{ // 4 bits
		0x7, 0x8, 0xb, 0xc, 0xe, 0xf,
	},
	{ // 5 bits
		0x12, 0x13, 0x14, 0x1b, 0x7, 0x8,
	},
	{ // 6 bits
		0x17, 0x18, 0x2a, 0x2b, 0x3, 0x34, 0x35, 0x7, 0x8,
	},
	{ // 7 bits
		0x13, 0x17, 0x18, 0x24, 0x27, 0x28, 0x2b, 0x3, 0x37, 0x4, 0x8, 0xc,
	},
	{ // 8 bits
		0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x1a, 0x1b, 0x2, 0x24, 0x25, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d,
		0x3, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x4, 0x4a, 0x4b, 0x5, 0x52, 0x53, 0x54, 0x55, 0x58, 0x59,
		0x5a, 0x5b, 0x64, 0x65, 0x67, 0x68, 0xa, 0xb,
	},
	{ // 9 bits
		0x98, 0x99, 0x9a, 0x9b, 0xcc, 0xcd, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0xda, 0xdb,
	},
	{ // 10 bits
	},
	{ // 11 bits
		0x8, 0xc, 0xd,
	},
	{ // 12 bits
		0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x1c, 0x1d, 0x1e, 0x1f,
	},
}

var whiteRunLengths = [][]int{
	{ // 4 bits
		2, 3, 4, 5, 6, 7,
	},
	{ // 5 bits
		128, 8, 9, 64, 10, 11,
	},
	{ // 6 bits
		192, 1664, 16, 17, 13, 14, 15, 1, 12,
	},
	{ // 7 bits
		26, 21, 28, 27, 18, 24, 25, 22, 256, 23, 20, 19,
	},
	{ // 8 bits
		33, 34, 35, 36, 37, 38, 31, 32, 29, 53, 54, 39, 40, 41, 42, 43, 44, 30, 61, 62, 63, 0, 320, 384, 45,
		59, 60, 46, 49, 50, 51, 52, 55, 56, 57, 58, 448, 512, 640, 576, 47, 48,
	},
	{ // 9 bits
		1472, 1536, 1600, 1728, 704, 768, 832, 896, 960, 1024, 1088, 1152, 1216, 1280, 1344, 1408,
	},
	{ // 10 bits
	},
	{ // 11 bits
		1792, 1856, 1920,
	},
	{ // 12 bits
		1984, 2048, 2112, 2176, 2240, 2304, 2368, 2432, 2496, 2560,
	},
}

// The sentinel values the trees carry in place of a run length.
const (
	valueEOL      = -2000
	valueFill     = -1000
	valuePassMode = -3000
	valueHMode    = -4000
)

var (
	ccittEOL     *ccittNode
	ccittFill    *ccittNode
	blackRunTree *ccittTree
	whiteRunTree *ccittTree
	eolOnlyTree  *ccittTree
	codeTree     *ccittTree
)

// Port of the static initialiser of CCITTFaxDecoderStream.
func init() {
	ccittEOL = &ccittNode{isLeaf: true, value: valueEOL}
	ccittFill = &ccittNode{value: valueFill}
	ccittFill.left = ccittFill
	ccittFill.right = ccittEOL

	eolOnlyTree = newCCITTTree()
	eolOnlyTree.fillNode(12, 0, ccittFill)
	eolOnlyTree.fillNode(12, 1, ccittEOL)

	blackRunTree = newCCITTTree()
	for i := range blackCodes {
		for j := range blackCodes[i] {
			blackRunTree.fillValue(i+2, blackCodes[i][j], blackRunLengths[i][j])
		}
	}
	blackRunTree.fillNode(12, 0, ccittFill)
	blackRunTree.fillNode(12, 1, ccittEOL)

	whiteRunTree = newCCITTTree()
	for i := range whiteCodes {
		for j := range whiteCodes[i] {
			whiteRunTree.fillValue(i+4, whiteCodes[i][j], whiteRunLengths[i][j])
		}
	}
	whiteRunTree.fillNode(12, 0, ccittFill)
	whiteRunTree.fillNode(12, 1, ccittEOL)

	codeTree = newCCITTTree()
	codeTree.fillValue(4, 1, valuePassMode) // pass mode
	codeTree.fillValue(3, 1, valueHMode)    // H mode
	codeTree.fillValue(1, 1, 0)             // V(0)
	codeTree.fillValue(3, 3, 1)             // V_R(1)
	codeTree.fillValue(6, 3, 2)             // V_R(2)
	codeTree.fillValue(7, 3, 3)             // V_R(3)
	codeTree.fillValue(3, 2, -1)            // V_L(1)
	codeTree.fillValue(6, 2, -2)            // V_L(2)
	codeTree.fillValue(7, 2, -3)            // V_L(3)
}
