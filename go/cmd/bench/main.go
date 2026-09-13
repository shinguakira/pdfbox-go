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
// Peak *live* heap, sampled while the work runs. Not HeapSys, which is the arena
// the runtime has taken from the OS and never gives back, and not the heap left
// at the end, which is near zero because the documents are closed. The Java side
// reads the same quantity out of the memory pools' peak usage.
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
	)
	flag.Parse()

	if *list == "" {
		fmt.Fprintln(os.Stderr, "usage: bench -list files.txt [-passes N] [-o timings.tsv]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	files, err := readList(*list)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bench:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "bench: %d files, %d passes\n", len(files), *passes)

	// Not timed. The JVM on the other side needs it for the JIT, running one
	// here keeps the two harnesses the same shape, and it warms the page cache
	// so neither side is timed against a cold disk.
	fmt.Fprintln(os.Stderr, "warmup ...")
	runPass(files)

	// ---- phase 1: throughput, and the peak heap while it runs

	stopSampling, peak := sampleHeap()

	bestTotal := time.Duration(1<<63 - 1)
	var chars int64
	for pass := 1; pass <= *passes; pass++ {
		started := time.Now()
		chars = runPass(files)
		elapsed := time.Since(started)
		if elapsed < bestTotal {
			bestTotal = elapsed
		}
		fmt.Fprintf(os.Stderr, "pass %d: %s\n", pass, elapsed.Round(time.Millisecond))
	}
	stopSampling()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// ---- phase 2: per document, repeated until measurable

	var perDoc []time.Duration
	if !*skipPer {
		fmt.Fprintln(os.Stderr, "per-document phase ...")
		perDoc = measurePerDocument(files)
	}

	report(files, bestTotal, chars, peak.Load(), mem.TotalAlloc, mem.HeapSys, perDoc)

	if *out != "" && perDoc != nil {
		if err := writeTimings(*out, files, perDoc); err != nil {
			fmt.Fprintln(os.Stderr, "bench:", err)
			os.Exit(1)
		}
	}
}

// sampleHeap watches the live heap while the work runs and remembers the
// highest it saw. Returns a stop function and the peak in bytes.
func sampleHeap() (func(), *atomic.Uint64) {
	peak := &atomic.Uint64{}
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		var mem runtime.MemStats
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				runtime.ReadMemStats(&mem)
				if mem.HeapAlloc > peak.Load() {
					peak.Store(mem.HeapAlloc)
				}
			}
		}
	}()
	var stopped bool
	return func() {
		if !stopped {
			stopped = true
			close(done)
		}
	}, peak
}

func runPass(files []string) int64 {
	var chars int64
	for i, path := range files {
		if i%200 == 0 {
			fmt.Fprintf(os.Stderr, "\r  %d/%d", i, len(files))
		}
		chars += scoreOne(path)
	}
	fmt.Fprintf(os.Stderr, "\r%-24s\r", "")
	return chars
}

// measurePerDocument repeats each document until the accumulated time is worth
// dividing, which is the only way to see past a 7µs clock.
func measurePerDocument(files []string) []time.Duration {
	each := make([]time.Duration, len(files))
	for i, path := range files {
		if i%200 == 0 {
			fmt.Fprintf(os.Stderr, "\r  %d/%d", i, len(files))
		}
		var elapsed time.Duration
		reps := 0
		for elapsed < perDocTarget && reps < perDocMaxReps {
			started := time.Now()
			scoreOne(path)
			elapsed += time.Since(started)
			reps++
		}
		each[i] = elapsed / time.Duration(reps)
	}
	fmt.Fprintf(os.Stderr, "\r%-24s\r", "")
	return each
}

// scoreOne is the unit of work: open, count pages, extract text. Every file in
// the list is one both implementations handle, so a failure here would be a
// change worth noticing rather than an expected outcome.
func scoreOne(path string) int64 {
	defer func() { _ = recover() }()

	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		return 0
	}
	defer document.Close()

	_ = document.NumberOfPages()

	stripper := text.NewPDFTextStripper()
	var builder strings.Builder
	stripper.SetOutput(&builder)
	if err := stripper.ProcessPages(document.Pages()); err != nil {
		return 0
	}
	return int64(len([]rune(builder.String())))
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

func report(files []string, total time.Duration, chars int64,
	peakHeap, totalAlloc, heapSys uint64, perDoc []time.Duration) {

	fmt.Println("implementation\tgo")
	fmt.Printf("files\t%d\n", len(files))
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

func writeTimings(path string, files []string, perDoc []time.Duration) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()
	fmt.Fprintln(w, "file\tms")
	for i, name := range files {
		fmt.Fprintf(w, "%s\t%.4f\n", name, float64(perDoc[i])/1e6)
	}
	return nil
}
