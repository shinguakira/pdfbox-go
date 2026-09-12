import java.io.File;
import org.apache.pdfbox.Loader;
import org.apache.pdfbox.cos.COSArray;
import org.apache.pdfbox.cos.COSBase;
import org.apache.pdfbox.cos.COSDictionary;
import org.apache.pdfbox.cos.COSName;
import org.apache.pdfbox.multipdf.PDFMergerUtility;
import org.apache.pdfbox.pdfwriter.compress.CompressParameters;
import org.apache.pdfbox.pdmodel.PDDocument;

/**
 * Reads javabug82-dest.pdf and javabug82-src.pdf, merges the second into the
 * first with PDFMergerUtility.appendDocument, and prints the /Title of every
 * article thread the result carries, and writes the merged document out as
 * javabug82-merged-java.pdf so that the result can be opened and read.
 *
 * Correct is the destination's one thread followed by the source's two. What
 * it prints instead is in JAVA-BUGS.md 82.
 *
 * This is the same device the raster branch used: the Java is the reference, so
 * a claim about it is worth more measured than argued. Build and run it from
 * the repository root, against the Java sources in the tree:
 *
 *   javac -encoding UTF-8 -proc:none -cp <log4j-api.jar> \
 *       -sourcepath "<stub>;pdfbox/src/main/java;fontbox/src/main/java;xmpbox/src/main/java;io/src/main/java" \
 *       -d <classes> Merge82Drv.java
 *   java -cp <classes>;<log4j-api.jar> Merge82Drv go/pdfbox/multipdf/testdata
 *
 * <stub> is a directory outside the repository holding a stand-in for
 * PublicKeySecurityHandler, which needs Bouncy Castle and has nothing to do
 * with merging; SecurityHandlerFactory registers it in a static initializer,
 * which is the only reason it is reached. Put it first on the sourcepath. The
 * repository's Java is never edited.
 */
public class Merge82Drv
{
    public static void main(String[] args) throws Exception
    {
        File dir = new File(args[0]);
        try (PDDocument dest = Loader.loadPDF(new File(dir, "javabug82-dest.pdf"));
             PDDocument src = Loader.loadPDF(new File(dir, "javabug82-src.pdf")))
        {
            System.out.println("before, destination: " + titles(dest));
            System.out.println("before, source:      " + titles(src));
            new PDFMergerUtility().appendDocument(dest, src);
            System.out.println("after,  destination: " + titles(dest));
            File merged = new File(dir, "javabug82-merged-java.pdf");
            dest.save(merged, CompressParameters.NO_COMPRESSION);
            System.out.println("wrote " + merged);
        }

        // The doubling compounds: each merge appends the destination to itself,
        // so N merges leave 2^N copies of it and none of any source.
        try (PDDocument dest = Loader.loadPDF(new File(dir, "javabug82-dest.pdf")))
        {
            for (int i = 1; i <= 3; i++)
            {
                try (PDDocument src = Loader.loadPDF(new File(dir, "javabug82-src.pdf")))
                {
                    new PDFMergerUtility().appendDocument(dest, src);
                }
                File out = new File(dir, "javabug82-merged" + i + "-java.pdf");
                dest.save(out, CompressParameters.NO_COMPRESSION);
                System.out.println("after " + i + " merge(s): " + out.length()
                        + " bytes, " + titles(dest));
            }
        }
    }

    private static String titles(PDDocument document)
    {
        COSArray threads = document.getDocumentCatalog().getCOSObject()
                .getCOSArray(COSName.THREADS);
        if (threads == null)
        {
            return "(no /Threads)";
        }
        StringBuilder out = new StringBuilder();
        for (int i = 0; i < threads.size(); i++)
        {
            COSBase base = threads.getObject(i);
            String title = "(not a dictionary)";
            if (base instanceof COSDictionary)
            {
                COSDictionary info = ((COSDictionary) base).getCOSDictionary(COSName.I);
                title = info == null ? "(no /I)" : info.getString(COSName.TITLE);
            }
            out.append(out.length() == 0 ? "" : " | ").append(title);
        }
        return threads.size() + ": " + out;
    }
}
