package rendering

// The operators a graphics engine runs.
//
// Java's PDFGraphicsStreamEngine constructor names all sixty in one run. The
// port's cannot: every processor holds the engine, so the operator packages
// import contentstream and it cannot import them back. The concrete engine
// registers them instead, the way text.NewLegacyPDFStreamEngine already does,
// and this is that list.

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	colorpr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/color"
	graphicspr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/graphics"
	markedcontentpr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/markedcontent"
	statepr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/state"
	textpr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/text"
)

// addAllOperators registers every operator PDFGraphicsStreamEngine's
// constructor registers.
//
// Java names the three marked content sequence operators and not the two
// marked content points, which is what AddSequenceOperators is for.
func addAllOperators(engine *contentstream.PDFGraphicsStreamEngine) {
	statepr.AddAll(engine.PDFStreamEngine)
	textpr.AddAll(engine.PDFStreamEngine)
	markedcontentpr.AddSequenceOperators(engine.PDFStreamEngine)
	colorpr.AddAll(engine.PDFStreamEngine)
	graphicspr.AddAll(engine)
}
