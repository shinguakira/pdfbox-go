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
		return onWhite(backend.dst), nil
	}
	return backend.Image(), nil
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
func onWhite(surface *goimage.RGBA) *goimage.RGBA {
	bounds := surface.Bounds()
	flattened := goimage.NewRGBA(bounds)
	fillOpaqueWhite(flattened)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			source := surface.RGBAAt(x, y)
			if source.A == 0 {
				continue
			}
			if source.A == 0xFF {
				flattened.SetRGBA(x, y, source)
				continue
			}
			alpha := uint32(source.A)
			over := func(from, onto uint8) uint8 {
				return uint8((uint32(from)*alpha + uint32(onto)*(255-alpha) + 127) / 255)
			}
			flattened.SetRGBA(x, y, goimagecolor.RGBA{
				R: over(source.R, 0xFF),
				G: over(source.G, 0xFF),
				B: over(source.B, 0xFF),
				A: 0xFF,
			})
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
