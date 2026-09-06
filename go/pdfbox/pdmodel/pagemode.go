package pdmodel

import "fmt"

// PageMode is a name object specifying how the document shall be displayed when
// opened.
//
// Port of PageMode. Java writes the constant's string value into the file, so
// the port is a string type whose values are those strings.
type PageMode string

const (
	// PageModeUseNone displays neither the outline nor the thumbnails.
	PageModeUseNone PageMode = "UseNone"
	// PageModeUseOutlines shows bookmarks when the pdf is opened.
	PageModeUseOutlines PageMode = "UseOutlines"
	// PageModeUseThumbs shows thumbnails when the pdf is opened.
	PageModeUseThumbs PageMode = "UseThumbs"
	// PageModeFullScreen is full screen mode with no menu bar, window controls.
	PageModeFullScreen PageMode = "FullScreen"
	// PageModeUseOptionalContent makes the optional content group panel visible
	// when opened.
	PageModeUseOptionalContent PageMode = "UseOC"
	// PageModeUseAttachments makes the attachments panel visible.
	PageModeUseAttachments PageMode = "UseAttachments"
)

// PageModeValues returns the constants in declaration order, which is what
// Java's PageMode.values() answers.
func PageModeValues() []PageMode {
	return []PageMode{
		PageModeUseNone,
		PageModeUseOutlines,
		PageModeUseThumbs,
		PageModeFullScreen,
		PageModeUseOptionalContent,
		PageModeUseAttachments,
	}
}

// PageModeFromString returns the mode with the given string value.
//
// Java throws IllegalArgumentException for a value that is not one of the
// constants. The value comes out of a PDF rather than from the library, so the
// port answers an error; see conventions/java-to-go.md. PDDocumentCatalog.PageMode
// checks it exactly where Java catches.
func PageModeFromString(value string) (PageMode, error) {
	for _, instance := range PageModeValues() {
		if string(instance) == value {
			return instance, nil
		}
	}
	return "", fmt.Errorf("pdmodel: %s", value)
}

// StringValue returns the string value, as used in a PDF file.
func (m PageMode) StringValue() string { return string(m) }

// PageLayout is a name object specifying the page layout that shall be used
// when the document is opened.
//
// Port of PageLayout.
type PageLayout string

const (
	// PageLayoutSinglePage displays one page at a time.
	PageLayoutSinglePage PageLayout = "SinglePage"
	// PageLayoutOneColumn displays the pages in one column.
	PageLayoutOneColumn PageLayout = "OneColumn"
	// PageLayoutTwoColumnLeft displays the pages in two columns, with odd
	// numbered pages on the left.
	PageLayoutTwoColumnLeft PageLayout = "TwoColumnLeft"
	// PageLayoutTwoColumnRight displays the pages in two columns, with odd
	// numbered pages on the right. See also
	// viewerpreferences.PDViewerPreferences.SetReadingDirection if dealing with
	// an RTL language.
	PageLayoutTwoColumnRight PageLayout = "TwoColumnRight"
	// PageLayoutTwoPageLeft displays the pages two at a time, with odd-numbered
	// pages on the left.
	PageLayoutTwoPageLeft PageLayout = "TwoPageLeft"
	// PageLayoutTwoPageRight displays the pages two at a time, with odd-numbered
	// pages on the right. See also
	// viewerpreferences.PDViewerPreferences.SetReadingDirection if dealing with
	// an RTL language.
	PageLayoutTwoPageRight PageLayout = "TwoPageRight"
)

// PageLayoutValues returns the constants in declaration order, which is what
// Java's PageLayout.values() answers.
func PageLayoutValues() []PageLayout {
	return []PageLayout{
		PageLayoutSinglePage,
		PageLayoutOneColumn,
		PageLayoutTwoColumnLeft,
		PageLayoutTwoColumnRight,
		PageLayoutTwoPageLeft,
		PageLayoutTwoPageRight,
	}
}

// PageLayoutFromString returns the layout with the given string value; see
// PageModeFromString for the error.
func PageLayoutFromString(value string) (PageLayout, error) {
	for _, instance := range PageLayoutValues() {
		if string(instance) == value {
			return instance, nil
		}
	}
	return "", fmt.Errorf("pdmodel: %s", value)
}

// StringValue returns the string value, as used in a PDF file.
func (l PageLayout) StringValue() string { return string(l) }
