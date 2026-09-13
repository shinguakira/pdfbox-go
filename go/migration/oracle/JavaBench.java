import java.io.BufferedWriter;
import java.io.File;
import java.io.StringWriter;
import java.lang.management.ManagementFactory;
import java.lang.management.MemoryPoolMXBean;
import java.lang.management.MemoryType;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import org.apache.pdfbox.Loader;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.text.PDFTextStripper;

/**
 * Times PDFBox over a list of PDFs, printing what go/cmd/bench prints so the two
 * can be put side by side. See {@code migration/BENCHMARK.md} for what they
 * produced.
 *
 * <p>Like {@link JavaCorpus} this sits outside the Maven module directories and
 * compiles against them; the Java tree itself is untouched.
 *
 * <p>The unit of work is one document: open the file, count the pages, extract
 * the text. A warmup pass runs first and is not timed -- for the JIT, and so
 * that neither side is timed against a cold page cache.
 *
 * <p>Two phases, for the reason the Go side documents at length: wall clock over
 * the whole list is a sound throughput number, and timing a few-hundred-byte
 * document once apiece is not. The second phase repeats each document until it
 * has accumulated enough time to divide, which is also what makes the two sides
 * comparable -- the same method on both, rather than each language's best clock.
 *
 * <p>Memory is the peak used across the heap pools, which is the same quantity
 * the Go side samples as peak live heap. Run with a fixed -Xmx: a heap free to
 * grow measures the collector's appetite rather than the program's need.
 *
 * <p>Usage: {@code java JavaBench <listfile> [passes] [timings.tsv]}
 */
public class JavaBench {

    static final long PER_DOC_TARGET_NS = 2_000_000L;
    static final int PER_DOC_MAX_REPS = 500;

    static long scoreOne(String path) {
        try (PDDocument doc = Loader.loadPDF(new File(path))) {
            doc.getNumberOfPages();
            PDFTextStripper stripper = new PDFTextStripper();
            // The port hardcodes LF; matching it here keeps the two writing the
            // same number of characters. See JavaCorpus.
            stripper.setLineSeparator(String.valueOf((char) 10));
            stripper.setPageEnd(String.valueOf((char) 10));
            StringWriter out = new StringWriter();
            stripper.writeText(doc, out);
            String text = out.toString();
            return text.codePointCount(0, text.length());
        } catch (Throwable t) {
            return 0;
        }
    }

    /**
     * Walks every file once, with {@code workers} in flight.
     *
     * <p>PDFBox does not process documents concurrently -- its only Thread is a
     * shutdown hook -- so this, like go/cmd/bench's -workers, is ground neither
     * implementation has taken. It is here so the Go side's scaling is compared
     * against something rather than against nothing.
     */
    static long runPass(List<String> files, int workers) throws Exception {
        if (workers <= 1) {
            long chars = 0;
            for (int i = 0; i < files.size(); i++) {
                if (i % 200 == 0) {
                    System.err.print("\r  " + i + "/" + files.size());
                }
                chars += scoreOne(files.get(i));
            }
            System.err.print("\r                        \r");
            return chars;
        }

        java.util.concurrent.ExecutorService pool =
                java.util.concurrent.Executors.newFixedThreadPool(workers);
        java.util.concurrent.atomic.AtomicLong chars = new java.util.concurrent.atomic.AtomicLong();
        for (String path : files) {
            pool.submit(() -> chars.addAndGet(scoreOne(path)));
        }
        pool.shutdown();
        pool.awaitTermination(1, java.util.concurrent.TimeUnit.HOURS);
        System.err.print("\r                        \r");
        return chars.get();
    }

    /** Repeats a document until the accumulated time is worth dividing. */
    static long measureOne(String path) {
        long elapsed = 0;
        int reps = 0;
        while (elapsed < PER_DOC_TARGET_NS && reps < PER_DOC_MAX_REPS) {
            long started = System.nanoTime();
            scoreOne(path);
            elapsed += System.nanoTime() - started;
            reps++;
        }
        return elapsed / reps;
    }

    /** Peak used across the heap pools: Go's peak live heap, by another name. */
    static long peakHeapBytes() {
        long peak = 0;
        for (MemoryPoolMXBean pool : ManagementFactory.getMemoryPoolMXBeans()) {
            if (pool.getType() == MemoryType.HEAP && pool.getPeakUsage() != null) {
                peak += pool.getPeakUsage().getUsed();
            }
        }
        return peak;
    }

    static void resetPeaks() {
        for (MemoryPoolMXBean pool : ManagementFactory.getMemoryPoolMXBeans()) {
            if (pool.getType() == MemoryType.HEAP) {
                pool.resetPeakUsage();
            }
        }
    }

    public static void main(String[] args) throws Exception {
        List<String> files = new ArrayList<>(Files.readAllLines(Paths.get(args[0])));
        files.removeIf(String::isBlank);
        int passes = args.length > 1 ? Integer.parseInt(args[1]) : 3;
        int workers = args.length > 3 ? Integer.parseInt(args[3]) : 1;
        String timingsPath = args.length > 2 && !args[2].equals("throughput-only") ? args[2] : null;
        // Symmetric with go/cmd/bench's -throughput-only: the per-document phase
        // repeats every file and is not wanted when what is being measured is
        // the CPU one pass costs.
        boolean throughputOnly = args.length > 2 && args[2].equals("throughput-only");

        System.err.println("bench: " + files.size() + " files, " + passes + " passes");
        System.err.println("warmup ...");
        runPass(files, workers);

        // ---- phase 1: throughput, and the peak heap while it runs
        resetPeaks();

        long bestTotal = Long.MAX_VALUE;
        long chars = 0;
        for (int pass = 1; pass <= passes; pass++) {
            long started = System.nanoTime();
            chars = runPass(files, workers);
            long elapsed = System.nanoTime() - started;
            if (elapsed < bestTotal) {
                bestTotal = elapsed;
            }
            System.err.println("pass " + pass + ": " + (elapsed / 1_000_000) + " ms");
        }
        long peakHeap = peakHeapBytes();

        // ---- phase 2: per document, repeated until measurable
        long[] perDoc = new long[files.size()];
        if (throughputOnly) {
            report(files, bestTotal, chars, peakHeap, perDoc);
            return;
        }
        System.err.println("per-document phase ...");
        for (int i = 0; i < files.size(); i++) {
            if (i % 200 == 0) {
                System.err.print("\r  " + i + "/" + files.size());
            }
            perDoc[i] = measureOne(files.get(i));
        }
        System.err.print("\r                        \r");

        report(files, bestTotal, chars, peakHeap, perDoc);

        if (timingsPath != null) {
            try (BufferedWriter w = Files.newBufferedWriter(
                    Paths.get(timingsPath), StandardCharsets.UTF_8)) {
                w.write("file\tms\n");
                for (int i = 0; i < files.size(); i++) {
                    w.write(files.get(i) + "\t" + String.format("%.4f", perDoc[i] / 1e6) + "\n");
                }
            }
        }
    }

    static void report(List<String> files, long total, long chars, long peakHeap, long[] perDoc) {
        long[] sorted = perDoc.clone();
        Arrays.sort(sorted);
        long sum = 0;
        for (long t : perDoc) {
            sum += t;
        }

        System.out.println("implementation\tjava");
        System.out.println("files\t" + files.size());
        System.out.println("chars\t" + chars);
        System.out.printf("total_ms\t%.1f%n", total / 1e6);
        System.out.printf("docs_per_sec\t%.1f%n", files.size() / (total / 1e9));
        System.out.printf("peak_live_heap_mb\t%.1f%n", peakHeap / 1048576.0);
        System.out.printf("total_allocated_mb\t%s%n", "n/a");
        System.out.printf("perdoc_mean_ms\t%.4f%n", sum / 1e6 / files.size());
        System.out.printf("perdoc_p50_ms\t%.4f%n", sorted[(int) (0.50 * (sorted.length - 1))] / 1e6);
        System.out.printf("perdoc_p90_ms\t%.4f%n", sorted[(int) (0.90 * (sorted.length - 1))] / 1e6);
        System.out.printf("perdoc_p99_ms\t%.4f%n", sorted[(int) (0.99 * (sorted.length - 1))] / 1e6);
        System.out.printf("perdoc_max_ms\t%.4f%n", sorted[sorted.length - 1] / 1e6);
    }
}
