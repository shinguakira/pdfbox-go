import java.awt.image.BufferedImage;
import java.io.File;
import javax.imageio.ImageIO;
import org.apache.pdfbox.Loader;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.rendering.ImageType;
import org.apache.pdfbox.rendering.PDFRenderer;

/**
 * Renders graphics.pdf with PDFBox and writes graphics-java.png.
 *
 * This is the whole of PDFBox's renderer -- the parser, PageDrawer, and
 * Graphics2D underneath it -- against the whole of the port, and it is the
 * comparison this branch exists to make possible. Everything else in this
 * package's tests looks at one call; this looks at a page.
 *
 * The page it reads is written by the Go side, testdata/genpdf.go, so both
 * renderers are given the same bytes. Regenerate the image whenever that
 * changes:
 *
 *   javac -cp <pdfbox classes>;<commons-logging>;<log4j-api> -d . RenderDrv.java
 *   java -cp .;<same> RenderDrv
 */
public class RenderDrv
{
    public static void main(String[] args) throws Exception
    {
        File pdf = new File(args.length > 0 ? args[0] : "graphics.pdf");
        File png = new File(args.length > 1 ? args[1] : "graphics-java.png");
        try (PDDocument document = Loader.loadPDF(pdf))
        {
            PDFRenderer renderer = new PDFRenderer(document);
            ImageType type = ImageType.RGB;
            if (args.length > 2)
            {
                type = ImageType.valueOf(args[2]);
            }
            BufferedImage image = renderer.renderImage(0, 1, type);
            ImageIO.write(image, "png", png);
            System.out.printf("%s %dx%d%n", png, image.getWidth(), image.getHeight());
        }
    }
}
