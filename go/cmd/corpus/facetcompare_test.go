package main

// What -comparefacets counts when the two tables do not have the same facets.
//
// Migration tooling, like oracle_test.go. A facet one table has and the other
// does not cannot be compared, and a comparison that only printed that and
// then counted nothing would pass two tables written by different versions of
// the drivers.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFacetTable puts a facet table in a temporary file and answers its path.
func writeFacetTable(t *testing.T, name string, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCompareFacetsCountsAFacetOneTableHasNot(t *testing.T) {
	// The rows agree on every facet both tables have; PDFBox's also has xmp.
	java := writeFacetTable(t, "java.tsv",
		"file\topen\tinfo\txmp",
		"go/testdata/corpus/s/a.pdf\tok\taaaa\tbbbb",
	)
	mine := writeFacetTable(t, "go.tsv",
		"file\topen\tinfo",
		"go/testdata/corpus/s/a.pdf\tok\taaaa",
	)
	differ, err := compareFacets(java, mine)
	if err != nil {
		t.Fatal(err)
	}
	if differ != 1 {
		t.Errorf("compareFacets = %d, want 1: xmp is in PDFBox's table only", differ)
	}

	// And the other way round.
	differ, err = compareFacets(mine, java)
	if err != nil {
		t.Fatal(err)
	}
	if differ != 1 {
		t.Errorf("compareFacets = %d, want 1: xmp is in this program's table only", differ)
	}
}

func TestCompareFacetsIsCleanWhenTheTablesAgree(t *testing.T) {
	java := writeFacetTable(t, "java.tsv",
		"file\topen\tinfo\txmp",
		"go/testdata/corpus/s/a.pdf\tok\taaaa\tbbbb",
	)
	mine := writeFacetTable(t, "go.tsv",
		"file\topen\tinfo\txmp",
		"go/testdata/corpus/s/a.pdf\tok\taaaa\tbbbb",
	)
	differ, err := compareFacets(java, mine)
	if err != nil {
		t.Fatal(err)
	}
	if differ != 0 {
		t.Errorf("compareFacets = %d, want 0", differ)
	}
}
