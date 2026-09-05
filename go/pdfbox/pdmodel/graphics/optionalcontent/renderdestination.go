package optionalcontent

// RenderDestination is where a page is being rendered to, which decides which
// optional content groups are visible.
//
// Port of the enum org.apache.pdfbox.rendering.RenderDestination.
//
// Java declares it in pdfbox/rendering, which imports this package for the
// optional content groups while this package imports it for getRenderState.
// Go forbids that cycle, so the enum is declared here, where the older of the
// two uses is, and pdfbox/rendering aliases it back to the Java name in the
// Java place -- the way pdmodel.ResourceCache aliases pdmodel/font's.
type RenderDestination int

const (
	// Export is rendering for export.
	Export RenderDestination = iota
	// View is rendering for the screen.
	View
	// Print is rendering for a printer.
	Print
)

// String returns the enum constant's name, which is Java's Enum.toString.
func (d RenderDestination) String() string {
	switch d {
	case Export:
		return "EXPORT"
	case View:
		return "VIEW"
	case Print:
		return "PRINT"
	}
	return "UNKNOWN"
}
