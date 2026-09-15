package text_test

import (
	"os"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// TestVerticalGlyphWidthIsScaledByTheCIDFontsEm pins the Type 0 half of the
// vertical branch of LegacyPDFStreamEngine.showGlyph.
//
// A vertical font's displacement has no x to sort by, so the Java measures the
// glyph's width instead and scales it by 1000 / unitsPerEm when the font program
// is a TrueType font -- a PDTrueTypeFont's own, or the descendant
// PDCIDFontType2's of a PDType0Font. The port had the first and left the
// second for "a later slice", which brought the CID fonts and not the branch,
// so every vertical Type 0 glyph over a TrueType program with an em other than
// 1000 came out too wide. The pdf.js corpus has no such file; the gap was found
// reading the method while the Type 0 displacement defect beside it was fixed.
//
// The font is LiberationSans, whose em is 2048, with no /W, so the width is the
// default 1000. The wanted values were printed by the running PDFBox for the
// same page: horizontally the width is the displacement, 12; vertically it is
// the scaled displacement, 5.859375, measured from a glyph origin the position
// vector has moved 6 to the left.
func TestVerticalGlyphWidthIsScaledByTheCIDFontsEm(t *testing.T) {
	program, err := os.ReadFile("../../../pdfbox/src/main/resources/org/apache/pdfbox/resources/ttf/" +
		"LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("reading LiberationSans-Regular.ttf: %v", err)
	}

	for _, c := range []struct {
		encoding  *cos.Name
		wantX     float32
		wantWidth float32
	}{
		{cos.IdentityV, 94, 11.859375},
		{cos.IdentityH, 100, 12},
	} {
		t.Run(c.encoding.Name(), func(t *testing.T) {
			tree := pdmodel.NewPDPageTreeOf(onePageTree(t, program, c.encoding))
			stripper := text.NewPDFTextStripper()
			var out strings.Builder
			stripper.SetOutput(&out)
			var positions []*text.TextPosition
			stripper.SetProcessTextPosition(func(p *text.TextPosition) error {
				positions = append(positions, p)
				return stripper.ProcessTextPosition(p)
			})
			if err := stripper.ProcessPages(tree); err != nil {
				t.Fatalf("ProcessPages: %v", err)
			}
			if len(positions) != 1 {
				t.Fatalf("%d text positions, want 1", len(positions))
			}
			p := positions[0]
			if p.Unicode() != "A" {
				t.Errorf("unicode %q, want %q", p.Unicode(), "A")
			}
			if p.X() != c.wantX {
				t.Errorf("x = %v, want %v", p.X(), c.wantX)
			}
			if p.Width() != c.wantWidth {
				t.Errorf("width = %v, want %v", p.Width(), c.wantWidth)
			}
		})
	}
}

// onePageTree is a page tree of one page that shows glyph 36 of an embedded
// LiberationSans, through a Type 0 font of the given encoding over a
// CIDFontType2 with an identity CIDToGIDMap, and maps it to "A".
func onePageTree(t *testing.T, program []byte, encoding *cos.Name) *cos.Dictionary {
	t.Helper()
	stream := func(data string) *cos.Stream {
		s := cos.NewStream(nil)
		w, err := s.CreateRawWriter()
		if err != nil {
			t.Fatalf("CreateRawWriter: %v", err)
		}
		if _, err := w.Write([]byte(data)); err != nil {
			t.Fatalf("writing a stream: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("closing a stream: %v", err)
		}
		return s
	}
	integers := func(values ...int64) *cos.Array {
		array := cos.NewArray()
		for _, v := range values {
			array.Add(cos.GetInteger(v))
		}
		return array
	}

	fontFile := stream(string(program))
	fontFile.SetInt(cos.Length1, len(program))
	descriptor := cos.NewDictionary()
	descriptor.SetItem(cos.Type, cos.FontDescriptor)
	descriptor.SetName(cos.FontName, "LiberationSans")
	descriptor.SetInt(cos.Flags, 32)
	descriptor.SetItem(cos.FontBBox, integers(-203, -303, 1050, 910))
	descriptor.SetItem(cos.FontFile2, fontFile)

	info := cos.NewDictionary()
	info.SetItem(cos.Registry, cos.NewStringObj("Adobe"))
	info.SetItem(cos.Ordering, cos.NewStringObj("Identity"))
	info.SetInt(cos.Supplement, 0)

	cidFont := cos.NewDictionary()
	cidFont.SetItem(cos.Type, cos.Font)
	cidFont.SetItem(cos.Subtype, cos.CIDFontType2)
	cidFont.SetName(cos.BaseFont, "LiberationSans")
	cidFont.SetItem(cos.CIDSystemInfo, info)
	cidFont.SetItem(cos.FontDescriptor, descriptor)
	cidFont.SetItem(cos.CIDToGIDMap, cos.Identity)

	descendants := cos.NewArray()
	descendants.Add(cidFont)
	type0 := cos.NewDictionary()
	type0.SetItem(cos.Type, cos.Font)
	type0.SetItem(cos.Subtype, cos.Type0)
	type0.SetName(cos.BaseFont, "LiberationSans")
	type0.SetItem(cos.Encoding, encoding)
	type0.SetItem(cos.DescendantFonts, descendants)
	type0.SetItem(cos.ToUnicode, stream("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n"+
		"/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n1 begincodespacerange\n<0000> <FFFF>\n"+
		"endcodespacerange\n1 beginbfchar\n<0024> <0041>\nendbfchar\nendcmap\n"+
		"CMapName currentdict /CMap defineresource pop\nend\nend\n"))

	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F1"), type0)
	resources := cos.NewDictionary()
	resources.SetItem(cos.Font, fonts)

	page := cos.NewDictionary()
	page.SetItem(cos.Type, cos.Page)
	page.SetItem(cos.MediaBox, integers(0, 0, 612, 792))
	page.SetItem(cos.Resources, resources)
	page.SetItem(cos.Contents, stream("BT /F1 12 Tf 100 700 Td <0024> Tj ET"))

	kids := cos.NewArray()
	kids.Add(page)
	root := cos.NewDictionary()
	root.SetItem(cos.Type, cos.Pages)
	root.SetInt(cos.Count, 1)
	root.SetItem(cos.Kids, kids)
	page.SetItem(cos.Parent, root)
	return root
}
