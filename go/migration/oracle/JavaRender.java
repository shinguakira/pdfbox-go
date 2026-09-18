import java.awt.image.BufferedImage;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.List;

import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.rendering.ImageType;
import org.apache.pdfbox.rendering.PDFRenderer;

/**
 * What PDFBox draws, page by page, as go/cmd/corpus -render writes it for the
 * port.
 *
 * <p>Each page is rendered at 72 dpi as RGB and written as a row: its size, a
 * digest of its pixels, and a 16 by 16 grid of the mean brightness of each cell.
 * The digest says whether two renderings are the same to the last bit, which two
 * rasterisers rarely are at the edges of what they draw. The grid says how far
 * apart they are where they are not: a cell is a mean over a sixteenth of the
 * page each way, so antialiasing washes out of it and a missing glyph, a wrong
 * colour or a shifted image does not.
 *
 * <p>Brightness is (299 R + 587 G + 114 B + 500) / 1000 and a cell's mean is the
 * integer quotient of its sum by its count, so both sides compute the same
 * number from the same pixels. A document is rendered up to {@link #MAX_PAGES}
 * pages.
 */
public final class JavaRender {

    static final int MAX_PAGES = 1000;
    static final int GRID = 16;

    private JavaRender() {
    }

    static String[] page(PDFRenderer renderer, int index) {
        try {
            BufferedImage image = renderer.renderImageWithDPI(index, 72, ImageType.RGB);
            int width = image.getWidth();
            int height = image.getHeight();
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            long[] sums = new long[GRID * GRID];
            long[] counts = new long[GRID * GRID];
            int[] row = new int[width];
            byte[] rgb = new byte[width * 3];
            for (int y = 0; y < height; y++) {
                image.getRGB(0, y, width, 1, row, 0, width);
                int gy = (int) ((long) y * GRID / height);
                for (int x = 0; x < width; x++) {
                    int p = row[x];
                    int r = (p >>> 16) & 0xff;
                    int g = (p >>> 8) & 0xff;
                    int b = p & 0xff;
                    rgb[x * 3] = (byte) r;
                    rgb[x * 3 + 1] = (byte) g;
                    rgb[x * 3 + 2] = (byte) b;
                    int gx = (int) ((long) x * GRID / width);
                    sums[gy * GRID + gx] += (299 * r + 587 * g + 114 * b + 500) / 1000;
                    counts[gy * GRID + gx]++;
                }
                digest.update(rgb);
            }
            StringBuilder grid = new StringBuilder(GRID * GRID * 2);
            for (int i = 0; i < GRID * GRID; i++) {
                long mean = counts[i] == 0 ? 0 : sums[i] / counts[i];
                grid.append(String.format("%02x", mean));
            }
            byte[] sum = digest.digest();
            StringBuilder hex = new StringBuilder();
            for (int i = 0; i < 8; i++) {
                hex.append(String.format("%02x", sum[i] & 0xff));
            }
            return new String[] { String.valueOf(index + 1), "ok", String.valueOf(width), String.valueOf(height),
                hex.toString(), grid.toString() };
        } catch (Throwable t) {
            return new String[] { String.valueOf(index + 1), "error", "0", "0", "-", "-" };
        }
    }

    /** The rows -render writes for one way of opening one file: one per page, or one for page 0 if it did not open. */
    static List<String[]> renderOf(String path, JavaCorpus.Open open) {
        List<String[]> rows = new ArrayList<>();
        PDDocument doc = null;
        try {
            doc = JavaCorpus.load(path, open);
            PDFRenderer renderer = new PDFRenderer(doc);
            int pages = Math.min(doc.getNumberOfPages(), MAX_PAGES);
            for (int i = 0; i < pages; i++) {
                rows.add(page(renderer, i));
            }
        } catch (Throwable t) {
            rows.add(new String[] { "0", JavaCorpus.shorten(t), "0", "0", "-", "-" });
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
}
