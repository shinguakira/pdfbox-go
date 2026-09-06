package annotation

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
)

// The line intents and line ending styles of PDAnnotationLine, which
// PDAnnotationPolyline and PDAnnotationFreeText also use.
const (
	// ITLineArrow is PDAnnotationLine.IT_LINE_ARROW.
	ITLineArrow = "LineArrow"
	// ITLineDimension is IT_LINE_DIMENSION.
	ITLineDimension = "LineDimension"
	// LESquare is LE_SQUARE.
	LESquare = "Square"
	// LECircle is LE_CIRCLE.
	LECircle = "Circle"
	// LEDiamond is LE_DIAMOND.
	LEDiamond = "Diamond"
	// LEOpenArrow is LE_OPEN_ARROW.
	LEOpenArrow = "OpenArrow"
	// LEClosedArrow is LE_CLOSED_ARROW.
	LEClosedArrow = "ClosedArrow"
	// LENone is LE_NONE, and the default.
	LENone = "None"
	// LEButt is LE_BUTT.
	LEButt = "Butt"
	// LEROpenArrow is LE_R_OPEN_ARROW.
	LEROpenArrow = "ROpenArrow"
	// LERClosedArrow is LE_R_CLOSED_ARROW.
	LERClosedArrow = "RClosedArrow"
	// LESlash is LE_SLASH.
	LESlash = "Slash"
)

// setStartPointEndingStyle is the body PDAnnotationLine and PDAnnotationPolyline
// both declare. The empty string is Java's null, which becomes LE_NONE.
func setStartPointEndingStyle(dict *cos.Dictionary, style string) {
	actualStyle := style
	if actualStyle == "" {
		actualStyle = LENone
	}
	array := dict.GetCOSArray(cos.LE)
	if array == nil || array.IsEmpty() {
		array = cos.NewArray()
		array.Add(cos.GetPDFName(actualStyle))
		array.Add(cos.GetPDFName(LENone))
		dict.SetItem(cos.LE, array)
	} else {
		array.SetName(0, actualStyle)
	}
}

func startPointEndingStyle(dict *cos.Dictionary) string {
	if array := dict.GetCOSArray(cos.LE); array != nil && array.Size() >= 2 {
		return array.GetName(0, LENone)
	}
	return LENone
}

func setEndPointEndingStyle(dict *cos.Dictionary, style string) {
	actualStyle := style
	if actualStyle == "" {
		actualStyle = LENone
	}
	array := dict.GetCOSArray(cos.LE)
	if array == nil || array.Size() < 2 {
		array = cos.NewArray()
		array.Add(cos.GetPDFName(LENone))
		array.Add(cos.GetPDFName(actualStyle))
		dict.SetItem(cos.LE, array)
	} else {
		array.SetName(1, actualStyle)
	}
}

func endPointEndingStyle(dict *cos.Dictionary) string {
	if array := dict.GetCOSArray(cos.LE); array != nil && array.Size() >= 2 {
		return array.GetName(1, LENone)
	}
	return LENone
}

// PDAnnotationLine is a straight line, with optional endings and a caption.
type PDAnnotationLine struct {
	PDAnnotationMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationLine)(nil)

// NewPDAnnotationLine creates a new line annotation.
func NewPDAnnotationLine() *PDAnnotationLine {
	a := &PDAnnotationLine{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeLine)
	// Dictionary value L is mandatory, fill in with arbitrary value
	a.SetLine([]float32{0, 0, 0, 0})
	return a
}

// NewPDAnnotationLineOf creates one over the given dictionary.
func NewPDAnnotationLineOf(field *cos.Dictionary) *PDAnnotationLine {
	a := &PDAnnotationLine{}
	a.InitAnnotationOf(field)
	return a
}

// SetLine sets the /L endpoints.
func (a *PDAnnotationLine) SetLine(l []float32) {
	a.AnnotationDictionary().SetItem(cos.L, cos.ArrayOfFloats(l))
}

// Line returns the /L endpoints, or nil.
func (a *PDAnnotationLine) Line() []float32 {
	if l := a.AnnotationDictionary().GetCOSArray(cos.L); l != nil {
		return l.ToFloatArray()
	}
	return nil
}

// SetStartPointEndingStyle sets the first /LE entry.
func (a *PDAnnotationLine) SetStartPointEndingStyle(style string) {
	setStartPointEndingStyle(a.AnnotationDictionary(), style)
}

// StartPointEndingStyle returns the first /LE entry, which defaults to None.
func (a *PDAnnotationLine) StartPointEndingStyle() string {
	return startPointEndingStyle(a.AnnotationDictionary())
}

// SetEndPointEndingStyle sets the second /LE entry.
func (a *PDAnnotationLine) SetEndPointEndingStyle(style string) {
	setEndPointEndingStyle(a.AnnotationDictionary(), style)
}

// EndPointEndingStyle returns the second /LE entry, which defaults to None.
func (a *PDAnnotationLine) EndPointEndingStyle() string {
	return endPointEndingStyle(a.AnnotationDictionary())
}

// SetInteriorColor sets the /IC interior colour.
func (a *PDAnnotationLine) SetInteriorColor(ic *color.PDColor) {
	a.AnnotationDictionary().SetItem(cos.IC, ic.ToCOSArray())
}

// InteriorColor returns the /IC interior colour, or nil.
func (a *PDAnnotationLine) InteriorColor() *color.PDColor { return a.ColorOf(cos.IC) }

// SetCaption sets the /Cap flag.
func (a *PDAnnotationLine) SetCaption(cap bool) {
	a.AnnotationDictionary().SetBoolean(cos.Cap, cap)
}

// HasCaption reports the /Cap flag.
func (a *PDAnnotationLine) HasCaption() bool {
	return a.AnnotationDictionary().GetBoolean(cos.Cap, false)
}

// LeaderLineLength returns the /LL entry.
func (a *PDAnnotationLine) LeaderLineLength() float32 {
	return a.AnnotationDictionary().GetFloat(cos.LL, 0)
}

// SetLeaderLineLength sets the /LL entry.
func (a *PDAnnotationLine) SetLeaderLineLength(leaderLineLength float32) {
	a.AnnotationDictionary().SetFloat(cos.LL, leaderLineLength)
}

// LeaderLineExtensionLength returns the /LLE entry.
func (a *PDAnnotationLine) LeaderLineExtensionLength() float32 {
	return a.AnnotationDictionary().GetFloat(cos.LLE, 0)
}

// SetLeaderLineExtensionLength sets the /LLE entry.
func (a *PDAnnotationLine) SetLeaderLineExtensionLength(v float32) {
	a.AnnotationDictionary().SetFloat(cos.LLE, v)
}

// LeaderLineOffsetLength returns the /LLO entry.
func (a *PDAnnotationLine) LeaderLineOffsetLength() float32 {
	return a.AnnotationDictionary().GetFloat(cos.LLO, 0)
}

// SetLeaderLineOffsetLength sets the /LLO entry.
func (a *PDAnnotationLine) SetLeaderLineOffsetLength(v float32) {
	a.AnnotationDictionary().SetFloat(cos.LLO, v)
}

// CaptionPositioning returns the /CP entry.
func (a *PDAnnotationLine) CaptionPositioning() string {
	return a.AnnotationDictionary().GetNameAsString(cos.CP, "")
}

// SetCaptionPositioning sets the /CP entry.
func (a *PDAnnotationLine) SetCaptionPositioning(captionPositioning string) {
	a.AnnotationDictionary().SetName(cos.CP, captionPositioning)
}

// SetCaptionHorizontalOffset sets the first /CO entry.
func (a *PDAnnotationLine) SetCaptionHorizontalOffset(offset float32) {
	array := a.AnnotationDictionary().GetCOSArray(cos.CO)
	if array == nil {
		a.AnnotationDictionary().SetItem(cos.CO, cos.ArrayOfFloats([]float32{offset, 0}))
	} else {
		array.Set(0, cos.NewFloat(offset))
	}
}

// CaptionHorizontalOffset returns the first /CO entry, or 0.
func (a *PDAnnotationLine) CaptionHorizontalOffset() float32 {
	if array := a.AnnotationDictionary().GetCOSArray(cos.CO); array != nil {
		return array.ToFloatArray()[0]
	}
	return 0
}

// SetCaptionVerticalOffset sets the second /CO entry.
func (a *PDAnnotationLine) SetCaptionVerticalOffset(offset float32) {
	array := a.AnnotationDictionary().GetCOSArray(cos.CO)
	if array == nil {
		a.AnnotationDictionary().SetItem(cos.CO, cos.ArrayOfFloats([]float32{0, offset}))
	} else {
		array.Set(1, cos.NewFloat(offset))
	}
}

// CaptionVerticalOffset returns the second /CO entry, or 0.
func (a *PDAnnotationLine) CaptionVerticalOffset() float32 {
	if array := a.AnnotationDictionary().GetCOSArray(cos.CO); array != nil {
		return array.ToFloatArray()[1]
	}
	return 0
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationLine) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationLine) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationPolygon is a closed polygon.
type PDAnnotationPolygon struct {
	PDAnnotationMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationPolygon)(nil)

// NewPDAnnotationPolygon creates a new polygon annotation.
func NewPDAnnotationPolygon() *PDAnnotationPolygon {
	a := &PDAnnotationPolygon{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypePolygon)
	return a
}

// NewPDAnnotationPolygonOf creates one over the given dictionary.
func NewPDAnnotationPolygonOf(dict *cos.Dictionary) *PDAnnotationPolygon {
	a := &PDAnnotationPolygon{}
	a.InitAnnotationOf(dict)
	return a
}

// SetInteriorColor sets the /IC colour.
//
// PDF 32000 specification has "the interior color with which to fill the
// annotation's line endings" but it is the inside of the polygon.
func (a *PDAnnotationPolygon) SetInteriorColor(ic *color.PDColor) {
	a.AnnotationDictionary().SetItem(cos.IC, ic.ToCOSArray())
}

// InteriorColor returns the /IC colour, or nil.
func (a *PDAnnotationPolygon) InteriorColor() *color.PDColor { return a.ColorOf(cos.IC) }

// SetBorderEffect sets the /BE border effect.
func (a *PDAnnotationPolygon) SetBorderEffect(be *PDBorderEffectDictionary) {
	setAnnotationItem(a.AnnotationDictionary(), cos.BE, be)
}

// BorderEffect returns the /BE border effect, or nil.
func (a *PDAnnotationPolygon) BorderEffect() *PDBorderEffectDictionary {
	if be := a.AnnotationDictionary().GetCOSDictionary(cos.BE); be != nil {
		return NewPDBorderEffectDictionaryOf(be)
	}
	return nil
}

// Vertices returns the /Vertices, or nil.
func (a *PDAnnotationPolygon) Vertices() []float32 {
	if array := a.AnnotationDictionary().GetCOSArray(cos.Vertices); array != nil {
		return array.ToFloatArray()
	}
	return nil
}

// SetVertices sets the /Vertices.
func (a *PDAnnotationPolygon) SetVertices(points []float32) {
	a.AnnotationDictionary().SetItem(cos.Vertices, cos.ArrayOfFloats(points))
}

// Path returns the /Path, or nil.
func (a *PDAnnotationPolygon) Path() [][]float32 {
	return arrayOfFloatArrays(a.AnnotationDictionary().GetCOSArray(cos.Path), nil)
}

// arrayOfFloatArrays reads an array of arrays of numbers, using the given empty
// value for an entry that is not an array.
func arrayOfFloatArrays(array *cos.Array, absent [][]float32) [][]float32 {
	if array == nil {
		return absent
	}
	out := make([][]float32, array.Size())
	for i := 0; i < array.Size(); i++ {
		if inner, ok := array.GetObject(i).(*cos.Array); ok {
			out[i] = inner.ToFloatArray()
		} else {
			out[i] = []float32{}
		}
	}
	return out
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationPolygon) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationPolygon) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationPolyline is an open polyline.
type PDAnnotationPolyline struct {
	PDAnnotationMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationPolyline)(nil)

// NewPDAnnotationPolyline creates a new polyline annotation.
func NewPDAnnotationPolyline() *PDAnnotationPolyline {
	a := &PDAnnotationPolyline{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypePolyLine)
	return a
}

// NewPDAnnotationPolylineOf creates one over the given dictionary.
func NewPDAnnotationPolylineOf(dict *cos.Dictionary) *PDAnnotationPolyline {
	a := &PDAnnotationPolyline{}
	a.InitAnnotationOf(dict)
	return a
}

// SetStartPointEndingStyle sets the first /LE entry.
func (a *PDAnnotationPolyline) SetStartPointEndingStyle(style string) {
	setStartPointEndingStyle(a.AnnotationDictionary(), style)
}

// StartPointEndingStyle returns the first /LE entry, which defaults to None.
func (a *PDAnnotationPolyline) StartPointEndingStyle() string {
	return startPointEndingStyle(a.AnnotationDictionary())
}

// SetEndPointEndingStyle sets the second /LE entry.
func (a *PDAnnotationPolyline) SetEndPointEndingStyle(style string) {
	setEndPointEndingStyle(a.AnnotationDictionary(), style)
}

// EndPointEndingStyle returns the second /LE entry, which defaults to None.
func (a *PDAnnotationPolyline) EndPointEndingStyle() string {
	return endPointEndingStyle(a.AnnotationDictionary())
}

// SetInteriorColor sets the /IC interior colour.
func (a *PDAnnotationPolyline) SetInteriorColor(ic *color.PDColor) {
	a.AnnotationDictionary().SetItem(cos.IC, ic.ToCOSArray())
}

// InteriorColor returns the /IC interior colour, or nil.
func (a *PDAnnotationPolyline) InteriorColor() *color.PDColor { return a.ColorOf(cos.IC) }

// Vertices returns the /Vertices, or nil.
func (a *PDAnnotationPolyline) Vertices() []float32 {
	if vertices := a.AnnotationDictionary().GetCOSArray(cos.Vertices); vertices != nil {
		return vertices.ToFloatArray()
	}
	return nil
}

// SetVertices sets the /Vertices.
func (a *PDAnnotationPolyline) SetVertices(points []float32) {
	a.AnnotationDictionary().SetItem(cos.Vertices, cos.ArrayOfFloats(points))
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationPolyline) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationPolyline) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationInk is a freehand scribble.
type PDAnnotationInk struct {
	PDAnnotationMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationInk)(nil)

// NewPDAnnotationInk creates a new ink annotation.
func NewPDAnnotationInk() *PDAnnotationInk {
	a := &PDAnnotationInk{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeInk)
	return a
}

// NewPDAnnotationInkOf creates one over the given dictionary.
func NewPDAnnotationInkOf(dict *cos.Dictionary) *PDAnnotationInk {
	a := &PDAnnotationInk{}
	a.InitAnnotationOf(dict)
	return a
}

// SetInkList sets the /InkList paths; nil removes the entry.
func (a *PDAnnotationInk) SetInkList(inkList [][]float32) {
	if inkList == nil {
		a.AnnotationDictionary().RemoveItem(cos.InkList)
		return
	}
	array := cos.NewArray()
	for _, path := range inkList {
		array.Add(cos.ArrayOfFloats(path))
	}
	a.AnnotationDictionary().SetItem(cos.InkList, array)
}

// InkList returns the /InkList paths, or an empty slice.
func (a *PDAnnotationInk) InkList() [][]float32 {
	return arrayOfFloatArrays(a.AnnotationDictionary().GetCOSArray(cos.InkList), [][]float32{})
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationInk) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationInk) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// The free text intents.
const (
	// ITFreeText is PDAnnotationFreeText.IT_FREE_TEXT.
	ITFreeText = "FreeText"
	// ITFreeTextCallout is IT_FREE_TEXT_CALLOUT.
	ITFreeTextCallout = "FreeTextCallout"
	// ITFreeTextTypeWriter is IT_FREE_TEXT_TYPE_WRITER.
	ITFreeTextTypeWriter = "FreeTextTypeWriter"
)

// PDAnnotationFreeText is text drawn straight onto the page.
type PDAnnotationFreeText struct {
	PDAnnotationMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationFreeText)(nil)

// NewPDAnnotationFreeText creates a new free text annotation.
func NewPDAnnotationFreeText() *PDAnnotationFreeText {
	a := &PDAnnotationFreeText{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeFreeText)
	return a
}

// NewPDAnnotationFreeTextOf creates one over the given dictionary.
func NewPDAnnotationFreeTextOf(field *cos.Dictionary) *PDAnnotationFreeText {
	a := &PDAnnotationFreeText{}
	a.InitAnnotationOf(field)
	return a
}

// DefaultAppearance returns the /DA string.
func (a *PDAnnotationFreeText) DefaultAppearance() string {
	return a.AnnotationDictionary().GetString(cos.DA, "")
}

// SetDefaultAppearance sets the /DA string.
func (a *PDAnnotationFreeText) SetDefaultAppearance(daValue string) {
	a.AnnotationDictionary().SetString(cos.DA, daValue)
}

// DefaultStyleString returns the /DS string.
func (a *PDAnnotationFreeText) DefaultStyleString() string {
	return a.AnnotationDictionary().GetString(cos.DS, "")
}

// SetDefaultStyleString sets the /DS string.
func (a *PDAnnotationFreeText) SetDefaultStyleString(defaultStyleString string) {
	a.AnnotationDictionary().SetString(cos.DS, defaultStyleString)
}

// Q returns the /Q quadding, which defaults to 0.
func (a *PDAnnotationFreeText) Q() int {
	return a.AnnotationDictionary().GetIntDefault(cos.Q, 0)
}

// SetQ sets the /Q quadding.
func (a *PDAnnotationFreeText) SetQ(q int) { a.AnnotationDictionary().SetInt(cos.Q, q) }

// SetRectDifferences sets the /RD margins, all four the same.
func (a *PDAnnotationFreeText) SetRectDifferences(difference float32) {
	a.SetRectDifferencesOf(difference, difference, difference, difference)
}

// SetRectDifferencesOf sets the four /RD margins.
func (a *PDAnnotationFreeText) SetRectDifferencesOf(left, top, right, bottom float32) {
	setRectDifferences(a.AnnotationDictionary(), left, top, right, bottom)
}

// RectDifferences returns the four /RD margins, or an empty slice.
func (a *PDAnnotationFreeText) RectDifferences() []float32 {
	return rectDifferences(a.AnnotationDictionary())
}

// SetRectDifference sets the /RD rectangle difference, which gives the gap
// between the rectangle of the annotation and where the drawing happens.
func (a *PDAnnotationFreeText) SetRectDifference(rd *common.PDRectangle) {
	setAnnotationItem(a.AnnotationDictionary(), cos.RD, rd)
}

// RectDifference returns the /RD rectangle difference, or nil.
func (a *PDAnnotationFreeText) RectDifference() *common.PDRectangle {
	if rectDifference := a.AnnotationDictionary().GetCOSArray(cos.RD); rectDifference != nil {
		return common.NewPDRectangleOfCOSArray(rectDifference)
	}
	return nil
}

// SetCallout sets the /CL callout line. Java declares it final.
func (a *PDAnnotationFreeText) SetCallout(callout []float32) {
	a.AnnotationDictionary().SetItem(cos.CL, cos.ArrayOfFloats(callout))
}

// Callout returns the /CL callout line, or nil.
func (a *PDAnnotationFreeText) Callout() []float32 {
	if callout := a.AnnotationDictionary().GetCOSArray(cos.CL); callout != nil {
		return callout.ToFloatArray()
	}
	return nil
}

// SetLineEndingStyle sets the /LE entry. Java declares it final.
//
// Unlike the line and polyline annotations, whose /LE is an array of two, a
// free text annotation's is a single name.
func (a *PDAnnotationFreeText) SetLineEndingStyle(style string) {
	a.AnnotationDictionary().SetName(cos.LE, style)
}

// LineEndingStyle returns the /LE entry, which defaults to None.
func (a *PDAnnotationFreeText) LineEndingStyle() string {
	return a.AnnotationDictionary().GetNameAsString(cos.LE, LENone)
}

// SetBorderEffect sets the /BE border effect.
func (a *PDAnnotationFreeText) SetBorderEffect(be *PDBorderEffectDictionary) {
	setAnnotationItem(a.AnnotationDictionary(), cos.BE, be)
}

// BorderEffect returns the /BE border effect, or nil.
func (a *PDAnnotationFreeText) BorderEffect() *PDBorderEffectDictionary {
	if effectDict := a.AnnotationDictionary().GetCOSDictionary(cos.BE); effectDict != nil {
		return NewPDBorderEffectDictionaryOf(effectDict)
	}
	return nil
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationFreeText) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationFreeText) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationWidget is the appearance of a form field on a page.
type PDAnnotationWidget struct {
	PDAnnotationBase
}

var _ PDAnnotation = (*PDAnnotationWidget)(nil)

// NewPDAnnotationWidget creates a new widget annotation.
func NewPDAnnotationWidget() *PDAnnotationWidget {
	a := &PDAnnotationWidget{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeWidget)
	return a
}

// NewPDAnnotationWidgetOf creates one over the given dictionary.
//
// Java writes the /Subtype here too, unlike every other annotation, so a
// dictionary read back as a widget is stamped as one.
func NewPDAnnotationWidgetOf(field *cos.Dictionary) *PDAnnotationWidget {
	a := &PDAnnotationWidget{}
	a.InitAnnotationOf(field)
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeWidget)
	return a
}

// HighlightingMode returns the /H mode, which defaults to "I".
func (a *PDAnnotationWidget) HighlightingMode() string {
	return a.AnnotationDictionary().GetNameAsString(cos.H, "I")
}

// SetHighlightingMode sets the /H mode, which must be one of N, I, O, P or T.
//
// Java throws IllegalArgumentException otherwise, which is unchecked, so the
// port panics. The empty string is Java's null, which it accepts.
func (a *PDAnnotationWidget) SetHighlightingMode(highlightingMode string) {
	switch highlightingMode {
	case "", "N", "I", "O", "P", "T":
		a.AnnotationDictionary().SetName(cos.H, highlightingMode)
	default:
		panic("Valid values for highlighting mode are 'N', 'N', 'O', 'P' or 'T'")
	}
}

// AppearanceCharacteristics returns the /MK dictionary, or nil.
func (a *PDAnnotationWidget) AppearanceCharacteristics() *PDAppearanceCharacteristicsDictionary {
	if mk := a.AnnotationDictionary().GetCOSDictionary(cos.MK); mk != nil {
		return NewPDAppearanceCharacteristicsDictionaryOf(mk)
	}
	return nil
}

// SetAppearanceCharacteristics sets the /MK dictionary.
func (a *PDAnnotationWidget) SetAppearanceCharacteristics(
	appearanceCharacteristics *PDAppearanceCharacteristicsDictionary) {
	setAnnotationItem(a.AnnotationDictionary(), cos.MK, appearanceCharacteristics)
}

// Action returns the /A action, or nil.
func (a *PDAnnotationWidget) Action() action.Action {
	if act := a.AnnotationDictionary().GetCOSDictionary(cos.A); act != nil {
		return action.CreateAction(act)
	}
	return nil
}

// SetAction sets the /A action.
func (a *PDAnnotationWidget) SetAction(act action.Action) {
	setAnnotationItem(a.AnnotationDictionary(), cos.A, act)
}

// Actions returns the /AA additional actions, or nil.
func (a *PDAnnotationWidget) Actions() *action.PDAnnotationAdditionalActions {
	if actions := a.AnnotationDictionary().GetCOSDictionary(cos.AA); actions != nil {
		return action.NewPDAnnotationAdditionalActionsOf(actions)
	}
	return nil
}

// SetActions sets the /AA additional actions.
func (a *PDAnnotationWidget) SetActions(actions *action.PDAnnotationAdditionalActions) {
	setAnnotationItem(a.AnnotationDictionary(), cos.AA, actions)
}

// SetBorderStyle sets the /BS border style.
func (a *PDAnnotationWidget) SetBorderStyle(bs *PDBorderStyleDictionary) {
	setAnnotationItem(a.AnnotationDictionary(), cos.BS, bs)
}

// BorderStyle returns the /BS border style, or nil.
func (a *PDAnnotationWidget) BorderStyle() *PDBorderStyleDictionary {
	if bs := a.AnnotationDictionary().GetCOSDictionary(cos.BS); bs != nil {
		return NewPDBorderStyleDictionaryOf(bs)
	}
	return nil
}

// PDAppearanceCharacteristicsDictionary is the /MK entry of a widget.
//
// Port of PDAppearanceCharacteristicsDictionary.
type PDAppearanceCharacteristicsDictionary struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDAppearanceCharacteristicsDictionary)(nil)

// NewPDAppearanceCharacteristicsDictionaryOf creates one over the given
// dictionary.
func NewPDAppearanceCharacteristicsDictionaryOf(dict *cos.Dictionary) *PDAppearanceCharacteristicsDictionary {
	return &PDAppearanceCharacteristicsDictionary{dictionary: dict}
}

// COSObject returns the dictionary.
func (d *PDAppearanceCharacteristicsDictionary) COSObject() cos.Base { return d.dictionary }

// Rotation returns the /R rotation, which defaults to 0.
func (d *PDAppearanceCharacteristicsDictionary) Rotation() int {
	return d.dictionary.GetIntDefault(cos.R, 0)
}

// SetRotation sets the /R rotation.
func (d *PDAppearanceCharacteristicsDictionary) SetRotation(rotation int) {
	d.dictionary.SetInt(cos.R, rotation)
}

// BorderColour returns the /BC colour, or nil.
func (d *PDAppearanceCharacteristicsDictionary) BorderColour() *color.PDColor {
	return d.colorOf(cos.BC)
}

// SetBorderColour sets the /BC colour.
func (d *PDAppearanceCharacteristicsDictionary) SetBorderColour(c *color.PDColor) {
	d.dictionary.SetItem(cos.BC, c.ToCOSArray())
}

// Background returns the /BG colour, or nil.
func (d *PDAppearanceCharacteristicsDictionary) Background() *color.PDColor {
	return d.colorOf(cos.BG)
}

// SetBackground sets the /BG colour.
func (d *PDAppearanceCharacteristicsDictionary) SetBackground(c *color.PDColor) {
	d.dictionary.SetItem(cos.BG, c.ToCOSArray())
}

// NormalCaption returns the /CA caption.
func (d *PDAppearanceCharacteristicsDictionary) NormalCaption() string {
	return d.dictionary.GetString(cos.CA, "")
}

// SetNormalCaption sets the /CA caption.
func (d *PDAppearanceCharacteristicsDictionary) SetNormalCaption(caption string) {
	d.dictionary.SetString(cos.CA, caption)
}

// RolloverCaption returns the /RC caption.
func (d *PDAppearanceCharacteristicsDictionary) RolloverCaption() string {
	return d.dictionary.GetString(cos.RC, "")
}

// SetRolloverCaption sets the /RC caption.
func (d *PDAppearanceCharacteristicsDictionary) SetRolloverCaption(caption string) {
	d.dictionary.SetString(cos.RC, caption)
}

// AlternateCaption returns the /AC caption.
func (d *PDAppearanceCharacteristicsDictionary) AlternateCaption() string {
	return d.dictionary.GetString(cos.AC, "")
}

// SetAlternateCaption sets the /AC caption.
func (d *PDAppearanceCharacteristicsDictionary) SetAlternateCaption(caption string) {
	d.dictionary.SetString(cos.AC, caption)
}

// NormalIcon returns the /I icon, or nil.
func (d *PDAppearanceCharacteristicsDictionary) NormalIcon() *form.PDFormXObject {
	return d.iconOf(cos.I)
}

// RolloverIcon returns the /RI icon, or nil.
func (d *PDAppearanceCharacteristicsDictionary) RolloverIcon() *form.PDFormXObject {
	return d.iconOf(cos.RI)
}

// AlternateIcon returns the /IX icon, or nil.
func (d *PDAppearanceCharacteristicsDictionary) AlternateIcon() *form.PDFormXObject {
	return d.iconOf(cos.IX)
}

func (d *PDAppearanceCharacteristicsDictionary) iconOf(key *cos.Name) *form.PDFormXObject {
	if stream, ok := d.dictionary.GetDictionaryObject(key).(*cos.Stream); ok {
		return form.NewPDFormXObjectOfStream(stream)
	}
	return nil
}

// colorOf is the private getColor.
//
// Unlike PDAnnotation.getColor it does not fold a three component grey into a
// one component one, and it returns nil for any other size.
func (d *PDAppearanceCharacteristicsDictionary) colorOf(itemName *cos.Name) *color.PDColor {
	cs := d.dictionary.GetCOSArray(itemName)
	if cs == nil {
		return nil
	}
	var colorSpace color.PDColorSpace
	switch cs.Size() {
	case 1:
		colorSpace = color.DeviceGray
	case 3:
		colorSpace = color.DeviceRGB
	case 4:
		colorSpace = color.DeviceCMYK
	default:
		return nil
	}
	return color.NewPDColorOfCOSArray(cs, colorSpace)
}
