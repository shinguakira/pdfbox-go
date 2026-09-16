package main

// A digest per page, and the comparison of two such tables.
//
// The document digest says that two extractions of the same length differ; it
// does not say where. Both findings it has made so far were located by dumping
// each side's whole text and diffing it by hand. A page table is that, done by
// the tool: one row per page per way of opening a file, with the page's text
// digested the same way the document's is. JavaCorpus writes the same table for
// PDFBox, and -comparepages joins the two.
//
// It is a mode of its own rather than a column of the main table, because a
// page table over the whole corpus is 19,000 rows of documents and a quarter of
// a million rows of pages, and because extracting per page costs a pass per
// page. The way it is meant to be used is: score the corpus, see which files
// disagree, then run this over those files alone.

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// pageHeader names the columns of a page table, and is also what -comparepages
// parses.
const pageHeader = "file\tpage\ttext\tchars\tdigest"

// pageRow is one page of one way of opening one file.
type pageRow struct {
	path   string
	label  string
	page   int    // 1-based, as PDFBox's setStartPage is
	text   string // ok, or the first thing that went wrong
	chars  int
	digest string
}

func (p pageRow) String() string {
	return fmt.Sprintf("%s%s\t%d\t%s\t%d\t%s", p.path, p.label, p.page, p.text, p.chars, p.digest)
}

// writePages extracts every page of every job and writes the table.
//
// A panic is recovered per page and written to that page's row, as the scoring
// path writes one to the stage that was running. This mode does not isolate
// each file in a child process: it is meant for the few files a comparison has
// already named, and the panics that cannot be recovered -- Go's stack overflow
// -- are what -isolate exists for in the scoring path.
func writePages(jobs []job, out string) error {
	file := os.Stdout
	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		file = f
	}
	if _, err := fmt.Fprintln(file, pageHeader); err != nil {
		return err
	}
	for i, j := range jobs {
		fmt.Fprintf(os.Stderr, "\r%d/%d %-60.60s", i+1, len(jobs), j.path)
		for _, row := range pagesOf(j) {
			if _, err := fmt.Fprintln(file, row); err != nil {
				return err
			}
		}
	}
	fmt.Fprintf(os.Stderr, "\r%-72s\r", "")
	return nil
}

// pagesOf extracts each page of one way of opening one file.
func pagesOf(j job) []pageRow {
	path := rootRelative(j.path)
	document, err := openDocument(j)
	if err != nil {
		return []pageRow{{path: path, label: j.label, page: 0, text: short(err), digest: "-"}}
	}
	defer document.Close()

	pages := document.NumberOfPages()
	rows := make([]pageRow, 0, pages)
	for page := 1; page <= pages; page++ {
		rows = append(rows, pageOf(document, path, j.label, page))
	}
	return rows
}

// pageOf extracts one page, the way PDFBox's own single-page idiom does: a
// stripper of its own, with the first and last page set to it.
func pageOf(document *pdmodel.PDDocument, path, label string, page int) (row pageRow) {
	row = pageRow{path: path, label: label, page: page, text: "ok", digest: "-"}
	defer func() {
		if p := recover(); p != nil {
			row.text = short(fmt.Errorf("panic: %v", p))
			row.chars, row.digest = 0, "-"
		}
	}()

	stripper := text.NewPDFTextStripper()
	stripper.SetStartPage(page)
	stripper.SetEndPage(page)
	var builder strings.Builder
	if err := stripper.WriteText(document, &builder); err != nil {
		row.text = short(err)
		return row
	}
	row.chars = len([]rune(builder.String()))
	row.digest = digest(builder.String())
	return row
}

// comparePages joins two page tables and reports every page the two disagree
// on. The first is PDFBox's, the second the port's.
func comparePages(javaPath, goPath string) (int, error) {
	java, err := readPages(javaPath)
	if err != nil {
		return 0, err
	}
	mine, err := readPages(goPath)
	if err != nil {
		return 0, err
	}

	keys := make([]string, 0, len(mine))
	for key := range mine {
		if _, known := java[key]; known {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var compared, differ int
	for _, key := range keys {
		them, now := java[key], mine[key]
		compared++
		switch {
		case them.text != now.text:
			differ++
			fmt.Printf("  TEXT   %s page %d\n           go:   %s\n           java: %s\n",
				now.path+now.label, now.page, now.text, them.text)
		case them.chars != now.chars:
			differ++
			fmt.Printf("  CHARS  %s page %d: go %d, java %d\n",
				now.path+now.label, now.page, now.chars, them.chars)
		case them.digest != now.digest && them.digest != "" && now.digest != "":
			differ++
			fmt.Printf("  DIGEST %s page %d: %d characters on both sides, and not the same ones\n",
				now.path+now.label, now.page, now.chars)
		}
	}

	fmt.Printf("\n%d pages compared, %d disagree\n", compared, differ)
	onlyOne := len(mine) - compared
	if onlyOne > 0 {
		fmt.Printf("%d pages the other table does not have\n", onlyOne)
	}
	return differ, nil
}

// readPages reads a page table, keyed by the file, the label and the page.
func readPages(path string) (map[string]pageRow, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rows := map[string]pageRow{}
	for i, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 5 {
			continue
		}
		// The first cell is the path and the label together, which is what both
		// drivers write and what makes a row of one table the row of the other.
		row := pageRow{path: fields[0], text: fields[2], digest: fields[4]}
		row.page, _ = strconv.Atoi(fields[1])
		row.chars, _ = strconv.Atoi(fields[3])
		rows[fmt.Sprintf("%s%s\t%d", row.path, row.label, row.page)] = row
	}
	return rows, nil
}
