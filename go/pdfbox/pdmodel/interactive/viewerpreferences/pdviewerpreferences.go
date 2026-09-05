// Package viewerpreferences holds the preferences a viewer applies when it
// opens a document.
//
// Port of org.apache.pdfbox.pdmodel.interactive.viewerpreferences.
package viewerpreferences

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The page mode a viewer uses when it leaves full screen.
//
// Port of the nested enum NON_FULL_SCREEN_PAGE_MODE. Java writes the constant's
// own name into the file, so the port is a string type whose values are those
// names.
type NonFullScreenPageMode string

const (
	// NonFullScreenPageModeUseNone shows neither outlines nor thumbnails.
	NonFullScreenPageModeUseNone NonFullScreenPageMode = "UseNone"
	// NonFullScreenPageModeUseOutlines shows the outline.
	NonFullScreenPageModeUseOutlines NonFullScreenPageMode = "UseOutlines"
	// NonFullScreenPageModeUseThumbs shows the thumbnails.
	NonFullScreenPageModeUseThumbs NonFullScreenPageMode = "UseThumbs"
	// NonFullScreenPageModeUseOC shows the optional content panel.
	NonFullScreenPageModeUseOC NonFullScreenPageMode = "UseOC"
)

// ReadingDirection is the order the pages are read in.
//
// Port of the nested enum READING_DIRECTION.
type ReadingDirection string

const (
	// ReadingDirectionL2R is left to right.
	ReadingDirectionL2R ReadingDirection = "L2R"
	// ReadingDirectionR2L is right to left.
	ReadingDirectionR2L ReadingDirection = "R2L"
)

// Boundary is one of a page's boxes.
//
// Port of the nested enum BOUNDARY.
type Boundary string

const (
	// BoundaryMediaBox is the media box.
	BoundaryMediaBox Boundary = "MediaBox"
	// BoundaryCropBox is the crop box.
	BoundaryCropBox Boundary = "CropBox"
	// BoundaryBleedBox is the bleed box.
	BoundaryBleedBox Boundary = "BleedBox"
	// BoundaryTrimBox is the trim box.
	BoundaryTrimBox Boundary = "TrimBox"
	// BoundaryArtBox is the art box.
	BoundaryArtBox Boundary = "ArtBox"
)

// Duplex is how the document is to be printed on both sides.
//
// Port of the nested enum DUPLEX.
type Duplex string

const (
	// DuplexSimplex prints on one side.
	DuplexSimplex Duplex = "Simplex"
	// DuplexFlipShortEdge prints on both, flipping on the short edge.
	DuplexFlipShortEdge Duplex = "DuplexFlipShortEdge"
	// DuplexFlipLongEdge prints on both, flipping on the long edge.
	DuplexFlipLongEdge Duplex = "DuplexFlipLongEdge"
)

// PrintScaling is how the page is scaled to the paper.
//
// Port of the nested enum PRINT_SCALING.
type PrintScaling string

const (
	// PrintScalingNone does not scale.
	PrintScalingNone PrintScaling = "None"
	// PrintScalingAppDefault leaves it to the application.
	PrintScalingAppDefault PrintScaling = "AppDefault"
)

// PDViewerPreferences is the viewer preferences dictionary of a document.
//
// Port of
// org.apache.pdfbox.pdmodel.interactive.viewerpreferences.PDViewerPreferences.
type PDViewerPreferences struct {
	prefs *cos.Dictionary
}

// NewPDViewerPreferences creates an empty preferences dictionary.
func NewPDViewerPreferences() *PDViewerPreferences {
	return &PDViewerPreferences{prefs: cos.NewDictionary()}
}

// NewPDViewerPreferencesOf creates preferences over the given dictionary.
func NewPDViewerPreferencesOf(dic *cos.Dictionary) *PDViewerPreferences {
	return &PDViewerPreferences{prefs: dic}
}

// COSObject returns the dictionary.
func (p *PDViewerPreferences) COSObject() cos.Base { return p.prefs }

// Dictionary returns the dictionary, typed.
func (p *PDViewerPreferences) Dictionary() *cos.Dictionary { return p.prefs }

// HideToolbar reports whether the viewer hides its toolbar.
func (p *PDViewerPreferences) HideToolbar() bool {
	return p.prefs.GetBoolean(cos.HideToolbar, false)
}

// SetHideToolbar sets whether the viewer hides its toolbar.
func (p *PDViewerPreferences) SetHideToolbar(value bool) {
	p.prefs.SetBoolean(cos.HideToolbar, value)
}

// HideMenubar reports whether the viewer hides its menu bar.
func (p *PDViewerPreferences) HideMenubar() bool {
	return p.prefs.GetBoolean(cos.HideMenubar, false)
}

// SetHideMenubar sets whether the viewer hides its menu bar.
func (p *PDViewerPreferences) SetHideMenubar(value bool) {
	p.prefs.SetBoolean(cos.HideMenubar, value)
}

// HideWindowUI reports whether the viewer hides its window controls.
func (p *PDViewerPreferences) HideWindowUI() bool {
	return p.prefs.GetBoolean(cos.HideWindowUI, false)
}

// SetHideWindowUI sets whether the viewer hides its window controls.
func (p *PDViewerPreferences) SetHideWindowUI(value bool) {
	p.prefs.SetBoolean(cos.HideWindowUI, value)
}

// FitWindow reports whether the window is resized to the first page.
func (p *PDViewerPreferences) FitWindow() bool {
	return p.prefs.GetBoolean(cos.FitWindow, false)
}

// SetFitWindow sets whether the window is resized to the first page.
func (p *PDViewerPreferences) SetFitWindow(value bool) {
	p.prefs.SetBoolean(cos.FitWindow, value)
}

// CenterWindow reports whether the window is centred on the screen.
func (p *PDViewerPreferences) CenterWindow() bool {
	return p.prefs.GetBoolean(cos.CenterWindow, false)
}

// SetCenterWindow sets whether the window is centred on the screen.
func (p *PDViewerPreferences) SetCenterWindow(value bool) {
	p.prefs.SetBoolean(cos.CenterWindow, value)
}

// DisplayDocTitle reports whether the title bar shows the document title
// rather than the file name.
func (p *PDViewerPreferences) DisplayDocTitle() bool {
	return p.prefs.GetBoolean(cos.DisplayDocTitle, false)
}

// SetDisplayDocTitle sets whether the title bar shows the document title.
func (p *PDViewerPreferences) SetDisplayDocTitle(value bool) {
	p.prefs.SetBoolean(cos.DisplayDocTitle, value)
}

// NonFullScreenPageMode returns the page mode used outside full screen, which
// defaults to UseNone.
func (p *PDViewerPreferences) NonFullScreenPageMode() string {
	return p.prefs.GetNameAsString(cos.NonFullScreenPageMode,
		string(NonFullScreenPageModeUseNone))
}

// SetNonFullScreenPageMode sets the page mode used outside full screen.
func (p *PDViewerPreferences) SetNonFullScreenPageMode(value NonFullScreenPageMode) {
	p.prefs.SetName(cos.NonFullScreenPageMode, string(value))
}

// ReadingDirection returns the reading direction, which defaults to L2R.
func (p *PDViewerPreferences) ReadingDirection() string {
	return p.prefs.GetNameAsString(cos.Direction, string(ReadingDirectionL2R))
}

// SetReadingDirection sets the reading direction.
func (p *PDViewerPreferences) SetReadingDirection(value ReadingDirection) {
	p.prefs.SetName(cos.Direction, string(value))
}

// ViewArea returns the box a viewer displays, which defaults to CropBox.
func (p *PDViewerPreferences) ViewArea() string {
	return p.prefs.GetNameAsString(cos.ViewArea, string(BoundaryCropBox))
}

// SetViewArea sets the box a viewer displays.
func (p *PDViewerPreferences) SetViewArea(value Boundary) {
	p.prefs.SetName(cos.ViewArea, string(value))
}

// ViewClip returns the box a viewer clips to, which defaults to CropBox.
func (p *PDViewerPreferences) ViewClip() string {
	return p.prefs.GetNameAsString(cos.ViewClip, string(BoundaryCropBox))
}

// SetViewClip sets the box a viewer clips to.
func (p *PDViewerPreferences) SetViewClip(value Boundary) {
	p.prefs.SetName(cos.ViewClip, string(value))
}

// PrintArea returns the box that is printed, which defaults to CropBox.
func (p *PDViewerPreferences) PrintArea() string {
	return p.prefs.GetNameAsString(cos.PrintArea, string(BoundaryCropBox))
}

// SetPrintArea sets the box that is printed.
func (p *PDViewerPreferences) SetPrintArea(value Boundary) {
	p.prefs.SetName(cos.PrintArea, string(value))
}

// PrintClip returns the box printing clips to, which defaults to CropBox.
func (p *PDViewerPreferences) PrintClip() string {
	return p.prefs.GetNameAsString(cos.PrintClip, string(BoundaryCropBox))
}

// SetPrintClip sets the box printing clips to.
func (p *PDViewerPreferences) SetPrintClip(value Boundary) {
	p.prefs.SetName(cos.PrintClip, string(value))
}

// Duplex returns the duplex mode, or "" where the document does not say.
//
// Java's getNameAsString(COSName) has no default, so it returns null here where
// every other getter above has one.
func (p *PDViewerPreferences) Duplex() string {
	return p.prefs.GetNameAsString(cos.Duplex, "")
}

// SetDuplex sets the duplex mode.
func (p *PDViewerPreferences) SetDuplex(value Duplex) {
	p.prefs.SetName(cos.Duplex, string(value))
}

// PrintScaling returns the print scaling, which defaults to AppDefault.
func (p *PDViewerPreferences) PrintScaling() string {
	return p.prefs.GetNameAsString(cos.PrintScaling, string(PrintScalingAppDefault))
}

// SetPrintScaling sets the print scaling.
func (p *PDViewerPreferences) SetPrintScaling(value PrintScaling) {
	p.prefs.SetName(cos.PrintScaling, string(value))
}
