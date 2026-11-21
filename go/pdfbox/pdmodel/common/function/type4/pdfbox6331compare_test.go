package type4

// PDFBOX-6331, Apache 47d20cb71, in the sync of 2026-11-21, and JAVA-BUGS.md
// 93.
//
// The operands of eq, ne, ge, gt, le and lt arrive as Number, which in a type 4
// program is an Integer or a Float, and every comparison narrowed both to
// float. A float carries 24 bits of mantissa, so two ints that differ above
// 16,777,216 compared equal and the ordering operators answered on the rounded
// values. Upstream moved Eq to Double.compare and Ge, Gt and Le to
// doubleValue().
//
// **Lt it did not move**, and the port follows the Java rather than the
// evident intention: TestLtStillComparesInFloat below is the pin on that, and
// it fails if someone widens lt to match the other three.
//
// 16777216 is 2^24 and 16777217 is the first integer a float cannot hold; the
// expected values below are what each operator answers for that pair in the
// Java as it now stands.

import "testing"

// TestEqOfTwoIntegersAboveTheFloatMantissa is the fix for eq, and ne with it:
// the two integers are different, and before this they compared equal because
// both became the same float.
func TestEqOfTwoIntegersAboveTheFloatMantissa(t *testing.T) {
	newType4Tester(t, "16777216 16777217 eq").popBool(false).isEmpty()
	newType4Tester(t, "16777216 16777217 ne").popBool(true).isEmpty()
}

// TestGtOfTwoIntegersAboveTheFloatMantissa is gt, which answered false for a
// pair it can now tell apart.
func TestGtOfTwoIntegersAboveTheFloatMantissa(t *testing.T) {
	newType4Tester(t, "16777217 16777216 gt").popBool(true).isEmpty()
}

// TestGeOfTwoIntegersAboveTheFloatMantissa is ge, which answered true for a
// pair where the left side is the smaller.
func TestGeOfTwoIntegersAboveTheFloatMantissa(t *testing.T) {
	newType4Tester(t, "16777216 16777217 ge").popBool(false).isEmpty()
}

// TestLeOfTwoIntegersAboveTheFloatMantissa is le, the mirror of ge.
func TestLeOfTwoIntegersAboveTheFloatMantissa(t *testing.T) {
	newType4Tester(t, "16777217 16777216 le").popBool(false).isEmpty()
}

// TestLtStillComparesInFloat is the Java upstream did not change. Lt.compare is
// still num1.floatValue() < num2.floatValue(), so both operands round to
// 16777216 and the answer is false where the other three operators now see the
// difference. The port keeps that, because the Java is the specification and
// the Lt arm of 47d20cb71 was not written.
//
// This test passes with the rest of this commit reverted, which is the point of
// it: it pins a divergence from the obvious reading of upstream's intent.
func TestLtStillComparesInFloat(t *testing.T) {
	newType4Tester(t, "16777216 16777217 lt").popBool(false).isEmpty()
}

// TestComparisonsOfOrdinaryNumbersAreUnchanged is the guard on the widening. It
// passes either way, and would stop passing if comparing in float64 moved an
// answer the Java's own tests hold.
func TestComparisonsOfOrdinaryNumbersAreUnchanged(t *testing.T) {
	newType4Tester(t, "0.5 0.4 gt 0.5 0.5 ge 2 3 le 7 7 eq 1 2 lt").
		popBool(true).popBool(true).popBool(true).popBool(true).popBool(true).isEmpty()
}
