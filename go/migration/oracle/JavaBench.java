import java.io.BufferedWriter;
import java.io.File;
import java.io.StringWriter;
import java.lang.management.ManagementFactory;
import java.lang.management.MemoryMXBean;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Set;
import java.util.TreeSet;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;

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
 * <p>Memory is the peak heap in use, sampled every millisecond while the timed
 * passes run, with a sample before the first pass and one after the last:
 * {@code MemoryMXBean.getHeapMemoryUsage}, the whole heap at one moment, which
 * is what the Go side samples as {@code HeapAlloc}. Not the memory pools' own
 * peaks added together. Each pool reaches its peak at a different moment -- the
 * young generation just before a collection, the old one somewhere else -- so
 * their sum describes a heap that never existed, and an earlier version of this
 * file reported exactly that. Run with a fixed -Xmx: a heap free to grow
 * measures the collector's appetite rather than the program's need.
 *
 * <p>Every file in the list is meant to be one both implementations handle. A
 * file that fails would otherwise be timed as a very fast document, so a failure
 * is counted, reported as {@code failed}, named on stderr, and makes the exit
 * status 1.
 *
 * <p>Usage: {@code java JavaBench <listfile> [passes] [timings.tsv|throughput-only] [workers]}
 */
public class JavaBench {

    static final long PER_DOC_TARGET_NS = 2_000_000L;
    static final int PER_DOC_MAX_REPS = 500;

    /** Files that did not come through scoreOne, across every pass and worker. */
    static final Set<String> FAILED = ConcurrentHashMap.newKeySet();

    /**
     * The unit of work. Answers the characters extracted; a document that throws
     * is recorded in {@link #FAILED} and answers 0, so it is told apart from a
     * document with no text in it.
     */
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
            FAILED.add(path);
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

        ExecutorService pool = Executors.newFixedThreadPool(workers);
        AtomicLong chars = new AtomicLong();
        for (String path : files) {
            pool.submit(() -> chars.addAndGet(scoreOne(path)));
        }
        pool.shutdown();
        pool.awaitTermination(1, TimeUnit.HOURS);
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

    /**
     * Watches the heap in use while the timed passes run. The same shape as
     * go/cmd/bench's sampleHeap: one sample at once, one every millisecond, and
     * one more after the sampling thread has been stopped and joined, so a run
     * shorter than a tick does not report zero and a peak at the very end is
     * not missed.
     */
    static final class HeapSampler implements Runnable {
        private final MemoryMXBean memory = ManagementFactory.getMemoryMXBean();
        private final Thread thread = new Thread(this, "heap-sampler");
        private volatile boolean stopped;
        // Written by the sampling thread, and by the caller only before start
        // and after join, so the thread's start and join order every access.
        private long peak;

        HeapSampler() {
            sample();
            thread.setDaemon(true);
            thread.start();
        }

        private void sample() {
            peak = Math.max(peak, memory.getHeapMemoryUsage().getUsed());
        }

        @Override
        public void run() {
            while (!stopped) {
                sample();
                try {
                    Thread.sleep(1);
                } catch (InterruptedException e) {
                    return;
                }
            }
        }

        long stop() throws InterruptedException {
            stopped = true;
            thread.join();
            sample();
            return peak;
        }
    }

    public static void main(String[] args) throws Exception {
        if (args.length < 1) {
            System.err.println("usage: java JavaBench <listfile> [passes] [timings.tsv|throughput-only] [workers]");
            System.exit(2);
        }
        List<String> files = new ArrayList<>(Files.readAllLines(Paths.get(args[0])));
        files.removeIf(String::isBlank);
        if (files.isEmpty()) {
            // Nothing to time, and every figure below would divide by it.
            System.err.println("bench: " + args[0] + " holds no files");
            System.exit(2);
        }
        int passes = args.length > 1 ? Integer.parseInt(args[1]) : 3;
        if (passes < 1) {
            System.err.println("bench: passes must be at least 1");
            System.exit(2);
        }
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
        HeapSampler sampler = new HeapSampler();

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
        long peakHeap = sampler.stop();

        // ---- phase 2: per document, repeated until measurable
        long[] perDoc = null;
        if (!throughputOnly) {
            System.err.println("per-document phase ...");
            perDoc = new long[files.size()];
            for (int i = 0; i < files.size(); i++) {
                if (i % 200 == 0) {
                    System.err.print("\r  " + i + "/" + files.size());
                }
                perDoc[i] = measureOne(files.get(i));
            }
            System.err.print("\r                        \r");
        }

        report(files, bestTotal, chars, peakHeap, perDoc);

        if (timingsPath != null && perDoc != null) {
            try (BufferedWriter w = Files.newBufferedWriter(
                    Paths.get(timingsPath), StandardCharsets.UTF_8)) {
                w.write("file\tms\n");
                for (int i = 0; i < files.size(); i++) {
                    w.write(files.get(i) + "\t" + String.format("%.4f", perDoc[i] / 1e6) + "\n");
                }
            }
        }

        if (!FAILED.isEmpty()) {
            for (String path : new TreeSet<>(FAILED)) {
                System.err.println("bench: failed: " + path);
            }
            System.err.println("bench: " + FAILED.size() + " of " + files.size()
                    + " files failed, and the figures above count them as documents");
            System.exit(1);
        }
    }

    /** Prints the figures; the per-document ones only when that phase ran. */
    static void report(List<String> files, long total, long chars, long peakHeap, long[] perDoc) {
        System.out.println("implementation\tjava");
        System.out.println("files\t" + files.size());
        System.out.println("failed\t" + FAILED.size());
        System.out.println("chars\t" + chars);
        System.out.printf("total_ms\t%.1f%n", total / 1e6);
        System.out.printf("docs_per_sec\t%.1f%n", files.size() / (total / 1e9));
        System.out.printf("peak_live_heap_mb\t%.1f%n", peakHeap / 1048576.0);
        System.out.printf("total_allocated_mb\t%s%n", "n/a");

        if (perDoc == null) {
            return;
        }

        long[] sorted = perDoc.clone();
        Arrays.sort(sorted);
        long sum = 0;
        for (long t : perDoc) {
            sum += t;
        }
        System.out.printf("perdoc_mean_ms\t%.4f%n", sum / 1e6 / files.size());
        System.out.printf("perdoc_p50_ms\t%.4f%n", sorted[(int) (0.50 * (sorted.length - 1))] / 1e6);
        System.out.printf("perdoc_p90_ms\t%.4f%n", sorted[(int) (0.90 * (sorted.length - 1))] / 1e6);
        System.out.printf("perdoc_p99_ms\t%.4f%n", sorted[(int) (0.99 * (sorted.length - 1))] / 1e6);
        System.out.printf("perdoc_max_ms\t%.4f%n", sorted[sorted.length - 1] / 1e6);
    }
}
