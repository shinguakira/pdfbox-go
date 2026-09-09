import java.io.File;
import java.io.PrintStream;
import org.apache.pdfbox.cos.COSName;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.pdmodel.PDPage;
import org.apache.pdfbox.pdmodel.common.PDRectangle;
import org.apache.pdfbox.pdmodel.graphics.color.PDColor;
import org.apache.pdfbox.pdmodel.graphics.color.PDDeviceRGB;
import org.apache.pdfbox.pdmodel.interactive.annotation.PDAnnotationSquiggly;

/**
 * Writes squiggly.pdf: one page with two squiggly annotations whose
 * appearances PDFBox generated.
 *
 * This is the reference for PDSquigglyAppearanceHandler, which is the one
 * appearance handler that fills with a tiling pattern -- the port could not
 * write it until `track/raster` brought PDTilingPattern,
 * PDPatternContentStream and the PDPattern colour space.
 *
 * The Go test regenerates the same two appearances with the port and compares
 * the content streams token for token, which is what
 * AppearanceGenerationTest.checkAnnotationTokens does.
 *
 * Regenerate with the PDFBox classes and log4j-api on the class path:
 *
 *   javac -cp &lt;classes&gt; -d . SquigglyDrv.java
 *   java -cp .;&lt;classes&gt;;&lt;log4j-api.jar&gt; SquigglyDrv squiggly.pdf
 */
public class SquigglyDrv
{
    public static void main(String[] args) throws Exception
    {
        File out = new File(args.length > 0 ? args[0] : "squiggly.pdf");
        try (PDDocument document = new PDDocument())
        {
            PDPage page = new PDPage(new PDRectangle(200, 120));
            document.addPage(page);

            // A wide one and a short one, so that the height-driven transform
            // and the form's horizontal size are both exercised.
            page.getAnnotations().add(squiggly(20, 80, 160, 96, 1, 0, 0));
            page.getAnnotations().add(squiggly(20, 30, 70, 54, 0, 0.4f, 0.8f));

            for (Object annotation : page.getAnnotations())
            {
                ((PDAnnotationSquiggly) annotation).constructAppearances();
            }
            document.save(out);
        }
        try (PrintStream unused = System.out)
        {
            System.out.println(out + " written");
        }
    }

    /** One squiggly over the quad (x0,y1)-(x1,y1)-(x0,y0)-(x1,y0). */
    static PDAnnotationSquiggly squiggly(float x0, float y0, float x1, float y1,
            float r, float g, float b)
    {
        PDAnnotationSquiggly annotation = new PDAnnotationSquiggly();
        annotation.setRectangle(new PDRectangle(x0, y0, x1 - x0, y1 - y0));
        // The quadpoints order is the one the spec gets wrong and every
        // producer writes: upper left, upper right, lower left, lower right.
        annotation.setQuadPoints(new float[] { x0, y1, x1, y1, x0, y0, x1, y0 });
        annotation.setColor(new PDColor(new float[] { r, g, b }, PDDeviceRGB.INSTANCE));
        annotation.setConstantOpacity(1);
        annotation.getCOSObject().removeItem(COSName.AP);
        return annotation;
    }
}
