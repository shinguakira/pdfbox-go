// Command bench times the port over a list of PDFs, so it can be compared with
// the same work done by PDFBox.
//
// This is not a port of anything. Its Java counterpart is
// migration/oracle/JavaBench.java, which prints the same numbers in the same
// shape, and migration/BENCHMARK.md is what the two produced.
//
// The unit of work is one document's worth of real use: open the file, count the
// pages, extract the text. Not a microbenchmark of a function -- the question is
// what a caller pays for a document.
//
// # Why it is in two phases
//
// Throughput and per-document cost cannot be measured the same way here.
//
// Wall clock over the whole list is a sound throughput number: tens of seconds
// against a clock good to microseconds.
//
// Per document it is not. time.Now() on this machine resolves about 7µs at
// best, and most of the corpus is veraPDF's clause tests, which are a few
// hundred bytes each. Timing those once apiece reports zero -- the first run of
// this program said p50 was 0.000ms for exactly that reason, which is a
// statement about the clock and not about the code. So the second phase repeats
// each document until it has accumulated enough time to divide, the way
// testing.B does.
//
// # Memory
//
// Peak heap in use, sampled every millisecond while the timed passes run, with
// one sample before the first pass and one after the last. Not HeapSys, which
// is the arena the runtime has taken from the OS and never gives back, and not
// the heap left at the end, which is near zero because the documents are
// closed. The Java side samples the whole heap the same way.
//
// # Failures
//
// Every file in the list is meant to be one both implementations handle. A file
// that does not load, does not extract or panics would otherwise be timed as a
// very fast document and pull every figure down with it, so a failure is
// counted, reported as failed, named on stderr, and makes the command exit 1.
//
// Usage:
//
//	go run ./cmd/bench -list files.txt -passes 3 -o go-timings.tsv
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// perDocTarget is how much time one document must accumulate before its cost is
// worth dividing out. Well above the clock's resolution.
const perDocTarget = 2 * time.Millisecond

// perDocMaxReps bounds the repetition for documents that are already slow.
const perDocMaxReps = 500

func main() {
	var (
		list    = flag.String("list", "", "file holding one PDF path per line")
		passes  = flag.Int("passes", 3, "timed passes over the list, for throughput")
		out     = flag.String("o", "", "write per-document timings here")
		skipPer = flag.Bool("throughput-only", false, "skip the per-document phase")
		workers = flag.Int("workers", 1, "documents to process at once; 1 is the shape PDFBox runs in")
	)
	flag.Parse()

	if *list == "" || *passes < 1 {
		fmt.Fprintln(os.Stderr, "usage: bench -list files.txt [-passes N] [-o timings.tsv]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	files, err := readList(*list)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bench:", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		// Nothing to time, and every figure below would divide by it.
		fmt.Fprintln(os.Stderr, "bench:", *list, "holds no files")
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "bench: %d files, %d passes\n", len(files), *passes)

	failed := newFailures()

	// Not timed. The JVM on the other side needs it for the JIT, running one
	// here keeps the two harnesses the same shape, and it warms the page cache
	// so neither side is timed against a cold disk.
	fmt.Fprintln(os.Stderr, "warmup ...")
	runPass(files, *workers, failed)

	// ---- phase 1: throughput, and the peak heap while it runs

	stopSampling := sampleHeap()

	bestTotal := time.Duration(1<<63 - 1)
	var chars int64
	for pass := 1; pass <= *passes; pass++ {
		started := time.Now()
		chars = runPass(files, *workers, failed)
		elapsed := time.Since(started)
		if elapsed < bestTotal {
			bestTotal = elapsed
		}
		fmt.Fprintf(os.Stderr, "pass %d: %s\n", pass, elapsed.Round(time.Millisecond))
	}
	peak := stopSampling()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// ---- phase 2: per document, repeated until measurable

	var perDoc []time.Duration
	if !*skipPer {
		fmt.Fprintln(os.Stderr, "per-document phase ...")
		perDoc = measurePerDocument(files, failed)
	}

	report(files, bestTotal, chars, failed.count(), peak, mem.TotalAlloc, mem.HeapSys, perDoc)

	if *out != "" && perDoc != nil {
		if err := writeTimings(*out, files, perDoc); err != nil {
			fmt.Fprintln(os.Stderr, "bench:", err)
			os.Exit(1)
		}
	}

	if n := failed.count(); n > 0 {
		for _, path := range failed.paths() {
			fmt.Fprintln(os.Stderr, "bench: failed:", path)
		}
		fmt.Fprintf(os.Stderr, "bench: %d of %d files failed, and the figures above count them as documents\n",
			n, len(files))
		os.Exit(1)
	}
}

// sampleHeap watches the heap in use while the work runs. It samples once
// straight away and then every millisecond; the function it returns stops the
// sampling, waits for it to end, takes a last sample and answers the highest
// value seen.
//
// The samples at either end are what stop a run shorter than a tick from
// reporting zero, and a peak in the last millisecond from being missed.
func sampleHeap() (stop func() uint64) {
	var peak uint64
	sample := func() {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		peak = max(peak, mem.HeapAlloc)
	}

	sample()
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				sample()
			}
		}
	}()

	var once sync.Once
	return func() uint64 {
		once.Do(func() {
			close(done)
			<-finished
			sample()
		})
		return peak
	}
}

// failures remembers which files did not come through scoreOne, across every
// pass and every worker.
type failures struct {
	mu    sync.Mutex
	files map[string]bool
}

func newFailures() *failures { return &failures{files: map[string]bool{}} }

func (f *failures) add(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[path] = true
}

func (f *failures) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.files)
}

func (f *failures) paths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	paths := make([]string, 0, len(f.files))
	for path := range f.files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// runPass walks every file once, with the given number in flight at a time.
//
// Neither implementation processes documents concurrently. PDFBox has exactly
// one Thread in its source and it is a shutdown hook that deletes temporary
// directories; the cores it uses beyond one are the JIT compiling while its
// single application thread runs, which restricting the compiler to one thread
// confirms -- 2.68 cores becomes 1.71.
//
// So -workers is not catching up with something Java does. It is ground neither
// has taken, on work that is embarrassingly parallel: documents do not know
// about each other, and each gets its own PDDocument and its own stripper.
func runPass(files []string, workers int, failed *failures) int64 {
	if workers <= 1 {
		var chars int64
		for i, path := range files {
			if i%200 == 0 {
				fmt.Fprintf(os.Stderr, "\r  %d/%d", i, len(files))
			}
			n, ok := scoreOne(path)
			if !ok {
				failed.add(path)
			}
			chars += n
		}
		fmt.Fprintf(os.Stderr, "\r%-24s\r", "")
		return chars
	}

	var chars atomic.Int64
	var wg sync.WaitGroup
	queue := make(chan string, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range queue {
				n, ok := scoreOne(path)
				if !ok {
					failed.add(path)
				}
				chars.Add(n)
			}
		}()
	}
	for i, path := range files {
		if i%200 == 0 {
			fmt.Fprintf(os.Stderr, "\r  %d/%d", i, len(files))
		}
		queue <- path
	}
	close(queue)
	wg.Wait()
	fmt.Fprintf(os.Stderr, "\r%-24s\r", "")
	return chars.Load()
}

// measurePerDocument repeats each document until the accumulated time is worth
// dividing, which is the only way to see past a 7µs clock.
func measurePerDocument(files []string, failed *failures) []time.Duration {
	each := make([]time.Duration, len(files))
	for i, path := range files {
		if i%200 == 0 {
			fmt.Fprintf(os.Stderr, "\r  %d/%d", i, len(files))
		}
		var elapsed time.Duration
		reps := 0
		for elapsed < perDocTarget && reps < perDocMaxReps {
			started := time.Now()
			_, ok := scoreOne(path)
			elapsed += time.Since(started)
			reps++
			if !ok {
				failed.add(path)
			}
		}
		each[i] = elapsed / time.Duration(reps)
	}
	fmt.Fprintf(os.Stderr, "\r%-24s\r", "")
	return each
}

// scoreOne is the unit of work: open, count pages, extract text. It answers
// the characters extracted and whether the document came through at all. A
// load error, an extraction error and a panic are each a document that did
// not, and are told apart from a document with no text in it.
func scoreOne(path string) (chars int64, ok bool) {
	defer func() {
		if recover() != nil {
			chars, ok = 0, false
		}
	}()

	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		return 0, false
	}
	defer document.Close()

	_ = document.NumberOfPages()

	stripper := text.NewPDFTextStripper()
	var builder strings.Builder
	stripper.SetOutput(&builder)
	if err := stripper.ProcessPages(document.Pages()); err != nil {
		return 0, false
	}
	return int64(len([]rune(builder.String()))), true
}

func readList(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var files []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			files = append(files, line)
		}
	}
	return files, scanner.Err()
}

func report(files []string, total time.Duration, chars int64, failed int,
	peakHeap, totalAlloc, heapSys uint64, perDoc []time.Duration) {

	fmt.Println("implementation\tgo")
	fmt.Printf("files\t%d\n", len(files))
	fmt.Printf("failed\t%d\n", failed)
	fmt.Printf("chars\t%d\n", chars)
	fmt.Printf("total_ms\t%.1f\n", float64(total)/1e6)
	fmt.Printf("docs_per_sec\t%.1f\n", float64(len(files))/total.Seconds())
	fmt.Printf("peak_live_heap_mb\t%.1f\n", float64(peakHeap)/1048576)
	fmt.Printf("heap_from_os_mb\t%.1f\n", float64(heapSys)/1048576)
	fmt.Printf("total_allocated_mb\t%.1f\n", float64(totalAlloc)/1048576)

	if perDoc == nil {
		return
	}

	sorted := append([]time.Duration(nil), perDoc...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	at := func(q float64) float64 {
		return float64(sorted[int(q*float64(len(sorted)-1))]) / 1e6
	}
	var sum time.Duration
	for _, d := range perDoc {
		sum += d
	}

	fmt.Printf("perdoc_mean_ms\t%.4f\n", float64(sum)/1e6/float64(len(perDoc)))
	fmt.Printf("perdoc_p50_ms\t%.4f\n", at(0.50))
	fmt.Printf("perdoc_p90_ms\t%.4f\n", at(0.90))
	fmt.Printf("perdoc_p99_ms\t%.4f\n", at(0.99))
	fmt.Printf("perdoc_max_ms\t%.4f\n", float64(sorted[len(sorted)-1])/1e6)
}

// writeTimings writes one line per document. A failed write is reported, not
// left in a buffer that a deferred flush would have thrown away with its error:
// a truncated timings file that looks complete is worse than none.
func writeTimings(path string, files []string, perDoc []time.Duration) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()

	w := bufio.NewWriter(f)
	fmt.Fprintln(w, "file\tms")
	for i, name := range files {
		fmt.Fprintf(w, "%s\t%.4f\n", name, float64(perDoc[i])/1e6)
	}
	// bufio.Writer keeps the first error it meets and Flush returns it, so this
	// one check covers every line above.
	return w.Flush()
}
