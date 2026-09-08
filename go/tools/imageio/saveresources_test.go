package imageio_test

// Port of TestImageIOUtils.checkSaveResources, which is the half of that test
// class that needs no renderer.
//
// It walks a page's XObjects, and for every image asks `ImageIOUtil` to write
// it in the format its suffix names, asserting that the writer answered true.
// Java recurses into form XObjects for the same. The corpus is the Java's --
// `tools/src/test/resources/input/ImageIOUtil` -- and it is read, never
// written to.
//
// Two of the Java's substitutions come across as they are: a `jpx` suffix
// becomes "JPEG2000" and a `jb2` becomes "PNG", "jbig2 usually not available".
// A third does not: Java gets both of those readers from
// `com.github.jai-imageio`, which `tools/pom.xml` puts on the class path in
// **test scope**, and this port has neither a JPEG 2000 nor a JBIG2 decoder --
// see migration/STATUS.md. So a document this port cannot decode is recorded
// here by name rather than skipped quietly, and the assertion is made for
// every document it can.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	pdimage "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/tools/imageio"
)

// corpus is the Java's `String inDir = "src/test/resources/input/ImageIOUtil"`.
const corpus = "../../../tools/src/test/resources/input/ImageIOUtil"

// undecodable names the documents whose images this port cannot decode, and
// which decoder each waits for.
//
// A document is here because a decoder is absent, not because writing it is
// hard: both are recorded in migration/STATUS.md as deliberate non-ports.
var undecodable = map[string]string{
	"JBIG2Image.pdf":  "JBIG2",
	"JPXTestCMYK.pdf": "JPEG 2000",
	"JPXTestGrey.pdf": "JPEG 2000",
	"JPXTestRGB.pdf":  "JPEG 2000",
}

// TestSaveResources is the Java's checkSaveResources over the same corpus.
func TestSaveResources(t *testing.T) {
	entries, err := os.ReadDir(corpus)
	if err != nil {
		t.Fatalf("reading the corpus: %v", err)
	}
	found := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".pdf") {
			continue
		}
		found++
		t.Run(name, func(t *testing.T) {
			if waiting, cannot := undecodable[name]; cannot {
				t.Skipf("this port has no %s decoder; see migration/STATUS.md", waiting)
			}
			document, err := pdfbox.LoadPDF(filepath.Join(corpus, name))
			if err != nil {
				t.Fatalf("loading %s: %v", name, err)
			}
			defer document.Close()
			saveResources(t, document.Page(0).Resources())
		})
	}
	if found != 8 {
		t.Errorf("the corpus holds %d documents, want the 8 the Java's is", found)
	}
}

// saveResources is the Java method, recursion included.
func saveResources(t *testing.T, resources *pdmodel.PDResources) {
	t.Helper()
	if resources == nil {
		return
	}
	for _, name := range resources.XObjectNames() {
		xobject, err := resources.GetXObject(name)
		if err != nil {
			t.Fatalf("reading the XObject %s: %v", name.Name(), err)
		}
		switch object := xobject.(type) {
		case *pdimage.PDImageXObject:
			saveImage(t, name.Name(), object)
		case *form.PDFormXObject:
			nested, isResources := object.Resources().(*pdmodel.PDResources)
			if isResources {
				saveResources(t, nested)
			}
		}
	}
}

// saveImage is the body of the Java's `if (xobject instanceof PDImageXObject)`.
func saveImage(t *testing.T, name string, object *pdimage.PDImageXObject) {
	t.Helper()
	suffix := object.Suffix()
	if suffix == "" {
		return
	}
	switch suffix {
	case "jpx":
		suffix = "JPEG2000"
	case "jb2":
		// jbig2 usually not available
		suffix = "PNG"
	}
	decoded, err := object.Image()
	if err != nil {
		t.Fatalf("decoding %s: %v", name, err)
	}
	if decoded == nil {
		t.Fatalf("%s decoded to nothing", name)
	}
	var out bytes.Buffer
	written, err := imageio.WriteImage(decoded, suffix, &out)
	if err != nil {
		t.Fatalf("writing %s as %s: %v", name, suffix, err)
	}
	if !written {
		t.Errorf("%s was not written as %s", name, suffix)
	}
	if out.Len() == 0 {
		t.Errorf("%s was written as %s and the file is empty", name, suffix)
	}
}
