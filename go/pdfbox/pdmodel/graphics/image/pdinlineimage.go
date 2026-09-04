package image

import (
	"bytes"
	"errors"
	"fmt"
	goimage "image"
	goimagecolor "image/color"
	"io"

	awtgeom "github.com/shinguakira/pdfbox-go/go/awt/geom"
	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// PDInlineImage is an inline image, written into a content stream between BI
// and EI rather than held as an XObject.
//
// Port of org.apache.pdfbox.pdmodel.graphics.image.PDInlineImage.
type PDInlineImage struct {
	parameters *cos.Dictionary
	resources  color.ResourcesLike

	// image data
	rawData     []byte
	decodedData []byte
}

var _ PDImage = (*PDInlineImage)(nil)

// NewPDInlineImage reads an inline image out of its parameters and data.
func NewPDInlineImage(parameters *cos.Dictionary, data []byte,
	resources color.ResourcesLike) (*PDInlineImage, error) {
	x := &PDInlineImage{parameters: parameters, resources: resources, rawData: data}

	filters := x.Filters()
	if len(filters) == 0 {
		x.decodedData = data
		return x, nil
	}

	in := bytes.NewReader(data)
	var ba []byte
	var lastResult filter.DecodeResult
	haveResult := false
	for i := 0; i < len(filters); i++ {
		// TODO handling of abbreviated names belongs here, rather than in other classes
		var out bytes.Buffer
		f, err := filter.ByName(cos.GetPDFName(filters[i]))
		if err != nil {
			return nil, err
		}
		result, err := f.Decode(&out, in, parameters, i)
		if err != nil {
			return nil, err
		}
		lastResult = result
		haveResult = true
		ba = out.Bytes()
		in = bytes.NewReader(ba)
	}
	x.decodedData = ba

	// repair parameters
	if haveResult && lastResult.Parameters != nil && lastResult.Parameters != parameters {
		parameters.AddAll(lastResult.Parameters)
	}
	return x, nil
}

// COSDictionary returns the parameters of this inline image.
func (x *PDInlineImage) COSDictionary() *cos.Dictionary { return x.parameters }

// BitsPerComponent returns how many bits one sample takes.
func (x *PDInlineImage) BitsPerComponent() int {
	if x.IsStencil() {
		return 1
	}
	return x.parameters.GetInt2(cos.BPC, cos.BitsPerComponent, -1)
}

// SetBitsPerComponent sets how many bits one sample takes.
func (x *PDInlineImage) SetBitsPerComponent(bitsPerComponent int) {
	x.parameters.SetInt(cos.BPC, bitsPerComponent)
}

// ColorSpace returns the colour space of this image.
func (x *PDInlineImage) ColorSpace() (color.PDColorSpace, error) {
	if cs := x.parameters.GetDictionaryObject2(cos.CS, cos.ColorSpace); cs != nil {
		return x.createColorSpace(cs)
	}
	if x.IsStencil() {
		// stencil mask color space must be gray, it is often missing
		return color.DeviceGray, nil
	}
	// an image without a color space is always broken
	return nil, errors.New("could not determine inline image color space")
}

// toLongName delivers the long name of a device colorspace, or the parameter.
func toLongName(cs cos.Base) cos.Base {
	switch cs {
	case cos.Base(cos.RGB):
		return cos.DeviceRGB
	case cos.Base(cos.CMYK):
		return cos.DeviceCMYK
	case cos.Base(cos.G):
		return cos.DeviceGray
	}
	return cs
}

func (x *PDInlineImage) createColorSpace(cs cos.Base) (color.PDColorSpace, error) {
	if _, isName := cs.(*cos.Name); isName {
		return color.CreateOfResources(toLongName(cs), x.resources)
	}
	if srcArray, isArray := cs.(*cos.Array); isArray && srcArray.Size() > 1 {
		csType := srcArray.Get(0)
		if csType == cos.Base(cos.I) || csType == cos.Base(cos.Indexed) {
			dstArray := cos.NewArray()
			dstArray.AddArray(srcArray)
			dstArray.Set(0, cos.Indexed)
			dstArray.Set(1, toLongName(srcArray.Get(1)))
			return color.CreateOfResources(dstArray, x.resources)
		}
		return nil, fmt.Errorf("Illegal type of inline image color space: %v", csType)
	}
	return nil, fmt.Errorf("Illegal type of object for inline image color space: %v", cs)
}

// SetColorSpace sets the colour space of this image.
func (x *PDInlineImage) SetColorSpace(colorSpace color.PDColorSpace) {
	var base cos.Base
	if colorSpace != nil {
		base = colorSpace.COSObject()
	}
	x.parameters.SetItem(cos.CS, base)
}

// Height returns the height in pixels.
func (x *PDInlineImage) Height() int { return x.parameters.GetInt2(cos.H, cos.Height, -1) }

// SetHeight sets the height in pixels.
func (x *PDInlineImage) SetHeight(height int) { x.parameters.SetInt(cos.H, height) }

// Width returns the width in pixels.
func (x *PDInlineImage) Width() int { return x.parameters.GetInt2(cos.W, cos.Width, -1) }

// SetWidth sets the width in pixels.
func (x *PDInlineImage) SetWidth(width int) { x.parameters.SetInt(cos.W, width) }

// Interpolate reports whether the image is to be interpolated when scaled.
func (x *PDInlineImage) Interpolate() bool {
	return x.parameters.GetBoolean2(cos.I, cos.Interpolate, false)
}

// SetInterpolate says whether the image is to be interpolated when scaled.
func (x *PDInlineImage) SetInterpolate(value bool) {
	x.parameters.SetBoolean(cos.I, value)
}

// Filters returns the names of the filters the data is encoded with.
func (x *PDInlineImage) Filters() []string {
	filters := x.parameters.GetDictionaryObject2(cos.F, cos.Filter)
	switch value := filters.(type) {
	case *cos.Name:
		return []string{value.Name()}
	case *cos.Array:
		names := make([]string, 0, value.Size())
		for i := 0; i < value.Size(); i++ {
			if name, ok := value.GetObject(i).(*cos.Name); ok {
				names = append(names, name.Name())
			}
		}
		return names
	}
	return nil
}

// SetFilters sets the names of the filters the data is encoded with.
func (x *PDInlineImage) SetFilters(filters []string) {
	x.parameters.SetItem(cos.F, cos.ArrayOfNames(filters))
}

// SetDecode sets the /D array.
func (x *PDInlineImage) SetDecode(decode *cos.Array) { x.parameters.SetItem(cos.D, decode) }

// Decode returns the /D array, or nil where there is none.
func (x *PDInlineImage) Decode() *cos.Array {
	if decode, ok := x.parameters.GetDictionaryObject2(cos.D, cos.Decode).(*cos.Array); ok {
		return decode
	}
	return nil
}

// IsStencil reports whether this image is a stencil mask.
func (x *PDInlineImage) IsStencil() bool {
	return x.parameters.GetBoolean2(cos.IM, cos.ImageMask, false)
}

// SetStencil says whether this image is a stencil mask.
func (x *PDInlineImage) SetStencil(isStencil bool) {
	x.parameters.SetBoolean(cos.IM, isStencil)
}

// CreateInputStream returns the decoded content of this image.
func (x *PDInlineImage) CreateInputStream() (io.Reader, error) {
	return bytes.NewReader(x.decodedData), nil
}

// CreateInputStreamWithOptions returns the decoded content of this image.
//
// Decode options are irrelevant for inline image, as the data is always
// buffered.
func (x *PDInlineImage) CreateInputStreamWithOptions(
	options *filter.DecodeOptions) (io.Reader, error) {
	return x.CreateInputStream()
}

// CreateInputStreamStopping returns the content of this image, decoded through
// every filter up to but not including the first named one.
func (x *PDInlineImage) CreateInputStreamStopping(stopFilters []string) (io.Reader, error) {
	filters := x.Filters()
	in := bytes.NewReader(x.rawData)
	for i := 0; i < len(filters); i++ {
		if containsString(stopFilters, filters[i]) {
			break
		}
		// TODO handling of abbreviated names belongs here, rather than in other classes
		f, err := filter.ByName(cos.GetPDFName(filters[i]))
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if _, err := f.Decode(&out, in, x.parameters, i); err != nil {
			return nil, err
		}
		in = bytes.NewReader(out.Bytes())
	}
	return in, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// IsEmpty reports whether this image has no data.
func (x *PDInlineImage) IsEmpty() bool { return len(x.decodedData) == 0 }

// Data returns the decoded content of this image.
func (x *PDInlineImage) Data() []byte { return x.decodedData }

// Image returns the whole image, at full size.
func (x *PDInlineImage) Image() (goimage.Image, error) { return getRGBImage(x, nil) }

// ImageOfRegion returns part of the image, subsampled.
func (x *PDInlineImage) ImageOfRegion(region *awtgeom.Rectangle,
	subsampling int) (goimage.Image, error) {
	return getRGBImageOfRegion(x, region, subsampling, nil)
}

// RawRaster returns the samples of the image, undecoded.
func (x *PDInlineImage) RawRaster() (*awtimage.Raster, error) { return getRawRaster(x) }

// RawImage returns the image in its own colour space, or nil where there is no
// way to hold one.
func (x *PDInlineImage) RawImage() (goimage.Image, error) {
	colorSpace, err := x.ColorSpace()
	if err != nil {
		return nil, err
	}
	raster, err := x.RawRaster()
	if err != nil {
		return nil, err
	}
	return colorSpace.ToRawImage(raster)
}

// StencilImage returns the content of this stencil mask filled with the given
// colour.
//
// Java throws IllegalStateException where the image is not a stencil, which is
// unchecked, so the port panics.
func (x *PDInlineImage) StencilImage(paint goimagecolor.Color) (goimage.Image, error) {
	if !x.IsStencil() {
		panic("Image is not a stencil")
	}
	return getStencilImage(x, paint)
}

// Suffix returns the file suffix the image data would have on its own.
func (x *PDInlineImage) Suffix() string {
	filters := x.Filters()
	if len(filters) == 0 {
		return "png"
	}
	if containsString(filters, cos.DCTDecode.Name()) ||
		containsString(filters, cos.DCT.Name()) {
		return "jpg"
	}
	if containsString(filters, cos.CCITTFaxDecode.Name()) ||
		containsString(filters, cos.CCF.Name()) {
		return "tiff"
	}
	// JPX and JBIG2 don't exist for inline images
	return "png"
}
