package rendering_test

// A soft mask whose paint is used after the graphics state that named it was
// restored.
//
// Java builds the mask when the paint is built: `applySoftMaskToPaint` renders
// the mask's transparency group there and then, while the graphics state still
// holds the mask, and `processSoftMask` reads it off that state with no null
// check. The port builds a paint that renders the mask when it is first used,
// which for a paint inside a non-isolated group is when the group is composited
// -- after the state was restored. So the state's soft mask was nil by then and
// `PDSoftMask.InitialTransformationMatrix` dereferenced it.
//
// Found by the corpus comparison of rendered pages:
// pdfjs/nonisolated_blend_smask.pdf is a page PDFBox draws and the port could
// not. The document below is written here rather than copied from it, and it is
// the smallest shape that reaches the same path: a page group, a non-isolated
// form group inside it, and inside that `q /Masked gs /Paint Do Q` where
// /Masked carries an /SMask over another group.
//
// The expected values are PDFBox's, printed by PDFRenderer.renderImageWithDPI at
// 72 dpi over the same bytes: 200 by 100, the masked rectangle ff3366e6 at
// (100,50) and at (30,30), and white outside it.

import (
	"bytes"
	"fmt"
	"image"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
)

// softMaskInNonIsolatedGroupPDF writes the document described above.
func softMaskInNonIsolatedGroupPDF() []byte {
	stream := func(number int, dictionary, content string) string {
		return fmt.Sprintf("%d 0 obj\n<< %s /Length %d >>\nstream\n%s\nendstream\nendobj\n",
			number, dictionary, len(content), content)
	}
	objects := []string{
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n",
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 100]" +
			" /Group << /Type /Group /S /Transparency /CS /DeviceRGB >>" +
			" /Resources << /XObject << /Outer 5 0 R >> >> /Contents 4 0 R >>\nendobj\n",
		stream(4, "", "/Outer Do"),
		stream(5, "/Type /XObject /Subtype /Form /FormType 1 /BBox [0 0 200 100]"+
			" /Group << /Type /Group /S /Transparency /CS /DeviceRGB /I false >>"+
			" /Resources << /ExtGState << /Masked 6 0 R >> /XObject << /Paint 7 0 R >> >>",
			"q /Masked gs /Paint Do Q"),
		"6 0 obj\n<< /Type /ExtGState /ca 1 /CA 1" +
			" /SMask << /Type /Mask /S /Alpha /G 8 0 R >> >>\nendobj\n",
		stream(7, "/Type /XObject /Subtype /Form /FormType 1 /BBox [0 0 200 100]"+
			" /Group << /Type /Group /S /Transparency /CS /DeviceRGB /I true >>"+
			" /Resources << >>", "0.2 0.4 0.9 rg\n20 20 160 60 re f"),
		stream(8, "/Type /XObject /Subtype /Form /FormType 1 /BBox [0 0 200 100]"+
			" /Group << /Type /Group /S /Transparency /CS /DeviceGray /I true >>"+
			" /Resources << >>", "1 g\n20 20 160 60 re f"),
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = out.Len()
		out.WriteString(object)
	}
	startxref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\n", len(objects)+1)
	fmt.Fprintf(&out, "startxref\n%d\n%%%%EOF\n", startxref)
	return out.Bytes()
}

func TestSoftMaskUsedAfterItsStateWasRestored(t *testing.T) {
	document, err := pdfbox.LoadPDFBytes(softMaskInNonIsolatedGroupPDF())
	if err != nil {
		t.Fatalf("loading the document: %v", err)
	}
	defer document.Close()

	rendered, err := raster.RenderPageWithDPI(document, 0, 72, rendering.RGB, false)
	if err != nil {
		t.Fatalf("rendering the page: %v", err)
	}
	bounds := rendered.Bounds()
	if bounds.Dx() != 200 || bounds.Dy() != 100 {
		t.Fatalf("the page is %dx%d, want PDFBox's 200x100", bounds.Dx(), bounds.Dy())
	}
	for _, c := range []struct {
		x, y int
		want string
	}{
		{100, 50, "ff3366e6"},
		{30, 30, "ff3366e6"},
		{5, 5, "ffffffff"},
		{190, 95, "ffffffff"},
	} {
		r, g, b, a := rendered.At(bounds.Min.X+c.x, bounds.Min.Y+c.y).RGBA()
		got := fmt.Sprintf("%02x%02x%02x%02x", a>>8, r>>8, g>>8, b>>8)
		if got != c.want {
			t.Errorf("the pixel at (%d,%d) is %s, want PDFBox's %s", c.x, c.y, got, c.want)
		}
	}
	var _ image.Image = rendered
}
