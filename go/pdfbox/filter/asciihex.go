package filter

import (
	"bufio"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// reverseHex maps a byte of an ASCIIHex stream to its nibble value, or -1 where
// the byte is not a hexadecimal digit.
//
// Port of ASCIIHexFilter.REVERSE_HEX, entry for entry.
var reverseHex = [256]int{
	/*   0 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/*  10 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/*  20 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/*  30 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/*  40 */ -1, -1, -1, -1, -1, -1, -1, -1, 0, 1,
	/*  50 */ 2, 3, 4, 5, 6, 7, 8, 9, -1, -1,
	/*  60 */ -1, -1, -1, -1, -1, 10, 11, 12, 13, 14,
	/*  70 */ 15, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/*  80 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/*  90 */ -1, -1, -1, -1, -1, -1, -1, 10, 11, 12,
	/* 100 */ 13, 14, 15, -1, -1, -1, -1, -1, -1, -1,
	/* 110 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 120 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 130 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 140 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 150 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 160 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 170 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 180 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 190 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 200 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 210 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 220 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 230 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 240 */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	/* 250 */ -1, -1, -1, -1, -1, -1,
}

// hexDigits is the alphabet Hex.writeHexByte writes with.
//
// Port of org.apache.pdfbox.util.Hex.DIGITS, which is upper case.
const hexDigits = "0123456789ABCDEF"

// ASCIIHex decodes data encoded in an ASCII hexadecimal form, reproducing the
// original binary data.
//
// Port of org.apache.pdfbox.filter.ASCIIHexFilter.
type ASCIIHex struct{}

var _ Filter = ASCIIHex{}

// Decode reads the hexadecimal digits and writes the bytes they name.
func (ASCIIHex) Decode(w io.Writer, r io.Reader, parameters *cos.Dictionary,
	index int) (DecodeResult, error) {
	result := DecodeResult{Parameters: parameters}
	encoded := bufio.NewReader(r)
	decoded := bufio.NewWriter(w)

	for {
		firstByte, err := encoded.ReadByte()
		if err != nil {
			break
		}
		// always after first char
		atEOF := false
		for isHexWhitespace(firstByte) {
			firstByte, err = encoded.ReadByte()
			if err != nil {
				atEOF = true
				break
			}
		}
		if atEOF || isHexEOD(firstByte) {
			break
		}

		if reverseHex[firstByte] == -1 {
			slog.Error("filter: invalid hex", "int", firstByte,
				"char", string(rune(firstByte)), "position", "1st byte")
		}
		value := reverseHex[firstByte] * 16
		secondByte, err := encoded.ReadByte()
		if err != nil || isHexEOD(secondByte) {
			// second value behaves like 0 in case of EOD
			if err2 := decoded.WriteByte(byte(value)); err2 != nil {
				return result, err2
			}
			break
		}
		if reverseHex[secondByte] == -1 {
			slog.Error("filter: invalid hex", "int", secondByte,
				"char", string(rune(secondByte)), "position", "2nd byte")
		}
		value += reverseHex[secondByte]
		if err := decoded.WriteByte(byte(value)); err != nil {
			return result, err
		}
	}
	return result, decoded.Flush()
}

// isHexWhitespace reports whether the byte is one PDFBox skips between digits:
//
//	 0  0x00  Null (NUL)
//	 9  0x09  Tab (HT)
//	10  0x0A  Line feed (LF)
//	12  0x0C  Form feed (FF)
//	13  0x0D  Carriage return (CR)
//	32  0x20  Space (SP)
func isHexWhitespace(c byte) bool {
	switch c {
	case 0, 9, 10, 12, 13, 32:
		return true
	default:
		return false
	}
}

func isHexEOD(c byte) bool { return c == '>' }

// Encode writes each byte as two upper case hexadecimal digits.
func (ASCIIHex) Encode(w io.Writer, r io.Reader, parameters *cos.Dictionary) error {
	input := bufio.NewReader(r)
	encoded := bufio.NewWriter(w)
	for {
		byteRead, err := input.ReadByte()
		if err != nil {
			break
		}
		// Port of Hex.writeHexByte, which writes the high nibble then the low.
		if err := encoded.WriteByte(hexDigits[(byteRead>>4)&0x0F]); err != nil {
			return err
		}
		if err := encoded.WriteByte(hexDigits[byteRead&0x0F]); err != nil {
			return err
		}
	}
	return encoded.Flush()
}
