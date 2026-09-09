import java.awt.*;
import java.awt.image.BufferedImage;
import java.lang.reflect.Constructor;
import org.apache.pdfbox.pdmodel.graphics.blend.BlendComposite;
import org.apache.pdfbox.pdmodel.graphics.blend.BlendMode;

/**
 * What PDFBox's own BlendComposite produces, for the arithmetic the Go
 * backend's blendInto is a port of.
 *
 * Unlike the shapes in Java2DDrv, this is real PDFBox: BlendComposite is
 * org.apache.pdfbox.pdmodel.graphics.blend.BlendComposite, and what it writes
 * is what a PDF with that `BM` entry renders to. It is driven the way
 * PageDrawer drives it -- setComposite, setColor, fill -- so the whole path
 * through BlendCompositeContext.compose is the one under test.
 *
 * Run it to regenerate blend.txt, with the PDFBox classes on the class path:
 *
 *   javac -cp <pdfbox classes> -d . BlendDrv.java
 *   java -cp .;<pdfbox classes> BlendDrv > blend.txt
 *
 * One line per case:
 *
 *   mode srcARGB dstARGB constantAlpha -> resultARGB
 *
 * all four as eight hex digits, alpha first.
 */
public class BlendDrv
{
    static final BlendMode[] MODES = {
        BlendMode.NORMAL, BlendMode.MULTIPLY, BlendMode.SCREEN, BlendMode.OVERLAY,
        BlendMode.DARKEN, BlendMode.LIGHTEN, BlendMode.COLOR_DODGE, BlendMode.COLOR_BURN,
        BlendMode.HARD_LIGHT, BlendMode.SOFT_LIGHT, BlendMode.DIFFERENCE,
        BlendMode.EXCLUSION, BlendMode.HUE, BlendMode.SATURATION, BlendMode.COLOR,
        BlendMode.LUMINOSITY
    };

    static final String[] NAMES = {
        "Normal", "Multiply", "Screen", "Overlay", "Darken", "Lighten", "ColorDodge",
        "ColorBurn", "HardLight", "SoftLight", "Difference", "Exclusion", "Hue",
        "Saturation", "Color", "Luminosity"
    };

    /**
     * The pairs the modes are exercised on. A mode that reads both operands
     * has to be given operands it can tell apart, in every channel and in both
     * directions, or half of them come out looking like Normal.
     */
    static final int[][] PAIRS = {
        // src         dst
        { 0xFFFF8000, 0xFF40C0FF },
        { 0xFF40C0FF, 0xFFFF8000 },
        { 0xFF000000, 0xFF7F7F7F },
        { 0xFFFFFFFF, 0xFF7F7F7F },
        { 0xFF7F7F7F, 0xFF7F7F7F },
        { 0xFF203040, 0xFFE0D0C0 },
        // and the same with alpha in play, on both sides
        { 0x80FF8000, 0xFF40C0FF },
        { 0xFFFF8000, 0x8040C0FF },
        { 0x80FF8000, 0x8040C0FF },
        { 0x40203040, 0xC0E0D0C0 },
    };

    static final float[] CONSTANTS = { 1f, 0.6f };

    /** Composes one source over one destination and answers the result. */
    static int compose(Composite composite, int src, int dst)
    {
        // The destination, as PageDrawer's surface would hold it.
        BufferedImage image = new BufferedImage(1, 1, BufferedImage.TYPE_INT_ARGB);
        image.setRGB(0, 0, dst);
        Graphics2D g = image.createGraphics();
        g.setComposite(composite);
        g.setColor(new Color(src, true));
        g.fillRect(0, 0, 1, 1);
        g.dispose();
        return image.getRGB(0, 0);
    }

    /**
     * A BlendComposite over NORMAL, which getInstance will not hand out.
     *
     * getInstance answers an AlphaComposite for NORMAL and COMPATIBLE, so a
     * document that asks for `/BM /Normal` never reaches compose at all. The
     * Go backend has only the one path -- with no blend function the first of
     * compose's three lines is the identity and what is left is source-over --
     * so it is held to both: to AlphaComposite under "Normal", and to what
     * compose itself would have made of the same pixels under "NormalBlend".
     */
    static Composite normalBlendComposite(float constantAlpha) throws Exception
    {
        Constructor<BlendComposite> c =
                BlendComposite.class.getDeclaredConstructor(BlendMode.class, float.class);
        c.setAccessible(true);
        return c.newInstance(BlendMode.NORMAL, constantAlpha);
    }

    public static void main(String[] args) throws Exception
    {
        for (int m = 0; m < MODES.length; m++)
        {
            for (int[] pair : PAIRS)
            {
                for (float constant : CONSTANTS)
                {
                    System.out.printf("%s %08x %08x %.2f -> %08x%n", NAMES[m], pair[0], pair[1],
                            constant,
                            compose(BlendComposite.getInstance(MODES[m], constant),
                                    pair[0], pair[1]));
                }
            }
        }
        for (int[] pair : PAIRS)
        {
            for (float constant : CONSTANTS)
            {
                System.out.printf("NormalBlend %08x %08x %.2f -> %08x%n", pair[0], pair[1],
                        constant,
                        compose(normalBlendComposite(constant), pair[0], pair[1]));
            }
        }
    }
}
