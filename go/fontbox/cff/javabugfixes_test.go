package cff

// JAVA-BUGS 17: `concatenateMatrix` multiplies one cell by the wrong matrix.
// In-package, because the function is.

import "testing"

// TestConcatenateMatrixUsesTheSecondMatrixThroughout is the defect.
//
// Five of the six cells read the second matrix; `matrixDest[1]` reads
// `b1 * d1` where every other row reads `d2`. The expected values are the
// product of the two matrices, which is what the comment above the function
// draws:
//
//	(a b 0)   (a b 0)
//	(c d 0) x (c d 0)
//	(x y 1)   (x y 1)
//
// For [1 2 3 4 5 6] concatenated with [7 8 9 10 11 12] that is
// [1*7+2*9, 1*8+2*10, 3*7+4*9, 3*8+4*10, 5*7+6*9+11, 5*8+6*10+12]
// = [25, 28, 57, 64, 100, 112]. Java answers 1*8 + 2*4 = 16 for the second.
func TestConcatenateMatrixUsesTheSecondMatrixThroughout(t *testing.T) {
	dest := []any{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}
	concat := []any{7.0, 8.0, 9.0, 10.0, 11.0, 12.0}
	concatenateMatrix(dest, concat)

	want := []float64{25, 28, 57, 64, 100, 112}
	for i, w := range want {
		if got := numberDouble(dest[i]); got != w {
			t.Errorf("cell %d is %v, want %v", i, got, w)
		}
	}
}
