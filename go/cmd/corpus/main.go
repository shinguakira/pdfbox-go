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
// ok or the first thing that went wrong, and a last column holding a digest of
// the text. The table is then read two ways.
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
// -passwords names a table of the ways a source project opens its encrypted
// files, and may be given more than once. Each line is one way of opening one
// file: "<path ending>\t<password>", or "<path ending>\t<password>\t<certificate>\t<private key>"
// for a file encrypted for the holder of a certificate, the two files named
// relative to the table's directory. fetch-corpus.ps1 writes one for a suite
// that publishes them, and run-oracle.ps1 -Passwords hands the same tables to
// PDFBox, so both sides open the same files the same ways. A file is opened once
// for every line that names it, and scored once for each; where there is more
// than one, each row's file is followed by the rest of its line in brackets, the
// fields joined by " | ". A file no line names that refuses a password is
// counted as encrypted and left out of the rates; a file opened as a line says
// that still refuses is a failure.
//
// Usage:
//
//	go run ./cmd/corpus testdata/corpus/verapdf > verapdf.tsv
//	go run ./cmd/corpus -render testdata/corpus/safedocs-targeted
//	go run ./cmd/corpus -baseline verapdf.tsv testdata/corpus/verapdf
//	go run ./cmd/corpus -oracle ../go/testdata/oracle/java-corpus.tsv testdata/corpus
//	go run ./cmd/corpus -passwords testdata/corpus/pdfjs/_passwords.tsv testdata/corpus/pdfjs
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// result is one file's row, or one of its rows where a passwords table opens it
// more than once. Each stage holds "ok", "-" for a stage that was not reached or
// not asked for, or a short description of what went wrong.
type result struct {
	path   string
	label  string // "" or " [<the rest of the passwords line>]"
	open   string
	pages  int
	text   string
	chars  int
	render string
	digest string // the first 8 bytes of the SHA-256 of the text, in hex, or "-"
}

// header names the columns, and is also what -baseline parses.
const header = "file\topen\tpages\ttext\tchars\trender\tdigest"

func (r result) String() string {
	return fmt.Sprintf("%s%s\t%s\t%d\t%s\t%d\t%s\t%s",
		r.path, r.label, r.open, r.pages, r.text, r.chars, r.render, r.digest)
}

// digest is what the digest column holds for a text: the first eight bytes of
// the SHA-256 of its UTF-8, in hex. JavaCorpus writes the same of the String
// PDFBox extracts, so two texts of the same length that differ anywhere show
// as two digests.
func digest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:8])
}

// job is one way of opening one file: with no password, or as one line of the
// passwords tables says.
type job struct {
	path  string
	open  int    // index into openings, or -1
	label string // what tells this job's row from the file's other rows
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
		pages    = flag.String("pages", "", "instead of scoring, write a digest per page of the files given, for narrowing a disagreement the document digest found to a page")
		cmpPages = flag.Bool("comparepages", false, "compare two page tables, PDFBox's first and this program's second, and report every page they disagree on")
		openLine = flag.Int("open", -1, "with -one, open the file as the line of this index in the passwords tables says, counting from 0 across them in order")
	)
	flag.Func("passwords", "a table of the ways a source project opens its encrypted documents: <path ending>\\t<password>, or <path ending>\\t<password>\\t<certificate>\\t<private key>; may be given more than once, and each file is opened once for every line naming it", func(table string) error {
		passwordsPaths = append(passwordsPaths, table)
		return loadPasswords(table)
	})
	flag.Parse()

	if *one {
		if flag.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "corpus: -one takes exactly one file")
			os.Exit(2)
		}
		if *openLine < -1 || *openLine >= len(openings) {
			fmt.Fprintln(os.Stderr, "corpus: -open names no line of the passwords tables")
			os.Exit(2)
		}
		fmt.Println(scoreNow(jobFor(flag.Arg(0), *openLine), *render, float32(*dpi)))
		return
	}

	if *cmpPages {
		if flag.NArg() != 2 {
			fmt.Fprintln(os.Stderr, "corpus: -comparepages takes two tables, PDFBox's first")
			os.Exit(2)
		}
		differ, err := comparePages(flag.Arg(0), flag.Arg(1))
		if err != nil {
			fmt.Fprintln(os.Stderr, "corpus:", err)
			os.Exit(1)
		}
		if differ > 0 {
			os.Exit(1)
		}
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
		scorer = func(j job, withRender bool, dpi float32, timeout time.Duration) result {
			return scoreIsolated(exe, j, withRender, dpi, timeout)
		}
	}

	jobs := jobsFor(files)

	if *pages != "" {
		if err := writePages(jobs, *pages); err != nil {
			fmt.Fprintln(os.Stderr, "corpus:", err)
			os.Exit(1)
		}
		return
	}

	results := make([]result, 0, len(jobs))
	started := time.Now()
	for i, j := range jobs {
		if !*quiet {
			fmt.Fprintf(os.Stderr, "\r%d/%d %-60.60s", i+1, len(jobs), filepath.Base(j.path))
		}
		results = append(results, scorer(j, *render, float32(*dpi), *timeout))
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
func scoreIsolated(exe string, j job, withRender bool, dpi float32, timeout time.Duration) result {
	args := []string{"-one"}
	if withRender {
		args = append(args, "-render", "-dpi", fmt.Sprint(dpi))
	}
	for _, table := range passwordsPaths {
		args = append(args, "-passwords", table)
	}
	args = append(args, "-open", fmt.Sprint(j.open), j.path)

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
		if fields := strings.Split(row, "\t"); len(fields) == 7 {
			pages := 0
			chars := 0
			fmt.Sscan(fields[2], &pages)
			fmt.Sscan(fields[4], &chars)
			return result{
				path: j.path, label: j.label, open: fields[1], pages: pages,
				text: fields[3], chars: chars, render: fields[5], digest: fields[6],
			}
		}
	}

	if ctx.Err() == context.DeadlineExceeded {
		return result{path: j.path, label: j.label, open: "timeout", text: "-", render: "-", digest: "-"}
	}
	return result{path: j.path, label: j.label, open: crashReason(stderr.String(), err), text: "-", render: "-", digest: "-"}
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
func score(j job, withRender bool, dpi float32, timeout time.Duration) result {
	done := make(chan result, 1)
	go func() { done <- scoreNow(j, withRender, dpi) }()

	if timeout <= 0 {
		return <-done
	}
	select {
	case r := <-done:
		return r
	case <-time.After(timeout):
		return result{path: j.path, label: j.label, open: "timeout", text: "-", render: "-", digest: "-"}
	}
}

func scoreNow(j job, withRender bool, dpi float32) (r result) {
	r = result{path: j.path, label: j.label, open: "ok", text: "-", render: "-", digest: "-"}
	// The stage in flight, which is the column a panic is written to. Opening
	// runs to the page count, because that is where JavaCorpus's outer try
	// ends: an unchecked exception from Loader.loadPDF or getNumberOfPages is an
	// open failure there, and a panic from the same calls is one here.
	stage := &r.open
	defer func() {
		if p := recover(); p != nil {
			// A panic is the finding, so it carries its message: Java reaches
			// most of these as a checked IOException, and which panic it is
			// says whether the port has a nil where Java has a null check.
			*stage = short(fmt.Errorf("panic: %v", p))
			if stage == &r.open {
				r.pages = 0
			}
		}
	}()

	document, err := openDocument(j)
	if err != nil {
		r.open = short(err)
		if j.open < 0 && strings.Contains(r.open, "password is incorrect") {
			// Not a failure of anything. The corpora carry encrypted documents
			// whose passwords live in the tests that read them; a file no line
			// of the -passwords tables names is counted apart from the rest so
			// the open rate is not quietly wrong. A file opened as a line says,
			// and refused, keeps its error: that is a failure.
			r.open = "encrypted"
		}
		return r
	}
	defer document.Close()

	r.pages = document.NumberOfPages()

	stage = &r.text
	stripper := text.NewPDFTextStripper()
	var builder strings.Builder
	stripper.SetOutput(&builder)
	if err := stripper.ProcessPages(document.Pages()); err != nil {
		r.text = short(err)
	} else {
		r.text = "ok"
		r.chars = len([]rune(builder.String()))
		r.digest = digest(builder.String())
	}

	if withRender {
		stage = &r.render
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
	// The rates are over rows, which are files unless a passwords table opens
	// some of them more than one way.
	n := len(results) - encrypted
	files := map[string]bool{}
	for _, r := range results {
		files[r.path] = true
	}
	fmt.Fprintf(os.Stderr, "\n%d files in %s", len(files), elapsed.Round(time.Millisecond))
	if len(files) != len(results) {
		fmt.Fprintf(os.Stderr, ", opened %d ways", len(results))
	}
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
		before, known := was[now.path+now.label]
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
				lines = append(lines, fmt.Sprintf("  WORSE  %-6s %s%s: ok -> %s", stage.name, now.path, now.label, stage.to))
			case stage.to == "ok":
				improved++
				lines = append(lines, fmt.Sprintf("  BETTER %-6s %s%s: %s -> ok", stage.name, now.path, now.label, stage.from))
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
// The text is compared twice. Its length catches a stage that silently produced
// nothing, a glyph that came out as two characters, a page that was not walked.
// Where the lengths agree and both tables carry a digest of the text, the
// digests are compared too, which catches the wrong character and the right
// characters in the wrong order; a table JavaCorpus wrote before it wrote
// digests is compared on length alone. Run the oracle with -Crlf once to see why
// either needs the separators forced equal.
func compareOracle(path string, results []result) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	type javaRow struct {
		open, text   string
		pages, chars int
		digest       string // "" in a table written before JavaCorpus wrote one
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
		if len(fields) > 5 {
			row.digest = fields[5]
		}
		java[filepath.ToSlash(fields[0])] = row
	}

	var (
		compared                                     int
		openBoth, openNeither, openBehind, openAhead int
		pageDiff                                     int
		textBoth, textNeither, textBehind, textAhead int
		charsSame, charsDiff                         int
		contentSame, contentDiff                     int
		notInOracle                                  int
		lines                                        []string
	)

	// The counters above count mismatches and one file can hold more than one
	// of them -- a page count and a character count can both differ on the same
	// document. This counts the files, which is what the summary line says it
	// is reporting.
	disagreed := map[string]bool{}
	files := map[string]bool{}

	note := func(path, format string, args ...any) {
		disagreed[path] = true
		lines = append(lines, fmt.Sprintf(format, args...))
	}

	for _, now := range results {
		them, known := java[rootRelative(now.path)+now.label]
		if !known {
			notInOracle++
			continue
		}
		compared++
		files[now.path] = true
		name := now.path + now.label

		goOpened, javaOpened := now.open == "ok", them.open == "ok"
		switch {
		case goOpened && javaOpened:
			openBoth++
		case !goOpened && !javaOpened:
			openNeither++
		case javaOpened:
			openBehind++
			note(now.path, "  OPEN   behind %s\n           go: %s\n           java: ok, %d pages", name, now.open, them.pages)
		default:
			openAhead++
			note(now.path, "  OPEN   ahead  %s\n           go: ok, %d pages\n           java: %s", name, now.pages, them.open)
		}
		if !goOpened || !javaOpened {
			continue
		}

		if now.pages != them.pages {
			pageDiff++
			note(now.path, "  PAGES  %s: go %d, java %d", name, now.pages, them.pages)
		}

		goText, javaText := now.text == "ok", them.text == "ok"
		switch {
		case goText && javaText:
			textBoth++
			if now.chars == them.chars {
				charsSame++
				// The same length says nothing about the characters. Where
				// both tables carry a digest of the text, compare those too.
				if them.digest != "" && now.digest != "" {
					if now.digest == them.digest {
						contentSame++
					} else {
						contentDiff++
						note(now.path, "  DIGEST %s: %d chars on both sides, not the same ones", name, now.chars)
					}
				}
			} else {
				charsDiff++
				note(now.path, "  CHARS  %s: go %d, java %d", name, now.chars, them.chars)
			}
		case !goText && !javaText:
			textNeither++
		case javaText:
			textBehind++
			note(now.path, "  TEXT   behind %s\n           go: %s\n           java: ok, %d chars", name, now.text, them.chars)
		default:
			textAhead++
			note(now.path, "  TEXT   ahead  %s\n           go: ok, %d chars\n           java: %s", name, now.chars, them.text)
		}
	}

	out := os.Stderr
	fmt.Fprintf(out, "\nagainst PDFBox (%s): %d files compared", path, len(files))
	if compared != len(files) {
		fmt.Fprintf(out, " in %d rows, the passwords tables opening some more than one way", compared)
	}
	if notInOracle > 0 {
		fmt.Fprintf(out, ", %d not in the oracle's table", notInOracle)
	}
	fmt.Fprintf(out, "\n\n  open    both %d, neither %d, behind %d, ahead %d\n",
		openBoth, openNeither, openBehind, openAhead)
	fmt.Fprintf(out, "  pages   %d disagree\n", pageDiff)
	fmt.Fprintf(out, "  text    both %d, neither %d, behind %d, ahead %d\n",
		textBoth, textNeither, textBehind, textAhead)
	fmt.Fprintf(out, "  chars   %d the same length, %d not\n", charsSame, charsDiff)
	if contentSame+contentDiff > 0 {
		fmt.Fprintf(out, "  digest  %d of the same length the same text, %d not\n", contentSame, contentDiff)
	}

	// Files, not mismatches: a document whose page count and character count both
	// differ is one file that disagrees, and summing the counters above would
	// call it two.
	fmt.Fprintf(out, "\n  %d of %d files disagree (%.2f%%)\n", len(disagreed), len(files),
		percent(len(disagreed), len(files)))

	sort.Strings(lines)
	for _, line := range lines {
		fmt.Fprintln(out, line)
	}
	return openBehind + textBehind, nil
}

// passwordsPaths are the -passwords tables, in the order given, handed on to
// each child process so that its -open indexes the same lines.
var passwordsPaths []string

// opening is one line of a passwords table: one way of opening one file.
type opening struct {
	ending      string // the path ending, slash separated
	password    string
	certificate string // "" for a password line; else resolved against the table's directory
	key         string
	label       string // the line after its path ending, its fields joined by " | "
}

// openings are the lines of every -passwords table, in the order read.
var openings []opening

// loadPasswords reads a passwords table. Each line is a path ending and a
// password, or a path ending, a password, a certificate and a private key, tab
// separated. A path ending matches a file whose slash-separated path is that
// ending or ends with "/" and that ending, so "pdfjs/issue3371.pdf" names the
// same file here and in the PDFBox driver, which is handed paths from the
// repository root. Blank lines and lines starting with # are ignored.
func loadPasswords(table string) error {
	content, err := os.ReadFile(table)
	if err != nil {
		return err
	}
	dir := filepath.Dir(table)
	for _, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		o := opening{ending: filepath.ToSlash(fields[0]), label: strings.Join(fields[1:], " | ")}
		switch {
		case o.ending != "" && len(fields) == 2:
			o.password = fields[1]
		case o.ending != "" && len(fields) == 4:
			o.password = fields[1]
			o.certificate = filepath.Join(dir, filepath.FromSlash(fields[2]))
			o.key = filepath.Join(dir, filepath.FromSlash(fields[3]))
		default:
			return fmt.Errorf("%s: a line that is neither a path ending and a password nor a path ending, a password, a certificate and a key: %q", table, line)
		}
		openings = append(openings, o)
	}
	return nil
}

// openingsFor answers the indexes of the lines that name a file, in table order.
func openingsFor(path string) []int {
	slashed := filepath.ToSlash(path)
	var lines []int
	for i, o := range openings {
		if slashed == o.ending || strings.HasSuffix(slashed, "/"+o.ending) {
			lines = append(lines, i)
		}
	}
	return lines
}

// jobsFor answers the ways the files are opened: once with no password for a
// file no line names, and once for every line that names one.
func jobsFor(files []string) []job {
	var jobs []job
	for _, path := range files {
		lines := openingsFor(path)
		if len(lines) == 0 {
			jobs = append(jobs, job{path: path, open: -1})
			continue
		}
		for _, line := range lines {
			jobs = append(jobs, jobFor(path, line))
		}
	}
	return jobs
}

// jobFor answers the job of opening a file as one line says, labelled the way
// JavaCorpus labels it: only when the tables open the file more than one way.
func jobFor(path string, line int) job {
	j := job{path: path, open: line}
	if line >= 0 && len(openingsFor(path)) > 1 {
		j.label = " [" + openings[line].label + "]"
	}
	return j
}

// openDocument opens a file as its job says.
func openDocument(j job) (*pdmodel.PDDocument, error) {
	if j.open < 0 {
		return pdfbox.LoadPDF(j.path)
	}
	o := openings[j.open]
	if o.certificate == "" {
		return pdfbox.LoadPDFWithPassword(j.path, o.password)
	}
	store, err := keyStoreFor(o)
	if err != nil {
		return nil, err
	}
	return pdfbox.LoadPDFWithKeyStore(j.path, o.password, bytes.NewReader(store), "")
}
