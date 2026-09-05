package rendering

// The kinds of image a page can be rendered into.
//
// Port of the enum org.apache.pdfbox.rendering.ImageType.

// ImageType is the type of a rendered image.
//
// Port of ImageType. Java's toBufferedImageType answers a BufferedImage
// constant; the port has no BufferedImage, and a Backend maps these to whatever
// its own surfaces are, so the method is not here. See migration/STATUS.md.
type ImageType int

const (
	// Binary is black or white.
	Binary ImageType = iota
	// Gray is shades of gray.
	Gray
	// RGB is red, green, blue.
	RGB
	// ARGB is alpha, red, green, blue.
	ARGB
	// BGR is blue, green, red.
	BGR
)

// String returns the enum constant's name, which is Java's Enum.toString.
func (t ImageType) String() string {
	switch t {
	case Binary:
		return "BINARY"
	case Gray:
		return "GRAY"
	case RGB:
		return "RGB"
	case ARGB:
		return "ARGB"
	case BGR:
		return "BGR"
	}
	return "UNKNOWN"
}

// HasAlpha reports whether this type carries an alpha channel, which decides
// whether a page is cleared to transparent or to white.
//
// Java asks image.getType() == TYPE_INT_ARGB instead, having a BufferedImage to
// ask; the port asks the type it was given.
func (t ImageType) HasAlpha() bool { return t == ARGB }
