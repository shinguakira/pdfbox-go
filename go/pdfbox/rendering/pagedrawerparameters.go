package rendering

// What a page drawer is built with.
//
// Port of org.apache.pdfbox.rendering.PageDrawerParameters.

import "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"

// PageDrawerParameters carries the settings a PageDrawer is built from.
//
// Port of PageDrawerParameters, which Java declares final and package-private
// to construct.
type PageDrawerParameters struct {
	renderer                              *PDFRenderer
	page                                  *pdmodel.PDPage
	subsamplingAllowed                    bool
	destination                           RenderDestination
	renderingHints                        RenderingHints
	imageDownscalingOptimizationThreshold float32
}

// newPageDrawerParameters returns the parameters, which only PDFRenderer builds.
func newPageDrawerParameters(renderer *PDFRenderer, page *pdmodel.PDPage,
	subsamplingAllowed bool, destination RenderDestination, renderingHints RenderingHints,
	imageDownscalingOptimizationThreshold float32) PageDrawerParameters {
	return PageDrawerParameters{
		renderer:                              renderer,
		page:                                  page,
		subsamplingAllowed:                    subsamplingAllowed,
		destination:                           destination,
		renderingHints:                        renderingHints,
		imageDownscalingOptimizationThreshold: imageDownscalingOptimizationThreshold,
	}
}

// Page returns the page to be rendered.
func (p PageDrawerParameters) Page() *pdmodel.PDPage { return p.page }

// Renderer returns the renderer the drawer belongs to.
func (p PageDrawerParameters) Renderer() *PDFRenderer { return p.renderer }

// IsSubsamplingAllowed reports whether the renderer may subsample images.
func (p PageDrawerParameters) IsSubsamplingAllowed() bool { return p.subsamplingAllowed }

// Destination returns what the page is being rendered for.
func (p PageDrawerParameters) Destination() RenderDestination { return p.destination }

// RenderingHints returns the hints to draw with.
func (p PageDrawerParameters) RenderingHints() RenderingHints { return p.renderingHints }

// ImageDownscalingOptimizationThreshold returns the scale below which a slower,
// higher-quality downscale is used.
func (p PageDrawerParameters) ImageDownscalingOptimizationThreshold() float32 {
	return p.imageDownscalingOptimizationThreshold
}
