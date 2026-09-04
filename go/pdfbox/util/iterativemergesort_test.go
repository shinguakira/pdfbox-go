package util

import (
	"math/rand"
	"slices"
	"testing"
)

// Port of org.apache.pdfbox.util.TestSort.
//
// Java seeds its Random with 12345 and runs 100 rounds of up to 20002 values;
// the two generators produce different numbers, so the port keeps the shape of
// the round -- the length, the range that forces duplicates, and the comparison
// against a separately sorted copy -- rather than the exact values.

func doSortTest(t *testing.T, input, expected []int) {
	t.Helper()
	list := make([]int, len(input))
	copy(list, input)
	IterativeMergeSort(list, func(a, b int) int { return a - b })
	if !slices.Equal(list, expected) {
		t.Errorf("sorting %v gave %v, want %v", input, list, expected)
	}
}

func TestSort(t *testing.T) {
	doSortTest(t, []int{9, 8, 7, 6, 5, 4, 3, 2, 1}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})
	doSortTest(t, []int{4, 3, 2, 1, 9, 8, 7, 6, 5}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})
	doSortTest(t, []int{}, []int{})
	doSortTest(t, []int{5}, []int{5})
	doSortTest(t, []int{5, 6}, []int{5, 6})
	doSortTest(t, []int{6, 5}, []int{5, 6})

	rnd := rand.New(rand.NewSource(12345))
	for cnt := 0; cnt < 100; cnt++ {
		length := rnd.Intn(20000) + 2
		input := make([]int, length)
		for i := range input {
			// choose values so that there are some duplicates
			input[i] = rnd.Intn(rnd.Intn(100) + 1)
		}
		expected := make([]int, length)
		copy(expected, input)
		slices.Sort(expected)
		doSortTest(t, input, expected)
	}
}

// TestSortIsStable checks that two entries that compare equal keep the order
// they were given in, which is what the text stripper depends on.
func TestSortIsStable(t *testing.T) {
	type entry struct {
		key   int
		order int
	}
	list := []entry{{1, 0}, {0, 1}, {1, 2}, {0, 3}, {1, 4}, {0, 5}}
	IterativeMergeSort(list, func(a, b entry) int { return a.key - b.key })

	want := []entry{{0, 1}, {0, 3}, {0, 5}, {1, 0}, {1, 2}, {1, 4}}
	if !slices.Equal(list, want) {
		t.Errorf("sort gave %v, want %v", list, want)
	}
}

// TestSortSurvivesAnIntransitiveComparison checks the reason PDFTextStripper
// reaches for this sort at all: a comparison that is not a total order makes
// the JDK's sort throw, and this one simply finishes.
func TestSortSurvivesAnIntransitiveComparison(t *testing.T) {
	// a < b, b < c, c < a
	cmp := func(a, b int) int {
		switch {
		case a == b:
			return 0
		case (a+1)%3 == b:
			return -1
		default:
			return 1
		}
	}
	list := []int{0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2}
	before := len(list)
	IterativeMergeSort(list, cmp)
	if len(list) != before {
		t.Errorf("the sort changed the length to %d, want %d", len(list), before)
	}
	counts := map[int]int{}
	for _, v := range list {
		counts[v]++
	}
	if counts[0] != 7 || counts[1] != 7 || counts[2] != 7 {
		t.Errorf("the sort lost or duplicated entries: %v", counts)
	}
}
