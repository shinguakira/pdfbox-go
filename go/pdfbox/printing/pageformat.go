package printing

// The two java.awt.print value types PDFPageable answers with.
//
// Port of what java.awt.print.Paper and java.awt.print.PageFormat carry, which
// is all PDFPageable.getPageFormat sets and all a caller of it reads. Neither
// class is ported whole: the rest of them is what the Java print system does
// with the values, and there is no print system here.

// PageOrientation is a PageFormat's orientation.
//
// Port of the three PageFormat constants, whose Java values are these.
type PageOrientation int

const (
	// LandscapePage is PageFormat.LANDSCAPE.
	LandscapePage PageOrientation = 0
	// PortraitPage is PageFormat.PORTRAIT.
	PortraitPage PageOrientation = 1
	// ReverseLandscapePage is PageFormat.REVERSE_LANDSCAPE.
	ReverseLandscapePage PageOrientation = 2
)

// Paper is a sheet of paper and the area of it that can be printed on, in
// 1/72nds of an inch.
//
// Port of the state of java.awt.print.Paper.
type Paper struct {
	Width, Height float64

	ImageableX, ImageableY          float64
	ImageableWidth, ImageableHeight float64
}

// SetSize sets the width and height of the sheet.
func (p *Paper) SetSize(width, height float64) {
	p.Width = width
	p.Height = height
}

// SetImageableArea sets the area of the sheet that can be printed on.
func (p *Paper) SetImageableArea(x, y, width, height float64) {
	p.ImageableX = x
	p.ImageableY = y
	p.ImageableWidth = width
	p.ImageableHeight = height
}

// PageFormat is a paper and the orientation a page is printed on it in.
//
// Port of the state of java.awt.print.PageFormat.
type PageFormat struct {
	Paper       Paper
	Orientation PageOrientation
}

// NewPageFormat returns a page format on default letter paper in portrait,
// which is what java.awt.print.PageFormat's constructor gives.
func NewPageFormat() PageFormat {
	format := PageFormat{Orientation: PortraitPage}
	format.Paper.SetSize(612, 792)
	format.Paper.SetImageableArea(72, 72, 612-144, 792-144)
	return format
}

// ImageableX returns the x of the printable area, in the page's own
// orientation.
//
// Java swaps the paper's imageable coordinates according to the orientation;
// the port does the same, so that PDFPrintable reads the same numbers Java's
// print method does.
func (f PageFormat) ImageableX() float64 {
	switch f.Orientation {
	case LandscapePage:
		return f.Paper.Height - (f.Paper.ImageableY + f.Paper.ImageableHeight)
	case ReverseLandscapePage:
		return f.Paper.ImageableY
	}
	return f.Paper.ImageableX
}

// ImageableY returns the y of the printable area, in the page's own
// orientation.
func (f PageFormat) ImageableY() float64 {
	switch f.Orientation {
	case LandscapePage:
		return f.Paper.ImageableX
	case ReverseLandscapePage:
		return f.Paper.Width - (f.Paper.ImageableX + f.Paper.ImageableWidth)
	}
	return f.Paper.ImageableY
}

// ImageableWidth returns the width of the printable area, in the page's own
// orientation.
func (f PageFormat) ImageableWidth() float64 {
	if f.Orientation == PortraitPage {
		return f.Paper.ImageableWidth
	}
	return f.Paper.ImageableHeight
}

// ImageableHeight returns the height of the printable area, in the page's own
// orientation.
func (f PageFormat) ImageableHeight() float64 {
	if f.Orientation == PortraitPage {
		return f.Paper.ImageableHeight
	}
	return f.Paper.ImageableWidth
}
