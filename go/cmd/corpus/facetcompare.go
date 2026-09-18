package main

// The comparison of two facet tables, PDFBox's and this program's.
//
// A facet table has a row per way of opening a file: whether it opened, then
// one cell per facet -- "<lines>:<digest>", "-" where the document has no such
// thing, or "error" where computing it failed. Both tables are walked, as
// -comparepages and -oracle walk theirs: a row only one table holds is a
// disagreement, so a document the port never answered for cannot pass as one
// that agrees.

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// facetTable is a facet table read back: its facet names in column order, and
// its rows by file.
type facetTable struct {
	names []string
	rows  map[string][]string
}

func readFacets(path string) (facetTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return facetTable{}, err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return facetTable{}, fmt.Errorf("%s: empty", path)
	}
	header := strings.Split(lines[0], "\t")
	if len(header) < 2 || header[0] != "file" || header[1] != "open" {
		return facetTable{}, fmt.Errorf("%s: not a facet table", path)
	}
	table := facetTable{names: header[2:], rows: map[string][]string{}}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cells := strings.Split(line, "\t")
		if len(cells) < 2 {
			continue
		}
		table.rows[cells[0]] = cells[1:]
	}
	return table, nil
}

// compareFacets reports every facet the two tables disagree on, PDFBox's table
// first, and answers how many disagreements it found.
func compareFacets(javaPath, goPath string) (int, error) {
	java, err := readFacets(javaPath)
	if err != nil {
		return 0, err
	}
	mine, err := readFacets(goPath)
	if err != nil {
		return 0, err
	}

	// A facet one table has and the other does not cannot be compared, and
	// saying nothing about it would read as agreement.
	column := map[string]int{}
	for i, name := range mine.names {
		column[name] = i
	}
	var shared []string
	for _, name := range java.names {
		if _, ok := column[name]; ok {
			shared = append(shared, name)
		} else {
			fmt.Printf("  FACET  %s: in PDFBox's table only, not compared\n", name)
		}
	}
	javaColumn := map[string]int{}
	for i, name := range java.names {
		javaColumn[name] = i
	}
	for _, name := range mine.names {
		if _, ok := javaColumn[name]; !ok {
			fmt.Printf("  FACET  %s: in this program's table only, not compared\n", name)
		}
	}

	keys := make([]string, 0, len(java.rows)+len(mine.rows))
	for key := range java.rows {
		keys = append(keys, key)
	}
	for key := range mine.rows {
		if _, known := java.rows[key]; !known {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	type tally struct{ same, differ int }
	tallies := map[string]*tally{}
	for _, name := range shared {
		tallies[name] = &tally{}
	}
	var rows, missing, openBoth, openNeither, openBehind, openAhead, differ int
	for _, key := range keys {
		them, inJava := java.rows[key]
		now, inGo := mine.rows[key]
		switch {
		case !inGo:
			missing++
			fmt.Printf("  MISSING %s: in PDFBox's table only\n", key)
			continue
		case !inJava:
			missing++
			fmt.Printf("  MISSING %s: in this program's table only\n", key)
			continue
		}
		rows++
		javaOpened, goOpened := them[0] == "ok", now[0] == "ok"
		switch {
		case javaOpened && goOpened:
			openBoth++
		case !javaOpened && !goOpened:
			openNeither++
			continue
		case javaOpened:
			openBehind++
			fmt.Printf("  OPEN   behind %s\n           go:   %s\n           java: ok\n", key, now[0])
			continue
		default:
			openAhead++
			fmt.Printf("  OPEN   ahead  %s\n           go:   ok\n           java: %s\n", key, them[0])
			continue
		}
		for _, name := range shared {
			j, g := cellAt(them, javaColumn[name]), cellAt(now, column[name])
			if j == g {
				tallies[name].same++
				continue
			}
			tallies[name].differ++
			differ++
			fmt.Printf("  %-11s %s: go %s, java %s\n", strings.ToUpper(name), key, g, j)
		}
	}

	fmt.Printf("\n%d rows compared, %d in one table only\n", rows, missing)
	fmt.Printf("  open        both %d, neither %d, behind %d, ahead %d\n", openBoth, openNeither, openBehind, openAhead)
	for _, name := range shared {
		fmt.Printf("  %-11s %d the same, %d not\n", name, tallies[name].same, tallies[name].differ)
	}
	return differ + missing + openBehind + openAhead, nil
}

// cellAt answers a row's cell for a facet, the open cell being the row's first.
func cellAt(row []string, facet int) string {
	if facet+1 < len(row) {
		return row[facet+1]
	}
	return ""
}
