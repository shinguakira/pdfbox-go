//go:build ignore

// Writes stencil.pdf, the page the stencil-mask comparison renders.
//
// Regenerate it and its reference together:
//
//	go run testdata/genstencil.go
//	javac ... RenderDrv.java && java ... RenderDrv stencil.pdf stencil-java.png
//
// A stencil is `/ImageMask true`: one bit per sample, no colour space, and
// where the bit says so the *current colour* shows and everywhere else nothing
// is painted at all. It is the one image the backend is handed with a paint
// beside it, and the only thing it can take from the image is which pixels are
// in -- so a page of them is what says whether it does.
package main

import (
	"log"
	"os"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

func main() {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(160, 100))
	document.AddPage(page)

	resources := pdmodel.NewPDResources()
	page.SetResources(resources)

	xobjects := cos.NewDictionary()
	// A checkerboard, 8 by 8, one bit a sample: alternating rows of 10101010
	// and 01010101, so half the samples are set and the two halves interleave.
	xobjects.SetItem(cos.GetPDFName("Im0"), stencil(8, 8, func(x, y int) bool {
		return (x+y)%2 == 0
	}))
	// And one with a shape in it, so that a wrong stencil shows as a filled
	// box rather than as a different pattern: a diagonal wedge.
	xobjects.SetItem(cos.GetPDFName("Im1"), stencil(16, 16, func(x, y int) bool {
		return x <= y
	}))
	resources.Dictionary().SetItem(cos.XObject, xobjects)

	// Each is drawn twice, in two colours, because the colour comes from the
	// graphics state rather than from the image.
	content := `
q 0.9 0.1 0.1 rg 40 0 0 35 10 55 cm /Im0 Do Q
q 0.1 0.3 0.9 rg 40 0 0 35 60 55 cm /Im0 Do Q
q 0 0.5 0.2 rg 40 0 0 35 10 10 cm /Im1 Do Q
q 0.2 0.2 0.2 rg 90 0 0 35 60 10 cm /Im1 Do Q
`
	stream := cos.NewStream(nil)
	output, err := stream.CreateWriter()
	check(err)
	_, err = output.Write([]byte(content))
	check(err)
	check(output.Close())
	page.Dictionary().SetItem(cos.Contents, stream)

	out, err := os.Create("testdata/stencil.pdf")
	check(err)
	check(document.Save(out))
	check(out.Close())
}

// stencil is an /ImageMask image XObject whose bit is set where set says so.
//
// A set bit is a 0 sample, because the default /Decode of an image mask is
// [0 1] and 0 means "paint here". The rows are padded to a byte, as 8.9.3
// requires.
func stencil(width, height int, set func(x, y int) bool) *cos.Stream {
	stream := cos.NewStream(nil)
	stream.Dictionary.SetItem(cos.Type, cos.XObject)
	stream.Dictionary.SetItem(cos.Subtype, cos.Image)
	stream.Dictionary.SetInt(cos.Width, width)
	stream.Dictionary.SetInt(cos.Height, height)
	stream.Dictionary.SetInt(cos.BitsPerComponent, 1)
	stream.Dictionary.SetItem(cos.ImageMask, cos.True)

	rowBytes := (width + 7) / 8
	samples := make([]byte, rowBytes*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if set(x, y) {
				continue
			}
			// Not set: leave the sample as 1, which paints nothing.
			samples[y*rowBytes+x/8] |= 0x80 >> (x % 8)
		}
	}

	output, err := stream.CreateWriter()
	check(err)
	_, err = output.Write(samples)
	check(err)
	check(output.Close())
	return stream
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
