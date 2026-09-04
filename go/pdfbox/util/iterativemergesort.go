package util

// IterativeMergeSort sorts the slice with the given comparison, which is a
// stable merge sort that never fails on a comparison that is not a total order.
//
// Port of org.apache.pdfbox.util.IterativeMergeSort. PDFTextStripper falls back
// to it because TextPositionComparator turns out not to be transitive on some
// documents, and the JDK's sort throws "Comparison method violates its general
// contract!" when it notices. Go's sort.SliceStable does not check, so it would
// not fail there; the port keeps this sort anyway, because it is what decides
// the order the stripper writes those documents out in.
//
// cmp returns a negative number where a sorts before b, zero where they tie,
// and a positive number otherwise -- the same contract as Java's Comparator.
func IterativeMergeSort[T any](list []T, cmp func(a, b T) int) {
	if len(list) < 2 {
		return
	}
	aux := make([]T, len(list))
	copy(aux, list)
	for blockSize := 1; blockSize < len(list); blockSize <<= 1 {
		for start := 0; start < len(list); start += blockSize << 1 {
			mergeRuns(list, aux, start, start+blockSize, start+(blockSize<<1), cmp)
		}
	}
}

// mergeRuns merges the two sorted runs [from, mid) and [mid, to) of arr into
// aux, then copies the result back.
func mergeRuns[T any](arr, aux []T, from, mid, to int, cmp func(a, b T) int) {
	if mid >= len(arr) {
		return
	}
	if to > len(arr) {
		to = len(arr)
	}
	i := from
	j := mid
	for k := from; k < to; k++ {
		switch {
		case i == mid:
			aux[k] = arr[j]
			j++
		case j == to:
			aux[k] = arr[i]
			i++
		case cmp(arr[j], arr[i]) < 0:
			aux[k] = arr[j]
			j++
		default:
			aux[k] = arr[i]
			i++
		}
	}
	copy(arr[from:to], aux[from:to])
}
