package main

// What -oracle counts when the two tables do not hold the same rows.
//
// Migration tooling, like pages_test.go. The case that matters is the one the
// comparison could not see when it walked this run's rows alone: a file PDFBox
// answered for, under the directories this run was given, that the run has no
// row for.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeOracle puts a PDFBox table in a temporary file and answers its path.
func writeOracle(t *testing.T, rows ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "java.tsv")
	body := "file\topen\tpages\ttext\tchars\tdigest\n" + strings.Join(rows, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// agreeing is this run's row for a file PDFBox read as one page of ten characters.
func agreeing(path, label string) result {
	return result{path: path, label: label, open: "ok", pages: 1, text: "ok", chars: 10, render: "-", digest: "aaaaaaaaaaaaaaaa"}
}

func TestCompareOracleCountsARowThisRunHasNone(t *testing.T) {
	table := writeOracle(t,
		"go/testdata/corpus/s/a.pdf\tok\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"go/testdata/corpus/s/b.pdf\tok\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"go/testdata/corpus/other/c.pdf\tok\t1\tok\t10\taaaaaaaaaaaaaaaa",
	)
	results := []result{agreeing("testdata/corpus/s/a.pdf", "")}

	behind, err := compareOracle(table, []string{"testdata/corpus/s"}, results)
	if err != nil {
		t.Fatal(err)
	}
	if behind != 1 {
		t.Errorf("compareOracle = %d, want 1: b.pdf is under the directory this run was given and has no row; c.pdf is not under it", behind)
	}
}

func TestCompareOracleCountsAWayOfOpeningThisRunHasNone(t *testing.T) {
	// A passwords table opens the file two ways on PDFBox's side, and this run
	// has one of them.
	table := writeOracle(t,
		"go/testdata/corpus/s/e.pdf [user]\tok\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"go/testdata/corpus/s/e.pdf [owner]\tok\t1\tok\t10\taaaaaaaaaaaaaaaa",
	)
	results := []result{agreeing("testdata/corpus/s/e.pdf", " [user]")}

	behind, err := compareOracle(table, []string{"./testdata/corpus/s"}, results)
	if err != nil {
		t.Fatal(err)
	}
	if behind != 1 {
		t.Errorf("compareOracle = %d, want 1: the owner row", behind)
	}
}

func TestCompareOracleIsCleanWhenEveryRowIsAnswered(t *testing.T) {
	// The table is wider than the run, which is how it is used: the whole
	// corpus's table, one suite at a time. The directory is given with a "./"
	// and the downloads through "../", as they are from go/.
	table := writeOracle(t,
		"go/testdata/corpus/s/a.pdf\tok\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"go/testdata/corpus/t/b.pdf\tok\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"pdfbox/target/pdfs/PDFBOX-1.pdf\tok\t1\tok\t10\taaaaaaaaaaaaaaaa",
	)
	results := []result{
		agreeing("testdata/corpus/s/a.pdf", ""),
		agreeing("../pdfbox/target/pdfs/PDFBOX-1.pdf", ""),
	}

	behind, err := compareOracle(table, []string{"./testdata/corpus/s", "../pdfbox/target/pdfs"}, results)
	if err != nil {
		t.Fatal(err)
	}
	if behind != 0 {
		t.Errorf("compareOracle = %d, want 0: t/b.pdf is outside the directories given", behind)
	}
}
