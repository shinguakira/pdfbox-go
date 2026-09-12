package text_test

// Defects the corpus found. See migration/TESTDATA.md for what the corpus is
// and how it is scored; each test here names the file that reached the fault,
// and each one fails without its fix.

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// TestHandleDirectionTakesAWordWithNoRuns pins a defect the port introduced by
// substituting the bidi implementation, not one the Java has.
//
// Java's handleDirection hands the word to java.text.Bidi, for which none of
// these inputs is special: the paragraph is not mixed and its base level is
// DIRECTION_LEFT_TO_RIGHT, so the first arm returns the word untouched. The
// port uses golang.org/x/text/unicode/bidi instead -- the JDK class has no
// counterpart in the standard library -- and that one resolves a word made only
// of paragraph separators to zero runs, then indexes the first of them when
// asked the direction. Asking panics with an index out of range.
//
// Three files of the corpus reach it through normalizeWord, two of them from
// the set the Java build downloads and therefore in front of the Java's own
// tests:
//
//	pdfbox/target/pdfs/PDFBOX-4418-000314.pdf    an empty word
//	pdfbox/target/pdfs/PDFBOX-4418-000671.pdf    one carriage return
//	verapdf PDF_A-4 6.2.10.9 "Use of .notdef glyph" t01-fail-b
//
// The expected value is Java's, read off the first arm of the Java method
// rather than off this package: the word comes back as it went in.
func TestHandleDirectionTakesAWordWithNoRuns(t *testing.T) {
	// LINE FEED, CARRIAGE RETURN, INFORMATION SEPARATOR FOUR, THREE and TWO,
	// and NEXT LINE: every code point of Unicode bidi class B, the paragraph
	// separators. Written as numbers because a control character in a string
	// literal cannot be read. These and the empty string are the inputs x/text
	// resolves to no runs.
	separators := []rune{0x000A, 0x000D, 0x001C, 0x001D, 0x001E, 0x0085}

	words := []string{""}
	for _, separator := range separators {
		words = append(words, string(separator))
	}

	for _, word := range words {
		if got := text.HandleDirectionForTest(word); got != word {
			t.Errorf("handleDirection(%q) = %q, want %q", word, got, word)
		}
	}
}

// TestProcessPagesWalksTheTreeTheWayJavaDoes pins the second defect the corpus
// found: qpdf/deep-pages.pdf, a page tree that contains itself.
//
// PDFBox reads that file. It prints "ERROR PDPageTree This page tree node has
// already been visited" and carries on, ending with one page and no text. The
// port panicked with "possible recursion found when searching for page 1".
//
// Both sides carry both of PDPageTree's guards, and the two differ on purpose.
// The indexed accessor -- Java's get(int, COSDictionary, int) -- throws
// IllegalStateException on a repeated node. The iterator's enqueueKids logs that
// error and skips the kid, and the comment on it cites PDFBOX-5009 and
// PDFBOX-3953. What the port had wrong was which one this method reaches: Java's
// processPages is `for (PDPage page : pages)` and the port walked `pages.Get(i)`
// over `pages.Count()`, so a tree Java skips past went through the throwing
// guard instead.
//
// The tree below is the smallest one that tells them apart. Its intermediate
// node lists itself before the page, so the iterator skips the repeat and yields
// the one page, and the indexed walk re-enters the node it is already inside.
func TestProcessPagesWalksTheTreeTheWayJavaDoes(t *testing.T) {
	page := cos.NewDictionary()
	page.SetItem(cos.Type, cos.Page)

	node := cos.NewDictionary()
	node.SetItem(cos.Type, cos.Pages)
	node.SetInt(cos.Count, 1)
	kids := cos.NewArray()
	kids.Add(node) // itself, first, so the indexed walk re-enters it
	kids.Add(page)
	node.SetItem(cos.Kids, kids)

	root := cos.NewDictionary()
	root.SetItem(cos.Type, cos.Pages)
	root.SetInt(cos.Count, 1)
	rootKids := cos.NewArray()
	rootKids.Add(node)
	root.SetItem(cos.Kids, rootKids)

	tree := pdmodel.NewPDPageTreeOf(root)

	// what Java's iterator answers, and therefore what processPages sees
	var walked int
	for range tree.All {
		walked++
	}
	if walked != 1 {
		t.Fatalf("the iterator walked %d pages, want 1 -- the fixture is wrong, not the code", walked)
	}

	stripper := text.NewPDFTextStripper()
	var out strings.Builder
	stripper.SetOutput(&out)
	if err := stripper.ProcessPages(tree); err != nil {
		t.Fatalf("ProcessPages: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "" {
		t.Errorf("text = %q, want empty -- the page has no contents", got)
	}
}
