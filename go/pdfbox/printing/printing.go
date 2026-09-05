// Package printing prints a PDF document.
//
// Port of org.apache.pdfbox.printing.
//
// Java prints through java.awt.print: a Printable draws onto a Graphics2D the
// print system hands it, and a Pageable answers a PageFormat per page. Go has
// no print system and, under slice 9's B0 decision, no rasteriser either. What
// is ported is everything those two classes compute -- the rotated crop and
// media boxes, the portrait-normalised paper, the scale-to-fit and centering
// arithmetic -- with rendering.Backend standing in for the Graphics2D and the
// two java.awt.print value types written out here. See migration/STATUS.md.
package printing

// Orientation is how a page is laid on the paper.
//
// Port of the enum org.apache.pdfbox.printing.Orientation.
type Orientation int

const (
	// Auto selects the orientation of each page based on its aspect ratio.
	Auto Orientation = iota
	// Landscape prints all pages as landscape.
	Landscape
	// ReverseLandscape prints all pages as reverse landscape, which is
	// Landscape rotated 180 degrees.
	ReverseLandscape
	// Portrait prints all pages as portrait.
	Portrait
)

// String returns the enum constant's name, which is Java's Enum.toString.
func (o Orientation) String() string {
	switch o {
	case Auto:
		return "AUTO"
	case Landscape:
		return "LANDSCAPE"
	case ReverseLandscape:
		return "REVERSE_LANDSCAPE"
	case Portrait:
		return "PORTRAIT"
	}
	return "UNKNOWN"
}

// Scaling is how a page is fitted to the paper.
//
// Port of the enum org.apache.pdfbox.printing.Scaling.
type Scaling int

const (
	// ActualSize prints the image at 100% scale.
	ActualSize Scaling = iota
	// ShrinkToFit shrinks the image to fit the page, if needed.
	ShrinkToFit
	// StretchToFit stretches the image to fill the page, if needed.
	StretchToFit
	// ScaleToFit stretches or shrinks the image to fill the page, as needed.
	ScaleToFit
)

// String returns the enum constant's name, which is Java's Enum.toString.
func (s Scaling) String() string {
	switch s {
	case ActualSize:
		return "ACTUAL_SIZE"
	case ShrinkToFit:
		return "SHRINK_TO_FIT"
	case StretchToFit:
		return "STRETCH_TO_FIT"
	case ScaleToFit:
		return "SCALE_TO_FIT"
	}
	return "UNKNOWN"
}
