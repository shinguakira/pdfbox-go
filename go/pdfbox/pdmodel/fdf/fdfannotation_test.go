package fdf_test

// Port of org.apache.pdfbox.pdmodel.fdf.FDFAnnotationTest.

import (
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fdf"
)

// fdfFixture is where the Java fdf test resources are.
const fdfFixture = "../../../../pdfbox/src/test/resources/org/apache/pdfbox/pdmodel/fdf/"

// annotationsOf is the chain every case starts with:
// fdfDoc.getCatalog().getFDF().getAnnotations().
func annotationsOf(t *testing.T, document *fdf.FDFDocument) []fdf.FDFAnnotation {
	t.Helper()
	list, err := document.Catalog().FDF().Annotations()
	if err != nil {
		t.Fatalf("Annotations: %v", err)
	}
	if list == nil {
		t.Fatal("Annotations() = nil, want the annotations of the document")
	}
	out := make([]fdf.FDFAnnotation, 0, list.Size())
	for i := 0; i < list.Size(); i++ {
		out = append(out, list.Get(i))
	}
	return out
}

// TestLoadXFDFAnnotations is loadXFDFAnnotations, which is PDFBOX-4345 and
// PDFBOX-3646.
//
// Before the fix the rich text came out as
//
//	...<p dir="ltr"><span style="...">P&2</span></p>...
//
// with the ampersand unescaped and "P&amp;1" and "P&amp;3" missing entirely.
func TestLoadXFDFAnnotations(t *testing.T) {
	document, err := pdfbox.LoadXFDF(fdfFixture + "xfdf-test-document-annotations.xml")
	if err != nil {
		t.Fatalf("LoadXFDF: %v", err)
	}
	defer document.Close()

	annotations := annotationsOf(t, document)
	if len(annotations) != 18 {
		t.Fatalf("the document holds %d annotations, want 18", len(annotations))
	}

	const wantRich = `<body style="font:12pt Helvetica; color:#D66C00;" ` +
		`xfa:APIVersion="Acrobat:7.0.8" xfa:spec="2.0.2" ` +
		`xmlns="http://www.w3.org/1999/xhtml" ` +
		`xmlns:xfa="http://www.xfa.org/schema/xfa-data/1.0/">` + "\n" +
		`          <p dir="ltr">P&amp;1 <span style="text-decoration:word;` +
		`font-family:Helvetica">P&amp;2</span> P&amp;3</p>` + "\n" +
		`        </body>`

	tested := false
	for _, annotation := range annotations {
		freeText, isFreeText := annotation.(*fdf.FDFAnnotationFreeText)
		if !isFreeText {
			continue
		}
		if freeText.Contents() != "P&1 P&2 P&3" {
			continue
		}
		tested = true
		if got := strings.TrimSpace(freeText.RichContents()); got != wantRich {
			t.Errorf("RichContents() =\n%q\nwant\n%q", got, wantRich)
		}
	}
	if !tested {
		t.Error("no free text annotation with the contents \"P&1 P&2 P&3\" was " +
			"found, so nothing was checked")
	}
}

// TestAnnotationWidth is testAnnotationWidth: a width of "0.00" gives a border
// style whose width is zero, rather than no border style at all.
func TestAnnotationWidth(t *testing.T) {
	const xfdf = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<xfdf xmlns="http://ns.adobe.com/xfdf/" xml:space="preserve">` +
		`<annots>` +
		`<freetext` +
		` width="0.00"` +
		` justification="left" page="0"` +
		` date="D:20251124141013+01'00'"` +
		` flags="print"` +
		` name="b525be7e-4735-4598-ab7f-163cd0c7e48b"` +
		` rect="372.339325,722.633545,531.075317,736.673523"` +
		` title="Username"` +
		` BBox="372.339325,722.633545,531.075317,736.673523"` +
		` Matrix="1.000000,0.000000,0.000000,1.000000,0.000000,0.000000"` +
		` creationdate="D:20251124141003+01'00'"` +
		` opacity="1"` +
		` subject="Texteingabe"` +
		` intent="FreeTextTypewriter"` +
		` IT="FreeTextTypewriter">` +
		`<defaultappearance>&#x20;/Helv 12 Tf 0.415686 0.756863 0.690196 rg</defaultappearance>` +
		`<defaultstyle>font: &apos;Helvetica&apos; ,sans-serif 12.00pt;color:#3049D1</defaultstyle>` +
		`<contents>Your text is here.</contents>` +
		`</freetext>` +
		`</annots>` +
		`<f href=".xfdf"/>` +
		`</xfdf>`

	document, err := pdfbox.LoadXFDFReader(strings.NewReader(xfdf))
	if err != nil {
		t.Fatalf("LoadXFDFReader: %v", err)
	}
	defer document.Close()

	annotations := annotationsOf(t, document)
	if len(annotations) != 1 {
		t.Fatalf("the document holds %d annotations, want 1", len(annotations))
	}

	freeText, isFreeText := annotations[0].(*fdf.FDFAnnotationFreeText)
	if !isFreeText {
		t.Fatalf("the annotation is %T, want a free text one", annotations[0])
	}
	borderStyle := freeText.BorderStyle()
	if borderStyle == nil {
		t.Fatal("BorderStyle() = nil, want a border style with a zero width")
	}
	if got := borderStyle.Width(); got < -0.01 || got > 0.01 {
		t.Errorf("Width() = %v, want 0", got)
	}
}
