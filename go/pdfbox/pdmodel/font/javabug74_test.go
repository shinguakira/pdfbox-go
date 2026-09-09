package font

// JAVA-BUGS 74: `TrueTypeEmbedder.getTag` guards a negative hash with
// `Math.abs(int)`, which answers `Integer.MIN_VALUE` unchanged for that one
// input, and the base-25 loop then indexes its alphabet with a negative
// remainder.

import (
	"math"
	"strings"
	"testing"
)

// TestSubsetTagOfTheOneHashAbsCannotFix is the defect.
//
// The expected value is derived, not read off the Java, because the Java
// raises rather than answering: `Math.abs((long) hashCode())` is the fix the
// entry names, so the number the loop encodes is 2147483648 -- MinInt32
// negated in a width that can hold it -- and the tag is that in base 25, the
// same encoding every other hash goes through.
//
// The check below does that arithmetic rather than repeating an answer, so
// what is tested is the derivation and not a copied string.
func TestSubsetTagOfTheOneHashAbsCannotFix(t *testing.T) {
	// one entry whose key ^ value is 0x80000000, so the map's hashCode is
	// exactly MinInt32
	pathological := map[int]int{0: math.MinInt32}
	var hash int32
	for gid, cid := range pathological {
		hash += int32(gid) ^ int32(cid)
	}
	if hash != math.MinInt32 {
		t.Fatalf("the map hashes to %d, want MinInt32; the case proves nothing", hash)
	}

	got := subsetTag(pathological)

	// the tag is six letters of the alphabet and a '+'
	if len(got) != 7 || !strings.HasSuffix(got, "+") {
		t.Fatalf("subsetTag = %q, want six letters and a '+'", got)
	}
	for i, c := range got[:6] {
		if !strings.ContainsRune(base25, c) {
			t.Errorf("character %d of %q is %q, which is not in the alphabet",
				i, got, c)
		}
	}

	// and it is the base-25 encoding of 2147483648, the value Math.abs of a
	// widened MinInt32 gives
	var want []byte
	for num := int64(math.MaxInt32) + 1; ; {
		want = append(want, base25[num%25])
		num /= 25
		if num == 0 || len(want) >= 6 {
			break
		}
	}
	for len(want) < 6 {
		want = append([]byte{'A'}, want...)
	}
	if wanted := string(want) + "+"; got != wanted {
		t.Errorf("subsetTag = %q, want %q", got, wanted)
	}
}

// TestSubsetTagOfAnOrdinaryMapIsUnchanged keeps the tag Java produces for
// every hash but that one, which has to match byte for byte because it becomes
// part of the font's name in the file.
func TestSubsetTagOfAnOrdinaryMapIsUnchanged(t *testing.T) {
	ordinary := map[int]int{1: 2, 3: 4}
	if got := subsetTag(ordinary); got != "AAAAAL+" {
		t.Errorf("subsetTag(%v) = %q, want Java's %q", ordinary, got, "AAAAAL+")
	}
}
