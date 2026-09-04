package font

// FontProvider is the external font service provider interface.
//
// Port of the abstract class org.apache.pdfbox.pdmodel.font.FontProvider, which
// has nothing but two abstract methods.
type FontProvider interface {
	// ToDebugString returns a string containing debugging information. This
	// will be written to the log if no suitable fonts are found and no fallback
	// fonts are available. May be empty.
	ToDebugString() string

	// GetFontInfo returns a list of information about fonts on the system.
	GetFontInfo() []FontInfo
}
