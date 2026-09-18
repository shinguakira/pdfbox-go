import java.io.ByteArrayOutputStream;
import java.io.StringWriter;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Calendar;
import java.util.HashMap;
import java.util.List;
import java.util.TimeZone;

import org.apache.pdfbox.Loader;
import org.apache.pdfbox.multipdf.Overlay;
import org.apache.pdfbox.multipdf.PDFMergerUtility;
import org.apache.pdfbox.multipdf.Splitter;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.pdmodel.PDDocumentInformation;
import org.apache.pdfbox.pdmodel.encryption.AccessPermission;
import org.apache.pdfbox.pdmodel.encryption.StandardProtectionPolicy;
import org.apache.pdfbox.pdmodel.interactive.digitalsignature.ExternalSigningSupport;
import org.apache.pdfbox.pdmodel.interactive.digitalsignature.PDSignature;
import org.apache.pdfbox.text.PDFTextStripper;

/**
 * What PDFBox writes, read back: the write paths the port implements, run on
 * each document and summarised the way go/cmd/corpus -writes summarises its own.
 *
 * <p>Each write facet loads the document afresh, does one thing to it, writes
 * it, loads what it wrote, and summarises that as its page count and a digest of
 * its text. The bytes of two writers are not expected to agree -- object
 * numbering, spacing and random encryption keys differ -- so they are not
 * compared; what each writer's output reads back as is. A document that was
 * opened with a password or a key is not written: the write facets are about
 * the writer, and encryption on writing has a facet of its own.
 *
 * <p>Cells are as JavaFacets writes them: {@code <lines>:<digest>}, {@code -}, or
 * {@code error}.
 */
public final class JavaWrites {

    static final String[] NAMES = { "save", "incremental", "encrypt", "split", "merge", "overlay", "sign" };

    /** Split documents beyond this many parts are counted, not summarised. */
    static final int MAX_PARTS = 200;

    private JavaWrites() {
    }

    interface Write {
        void compute(String path, JavaCorpus.Open open, JavaFacets.Facet facet) throws Exception;
    }

    static String summary(PDDocument doc) throws Exception {
        PDFTextStripper stripper = new PDFTextStripper();
        stripper.setLineSeparator("\n");
        stripper.setPageEnd("\n");
        StringWriter out = new StringWriter();
        stripper.writeText(doc, out);
        return "pages=" + doc.getNumberOfPages() + " text=" + JavaCorpus.digest(out.toString());
    }

    static byte[] save(PDDocument doc) throws Exception {
        ByteArrayOutputStream out = new ByteArrayOutputStream();
        doc.save(out);
        return out.toByteArray();
    }

    static void save(String path, JavaCorpus.Open open, JavaFacets.Facet facet) throws Exception {
        try (PDDocument doc = JavaCorpus.load(path, open)) {
            byte[] written = save(doc);
            try (PDDocument back = Loader.loadPDF(written)) {
                facet.line(summary(back));
            }
        }
    }

    static void incremental(String path, JavaCorpus.Open open, JavaFacets.Facet facet) throws Exception {
        try (PDDocument doc = JavaCorpus.load(path, open)) {
            PDDocumentInformation info = doc.getDocumentInformation();
            info.setTitle("corpus facet");
            info.getCOSObject().setNeedToBeUpdated(true);
            ByteArrayOutputStream out = new ByteArrayOutputStream();
            doc.saveIncremental(out);
            try (PDDocument back = Loader.loadPDF(out.toByteArray())) {
                facet.line(summary(back) + " title=" + JavaFacets.s(back.getDocumentInformation().getTitle()));
            }
        }
    }

    static void encrypt(String path, JavaCorpus.Open open, JavaFacets.Facet facet) throws Exception {
        try (PDDocument doc = JavaCorpus.load(path, open)) {
            StandardProtectionPolicy policy = new StandardProtectionPolicy("corpus-owner", "corpus-user",
                    new AccessPermission());
            policy.setEncryptionKeyLength(256);
            doc.protect(policy);
            byte[] written = save(doc);
            try (PDDocument back = Loader.loadPDF(written, "corpus-user")) {
                facet.line(summary(back) + " encrypted=" + back.isEncrypted());
            }
        }
    }

    static void split(String path, JavaCorpus.Open open, JavaFacets.Facet facet) throws Exception {
        try (PDDocument doc = JavaCorpus.load(path, open)) {
            List<PDDocument> parts = new Splitter().split(doc);
            try {
                facet.line("parts=" + parts.size());
                for (int i = 0; i < parts.size() && i < MAX_PARTS; i++) {
                    facet.line("part " + i + " " + summary(parts.get(i)));
                }
            } finally {
                for (PDDocument part : parts) {
                    part.close();
                }
            }
        }
    }

    static void merge(String path, JavaCorpus.Open open, JavaFacets.Facet facet) throws Exception {
        try (PDDocument first = JavaCorpus.load(path, open);
                PDDocument second = JavaCorpus.load(path, open);
                PDDocument destination = new PDDocument()) {
            PDFMergerUtility merger = new PDFMergerUtility();
            merger.appendDocument(destination, first);
            merger.appendDocument(destination, second);
            byte[] written = save(destination);
            try (PDDocument back = Loader.loadPDF(written)) {
                facet.line(summary(back));
            }
        }
    }

    static void overlay(String path, JavaCorpus.Open open, JavaFacets.Facet facet) throws Exception {
        try (PDDocument input = JavaCorpus.load(path, open); PDDocument over = JavaCorpus.load(path, open)) {
            Overlay overlay = new Overlay();
            overlay.setInputPDF(input);
            overlay.setDefaultOverlayPDF(over);
            PDDocument result = overlay.overlay(new HashMap<>());
            byte[] written = save(result);
            try (PDDocument back = Loader.loadPDF(written)) {
                facet.line(summary(back));
            }
        }
    }

    static void sign(String path, JavaCorpus.Open open, JavaFacets.Facet facet) throws Exception {
        try (PDDocument doc = JavaCorpus.load(path, open)) {
            PDSignature signature = new PDSignature();
            signature.setFilter(PDSignature.FILTER_ADOBE_PPKLITE);
            signature.setSubFilter(PDSignature.SUBFILTER_ADBE_PKCS7_DETACHED);
            signature.setName("corpus facet");
            Calendar date = Calendar.getInstance(TimeZone.getTimeZone("UTC"));
            date.setTimeInMillis(1767225600000L);
            signature.setSignDate(date);
            doc.addSignature(signature);
            ByteArrayOutputStream out = new ByteArrayOutputStream();
            ExternalSigningSupport external = doc.saveIncrementalForExternalSigning(out);
            byte[] handed = external.getContent().readAllBytes();
            external.setSignature(new byte[] { 0x30, 0x03, 0x02, 0x01, 0x00 });
            byte[] written = out.toByteArray();
            try (PDDocument back = Loader.loadPDF(written)) {
                PDSignature last = back.getLastSignatureDictionary();
                int[] range = last.getByteRange();
                boolean covers = range.length == 4 && range[0] == 0 && range[2] + range[3] == written.length;
                boolean same = Arrays.equals(handed, last.getSignedContent(written));
                facet.line(summary(back) + " signatures=" + back.getSignatureDictionaries().size() + " ranges="
                        + range.length + " covers=" + covers + " handed=" + same);
            }
        }
    }

    static final Write[] WRITES = {
        JavaWrites::save, JavaWrites::incremental, JavaWrites::encrypt, JavaWrites::split, JavaWrites::merge,
        JavaWrites::overlay, JavaWrites::sign
    };

    /** The row -writes writes for one way of opening one file. */
    static List<String[]> writesOf(String path, JavaCorpus.Open open, boolean keep) {
        List<String[]> out = new ArrayList<>();
        String[] row = new String[NAMES.length + 1];
        try (PDDocument probe = JavaCorpus.load(path, open)) {
            row[0] = "ok";
            if (open != null || probe.isEncrypted()) {
                Arrays.fill(row, 1, row.length, "-");
                if (keep) {
                    out.add(new String[] { "open", "encrypted, not written" });
                    return out;
                }
                out.add(row);
                return out;
            }
        } catch (Throwable t) {
            row[0] = JavaCorpus.shorten(t);
            Arrays.fill(row, 1, row.length, "-");
            if (keep) {
                out.add(new String[] { "open", JavaFacets.s(row[0]) });
                return out;
            }
            out.add(row);
            return out;
        }
        for (int i = 0; i < WRITES.length; i++) {
            JavaFacets.Facet facet = new JavaFacets.Facet(keep);
            String cell;
            try {
                WRITES[i].compute(path, open, facet);
                cell = facet.cell();
            } catch (Throwable t) {
                cell = "error";
                if (keep) {
                    facet.kept.add("error " + JavaFacets.s(JavaCorpus.shorten(t)));
                }
            }
            row[i + 1] = cell;
            if (keep) {
                for (String line : facet.kept) {
                    out.add(new String[] { NAMES[i], line });
                }
            }
        }
        if (!keep) {
            out.add(row);
        }
        return out;
    }
}
