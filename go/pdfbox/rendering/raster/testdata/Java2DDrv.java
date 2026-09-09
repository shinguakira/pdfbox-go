import java.awt.*;
import java.awt.geom.*;
import java.awt.image.BufferedImage;

/**
 * What java.awt.Graphics2D actually draws, for the cases the Go backend has to
 * reproduce. PDFBox has no rendering test of its own; Graphics2D is what the
 * port is a substitution for, so it is the reference.
 *
 * Run it to regenerate java2d.txt:
 *
 *   javac -d . Java2DDrv.java
 *   java -Djava.awt.headless=true -cp . Java2DDrv > java2d.txt
 *
 * Every case is drawn twice, under the two values of KEY_STROKE_CONTROL:
 *
 *   name       VALUE_STROKE_PURE, which renders the geometry as given
 *   nameNorm   VALUE_STROKE_NORMALIZE, which is what PDFBox gets -- see
 *              PDFRenderer.createDefaultRenderingHints, which sets three hints
 *              and not this one, so the default applies, and the default is
 *              NORMALIZE
 *
 * The other hints are PDFRenderer's, non-bitonal, copied from that method.
 *
 * Each case prints a grid of the red channel in hex, one row per line.
 */
public class Java2DDrv
{
    static void dump(String name, BufferedImage img)
    {
        System.out.println("### " + name + " " + img.getWidth() + "x" + img.getHeight());
        for (int y = 0; y < img.getHeight(); y++)
        {
            StringBuilder sb = new StringBuilder();
            for (int x = 0; x < img.getWidth(); x++)
            {
                sb.append(String.format("%02x", (img.getRGB(x, y) >> 16) & 0xFF));
            }
            System.out.println(sb);
        }
    }

    /** A white page with PDFRenderer's hints on it. */
    static Graphics2D graphics(BufferedImage img, boolean antiAlias, Object strokeControl)
    {
        Graphics2D g = img.createGraphics();
        g.setColor(Color.WHITE);
        g.fillRect(0, 0, img.getWidth(), img.getHeight());
        RenderingHints r = new RenderingHints(null);
        r.put(RenderingHints.KEY_INTERPOLATION, RenderingHints.VALUE_INTERPOLATION_BICUBIC);
        r.put(RenderingHints.KEY_RENDERING, RenderingHints.VALUE_RENDER_QUALITY);
        r.put(RenderingHints.KEY_ANTIALIASING, antiAlias
                ? RenderingHints.VALUE_ANTIALIAS_ON : RenderingHints.VALUE_ANTIALIAS_OFF);
        g.addRenderingHints(r);
        g.setRenderingHint(RenderingHints.KEY_STROKE_CONTROL, strokeControl);
        g.setColor(Color.BLACK);
        return g;
    }

    /** What one case draws, given a graphics to draw it on. */
    interface Case
    {
        void draw(Graphics2D g);
    }

    /** Draws a case under both stroke controls and dumps each. */
    static void both(String name, int w, int h, boolean antiAlias, Case c)
    {
        for (Object control : new Object[] { RenderingHints.VALUE_STROKE_PURE,
                RenderingHints.VALUE_STROKE_NORMALIZE })
        {
            BufferedImage img = new BufferedImage(w, h, BufferedImage.TYPE_INT_RGB);
            Graphics2D g = graphics(img, antiAlias, control);
            c.draw(g);
            g.dispose();
            dump(control == RenderingHints.VALUE_STROKE_PURE ? name : name + "Norm", img);
        }
    }

    /** The line the caps and the dashes are drawn on. */
    static Path2D.Double line(double x0, double y0, double x1, double y1)
    {
        Path2D.Double p = new Path2D.Double();
        p.moveTo(x0, y0);
        p.lineTo(x1, y1);
        return p;
    }

    public static void main(String[] args) throws Exception
    {
        // 1. a filled rectangle
        both("fillRect", 20, 20, false, g -> g.fill(new Rectangle2D.Double(5, 5, 10, 10)));

        // 2. the same rectangle through a translation
        both("fillTranslated", 20, 20, false, g ->
        {
            g.translate(5, 5);
            g.fill(new Rectangle2D.Double(0, 0, 5, 5));
        });

        // 3. clipped
        both("fillClipped", 20, 20, false, g ->
        {
            g.clip(new Rectangle2D.Double(0, 0, 10, 20));
            g.fill(new Rectangle2D.Double(5, 5, 10, 10));
        });

        // 4. a fill on half-pixel boundaries, anti-aliased, which is where a
        // fill would show a normalization if fills were normalized
        both("fillHalfAA", 20, 20, true, g -> g.fill(new Rectangle2D.Double(5.5, 5.5, 9, 9)));

        // 5. a stroked line, width 4, butt cap, miter join
        both("strokeLine", 20, 20, false, g ->
        {
            g.setStroke(new BasicStroke(4, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 10));
            g.draw(line(2, 10, 18, 10));
        });

        // 6. the winding rules, over a square with a square inside it
        for (int rule : new int[] { Path2D.WIND_NON_ZERO, Path2D.WIND_EVEN_ODD })
        {
            both(rule == Path2D.WIND_NON_ZERO ? "windNonZero" : "windEvenOdd", 20, 20, false, g ->
            {
                Path2D.Double p = new Path2D.Double(rule);
                p.moveTo(2, 2);
                p.lineTo(18, 2);
                p.lineTo(18, 18);
                p.lineTo(2, 18);
                p.closePath();
                p.moveTo(6, 6);
                p.lineTo(14, 6);
                p.lineTo(14, 14);
                p.lineTo(6, 14);
                p.closePath();
                g.fill(p);
            });
        }

        // 7. an anti-aliased right angle, to see the joins
        for (int join : new int[] { BasicStroke.JOIN_MITER, BasicStroke.JOIN_ROUND,
                BasicStroke.JOIN_BEVEL })
        {
            both(join == BasicStroke.JOIN_MITER ? "joinMiter"
                    : join == BasicStroke.JOIN_ROUND ? "joinRound" : "joinBevel",
                    32, 32, true, g ->
            {
                g.setStroke(new BasicStroke(8, BasicStroke.CAP_BUTT, join, 10));
                Path2D.Double p = new Path2D.Double();
                p.moveTo(0, 20);
                p.lineTo(20, 20);
                p.lineTo(20, 0);
                g.draw(p);
            });
        }

        // 8. the caps, on a short horizontal line, anti-aliased. The line runs
        // from x=8 to x=16 so a square or a round cap has room to show.
        for (int cap : new int[] { BasicStroke.CAP_BUTT, BasicStroke.CAP_ROUND,
                BasicStroke.CAP_SQUARE })
        {
            both(cap == BasicStroke.CAP_BUTT ? "capButt"
                    : cap == BasicStroke.CAP_ROUND ? "capRound" : "capSquare",
                    24, 24, true, g ->
            {
                g.setStroke(new BasicStroke(8, cap, BasicStroke.JOIN_MITER, 10));
                g.draw(line(8, 12, 16, 12));
            });
        }

        // 9. a dashed line: 4 on, 4 off, no phase
        both("dashed", 24, 12, false, g ->
        {
            g.setStroke(new BasicStroke(4, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 10,
                    new float[] { 4, 4 }, 0));
            g.draw(line(0, 6, 24, 6));
        });

        // 10. the same dashes with a phase of 2, which shifts where they start
        both("dashedPhase", 24, 12, false, g ->
        {
            g.setStroke(new BasicStroke(4, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 10,
                    new float[] { 4, 4 }, 2));
            g.draw(line(0, 6, 24, 6));
        });

        // 11. a miter that exceeds the limit, which Java draws as a bevel
        both("miterLimited", 40, 24, true, g ->
        {
            g.setStroke(new BasicStroke(6, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 2));
            Path2D.Double p = new Path2D.Double();
            p.moveTo(2, 20);
            p.lineTo(20, 4);
            p.lineTo(38, 20);
            g.draw(p);
        });

        // 12. a stroked curve, which is where the flattener shows
        both("strokeCurve", 32, 32, true, g ->
        {
            g.setStroke(new BasicStroke(5, BasicStroke.CAP_BUTT, BasicStroke.JOIN_ROUND, 10));
            Path2D.Double p = new Path2D.Double();
            p.moveTo(4, 26);
            p.curveTo(4, 4, 28, 4, 28, 26);
            g.draw(p);
        });
    }
}
