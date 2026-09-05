// Package annotation holds the annotations a page carries: links, notes,
// highlights, form widgets.
//
// Port of org.apache.pdfbox.pdmodel.interactive.annotation.
package annotation

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// The annotation flags, bit by bit.
//
// Port of the ten private FLAG_ constants of PDAnnotation.
const (
	flagInvisible      = 1 << 0
	flagHidden         = 1 << 1
	flagPrinted        = 1 << 2
	flagNoZoom         = 1 << 3
	flagNoRotate       = 1 << 4
	flagNoView         = 1 << 5
	flagReadOnly       = 1 << 6
	flagLocked         = 1 << 7
	flagToggleNoView   = 1 << 8
	flagLockedContents = 1 << 9
)

// PageLike is the page an annotation sits on.
//
// Java names PDPage and PDDocument, which live in pdmodel; pdmodel imports this
// package for PDPage.getAnnotations, so the dependency cannot run both ways.
// The port names what is used and takes the constructor below, which pdmodel
// sets from its init.
type PageLike interface {
	common.COSObjectable
}

// NewPageFromDictionary builds a page from its dictionary. pdmodel sets it.
var NewPageFromDictionary func(dic *cos.Dictionary) PageLike

// PDAnnotation is what every annotation is.
//
// Java's PDAnnotation is an abstract class; the port splits it into this
// interface for the contract and the struct below for the state.
type PDAnnotation interface {
	common.COSObjectable

	// AnnotationDictionary returns the dictionary, which getCOSObject narrows
	// to in Java.
	AnnotationDictionary() *cos.Dictionary

	// Subtype returns the /Subtype of the annotation.
	Subtype() string

	// ConstructAppearancesInDocument builds the appearance stream of this
	// annotation, with the document its streams belong to.
	ConstructAppearancesInDocument(document common.COSDocumentLike) error

	// ConstructAppearances builds the appearance stream of this annotation
	// where it has an appearance handler. The base does nothing.
	ConstructAppearances() error

	// Rectangle returns the /Rect of the annotation.
	Rectangle() *common.PDRectangle

	// SetRectangle sets the /Rect of the annotation.
	SetRectangle(rectangle *common.PDRectangle)

	// Color returns the /C colour of the annotation.
	Color() *color.PDColor

	// Border returns the /Border of the annotation.
	Border() *cos.Array

	// Appearance returns the /AP appearance dictionary, or nil.
	Appearance() *PDAppearanceDictionary

	// SetAppearance sets the /AP appearance dictionary.
	SetAppearance(appearance *PDAppearanceDictionary)

	// NormalAppearanceStream returns the normal appearance stream, or nil.
	NormalAppearanceStream() *PDAppearanceStream

	// IsInvisible reports the invisible flag.
	IsInvisible() bool

	// IsHidden reports the hidden flag.
	IsHidden() bool

	// IsPrinted reports the print flag.
	IsPrinted() bool

	// IsNoView reports the no-view flag.
	IsNoView() bool

	// IsNoRotate reports the no-rotate flag.
	IsNoRotate() bool

	// OptionalContent returns the /OC optional content of this annotation, or
	// nil.
	OptionalContent() markedcontent.PropertyList
}

// annotationFactories maps a /Subtype to the constructor that builds it.
//
// Java's PDAnnotation.createAnnotation is a switch naming all nineteen
// subclasses, which in Go would be a cycle only if they lived elsewhere; they
// do not, so this is a plain switch in CreateAnnotation below and this registry
// is unused. It is kept out.

// PDAnnotationBase carries the state and the concrete methods every annotation
// shares.
//
// Port of the non-abstract half of PDAnnotation.
type PDAnnotationBase struct {
	dictionary *cos.Dictionary
}

// InitAnnotation is the protected PDAnnotation() constructor.
func (a *PDAnnotationBase) InitAnnotation() {
	a.dictionary = cos.NewDictionary()
	a.dictionary.SetItem(cos.Type, cos.Annot)
}

// InitAnnotationOf is the protected PDAnnotation(COSDictionary) constructor.
func (a *PDAnnotationBase) InitAnnotationOf(dict *cos.Dictionary) {
	a.dictionary = dict
	annotationType := dict.GetDictionaryObject(cos.Type)
	if annotationType == nil {
		a.dictionary.SetItem(cos.Type, cos.Annot)
	} else if annotationType != cos.Base(cos.Annot) {
		slog.Warn("annotation: Annotation has an unexpected type, further mayhem may follow",
			"type", annotationType)
	}
}

// COSObject returns the dictionary.
func (a *PDAnnotationBase) COSObject() cos.Base { return a.dictionary }

// AnnotationDictionary returns the dictionary, typed.
func (a *PDAnnotationBase) AnnotationDictionary() *cos.Dictionary { return a.dictionary }

// Equals reports whether the other annotation is over an equal dictionary,
// which is what Java's equals compares.
func (a *PDAnnotationBase) Equals(o any) bool {
	if other, ok := o.(PDAnnotation); ok {
		return other.AnnotationDictionary() == a.dictionary
	}
	return false
}

// SetSubtype sets the /Subtype. Java declares it protected and final.
func (a *PDAnnotationBase) SetSubtype(subType string) {
	a.dictionary.SetName(cos.Subtype, subType)
}

// Subtype returns the /Subtype.
func (a *PDAnnotationBase) Subtype() string {
	return a.dictionary.GetNameAsString(cos.Subtype, "")
}

// Rectangle returns the /Rect of this annotation, or nil where it is not four
// numbers.
func (a *PDAnnotationBase) Rectangle() *common.PDRectangle {
	rectArray := a.dictionary.GetCOSArray(cos.Rect)
	if rectArray == nil {
		return nil
	}
	if rectArray.Size() == 4 && isNumber(rectArray.GetObject(0)) && isNumber(rectArray.GetObject(1)) &&
		isNumber(rectArray.GetObject(2)) && isNumber(rectArray.GetObject(3)) {
		return common.NewPDRectangleOfCOSArray(rectArray)
	}
	slog.Warn("annotation: not a rectangle array, returning null", "array", rectArray)
	return nil
}

func isNumber(base cos.Base) bool {
	_, ok := base.(cos.Number)
	return ok
}

// SetRectangle sets the /Rect of this annotation.
func (a *PDAnnotationBase) SetRectangle(rectangle *common.PDRectangle) {
	a.dictionary.SetItem(cos.Rect, rectangle.COSArray())
}

// AnnotationFlags returns the /F flags, which default to 0.
func (a *PDAnnotationBase) AnnotationFlags() int {
	return a.dictionary.GetIntDefault(cos.F, 0)
}

// SetAnnotationFlags sets the /F flags.
func (a *PDAnnotationBase) SetAnnotationFlags(flags int) {
	a.dictionary.SetInt(cos.F, flags)
}

// AppearanceState returns the /AS entry, or nil.
func (a *PDAnnotationBase) AppearanceState() *cos.Name {
	return a.dictionary.GetCOSName(cos.AS)
}

// SetAppearanceState sets the /AS entry.
func (a *PDAnnotationBase) SetAppearanceState(as string) {
	a.dictionary.SetName(cos.AS, as)
}

// SetAppearanceStateName sets the /AS entry from a name.
func (a *PDAnnotationBase) SetAppearanceStateName(as *cos.Name) {
	a.dictionary.SetItem(cos.AS, as)
}

// Appearance returns the /AP appearance dictionary, or nil.
func (a *PDAnnotationBase) Appearance() *PDAppearanceDictionary {
	if appearance := a.dictionary.GetCOSDictionary(cos.AP); appearance != nil {
		return NewPDAppearanceDictionaryOf(appearance)
	}
	return nil
}

// SetAppearance sets the /AP appearance dictionary.
func (a *PDAnnotationBase) SetAppearance(appearance *PDAppearanceDictionary) {
	if appearance == nil {
		a.dictionary.SetItem(cos.AP, nil)
		return
	}
	a.dictionary.SetItem(cos.AP, appearance.COSObject())
}

// NormalAppearanceStream returns the normal appearance of this annotation in
// its current state, or nil.
func (a *PDAnnotationBase) NormalAppearanceStream() *PDAppearanceStream {
	appearanceDict := a.Appearance()
	if appearanceDict == nil {
		return nil
	}
	normalAppearance := appearanceDict.NormalAppearance()
	if normalAppearance == nil {
		return nil
	}
	if normalAppearance.IsSubDictionary() {
		state := a.AppearanceState()
		streams, _ := normalAppearance.SubDictionary().Get(nameOrEmpty(state))
		return streams
	}
	// PDAppearanceStream extends PDFormXObject, but does not reference the resource cache
	return normalAppearance.AppearanceStream()
}

func nameOrEmpty(name *cos.Name) string {
	if name == nil {
		return ""
	}
	return name.Name()
}

// IsInvisible reports the invisible flag.
func (a *PDAnnotationBase) IsInvisible() bool { return a.dictionary.GetFlag(cos.F, flagInvisible) }

// SetInvisible sets the invisible flag.
func (a *PDAnnotationBase) SetInvisible(invisible bool) {
	a.dictionary.SetFlag(cos.F, flagInvisible, invisible)
}

// IsHidden reports the hidden flag.
func (a *PDAnnotationBase) IsHidden() bool { return a.dictionary.GetFlag(cos.F, flagHidden) }

// SetHidden sets the hidden flag.
func (a *PDAnnotationBase) SetHidden(hidden bool) {
	a.dictionary.SetFlag(cos.F, flagHidden, hidden)
}

// IsPrinted reports the printed flag.
func (a *PDAnnotationBase) IsPrinted() bool { return a.dictionary.GetFlag(cos.F, flagPrinted) }

// SetPrinted sets the printed flag.
func (a *PDAnnotationBase) SetPrinted(printed bool) {
	a.dictionary.SetFlag(cos.F, flagPrinted, printed)
}

// IsNoZoom reports the no-zoom flag.
func (a *PDAnnotationBase) IsNoZoom() bool { return a.dictionary.GetFlag(cos.F, flagNoZoom) }

// SetNoZoom sets the no-zoom flag.
func (a *PDAnnotationBase) SetNoZoom(noZoom bool) {
	a.dictionary.SetFlag(cos.F, flagNoZoom, noZoom)
}

// IsNoRotate reports the no-rotate flag.
func (a *PDAnnotationBase) IsNoRotate() bool { return a.dictionary.GetFlag(cos.F, flagNoRotate) }

// SetNoRotate sets the no-rotate flag.
func (a *PDAnnotationBase) SetNoRotate(noRotate bool) {
	a.dictionary.SetFlag(cos.F, flagNoRotate, noRotate)
}

// IsNoView reports the no-view flag.
func (a *PDAnnotationBase) IsNoView() bool { return a.dictionary.GetFlag(cos.F, flagNoView) }

// SetNoView sets the no-view flag.
func (a *PDAnnotationBase) SetNoView(noView bool) {
	a.dictionary.SetFlag(cos.F, flagNoView, noView)
}

// IsReadOnly reports the read-only flag.
func (a *PDAnnotationBase) IsReadOnly() bool { return a.dictionary.GetFlag(cos.F, flagReadOnly) }

// SetReadOnly sets the read-only flag.
func (a *PDAnnotationBase) SetReadOnly(readOnly bool) {
	a.dictionary.SetFlag(cos.F, flagReadOnly, readOnly)
}

// IsLocked reports the locked flag.
func (a *PDAnnotationBase) IsLocked() bool { return a.dictionary.GetFlag(cos.F, flagLocked) }

// SetLocked sets the locked flag.
func (a *PDAnnotationBase) SetLocked(locked bool) {
	a.dictionary.SetFlag(cos.F, flagLocked, locked)
}

// IsToggleNoView reports the toggle-no-view flag.
func (a *PDAnnotationBase) IsToggleNoView() bool {
	return a.dictionary.GetFlag(cos.F, flagToggleNoView)
}

// SetToggleNoView sets the toggle-no-view flag.
func (a *PDAnnotationBase) SetToggleNoView(toggleNoView bool) {
	a.dictionary.SetFlag(cos.F, flagToggleNoView, toggleNoView)
}

// IsLockedContents reports the locked-contents flag.
func (a *PDAnnotationBase) IsLockedContents() bool {
	return a.dictionary.GetFlag(cos.F, flagLockedContents)
}

// SetLockedContents sets the locked-contents flag.
func (a *PDAnnotationBase) SetLockedContents(lockedContents bool) {
	a.dictionary.SetFlag(cos.F, flagLockedContents, lockedContents)
}

// Contents returns the /Contents text of the annotation.
func (a *PDAnnotationBase) Contents() string { return a.dictionary.GetString(cos.Contents, "") }

// SetContents sets the /Contents text of the annotation.
func (a *PDAnnotationBase) SetContents(value string) { a.dictionary.SetString(cos.Contents, value) }

// ModifiedDate returns the /M date as the string the file holds.
func (a *PDAnnotationBase) ModifiedDate() string { return a.dictionary.GetString(cos.M, "") }

// SetModifiedDate sets the /M date.
func (a *PDAnnotationBase) SetModifiedDate(m string) { a.dictionary.SetString(cos.M, m) }

// SetModifiedDateTime sets the /M date from a date rather than the string the
// file holds, which is the setModifiedDate(Calendar) of Java.
func (a *PDAnnotationBase) SetModifiedDateTime(c time.Time) {
	util.SetDictionaryDate(a.dictionary, cos.M, c)
}

// AnnotationName returns the /NM name of the annotation.
func (a *PDAnnotationBase) AnnotationName() string { return a.dictionary.GetString(cos.NM, "") }

// SetAnnotationName sets the /NM name of the annotation.
func (a *PDAnnotationBase) SetAnnotationName(nm string) { a.dictionary.SetString(cos.NM, nm) }

// StructParent returns the /StructParent entry.
func (a *PDAnnotationBase) StructParent() int { return a.dictionary.GetInt(cos.StructParent) }

// SetStructParent sets the /StructParent entry.
func (a *PDAnnotationBase) SetStructParent(structParent int) {
	a.dictionary.SetInt(cos.StructParent, structParent)
}

// OptionalContent returns the /OC property list, or nil.
func (a *PDAnnotationBase) OptionalContent() markedcontent.PropertyList {
	if optionalContent := a.dictionary.GetCOSDictionary(cos.OC); optionalContent != nil {
		return markedcontent.CreatePropertyList(optionalContent)
	}
	return nil
}

// SetOptionalContent sets the /OC property list.
func (a *PDAnnotationBase) SetOptionalContent(oc markedcontent.PropertyList) {
	if oc == nil {
		a.dictionary.SetItem(cos.OC, nil)
		return
	}
	a.dictionary.SetItem(cos.OC, oc.COSObject())
}

// Border returns the /Border array, padded to three entries where the file is
// short and defaulted to [0 0 1] where it has none.
func (a *PDAnnotationBase) Border() *cos.Array {
	border := a.dictionary.GetCOSArray(cos.Border)
	if border != nil {
		if border.Size() < 3 {
			// create a copy to avoid altering the PDF
			newBorder := cos.NewArray()
			newBorder.AddAll(border.ToList())
			border = newBorder
			// Adobe Reader behaves as if missing elements are 0.
			for border.Size() < 3 {
				border.Add(cos.GetInteger(0))
			}
		}
		return border
	}
	border = cos.NewArray()
	border.Add(cos.GetInteger(0))
	border.Add(cos.GetInteger(0))
	border.Add(cos.GetInteger(1))
	return border
}

// SetBorder sets the /Border array.
func (a *PDAnnotationBase) SetBorder(borderArray *cos.Array) {
	a.dictionary.SetItem(cos.Border, borderArray)
}

// SetColor sets the /C colour of the annotation.
func (a *PDAnnotationBase) SetColor(c *color.PDColor) {
	a.dictionary.SetItem(cos.C, c.ToCOSArray())
}

// Color returns the /C colour of the annotation, or nil.
func (a *PDAnnotationBase) Color() *color.PDColor {
	return a.ColorOf(cos.C)
}

// ColorOf returns the colour under the given key, or nil.
//
// Java declares it protected; the port exports it because the subclasses in
// this package call it and Go has no such level.
func (a *PDAnnotationBase) ColorOf(itemName *cos.Name) *color.PDColor {
	cs := a.dictionary.GetCOSArray(itemName)
	if cs == nil {
		return nil
	}
	var colorSpace color.PDColorSpace
	switch cs.Size() {
	case 1:
		colorSpace = color.DeviceGray
	case 3:
		fa := cs.ToFloatArray()
		if fa[0] == fa[1] && fa[2] == fa[1] {
			// discovered while working on AppearanceGenerationTest.rectangleFullStrokeNoFill():
			// Adobe converts "rg" into "g" so lets do that too.
			return color.NewPDColorOfComponents([]float32{fa[0]}, color.DeviceGray)
		}
		return color.NewPDColorOfComponents(fa, color.DeviceRGB)
	case 4:
		colorSpace = color.DeviceCMYK
	}
	return color.NewPDColorOfCOSArray(cs, colorSpace)
}

// SetPage sets the /P page this annotation is on.
func (a *PDAnnotationBase) SetPage(page PageLike) {
	if page == nil {
		a.dictionary.SetItem(cos.P, nil)
		return
	}
	a.dictionary.SetItem(cos.P, page.COSObject())
}

// Page returns the /P page this annotation is on, or nil.
func (a *PDAnnotationBase) Page() PageLike {
	if page := a.dictionary.GetCOSDictionary(cos.P); page != nil {
		return NewPageFromDictionary(page)
	}
	return nil
}

// ConstructAppearances builds the appearance stream of this annotation. The
// base does nothing; the subclasses with a handler override it.
func (a *PDAnnotationBase) ConstructAppearances() error { return nil }

// ConstructAppearancesInDocument builds the appearance stream of this
// annotation. The base does nothing; the subclasses with a handler override it.
func (a *PDAnnotationBase) ConstructAppearancesInDocument(
	document common.COSDocumentLike) error {
	return nil
}

// CreateAnnotation returns the annotation the given object holds.
//
// Port of the static createAnnotation(COSBase).
func CreateAnnotation(base cos.Base) (PDAnnotation, error) {
	annotDic, ok := asDictionary(base)
	if !ok {
		return nil, fmt.Errorf("Error: Unknown annotation type %v", base)
	}
	subtype := annotDic.GetNameAsString(cos.Subtype, "")
	if subtype == "" {
		slog.Debug("annotation: Unknown annotation subtype")
		return NewPDAnnotationUnknownOf(annotDic), nil
	}
	switch subtype {
	case SubTypeFileAttachment:
		return NewPDAnnotationFileAttachmentOf(annotDic), nil
	case SubTypeLine:
		return NewPDAnnotationLineOf(annotDic), nil
	case SubTypeLink:
		return NewPDAnnotationLinkOf(annotDic), nil
	case SubTypePopup:
		return NewPDAnnotationPopupOf(annotDic), nil
	case SubTypeRubberStamp:
		return NewPDAnnotationRubberStampOf(annotDic), nil
	case SubTypeSquare:
		return NewPDAnnotationSquareOf(annotDic), nil
	case SubTypeCircle:
		return NewPDAnnotationCircleOf(annotDic), nil
	case SubTypePolygon:
		return NewPDAnnotationPolygonOf(annotDic), nil
	case SubTypePolyLine:
		return NewPDAnnotationPolylineOf(annotDic), nil
	case SubTypeInk:
		return NewPDAnnotationInkOf(annotDic), nil
	case SubTypeText:
		return NewPDAnnotationTextOf(annotDic), nil
	case SubTypeHighlight:
		return NewPDAnnotationHighlightOf(annotDic), nil
	case SubTypeUnderline:
		return NewPDAnnotationUnderlineOf(annotDic), nil
	case SubTypeStrikeOut:
		return NewPDAnnotationStrikeoutOf(annotDic), nil
	case SubTypeSquiggly:
		return NewPDAnnotationSquigglyOf(annotDic), nil
	case SubTypeWidget:
		return NewPDAnnotationWidgetOf(annotDic), nil
	case SubTypeFreeText:
		return NewPDAnnotationFreeTextOf(annotDic), nil
	case SubTypeCaret:
		return NewPDAnnotationCaretOf(annotDic), nil
	case SubTypeSound:
		return NewPDAnnotationSoundOf(annotDic), nil
	}
	// TODO not yet implemented:
	// Movie, Screen, PrinterMark, TrapNet, Watermark, 3D, Redact
	slog.Debug("annotation: Unknown or unsupported annotation subtype", "subtype", subtype)
	return NewPDAnnotationUnknownOf(annotDic), nil
}

// asDictionary is Java's `instanceof COSDictionary`, which a COSStream also
// satisfies.
func asDictionary(base cos.Base) (*cos.Dictionary, bool) {
	switch value := base.(type) {
	case *cos.Stream:
		return &value.Dictionary, true
	case *cos.Dictionary:
		return value, true
	}
	return nil, false
}

// AnnotationFilter decides whether an annotation is wanted.
//
// Port of the interface AnnotationFilter, whose single method Go writes as a
// function.
type AnnotationFilter func(annotation PDAnnotation) bool

// AppearanceContentStream is what an appearance handler writes an annotation's
// appearance through.
//
// Java names PDAppearanceContentStream, which lives in pdmodel; pdmodel imports
// this package for the page's annotations, so the dependency cannot run both
// ways. The port names what the handlers use and takes the constructor below,
// which pdmodel sets from its init.
type AppearanceContentStream interface {
	// MoveTo begins a new subpath at the given point.
	MoveTo(x, y float32) error

	// LineTo appends a straight line to the current path.
	LineTo(x, y float32) error

	// CurveTo appends a cubic Bezier curve to the current path.
	CurveTo(x1, y1, x2, y2, x3, y3 float32) error

	// AddRect adds a rectangle to the current path.
	AddRect(x, y, width, height float32) error

	// ClosePath closes the current subpath.
	ClosePath() error

	// Clip intersects the clipping path with the current path.
	Clip() error

	// Fill fills the current path.
	Fill() error

	// Stroke strokes the current path.
	Stroke() error

	// FillAndStroke fills and strokes the current path.
	FillAndStroke() error

	// CloseAndFillAndStroke closes, fills and strokes the current path.
	CloseAndFillAndStroke() error

	// DrawShape closes the current path the way the given stroke and fill ask.
	DrawShape(lineWidth float32, hasStroke, hasFill bool) error

	// Transform concatenates the given matrix onto the current transformation
	// matrix.
	Transform(matrix *util.Matrix) error

	// SaveGraphicsState pushes the graphics state.
	SaveGraphicsState() error

	// RestoreGraphicsState pops the graphics state.
	RestoreGraphicsState() error

	// SetGraphicsStateParameters sets the graphics state from the given
	// extended graphics state.
	SetGraphicsStateParameters(extGState *state.PDExtendedGraphicsState) error

	// SetLineWidth sets the line width.
	SetLineWidth(lineWidth float32) error

	// SetLineWidthOnDemand sets the line width unless it is the default of one.
	SetLineWidthOnDemand(lineWidth float32) error

	// SetLineCapStyle sets the line cap style.
	SetLineCapStyle(lineCapStyle int) error

	// SetLineJoinStyle sets the line join style.
	SetLineJoinStyle(lineJoinStyle int) error

	// SetMiterLimit sets the miter limit.
	SetMiterLimit(miterLimit float32) error

	// SetLineDashPattern sets the line dash pattern.
	SetLineDashPattern(pattern []float32, phase float32) error

	// SetBorderLine sets the dash pattern and the width of a border.
	SetBorderLine(lineWidth float32, bs *PDBorderStyleDictionary, border *cos.Array) error

	// SetStrokingColor sets the colour to stroke with.
	SetStrokingColor(value *color.PDColor) error

	// SetStrokingColorComponents sets the colour to stroke with from its
	// components.
	SetStrokingColorComponents(components []float32) error

	// SetStrokingColorOnDemand sets the stroking colour where there is one, and
	// reports whether it did.
	SetStrokingColorOnDemand(value *color.PDColor) (bool, error)

	// SetNonStrokingColor sets the colour to fill with.
	SetNonStrokingColor(value *color.PDColor) error

	// SetNonStrokingColorGray sets the colour to fill with, in device gray.
	SetNonStrokingColorGray(g float32) error

	// SetNonStrokingColorComponents sets the colour to fill with from its
	// components.
	SetNonStrokingColorComponents(components []float32) error

	// SetNonStrokingColorOnDemand sets the non-stroking colour where there is
	// one, and reports whether it did.
	SetNonStrokingColorOnDemand(value *color.PDColor) (bool, error)

	// BeginText begins a text object.
	BeginText() error

	// EndText ends a text object.
	EndText() error

	// SetFont sets the font and size to draw text with.
	SetFont(f font.PDFont, fontSize float32) error

	// ShowText writes the given text.
	ShowText(text string) error

	// NewLineAtOffset moves to the start of the next line of text.
	NewLineAtOffset(tx, ty float32) error

	// DrawForm draws the given form XObject.
	DrawForm(formXObject *form.PDFormXObject) error

	// Close closes the stream.
	Close() error
}

// NewAppearanceContentStream writes into the given appearance, deflating the
// content where compress is true. pdmodel sets it.
var NewAppearanceContentStream func(appearance *PDAppearanceStream,
	compress bool) (AppearanceContentStream, error)

// FormContentStream is what an appearance handler writes the content of a form
// XObject through.
//
// Java names PDFormContentStream, which lives in pdmodel for the same reason
// PDAppearanceContentStream does; this is the part of it the handlers use, and
// pdmodel sets the constructor below from its init.
type FormContentStream interface {
	// MoveTo begins a new subpath at the given point.
	MoveTo(x, y float32) error

	// LineTo appends a straight line to the current path.
	LineTo(x, y float32) error

	// CurveTo appends a cubic Bezier curve to the current path.
	CurveTo(x1, y1, x2, y2, x3, y3 float32) error

	// AddRect adds a rectangle to the current path.
	AddRect(x, y, width, height float32) error

	// Fill fills the current path.
	Fill() error

	// DrawForm draws the given form XObject.
	DrawForm(formXObject *form.PDFormXObject) error

	// SetNonStrokingColor sets the colour to fill with.
	SetNonStrokingColor(value *color.PDColor) error

	// Close closes the stream.
	Close() error
}

// NewFormContentStream writes into the given form XObject. pdmodel sets it.
var NewFormContentStream func(formXObject *form.PDFormXObject) (FormContentStream, error)
