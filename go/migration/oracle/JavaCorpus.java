import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileDescriptor;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.PrintStream;
import java.io.StringReader;
import java.io.StringWriter;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.KeyStore;
import java.security.MessageDigest;
import java.security.PrivateKey;
import java.security.Provider;
import java.security.cert.Certificate;
import java.security.cert.CertificateFactory;
import java.util.ArrayList;
import java.util.Arrays;
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
import org.bouncycastle.asn1.pkcs.PrivateKeyInfo;
import org.bouncycastle.cert.X509CertificateHolder;
import org.bouncycastle.cert.jcajce.JcaX509CertificateConverter;
import org.bouncycastle.jce.provider.BouncyCastleProvider;
import org.bouncycastle.openssl.PEMKeyPair;
import org.bouncycastle.openssl.PEMParser;
import org.bouncycastle.openssl.jcajce.JcaPEMKeyConverter;
import org.bouncycastle.openssl.jcajce.JceOpenSSLPKCS8DecryptorProviderBuilder;
import org.bouncycastle.pkcs.PKCS8EncryptedPrivateKeyInfo;

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
 * chars, and a digest of the text -- so the two can be joined row by row and
 * `corpus -oracle` can report where they disagree. Render is deliberately not attempted: PDFBox rasterizes
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
 * <p>Pass passwords tables as the fourth argument and after, to open encrypted
 * files the way their source project does. Each line of a table, in UTF-8, is
 * one way of opening one file: a path ending, a tab, and the password; or a path
 * ending, the password, a certificate and a private key, tab separated, for a
 * file encrypted for the holder of a certificate. The certificate and the key
 * are files, named relative to the table's directory -- PEM or DER, the key
 * PKCS#8 and, when encrypted, encrypted with the password -- and they are put
 * into a one-entry PKCS#12 keystore protected by the same password, which is
 * what PDFBox reads. A line of three fields -- a path ending, a passphrase and
 * a PKCS#12 keystore -- names a project's own keystore, holding that
 * certificate and key already, and it is read as it is. A file whose path ends
 * that way is opened once for every line naming it, and a row is written for
 * each; when there is more than one, each row's file is followed by the line
 * after its path ending, in brackets, its fields joined by " | ". A file no
 * line names is opened with no password.
 * go/cmd/corpus reads the same tables the same way.
 *
 * <p>Usage: {@code java JavaCorpus <listfile> [timeoutSeconds] [lf|crlf]
 * [passwordsfile...]}, where listfile holds one repository-relative path per
 * line. Driven by {@code migration/scripts/run-oracle.ps1}.
 */
public class JavaCorpus {

    // How many timed-out tasks may still be running before the run gives up.
    // Each one holds a thread, a document and a file handle that nothing can
    // reclaim; see the comment at the check.
    static final int MAX_STUCK = 4;

    static boolean LF = false;

    /** One line of a passwords table: one way of opening one file. */
    static final class Open {
        /** The path ending the line names, with forward slashes. */
        final String ending;
        final String password;
        /** The certificate and private key, or null for a password line. */
        final Path certificate;
        final Path key;
        /** The project's own PKCS#12 keystore, or null for any other line. */
        final Path keystore;
        /** The line after its path ending, which tells two opens of one file apart. */
        final String label;

        Open(String ending, String password, Path certificate, Path key, Path keystore, String label) {
            this.ending = ending;
            this.password = password;
            this.certificate = certificate;
            this.key = key;
            this.keystore = keystore;
            this.label = label;
        }
    }

    /**
     * The lines of every passwords table, in the order the tables and their
     * lines were given. The same tables, matched the same way, as
     * go/cmd/corpus's -passwords, so each side opens the same files the same
     * ways.
     */
    static final List<Open> OPENS = new ArrayList<>();

    /** The lines that name a file, in table order. */
    static List<Open> opensFor(String path) {
        String slashed = path.replace('\\', '/');
        List<Open> opens = new ArrayList<>();
        for (Open open : OPENS) {
            if (slashed.equals(open.ending) || slashed.endsWith("/" + open.ending)) {
                opens.add(open);
            }
        }
        return opens;
    }

    static void readTable(String table) throws Exception {
        Path dir = Paths.get(table).toAbsolutePath().getParent();
        for (String line : Files.readAllLines(Paths.get(table), StandardCharsets.UTF_8)) {
            if (line.isBlank() || line.startsWith("#")) {
                continue;
            }
            String[] fields = line.split(String.valueOf((char) 9), -1);
            String ending = fields[0].replace('\\', '/');
            String label = String.join(" | ", Arrays.asList(fields).subList(1, fields.length));
            if (fields.length == 2 && !ending.isEmpty()) {
                OPENS.add(new Open(ending, fields[1], null, null, null, label));
            } else if (fields.length == 3 && !ending.isEmpty()) {
                OPENS.add(new Open(ending, fields[1], null, null, dir.resolve(fields[2]), label));
            } else if (fields.length == 4 && !ending.isEmpty()) {
                OPENS.add(new Open(ending, fields[1], dir.resolve(fields[2]), dir.resolve(fields[3]), null, label));
            } else {
                throw new IllegalArgumentException(table + ": a line that is none of a path ending and a "
                        + "password, a path ending, a passphrase and a keystore, and a path ending, a "
                        + "password, a certificate and a key: " + line);
            }
        }
    }

    /** Bouncy Castle, handed to the PEM reading below and registered nowhere. */
    static final Provider BC = new BouncyCastleProvider();

    /**
     * The one-entry PKCS#12 keystore a certificate line describes, as the
     * bytes PDFBox's loader reads, protected by the line's password -- or, for a
     * keystore line, the project's own store, as it is.
     */
    static InputStream keyStoreFor(Open open) throws Exception {
        if (open.keystore != null) {
            // The project's own store, in the shape PDFBox's loader reads.
            return new ByteArrayInputStream(Files.readAllBytes(open.keystore));
        }
        Certificate certificate = certificate(open.certificate);
        PrivateKey key = privateKey(open.key, open.password);
        KeyStore store = KeyStore.getInstance("PKCS12");
        store.load(null, null);
        store.setKeyEntry("key", key, open.password.toCharArray(), new Certificate[] { certificate });
        ByteArrayOutputStream out = new ByteArrayOutputStream();
        store.store(out, open.password.toCharArray());
        return new ByteArrayInputStream(out.toByteArray());
    }

    /** The first certificate of a PEM file, or the DER certificate a file is. */
    static Certificate certificate(Path file) throws Exception {
        byte[] bytes = Files.readAllBytes(file);
        String text = new String(bytes, StandardCharsets.ISO_8859_1);
        if (text.contains("-----BEGIN")) {
            try (PEMParser parser = new PEMParser(new StringReader(text))) {
                for (Object object = parser.readObject(); object != null; object = parser.readObject()) {
                    if (object instanceof X509CertificateHolder) {
                        return new JcaX509CertificateConverter().setProvider(BC)
                                .getCertificate((X509CertificateHolder) object);
                    }
                }
            }
            throw new IllegalArgumentException(file + ": no certificate in it");
        }
        return CertificateFactory.getInstance("X.509").generateCertificate(new ByteArrayInputStream(bytes));
    }

    /** The first private key of a PEM file, decrypted with the password if it is encrypted. */
    static PrivateKey privateKey(Path file, String password) throws Exception {
        byte[] bytes = Files.readAllBytes(file);
        String text = new String(bytes, StandardCharsets.ISO_8859_1);
        JcaPEMKeyConverter converter = new JcaPEMKeyConverter().setProvider(BC);
        if (!text.contains("-----BEGIN")) {
            return converter.getPrivateKey(PrivateKeyInfo.getInstance(bytes));
        }
        try (PEMParser parser = new PEMParser(new StringReader(text))) {
            for (Object object = parser.readObject(); object != null; object = parser.readObject()) {
                if (object instanceof PKCS8EncryptedPrivateKeyInfo) {
                    PrivateKeyInfo info = ((PKCS8EncryptedPrivateKeyInfo) object).decryptPrivateKeyInfo(
                            new JceOpenSSLPKCS8DecryptorProviderBuilder().setProvider(BC)
                                    .build(password.toCharArray()));
                    return converter.getPrivateKey(info);
                }
                if (object instanceof PrivateKeyInfo) {
                    return converter.getPrivateKey((PrivateKeyInfo) object);
                }
                if (object instanceof PEMKeyPair) {
                    return converter.getKeyPair((PEMKeyPair) object).getPrivate();
                }
            }
        }
        throw new IllegalArgumentException(file + ": no private key in it");
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

    /**
     * What the digest column holds for a text: the first eight bytes of the
     * SHA-256 of its UTF-8, in hex, as go/cmd/corpus writes it.
     */
    static String digest(String text) throws Exception {
        byte[] sum = MessageDigest.getInstance("SHA-256").digest(text.getBytes(StandardCharsets.UTF_8));
        StringBuilder hex = new StringBuilder();
        for (int i = 0; i < 8; i++) {
            hex.append(Character.forDigit((sum[i] >> 4) & 15, 16)).append(Character.forDigit(sum[i] & 15, 16));
        }
        return hex.toString();
    }

    /**
     * Every page of one way of opening one file, as the rows of a page table:
     * page, text, chars and a digest of that page's text. The digest column of
     * the document table says two extractions of the same length differ; this
     * says which page they differ on. go/cmd/corpus -pages writes the same
     * table for the port, and -comparepages joins the two.
     */
    static List<String[]> pagesOf(String path, Open open) {
        List<String[]> rows = new ArrayList<>();
        PDDocument doc = null;
        try {
            doc = load(path, open);
            int pages = doc.getNumberOfPages();
            for (int page = 1; page <= pages; page++) {
                String text = "ok";
                int chars = 0;
                String digest = "-";
                try {
                    PDFTextStripper stripper = new PDFTextStripper();
                    stripper.setStartPage(page);
                    stripper.setEndPage(page);
                    if (LF) {
                        stripper.setLineSeparator(LFS);
                        stripper.setPageEnd(LFS);
                    }
                    StringWriter out = new StringWriter();
                    stripper.writeText(doc, out);
                    chars = out.toString().codePointCount(0, out.toString().length());
                    digest = digest(out.toString());
                } catch (Throwable t) {
                    text = shorten(t);
                }
                rows.add(new String[] { String.valueOf(page), text, String.valueOf(chars), digest });
            }
        } catch (Throwable t) {
            rows.add(new String[] { "0", shorten(t), "0", "-" });
        } finally {
            if (doc != null) {
                try {
                    doc.close();
                } catch (Throwable ignored) {
                    // closing is not what is being measured
                }
            }
        }
        return rows;
    }

    /** Opens one file the way one job says to. */
    static PDDocument load(String path, Open open) throws Exception {
        if (open == null) {
            return Loader.loadPDF(new File(path));
        }
        if (open.certificate == null && open.keystore == null) {
            return Loader.loadPDF(new File(path), open.password);
        }
        // The keystore is built inside the fence, so a key that cannot be read
        // is this row's failure rather than the run's.
        return Loader.loadPDF(new File(path), open.password, keyStoreFor(open), null);
    }

    static String[] score(String path, Open open) {
        String result = "ok";
        int pages = 0;
        String text = "-";
        int chars = 0;
        String digest = "-";

        PDDocument doc = null;
        try {
            // A key that cannot be read is this row's failure rather than the
            // run's, and go/cmd/corpus fails the same line the same way.
            doc = load(path, open);
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
                digest = digest(out.toString());
            } catch (Throwable t) {
                text = shorten(t);
            }
        } catch (Throwable t) {
            result = shorten(t);
        } finally {
            if (doc != null) {
                try {
                    doc.close();
                } catch (Throwable ignored) {
                    // closing is not what is being measured
                }
            }
        }
        return new String[] { result, String.valueOf(pages), text, String.valueOf(chars), digest };
    }

    /** The row a job that timed out or threw gets, in the shape of the table being written. */
    static String[] failedRow(boolean perPage, int columns, boolean lines, String why) {
        if (perPage) {
            return new String[] { "0", why, "0", "-" };
        }
        if (lines) {
            return new String[] { "open", why };
        }
        if (columns > 0) {
            String[] row = new String[columns + 1];
            java.util.Arrays.fill(row, "-");
            row[0] = why;
            return row;
        }
        return new String[] { why, "0", "-", "0", "-" };
    }

    public static void main(String[] args) throws Exception {
        List<String> files = Files.readAllLines(Paths.get(args[0]));
        files.removeIf(String::isBlank);
        files.sort(Comparator.naturalOrder());

        long timeoutSeconds = args.length > 1 ? Long.parseLong(args[1]) : 20;
        LF = args.length > 2 && args[2].equals("lf");
        // args[3] says which table to write: "rows", the one row per file the
        // oracle joins; "pages", a digest per page for narrowing one of its
        // findings to a page; "facets", a digest per facet of the document
        // beyond its text (JavaFacets); or "facetlines", every line those
        // digests are made of, for narrowing a facet that differs. The
        // passwords tables follow it.
        String mode = args.length > 3 ? args[3] : "rows";
        boolean perPage = mode.equals("pages");
        boolean facets = mode.equals("facets");
        boolean facetLines = mode.equals("facetlines");
        // "writes" and "writelines" are the same for JavaWrites: what PDFBox writes,
        // read back.
        boolean writes = mode.equals("writes");
        boolean writeLines = mode.equals("writelines");
        int columns = facets ? JavaFacets.NAMES.length : writes ? JavaWrites.NAMES.length : 0;
        boolean lines = facetLines || writeLines;
        // "render" draws every page, JavaRender.
        boolean render = mode.equals("render");
        for (int t = 4; t < args.length; t++) {
            readTable(args[t]);
        }

        // One job per way of opening a file: a file no table names is opened
        // once with no password, and a file the tables name once for every line.
        List<String> names = new ArrayList<>();
        List<String> paths = new ArrayList<>();
        List<Open> jobs = new ArrayList<>();
        for (String path : files) {
            List<Open> opens = opensFor(path);
            if (opens.isEmpty()) {
                names.add(path);
                paths.add(path);
                jobs.add(null);
                continue;
            }
            for (Open open : opens) {
                names.add(opens.size() == 1 ? path : path + " [" + open.label + "]");
                paths.add(path);
                jobs.add(open);
            }
        }

        // Daemon threads, so a file that will not return cannot keep the JVM
        // alive. Each one gets its own executor because the previous one may
        // still be stuck in it.
        // The table goes out in UTF-8, whatever the console's code page: a row
        // is named by its file and by the password line that opened it, and
        // either can be written in any script.
        PrintStream table = new PrintStream(new FileOutputStream(FileDescriptor.out), true, StandardCharsets.UTF_8);
        table.println(render ? "file\tpage\tstatus\twidth\theight\texact\tgrid"
                : perPage ? "file\tpage\ttext\tchars\tdigest"
                : facets ? "file\topen\t" + String.join("\t", JavaFacets.NAMES)
                : writes ? "file\topen\t" + String.join("\t", JavaWrites.NAMES)
                : lines ? "file\tfacet\tline"
                : "file\topen\tpages\ttext\tchars\tdigest");
        List<Future<List<String[]>>> stuck = new ArrayList<>();
        for (int i = 0; i < jobs.size(); i++) {
            String path = paths.get(i);
            Open open = jobs.get(i);
            System.err.print("\r" + (i + 1) + "/" + jobs.size() + " " + new File(path).getName() + "                    ");

            ExecutorService pool = Executors.newSingleThreadExecutor(r -> {
                Thread t = new Thread(r);
                t.setDaemon(true);
                return t;
            });
            List<String[]> rows;
            Future<List<String[]>> future = null;
            try {
                future = pool.submit(render
                        ? (Callable<List<String[]>>) () -> JavaRender.renderOf(path, open)
                        : perPage
                        ? (Callable<List<String[]>>) () -> pagesOf(path, open)
                        : facets || facetLines
                        ? (Callable<List<String[]>>) () -> JavaFacets.facetsOf(path, open, facetLines)
                        : writes || writeLines
                        ? (Callable<List<String[]>>) () -> JavaWrites.writesOf(path, open, writeLines)
                        : (Callable<List<String[]>>) () -> List.<String[]>of(score(path, open)));
                rows = future.get(timeoutSeconds, TimeUnit.SECONDS);
            } catch (TimeoutException e) {
                rows = List.<String[]>of(render ? new String[] { "0", "timeout", "0", "0", "-", "-" }
                        : failedRow(perPage, columns, lines, "timeout"));
                stuck.add(future);
            } catch (Throwable t) {
                rows = List.<String[]>of(render ? new String[] { "0", shorten(t), "0", "0", "-", "-" }
                        : failedRow(perPage, columns, lines, shorten(t)));
            } finally {
                pool.shutdownNow();
            }
            for (String[] row : rows) {
                table.println(names.get(i) + "\t" + String.join("\t", row));
            }
            table.flush();

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
                        + "running; stopping at " + (i + 1) + " of " + jobs.size() + ".");
                System.err.println("The table above this line is complete. Narrow the list and run the rest "
                        + "separately, or raise the timeout if these files are slow rather than stuck.");
                System.exit(3);
            }
        }
        System.err.println();
    }
}
