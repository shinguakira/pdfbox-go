import java.awt.image.BufferedImage;
import java.io.Writer;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Collections;
import java.util.IdentityHashMap;
import java.util.List;
import java.util.Map;
import java.util.function.Consumer;

import org.apache.pdfbox.cos.COSArray;
import org.apache.pdfbox.cos.COSBase;
import org.apache.pdfbox.cos.COSBoolean;
import org.apache.pdfbox.cos.COSDictionary;
import org.apache.pdfbox.cos.COSFloat;
import org.apache.pdfbox.cos.COSInteger;
import org.apache.pdfbox.cos.COSName;
import org.apache.pdfbox.cos.COSNull;
import org.apache.pdfbox.cos.COSStream;
import org.apache.pdfbox.cos.COSString;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.pdmodel.PDDocumentCatalog;
import org.apache.pdfbox.pdmodel.PDPage;
import org.apache.pdfbox.pdmodel.PDResources;
import org.apache.pdfbox.pdmodel.common.PDMetadata;
import org.apache.pdfbox.pdmodel.common.PDPageLabels;
import org.apache.pdfbox.pdmodel.common.PDRectangle;
import org.apache.pdfbox.pdmodel.documentinterchange.logicalstructure.PDMarkedContentReference;
import org.apache.pdfbox.pdmodel.documentinterchange.logicalstructure.PDObjectReference;
import org.apache.pdfbox.pdmodel.documentinterchange.logicalstructure.PDStructureElement;
import org.apache.pdfbox.pdmodel.documentinterchange.logicalstructure.PDStructureTreeRoot;
import org.apache.pdfbox.pdmodel.graphics.PDXObject;
import org.apache.pdfbox.pdmodel.graphics.form.PDFormXObject;
import org.apache.pdfbox.pdmodel.graphics.image.PDImageXObject;
import org.apache.pdfbox.pdmodel.interactive.annotation.PDAnnotation;
import org.apache.pdfbox.pdmodel.interactive.annotation.PDAppearanceDictionary;
import org.apache.pdfbox.pdmodel.interactive.documentnavigation.destination.PDDestination;
import org.apache.pdfbox.pdmodel.interactive.documentnavigation.destination.PDNamedDestination;
import org.apache.pdfbox.pdmodel.interactive.documentnavigation.destination.PDPageDestination;
import org.apache.pdfbox.pdmodel.interactive.documentnavigation.outline.PDDocumentOutline;
import org.apache.pdfbox.pdmodel.interactive.documentnavigation.outline.PDOutlineItem;
import org.apache.pdfbox.pdmodel.interactive.documentnavigation.outline.PDOutlineNode;
import org.apache.pdfbox.pdmodel.interactive.form.PDAcroForm;
import org.apache.pdfbox.pdmodel.interactive.form.PDCheckBox;
import org.apache.pdfbox.pdmodel.interactive.form.PDChoice;
import org.apache.pdfbox.pdmodel.interactive.form.PDComboBox;
import org.apache.pdfbox.pdmodel.interactive.form.PDField;
import org.apache.pdfbox.pdmodel.interactive.form.PDListBox;
import org.apache.pdfbox.pdmodel.interactive.form.PDNonTerminalField;
import org.apache.pdfbox.pdmodel.interactive.form.PDPushButton;
import org.apache.pdfbox.pdmodel.interactive.form.PDRadioButton;
import org.apache.pdfbox.pdmodel.interactive.form.PDSignatureField;
import org.apache.pdfbox.pdmodel.interactive.form.PDTerminalField;
import org.apache.pdfbox.pdmodel.interactive.form.PDTextField;
import org.apache.pdfbox.text.PDFTextStripper;
import org.apache.pdfbox.text.TextPosition;
import org.apache.xmpbox.XMPMetadata;
import org.apache.xmpbox.schema.XMPSchema;
import org.apache.xmpbox.xml.DomXmpParser;

/**
 * What PDFBox makes of a document beyond its text, as digests go/cmd/corpus
 * -facets computes the same way.
 *
 * <p>The row table compares whether a document opens, its page count and its
 * text. Everything else the port implements -- where each glyph sits, the
 * document information and XMP, the outline, page labels, page boxes, the
 * structure tree, annotations, form fields, and the images and what they decode
 * to -- was compared by nothing. Each facet here is a sequence of lines, written
 * the same way on both sides, digested; a facet whose digests differ is one the
 * two implementations do not agree on, and -facetlines writes the lines
 * themselves so the difference can be found.
 *
 * <p>A facet cell is {@code <lines>:<digest>}, {@code -} where the document has
 * no such thing, or {@code error} where computing it threw. The error is not
 * named: the two languages word their failures differently, and what is
 * compared is whether each side failed.
 *
 * <p>Floats are written as the bits of the float, so a difference in the last
 * bit is a difference. Strings are escaped so that a line is one line.
 */
public final class JavaFacets {

    /** The facet columns, in the order they are computed and written. */
    static final String[] NAMES = {
        "positions", "info", "xmp", "xmpschemas", "outline", "labels", "boxes", "struct",
        "annots", "fields", "images", "imagepixels"
    };

    /** An image larger than this is not converted to pixels; its cell says so. */
    static final long MAX_PIXELS = 40_000_000L;

    /** Outline, structure and field walks stop here, on a file that loops. */
    static final int MAX_ITEMS = 200_000;

    private JavaFacets() {
    }

    /** One facet's lines, digested as they arrive, and kept when dumping. */
    static final class Facet {
        final MessageDigest digest;
        final List<String> kept;
        int lines;

        Facet(boolean keep) {
            try {
                digest = MessageDigest.getInstance("SHA-256");
            } catch (Exception e) {
                throw new IllegalStateException(e);
            }
            kept = keep ? new ArrayList<>() : null;
        }

        void line(String line) {
            digest.update((line + "\n").getBytes(StandardCharsets.UTF_8));
            lines++;
            if (kept != null) {
                kept.add(line);
            }
        }

        void bytes(byte[] data) {
            digest.update(data);
            lines += data.length;
            if (kept != null) {
                kept.add("bytes " + data.length);
            }
        }

        String cell() {
            StringBuilder hex = new StringBuilder(lines + ":");
            byte[] sum = digest.digest();
            for (int i = 0; i < 8; i++) {
                hex.append(String.format("%02x", sum[i] & 0xff));
            }
            return hex.toString();
        }
    }

    /** Why a facet has no value: the document has no such thing. */
    static final class Absent extends Exception {
        Absent() {
            super(null, null, false, false);
        }
    }

    interface Computation {
        void compute(PDDocument doc, Facet facet) throws Exception;
    }

    /**
     * A string as a facet line writes it. Null is written as the empty string:
     * the port's getters answer "" where Java's answer null, so the distinction
     * cannot be compared, and writing it would make every such line differ.
     * Where both sides can tell absence apart -- a destination, an action, an
     * appearance state, a date -- the line writes {@code <null>} itself.
     */
    static String s(String value) {
        if (value == null) {
            return "";
        }
        StringBuilder out = new StringBuilder(value.length());
        for (int i = 0; i < value.length(); i++) {
            char c = value.charAt(i);
            switch (c) {
                case '\\':
                    out.append("\\\\");
                    break;
                case '\n':
                    out.append("\\n");
                    break;
                case '\r':
                    out.append("\\r");
                    break;
                case '\t':
                    out.append("\\t");
                    break;
                default:
                    out.append(c);
            }
        }
        return out.toString();
    }

    static String f(float value) {
        return String.format("%08x", Float.floatToRawIntBits(value));
    }

    static String rect(PDRectangle r) {
        if (r == null) {
            return "<null>";
        }
        return f(r.getLowerLeftX()) + "," + f(r.getLowerLeftY()) + "," + f(r.getUpperRightX()) + ","
                + f(r.getUpperRightY());
    }

    static String calendar(Calendar c) {
        if (c == null) {
            return "<null>";
        }
        return c.getTimeInMillis() + "|" + c.getTimeZone().getRawOffset();
    }

    /** A COS value as a facet line writes it: its kind, and for a scalar its value. */
    static String cos(COSBase value) {
        if (value == null || value instanceof COSNull) {
            return "null";
        }
        if (value instanceof COSString) {
            return "str:" + s(((COSString) value).getString());
        }
        if (value instanceof COSName) {
            return "name:" + s(((COSName) value).getName());
        }
        if (value instanceof COSInteger) {
            return "int:" + ((COSInteger) value).longValue();
        }
        if (value instanceof COSFloat) {
            return "float:" + f(((COSFloat) value).floatValue());
        }
        if (value instanceof COSBoolean) {
            return "bool:" + ((COSBoolean) value).getValue();
        }
        if (value instanceof COSStream) {
            return "stream";
        }
        if (value instanceof COSDictionary) {
            return "dict";
        }
        if (value instanceof COSArray) {
            return "array";
        }
        return "other";
    }

    // ------------------------------------------------------------------ facets

    static final class PositionStripper extends PDFTextStripper {
        private final Facet facet;

        PositionStripper(Facet facet) {
            this.facet = facet;
        }

        @Override
        protected void processTextPosition(TextPosition t) {
            StringBuilder codes = new StringBuilder();
            for (int code : t.getCharacterCodes()) {
                if (codes.length() > 0) {
                    codes.append(',');
                }
                codes.append(code);
            }
            facet.line("u=" + s(t.getUnicode()) + " c=" + codes + " x=" + f(t.getXDirAdj()) + " y="
                    + f(t.getYDirAdj()) + " w=" + f(t.getWidthDirAdj()) + " h=" + f(t.getHeightDir())
                    + " fs=" + f(t.getFontSizeInPt()) + " font="
                    + s(t.getFont() == null ? null : t.getFont().getName()));
            super.processTextPosition(t);
        }
    }

    static void positions(PDDocument doc, Facet facet) throws Exception {
        PositionStripper stripper = new PositionStripper(facet);
        stripper.setLineSeparator("\n");
        stripper.setPageEnd("\n");
        stripper.writeText(doc, Writer.nullWriter());
    }

    static void info(PDDocument doc, Facet facet) throws Exception {
        if (doc.getDocument().getTrailer().getCOSDictionary(COSName.INFO) == null) {
            throw new Absent();
        }
        COSDictionary dictionary = doc.getDocumentInformation().getCOSObject();
        List<COSName> keys = new ArrayList<>(dictionary.keySet());
        keys.sort((a, b) -> a.getName().compareTo(b.getName()));
        for (COSName key : keys) {
            facet.line(s(key.getName()) + "=" + cos(dictionary.getDictionaryObject(key)));
        }
        facet.line("CreationDate.parsed=" + calendar(doc.getDocumentInformation().getCreationDate()));
        facet.line("ModDate.parsed=" + calendar(doc.getDocumentInformation().getModificationDate()));
    }

    static byte[] xmpBytes(PDDocument doc) throws Exception {
        PDMetadata metadata = doc.getDocumentCatalog().getMetadata();
        if (metadata == null) {
            throw new Absent();
        }
        return metadata.toByteArray();
    }

    static void xmp(PDDocument doc, Facet facet) throws Exception {
        facet.bytes(xmpBytes(doc));
    }

    static void xmpSchemas(PDDocument doc, Facet facet) throws Exception {
        XMPMetadata metadata = new DomXmpParser().parse(xmpBytes(doc));
        List<String> namespaces = new ArrayList<>();
        for (XMPSchema schema : metadata.getAllSchemas()) {
            namespaces.add(s(schema.getNamespace()));
        }
        Collections.sort(namespaces);
        namespaces.forEach(facet::line);
    }

    static void outline(PDDocument doc, Facet facet) throws Exception {
        PDDocumentOutline outline = doc.getDocumentCatalog().getDocumentOutline();
        if (outline == null) {
            throw new Absent();
        }
        walkOutline(outline, 0, facet, new IdentityHashMap<>());
    }

    private static void walkOutline(PDOutlineNode node, int depth, Facet facet,
            Map<COSDictionary, Boolean> seen) throws Exception {
        for (PDOutlineItem item = node.getFirstChild(); item != null; item = item.getNextSibling()) {
            if (seen.put(item.getCOSObject(), Boolean.TRUE) != null || facet.lines >= MAX_ITEMS) {
                facet.line("stop");
                return;
            }
            String destination;
            try {
                PDDestination d = item.getDestination();
                if (d == null) {
                    destination = "<null>";
                } else if (d instanceof PDPageDestination) {
                    destination = "page:" + ((PDPageDestination) d).retrievePageNumber();
                } else if (d instanceof PDNamedDestination) {
                    destination = "named:" + s(((PDNamedDestination) d).getNamedDestination());
                } else {
                    destination = "other";
                }
            } catch (Exception e) {
                destination = "error";
            }
            String action = item.getAction() == null ? "<null>" : s(item.getAction().getSubType());
            facet.line("d=" + depth + " t=" + s(item.getTitle()) + " open=" + item.isNodeOpen() + " dest="
                    + destination + " action=" + action);
            walkOutline(item, depth + 1, facet, seen);
        }
    }

    static void labels(PDDocument doc, Facet facet) throws Exception {
        PDPageLabels labels = doc.getDocumentCatalog().getPageLabels();
        if (labels == null) {
            throw new Absent();
        }
        String[] byIndex = labels.getLabelsByPageIndices();
        for (int i = 0; i < byIndex.length; i++) {
            facet.line(i + "=" + s(byIndex[i]));
        }
    }

    static void boxes(PDDocument doc, Facet facet) throws Exception {
        int i = 0;
        for (PDPage page : doc.getPages()) {
            facet.line("p" + i + " m=" + rect(page.getMediaBox()) + " c=" + rect(page.getCropBox()) + " b="
                    + rect(page.getBleedBox()) + " t=" + rect(page.getTrimBox()) + " a="
                    + rect(page.getArtBox()) + " r=" + page.getRotation());
            i++;
        }
    }

    static void struct(PDDocument doc, Facet facet) throws Exception {
        PDStructureTreeRoot root = doc.getDocumentCatalog().getStructureTreeRoot();
        if (root == null) {
            throw new Absent();
        }
        walkStructure(root.getKids(), 0, facet, new IdentityHashMap<>());
    }

    private static void walkStructure(List<Object> kids, int depth, Facet facet, Map<Object, Boolean> seen) {
        for (Object kid : kids) {
            if (facet.lines >= MAX_ITEMS) {
                facet.line("stop");
                return;
            }
            if (kid instanceof PDStructureElement) {
                PDStructureElement element = (PDStructureElement) kid;
                if (seen.put(element.getCOSObject(), Boolean.TRUE) != null) {
                    facet.line("d=" + depth + " cycle");
                    continue;
                }
                facet.line("d=" + depth + " S=" + s(element.getStructureType()) + " std="
                        + s(element.getStandardStructureType()) + " alt=" + s(element.getAlternateDescription())
                        + " actual=" + s(element.getActualText()) + " title=" + s(element.getTitle()) + " lang="
                        + s(element.getLanguage()));
                walkStructure(element.getKids(), depth + 1, facet, seen);
            } else if (kid instanceof Integer) {
                facet.line("d=" + depth + " mcid=" + kid);
            } else if (kid instanceof PDMarkedContentReference) {
                facet.line("d=" + depth + " mcr=" + ((PDMarkedContentReference) kid).getMCID());
            } else if (kid instanceof PDObjectReference) {
                facet.line("d=" + depth + " objr");
            } else {
                facet.line("d=" + depth + " other");
            }
        }
    }

    static void annots(PDDocument doc, Facet facet) throws Exception {
        int i = 0;
        for (PDPage page : doc.getPages()) {
            for (PDAnnotation annotation : page.getAnnotations()) {
                COSName state = annotation.getAppearanceState();
                PDAppearanceDictionary appearance = annotation.getAppearance();
                boolean normal = appearance != null && appearance.getNormalAppearance() != null;
                facet.line("p" + i + " st=" + s(annotation.getSubtype()) + " r=" + rect(annotation.getRectangle())
                        + " c=" + s(annotation.getContents()) + " nm=" + s(annotation.getAnnotationName()) + " f="
                        + annotation.getAnnotationFlags() + " as=" + (state == null ? "<null>" : s(state.getName()))
                        + " ap=" + normal);
            }
            i++;
        }
    }

    static void fields(PDDocument doc, Facet facet) throws Exception {
        PDAcroForm form = doc.getDocumentCatalog().getAcroForm();
        if (form == null) {
            throw new Absent();
        }
        for (PDField field : form.getFieldTree()) {
            if (facet.lines >= MAX_ITEMS) {
                facet.line("stop");
                return;
            }
            String kind;
            String value;
            if (field instanceof PDNonTerminalField) {
                kind = "nonterminal";
                value = "-";
            } else if (field instanceof PDTextField) {
                kind = "text";
                value = s(((PDTextField) field).getValue());
            } else if (field instanceof PDCheckBox) {
                kind = "checkbox";
                value = s(((PDCheckBox) field).getValue());
            } else if (field instanceof PDRadioButton) {
                kind = "radio";
                value = s(((PDRadioButton) field).getValue());
            } else if (field instanceof PDPushButton) {
                kind = "pushbutton";
                value = "-";
            } else if (field instanceof PDComboBox || field instanceof PDListBox) {
                kind = field instanceof PDComboBox ? "combo" : "list";
                StringBuilder joined = new StringBuilder();
                for (String v : ((PDChoice) field).getValue()) {
                    if (joined.length() > 0) {
                        joined.append('|');
                    }
                    joined.append(s(v));
                }
                value = joined.toString();
            } else if (field instanceof PDSignatureField) {
                kind = "signature";
                value = ((PDSignatureField) field).getSignature() != null ? "present" : "absent";
            } else {
                kind = "other";
                value = "-";
            }
            String widgets = field instanceof PDTerminalField
                    ? String.valueOf(((PDTerminalField) field).getWidgets().size())
                    : "-";
            facet.line("n=" + s(field.getFullyQualifiedName()) + " ft=" + s(field.getFieldType()) + " ff="
                    + field.getFieldFlags() + " kind=" + kind + " w=" + widgets + " v=" + value);
        }
    }

    /** Every image XObject the pages reach, through form XObjects, once each, in page and name order. */
    static void eachImage(PDDocument doc, Consumer<PDImageXObject> visit) throws Exception {
        Map<COSStream, Boolean> seen = new IdentityHashMap<>();
        for (PDPage page : doc.getPages()) {
            eachImage(page.getResources(), visit, seen);
        }
    }

    private static void eachImage(PDResources resources, Consumer<PDImageXObject> visit,
            Map<COSStream, Boolean> seen) throws Exception {
        if (resources == null) {
            return;
        }
        List<COSName> names = new ArrayList<>();
        resources.getXObjectNames().forEach(names::add);
        names.sort((a, b) -> a.getName().compareTo(b.getName()));
        for (COSName name : names) {
            PDXObject xobject = resources.getXObject(name);
            if (xobject == null || seen.put(xobject.getCOSObject(), Boolean.TRUE) != null) {
                continue;
            }
            if (xobject instanceof PDImageXObject) {
                visit.accept((PDImageXObject) xobject);
            } else if (xobject instanceof PDFormXObject) {
                eachImage(((PDFormXObject) xobject).getResources(), visit, seen);
            }
        }
    }

    static void images(PDDocument doc, Facet facet) throws Exception {
        eachImage(doc, image -> {
            String colorSpace;
            try {
                colorSpace = s(image.getColorSpace().getName());
            } catch (Exception e) {
                colorSpace = "error";
            }
            StringBuilder filters = new StringBuilder();
            List<COSName> list = image.getStream().getFilters();
            for (COSName filter : list) {
                if (filters.length() > 0) {
                    filters.append(',');
                }
                filters.append(s(filter.getName()));
            }
            String data;
            try {
                Facet bytes = new Facet(false);
                bytes.bytes(image.getStream().toByteArray());
                data = bytes.cell();
            } catch (Exception e) {
                data = "error";
            }
            facet.line("w=" + image.getWidth() + " h=" + image.getHeight() + " bpc=" + image.getBitsPerComponent()
                    + " cs=" + colorSpace + " filters=" + filters + " stencil=" + image.isStencil() + " data="
                    + data);
        });
    }

    static void imagePixels(PDDocument doc, Facet facet) throws Exception {
        eachImage(doc, image -> {
            long pixels = (long) image.getWidth() * image.getHeight();
            if (pixels > MAX_PIXELS) {
                facet.line("skipped " + image.getWidth() + "x" + image.getHeight());
                return;
            }
            try {
                BufferedImage rendered = image.getImage();
                int width = rendered.getWidth();
                int height = rendered.getHeight();
                Facet argb = new Facet(false);
                int[] row = new int[width];
                byte[] bytes = new byte[width * 4];
                for (int y = 0; y < height; y++) {
                    rendered.getRGB(0, y, width, 1, row, 0, width);
                    for (int x = 0; x < width; x++) {
                        int p = row[x];
                        bytes[x * 4] = (byte) (p >>> 24);
                        bytes[x * 4 + 1] = (byte) (p >>> 16);
                        bytes[x * 4 + 2] = (byte) (p >>> 8);
                        bytes[x * 4 + 3] = (byte) p;
                    }
                    argb.digest.update(bytes);
                }
                argb.lines = width * height;
                facet.line(width + "x" + height + " " + argb.cell());
            } catch (Throwable e) {
                facet.line("error");
            }
        });
    }

    static final Computation[] COMPUTATIONS = {
        JavaFacets::positions, JavaFacets::info, JavaFacets::xmp, JavaFacets::xmpSchemas, JavaFacets::outline,
        JavaFacets::labels, JavaFacets::boxes, JavaFacets::struct, JavaFacets::annots, JavaFacets::fields,
        JavaFacets::images, JavaFacets::imagePixels
    };

    /**
     * The row -facets writes for one way of opening one file: whether it opened,
     * then one cell per facet. With {@code keep}, the second element holds every
     * facet's lines for -facetlines.
     */
    static List<String[]> facetsOf(String path, JavaCorpus.Open open, boolean keep) {
        List<String[]> out = new ArrayList<>();
        String[] row = new String[NAMES.length + 1];
        PDDocument doc = null;
        try {
            doc = JavaCorpus.load(path, open);
            row[0] = "ok";
            for (int i = 0; i < COMPUTATIONS.length; i++) {
                Facet facet = new Facet(keep);
                String cell;
                try {
                    COMPUTATIONS[i].compute(doc, facet);
                    cell = facet.cell();
                } catch (Absent absent) {
                    cell = "-";
                } catch (Throwable t) {
                    cell = "error";
                    if (keep) {
                        facet.kept.add("error " + s(JavaCorpus.shorten(t)));
                    }
                }
                row[i + 1] = cell;
                if (keep) {
                    for (String line : facet.kept) {
                        out.add(new String[] { NAMES[i], line });
                    }
                }
            }
        } catch (Throwable t) {
            row[0] = JavaCorpus.shorten(t);
            for (int i = 1; i < row.length; i++) {
                row[i] = "-";
            }
        } finally {
            if (doc != null) {
                try {
                    doc.close();
                } catch (Throwable ignored) {
                    // closing is not what is being measured
                }
            }
        }
        if (keep) {
            return out;
        }
        out.add(row);
        return out;
    }
}
