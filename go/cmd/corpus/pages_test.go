package main

// The page table, and what comparing two of them reports.
//
// This is migration tooling rather than a port, so there is no Java test to
// carry across. What these cases hold is the shape both drivers write and the
// four answers the comparison can give, because a comparison that quietly joins
// nothing would report "0 disagree" for two tables of different files and read
// as good news.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTable puts a page table in a temporary file and answers its path.
func writeTable(t *testing.T, name string, rows ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	body := pageHeader + "\n" + strings.Join(rows, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestComparePagesReportsThePageThatDiffers(t *testing.T) {
	java := writeTable(t, "java.tsv",
		"a.pdf\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"a.pdf\t2\tok\t20\tbbbbbbbbbbbbbbbb",
		"a.pdf\t3\tok\t30\tcccccccccccccccc",
	)
	mine := writeTable(t, "go.tsv",
		"a.pdf\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"a.pdf\t2\tok\t20\tdddddddddddddddd",
		"a.pdf\t3\tok\t30\tcccccccccccccccc",
	)

	differ, err := comparePages(java, mine)
	if err != nil {
		t.Fatal(err)
	}
	if differ != 1 {
		t.Errorf("comparePages = %d, want 1: page 2 has the same length and not the same text", differ)
	}
}

func TestComparePagesTellsTheFourAnswersApart(t *testing.T) {
	java := writeTable(t, "java.tsv",
		"same.pdf\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"length.pdf\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"content.pdf\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"failed.pdf\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"only-java.pdf\t1\tok\t10\taaaaaaaaaaaaaaaa",
	)
	mine := writeTable(t, "go.tsv",
		"same.pdf\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"length.pdf\t1\tok\t11\taaaaaaaaaaaaaaaa",
		"content.pdf\t1\tok\t10\tbbbbbbbbbbbbbbbb",
		"failed.pdf\t1\tpanic: nil map\t0\t-",
		"only-go.pdf\t1\tok\t10\taaaaaaaaaaaaaaaa",
	)

	differ, err := comparePages(java, mine)
	if err != nil {
		t.Fatal(err)
	}
	if differ != 3 {
		t.Errorf("comparePages = %d, want 3: the length, the content and the failure", differ)
	}
}

func TestReadPagesKeepsTheLabelWithTheFile(t *testing.T) {
	// A file a passwords table opens more than one way gets a row per way, and
	// the way is part of the file cell on both sides. Two rows of one file and
	// one page must not collide.
	table := writeTable(t, "pages.tsv",
		"e.pdf [secret]\t1\tok\t10\taaaaaaaaaaaaaaaa",
		"e.pdf [owner | cert.pem | key.pem]\t1\tok\t12\tbbbbbbbbbbbbbbbb",
	)
	rows, err := readPages(table)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("readPages read %d rows, want 2: the label tells the two ways of opening one file apart", len(rows))
	}
}
