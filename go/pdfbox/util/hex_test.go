package util_test

// Port of org.apache.pdfbox.util.TestHexUtil.

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// TestGetCharsFromShortWithoutPassingInABuffer is
// testGetCharsFromShortWithoutPassingInABuffer.
//
// The last row is the interesting one: 0xCAFEBABE narrowed to a short is
// 0xBABE, so the four characters are those of the low half.
func TestGetCharsFromShortWithoutPassingInABuffer(t *testing.T) {
	// The values are held wide and narrowed at the call, which is Java's
	// (short) cast. A Go constant conversion would be rejected at compile time
	// rather than narrowing, so the cast has to happen at run time.
	for _, row := range []struct {
		value uint32
		want  string
	}{
		{0x0000, "0000"},
		{0x000F, "000F"},
		{0xABCD, "ABCD"},
		{0xCAFEBABE, "BABE"},
	} {
		t.Run(row.want, func(t *testing.T) {
			narrowed := int16(uint16(row.value))
			if got := string(util.HexChars(narrowed)); got != row.want {
				t.Errorf("HexChars(%#04x) = %q, want %q", uint16(row.value), got, row.want)
			}
		})
	}
}

// TestGetCharsUTF16BE is testGetCharsUTF16BE.
func TestGetCharsUTF16BE(t *testing.T) {
	for _, row := range []struct {
		input string
		want  string
	}{
		{"ab", "00610062"},
		{"帮助", "5E2E52A9"},
	} {
		t.Run(row.input, func(t *testing.T) {
			if got := string(util.HexCharsUTF16BE(row.input)); got != row.want {
				t.Errorf("HexCharsUTF16BE(%q) = %q, want %q", row.input, got, row.want)
			}
		})
	}
}

// TestMisc is testMisc: getBytes, getString and decodeHex over every byte, and
// then over all 256 of them at once.
func TestMisc(t *testing.T) {
	byteSrcArray := make([]byte, 256)
	for i := 0; i < 256; i++ {
		byteSrcArray[i] = byte(i)

		got := util.HexBytes(byte(i))
		if len(got) != 2 {
			t.Fatalf("HexBytes(%d) is %d bytes, want 2", i, len(got))
		}
		want := fmt.Sprintf("%02X", i)
		if string(got) != want {
			t.Errorf("HexBytes(%d) = %q, want %q", i, got, want)
		}
		if s := util.HexString(byte(i)); s != want {
			t.Errorf("HexString(%d) = %q, want %q", i, s, want)
		}
		if decoded := util.DecodeHex(want); !bytes.Equal(decoded, []byte{byte(i)}) {
			t.Errorf("DecodeHex(%q) = %v, want [%d]", want, decoded, i)
		}
	}

	byteDstArray := util.HexBytesOfBytes(byteSrcArray)
	if len(byteDstArray) != len(byteSrcArray)*2 {
		t.Fatalf("HexBytesOfBytes gave %d bytes, want %d",
			len(byteDstArray), len(byteSrcArray)*2)
	}

	dstString := util.HexStringOfBytes(byteSrcArray)
	if len(dstString) != len(byteSrcArray)*2 {
		t.Fatalf("HexStringOfBytes gave %d characters, want %d",
			len(dstString), len(byteSrcArray)*2)
	}
	if dstString != string(byteDstArray) {
		t.Error("HexStringOfBytes and HexBytesOfBytes disagree")
	}
	if decoded := util.DecodeHex(dstString); !bytes.Equal(decoded, byteSrcArray) {
		t.Error("DecodeHex did not give back the bytes it was built from")
	}
}

// TestGetHexValue is testGetHexValue: the twenty-two hex characters answer
// their value and everything else answers -256.
//
// -256 is chosen so that two of them added together stay negative, which is how
// the caller tells a bad pair from a good one.
func TestGetHexValue(t *testing.T) {
	valid := map[rune]bool{}
	for _, span := range []struct {
		from, to rune
		base     int
	}{
		{'0', '9', 0},
		{'a', 'f', 10},
		{'A', 'F', 10},
	} {
		for c := span.from; c <= span.to; c++ {
			valid[c] = true
			want := span.base + int(c-span.from)
			if got := util.HexValue(c); got != want {
				t.Errorf("HexValue(%q) = %d, want %d", c, got, want)
			}
		}
	}
	if len(valid) != 22 {
		t.Fatalf("%d characters were treated as valid hex, want 22", len(valid))
	}

	for c := rune(0); c < 256; c++ {
		if valid[c] {
			continue
		}
		if got := util.HexValue(c); got != -256 {
			t.Errorf("HexValue(%q) = %d, want -256", c, got)
		}
	}
}
