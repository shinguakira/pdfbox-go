// Command corpus scores the port against a directory of PDFs.
//
// This is not a port of anything. PDFBox has no such tool; it is migration
// tooling, and it exists because the corpora worth checking against are far too
// large to turn into assertions by hand. migration/scripts/fetch-corpus.ps1
// brings down about three thousand files, and the question they answer is not
// "is this byte right" but "which of these can the port open, read and draw at
// all, and did that change since the last run".
//
// What it produces is a table: one row per file, one column per stage, each cell
// ok or the first thing that went wrong. The table is then read two ways.
//
// -baseline compares it against an earlier run of itself, which makes it a
// regression check: a file that used to open and no longer does is a defect the
// unit tests did not catch, and a file that used to fail and now opens is a
// slice landing.
//
// -oracle compares it against the same table produced by PDFBox, which makes it
// the thing migration/README.md has always asked for and never had at this
// scale -- "PDFBox is the oracle" run over three thousand files instead of one.
// migration/scripts/run-oracle.ps1 produces that table; it needs a JDK and no
// Maven. Exit status is non-zero when the port is behind the Java.
//
// Usage:
//
//	go run ./cmd/corpus testdata/corpus/verapdf > verapdf.tsv
//	go run ./cmd/corpus -render testdata/corpus/safedocs-targeted
//	go run ./cmd/corpus -baseline verapdf.tsv testdata/corpus/verapdf
//	go run ./cmd/corpus -oracle ../go/testdata/oracle/java-corpus.tsv testdata/corpus
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// result is one file's row. Each stage holds "ok", "-" for a stage that was
// not reached or not asked for, or a short description of what went wrong.
type result struct {
	path   string
	open   string
	pages  int
	text   string
	chars  int
	render string
}

// header names the columns, and is also what -baseline parses.
const header = "file\topen\tpages\ttext\tchars\trender"

func (r result) String() string {
	return fmt.Sprintf("%s\t%s\t%d\t%s\t%d\t%s",
		r.path, r.open, r.pages, r.text, r.chars, r.render)
}

func main() {
	var (
		render   = flag.Bool("render", false, "also rasterise the first page")
		dpi      = flag.Float64("dpi", 72, "DPI for -render")
		timeout  = flag.Duration("timeout", 30*time.Second, "give up on one file after this; 0 disables")
		out      = flag.String("o", "", "write the table here instead of stdout")
		baseline = flag.String("baseline", "", "compare against a table from a previous run and report only the changes")
		oracle   = flag.String("oracle", "", "compare against a table PDFBox produced (migration/scripts/run-oracle.ps1) and report where the port and the Java disagree")
		quiet    = flag.Bool("q", false, "summary only")
		isolate  = flag.Bool("isolate", true, "score each file in its own process, so one that takes the runtime down does not end the run")
		one      = flag.Bool("one", false, "score exactly one file and print its row; how -isolate re-enters this program")
	)
	flag.Parse()

	if *one {
		if flag.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "corpus: -one takes exactly one file")
			os.Exit(2)
		}
		fmt.Println(scoreNow(flag.Arg(0), *render, float32(*dpi)))
		return
	}

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: corpus [flags] <directory>...")
		flag.PrintDefaults()
		os.Exit(2)
	}

	files, err := collect(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "corpus:", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "corpus: no PDFs under", strings.Join(flag.Args(), " "))
		os.Exit(1)
	}

	scorer := score
	if *isolate {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintln(os.Stderr, "corpus: -isolate needs this program's own path:", err)
			os.Exit(1)
		}
		scorer = func(path string, withRender bool, dpi float32, timeout time.Duration) result {
			return scoreIsolated(exe, path, withRender, dpi, timeout)
		}
	}

	results := make([]result, 0, len(files))
	started := time.Now()
	for i, path := range files {
		if !*quiet {
			fmt.Fprintf(os.Stderr, "\r%d/%d %-60.60s", i+1, len(files), filepath.Base(path))
		}
		results = append(results, scorer(path, *render, float32(*dpi), *timeout))
	}
	if !*quiet {
		fmt.Fprintf(os.Stderr, "\r%-72s\r", "")
	}

	if err := write(results, *out); err != nil {
		fmt.Fprintln(os.Stderr, "corpus:", err)
		os.Exit(1)
	}

	summarise(results, time.Since(started), *render)

	if *baseline != "" {
		regressed, err := compare(*baseline, results)
		if err != nil {
			fmt.Fprintln(os.Stderr, "corpus:", err)
			os.Exit(1)
		}
		if regressed > 0 {
			os.Exit(1)
		}
	}

	if *oracle != "" {
		behind, err := compareOracle(*oracle, results)
		if err != nil {
			fmt.Fprintln(os.Stderr, "corpus:", err)
			os.Exit(1)
		}
		if behind > 0 {
			os.Exit(1)
		}
	}
}

// collect walks the arguments for PDFs, in a stable order so two runs line up.
func collect(roots []string) ([]string, error) {
	var files []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(path), ".pdf") {
				files = append(files, filepath.ToSlash(path))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

// scoreIsolated scores one file in a child process, which is the only way to
// survive the two things recover cannot catch.
//
// Both showed up on the eighth file of the first corpus this was pointed at.
// safedocs-targeted/ContentStreamCycleType3insideType3.pdf is a Type 3 font
// whose glyph draws itself, and neither PDFBox nor the port bounds that
// recursion -- the level guard both carry is wired into the three DrawObject
// operators and not into showType3Glyph. Java gets a StackOverflowError, which
// is an Error and catchable, and its own SECURITY.md calls that a known
// limitation of reading malformed PDFs. Go gets `fatal error: stack overflow`
// after 3.9 million frames, which no deferred recover runs for and which takes
// the process with it. The other is the hang: a goroutine that will not return
// cannot be stopped from outside, so an in-process timeout leaks it.
//
// A child process has neither problem. It costs a spawn per file, which on
// three thousand files is a couple of minutes, and it is the default because
// every corpus worth pointing this at is a corpus of files chosen to break
// parsers.
func scoreIsolated(exe, path string, withRender bool, dpi float32, timeout time.Duration) result {
	args := []string{"-one"}
	if withRender {
		args = append(args, "-render", "-dpi", fmt.Sprint(dpi))
	}
	args = append(args, path)

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, exe, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if row := strings.TrimSpace(stdout.String()); row != "" {
		if fields := strings.Split(row, "\t"); len(fields) == 6 {
			pages := 0
			chars := 0
			fmt.Sscan(fields[2], &pages)
			fmt.Sscan(fields[4], &chars)
			return result{
				path: path, open: fields[1], pages: pages,
				text: fields[3], chars: chars, render: fields[5],
			}
		}
	}

	if ctx.Err() == context.DeadlineExceeded {
		return result{path: path, open: "timeout", text: "-", render: "-"}
	}
	return result{path: path, open: crashReason(stderr.String(), err), text: "-", render: "-"}
}

// crashReason picks the one line of a dead child's output worth a column.
func crashReason(stderr string, err error) string {
	for _, line := range strings.Split(strings.ReplaceAll(stderr, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "fatal error:") || strings.HasPrefix(line, "panic:") {
			return short(fmt.Errorf("%s", line))
		}
	}
	if err != nil {
		return short(fmt.Errorf("child: %w", err))
	}
	return "child said nothing"
}

// score runs one file through the stages in this process. A malformed document
// can reach a panic where Java throws an unchecked exception -- the corpora are
// full of files chosen for exactly that -- so each stage is fenced.
//
// The timeout runs the work on another goroutine and stops waiting for it. The
// goroutine is not stopped, because Go has no way to; a file that hangs leaks
// one for the rest of the run. That is why -isolate is the default and this is
// the fast path for a corpus already known to be survivable.
func score(path string, withRender bool, dpi float32, timeout time.Duration) result {
	done := make(chan result, 1)
	go func() { done <- scoreNow(path, withRender, dpi) }()

	if timeout <= 0 {
		return <-done
	}
	select {
	case r := <-done:
		return r
	case <-time.After(timeout):
		return result{path: path, open: "timeout", text: "-", render: "-"}
	}
}

func scoreNow(path string, withRender bool, dpi float32) (r result) {
	r = result{path: path, open: "ok", text: "-", render: "-"}
	defer func() {
		if p := recover(); p != nil {
			// A panic is the finding, so it carries its message: Java reaches
			// most of these as a checked IOException, and which panic it is
			// says whether the port has a nil where Java has a null check.
			why := short(fmt.Errorf("panic: %v", p))
			if r.open != "ok" {
				return
			}
			// whichever stage was in flight
			switch {
			case r.text == "-":
				r.text = why
			case r.render == "-":
				r.render = why
			}
		}
	}()

	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		r.open = short(err)
		if strings.Contains(r.open, "password is incorrect") {
			// Not a failure of anything. The corpora carry encrypted documents
			// whose passwords live in the Java test that reads them, and this
			// tool has no way to know one. Counted apart from the rest so the
			// open rate is not quietly wrong.
			r.open = "encrypted"
		}
		return r
	}
	defer document.Close()

	r.pages = document.NumberOfPages()

	stripper := text.NewPDFTextStripper()
	var builder strings.Builder
	stripper.SetOutput(&builder)
	if err := stripper.ProcessPages(document.Pages()); err != nil {
		r.text = short(err)
	} else {
		r.text = "ok"
		r.chars = len([]rune(builder.String()))
	}

	if withRender {
		if r.pages == 0 {
			r.render = "no pages"
		} else if _, err := raster.RenderPageWithDPI(document, 0, dpi,
			rendering.RGB, true); err != nil {
			r.render = short(err)
		} else {
			r.render = "ok"
		}
	}
	return r
}

// short reduces an error to something that fits a column and, more to the
// point, stays the same between runs so -baseline can compare two of them.
func short(err error) string {
	s := err.Error()
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	s = strings.Join(strings.Fields(s), " ")
	const max = 70
	if len(s) > max {
		s = s[:max]
	}
	return s
}

func write(results []result, path string) error {
	var w io.Writer = os.Stdout
	if path != "" {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	if _, err := fmt.Fprintln(w, header); err != nil {
		return err
	}
	for _, r := range results {
		if _, err := fmt.Fprintln(w, r); err != nil {
			return err
		}
	}
	return nil
}

func summarise(results []result, elapsed time.Duration, withRender bool) {
	opened, stripped, rendered, encrypted := 0, 0, 0, 0
	reasons := map[string]int{}
	for _, r := range results {
		switch r.open {
		case "ok":
			opened++
		case "encrypted":
			encrypted++
		default:
			reasons[r.open]++
		}
		if r.text == "ok" {
			stripped++
		}
		if r.render == "ok" {
			rendered++
		}
	}

	// An encrypted file the tool has no password for is neither a pass nor a
	// failure, so it leaves the denominator rather than joining the numerator.
	n := len(results) - encrypted
	fmt.Fprintf(os.Stderr, "\n%d files in %s", len(results), elapsed.Round(time.Millisecond))
	if encrypted > 0 {
		fmt.Fprintf(os.Stderr, ", %d of them encrypted and skipped", encrypted)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  open   %d/%d (%.1f%%)\n", opened, n, percent(opened, n))
	fmt.Fprintf(os.Stderr, "  text   %d/%d (%.1f%%)\n", stripped, n, percent(stripped, n))
	if withRender {
		fmt.Fprintf(os.Stderr, "  render %d/%d (%.1f%%)\n", rendered, n, percent(rendered, n))
	}

	if len(reasons) > 0 {
		type reason struct {
			why string
			n   int
		}
		ranked := make([]reason, 0, len(reasons))
		for why, count := range reasons {
			ranked = append(ranked, reason{why, count})
		}
		sort.Slice(ranked, func(i, j int) bool {
			if ranked[i].n != ranked[j].n {
				return ranked[i].n > ranked[j].n
			}
			return ranked[i].why < ranked[j].why
		})
		fmt.Fprintln(os.Stderr, "\nwhy they did not open:")
		for i, r := range ranked {
			if i == 15 {
				fmt.Fprintf(os.Stderr, "  ... and %d more kinds\n", len(ranked)-15)
				break
			}
			fmt.Fprintf(os.Stderr, "  %4d  %s\n", r.n, r.why)
		}
	}
}

func percent(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return 100 * float64(part) / float64(whole)
}

// compare reports the rows that differ from a previous table and returns how
// many of them got worse. A file the baseline does not mention is new and is
// reported but not counted as a regression.
func compare(path string, results []result) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	was := map[string]result{}
	for i, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			continue
		}
		was[fields[0]] = result{
			path: fields[0], open: fields[1], text: fields[3], render: fields[5],
		}
	}

	regressed, improved, added := 0, 0, 0
	var lines []string
	for _, now := range results {
		before, known := was[now.path]
		if !known {
			added++
			continue
		}
		for _, stage := range []struct {
			name     string
			from, to string
		}{
			{"open", before.open, now.open},
			{"text", before.text, now.text},
			{"render", before.render, now.render},
		} {
			// One failure becoming a different failure is neither better nor
			// worse -- the error text moves whenever a message is reworded --
			// so only a crossing of the ok line is reported.
			switch {
			case stage.from == stage.to:
			case stage.from == "ok":
				regressed++
				lines = append(lines, fmt.Sprintf("  WORSE  %-6s %s: ok -> %s", stage.name, now.path, stage.to))
			case stage.to == "ok":
				improved++
				lines = append(lines, fmt.Sprintf("  BETTER %-6s %s: %s -> ok", stage.name, now.path, stage.from))
			}
		}
	}

	sort.Strings(lines)
	fmt.Fprintf(os.Stderr, "\nagainst %s: %d worse, %d better, %d not in the baseline\n",
		path, regressed, improved, added)
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, line)
	}
	return regressed, nil
}

// rootRelative puts a path this program produced into the form the Java oracle
// produces. cmd/corpus is run from go/, so it says "testdata/corpus/x.pdf" and
// "../pdfbox/target/pdfs/y.pdf"; JavaCorpus is run from the repository root and
// says "go/testdata/corpus/x.pdf" and "pdfbox/target/pdfs/y.pdf".
func rootRelative(p string) string {
	// Clean first: a directory argument of "./testdata/corpus" leaves every
	// path under it with a "./" that the oracle's table does not have.
	p = path.Clean(filepath.ToSlash(p))
	if trimmed := strings.TrimPrefix(p, "../"); trimmed != p {
		return trimmed
	}
	if strings.HasPrefix(p, "go/") {
		return p
	}
	return "go/" + p
}

// compareOracle reports where the port and PDFBox disagree, and returns how
// many files the port is behind on.
//
// "Behind" is a document PDFBox reads and the port does not, at either stage.
// The other direction -- the port reading what PDFBox refuses -- is reported
// just as loudly and is not a success: this is a port, and being more permissive
// than the thing being reproduced is a difference like any other.
//
// The character counts are compared for length only, which is worth saying
// plainly: two documents of the same length are not two identical documents.
// Length catches a stage that silently produced nothing, a glyph that came out
// as two characters, a page that was not walked. It does not catch a wrong
// character. Run the oracle with -Crlf once to see why even this much needs
// the separators forced equal.
func compareOracle(path string, results []result) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	type javaRow struct {
		open, text   string
		pages, chars int
	}
	java := map[string]javaRow{}
	for i, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 5 {
			continue
		}
		row := javaRow{open: fields[1], text: fields[3]}
		fmt.Sscan(fields[2], &row.pages)
		fmt.Sscan(fields[4], &row.chars)
		java[filepath.ToSlash(fields[0])] = row
	}

	var (
		compared                                     int
		openBoth, openNeither, openBehind, openAhead int
		pageDiff                                     int
		textBoth, textNeither, textBehind, textAhead int
		charsSame, charsDiff                         int
		notInOracle                                  int
		lines                                        []string
	)

	note := func(format string, args ...any) {
		lines = append(lines, fmt.Sprintf(format, args...))
	}

	for _, now := range results {
		them, known := java[rootRelative(now.path)]
		if !known {
			notInOracle++
			continue
		}
		compared++

		goOpened, javaOpened := now.open == "ok", them.open == "ok"
		switch {
		case goOpened && javaOpened:
			openBoth++
		case !goOpened && !javaOpened:
			openNeither++
		case javaOpened:
			openBehind++
			note("  OPEN   behind %s\n           go: %s\n           java: ok, %d pages", now.path, now.open, them.pages)
		default:
			openAhead++
			note("  OPEN   ahead  %s\n           go: ok, %d pages\n           java: %s", now.path, now.pages, them.open)
		}
		if !goOpened || !javaOpened {
			continue
		}

		if now.pages != them.pages {
			pageDiff++
			note("  PAGES  %s: go %d, java %d", now.path, now.pages, them.pages)
		}

		goText, javaText := now.text == "ok", them.text == "ok"
		switch {
		case goText && javaText:
			textBoth++
			if now.chars == them.chars {
				charsSame++
			} else {
				charsDiff++
				note("  CHARS  %s: go %d, java %d", now.path, now.chars, them.chars)
			}
		case !goText && !javaText:
			textNeither++
		case javaText:
			textBehind++
			note("  TEXT   behind %s\n           go: %s\n           java: ok, %d chars", now.path, now.text, them.chars)
		default:
			textAhead++
			note("  TEXT   ahead  %s\n           go: ok, %d chars\n           java: %s", now.path, now.chars, them.text)
		}
	}

	out := os.Stderr
	fmt.Fprintf(out, "\nagainst PDFBox (%s): %d files compared", path, compared)
	if notInOracle > 0 {
		fmt.Fprintf(out, ", %d not in the oracle's table", notInOracle)
	}
	fmt.Fprintf(out, "\n\n  open    both %d, neither %d, behind %d, ahead %d\n",
		openBoth, openNeither, openBehind, openAhead)
	fmt.Fprintf(out, "  pages   %d disagree\n", pageDiff)
	fmt.Fprintf(out, "  text    both %d, neither %d, behind %d, ahead %d\n",
		textBoth, textNeither, textBehind, textAhead)
	fmt.Fprintf(out, "  chars   %d the same length, %d not\n", charsSame, charsDiff)

	disagreements := openBehind + openAhead + pageDiff + textBehind + textAhead + charsDiff
	fmt.Fprintf(out, "\n  %d of %d files disagree (%.2f%%)\n", disagreements, compared,
		percent(disagreements, compared))

	sort.Strings(lines)
	for _, line := range lines {
		fmt.Fprintln(out, line)
	}
	return openBehind + textBehind, nil
}
