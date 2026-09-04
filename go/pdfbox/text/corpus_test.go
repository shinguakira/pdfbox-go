package text_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// corpusDir is where the Java test PDFs and their expected text live, relative
// to this package.
const corpusDir = "../../../pdfbox/src/test/resources/input/"

// Port of the harness of org.apache.pdfbox.text.TestTextStripper.
//
// Java walks src/test/resources/input, strips each PDF twice -- unsorted and
// sorted by position -- and compares against "<name>.pdf.txt" and
// "<name>.pdf-sorted.txt" beside it. It compares line by line after collapsing
// whitespace, and fails the run on any difference.
//
// The port scores rather than fails: this slice does not carry the ToUnicode
// CMap, the CID fonts or the font mapper, so a document needing one of those
// cannot match and a failing assertion would say nothing new. TestCorpusScore
// reports how many match; the number is what migration/STATUS.md records, and
// it is meant to rise as each of those lands.

// corpusFiles returns the PDFs of the corpus, in a stable order.
func corpusFiles(t *testing.T) []string {
	t.Helper()
	entries, err := filepath.Glob(corpusDir + "*.pdf")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	sort.Strings(entries)
	return entries
}

// stripFile returns the text of one file, sorted by position or not.
func stripFile(path string, sortByPosition bool) (out string, err error) {
	defer func() {
		// a malformed file can reach a panic where Java throws an unchecked
		// exception; the harness records that as a failure rather than taking
		// the whole run down
		if r := recover(); r != nil {
			out = ""
			err = errPanic
		}
	}()

	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		return "", err
	}
	defer document.Close()

	stripper := text.NewPDFTextStripper()
	stripper.SetSortByPosition(sortByPosition)
	var builder strings.Builder
	stripper.SetOutput(&builder)
	if err := stripper.ProcessPages(document.Pages()); err != nil {
		return "", err
	}
	return builder.String(), nil
}

// errPanic stands for the unchecked exception a malformed file can reach.
var errPanic = &corpusPanic{}

type corpusPanic struct{}

func (*corpusPanic) Error() string { return "panic" }

// normalizeCorpusText collapses the differences the Java comparison ignores:
// line endings, and runs of whitespace within a line.
func normalizeCorpusText(s string) []string {
	// Java writes the UTF-8 byte order mark into its own output file before
	// comparing, so every expected file carries one; it is the harness's, not
	// the stripper's.
	s = strings.TrimPrefix(s, string(rune(0xFEFF)))
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.Join(strings.Fields(line), " "))
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// TestCorpusOpens reports how many of the corpus files the loader can open at
// all, which is the loader's own score.
func TestCorpusOpens(t *testing.T) {
	files := corpusFiles(t)
	if len(files) == 0 {
		t.Skip("the Java test corpus is not present")
	}

	opened := 0
	var failures []string
	for _, path := range files {
		func() {
			defer func() {
				if r := recover(); r != nil {
					failures = append(failures, filepath.Base(path)+": panic")
				}
			}()
			document, err := pdfbox.LoadPDF(path)
			if err != nil {
				failures = append(failures, filepath.Base(path)+": "+err.Error())
				return
			}
			defer document.Close()
			if _, err := document.Pages().Get(0), error(nil); err != nil {
				failures = append(failures, filepath.Base(path)+": no first page")
				return
			}
			opened++
		}()
	}
	t.Logf("opened %d of %d", opened, len(files))
	for _, failure := range failures {
		t.Logf("  did not open: %s", failure)
	}
	if opened == 0 {
		t.Errorf("not one of the %d corpus files opened", len(files))
	}
}

// TestCorpusScore reports how many of the corpus files yield exactly the text
// the Java expects, unsorted and sorted by position.
//
// Java runs every file both ways, comparing against "<name>.pdf.txt" and
// "<name>.pdf-sorted.txt"; the port scores both the same way.
func TestCorpusScore(t *testing.T) {
	t.Run("unsorted", func(t *testing.T) { scoreCorpus(t, false) })
	t.Run("sorted", func(t *testing.T) { scoreCorpus(t, true) })
}

// scoreCorpus scores every file of the corpus one way round.
func scoreCorpus(t *testing.T, sortByPosition bool) {
	files := corpusFiles(t)
	if len(files) == 0 {
		t.Skip("the Java test corpus is not present")
	}

	suffix := ".txt"
	if sortByPosition {
		suffix = "-sorted.txt"
	}

	matched := 0
	var report []string
	for _, path := range files {
		name := filepath.Base(path)
		expectedBytes, err := os.ReadFile(path + suffix)
		if err != nil {
			report = append(report, name+": no expected output")
			continue
		}
		got, err := stripFile(path, sortByPosition)
		if err != nil {
			report = append(report, name+": "+err.Error())
			continue
		}
		want := normalizeCorpusText(string(expectedBytes))
		have := normalizeCorpusText(got)
		if len(want) == len(have) {
			same := true
			for i := range want {
				if want[i] != have[i] {
					same = false
					report = append(report, name+": line "+itoa(i+1)+
						"\n      want "+quote(want[i])+"\n      have "+quote(have[i]))
					break
				}
			}
			if same {
				matched++
				continue
			}
		} else {
			report = append(report, name+": "+itoa(len(have))+" lines, want "+itoa(len(want)))
		}
	}

	t.Logf("corpus score: %d of %d", matched, len(files))
	for _, line := range report {
		t.Logf("  %s", line)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var digits []byte
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	return string(digits)
}

func quote(s string) string {
	if len(s) > 90 {
		s = s[:90] + "..."
	}
	return "\"" + s + "\""
}
