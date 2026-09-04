package image

import (
	"errors"
	goimage "image"
	goimagecolor "image/color"
	"io"
	"log/slog"

	awtgeom "github.com/shinguakira/pdfbox-go/go/awt/geom"
	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// PDImageXObject is an image XObject.
//
// Port of org.apache.pdfbox.pdmodel.graphics.image.PDImageXObject.
type PDImageXObject struct {
	graphics.PDXObject

	// cachedImage is Java's SoftReference<BufferedImage>. Go has no soft
	// reference, and the port holds the image outright; a caller that wants the
	// memory back drops the PDImageXObject.
	cachedImage goimage.Image
	colorSpace  color.PDColorSpace
	// initialize to MaxInt as we prefer lower subsampling when keeping or
	// replacing cache.
	cachedImageSubsampling int

	// hasJPXFilter indicates whether this image has an JPX-based filter applied
	hasJPXFilter bool
	// jpxValuesInitialized is set to true after reading some values from a
	// JPX-based image
	jpxValuesInitialized bool
	jpxSMask             goimage.Image

	resources color.ResourcesLike
}

var _ PDImage = (*PDImageXObject)(nil)

const maxSubsampling = int(^uint(0) >> 1)

// NewPDImageXObject reads an image XObject out of its stream.
//
// Port of PDImageXObject(PDStream, PDResources).
func NewPDImageXObject(stream *common.PDStream, resources color.ResourcesLike) *PDImageXObject {
	x := &PDImageXObject{
		PDXObject:              graphics.NewPDXObjectOfPDStream(stream, cos.Image),
		resources:              resources,
		cachedImageSubsampling: maxSubsampling,
	}
	filters := stream.Filters()
	if len(filters) > 0 && filters[len(filters)-1] == cos.JPXDecode {
		x.hasJPXFilter = true
	}
	return x
}

// NewThumbnail reads a thumbnail image.
//
// Port of createThumbnail: thumbnails are special, any non-null subtype is
// treated as being "Image".
func NewThumbnail(cosStream *cos.Stream) *PDImageXObject {
	return NewPDImageXObject(common.NewPDStream(cosStream), nil)
}

// COSDictionary returns the dictionary below this image.
func (x *PDImageXObject) COSDictionary() *cos.Dictionary { return &x.Stream().Dictionary }

// Image returns the whole image, at full size.
func (x *PDImageXObject) Image() (goimage.Image, error) {
	return x.ImageOfRegion(nil, 1)
}

// ImageOfRegion returns part of the image, subsampled.
func (x *PDImageXObject) ImageOfRegion(region *awtgeom.Rectangle,
	subsampling int) (goimage.Image, error) {
	if region == nil && subsampling == x.cachedImageSubsampling && x.cachedImage != nil {
		return x.cachedImage, nil
	}

	x.initJPXValues()

	var image goimage.Image
	var err error
	softMask := x.SoftMask()
	mask := x.Mask()

	switch {
	case x.jpxSMask != nil:
		// PDFBOX-5657: handle JPEG2000 SMaskInData
		var base goimage.Image
		if base, err = getRGBImageOfRegion(x, region, subsampling, x.ColorKeyMask()); err == nil {
			image = applyMask(base, x.jpxSMask, false, true, nil, x.Interpolate())
		}

	case softMask != nil:
		// soft mask (overrides explicit mask)
		var base, maskImage goimage.Image
		if base, err = getRGBImageOfRegion(x, region, subsampling, x.ColorKeyMask()); err == nil {
			if maskImage, err = softMask.OpaqueImageOfRegion(region, subsampling); err == nil {
				var matte []float32
				if matte, err = x.extractMatte(softMask); err == nil {
					image = applyMask(base, maskImage, softMask.Interpolate(), true, matte,
						x.Interpolate())
				}
			}
		}

	case mask != nil && mask.IsStencil():
		// explicit mask - to be applied only if /ImageMask true
		var base, maskImage goimage.Image
		if base, err = getRGBImageOfRegion(x, region, subsampling, x.ColorKeyMask()); err == nil {
			if maskImage, err = mask.OpaqueImageOfRegion(region, subsampling); err == nil {
				image = applyMask(base, maskImage, mask.Interpolate(), false, nil, x.Interpolate())
			}
		}

	default:
		image, err = getRGBImageOfRegion(x, region, subsampling, x.ColorKeyMask())
	}
	if err != nil {
		return nil, err
	}

	if region == nil && subsampling <= x.cachedImageSubsampling {
		// only cache full-image renders, and prefer lower subsampling frequency, as lower
		// subsampling means higher quality and longer render times.
		x.cachedImageSubsampling = subsampling
		x.cachedImage = image
	}
	return image, nil
}

// RawImage returns the image in its own colour space, or nil where there is no
// way to hold one.
func (x *PDImageXObject) RawImage() (goimage.Image, error) {
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

// RawRaster returns the samples of the image, undecoded.
func (x *PDImageXObject) RawRaster() (*awtimage.Raster, error) { return getRawRaster(x) }

func (x *PDImageXObject) extractMatte(softMask *PDImageXObject) ([]float32, error) {
	base := softMask.COSDictionary().GetItem(cos.Matte)
	array, ok := base.(*cos.Array)
	if !ok {
		return nil, nil
	}
	// PDFBOX-4267: process /Matte
	// see PDF specification 1.7, 11.6.5.3 Soft-Mask Images
	matte := array.ToFloatArray()
	colorSpace, err := x.ColorSpace()
	if err != nil {
		return nil, err
	}
	// convert to RGB
	if len(matte) < colorSpace.NumberOfComponents() {
		slog.Error("image: /Matte entry not long enough for colorspace, skipped")
		return nil, nil
	}
	return colorSpace.ToRGB(matte)
}

// StencilImage returns the content of this stencil mask filled with the given
// colour.
//
// Java throws IllegalStateException where the image is not a stencil, which is
// unchecked, so the port panics.
func (x *PDImageXObject) StencilImage(paint goimagecolor.Color) (goimage.Image, error) {
	if !x.IsStencil() {
		panic("Image is not a stencil")
	}
	return getStencilImage(x, paint)
}

// OpaqueImage returns the image without any mask applied.
func (x *PDImageXObject) OpaqueImage() (goimage.Image, error) {
	return x.OpaqueImageOfRegion(nil, 1)
}

// OpaqueImageOfRegion returns part of the image without any mask applied.
func (x *PDImageXObject) OpaqueImageOfRegion(region *awtgeom.Rectangle,
	subsampling int) (goimage.Image, error) {
	return getRGBImageOfRegion(x, region, subsampling, nil)
}

// initJPXValues reads what a JPEG 2000 image carries in its own data rather
// than in its dictionary.
//
// The JPX filter is declared and unsupported in this port -- see the note in
// pdfbox/filter -- so createInputStream fails here and there is nothing to
// read. Java catches the IOException and logs it, which is the same outcome on
// a build without the JAI tools.
func (x *PDImageXObject) initJPXValues() {
	if !x.hasJPXFilter || x.jpxValuesInitialized {
		return
	}
	if _, err := x.PDStream().CreateInputStream(); err != nil {
		slog.Debug("image: can't initialize JPX based values", "err", err)
		return
	}
	// The decode result Java reads the width, height, bits per component and
	// colour space out of comes from the JPX filter, which this port does not
	// have; nothing is learned here and the flag is not set, so a later call
	// tries again exactly as Java's does after a failure.
	slog.Debug("image: JPX images are not decoded by this port")
}

// Mask returns the explicit mask of this image, or nil where there is none.
func (x *PDImageXObject) Mask() *PDImageXObject {
	if mask := x.COSDictionary().GetCOSArray(cos.Mask); mask != nil {
		// color key mask, no explicit mask to return
		return nil
	}
	if cosStream := x.COSDictionary().GetCOSStream(cos.Mask); cosStream != nil {
		// always DeviceGray
		return NewPDImageXObject(common.NewPDStream(cosStream), nil)
	}
	return nil
}

// ColorKeyMask returns the colour key mask of this image, or nil.
func (x *PDImageXObject) ColorKeyMask() *cos.Array {
	return x.COSDictionary().GetCOSArray(cos.Mask)
}

// SoftMask returns the soft mask of this image, or nil where there is none.
func (x *PDImageXObject) SoftMask() *PDImageXObject {
	if cosStream := x.COSDictionary().GetCOSStream(cos.SMask); cosStream != nil {
		// always DeviceGray
		return NewPDImageXObject(common.NewPDStream(cosStream), nil)
	}
	return nil
}

// BitsPerComponent returns how many bits one sample takes.
func (x *PDImageXObject) BitsPerComponent() int {
	if x.IsStencil() {
		return 1
	}
	x.initJPXValues()
	return x.COSDictionary().GetInt2(cos.BitsPerComponent, cos.BPC, 0)
}

// SetBitsPerComponent sets how many bits one sample takes.
func (x *PDImageXObject) SetBitsPerComponent(bpc int) {
	x.COSDictionary().SetInt(cos.BitsPerComponent, bpc)
}

// ColorSpace returns the colour space of this image.
func (x *PDImageXObject) ColorSpace() (color.PDColorSpace, error) {
	if x.colorSpace != nil {
		return x.colorSpace, nil
	}
	cosBase := x.COSDictionary().GetItem2(cos.ColorSpace, cos.CS)
	switch {
	case cosBase != nil:
		var indirect *cos.Object
		if object, ok := cosBase.(*cos.Object); ok && x.resources != nil {
			// PDFBOX-4022: use the resource cache because several images
			// might have the same colorspace indirect object.
			indirect = object
			if cached := x.resources.CachedColorSpace(indirect); cached != nil {
				x.colorSpace = cached
				return x.colorSpace, nil
			}
		}
		space, err := color.CreateOfResources(cosBase, x.resources)
		if err != nil {
			return nil, err
		}
		x.colorSpace = space
		if indirect != nil {
			x.resources.CacheColorSpace(indirect, space)
		}

	case x.IsStencil():
		// stencil mask color space must be gray, it is often missing
		x.colorSpace = color.DeviceGray

	default:
		x.initJPXValues()
	}
	if x.colorSpace == nil {
		// an image without a color space is always broken
		return nil, errors.New("could not determine color space")
	}
	return x.colorSpace, nil
}

// CreateInputStream returns the decoded content of this image.
func (x *PDImageXObject) CreateInputStream() (io.Reader, error) {
	return x.PDStream().CreateInputStream()
}

// CreateInputStreamWithOptions returns the decoded content of this image,
// letting a filter honour the given options.
func (x *PDImageXObject) CreateInputStreamWithOptions(
	options *filter.DecodeOptions) (io.Reader, error) {
	// The port's PDStream has no overload taking DecodeOptions, because no
	// filter here honours them: the two that do in Java are DCT and JBIG2, and
	// both hand the subsampling to an ImageIO reader Go has not got. The caller
	// then sees IsFilterSubsampled false and clips and subsamples itself, which
	// is the path Java takes for every other filter.
	return x.PDStream().CreateInputStream()
}

// CreateInputStreamStopping returns the content of this image, decoded through
// every filter up to but not including the first named one.
func (x *PDImageXObject) CreateInputStreamStopping(stopFilters []string) (io.Reader, error) {
	return x.PDStream().CreateInputStreamStopping(stopFilters)
}

// IsEmpty reports whether this image has no data.
func (x *PDImageXObject) IsEmpty() bool {
	length, err := x.Stream().Length()
	return err != nil || length == 0
}

// SetColorSpace sets the colour space of this image.
func (x *PDImageXObject) SetColorSpace(cs color.PDColorSpace) {
	var base cos.Base
	if cs != nil {
		base = cs.COSObject()
	}
	x.COSDictionary().SetItem(cos.ColorSpace, base)
	x.colorSpace = nil
	x.cachedImage = nil
}

// Height returns the height in pixels.
func (x *PDImageXObject) Height() int {
	x.initJPXValues()
	return x.COSDictionary().GetInt(cos.Height)
}

// SetHeight sets the height in pixels.
func (x *PDImageXObject) SetHeight(h int) { x.COSDictionary().SetInt(cos.Height, h) }

// Width returns the width in pixels.
func (x *PDImageXObject) Width() int {
	x.initJPXValues()
	return x.COSDictionary().GetInt(cos.Width)
}

// SetWidth sets the width in pixels.
func (x *PDImageXObject) SetWidth(w int) { x.COSDictionary().SetInt(cos.Width, w) }

// Interpolate reports whether the image is to be interpolated when scaled.
func (x *PDImageXObject) Interpolate() bool {
	return x.COSDictionary().GetBoolean(cos.Interpolate, false)
}

// SetInterpolate says whether the image is to be interpolated when scaled.
func (x *PDImageXObject) SetInterpolate(value bool) {
	x.COSDictionary().SetBoolean(cos.Interpolate, value)
}

// SetDecode sets the /Decode array.
func (x *PDImageXObject) SetDecode(decode *cos.Array) {
	x.COSDictionary().SetItem(cos.Decode, decode)
}

// Decode returns the /Decode array, or nil where there is none.
func (x *PDImageXObject) Decode() *cos.Array { return x.COSDictionary().GetCOSArray(cos.Decode) }

// IsStencil reports whether this image is a stencil mask.
func (x *PDImageXObject) IsStencil() bool {
	return x.COSDictionary().GetBoolean(cos.ImageMask, false)
}

// SetStencil says whether this image is a stencil mask.
func (x *PDImageXObject) SetStencil(isStencil bool) {
	x.COSDictionary().SetBoolean(cos.ImageMask, isStencil)
}

// StructParent returns the /StructParent key.
func (x *PDImageXObject) StructParent() int { return x.COSDictionary().GetInt(cos.StructParent) }

// SetStructParent sets the /StructParent key.
func (x *PDImageXObject) SetStructParent(key int) {
	x.COSDictionary().SetInt(cos.StructParent, key)
}

// Suffix returns the file suffix the image data would have on its own.
//
// Java returns null where it does not know, which the port reports as the empty
// string; a caller wanting Java's null tests for it.
func (x *PDImageXObject) Suffix() string {
	filters := x.PDStream().Filters()
	switch {
	case len(filters) == 0:
		return "png"
	case containsName(filters, cos.DCTDecode):
		return "jpg"
	case containsName(filters, cos.JPXDecode):
		return "jpx"
	case containsName(filters, cos.CCITTFaxDecode):
		return "tiff"
	case containsName(filters, cos.FlateDecode),
		containsName(filters, cos.LZWDecode),
		containsName(filters, cos.RunLengthDecode):
		return "png"
	case containsName(filters, cos.JBIG2Decode):
		return "jb2"
	}
	slog.Warn("image: Suffix returns nothing", "filters", filters)
	return ""
}

func containsName(names []*cos.Name, want *cos.Name) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
