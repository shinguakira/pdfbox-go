package raster

import (
	goimage "image"
	goimagecolor "image/color"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// RenderPage renders one page of a document and answers the pixels.
//
// This is Java's PDFRenderer.renderImage(int, float, ImageType), which makes a
// BufferedImage, draws onto it and hands it back. The port splits those in two
// -- the surface belongs to the caller, so PDFRenderer draws through a Backend
// that is already there -- and this puts them back together for a caller that
// only wants an image.
func RenderPage(document *pdmodel.PDDocument, pageIndex int, scale float32,
	imageType rendering.ImageType) (goimage.Image, error) {
	return renderPage(rendering.NewPDFRenderer(document), document, pageIndex, scale, imageType)
}

// renderPage is what the two entry points share.
func renderPage(renderer *rendering.PDFRenderer, document *pdmodel.PDDocument,
	pageIndex int, scale float32, imageType rendering.ImageType) (goimage.Image, error) {
	width, height, surfaceType, err := renderer.SurfaceSizeOfPage(pageIndex, scale, imageType)
	if err != nil {
		return nil, err
	}
	backend := NewImage(width, height, surfaceType)
	renderer.SetBackend(backend, surfaceType == rendering.Binary)
	if err := renderer.RenderImageOfType(pageIndex, scale, imageType); err != nil {
		return nil, err
	}
	if surfaceType != imageType {
		return asImageType(onWhite(backend.dst, imageType), imageType), nil
	}
	return asImageType(backend.dst, imageType), nil
}

// onWhite puts a transparent surface onto a white one.
//
// SurfaceSizeOfPage answers ARGB for a page that blends at the top level even
// where the caller asked for something opaque -- PDFBOX-4095, which draws the
// page on nothing so that a blend mode has nothing of the page's own ground to
// blend with. Java's renderImage ends by putting that surface on white, and
// this is that:
//
//	dstGraphics.setBackground(Color.WHITE);
//	dstGraphics.clearRect(0, 0, image.getWidth(), image.getHeight());
//	dstGraphics.drawImage(image, 0, 0, null);
//
// which is source-over onto opaque white, and needs none of the machinery of
// DrawImage because there is no transform, no clip and no interpolation.
func onWhite(surface *goimage.NRGBA, imageType rendering.ImageType) *goimage.NRGBA {
	bounds := surface.Bounds()
	flattened := goimage.NewNRGBA(bounds)
	fillOpaqueWhite(flattened)
	// The new image is of the type that was asked for, so it quantizes what is
	// drawn onto it -- which for this path is everything at once, at the end.
	into := &Image{dst: flattened, imageType: imageType}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			source := surface.NRGBAAt(x, y)
			if source.A == 0 {
				continue
			}
			if source.A == 0xFF {
				flattened.SetNRGBA(x, y, into.quantizeColor(source))
				continue
			}
			alpha := uint32(source.A)
			over := func(from, onto uint8) uint8 {
				return uint8((uint32(from)*alpha + uint32(onto)*(255-alpha) + 127) / 255)
			}
			flattened.SetNRGBA(x, y, into.quantizeColor(goimagecolor.NRGBA{
				R: over(source.R, 0xFF),
				G: over(source.G, 0xFF),
				B: over(source.B, 0xFF),
				A: 0xFF,
			}))
		}
	}
	return flattened
}

// RenderPageWithDPI renders one page at the given DPI and answers the pixels.
//
// This is Java's PDFRenderer.renderImageWithDPI(int, float, ImageType), which
// is renderImage at `dpi / 72`, plus the setSubsamplingAllowed a caller would
// have made on the renderer first.
func RenderPageWithDPI(document *pdmodel.PDDocument, pageIndex int, dpi float32,
	imageType rendering.ImageType, subsamplingAllowed bool) (goimage.Image, error) {
	renderer := rendering.NewPDFRenderer(document)
	renderer.SetSubsamplingAllowed(subsamplingAllowed)
	return renderPage(renderer, document, pageIndex, dpi/72, imageType)
}

// asImageType answers the surface as the kind of image Java's renderImage
// would have made.
//
// The pixels are already quantized -- quantize.go does that as they are
// written -- so this only changes what holds them, which is what the caller
// gets and what an encoder writes out. Java's ImageType.toBufferedImageType
// names TYPE_BYTE_GRAY and TYPE_BYTE_BINARY, and ImageIO writes those as a
// greyscale and a one-bit PNG; handing back an RGBA image of grey pixels would
// draw the same picture into a file three times the size.
func asImageType(surface *goimage.NRGBA, imageType rendering.ImageType) goimage.Image {
	bounds := surface.Bounds()
	switch imageType {
	case rendering.Gray:
		gray := goimage.NewGray(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				// Every channel is the same by now, so any of them is the grey.
				gray.SetGray(x, y, goimagecolor.Gray{Y: surface.NRGBAAt(x, y).R})
			}
		}
		return gray

	case rendering.Binary:
		// The two-entry colour model of TYPE_BYTE_BINARY, in the order Java
		// builds it: index 0 black, index 1 white.
		binary := goimage.NewPaletted(bounds, goimagecolor.Palette{
			goimagecolor.Gray{Y: 0}, goimagecolor.Gray{Y: 0xFF},
		})
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				index := uint8(0)
				if surface.NRGBAAt(x, y).R != 0 {
					index = 1
				}
				binary.SetColorIndex(x, y, index)
			}
		}
		return binary
	}
	return surface
}
