//go:build ignore

// Writes patternscale.pdf, the page that shows what JAVA-BUGS.md 85 costs.
//
// Regenerate it and its reference together:
//
//	go run testdata/genpatternscale.go
//	javac ... RenderDrv.java && java ... RenderDrv patternscale.pdf patternscale-java.png
//
// `patterns.pdf` cannot show it. Its tiles measure a whole number of device
// pixels across, and `TilingPaint.ceiling` answers the same thing as a real
// ceiling for a whole number. What the entry is about is a tile that does not:
// the pattern here carries a `/Matrix` that scales by 1.37, so its 10-unit
// step is 13.7 device pixels, and Java rasterizes 13 of them and stretches
// them over 13.7 while the corrected port rasterizes 14.
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

	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(120, 60))
	document.AddPage(page)

	resources := pdmodel.NewPDResources()
	page.SetResources(resources)

	patterns := cos.NewDictionary()
	patterns.SetItem(cos.GetPDFName("P0"), scaledPattern().ContentStream().COSObject())
	resources.Dictionary().SetItem(cos.Pattern, patterns)

	spaces := cos.NewDictionary()
	spaces.SetItem(cos.GetPDFName("CsP"), cos.Pattern)
	resources.Dictionary().SetItem(cos.ColorSpace, spaces)

	content := "q /CsP cs /P0 scn 10 10 100 40 re f Q\n"
	stream := cos.NewStream(nil)
	output, err := stream.CreateWriter()
	check(err)
	_, err = output.Write([]byte(content))
	check(err)
	check(output.Close())
	page.Dictionary().SetItem(cos.Contents, stream)

	out, err := os.Create("testdata/patternscale.pdf")
	check(err)
	check(document.Save(out))
	check(out.Close())
}

// scaledPattern is a 10 by 10 tile under a /Matrix that scales by 1.37, so the
// tile is 13.7 device pixels across and the rounding of its raster shows.
//
// What is in the tile is a quarter circle of steps rather than a rectangle,
// because a rectangle that happens to land on the raster's own boundary would
// look the same at either size.
func scaledPattern() *pattern.PDTilingPattern {
	p := pattern.NewPDTilingPattern(nil)
	p.SetPaintType(1)
	p.SetTilingType(1)
	p.SetBBox(common.NewPDRectangleOfSize(10, 10))
	p.SetXStep(10)
	p.SetYStep(10)
	p.Dictionary().SetItem(cos.Matrix, floats(1.37, 0, 0, 1.37, 0, 0))

	output, err := p.ContentStream().CreateOutputStream()
	check(err)
	_, err = output.Write([]byte(
		"0.1 0.2 0.8 rg 0 0 7 3 re f 0 3 5 3 re f 0 6 3 4 re f\n"))
	check(err)
	check(output.Close())
	return p
}

func floats(values ...float32) *cos.Array {
	array := cos.NewArray()
	for _, v := range values {
		array.Add(cos.NewFloat(v))
	}
	return array
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
