// Package filetypedetector names an image format from the bytes it starts
// with.
//
// Port of org.apache.pdfbox.util.filetypedetector, which PDFBox took from
// metadata-extractor.
package filetypedetector

// FileType is a file format the detector knows.
//
// Port of org.apache.pdfbox.util.filetypedetector.FileType, an enum; the port
// makes it an int with a String method, so that a value prints its own name the
// way Java's enum does.
type FileType int

// The formats, in the order the Java enum declares them.
const (
	Unknown FileType = iota
	JPEG
	TIFF
	PSD
	PNG
	BMP
	GIF
	ICO
	PCX
	RIFF

	ARW
	CRW
	CR2
	NEF
	ORF
	RAF
	RW2
)

var fileTypeNames = [...]string{
	"UNKNOWN", "JPEG", "TIFF", "PSD", "PNG", "BMP", "GIF", "ICO", "PCX", "RIFF",
	"ARW", "CRW", "CR2", "NEF", "ORF", "RAF", "RW2",
}

// String returns the name of the constant, which is what Java's Enum.toString
// returns.
func (f FileType) String() string {
	if f < 0 || int(f) >= len(fileTypeNames) {
		return "UNKNOWN"
	}
	return fileTypeNames[f]
}
