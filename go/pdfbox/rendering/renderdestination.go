// Package rendering draws a PDF page.
//
// Port of org.apache.pdfbox.rendering, which is slice 9's.
//
// Java draws through java.awt.Graphics2D. Go has nothing equivalent, and slice
// 9's B0 decision is to port everything that computes and to put the last
// drawing step behind Backend. So PageDrawer decides what to draw and says so,
// and PDFRenderer's renderImage answers ErrNoBackend where no Backend is
// installed, rather than a blank page.
//
// **rendering/raster is one.** `track/raster` wrote it, over
// github.com/srwiley/rasterx and github.com/golang/freetype/raster, and
// `raster.RenderPage` puts the two halves back together for a caller that only
// wants an image. It is a substitution rather than a transliteration, and what
// it draws differently from Graphics2D is measured and pinned; see
// migration/STATUS.md.
package rendering

import "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"

// RenderDestination is where a page is being rendered to, which decides which
// optional content groups are visible.
//
// Port of the enum org.apache.pdfbox.rendering.RenderDestination, which is
// declared in pdmodel/graphics/optionalcontent: that package needs it for
// getRenderState and this one imports that package, which in Go the enum cannot
// straddle. The alias puts the Java name in the Java place.
type RenderDestination = optionalcontent.RenderDestination

// The three destinations, re-exported so that a caller of this package need not
// name the package the enum had to move to.
const (
	// Export is rendering for export.
	Export = optionalcontent.Export
	// View is rendering for the screen.
	View = optionalcontent.View
	// Print is rendering for a printer.
	Print = optionalcontent.Print
)
