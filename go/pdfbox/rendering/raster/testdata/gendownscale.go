//go:build ignore

// Writes downscale.pdf, the page the image downscaling comparison renders.
//
// Regenerate it and its reference together:
//
//	go run testdata/gendownscale.go
//	javac ... RenderDrv.java && java ... RenderDrv downscale.pdf downscale-java.png
//
// PDFBox draws an image it shrinks below half size -- imageDownscalingOptimization
// Threshold, 0.5 unless the caller says otherwise -- in two steps:
// getScaledInstance(SCALE_SMOOTH) averages it down to the size it will have on
// the page, and drawImage puts that down with bicubic interpolation. Above the
// threshold it draws the image directly. The page has one of each, and the
// images whose averaging the JDK does differently: an RGB image, one with a
// soft mask, a one-bit grey image, which PDFBox holds as TYPE_BYTE_GRAY, and
// a stencil filled with a colour.
package main

import (
	"log"
	"os"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

func main() {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(240, 160))
	document.AddPage(page)

	resources := pdmodel.NewPDResources()
	page.SetResources(resources)

	xobjects := cos.NewDictionary()
	xobjects.SetItem(cos.GetPDFName("A"), rgbImage())
	xobjects.SetItem(cos.GetPDFName("D"), maskedImage())
	xobjects.SetItem(cos.GetPDFName("E"), bilevelImage())
	xobjects.SetItem(cos.GetPDFName("F"), stencil())
	resources.Dictionary().SetItem(cos.XObject, xobjects)

	// A at a fifth of its size, at nearly a half, and at more than a half,
	// where PDFBox draws it directly; D at a quarter, off the pixel grid; E and
	// F at a quarter.
	content := `
q 16 0 0 12 10 138 cm /A Do Q
q 36 0 0 27 40 110 cm /A Do Q
q 48 0 0 36 100 100 cm /A Do Q
q 20 0 0 15 180.3 127.1 cm /D Do Q
q 40 0 0 40 10 50 cm /E Do Q
q 1 0 0 rg 40 0 0 40 60 50 cm /F Do Q
`
	stream := cos.NewStream(nil)
	output, err := stream.CreateWriter()
	check(err)
	_, err = output.Write([]byte(content))
	check(err)
	check(output.Close())
	page.Dictionary().SetItem(cos.Contents, stream)

	out, err := os.Create("testdata/downscale.pdf")
	check(err)
	check(document.Save(out))
	check(out.Close())
}

// rgbImage is 80 by 60 of colour that changes from pixel to pixel, so that
// what an average makes of it shows.
func rgbImage() *cos.Stream {
	const width, height = 80, 60
	samples := make([]byte, 0, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			samples = append(samples,
				byte(x*37+y*11), byte(x*x+y*3), byte((x^y)*9))
		}
	}
	return image(width, height, 8, cos.DeviceRGB, samples)
}

// maskedImage is the RGB image with a soft mask that fades it out diagonally.
func maskedImage() *cos.Stream {
	const width, height = 80, 60
	alpha := make([]byte, 0, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			alpha = append(alpha, byte(255-min(255, (x+y)*255/(width+height-2))))
		}
	}
	stream := rgbImage()
	stream.SetItem(cos.SMask, image(width, height, 8, cos.DeviceGray, alpha))
	return stream
}

// bilevelImage is 160 by 160 one-bit grey: stripes and a diagonal, which is
// what a scanned page is made of.
func bilevelImage() *cos.Stream {
	return image(160, 160, 1, cos.DeviceGray, bits(160, 160, func(x, y int) bool {
		return (x/3+y/5)%2 == 0 != (x == y || x == 159-y)
	}))
}

// stencil is a 160 by 160 image mask: rings, so that the edge of the mask
// runs in every direction.
func stencil() *cos.Stream {
	stream := cos.NewStream(filter.Provider{})
	stream.SetItem(cos.Type, cos.XObject)
	stream.SetItem(cos.Subtype, cos.Image)
	stream.SetInt(cos.Width, 160)
	stream.SetInt(cos.Height, 160)
	stream.SetInt(cos.BitsPerComponent, 1)
	stream.SetBoolean(cos.ImageMask, true)
	write(stream, bits(160, 160, func(x, y int) bool {
		dx, dy := x-80, y-80
		return (dx*dx+dy*dy)/300%2 == 0
	}))
	return stream
}

// bits packs a one-bit image, a row at a time, the first pixel in the high bit.
func bits(width, height int, set func(x, y int) bool) []byte {
	stride := (width + 7) / 8
	samples := make([]byte, stride*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if set(x, y) {
				samples[y*stride+x/8] |= 0x80 >> uint(x%8)
			}
		}
	}
	return samples
}

// image is an image XObject.
func image(width, height, bitsPerComponent int, space *cos.Name, samples []byte) *cos.Stream {
	stream := cos.NewStream(filter.Provider{})
	stream.SetItem(cos.Type, cos.XObject)
	stream.SetItem(cos.Subtype, cos.Image)
	stream.SetInt(cos.Width, width)
	stream.SetInt(cos.Height, height)
	stream.SetInt(cos.BitsPerComponent, bitsPerComponent)
	stream.SetItem(cos.ColorSpace, space)
	write(stream, samples)
	return stream
}

// write puts the samples in the stream, deflated so that the file stays small.
func write(stream *cos.Stream, samples []byte) {
	output, err := stream.CreateWriterWithFilters(cos.FlateDecode)
	check(err)
	_, err = output.Write(samples)
	check(err)
	check(output.Close())
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
