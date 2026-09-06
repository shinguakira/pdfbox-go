package fdf

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/w3c/dom"
)

// The /Subtype of each FDF annotation.
//
// Port of the SUBTYPE constant of each class.
const (
	SubTypeText           = "Text"
	SubTypeCaret          = "Caret"
	SubTypeFreeText       = "FreeText"
	SubTypeFileAttachment = "FileAttachment"
	SubTypeHighlight      = "Highlight"
	SubTypeInk            = "Ink"
	SubTypeLine           = "Line"
	SubTypeLink           = "Link"
	SubTypeCircle         = "Circle"
	SubTypeSquare         = "Square"
	SubTypePolygon        = "Polygon"
	SubTypePolyline       = "Polyline"
	SubTypeSound          = "Sound"
	SubTypeSquiggly       = "Squiggly"
	SubTypeStamp          = "Stamp"
	SubTypeStrikeOut      = "StrikeOut"
	SubTypeUnderline      = "Underline"
)

// interiorColorOfAttribute reads the interior-color attribute and sets it, which
// the circle, square, polygon, polyline and line annotations each write out.
func (a *FDFAnnotationBase) interiorColorOfAttribute(element *dom.Element) {
	color := element.GetAttribute("interior-color")
	if len(color) == 7 && color[0] == '#' {
		colorValue, err := strconv.ParseInt(color[1:7], 16, 64)
		if err != nil {
			panic(fmt.Sprintf("For input string: %q", color[1:7]))
		}
		c := awt.NewColorOfRGB(int(colorValue))
		a.setInteriorColor(&c)
	}
}

// setInteriorColor sets the /IC of the annotation, and removes it for a nil
// colour. Java repeats it in every class that has one.
func (a *FDFAnnotationBase) setInteriorColor(color *awt.Color) {
	var array *cos.Array
	if color != nil {
		r, g, b := color.RGBColorComponents()
		array = cos.ArrayOfFloats([]float32{r, g, b})
	}
	a.annot.SetItem(cos.IC, array)
}

// interiorColor returns the /IC of the annotation, or nil where it has none.
func (a *FDFAnnotationBase) interiorColor() *awt.Color { return a.ColorOf(cos.IC) }

// setFringe sets the /RD of the annotation. Java repeats it in every class that
// has one.
func (a *FDFAnnotationBase) setFringe(fringe *common.PDRectangle) {
	a.annot.SetItem(cos.RD, common.COSObjectOrNil(fringe))
}

// fringe returns the /RD of the annotation, or nil where it has none.
func (a *FDFAnnotationBase) fringe() *common.PDRectangle {
	rd := a.annot.GetCOSArray(cos.RD)
	if rd != nil {
		return common.NewPDRectangleOfCOSArray(rd)
	}
	return nil
}

// initFringe reads the fringe attribute and sets it. Java declares it private in
// every class that has one.
func (a *FDFAnnotationBase) initFringe(element *dom.Element) error {
	fringe := element.GetAttribute("fringe")
	if fringe != "" {
		rect, err := createRectangleFromAttributes(fringe,
			"Error: wrong amount of numbers in attribute 'fringe'")
		if err != nil {
			return err
		}
		a.setFringe(rect)
	}
	return nil
}

// setStartPointEndingStyle sets the first entry of the /LE of the annotation.
// Java repeats it in the line and polyline classes.
func (a *FDFAnnotationBase) setStartPointEndingStyle(style string) {
	actualStyle := style
	if actualStyle == "" {
		// Java takes null here, which the port writes as the empty string; the
		// callers only ever pass a non-empty style.
		actualStyle = annotation.LENone
	}
	array := a.annot.GetCOSArray(cos.LE)
	if array == nil {
		array = cos.NewArray()
		array.Add(cos.GetPDFName(actualStyle))
		array.Add(cos.GetPDFName(annotation.LENone))
		a.annot.SetItem(cos.LE, array)
	} else {
		array.SetName(0, actualStyle)
	}
}

// startPointEndingStyle returns the first entry of the /LE of the annotation.
func (a *FDFAnnotationBase) startPointEndingStyle() string {
	array := a.annot.GetCOSArray(cos.LE)
	if array != nil {
		return array.GetName(0, "")
	}
	return annotation.LENone
}

// setEndPointEndingStyle sets the second entry of the /LE of the annotation.
func (a *FDFAnnotationBase) setEndPointEndingStyle(style string) {
	actualStyle := style
	if actualStyle == "" {
		actualStyle = annotation.LENone
	}
	array := a.annot.GetCOSArray(cos.LE)
	if array == nil {
		array = cos.NewArray()
		array.Add(cos.GetPDFName(annotation.LENone))
		array.Add(cos.GetPDFName(actualStyle))
		a.annot.SetItem(cos.LE, array)
	} else {
		array.SetName(1, actualStyle)
	}
}

// endPointEndingStyle returns the second entry of the /LE of the annotation.
func (a *FDFAnnotationBase) endPointEndingStyle() string {
	array := a.annot.GetCOSArray(cos.LE)
	if array != nil {
		return array.GetName(1, "")
	}
	return annotation.LENone
}

// setVertices sets the /Vertices of the annotation. Java repeats it in the
// polygon and polyline classes.
func (a *FDFAnnotationBase) setVertices(vertices []float32) {
	a.annot.SetItem(cos.Vertices, cos.ArrayOfFloats(vertices))
}

// vertices returns the /Vertices of the annotation, or nil where it has none.
func (a *FDFAnnotationBase) vertices() []float32 {
	array := a.annot.GetCOSArray(cos.Vertices)
	if array != nil {
		return array.ToFloatArray()
	}
	return nil
}

// FDFAnnotationText is a text annotation of an FDF document.
//
// Port of FDFAnnotationText.
type FDFAnnotationText struct{ FDFAnnotationBase }

// NewFDFAnnotationText returns an empty text annotation.
func NewFDFAnnotationText() *FDFAnnotationText {
	a := &FDFAnnotationText{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeText)
	return a
}

// NewFDFAnnotationTextOf returns the text annotation the given dictionary holds.
func NewFDFAnnotationTextOf(dictionary *cos.Dictionary) *FDFAnnotationText {
	a := &FDFAnnotationText{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationTextOfXML returns the text annotation the given XFDF element
// describes.
func NewFDFAnnotationTextOfXML(element *dom.Element) (*FDFAnnotationText, error) {
	a := &FDFAnnotationText{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeText)
	icon := element.GetAttribute("icon")
	if icon != "" {
		a.SetIcon(icon)
	}
	state := element.GetAttribute("state")
	if state != "" {
		statemodel := element.GetAttribute("statemodel")
		if statemodel != "" {
			a.SetState(state)
			a.SetStateModel(statemodel)
		}
	}
	return a, nil
}

// SetIcon sets the /Name of the annotation, which names its icon.
func (a *FDFAnnotationText) SetIcon(icon string) { a.annot.SetName(cos.NameKey, icon) }

// Icon returns the /Name of the annotation, which is Note where it has none.
func (a *FDFAnnotationText) Icon() string {
	return a.annot.GetNameAsString(cos.NameKey, annotation.TextNameNote)
}

// State returns the /State of the annotation, or the empty string where it has
// none.
func (a *FDFAnnotationText) State() string { return a.annot.GetString(cos.State, "") }

// SetState sets the /State of the annotation.
func (a *FDFAnnotationText) SetState(state string) { a.annot.SetString(cos.State, state) }

// StateModel returns the /StateModel of the annotation, or the empty string
// where it has none.
func (a *FDFAnnotationText) StateModel() string { return a.annot.GetString(cos.StateModel, "") }

// SetStateModel sets the /StateModel of the annotation.
func (a *FDFAnnotationText) SetStateModel(stateModel string) {
	a.annot.SetString(cos.StateModel, stateModel)
}

// FDFAnnotationCaret is a caret annotation of an FDF document.
//
// Port of FDFAnnotationCaret.
type FDFAnnotationCaret struct{ FDFAnnotationBase }

// NewFDFAnnotationCaret returns an empty caret annotation.
func NewFDFAnnotationCaret() *FDFAnnotationCaret {
	a := &FDFAnnotationCaret{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeCaret)
	return a
}

// NewFDFAnnotationCaretOf returns the caret annotation the given dictionary
// holds.
func NewFDFAnnotationCaretOf(dictionary *cos.Dictionary) *FDFAnnotationCaret {
	a := &FDFAnnotationCaret{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationCaretOfXML returns the caret annotation the given XFDF element
// describes.
func NewFDFAnnotationCaretOfXML(element *dom.Element) (*FDFAnnotationCaret, error) {
	a := &FDFAnnotationCaret{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeCaret)
	if err := a.initFringe(element); err != nil {
		return nil, err
	}
	symbol := element.GetAttribute("symbol")
	if symbol != "" {
		a.SetSymbol(symbol)
	}
	return a, nil
}

// SetFringe sets the /RD of the annotation.
func (a *FDFAnnotationCaret) SetFringe(fringe *common.PDRectangle) { a.setFringe(fringe) }

// Fringe returns the /RD of the annotation, or nil where it has none.
func (a *FDFAnnotationCaret) Fringe() *common.PDRectangle { return a.fringe() }

// SetSymbol sets the /Sy of the annotation, which is P for a paragraph symbol
// and None for anything else.
func (a *FDFAnnotationCaret) SetSymbol(symbol string) {
	newSymbol := "None"
	if symbol == "paragraph" {
		newSymbol = "P"
	}
	a.annot.SetString(cos.Sy, newSymbol)
}

// Symbol returns the /Sy of the annotation, or the empty string where it has
// none.
func (a *FDFAnnotationCaret) Symbol() string { return a.annot.GetString(cos.Sy, "") }

// FDFAnnotationTextMarkup is the shared half of the four text markup
// annotations.
//
// Port of the abstract FDFAnnotationTextMarkup.
type FDFAnnotationTextMarkup struct{ FDFAnnotationBase }

// initTextMarkupOfXML is the protected FDFAnnotationTextMarkup(Element)
// constructor.
func (a *FDFAnnotationTextMarkup) initTextMarkupOfXML(element *dom.Element) error {
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return err
	}
	coords := element.GetAttribute("coords")
	if coords == "" {
		return errors.New("Error: missing attribute 'coords'")
	}
	coordsValues := strings.Split(coords, ",")
	if len(coordsValues) < 8 {
		return errors.New("Error: too little numbers in attribute 'coords'")
	}
	values := parseFloats(coordsValues)
	a.SetCoords(values)
	return nil
}

// SetCoords sets the /QuadPoints of the annotation.
func (a *FDFAnnotationTextMarkup) SetCoords(coords []float32) {
	a.annot.SetItem(cos.QuadPoints, cos.ArrayOfFloats(coords))
}

// Coords returns the /QuadPoints of the annotation, or nil where it has none --
// which should never happen, as this is a required item.
func (a *FDFAnnotationTextMarkup) Coords() []float32 {
	quadPoints := a.annot.GetCOSArray(cos.QuadPoints)
	if quadPoints != nil {
		return quadPoints.ToFloatArray()
	}
	return nil
}

// FDFAnnotationHighlight is a highlight annotation of an FDF document.
//
// Port of FDFAnnotationHighlight.
type FDFAnnotationHighlight struct{ FDFAnnotationTextMarkup }

// NewFDFAnnotationHighlight returns an empty highlight annotation.
func NewFDFAnnotationHighlight() *FDFAnnotationHighlight {
	a := &FDFAnnotationHighlight{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeHighlight)
	return a
}

// NewFDFAnnotationHighlightOf returns the highlight annotation the given
// dictionary holds.
func NewFDFAnnotationHighlightOf(dictionary *cos.Dictionary) *FDFAnnotationHighlight {
	a := &FDFAnnotationHighlight{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationHighlightOfXML returns the highlight annotation the given XFDF
// element describes.
func NewFDFAnnotationHighlightOfXML(element *dom.Element) (*FDFAnnotationHighlight, error) {
	a := &FDFAnnotationHighlight{}
	if err := a.initTextMarkupOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeHighlight)
	return a, nil
}

// FDFAnnotationSquiggly is a squiggly underline annotation of an FDF document.
//
// Port of FDFAnnotationSquiggly.
type FDFAnnotationSquiggly struct{ FDFAnnotationTextMarkup }

// NewFDFAnnotationSquiggly returns an empty squiggly annotation.
func NewFDFAnnotationSquiggly() *FDFAnnotationSquiggly {
	a := &FDFAnnotationSquiggly{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeSquiggly)
	return a
}

// NewFDFAnnotationSquigglyOf returns the squiggly annotation the given
// dictionary holds.
func NewFDFAnnotationSquigglyOf(dictionary *cos.Dictionary) *FDFAnnotationSquiggly {
	a := &FDFAnnotationSquiggly{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationSquigglyOfXML returns the squiggly annotation the given XFDF
// element describes.
func NewFDFAnnotationSquigglyOfXML(element *dom.Element) (*FDFAnnotationSquiggly, error) {
	a := &FDFAnnotationSquiggly{}
	if err := a.initTextMarkupOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeSquiggly)
	return a, nil
}

// FDFAnnotationStrikeOut is a strike out annotation of an FDF document.
//
// Port of FDFAnnotationStrikeOut.
type FDFAnnotationStrikeOut struct{ FDFAnnotationTextMarkup }

// NewFDFAnnotationStrikeOut returns an empty strike out annotation.
func NewFDFAnnotationStrikeOut() *FDFAnnotationStrikeOut {
	a := &FDFAnnotationStrikeOut{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeStrikeOut)
	return a
}

// NewFDFAnnotationStrikeOutOf returns the strike out annotation the given
// dictionary holds.
func NewFDFAnnotationStrikeOutOf(dictionary *cos.Dictionary) *FDFAnnotationStrikeOut {
	a := &FDFAnnotationStrikeOut{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationStrikeOutOfXML returns the strike out annotation the given
// XFDF element describes.
func NewFDFAnnotationStrikeOutOfXML(element *dom.Element) (*FDFAnnotationStrikeOut, error) {
	a := &FDFAnnotationStrikeOut{}
	if err := a.initTextMarkupOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeStrikeOut)
	return a, nil
}

// FDFAnnotationUnderline is an underline annotation of an FDF document.
//
// Port of FDFAnnotationUnderline.
type FDFAnnotationUnderline struct{ FDFAnnotationTextMarkup }

// NewFDFAnnotationUnderline returns an empty underline annotation.
func NewFDFAnnotationUnderline() *FDFAnnotationUnderline {
	a := &FDFAnnotationUnderline{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeUnderline)
	return a
}

// NewFDFAnnotationUnderlineOf returns the underline annotation the given
// dictionary holds.
func NewFDFAnnotationUnderlineOf(dictionary *cos.Dictionary) *FDFAnnotationUnderline {
	a := &FDFAnnotationUnderline{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationUnderlineOfXML returns the underline annotation the given XFDF
// element describes.
func NewFDFAnnotationUnderlineOfXML(element *dom.Element) (*FDFAnnotationUnderline, error) {
	a := &FDFAnnotationUnderline{}
	if err := a.initTextMarkupOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeUnderline)
	return a, nil
}

// FDFAnnotationSound is a sound annotation of an FDF document.
//
// Port of FDFAnnotationSound, which adds no accessor of its own.
type FDFAnnotationSound struct{ FDFAnnotationBase }

// NewFDFAnnotationSound returns an empty sound annotation.
func NewFDFAnnotationSound() *FDFAnnotationSound {
	a := &FDFAnnotationSound{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeSound)
	return a
}

// NewFDFAnnotationSoundOf returns the sound annotation the given dictionary
// holds.
func NewFDFAnnotationSoundOf(dictionary *cos.Dictionary) *FDFAnnotationSound {
	a := &FDFAnnotationSound{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationSoundOfXML returns the sound annotation the given XFDF element
// describes.
func NewFDFAnnotationSoundOfXML(element *dom.Element) (*FDFAnnotationSound, error) {
	a := &FDFAnnotationSound{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeSound)
	return a, nil
}

// FDFAnnotationFileAttachment is a file attachment annotation of an FDF
// document.
//
// Port of FDFAnnotationFileAttachment, which adds no accessor of its own.
type FDFAnnotationFileAttachment struct{ FDFAnnotationBase }

// NewFDFAnnotationFileAttachment returns an empty file attachment annotation.
func NewFDFAnnotationFileAttachment() *FDFAnnotationFileAttachment {
	a := &FDFAnnotationFileAttachment{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeFileAttachment)
	return a
}

// NewFDFAnnotationFileAttachmentOf returns the file attachment annotation the
// given dictionary holds.
func NewFDFAnnotationFileAttachmentOf(dictionary *cos.Dictionary) *FDFAnnotationFileAttachment {
	a := &FDFAnnotationFileAttachment{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationFileAttachmentOfXML returns the file attachment annotation the
// given XFDF element describes.
func NewFDFAnnotationFileAttachmentOfXML(
	element *dom.Element) (*FDFAnnotationFileAttachment, error) {
	a := &FDFAnnotationFileAttachment{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeFileAttachment)
	return a, nil
}

// FDFAnnotationCircle is a circle annotation of an FDF document.
//
// Port of FDFAnnotationCircle.
type FDFAnnotationCircle struct{ FDFAnnotationBase }

// NewFDFAnnotationCircle returns an empty circle annotation.
func NewFDFAnnotationCircle() *FDFAnnotationCircle {
	a := &FDFAnnotationCircle{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeCircle)
	return a
}

// NewFDFAnnotationCircleOf returns the circle annotation the given dictionary
// holds.
func NewFDFAnnotationCircleOf(dictionary *cos.Dictionary) *FDFAnnotationCircle {
	a := &FDFAnnotationCircle{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationCircleOfXML returns the circle annotation the given XFDF
// element describes.
func NewFDFAnnotationCircleOfXML(element *dom.Element) (*FDFAnnotationCircle, error) {
	a := &FDFAnnotationCircle{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeCircle)
	a.interiorColorOfAttribute(element)
	if err := a.initFringe(element); err != nil {
		return nil, err
	}
	return a, nil
}

// SetInteriorColor sets the /IC of the annotation.
func (a *FDFAnnotationCircle) SetInteriorColor(color *awt.Color) { a.setInteriorColor(color) }

// InteriorColor returns the /IC of the annotation, or nil where it has none.
func (a *FDFAnnotationCircle) InteriorColor() *awt.Color { return a.interiorColor() }

// SetFringe sets the /RD of the annotation.
func (a *FDFAnnotationCircle) SetFringe(fringe *common.PDRectangle) { a.setFringe(fringe) }

// Fringe returns the /RD of the annotation, or nil where it has none.
func (a *FDFAnnotationCircle) Fringe() *common.PDRectangle { return a.fringe() }

// FDFAnnotationSquare is a square annotation of an FDF document.
//
// Port of FDFAnnotationSquare.
type FDFAnnotationSquare struct{ FDFAnnotationBase }

// NewFDFAnnotationSquare returns an empty square annotation.
func NewFDFAnnotationSquare() *FDFAnnotationSquare {
	a := &FDFAnnotationSquare{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeSquare)
	return a
}

// NewFDFAnnotationSquareOf returns the square annotation the given dictionary
// holds.
func NewFDFAnnotationSquareOf(dictionary *cos.Dictionary) *FDFAnnotationSquare {
	a := &FDFAnnotationSquare{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationSquareOfXML returns the square annotation the given XFDF
// element describes.
func NewFDFAnnotationSquareOfXML(element *dom.Element) (*FDFAnnotationSquare, error) {
	a := &FDFAnnotationSquare{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeSquare)
	a.interiorColorOfAttribute(element)
	if err := a.initFringe(element); err != nil {
		return nil, err
	}
	return a, nil
}

// SetInteriorColor sets the /IC of the annotation.
func (a *FDFAnnotationSquare) SetInteriorColor(color *awt.Color) { a.setInteriorColor(color) }

// InteriorColor returns the /IC of the annotation, or nil where it has none.
func (a *FDFAnnotationSquare) InteriorColor() *awt.Color { return a.interiorColor() }

// SetFringe sets the /RD of the annotation.
func (a *FDFAnnotationSquare) SetFringe(fringe *common.PDRectangle) { a.setFringe(fringe) }

// Fringe returns the /RD of the annotation, or nil where it has none.
func (a *FDFAnnotationSquare) Fringe() *common.PDRectangle { return a.fringe() }

// FDFAnnotationInk is an ink annotation of an FDF document.
//
// Port of FDFAnnotationInk.
type FDFAnnotationInk struct{ FDFAnnotationBase }

// NewFDFAnnotationInk returns an empty ink annotation.
func NewFDFAnnotationInk() *FDFAnnotationInk {
	a := &FDFAnnotationInk{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeInk)
	return a
}

// NewFDFAnnotationInkOf returns the ink annotation the given dictionary holds.
func NewFDFAnnotationInkOf(dictionary *cos.Dictionary) *FDFAnnotationInk {
	a := &FDFAnnotationInk{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationInkOfXML returns the ink annotation the given XFDF element
// describes.
func NewFDFAnnotationInkOfXML(element *dom.Element) (*FDFAnnotationInk, error) {
	a := &FDFAnnotationInk{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeInk)
	gestures := dom.ElementsByPath(element, "inklist", "gesture")
	if len(gestures) == 0 {
		return nil, errors.New("Error: missing element 'gesture'")
	}
	inklist := [][]float32{}
	for _, node := range gestures {
		gesture := node.FirstChild().NodeValue()
		gestureValues := splitOnCommaOrSemicolon(gesture)
		values := parseFloats(gestureValues)
		inklist = append(inklist, values)
	}
	a.SetInkList(inklist)
	return a, nil
}

// SetInkList sets the /InkList of the annotation.
func (a *FDFAnnotationInk) SetInkList(inklist [][]float32) {
	newInklist := cos.NewArray()
	for _, array := range inklist {
		newInklist.Add(cos.ArrayOfFloats(array))
	}
	a.annot.SetItem(cos.InkList, newInklist)
}

// InkList returns the /InkList of the annotation, or nil where it has none --
// which should never happen, as this is a required item.
//
// Java casts every entry to COSArray without a check; the port panics where one
// is not, which is the same unchecked failure.
func (a *FDFAnnotationInk) InkList() [][]float32 {
	array := a.annot.GetCOSArray(cos.InkList)
	if array == nil {
		return nil
	}
	retval := make([][]float32, 0, array.Size())
	for _, entry := range array.ToList() {
		inner, isArray := entry.(*cos.Array)
		if !isArray {
			panic(fmt.Sprintf("fdf: %T cannot be cast to COSArray", entry))
		}
		retval = append(retval, inner.ToFloatArray())
	}
	return retval
}

// splitOnCommaOrSemicolon is String.split("[,;]").
func splitOnCommaOrSemicolon(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == ';' })
}

// FDFAnnotationLink is a link annotation of an FDF document.
//
// Port of FDFAnnotationLink, which adds no accessor of its own.
type FDFAnnotationLink struct{ FDFAnnotationBase }

// NewFDFAnnotationLink returns an empty link annotation.
//
// Java writes no super() call here, which Java supplies: the no-argument
// FDFAnnotation() runs first and builds the dictionary.
func NewFDFAnnotationLink() *FDFAnnotationLink {
	a := &FDFAnnotationLink{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeLink)
	return a
}

// NewFDFAnnotationLinkOf returns the link annotation the given dictionary holds.
func NewFDFAnnotationLinkOf(dictionary *cos.Dictionary) *FDFAnnotationLink {
	a := &FDFAnnotationLink{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationLinkOfXML returns the link annotation the given XFDF element
// describes.
func NewFDFAnnotationLinkOfXML(element *dom.Element) (*FDFAnnotationLink, error) {
	a := &FDFAnnotationLink{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeLink)
	uri := dom.ElementsByPath(element, "OnActivation", "Action", "URI")
	if len(uri) > 0 {
		namedItem := uri[0].Attributes().GetNamedItem("Name")
		if namedItem != nil && namedItem.NodeValue() != "" {
			actionURI := action.NewPDActionURI()
			actionURI.SetURI(namedItem.NodeValue())
			a.annot.SetItem(cos.A, actionURI.COSObject())
		}
	}
	// GoTo is more tricky, because because page destination needs page tree
	// to convert number into PDPage object
	return a, nil
}

// FDFAnnotationPolygon is a polygon annotation of an FDF document.
//
// Port of FDFAnnotationPolygon.
type FDFAnnotationPolygon struct{ FDFAnnotationBase }

// NewFDFAnnotationPolygon returns an empty polygon annotation.
func NewFDFAnnotationPolygon() *FDFAnnotationPolygon {
	a := &FDFAnnotationPolygon{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypePolygon)
	return a
}

// NewFDFAnnotationPolygonOf returns the polygon annotation the given dictionary
// holds.
func NewFDFAnnotationPolygonOf(dictionary *cos.Dictionary) *FDFAnnotationPolygon {
	a := &FDFAnnotationPolygon{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationPolygonOfXML returns the polygon annotation the given XFDF
// element describes.
func NewFDFAnnotationPolygonOfXML(element *dom.Element) (*FDFAnnotationPolygon, error) {
	a := &FDFAnnotationPolygon{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypePolygon)
	// Java evaluates the XPath "vertices" for its string value, which is the
	// text of the first vertices child and the empty string where there is none.
	vertices := ""
	if element := dom.FirstElementByTagName(element, "vertices"); element != nil {
		vertices = dom.TextContent(element)
	}
	if vertices == "" {
		return nil, errors.New("Error: missing element 'vertices'")
	}
	verticesValues := splitOnCommaOrSemicolon(vertices)
	a.SetVertices(parseFloats(verticesValues))
	a.interiorColorOfAttribute(element)
	return a, nil
}

// SetVertices sets the /Vertices of the annotation.
func (a *FDFAnnotationPolygon) SetVertices(vertices []float32) { a.setVertices(vertices) }

// Vertices returns the /Vertices of the annotation, or nil where it has none.
func (a *FDFAnnotationPolygon) Vertices() []float32 { return a.vertices() }

// SetInteriorColor sets the /IC of the annotation.
func (a *FDFAnnotationPolygon) SetInteriorColor(color *awt.Color) { a.setInteriorColor(color) }

// InteriorColor returns the /IC of the annotation, or nil where it has none.
func (a *FDFAnnotationPolygon) InteriorColor() *awt.Color { return a.interiorColor() }

// FDFAnnotationPolyline is a polyline annotation of an FDF document.
//
// Port of FDFAnnotationPolyline.
type FDFAnnotationPolyline struct{ FDFAnnotationBase }

// NewFDFAnnotationPolyline returns an empty polyline annotation.
func NewFDFAnnotationPolyline() *FDFAnnotationPolyline {
	a := &FDFAnnotationPolyline{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypePolyline)
	return a
}

// NewFDFAnnotationPolylineOf returns the polyline annotation the given
// dictionary holds.
func NewFDFAnnotationPolylineOf(dictionary *cos.Dictionary) *FDFAnnotationPolyline {
	a := &FDFAnnotationPolyline{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationPolylineOfXML returns the polyline annotation the given XFDF
// element describes.
func NewFDFAnnotationPolylineOfXML(element *dom.Element) (*FDFAnnotationPolyline, error) {
	a := &FDFAnnotationPolyline{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypePolyline)
	if err := a.initVertices(element); err != nil {
		return nil, err
	}
	a.initStyles(element)
	return a, nil
}

// initVertices reads the vertices element and sets it. Java declares it private.
func (a *FDFAnnotationPolyline) initVertices(element *dom.Element) error {
	vertices := ""
	if first := dom.FirstElementByTagName(element, "vertices"); first != nil {
		vertices = dom.TextContent(first)
	}
	if vertices == "" {
		return errors.New("Error: missing element 'vertices'")
	}
	verticesValues := splitOnCommaOrSemicolon(vertices)
	a.SetVertices(parseFloats(verticesValues))
	return nil
}

// initStyles reads the head, tail and interior-color attributes and sets them.
// Java declares it private.
func (a *FDFAnnotationPolyline) initStyles(element *dom.Element) {
	startStyle := element.GetAttribute("head")
	if startStyle != "" {
		a.SetStartPointEndingStyle(startStyle)
	}
	endStyle := element.GetAttribute("tail")
	if endStyle != "" {
		a.SetEndPointEndingStyle(endStyle)
	}
	a.interiorColorOfAttribute(element)
}

// SetVertices sets the /Vertices of the annotation.
func (a *FDFAnnotationPolyline) SetVertices(vertices []float32) { a.setVertices(vertices) }

// Vertices returns the /Vertices of the annotation, or nil where it has none.
func (a *FDFAnnotationPolyline) Vertices() []float32 { return a.vertices() }

// SetStartPointEndingStyle sets the first entry of the /LE of the annotation.
func (a *FDFAnnotationPolyline) SetStartPointEndingStyle(style string) {
	a.setStartPointEndingStyle(style)
}

// StartPointEndingStyle returns the first entry of the /LE of the annotation.
func (a *FDFAnnotationPolyline) StartPointEndingStyle() string {
	return a.startPointEndingStyle()
}

// SetEndPointEndingStyle sets the second entry of the /LE of the annotation.
func (a *FDFAnnotationPolyline) SetEndPointEndingStyle(style string) {
	a.setEndPointEndingStyle(style)
}

// EndPointEndingStyle returns the second entry of the /LE of the annotation.
func (a *FDFAnnotationPolyline) EndPointEndingStyle() string { return a.endPointEndingStyle() }

// SetInteriorColor sets the /IC of the annotation.
func (a *FDFAnnotationPolyline) SetInteriorColor(color *awt.Color) { a.setInteriorColor(color) }

// InteriorColor returns the /IC of the annotation, or nil where it has none.
func (a *FDFAnnotationPolyline) InteriorColor() *awt.Color { return a.interiorColor() }
