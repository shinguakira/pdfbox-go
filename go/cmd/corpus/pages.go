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
//
// Both tables are walked, not one of them. A file one side opened and the
// other did not, a page one table has and the other does not, and a file only
// one table holds are each a disagreement: joining on the pages both tables
// hold would compare nothing for a document the port cannot open, and report
// that as agreement.
func comparePages(javaPath, goPath string) (int, error) {
	java, err := readPages(javaPath)
	if err != nil {
		return 0, err
	}
	mine, err := readPages(goPath)
	if err != nil {
		return 0, err
	}
	javaFiles, goFiles := byFile(java), byFile(mine)

	names := make([]string, 0, len(javaFiles)+len(goFiles))
	for name := range javaFiles {
		names = append(names, name)
	}
	for name := range goFiles {
		if _, known := javaFiles[name]; !known {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var compared, differ int
	for _, name := range names {
		them, now := javaFiles[name], goFiles[name]
		switch {
		case len(them) == 0:
			differ++
			fmt.Printf("  MISSING %s: in the Go table only\n", name)
			continue
		case len(now) == 0:
			differ++
			fmt.Printf("  MISSING %s: in the PDFBox table only\n", name)
			continue
		}

		// Both drivers write a file they could not open as one row for page 0,
		// holding the error.
		javaError, javaFailed := them[0]
		goError, goFailed := now[0]
		switch {
		case javaFailed && goFailed:
			// Neither side opened it: nothing to compare, and nothing disagrees.
			continue
		case goFailed:
			differ++
			fmt.Printf("  OPEN   behind %s\n           go:   %s\n           java: ok, %d pages\n",
				name, goError.text, len(them))
			continue
		case javaFailed:
			differ++
			fmt.Printf("  OPEN   ahead  %s\n           go:   ok, %d pages\n           java: %s\n",
				name, len(now), javaError.text)
			continue
		}

		pages := make([]int, 0, len(them)+len(now))
		for page := range them {
			pages = append(pages, page)
		}
		for page := range now {
			if _, known := them[page]; !known {
				pages = append(pages, page)
			}
		}
		sort.Ints(pages)

		for _, page := range pages {
			j, inJava := them[page]
			g, inGo := now[page]
			switch {
			case !inGo:
				differ++
				fmt.Printf("  PAGE   %s page %d: in the PDFBox table only\n", name, page)
				continue
			case !inJava:
				differ++
				fmt.Printf("  PAGE   %s page %d: in the Go table only\n", name, page)
				continue
			}
			compared++
			switch {
			case j.text != g.text:
				differ++
				fmt.Printf("  TEXT   %s page %d\n           go:   %s\n           java: %s\n",
					name, page, g.text, j.text)
			case j.chars != g.chars:
				differ++
				fmt.Printf("  CHARS  %s page %d: go %d, java %d\n", name, page, g.chars, j.chars)
			case j.digest != g.digest && j.digest != "" && g.digest != "":
				differ++
				fmt.Printf("  DIGEST %s page %d: %d characters on both sides, and not the same ones\n",
					name, page, g.chars)
			}
		}
	}

	fmt.Printf("\n%d pages compared, %d disagree\n", compared, differ)
	return differ, nil
}

// byFile groups a page table's rows by the file and label they belong to, and
// within that by page.
func byFile(rows map[string]pageRow) map[string]map[int]pageRow {
	files := map[string]map[int]pageRow{}
	for _, row := range rows {
		name := row.path + row.label
		if files[name] == nil {
			files[name] = map[int]pageRow{}
		}
		files[name][row.page] = row
	}
	return files
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
