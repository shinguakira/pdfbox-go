import java.io.File;
import java.io.StringWriter;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import org.apache.pdfbox.Loader;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.text.PDFTextStripper;

/**
 * The Java half of the corpus comparison, and the only .java file this
 * repository owns.
 *
 * <p>Everything under the Maven module directories is a frozen snapshot of
 * upstream and strictly read-only; this is not part of it. It is a driver that
 * sits outside those directories, compiles against them, and exists so that
 * "PDFBox is the oracle" can be run over three thousand files rather than
 * asserted. See {@code go/migration/README.md}, "Checking a port against the
 * running Java", and {@code TESTDATA.md} for what the corpus is.
 *
 * <p>It prints the same table go/cmd/corpus prints -- file, open, pages, text,
 * chars -- so the two can be joined row by row and `corpus -oracle` can report
 * where they disagree. Render is deliberately not attempted: PDFBox rasterizes
 * through java.awt and the port through its own backend, so the only thing the
 * two could be compared on cheaply is whether it threw, which the text column
 * already answers.
 *
 * <p>Pass "lf" as the third argument to force the line separator and the page
 * end to LF. Java defaults both to {@link System#lineSeparator()}, which is
 * CRLF on Windows, and the port hardcodes LF and says so at
 * {@code text/pdftextstripper.go:22} -- so without this every document with any
 * text in it differs by one character per line, and the comparison says nothing
 * about content.
 *
 * <p>Pass a passwords file as the fourth argument to open encrypted files the
 * way their source project does: one line per file, a path ending, a tab, and
 * the password, in UTF-8. A file whose path ends that way is opened with that
 * password; every other file is opened with none.
 *
 * <p>Usage: {@code java JavaCorpus <listfile> [timeoutSeconds] [lf|crlf]
 * [passwordsfile]}, where listfile holds one repository-relative path per line.
 * Driven by {@code migration/scripts/run-oracle.ps1}.
 */
public class JavaCorpus {

    // How many timed-out tasks may still be running before the run gives up.
    // Each one holds a thread, a document and a file handle that nothing can
    // reclaim; see the comment at the check.
    static final int MAX_STUCK = 4;

    static boolean LF = false;

    /**
     * Rows of the optional passwords table: a path ending and the password for
     * the file whose path ends that way. The same table, matched the same way,
     * as go/cmd/corpus's -passwords, so each side opens the same files with the
     * same passwords.
     */
    static final List<String[]> PASSWORDS = new ArrayList<>();

    static String passwordFor(String path) {
        String slashed = path.replace('\\', '/');
        for (String[] row : PASSWORDS) {
            if (slashed.equals(row[0]) || slashed.endsWith("/" + row[0])) {
                return row[1];
            }
        }
        return null;
    }

    static final String LFS = String.valueOf((char) 10);

    static String shorten(Throwable t) {
        String name = t.getClass().getSimpleName();
        String msg = t.getMessage();
        String s = (msg == null || msg.isEmpty()) ? name : name + ": " + msg;
        int nl = s.indexOf('\n');
        if (nl >= 0) {
            s = s.substring(0, nl);
        }
        s = s.replace('\t', ' ').replace('\r', ' ').trim();
        return s.length() > 70 ? s.substring(0, 70) : s;
    }

    static String[] score(String path) {
        String open = "ok";
        int pages = 0;
        String text = "-";
        int chars = 0;

        PDDocument doc = null;
        try {
            String password = passwordFor(path);
            doc = password == null ? Loader.loadPDF(new File(path)) : Loader.loadPDF(new File(path), password);
            pages = doc.getNumberOfPages();
            try {
                PDFTextStripper stripper = new PDFTextStripper();
                if (LF) {
                    // Java defaults both of these to System.lineSeparator(), which is
                    // CRLF on Windows; the port hardcodes LF and says so at
                    // text/pdftextstripper.go:22. Forcing them equal is what turns
                    // "the counts differ" into a statement about content.
                    stripper.setLineSeparator(LFS);
                    stripper.setPageEnd(LFS);
                }
                StringWriter out = new StringWriter();
                stripper.writeText(doc, out);
                text = "ok";
                chars = out.toString().codePointCount(0, out.toString().length());
            } catch (Throwable t) {
                text = shorten(t);
            }
        } catch (Throwable t) {
            open = shorten(t);
        } finally {
            if (doc != null) {
                try {
                    doc.close();
                } catch (Throwable ignored) {
                    // closing is not what is being measured
                }
            }
        }
        return new String[] { open, String.valueOf(pages), text, String.valueOf(chars) };
    }

    public static void main(String[] args) throws Exception {
        List<String> files = Files.readAllLines(Paths.get(args[0]));
        files.removeIf(String::isBlank);
        files.sort(Comparator.naturalOrder());

        long timeoutSeconds = args.length > 1 ? Long.parseLong(args[1]) : 20;
        LF = args.length > 2 && args[2].equals("lf");
        if (args.length > 3) {
            for (String line : Files.readAllLines(Paths.get(args[3]), StandardCharsets.UTF_8)) {
                if (line.isBlank() || line.startsWith("#")) {
                    continue;
                }
                int tab = line.indexOf('\t');
                if (tab <= 0) {
                    throw new IllegalArgumentException(args[3] + ": a line with no tab: " + line);
                }
                PASSWORDS.add(new String[] { line.substring(0, tab).replace('\\', '/'), line.substring(tab + 1) });
            }
        }

        // Daemon threads, so a file that will not return cannot keep the JVM
        // alive. Each one gets its own executor because the previous one may
        // still be stuck in it.
        System.out.println("file\topen\tpages\ttext\tchars");
        List<Future<String[]>> stuck = new ArrayList<>();
        int i = 0;
        for (String path : files) {
            i++;
            System.err.print("\r" + i + "/" + files.size() + " " + new File(path).getName() + "                    ");

            ExecutorService pool = Executors.newSingleThreadExecutor(r -> {
                Thread t = new Thread(r);
                t.setDaemon(true);
                return t;
            });
            String[] row;
            Future<String[]> future = null;
            try {
                future = pool.submit((Callable<String[]>) () -> score(path));
                row = future.get(timeoutSeconds, TimeUnit.SECONDS);
            } catch (TimeoutException e) {
                row = new String[] { "timeout", "0", "-", "0" };
                stuck.add(future);
            } catch (Throwable t) {
                row = new String[] { shorten(t), "0", "-", "0" };
            } finally {
                pool.shutdownNow();
            }
            System.out.println(path + "\t" + row[0] + "\t" + row[1] + "\t" + row[2] + "\t" + row[3]);
            System.out.flush();

            // shutdownNow can only interrupt, and a parser that is not sitting
            // at an interruptible point ignores it, so a timed-out task keeps
            // its thread, its PDDocument and its file handle for the rest of
            // the run. Go solved this by scoring each file in a child process;
            // doing that here would mean a JVM start per file. So it is bounded
            // instead: a few are affordable, and past that the run stops rather
            // than going on measuring files on a JVM that is still busy with
            // the ones before them.
            stuck.removeIf(Future::isDone);
            if (stuck.size() > MAX_STUCK) {
                System.err.println();
                System.err.println("JavaCorpus: " + stuck.size() + " files have timed out and are still "
                        + "running; stopping at " + i + " of " + files.size() + ".");
                System.err.println("The table above this line is complete. Narrow the list and run the rest "
                        + "separately, or raise the timeout if these files are slow rather than stuck.");
                System.exit(3);
            }
        }
        System.err.println();
    }
}
