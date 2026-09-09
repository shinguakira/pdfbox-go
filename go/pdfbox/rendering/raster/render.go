package raster

import (
	goimage "image"

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
	renderer := rendering.NewPDFRenderer(document)
	width, height, surfaceType, err := renderer.SurfaceSizeOfPage(pageIndex, scale, imageType)
	if err != nil {
		return nil, err
	}
	backend := NewImage(width, height, surfaceType)
	renderer.SetBackend(backend, surfaceType == rendering.Binary)
	if err := renderer.RenderImageOfType(pageIndex, scale, imageType); err != nil {
		return nil, err
	}
	return backend.Image(), nil
}
