//go:build ignore

// Writes patterns.pdf, the page the tiling-pattern comparison renders.
//
// Kept apart from graphics.pdf so that adding a case here does not move the
// pixel counts that page is pinned to. Regenerate the pair together:
//
//	go run testdata/genpatterns.go
//	javac ... RenderDrv.java && java ... RenderDrv patterns.pdf patterns-java.png
//
// A tiling pattern is the one paint a Backend cannot answer on its own: the
// tile is a content stream, so drawing it means going back through the
// PageDrawer. Both kinds are here, because they take different paths --
// PaintType 1 carries its own colours, PaintType 2 is painted in the colour in
// force -- and a pattern is filled, stroked and clipped, because each reaches
// the paint differently.
package main

import (
	"log"
	"os"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
)

func main() {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(160, 120))
	document.AddPage(page)

	resources := page.Resources()
	if resources == nil {
		resources = pdmodel.NewPDResources()
		page.SetResources(resources)
	}
	patterns := cos.NewDictionary()
	// The stream, not COSObject(): a Go *cos.Stream carries its dictionary
	// rather than being one, so the pattern has to go into the resources as
	// the stream or its content is left behind.
	patterns.SetItem(cos.GetPDFName("P0"), coloredPattern().ContentStream().COSObject())
	patterns.SetItem(cos.GetPDFName("P1"), uncoloredPattern().ContentStream().COSObject())
	resources.Dictionary().SetItem(cos.Pattern, patterns)

	// The two pattern colour spaces: /Pattern on its own for a coloured
	// pattern, and /Pattern /DeviceRGB for an uncoloured one, which takes the
	// colour from the operands of `scn`.
	spaces := cos.NewDictionary()
	spaces.SetItem(cos.GetPDFName("CsP"), cos.Pattern)
	underlying := cos.NewArray()
	underlying.Add(cos.Pattern)
	underlying.Add(cos.DeviceRGB)
	spaces.SetItem(cos.GetPDFName("CsPU"), underlying)
	resources.Dictionary().SetItem(cos.ColorSpace, spaces)

	// The content stream is written by hand rather than through
	// PDPageContentStream, which has no operator for `scn` with a pattern name.
	content := `
q /CsP cs /P0 scn 10 70 60 40 re f Q
q /CsPU cs 0.9 0.1 0.1 /P1 scn 80 70 70 40 re f Q
q /CsP CS /P0 SCN 6 w 15 20 50 30 re S Q
q 90 15 60 40 re W n /CsPU cs 0.1 0.3 0.9 /P1 scn 0 0 160 120 re f Q
`
	stream := cos.NewStream(nil)
	output, err := stream.CreateWriter()
	check(err)
	_, err = output.Write([]byte(content))
	check(err)
	check(output.Close())
	page.Dictionary().SetItem(cos.Contents, stream)

	out, err := os.Create("testdata/patterns.pdf")
	check(err)
	check(document.Save(out))
	check(out.Close())
}

// coloredPattern is PaintType 1: a red bar and a blue bar, its own colours.
func coloredPattern() *pattern.PDTilingPattern {
	return tiling(1, 20, 20, "1 0 0 rg 0 0 20 10 re f 0 0 1 rg 0 10 20 10 re f\n")
}

// uncoloredPattern is PaintType 2: a diagonal, drawn in whatever colour the
// `scn` that names it carries, which is why it sets none.
func uncoloredPattern() *pattern.PDTilingPattern {
	return tiling(2, 16, 16, "2 w 0 0 m 16 16 l S 0 16 m 16 0 l S\n")
}

func tiling(paintType int, xStep, yStep float32, content string) *pattern.PDTilingPattern {
	p := pattern.NewPDTilingPattern(nil)
	p.SetPaintType(paintType)
	p.SetTilingType(1)
	p.SetBBox(common.NewPDRectangleOfSize(xStep, yStep))
	p.SetXStep(xStep)
	p.SetYStep(yStep)

	output, err := p.ContentStream().CreateOutputStream()
	check(err)
	_, err = output.Write([]byte(content))
	check(err)
	check(output.Close())
	return p
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
