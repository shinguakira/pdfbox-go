//go:build ignore

// Writes softmaskimage.pdf, the page the images-under-a-soft-mask comparison
// renders.
//
// Regenerate it and its reference together:
//
//	go run testdata/gensoftmaskimage.go
//	javac ... RenderDrv.java && java ... RenderDrv softmaskimage.pdf softmaskimage-java.png
//
// PageDrawer.drawBufferedImage draws an image under the graphics state's soft
// mask as a TexturePaint seen through the mask, filled over the image's
// rectangle -- unless the image has a mask of its own, a /Mask or an /SMask,
// which "shall override the current soft mask in the graphics state"
// (PDFBOX-5307). A stencil filled with a colour goes the same way: drawImage
// makes it an image with getStencilImage and hands that to drawBufferedImage.
// The page has one soft mask, a luminosity ramp from white at the left edge to
// black at the right, and seven images drawn with and without it: a plain
// image at one to one and at one and a half, an image with an /SMask of its
// own, one with a colour key /Mask, a stencil filled with red, a stencil
// filled with blue that has an /SMask key, and the plain image with no soft
// mask at all.
//
// The ramp runs from white so that the stencils show: getStencilImage fills a
// stencil with a paint that already carries the soft mask, in the stencil's
// own pixels, which reads the mask at the left edge of the page.
package main

import (
	"fmt"
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

	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(200, 170))
	document.AddPage(page)

	resources := pdmodel.NewPDResources()
	page.SetResources(resources)

	states := cos.NewDictionary()
	states.SetItem(cos.GetPDFName("GsMask"), softMaskState(rampGroup(0, 0, 200, 170)))
	resources.Dictionary().SetItem(cos.ExtGState, states)

	xobjects := cos.NewDictionary()
	xobjects.SetItem(cos.GetPDFName("Plain"), rgbImage())
	xobjects.SetItem(cos.GetPDFName("OwnSMask"), withSoftMask())
	xobjects.SetItem(cos.GetPDFName("OwnMask"), withColourKey())
	xobjects.SetItem(cos.GetPDFName("Stencil"), stencil())
	xobjects.SetItem(cos.GetPDFName("StencilOwnSMask"), stencilWithSoftMask())
	resources.Dictionary().SetItem(cos.XObject, xobjects)

	content := `
q /GsMask gs 40 0 0 30 10 80 cm /Plain Do Q
q /GsMask gs 60 0 0 45 60 65 cm /Plain Do Q
q /GsMask gs 40 0 0 30 140 80 cm /OwnSMask Do Q
q /GsMask gs 40 0 0 30 10 20 cm /OwnMask Do Q
q /GsMask gs 0.8 0.1 0.1 rg 40 0 0 30 75 20 cm /Stencil Do Q
q /GsMask gs 0.1 0.3 0.8 rg 40 0 0 30 75 125 cm /StencilOwnSMask Do Q
q 40 0 0 30 140 20 cm /Plain Do Q
`
	stream := cos.NewStream(nil)
	output, err := stream.CreateWriter()
	check(err)
	_, err = output.Write([]byte(content))
	check(err)
	check(output.Close())
	page.Dictionary().SetItem(cos.Contents, stream)

	out, err := os.Create("testdata/softmaskimage.pdf")
	check(err)
	check(document.Save(out))
	check(out.Close())
}

// rgbImage is 40 by 30 of colour that changes from pixel to pixel.
func rgbImage() *cos.Stream {
	const width, height = 40, 30
	samples := make([]byte, 0, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			samples = append(samples, byte(x*37+y*11), byte(x*x+y*3), byte((x^y)*9))
		}
	}
	return image(width, height, cos.DeviceRGB, samples)
}

// withSoftMask is the RGB image with an /SMask of its own, fading it from the
// top down.
func withSoftMask() *cos.Stream {
	const width, height = 40, 30
	alpha := make([]byte, 0, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			alpha = append(alpha, byte(255-y*255/(height-1)))
		}
	}
	stream := rgbImage()
	stream.SetItem(cos.SMask, image(width, height, cos.DeviceGray, alpha))
	return stream
}

// withColourKey is the RGB image with a colour key /Mask that takes out every
// pixel whose red is 60 or less.
func withColourKey() *cos.Stream {
	stream := rgbImage()
	key := cos.NewArray()
	for _, v := range []int64{0, 60, 0, 255, 0, 255} {
		key.Add(cos.GetInteger(v))
	}
	stream.SetItem(cos.Mask, key)
	return stream
}

// image is an 8 bit image XObject, deflated.
func image(width, height int, space *cos.Name, samples []byte) *cos.Stream {
	stream := cos.NewStream(filter.Provider{})
	stream.SetItem(cos.Type, cos.XObject)
	stream.SetItem(cos.Subtype, cos.Image)
	stream.SetInt(cos.Width, width)
	stream.SetInt(cos.Height, height)
	stream.SetInt(cos.BitsPerComponent, 8)
	stream.SetItem(cos.ColorSpace, space)
	output, err := stream.CreateWriterWithFilters(cos.FlateDecode)
	check(err)
	_, err = output.Write(samples)
	check(err)
	check(output.Close())
	return stream
}

// softMaskState is an ExtGState whose /SMask is a luminosity mask of the
// given group.
func softMaskState(group *cos.Stream) *cos.Dictionary {
	mask := cos.NewDictionary()
	mask.SetItem(cos.Type, cos.GetPDFName("Mask"))
	mask.SetItem(cos.S, cos.GetPDFName("Luminosity"))
	mask.SetItem(cos.G, group)

	state := cos.NewDictionary()
	state.SetItem(cos.Type, cos.ExtGState)
	state.SetItem(cos.SMask, mask)
	return state
}

// rampGroup is a group painted with an axial shading from white to black, so
// its luminosity runs from 1 to 0 across the box.
func rampGroup(x, y, w, h float32) *cos.Stream {
	shadings := cos.NewDictionary()
	shadings.SetItem(cos.GetPDFName("Sh0"), rampShading(x, w))
	resources := cos.NewDictionary()
	resources.SetItem(cos.Shading, shadings)

	stream := cos.NewStream(nil)
	stream.Dictionary.SetItem(cos.Type, cos.XObject)
	stream.Dictionary.SetItem(cos.Subtype, cos.Form)
	stream.Dictionary.SetItem(cos.BBox, floats(x, y, x+w, y+h))
	stream.Dictionary.SetItem(cos.Resources, resources)
	attributes := cos.NewDictionary()
	attributes.SetItem(cos.S, cos.Transparency)
	attributes.SetItem(cos.CS, cos.DeviceGray)
	stream.Dictionary.SetItem(cos.Group, attributes)

	output, err := stream.CreateWriter()
	check(err)
	_, err = output.Write([]byte(fmt.Sprintf("q %g %g %g %g re W n /Sh0 sh Q\n", x, y, w, h)))
	check(err)
	check(output.Close())
	return stream
}

// rampShading is an axial shading in DeviceGray from white to black.
func rampShading(x, w float32) *cos.Dictionary {
	f := cos.NewDictionary()
	f.SetInt(cos.FunctionType, 2)
	f.SetItem(cos.Domain, floats(0, 1))
	f.SetItem(cos.C0, floats(1))
	f.SetItem(cos.C1, floats(0))
	f.SetInt(cos.N, 1)

	shading := cos.NewDictionary()
	shading.SetInt(cos.ShadingType, 2)
	shading.SetItem(cos.ColorSpace, cos.DeviceGray)
	shading.SetItem(cos.Coords, floats(x, 0, x+w, 0))
	shading.SetItem(cos.Function, f)
	extend := cos.NewArray()
	extend.Add(cos.GetBoolean(true))
	extend.Add(cos.GetBoolean(true))
	shading.SetItem(cos.Extend, extend)
	return shading
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

// stencilWithSoftMask is the stencil with an /SMask key, which a stencil has no
// use for; drawBufferedImage only asks whether the key is there.
func stencilWithSoftMask() *cos.Stream {
	const width, height = 40, 30
	alpha := make([]byte, width*height)
	for k := range alpha {
		alpha[k] = 0xFF
	}
	stream := stencil()
	stream.SetItem(cos.SMask, image(width, height, cos.DeviceGray, alpha))
	return stream
}

// stencil is a 40 by 30 image mask of diagonal bands, each four samples wide.
// A sample of 0 is painted, which is the default /Decode.
func stencil() *cos.Stream {
	const width, height = 40, 30
	const rowBytes = (width + 7) / 8
	samples := make([]byte, rowBytes*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x+y)/4%2 == 1 {
				samples[y*rowBytes+x/8] |= 0x80 >> (x % 8)
			}
		}
	}
	stream := cos.NewStream(filter.Provider{})
	stream.SetItem(cos.Type, cos.XObject)
	stream.SetItem(cos.Subtype, cos.Image)
	stream.SetInt(cos.Width, width)
	stream.SetInt(cos.Height, height)
	stream.SetInt(cos.BitsPerComponent, 1)
	stream.SetItem(cos.ImageMask, cos.GetBoolean(true))
	output, err := stream.CreateWriterWithFilters(cos.FlateDecode)
	check(err)
	_, err = output.Write(samples)
	check(err)
	check(output.Close())
	return stream
}
