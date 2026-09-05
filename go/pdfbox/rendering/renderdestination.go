// Package rendering draws a PDF page.
//
// Port of org.apache.pdfbox.rendering, which is slice 9's. Only
// RenderDestination is here: PDOptionalContentGroup.getRenderState takes one,
// and that is slice 8's. See migration/STATUS.md.
package rendering

// RenderDestination is where a page is being rendered to, which decides which
// optional content groups are visible.
//
// Port of the enum org.apache.pdfbox.rendering.RenderDestination.
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
