import java.awt.*;
import java.awt.geom.*;
import java.awt.image.*;
import java.util.ArrayList;
import java.util.List;

/**
 * What java.awt.TexturePaintContext answers, and where Java2D asks it.
 *
 * PDFBox paints a tiling pattern, and an image drawn under a soft mask, with a
 * java.awt.TexturePaint. Its PaintContext does not map each device pixel back
 * into the texture on its own: it maps the corner of each rectangle Java2D asks
 * for, getRaster(x, y, w, h), and walks from there in 31-bit fixed point. So
 * what a pixel shows depends on the walk and on where the rectangle began, and
 * this prints both.
 *
 * Run it to regenerate texturepaint.txt:
 *
 *   javac -d . TexturePaintDrv.java
 *   java -Djava.awt.headless=true -cp . TexturePaintDrv > texturepaint.txt
 *
 * A "texture" case makes a TexturePaint over a small image, asks its context
 * for each rectangle and prints every pixel as ARGB, eight hex digits, one row
 * of the rectangle per line. A grey texture's pixels are printed with the grey
 * in all three channels, which is what Java2D's blit from ByteGray onto an RGB
 * page makes of them, and not through the grey colour model's getRGB, which
 * would convert them.
 *
 * A "requests" case fills or strokes one shape on a page with PDFRenderer's
 * hints, through a Paint that hands out a TexturePaint's context and records
 * every rectangle asked of it. The Paint is not a TexturePaint, so Java2D
 * treats it as a custom paint, which is what PDFBox's TilingPaint and SoftMask
 * are.
 *
 * The textures are made from formulas the Go test repeats, rgbAt, alphaAt and
 * greyAt.
 *
 * Last come two tables of TYPE_BYTE_GRAY's colour model, which a filtered grey
 * texture goes through both ways: getRGB of each grey, and the grey
 * getDataElements makes of each sRGB grey (s, s, s).
 */
public class TexturePaintDrv
{
    static int rgbAt(int x, int y)
    {
        int r = (x * 37 + y * 11) & 0xff;
        int g = (x * x + y * 3) & 0xff;
        int b = ((x ^ y) * 9) & 0xff;
        return (r << 16) | (g << 8) | b;
    }

    static int alphaAt(int x, int y)
    {
        if ((x + 2 * y) % 7 == 0)
        {
            return 0;
        }
        return (x * 29 + y * 53) & 0xff;
    }

    static int greyAt(int x, int y)
    {
        return ((x / 3 + y / 2) % 2 == 0) ? 0 : 255;
    }

    static BufferedImage texture(int type, int w, int h)
    {
        BufferedImage img = new BufferedImage(w, h, type);
        for (int y = 0; y < h; y++)
        {
            for (int x = 0; x < w; x++)
            {
                switch (type)
                {
                    case BufferedImage.TYPE_INT_RGB:
                        img.setRGB(x, y, rgbAt(x, y));
                        break;
                    case BufferedImage.TYPE_INT_ARGB:
                        img.setRGB(x, y, (alphaAt(x, y) << 24) | rgbAt(x, y));
                        break;
                    case BufferedImage.TYPE_BYTE_GRAY:
                        img.getRaster().setSample(x, y, 0, greyAt(x, y));
                        break;
                    default:
                        throw new IllegalArgumentException("type " + type);
                }
            }
        }
        return img;
    }

    static RenderingHints hints(boolean filter)
    {
        RenderingHints r = new RenderingHints(null);
        r.put(RenderingHints.KEY_INTERPOLATION, filter
                ? RenderingHints.VALUE_INTERPOLATION_BICUBIC
                : RenderingHints.VALUE_INTERPOLATION_NEAREST_NEIGHBOR);
        r.put(RenderingHints.KEY_RENDERING, RenderingHints.VALUE_RENDER_QUALITY);
        return r;
    }

    static void texture(String name, int type, int w, int h, Rectangle2D anchor,
            AffineTransform xform, boolean filter, int[][] requests)
    {
        BufferedImage img = texture(type, w, h);
        TexturePaint paint = new TexturePaint(img, anchor);
        PaintContext context = paint.createContext(ColorModel.getRGBdefault(),
                new Rectangle(0, 0, 200, 200), anchor, new AffineTransform(xform),
                hints(filter));
        ColorModel cm = context.getColorModel();
        System.out.println("### texture " + name + " " + context.getClass().getSimpleName());
        for (int[] q : requests)
        {
            Raster raster = context.getRaster(q[0], q[1], q[2], q[3]);
            System.out.println("request " + q[0] + " " + q[1] + " " + q[2] + " " + q[3]);
            for (int j = 0; j < q[3]; j++)
            {
                StringBuilder sb = new StringBuilder();
                for (int i = 0; i < q[2]; i++)
                {
                    int argb;
                    if (type == BufferedImage.TYPE_BYTE_GRAY)
                    {
                        int g = raster.getSample(i, j, 0);
                        argb = 0xff000000 | (g << 16) | (g << 8) | g;
                    }
                    else
                    {
                        argb = cm.getRGB(raster.getDataElements(i, j, null));
                    }
                    sb.append(String.format("%08x", argb));
                }
                System.out.println(sb);
            }
        }
        context.dispose();
    }

    /** A Paint that records the rectangles its context is asked for. */
    static final class RecordingPaint implements Paint
    {
        final Paint inner;
        final List<int[]> requests = new ArrayList<>();

        RecordingPaint(Paint inner)
        {
            this.inner = inner;
        }

        @Override
        public PaintContext createContext(ColorModel cm, Rectangle deviceBounds,
                Rectangle2D userBounds, AffineTransform xform, RenderingHints hints)
        {
            PaintContext context = inner.createContext(cm, deviceBounds, userBounds, xform, hints);
            return new PaintContext()
            {
                @Override
                public void dispose()
                {
                    context.dispose();
                }

                @Override
                public ColorModel getColorModel()
                {
                    return context.getColorModel();
                }

                @Override
                public Raster getRaster(int x, int y, int w, int h)
                {
                    requests.add(new int[] { x, y, w, h });
                    return context.getRaster(x, y, w, h);
                }
            };
        }

        @Override
        public int getTransparency()
        {
            return Transparency.TRANSLUCENT;
        }
    }

    /** What one requests case draws. */
    interface Draw
    {
        void draw(Graphics2D g);
    }

    static void requests(String name, int w, int h, boolean antiAlias,
            AffineTransform transform, Shape clip, Draw draw)
    {
        BufferedImage page = new BufferedImage(w, h, BufferedImage.TYPE_INT_RGB);
        Graphics2D g = page.createGraphics();
        g.setColor(Color.WHITE);
        g.fillRect(0, 0, w, h);
        RenderingHints r = new RenderingHints(null);
        r.put(RenderingHints.KEY_INTERPOLATION, RenderingHints.VALUE_INTERPOLATION_BICUBIC);
        r.put(RenderingHints.KEY_RENDERING, RenderingHints.VALUE_RENDER_QUALITY);
        r.put(RenderingHints.KEY_ANTIALIASING, antiAlias
                ? RenderingHints.VALUE_ANTIALIAS_ON : RenderingHints.VALUE_ANTIALIAS_OFF);
        g.addRenderingHints(r);
        if (clip != null)
        {
            g.setClip(clip);
        }
        g.setTransform(transform);
        RecordingPaint paint = new RecordingPaint(new TexturePaint(
                texture(BufferedImage.TYPE_INT_ARGB, 13, 13),
                new Rectangle2D.Float(0, 0, 13.7f, 13.7f)));
        g.setPaint(paint);
        draw.draw(g);
        g.dispose();
        System.out.println("### requests " + name);
        for (int[] q : paint.requests)
        {
            System.out.println(q[0] + " " + q[1] + " " + q[2] + " " + q[3]);
        }
    }

    static void greyTables()
    {
        ColorModel cm = new BufferedImage(1, 1, BufferedImage.TYPE_BYTE_GRAY).getColorModel();
        StringBuilder toRGB = new StringBuilder();
        StringBuilder fromRGB = new StringBuilder();
        for (int v = 0; v < 256; v++)
        {
            byte[] grey = { (byte) v };
            toRGB.append(String.format("%02x", cm.getRGB(grey) & 0xff));
            byte[] back = (byte[]) cm.getDataElements(0xff000000 | (v * 0x010101), null);
            fromRGB.append(String.format("%02x", back[0] & 0xff));
        }
        System.out.println("### table greyToSRGB");
        System.out.println(toRGB);
        System.out.println("### table sRGBToGrey");
        System.out.println(fromRGB);
    }

    public static void main(String[] args)
    {
        int[][] imageTiles = { { 60, 60, 32, 32 }, { 92, 60, 28, 32 }, { 60, 92, 32, 13 },
                { 92, 92, 28, 13 } };
        AffineTransform image = new AffineTransform(1.5, 0, 0, 1.5, 60, 60);
        texture("imageNearest", BufferedImage.TYPE_INT_RGB, 40, 30,
                new Rectangle2D.Float(0, 0, 40, 30), image, false, imageTiles);
        texture("imageFiltered", BufferedImage.TYPE_INT_RGB, 40, 30,
                new Rectangle2D.Float(0, 0, 40, 30), image, true, imageTiles);

        AffineTransform flip = new AffineTransform(1, 0, 0, -1, 0, 60);
        Rectangle2D tile = new Rectangle2D.Float(0, 0, 13.7f, 13.7f);
        texture("patternRows", BufferedImage.TYPE_INT_ARGB, 13, 13, tile, flip, true,
                new int[][] { { 10, 10, 32, 1 }, { 42, 10, 32, 1 }, { 74, 10, 32, 1 },
                        { 106, 10, 4, 1 }, { 10, 11, 32, 1 }, { 10, 49, 32, 1 } });
        texture("patternTiles", BufferedImage.TYPE_INT_ARGB, 13, 13, tile, flip, true,
                new int[][] { { 10, 10, 32, 32 }, { 42, 10, 32, 32 } });

        AffineTransform skew = new AffineTransform(1.2, 0.5, -0.4, 1.3, 20.25, 7.5);
        Rectangle2D skewAnchor = new Rectangle2D.Float(2.5f, -3.25f, 11.2f, 15.6f);
        int[][] skewTiles = { { 0, 0, 32, 32 }, { 32, 0, 20, 32 }, { 0, 32, 32, 20 } };
        texture("skewFiltered", BufferedImage.TYPE_INT_ARGB, 16, 12, skewAnchor, skew, true,
                skewTiles);
        texture("skewNearest", BufferedImage.TYPE_INT_ARGB, 16, 12, skewAnchor, skew, false,
                skewTiles);

        texture("shrinkFiltered", BufferedImage.TYPE_INT_RGB, 40, 30,
                new Rectangle2D.Double(0, 0, 26.5, 19.25),
                new AffineTransform(1, 0, 0, 1, 3, 4), true, new int[][] { { 3, 4, 27, 20 } });

        Rectangle2D greyAnchor = new Rectangle2D.Float(0, 0, 30, 15);
        texture("greyNearest", BufferedImage.TYPE_BYTE_GRAY, 20, 10, greyAnchor,
                new AffineTransform(), false, new int[][] { { 0, 0, 30, 15 } });
        texture("greyFiltered", BufferedImage.TYPE_BYTE_GRAY, 20, 10, greyAnchor,
                new AffineTransform(), true, new int[][] { { 0, 0, 30, 15 } });

        Rectangle2D imageRect = new Rectangle2D.Float(0, 0, 40, 30);
        requests("imageTiles", 200, 170, true, image, null, g -> g.fill(imageRect));
        requests("imageFraction", 200, 170, true,
                new AffineTransform(1.3, 0, 0, 1.7, 10.4, 20.95), null, g -> g.fill(imageRect));
        requests("imageNoAntiAlias", 200, 170, false,
                new AffineTransform(1.5, 0, 0, 1.5, 60.7, 60.2), null, g -> g.fill(imageRect));
        requests("patternNoAntiAlias", 120, 60, false, flip, null,
                g -> g.fill(new Area(new Rectangle2D.Float(10, 10, 100, 40))));
        Path2D.Double quad = new Path2D.Double();
        quad.moveTo(10.3, 5.2);
        quad.lineTo(70.6, 12.9);
        quad.lineTo(55.5, 50.4);
        quad.lineTo(4.8, 33.3);
        quad.closePath();
        requests("patternAntiAlias", 120, 60, true, flip, null, g -> g.fill(new Area(quad)));
        requests("clipRectangle", 120, 90, true, new AffineTransform(),
                new Rectangle(20, 15, 50, 40),
                g -> g.fill(new Rectangle2D.Float(0, 0, 100, 80)));
        Shape ellipse = new Ellipse2D.Double(10, 10, 60, 40);
        requests("clipShapeNoAntiAlias", 100, 60, false, new AffineTransform(), ellipse,
                g -> g.fill(new Area(new Rectangle2D.Float(0, 0, 100, 60))));
        requests("clipShapeAntiAlias", 100, 60, true, new AffineTransform(), ellipse,
                g -> g.fill(new Rectangle2D.Float(0, 0, 100, 60)));
        Line2D line = new Line2D.Double(10.2, 20.7, 90.4, 55.1);
        requests("strokeAntiAlias", 100, 70, true, new AffineTransform(), null, g -> {
            g.setStroke(new BasicStroke(3.5f));
            g.draw(line);
        });
        requests("strokeNoAntiAlias", 100, 70, false, new AffineTransform(), null, g -> {
            g.setStroke(new BasicStroke(3.5f));
            g.draw(line);
        });
        requests("bigTile", 120, 90, true, new AffineTransform(), null,
                g -> g.fill(new Rectangle2D.Double(5.5, 3.25, 100, 70)));
        greyTables();
    }
}
