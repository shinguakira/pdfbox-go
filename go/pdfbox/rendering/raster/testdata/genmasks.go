//go:build ignore

// Writes masks.pdf, the page the soft-mask comparison renders.
//
// Regenerate it and its reference together:
//
//	go run testdata/genmasks.go
//	javac ... RenderDrv.java && java ... RenderDrv masks.pdf masks-java.png
//
// A soft mask is the deepest thing a Backend is asked for: the mask is a
// transparency group, so building it means running a second content stream
// through the drawer, into a surface of its own, at its own scale, and then
// reading either its luminosity or its alpha back as an alpha channel. Both
// subtypes are here, because they read different channels, and one of them
// carries a backdrop colour and a transfer function, because those are the two
// things that change what a grey means.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

func main() {
	write("testdata/masks.pdf", 0)
	// The same page turned a quarter turn. A rotated page is the one thing
	// that makes PageDrawer.adjustImage do anything: it rescales the mask by
	// the device transform with its own scaling taken out, and for a page whose
	// transform is a plain scale that is the identity and Java short-circuits.
	write("testdata/masksrot.pdf", 90)
}

func write(path string, rotation int) {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(200, 120))
	page.SetRotation(rotation)
	document.AddPage(page)

	resources := pdmodel.NewPDResources()
	page.SetResources(resources)

	states := cos.NewDictionary()
	// A luminosity mask: a grey ramp from black to white across the box, so
	// the paint under it fades out to the left.
	states.SetItem(cos.GetPDFName("GsLum"),
		softMaskState("Luminosity", rampGroup(10, 70, 90, 40), nil))
	// An alpha mask: the same group, read for its alpha instead, which the
	// ramp does not have -- so it masks by where the group drew, not by how
	// bright it is.
	states.SetItem(cos.GetPDFName("GsAlpha"),
		softMaskState("Alpha", alphaGroup(105, 70, 90, 40), nil))
	// A luminosity mask with a backdrop colour, which is what the group is
	// composited onto and so what a pixel it never reached is worth.
	backdrop := cos.NewArray()
	backdrop.Add(cos.NewFloat(0.5))
	states.SetItem(cos.GetPDFName("GsBackdrop"),
		softMaskState("Luminosity", halfGroup(10, 10, 90, 40), backdrop))
	// A luminosity mask whose group is in DeviceRGB, which is the arm where
	// Java has a colour to convert: isGray is false, so the group is drawn
	// into an ARGB image and read back through the JDK's colour management.
	states.SetItem(cos.GetPDFName("GsRGB"),
		softMaskState("Luminosity", rgbGroup(105, 10, 90, 40), nil))
	resources.Dictionary().SetItem(cos.ExtGState, states)

	content := `
q /GsLum gs 1 0 0 rg 10 70 90 40 re f Q
q /GsAlpha gs 0 0.5 0 rg 105 70 90 40 re f Q
q /GsBackdrop gs 0 0 1 rg 10 10 90 40 re f Q
q /GsRGB gs 0.2 0.2 0.2 rg 105 10 90 40 re f Q
`
	stream := cos.NewStream(nil)
	output, err := stream.CreateWriter()
	check(err)
	_, err = output.Write([]byte(content))
	check(err)
	check(output.Close())
	page.Dictionary().SetItem(cos.Contents, stream)

	out, err := os.Create(path)
	check(err)
	check(document.Save(out))
	check(out.Close())
}

// softMaskState is an ExtGState whose /SMask names the given group.
func softMaskState(subType string, group *cos.Stream, backdrop *cos.Array) *cos.Dictionary {
	mask := cos.NewDictionary()
	mask.SetItem(cos.Type, cos.GetPDFName("Mask"))
	mask.SetItem(cos.S, cos.GetPDFName(subType))
	mask.SetItem(cos.G, group)
	if backdrop != nil {
		mask.SetItem(cos.BC, backdrop)
	}

	state := cos.NewDictionary()
	state.SetItem(cos.Type, cos.ExtGState)
	state.SetItem(cos.SMask, mask)
	return state
}

// rampGroup is a group painted with an axial shading from black to white, so
// its luminosity runs from 0 to 1 across the box.
func rampGroup(x, y, w, h float32) *cos.Stream {
	shadings := cos.NewDictionary()
	shadings.SetItem(cos.GetPDFName("Sh0"), rampShading(x, w))
	resources := cos.NewDictionary()
	resources.SetItem(cos.Shading, shadings)
	return group(x, y, w, h, resources,
		fmt.Sprintf("q %g %g %g %g re W n /Sh0 sh Q\n", x, y, w, h))
}

// alphaGroup draws two opaque boxes and leaves the rest of the group untouched,
// so its alpha is 1 where it drew and 0 elsewhere.
func alphaGroup(x, y, w, h float32) *cos.Stream {
	return group(x, y, w, h, cos.NewDictionary(),
		fmt.Sprintf("1 g %g %g 40 40 re f 0.5 g %g %g 40 20 re f\n", x, y, x+50, y+10))
}

// halfGroup covers only part of its box, so the rest of the masked paint is
// worth whatever the backdrop colour says.
func halfGroup(x, y, w, h float32) *cos.Stream {
	return group(x, y, w, h, cos.NewDictionary(),
		fmt.Sprintf("1 g %g %g 45 40 re f\n", x, y))
}

// rgbGroup is the same ramp in DeviceRGB, so that the mask has to convert a
// colour rather than read a channel.
func rgbGroup(x, y, w, h float32) *cos.Stream {
	shadings := cos.NewDictionary()
	shadings.SetItem(cos.GetPDFName("Sh0"), rgbRampShading(x, w))
	resources := cos.NewDictionary()
	resources.SetItem(cos.Shading, shadings)
	stream := group(x, y, w, h, resources,
		fmt.Sprintf("q %g %g %g %g re W n /Sh0 sh Q\n", x, y, w, h))
	attributes := stream.Dictionary.GetCOSDictionary(cos.Group)
	attributes.SetItem(cos.CS, cos.DeviceRGB)
	return stream
}

// rgbRampShading runs from black to a saturated colour, so the conversion has
// something other than a grey to work on.
func rgbRampShading(x, w float32) *cos.Dictionary {
	f := cos.NewDictionary()
	f.SetInt(cos.FunctionType, 2)
	f.SetItem(cos.Domain, floats(0, 1))
	f.SetItem(cos.C0, floats(0, 0, 1))
	f.SetItem(cos.C1, floats(1, 1, 0))
	f.SetInt(cos.N, 1)

	shading := cos.NewDictionary()
	shading.SetInt(cos.ShadingType, 2)
	shading.SetItem(cos.ColorSpace, cos.DeviceRGB)
	shading.SetItem(cos.Coords, floats(x, 0, x+w, 0))
	shading.SetItem(cos.Function, f)
	shading.SetItem(cos.Extend, booleans(true, true))
	return shading
}

// group is a transparency group form XObject over the given box.
func group(x, y, w, h float32,
	resources *cos.Dictionary, content string) *cos.Stream {
	stream := cos.NewStream(nil)
	stream.Dictionary.SetItem(cos.Type, cos.XObject)
	stream.Dictionary.SetItem(cos.Subtype, cos.Form)
	stream.Dictionary.SetItem(cos.BBox, box(x, y, w, h))
	stream.Dictionary.SetItem(cos.Resources, resources)

	attributes := cos.NewDictionary()
	attributes.SetItem(cos.S, cos.Transparency)
	attributes.SetItem(cos.CS, cos.DeviceGray)
	stream.Dictionary.SetItem(cos.Group, attributes)

	output, err := stream.CreateWriter()
	check(err)
	_, err = output.Write([]byte(content))
	check(err)
	check(output.Close())
	return stream
}

// rampShading is an axial shading in DeviceGray from black to white.
func rampShading(x, w float32) *cos.Dictionary {
	f := cos.NewDictionary()
	f.SetInt(cos.FunctionType, 2)
	f.SetItem(cos.Domain, floats(0, 1))
	f.SetItem(cos.C0, floats(0))
	f.SetItem(cos.C1, floats(1))
	f.SetInt(cos.N, 1)

	shading := cos.NewDictionary()
	shading.SetInt(cos.ShadingType, 2)
	shading.SetItem(cos.ColorSpace, cos.DeviceGray)
	shading.SetItem(cos.Coords, floats(x, 0, x+w, 0))
	shading.SetItem(cos.Function, f)
	shading.SetItem(cos.Extend, booleans(true, true))
	return shading
}

func box(x, y, w, h float32) *cos.Array { return floats(x, y, x+w, y+h) }

func floats(values ...float32) *cos.Array {
	array := cos.NewArray()
	for _, v := range values {
		array.Add(cos.NewFloat(v))
	}
	return array
}

func booleans(values ...bool) *cos.Array {
	array := cos.NewArray()
	for _, v := range values {
		array.Add(cos.GetBoolean(v))
	}
	return array
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
