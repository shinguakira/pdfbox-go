package filter

import "testing"

// Ported from pdfbox/src/test/java/org/apache/pdfbox/filter/PredictorTest.java.
//
// The Java assertions are written as Integer.parseInt("...", 2) so the bit
// patterns are readable; Go has 0b literals, which say the same thing.

func TestGetBitSeq(t *testing.T) {
	cases := []struct {
		by, startBit, bitSize, want int
	}{
		{0b11111111, 0, 8, 0b11111111},
		{0b00000000, 0, 8, 0b00000000},
		{0b11111111, 0, 1, 0b1},
		{0b00000000, 0, 1, 0b0},
		{0b00110001, 0, 3, 0b001},
		{0b10101010, 0, 8, 0b10101010},
		{0b10101010, 0, 2, 0b10},
		{0b10101010, 1, 2, 0b01},
		{0b10101010, 2, 2, 0b10},
		{0b10101010, 3, 3, 0b101},
		{0b10101010, 1, 7, 0b1010101},
		{0b10101010, 3, 2, 0b01},
		{0b00110001, 0, 8, 0b00110001},
		{0b00110001, 0, 5, 0b10001},
		{0b00110001, 4, 4, 0b0011},
		{0b00110001, 3, 3, 0b110},
		{0b00110001, 6, 2, 0b00},
		{0b11110000, 4, 4, 0b1111},
		{0b11110000, 6, 2, 0b11},
		{0b11110000, 0, 4, 0b0000},
	}
	for _, c := range cases {
		if got := getBitSeq(c.by, c.startBit, c.bitSize); got != c.want {
			t.Errorf("getBitSeq(%08b, %d, %d) = %08b, want %08b",
				c.by, c.startBit, c.bitSize, got, c.want)
		}
	}
}

func TestCalcSetBitSeq(t *testing.T) {
	cases := []struct {
		by, startBit, bitSize, val, want int
	}{
		{0b11111111, 0, 8, 0, 0b00000000},
		{0b11111111, 0, 8, 1, 0b00000001},
		{0b11111111, 0, 1, 1, 0b11111111},
		{0b11111111, 0, 2, 1, 0b11111101},
		{0b11111111, 0, 3, 1, 0b11111001},
		{0b00000000, 0, 2, 1, 0b00000001},
		{0b11111111, 0, 4, 1, 0b11110001},
		{0b11111111, 1, 4, 1, 0b11100011},
		{0b00000000, 1, 1, 1, 0b00000010},
		{0b11111111, 7, 1, 1, 0b11111111},
		{0b11111111, 7, 1, 0, 0b01111111},
		{0b00000000, 7, 1, 1, 0b10000000},
		{0b00000000, 7, 1, 0, 0b00000000},
		{0b00000000, 6, 1, 1, 0b01000000},
		{0b00000000, 6, 1, 0, 0b00000000},
		{0b00000000, 3, 3, 6, 0b00110000},
		{0b00000000, 4, 3, 6, 0b01100000},
		{0b00000000, 5, 3, 6, 0b11000000},
		{0b00000000, 0, 8, 0xFF, 0b11111111},
		{0b11111111, 0, 8, 0xFF, 0b11111111},
		{0xA5, 0, 8, 0xD9 + 0xA5, 0x7E},
		// the value is truncated to bitSize bits
		{0b00000000, 1, 1, 3, 0b00000010},
	}
	for _, c := range cases {
		if got := calcSetBitSeq(c.by, c.startBit, c.bitSize, c.val); got != c.want {
			t.Errorf("calcSetBitSeq(%08b, %d, %d, %d) = %08b, want %08b",
				c.by, c.startBit, c.bitSize, c.val, got, c.want)
		}
	}
}

// TestCalculateRowLength covers the row-length arithmetic every predictor
// depends on. Java has no direct test for it.
func TestCalculateRowLength(t *testing.T) {
	cases := []struct {
		colors, bitsPerComponent, columns, want int
	}{
		{1, 8, 1, 1},
		{3, 8, 4, 12},
		{1, 1, 8, 1},
		{1, 1, 9, 2}, // rounds up to a whole byte
		{1, 4, 3, 2}, // 12 bits rounds up to 2
		{4, 8, 5, 20},
	}
	for _, c := range cases {
		got := calculateRowLength(c.colors, c.bitsPerComponent, c.columns)
		if got != c.want {
			t.Errorf("calculateRowLength(%d, %d, %d) = %d, want %d",
				c.colors, c.bitsPerComponent, c.columns, got, c.want)
		}
	}
}

// TestDecodePredictorRowPNG covers the PNG predictors, which are what a PDF
// cross-reference stream uses — predictor 12 (Up) almost always.
func TestDecodePredictorRowPNG(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		row := []byte{1, 2, 3}
		decodePredictorRow(10, 1, 8, 3, row, []byte{9, 9, 9})
		assertBytes(t, []byte{1, 2, 3}, row)
	})

	t.Run("sub", func(t *testing.T) {
		// each byte adds the one bytesPerPixel to its left
		row := []byte{10, 5, 5}
		decodePredictorRow(11, 1, 8, 3, row, []byte{0, 0, 0})
		assertBytes(t, []byte{10, 15, 20}, row)
	})

	t.Run("up", func(t *testing.T) {
		// each byte adds the byte above it
		row := []byte{1, 2, 3}
		decodePredictorRow(12, 1, 8, 3, row, []byte{10, 20, 30})
		assertBytes(t, []byte{11, 22, 33}, row)
	})

	t.Run("up wraps at 255", func(t *testing.T) {
		row := []byte{1}
		decodePredictorRow(12, 1, 8, 1, row, []byte{255})
		assertBytes(t, []byte{0}, row)
	})

	t.Run("average", func(t *testing.T) {
		// value + (left + up) / 2, with left 0 at the start of the row
		row := []byte{10, 10}
		decodePredictorRow(13, 1, 8, 2, row, []byte{20, 20})
		// p0: 10 + (0+20)/2 = 20 ; p1: 10 + (20+20)/2 = 30
		assertBytes(t, []byte{20, 30}, row)
	})

	t.Run("paeth", func(t *testing.T) {
		// with a zero prior row and zero left, the predictor contributes 0
		row := []byte{7, 8, 9}
		decodePredictorRow(14, 1, 8, 3, row, []byte{0, 0, 0})
		assertBytes(t, []byte{7, 15, 24}, row)
	})
}

// TestDecodePredictorRowTIFF covers predictor 2, which for 8 bits per component
// is the same algorithm as the PNG Sub predictor.
func TestDecodePredictorRowTIFF(t *testing.T) {
	row := []byte{10, 5, 5}
	decodePredictorRow(2, 1, 8, 3, row, []byte{0, 0, 0})
	assertBytes(t, []byte{10, 15, 20}, row)
}

func TestDecodePredictorRowNoPrediction(t *testing.T) {
	row := []byte{1, 2, 3}
	decodePredictorRow(1, 1, 8, 3, row, []byte{9, 9, 9})
	assertBytes(t, []byte{1, 2, 3}, row)
}

func assertBytes(t *testing.T, want, got []byte) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("byte %d = %d, want %d (got % d, want % d)", i, got[i], want[i], got, want)
		}
	}
}

// TestCalculateRowLengthOverflowsAsJavaDoes pins the width of the arithmetic,
// which is load-bearing here and is not the width Go's int has.
//
// Found by benchmarking: qpdf/issue-1688a.pdf is 531 bytes and declares
// `/Predictor 2 /Colors 536870913 /Columns 1 /BitsPerComponent 8`. PDFBox reads
// it in 0.3ms; the port took 435ms, and 83% of that was the runtime zeroing
// memory.
//
// The reason is the multiply. Java's int is 32 bits, so 536870913 * 8 wraps to
// 8, bytesPerPixel is 1, and the row is one byte long. Go's int is 64 bits on
// this machine, so the same expression is 4294967304, the row is 536,870,913
// bytes, and decodePredictor allocates two of them -- a gigabyte of zeroed
// memory from a 531-byte file.
//
// conventions/java-to-go.md already said which width to use: "Java int is
// 32-bit: use int32 where the width is load-bearing (format fields,
// overflow-sensitive arithmetic)". This is that case, and the port had int.
//
// So the expected values are Java's, overflow included. Reproducing the wrap is
// the point: PDFBox's answer for this file is a one-byte row, and a port that
// computes the arithmetically honest answer instead disagrees with it and hands
// an attacker a gigabyte for 531 bytes. Recorded in JAVA-BUGS.md.
func TestCalculateRowLengthOverflowsAsJavaDoes(t *testing.T) {
	cases := []struct {
		name                              string
		colors, bitsPerComponent, columns int
		want                              int
	}{
		{"the ordinary case", 3, 8, 100, 300},
		{"one bit per component", 1, 1, 17, 3},
		// 536870913 * 8 == 0x2_0000_0008, which is 8 in 32 bits
		{"qpdf issue-1688a", 536870913, 8, 1, 1},
		// 268435457 * 16 == 0x1_0000_0010, which is 16 in 32 bits
		{"the same wrap at 16 bits per component", 268435457, 16, 1, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := calculateRowLength(c.colors, c.bitsPerComponent, c.columns); got != c.want {
				t.Errorf("calculateRowLength(%d, %d, %d) = %d, want %d",
					c.colors, c.bitsPerComponent, c.columns, got, c.want)
			}
		})
	}
}
