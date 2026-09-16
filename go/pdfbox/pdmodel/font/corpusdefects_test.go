package font

// Defects the corpus found. See migration/TESTDATA.md for what the corpus is
// and how it is scored; each test here names the files that reached the fault,
// and each one fails without its fix.

import (
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestType0DisplacementIsTheDescendantWidth pins the advance of a Type 0 font:
// the width its descendant's /W and /DW give, not the one in the font program.
//
// Java's PDType0Font.getDisplacement hands a horizontal font to
// PDFont.getDisplacement, whose getWidth(code) is a virtual call and so reaches
// PDType0Font.getWidth -- the descendant's /W, then /DW, then 1000. The port's
// pdFont.Displacement called its own Width instead, the simple-font lookup,
// which finds no /Widths in a Type 0 dictionary and falls through to the font
// program's advance.
//
// Every TextPosition's width, and the pen's advance between glyphs, comes from
// this displacement, and the text stripper decides where words break and which
// glyphs are duplicates from both. Twelve files of pdf.js's test corpus
// disagreed with PDFBox on the length of their text through it, among them
// issue1687.pdf: its two "l"s of "Ellis" are 4.264 apart, the stripper drops a
// repeat within a third of the glyph's width, and that third is 4.267 at
// PDFBox's 1000 and was 3.319 at the font program's 777.8.
//
// The wanted values were printed by the running PDFBox for the same four
// dictionaries over the same font file.
func TestType0DisplacementIsTheDescendantWidth(t *testing.T) {
	program, err := os.ReadFile("../../../../pdfbox/src/main/resources/org/apache/pdfbox/resources/ttf/" +
		"LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("reading LiberationSans-Regular.ttf: %v", err)
	}

	for _, c := range []struct {
		name       string
		dw         int // 0 for none
		w          bool
		want36     float32
		want37     float32
		wantWidth  [2]float32
		fromFont36 float32
	}{
		{"no /W, no /DW", 0, false, 1.0, 1.0, [2]float32{1000, 1000}, 666.9922},
		{"/DW 500", 500, false, 0.5, 0.5, [2]float32{500, 500}, 666.9922},
		{"/W [36 [600]]", 0, true, 0.6, 1.0, [2]float32{600, 1000}, 666.9922},
		{"/W [36 [600]], /DW 500", 500, true, 0.6, 0.5, [2]float32{600, 500}, 666.9922},
	} {
		t.Run(c.name, func(t *testing.T) {
			font, err := CreateFont(type0OverLiberationSans(t, program, c.dw, c.w), nil)
			if err != nil {
				t.Fatalf("CreateFont: %v", err)
			}
			for i, code := range []int{36, 37} {
				want := []float32{c.want36, c.want37}[i]
				displacement, err := font.Displacement(code)
				if err != nil {
					t.Fatalf("Displacement(%d): %v", code, err)
				}
				if displacement.X() != want || displacement.Y() != 0 {
					t.Errorf("Displacement(%d) = (%v, %v), want (%v, 0)", code, displacement.X(), displacement.Y(), want)
				}
				width, err := font.Width(code)
				if err != nil {
					t.Fatalf("Width(%d): %v", code, err)
				}
				if width != c.wantWidth[i] {
					t.Errorf("Width(%d) = %v, want %v", code, width, c.wantWidth[i])
				}
			}
			// The width the port used to answer with, so a pass is not the
			// font program happening to agree with the dictionary.
			fromFont, err := font.WidthFromFont(36)
			if err != nil {
				t.Fatalf("WidthFromFont(36): %v", err)
			}
			if fromFont != c.fromFont36 {
				t.Errorf("WidthFromFont(36) = %v, want %v", fromFont, c.fromFont36)
			}
		})
	}
}

// type0OverLiberationSans is a Type 0 font, Identity-H, over an embedded
// CIDFontType2 with an identity CIDToGIDMap, with /DW and a one-entry /W when
// asked for.
func type0OverLiberationSans(t *testing.T, program []byte, dw int, w bool) *cos.Dictionary {
	t.Helper()
	fontFile := cos.NewStream(nil)
	out, err := fontFile.CreateRawWriter()
	if err != nil {
		t.Fatalf("CreateRawWriter: %v", err)
	}
	if _, err := out.Write(program); err != nil {
		t.Fatalf("writing the font program: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("closing the font program: %v", err)
	}
	fontFile.SetInt(cos.Length1, len(program))

	descriptor := cos.NewDictionary()
	descriptor.SetItem(cos.Type, cos.FontDescriptor)
	descriptor.SetName(cos.FontName, "LiberationSans")
	descriptor.SetInt(cos.Flags, 32)
	bbox := cos.NewArray()
	for _, v := range []int64{-203, -303, 1050, 910} {
		bbox.Add(cos.GetInteger(v))
	}
	descriptor.SetItem(cos.FontBBox, bbox)
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
	if dw != 0 {
		cidFont.SetInt(cos.DW, dw)
	}
	if w {
		widths := cos.NewArray()
		widths.Add(cos.GetInteger(600))
		array := cos.NewArray()
		array.Add(cos.GetInteger(36))
		array.Add(widths)
		cidFont.SetItem(cos.W, array)
	}

	descendants := cos.NewArray()
	descendants.Add(cidFont)
	type0 := cos.NewDictionary()
	type0.SetItem(cos.Type, cos.Font)
	type0.SetItem(cos.Subtype, cos.Type0)
	type0.SetName(cos.BaseFont, "LiberationSans")
	type0.SetItem(cos.Encoding, cos.IdentityH)
	type0.SetItem(cos.DescendantFonts, descendants)
	return type0
}
