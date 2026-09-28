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

        // 13. a stroke through a scale of 2, which widens it with everything
        // else: the line is 8 pixels thick. This is a page rendered at 144 dpi.
        both("strokeScaled", 24, 20, true, g ->
        {
            g.scale(2, 2);
            g.setStroke(new BasicStroke(4, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 10));
            g.draw(line(2, 5, 10, 5));
        });

        // 14. and through a scale of a half, 4 pixels thick
        both("strokeShrunk", 24, 20, true, g ->
        {
            g.scale(0.5, 0.5);
            g.setStroke(new BasicStroke(8, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 10));
            g.draw(line(4, 20, 44, 20));
        });

        // 15. the dashes of case 10 at half their size through a scale of 2,
        // which scales the dashes and the phase as well as the width
        both("dashedScaled", 24, 12, false, g ->
        {
            g.scale(2, 2);
            g.setStroke(new BasicStroke(2, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 10,
                    new float[] { 2, 2 }, 1));
            g.draw(line(0, 3, 12, 3));
        });

        // 16. a scale that is not the same both ways, through which the pen is
        // an ellipse: the horizontal arm is 4 pixels thick and the vertical one
        // 8 pixels wide
        both("strokeNonUniform", 32, 28, true, g ->
        {
            g.scale(2, 1);
            g.setStroke(new BasicStroke(4, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 10));
            Path2D.Double p = new Path2D.Double();
            p.moveTo(2, 6);
            p.lineTo(12, 6);
            p.lineTo(12, 24);
            g.draw(p);
        });

        // 17. a transform that flattens everything to a line, through which
        // nothing is stroked
        both("strokeSingular", 20, 20, true, g ->
        {
            g.scale(1, 0);
            g.setStroke(new BasicStroke(4, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 10));
            g.draw(line(2, 10, 18, 10));
        });

        // 18. images, drawn with drawImage(image, transform, null) under
        // PDFRenderer's bicubic hint. What Java2D does with one depends on the
        // transform: an image that lands one to one on whole pixels is copied,
        // and anything else goes through TransformHelper, which steps the
        // inverse transform in 32.32 fixed point and interpolates a 4 by 4
        // block around each pixel centre, whatever the scale.
        BufferedImage rgb = pattern(8, 6, BufferedImage.TYPE_INT_RGB);
        both("imageCopy", 20, 20, true, g ->
            g.drawImage(rgb, AffineTransform.getTranslateInstance(3, 4), null));
        both("imageOffset", 20, 20, true, g ->
            g.drawImage(rgb, AffineTransform.getTranslateInstance(3.4, 4.6), null));
        both("imageScaled", 24, 20, true, g ->
            g.drawImage(rgb, new AffineTransform(1.7, 0, 0, 1.7, 2.3, 3.1), null));
        both("imageShrunk", 20, 20, true, g ->
            g.drawImage(pattern(16, 12, BufferedImage.TYPE_INT_RGB),
                    new AffineTransform(0.7, 0, 0, 0.7, 1.5, 2.25), null));
        both("imageRotated", 24, 24, true, g ->
        {
            AffineTransform at = AffineTransform.getRotateInstance(Math.toRadians(30), 12, 12);
            at.translate(6.5, 7.25);
            at.scale(1.2, 1.2);
            g.drawImage(rgb, at, null);
        });
        both("imageAlpha", 20, 20, true, g ->
            g.drawImage(pattern(6, 5, BufferedImage.TYPE_INT_ARGB),
                    new AffineTransform(1.6, 0, 0, 1.6, 2.2, 2.7), null));
        both("imageGray", 20, 20, true, g ->
            g.drawImage(pattern(12, 10, BufferedImage.TYPE_BYTE_GRAY),
                    new AffineTransform(0.75, 0, 0, 0.75, 3.3, 2.1), null));
        // The way PageDrawer draws one: the page's transform flips y, and the
        // image's own transform flips it back.
        both("imageFlipped", 20, 20, true, g ->
        {
            g.translate(0, 20);
            g.scale(1, -1);
            g.drawImage(rgb, new AffineTransform(1.5, 0, 0, -1.5, 2.4, 16.3), null);
        });

        // 21. fills whose edges fall between pixel centres, with antialiasing
        // off, which is how PDFBox fills every rectangular path (PDFBOX-2302).
        // Java2D fills those in native code, and which of two native fillers
        // runs depends on the stroke in force, so each shape is filled with a
        // thin stroke set and again with a wide or a dashed one:
        //
        //   thin (strokeState STROKE_THIN)  ProcessPath.c, whose default is
        //                                   "pixels at corners", (x,y) -> (x,y),
        //                                   and whose PURE mode takes half a
        //                                   pixel off to put them at centres
        //   otherwise                       ShapeSpanIterator.c, which samples
        //                                   pixel centres and, unless PURE,
        //                                   first rounds every coordinate to
        //                                   the nearest quarter, floor(v+0.25)+0.25
        //
        // The rectangles are built as PDFBox builds them, a path of four
        // lines, because a Rectangle2D goes to a third filler again.
        both("fillFractional", 16, 16, false, g -> g.fill(rect(0, 0, 9.59f, 4.11f)));
        both("fillFractionalOffset", 16, 16, false, g -> g.fill(rect(0.3f, 0.3f, 4.41f, 4.41f)));
        both("fillFractionalWide", 16, 16, false, g ->
        {
            g.setStroke(new BasicStroke(3));
            g.fill(rect(0.3f, 0.3f, 4.41f, 4.41f));
        });
        both("fillFractionalDashed", 16, 16, false, g ->
        {
            g.setStroke(new BasicStroke(1, BasicStroke.CAP_BUTT, BasicStroke.JOIN_MITER, 10,
                    new float[] { 4, 4 }, 0));
            g.fill(rect(0.3f, 0.3f, 4.41f, 4.41f));
        });
        // The tile of patternscale.pdf: 7 by 3 pattern units at 1.37, which
        // the default stroke makes wide, and a thin stroke leaves thin.
        both("fillFractionalScaled", 16, 16, false, g ->
        {
            g.scale(1.37, 1.37);
            g.fill(rect(0, 0, 7, 3));
        });
        both("fillFractionalScaledThin", 16, 16, false, g ->
        {
            g.setStroke(new BasicStroke(0.5f));
            g.scale(1.37, 1.37);
            g.fill(rect(0, 0, 7, 3));
        });
        // Edges that are not axis-aligned, where a sample at a point and a
        // coverage of half a pixel are not the same test.
        both("fillFractionalTriangle", 16, 16, false, g ->
        {
            Path2D.Double p = new Path2D.Double();
            p.moveTo(1.3, 1.2);
            p.lineTo(13.8, 4.4);
            p.lineTo(4.6, 14.1);
            p.closePath();
            g.fill(p);
        });
        both("fillFractionalCurve", 16, 16, false, g ->
        {
            Path2D.Double p = new Path2D.Double();
            p.moveTo(2.4, 13.6);
            p.curveTo(2.4, 2.2, 13.7, 2.2, 13.7, 13.6);
            p.closePath();
            g.fill(p);
        });
    }

    /** A rectangle as PDFBox builds one: four lines and a close. */
    static Path2D.Float rect(float x0, float y0, float x1, float y1)
    {
        Path2D.Float p = new Path2D.Float();
        p.moveTo(x0, y0);
        p.lineTo(x1, y0);
        p.lineTo(x1, y1);
        p.lineTo(x0, y1);
        p.closePath();
        return p;
    }

    /** An image whose every pixel differs from its neighbours, of the given type. */
    static BufferedImage pattern(int w, int h, int type)
    {
        BufferedImage image = new BufferedImage(w, h, type);
        for (int y = 0; y < h; y++)
        {
            for (int x = 0; x < w; x++)
            {
                int r = (x * 37 + y * 11) & 0xff;
                int g = (x * x + y * 51) & 0xff;
                int b = ((x ^ y) * 29) & 0xff;
                int a = type == BufferedImage.TYPE_INT_ARGB ? 64 + ((x * 40 + y * 13) & 0xbf) : 0xff;
                if (type == BufferedImage.TYPE_BYTE_GRAY)
                {
                    image.getRaster().setSample(x, y, 0, (x * 31 + y * 47) & 0xff);
                }
                else
                {
                    image.setRGB(x, y, (a << 24) | (r << 16) | (g << 8) | b);
                }
            }
        }
        return image;
    }
}
