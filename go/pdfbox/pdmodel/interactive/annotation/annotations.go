package annotation

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/filespecification"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
)

// The /Subtype names, which Java declares one per class as SUB_TYPE.
const (
	// SubTypeCaret is PDAnnotationCaret.SUB_TYPE.
	SubTypeCaret = "Caret"
	// SubTypeCircle is PDAnnotationCircle.SUB_TYPE.
	SubTypeCircle = "Circle"
	// SubTypeFileAttachment is PDAnnotationFileAttachment.SUB_TYPE.
	SubTypeFileAttachment = "FileAttachment"
	// SubTypeFreeText is PDAnnotationFreeText.SUB_TYPE.
	SubTypeFreeText = "FreeText"
	// SubTypeHighlight is PDAnnotationHighlight.SUB_TYPE.
	SubTypeHighlight = "Highlight"
	// SubTypeInk is PDAnnotationInk.SUB_TYPE.
	SubTypeInk = "Ink"
	// SubTypeLine is PDAnnotationLine.SUB_TYPE.
	SubTypeLine = "Line"
	// SubTypeLink is PDAnnotationLink.SUB_TYPE.
	SubTypeLink = "Link"
	// SubTypePolygon is PDAnnotationPolygon.SUB_TYPE.
	SubTypePolygon = "Polygon"
	// SubTypePolyLine is PDAnnotationPolyline.SUB_TYPE.
	SubTypePolyLine = "PolyLine"
	// SubTypePopup is PDAnnotationPopup.SUB_TYPE.
	SubTypePopup = "Popup"
	// SubTypeRubberStamp is PDAnnotationRubberStamp.SUB_TYPE.
	SubTypeRubberStamp = "Stamp"
	// SubTypeSound is PDAnnotationSound.SUB_TYPE.
	SubTypeSound = "Sound"
	// SubTypeSquare is PDAnnotationSquare.SUB_TYPE.
	SubTypeSquare = "Square"
	// SubTypeSquiggly is PDAnnotationSquiggly.SUB_TYPE.
	SubTypeSquiggly = "Squiggly"
	// SubTypeStrikeOut is PDAnnotationStrikeout.SUB_TYPE.
	SubTypeStrikeOut = "StrikeOut"
	// SubTypeText is PDAnnotationText.SUB_TYPE.
	SubTypeText = "Text"
	// SubTypeUnderline is PDAnnotationUnderline.SUB_TYPE.
	SubTypeUnderline = "Underline"
	// SubTypeWidget is PDAnnotationWidget.SUB_TYPE.
	SubTypeWidget = "Widget"
)

// PDAppearanceHandler builds the appearance streams of one annotation.
//
// Port of the interface PDAppearanceHandler. The handlers themselves live in
// the handlers subpackage, which imports this one; an annotation reaches its
// default handler through DefaultAppearanceHandlers, which that package fills
// from its init.
type PDAppearanceHandler interface {
	// GenerateAppearanceStreams builds the normal, rollover and down
	// appearances.
	GenerateAppearanceStreams() error
}

// DefaultAppearanceHandlers maps a /Subtype to the handler that draws it. The
// handlers subpackage fills it; until it does, ConstructAppearances on an
// annotation with no custom handler does nothing, which is what Java does for
// the subtypes that have none.
var DefaultAppearanceHandlers = map[string]func(annotation PDAnnotation,
	document common.COSDocumentLike) PDAppearanceHandler{}

// customHandler is the `private PDAppearanceHandler customAppearanceHandler`
// that fourteen of the annotations declare, and the constructAppearances pair
// that reads it.
type customHandler struct {
	customAppearanceHandler PDAppearanceHandler
}

// SetCustomAppearanceHandler sets the handler this annotation draws through.
func (c *customHandler) SetCustomAppearanceHandler(appearanceHandler PDAppearanceHandler) {
	c.customAppearanceHandler = appearanceHandler
}

// constructAppearances is the body every constructAppearances shares: the
// custom handler where there is one, the subtype's default otherwise.
func (c *customHandler) constructAppearances(annotation PDAnnotation,
	document common.COSDocumentLike) error {
	if c.customAppearanceHandler != nil {
		return c.customAppearanceHandler.GenerateAppearanceStreams()
	}
	if factory := DefaultAppearanceHandlers[annotation.Subtype()]; factory != nil {
		return factory(annotation, document).GenerateAppearanceStreams()
	}
	return nil
}

// The two reply types of a markup annotation.
const (
	// RTReply is PDAnnotationMarkup.RT_REPLY.
	RTReply = "R"
	// RTGroup is PDAnnotationMarkup.RT_GROUP.
	RTGroup = "Group"
)

// PDAnnotationMarkup is an annotation the user made: it has an author, a date
// and a popup.
//
// Port of PDAnnotationMarkup.
type PDAnnotationMarkup struct {
	PDAnnotationBase
}

var _ PDAnnotation = (*PDAnnotationMarkup)(nil)

// NewPDAnnotationMarkup creates an empty markup annotation.
func NewPDAnnotationMarkup() *PDAnnotationMarkup {
	a := &PDAnnotationMarkup{}
	a.InitAnnotation()
	return a
}

// NewPDAnnotationMarkupOf creates one over the given dictionary.
func NewPDAnnotationMarkupOf(dict *cos.Dictionary) *PDAnnotationMarkup {
	a := &PDAnnotationMarkup{}
	a.InitAnnotationOf(dict)
	return a
}

// TitlePopup returns the /T title of the popup.
func (a *PDAnnotationMarkup) TitlePopup() string {
	return a.AnnotationDictionary().GetString(cos.T, "")
}

// SetTitlePopup sets the /T title of the popup.
func (a *PDAnnotationMarkup) SetTitlePopup(t string) {
	a.AnnotationDictionary().SetString(cos.T, t)
}

// Popup returns the /Popup annotation, or nil.
func (a *PDAnnotationMarkup) Popup() *PDAnnotationPopup {
	if popup := a.AnnotationDictionary().GetCOSDictionary(cos.Popup); popup != nil {
		return NewPDAnnotationPopupOf(popup)
	}
	return nil
}

// SetPopup sets the /Popup annotation.
func (a *PDAnnotationMarkup) SetPopup(popup *PDAnnotationPopup) {
	setAnnotationItem(a.AnnotationDictionary(), cos.Popup, popup)
}

// setAnnotationItem stores a COSObjectable, or clears the entry where it is nil.
func setAnnotationItem(dict *cos.Dictionary, key *cos.Name, value common.COSObjectable) {
	if value == nil {
		dict.SetItem(key, nil)
		return
	}
	dict.SetItem(key, value.COSObject())
}

// ConstantOpacity returns the /CA opacity, which defaults to 1.
func (a *PDAnnotationMarkup) ConstantOpacity() float32 {
	return a.AnnotationDictionary().GetFloat(cos.CA, 1)
}

// SetConstantOpacity sets the /CA opacity.
func (a *PDAnnotationMarkup) SetConstantOpacity(ca float32) {
	a.AnnotationDictionary().SetFloat(cos.CA, ca)
}

// RichContents returns the /RC rich text, which may be a string or a stream.
func (a *PDAnnotationMarkup) RichContents() (string, error) {
	switch value := a.AnnotationDictionary().GetDictionaryObject(cos.RC).(type) {
	case *cos.StringObj:
		return value.Value(), nil
	case *cos.Stream:
		return value.TextString()
	}
	return "", nil
}

// SetRichContents sets the /RC rich text.
func (a *PDAnnotationMarkup) SetRichContents(rc string) {
	a.AnnotationDictionary().SetItem(cos.RC, cos.NewStringObj(rc))
}

// InReplyTo returns the /IRT annotation this one replies to, or nil.
func (a *PDAnnotationMarkup) InReplyTo() (PDAnnotation, error) {
	base := a.AnnotationDictionary().GetCOSDictionary(cos.IRT)
	if base == nil {
		return nil, nil
	}
	return CreateAnnotation(base)
}

// SetInReplyTo sets the /IRT annotation this one replies to.
func (a *PDAnnotationMarkup) SetInReplyTo(irt PDAnnotation) {
	setAnnotationItem(a.AnnotationDictionary(), cos.IRT, irt)
}

// Subject returns the /Subj subject.
func (a *PDAnnotationMarkup) Subject() string {
	return a.AnnotationDictionary().GetString(cos.Subj, "")
}

// SetSubject sets the /Subj subject.
func (a *PDAnnotationMarkup) SetSubject(subj string) {
	a.AnnotationDictionary().SetString(cos.Subj, subj)
}

// ReplyType returns the /RT reply type, which defaults to R.
func (a *PDAnnotationMarkup) ReplyType() string {
	return a.AnnotationDictionary().GetNameAsString(cos.RT, RTReply)
}

// SetReplyType sets the /RT reply type.
func (a *PDAnnotationMarkup) SetReplyType(rt string) {
	a.AnnotationDictionary().SetName(cos.RT, rt)
}

// Intent returns the /IT intent.
func (a *PDAnnotationMarkup) Intent() string {
	return a.AnnotationDictionary().GetNameAsString(cos.IT, "")
}

// SetIntent sets the /IT intent.
func (a *PDAnnotationMarkup) SetIntent(it string) {
	a.AnnotationDictionary().SetName(cos.IT, it)
}

// ExternalData returns the /ExData dictionary, or nil.
func (a *PDAnnotationMarkup) ExternalData() *PDExternalDataDictionary {
	if exData := a.AnnotationDictionary().GetCOSDictionary(cos.ExData); exData != nil {
		return NewPDExternalDataDictionaryOf(exData)
	}
	return nil
}

// SetExternalData sets the /ExData dictionary.
func (a *PDAnnotationMarkup) SetExternalData(externalData *PDExternalDataDictionary) {
	setAnnotationItem(a.AnnotationDictionary(), cos.ExData, externalData)
}

// SetBorderStyle sets the /BS border style.
func (a *PDAnnotationMarkup) SetBorderStyle(bs *PDBorderStyleDictionary) {
	setAnnotationItem(a.AnnotationDictionary(), cos.BS, bs)
}

// BorderStyle returns the /BS border style, or nil.
func (a *PDAnnotationMarkup) BorderStyle() *PDBorderStyleDictionary {
	if bs := a.AnnotationDictionary().GetCOSDictionary(cos.BS); bs != nil {
		return NewPDBorderStyleDictionaryOf(bs)
	}
	return nil
}

// PDAnnotationTextMarkup is a markup annotation over a run of text.
//
// Port of the abstract PDAnnotationTextMarkup.
type PDAnnotationTextMarkup struct {
	PDAnnotationMarkup
}

// initTextMarkup is the protected PDAnnotationTextMarkup(String) constructor.
func (a *PDAnnotationTextMarkup) initTextMarkup(subType string) {
	a.InitAnnotation()
	a.SetSubtype(subType)
	// Quad points are required, set an empty array
	a.SetQuadPoints(nil)
}

// SetQuadPoints sets the /QuadPoints of this annotation. Java declares it
// final.
func (a *PDAnnotationTextMarkup) SetQuadPoints(quadPoints []float32) {
	a.AnnotationDictionary().SetItem(cos.QuadPoints, cos.ArrayOfFloats(quadPoints))
}

// QuadPoints returns the /QuadPoints of this annotation, or nil.
func (a *PDAnnotationTextMarkup) QuadPoints() []float32 {
	if array := a.AnnotationDictionary().GetCOSArray(cos.QuadPoints); array != nil {
		return array.ToFloatArray()
	}
	return nil
}

// PDAnnotationSquareCircle is a square or a circle.
//
// Port of the abstract PDAnnotationSquareCircle.
type PDAnnotationSquareCircle struct {
	PDAnnotationMarkup
}

// SetInteriorColor sets the /IC interior colour.
func (a *PDAnnotationSquareCircle) SetInteriorColor(ic *color.PDColor) {
	a.AnnotationDictionary().SetItem(cos.IC, ic.ToCOSArray())
}

// InteriorColor returns the /IC interior colour, or nil.
func (a *PDAnnotationSquareCircle) InteriorColor() *color.PDColor {
	return a.ColorOf(cos.IC)
}

// SetBorderEffect sets the /BE border effect.
func (a *PDAnnotationSquareCircle) SetBorderEffect(be *PDBorderEffectDictionary) {
	setAnnotationItem(a.AnnotationDictionary(), cos.BE, be)
}

// BorderEffect returns the /BE border effect, or nil.
func (a *PDAnnotationSquareCircle) BorderEffect() *PDBorderEffectDictionary {
	if borderEffect := a.AnnotationDictionary().GetCOSDictionary(cos.BE); borderEffect != nil {
		return NewPDBorderEffectDictionaryOf(borderEffect)
	}
	return nil
}

// SetRectDifference sets the /RD rectangle difference.
func (a *PDAnnotationSquareCircle) SetRectDifference(rd *common.PDRectangle) {
	setAnnotationItem(a.AnnotationDictionary(), cos.RD, rd)
}

// RectDifference returns the /RD rectangle difference, or nil.
func (a *PDAnnotationSquareCircle) RectDifference() *common.PDRectangle {
	if difference := a.AnnotationDictionary().GetCOSArray(cos.RD); difference != nil {
		return common.NewPDRectangleOfCOSArray(difference)
	}
	return nil
}

// SetRectDifferences sets the /RD margins, all four the same.
func (a *PDAnnotationSquareCircle) SetRectDifferences(difference float32) {
	a.SetRectDifferencesOf(difference, difference, difference, difference)
}

// SetRectDifferencesOf sets the four /RD margins.
func (a *PDAnnotationSquareCircle) SetRectDifferencesOf(left, top, right, bottom float32) {
	setRectDifferences(a.AnnotationDictionary(), left, top, right, bottom)
}

// setRectDifferences is the body PDAnnotationSquareCircle and PDAnnotationCaret
// both declare.
func setRectDifferences(dict *cos.Dictionary, left, top, right, bottom float32) {
	margins := cos.NewArray()
	margins.Add(cos.NewFloat(left))
	margins.Add(cos.NewFloat(top))
	margins.Add(cos.NewFloat(right))
	margins.Add(cos.NewFloat(bottom))
	dict.SetItem(cos.RD, margins)
}

// RectDifferences returns the four /RD margins, or an empty slice.
func (a *PDAnnotationSquareCircle) RectDifferences() []float32 {
	return rectDifferences(a.AnnotationDictionary())
}

func rectDifferences(dict *cos.Dictionary) []float32 {
	if margin := dict.GetCOSArray(cos.RD); margin != nil {
		return margin.ToFloatArray()
	}
	return []float32{}
}

// PDAnnotationUnknown is an annotation whose subtype PDFBox does not know.
//
// Port of PDAnnotationUnknown.
type PDAnnotationUnknown struct {
	PDAnnotationBase
}

var _ PDAnnotation = (*PDAnnotationUnknown)(nil)

// NewPDAnnotationUnknownOf creates one over the given dictionary.
func NewPDAnnotationUnknownOf(dic *cos.Dictionary) *PDAnnotationUnknown {
	a := &PDAnnotationUnknown{}
	a.InitAnnotationOf(dic)
	return a
}

// PDAnnotationCaret marks where text is to be inserted.
type PDAnnotationCaret struct {
	PDAnnotationMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationCaret)(nil)

// NewPDAnnotationCaret creates a new caret annotation.
func NewPDAnnotationCaret() *PDAnnotationCaret {
	a := &PDAnnotationCaret{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeCaret)
	return a
}

// NewPDAnnotationCaretOf creates one over the given dictionary.
func NewPDAnnotationCaretOf(field *cos.Dictionary) *PDAnnotationCaret {
	a := &PDAnnotationCaret{}
	a.InitAnnotationOf(field)
	return a
}

// SetRectDifferences sets the /RD margins, all four the same.
func (a *PDAnnotationCaret) SetRectDifferences(difference float32) {
	a.SetRectDifferencesOf(difference, difference, difference, difference)
}

// SetRectDifferencesOf sets the four /RD margins.
func (a *PDAnnotationCaret) SetRectDifferencesOf(left, top, right, bottom float32) {
	setRectDifferences(a.AnnotationDictionary(), left, top, right, bottom)
}

// RectDifferences returns the four /RD margins, or an empty slice.
func (a *PDAnnotationCaret) RectDifferences() []float32 {
	return rectDifferences(a.AnnotationDictionary())
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationCaret) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationCaret) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationCircle is an ellipse.
type PDAnnotationCircle struct {
	PDAnnotationSquareCircle
	customHandler
}

var _ PDAnnotation = (*PDAnnotationCircle)(nil)

// NewPDAnnotationCircle creates a new circle annotation.
func NewPDAnnotationCircle() *PDAnnotationCircle {
	a := &PDAnnotationCircle{}
	a.InitAnnotation()
	a.SetSubtype(SubTypeCircle)
	return a
}

// NewPDAnnotationCircleOf creates one over the given dictionary.
func NewPDAnnotationCircleOf(field *cos.Dictionary) *PDAnnotationCircle {
	a := &PDAnnotationCircle{}
	a.InitAnnotationOf(field)
	return a
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationCircle) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationCircle) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationSquare is a rectangle.
type PDAnnotationSquare struct {
	PDAnnotationSquareCircle
	customHandler
}

var _ PDAnnotation = (*PDAnnotationSquare)(nil)

// NewPDAnnotationSquare creates a new square annotation.
func NewPDAnnotationSquare() *PDAnnotationSquare {
	a := &PDAnnotationSquare{}
	a.InitAnnotation()
	a.SetSubtype(SubTypeSquare)
	return a
}

// NewPDAnnotationSquareOf creates one over the given dictionary.
func NewPDAnnotationSquareOf(field *cos.Dictionary) *PDAnnotationSquare {
	a := &PDAnnotationSquare{}
	a.InitAnnotationOf(field)
	return a
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationSquare) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationSquare) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationHighlight highlights a run of text.
type PDAnnotationHighlight struct {
	PDAnnotationTextMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationHighlight)(nil)

// NewPDAnnotationHighlight creates a new highlight annotation.
func NewPDAnnotationHighlight() *PDAnnotationHighlight {
	a := &PDAnnotationHighlight{}
	a.initTextMarkup(SubTypeHighlight)
	return a
}

// NewPDAnnotationHighlightOf creates one over the given dictionary.
func NewPDAnnotationHighlightOf(dict *cos.Dictionary) *PDAnnotationHighlight {
	a := &PDAnnotationHighlight{}
	a.InitAnnotationOf(dict)
	return a
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationHighlight) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationHighlight) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationUnderline underlines a run of text.
type PDAnnotationUnderline struct {
	PDAnnotationTextMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationUnderline)(nil)

// NewPDAnnotationUnderline creates a new underline annotation.
func NewPDAnnotationUnderline() *PDAnnotationUnderline {
	a := &PDAnnotationUnderline{}
	a.initTextMarkup(SubTypeUnderline)
	return a
}

// NewPDAnnotationUnderlineOf creates one over the given dictionary.
func NewPDAnnotationUnderlineOf(dict *cos.Dictionary) *PDAnnotationUnderline {
	a := &PDAnnotationUnderline{}
	a.InitAnnotationOf(dict)
	return a
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationUnderline) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationUnderline) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationStrikeout strikes out a run of text.
type PDAnnotationStrikeout struct {
	PDAnnotationTextMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationStrikeout)(nil)

// NewPDAnnotationStrikeout creates a new strikeout annotation.
func NewPDAnnotationStrikeout() *PDAnnotationStrikeout {
	a := &PDAnnotationStrikeout{}
	a.initTextMarkup(SubTypeStrikeOut)
	return a
}

// NewPDAnnotationStrikeoutOf creates one over the given dictionary.
func NewPDAnnotationStrikeoutOf(dict *cos.Dictionary) *PDAnnotationStrikeout {
	a := &PDAnnotationStrikeout{}
	a.InitAnnotationOf(dict)
	return a
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationStrikeout) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationStrikeout) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationSquiggly underlines a run of text with a wavy line.
type PDAnnotationSquiggly struct {
	PDAnnotationTextMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationSquiggly)(nil)

// NewPDAnnotationSquiggly creates a new squiggly annotation.
func NewPDAnnotationSquiggly() *PDAnnotationSquiggly {
	a := &PDAnnotationSquiggly{}
	a.initTextMarkup(SubTypeSquiggly)
	return a
}

// NewPDAnnotationSquigglyOf creates one over the given dictionary.
func NewPDAnnotationSquigglyOf(dict *cos.Dictionary) *PDAnnotationSquiggly {
	a := &PDAnnotationSquiggly{}
	a.InitAnnotationOf(dict)
	return a
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationSquiggly) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationSquiggly) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// PDAnnotationPopup is the window that shows a markup annotation's text.
type PDAnnotationPopup struct {
	PDAnnotationBase
}

var _ PDAnnotation = (*PDAnnotationPopup)(nil)

// NewPDAnnotationPopup creates a new popup annotation.
func NewPDAnnotationPopup() *PDAnnotationPopup {
	a := &PDAnnotationPopup{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypePopup)
	return a
}

// NewPDAnnotationPopupOf creates one over the given dictionary.
func NewPDAnnotationPopupOf(field *cos.Dictionary) *PDAnnotationPopup {
	a := &PDAnnotationPopup{}
	a.InitAnnotationOf(field)
	return a
}

// SetOpen sets whether the popup is open.
func (a *PDAnnotationPopup) SetOpen(open bool) {
	a.AnnotationDictionary().SetBoolean(cos.GetPDFName("Open"), open)
}

// Open reports whether the popup is open.
func (a *PDAnnotationPopup) Open() bool {
	return a.AnnotationDictionary().GetBoolean(cos.GetPDFName("Open"), false)
}

// SetParent sets the markup annotation this popup belongs to.
func (a *PDAnnotationPopup) SetParent(annot *PDAnnotationMarkup) {
	a.AnnotationDictionary().SetItem(cos.Parent, annot.COSObject())
}

// Parent returns the markup annotation this popup belongs to, or nil.
func (a *PDAnnotationPopup) Parent() *PDAnnotationMarkup {
	ann, err := CreateAnnotation(a.AnnotationDictionary().GetDictionaryObject2(cos.Parent, cos.P))
	if err != nil {
		slog.Debug("annotation: An exception while trying to get the parent markup - ignoring",
			"err", err)
		// Couldn't construct the annotation, so return null i.e. do nothing
		return nil
	}
	markup, ok := ann.(*PDAnnotationMarkup)
	if !ok {
		slog.Error("annotation: parent annotation is not a markup annotation",
			"type", ann)
		return nil
	}
	return markup
}

// PDAnnotationSound plays a sound.
type PDAnnotationSound struct {
	PDAnnotationMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationSound)(nil)

// NewPDAnnotationSound creates a new sound annotation.
func NewPDAnnotationSound() *PDAnnotationSound {
	a := &PDAnnotationSound{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeSound)
	return a
}

// NewPDAnnotationSoundOf creates one over the given dictionary.
func NewPDAnnotationSoundOf(field *cos.Dictionary) *PDAnnotationSound {
	a := &PDAnnotationSound{}
	a.InitAnnotationOf(field)
	return a
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationSound) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationSound) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// The stamp names, PDF 32000-1:2008 Table 181.
const (
	// StampNameApproved is PDAnnotationRubberStamp.NAME_APPROVED.
	StampNameApproved = "Approved"
	// StampNameExperimental is NAME_EXPERIMENTAL.
	StampNameExperimental = "Experimental"
	// StampNameNotApproved is NAME_NOT_APPROVED.
	StampNameNotApproved = "NotApproved"
	// StampNameAsIs is NAME_AS_IS.
	StampNameAsIs = "AsIs"
	// StampNameExpired is NAME_EXPIRED.
	StampNameExpired = "Expired"
	// StampNameNotForPublicRelease is NAME_NOT_FOR_PUBLIC_RELEASE.
	StampNameNotForPublicRelease = "NotForPublicRelease"
	// StampNameForPublicRelease is NAME_FOR_PUBLIC_RELEASE.
	StampNameForPublicRelease = "ForPublicRelease"
	// StampNameDraft is NAME_DRAFT, and the default.
	StampNameDraft = "Draft"
	// StampNameForComment is NAME_FOR_COMMENT.
	StampNameForComment = "ForComment"
	// StampNameTopSecret is NAME_TOP_SECRET.
	StampNameTopSecret = "TopSecret"
	// StampNameDepartmental is NAME_DEPARTMENTAL.
	StampNameDepartmental = "Departmental"
	// StampNameConfidential is NAME_CONFIDENTIAL.
	StampNameConfidential = "Confidential"
	// StampNameFinal is NAME_FINAL.
	StampNameFinal = "Final"
	// StampNameSold is NAME_SOLD.
	StampNameSold = "Sold"
)

// PDAnnotationRubberStamp stamps a word on the page.
type PDAnnotationRubberStamp struct {
	PDAnnotationMarkup
}

var _ PDAnnotation = (*PDAnnotationRubberStamp)(nil)

// NewPDAnnotationRubberStamp creates a new stamp annotation.
func NewPDAnnotationRubberStamp() *PDAnnotationRubberStamp {
	a := &PDAnnotationRubberStamp{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeRubberStamp)
	return a
}

// NewPDAnnotationRubberStampOf creates one over the given dictionary.
func NewPDAnnotationRubberStampOf(field *cos.Dictionary) *PDAnnotationRubberStamp {
	a := &PDAnnotationRubberStamp{}
	a.InitAnnotationOf(field)
	return a
}

// SetName sets the /Name of the stamp.
func (a *PDAnnotationRubberStamp) SetName(name string) {
	a.AnnotationDictionary().SetName(cos.NameKey, name)
}

// Name returns the /Name of the stamp, which defaults to Draft.
func (a *PDAnnotationRubberStamp) Name() string {
	return a.AnnotationDictionary().GetNameAsString(cos.NameKey, StampNameDraft)
}

// The file attachment icon names, PDF 32000-1:2008 Table 184.
const (
	// AttachmentNamePushPin is ATTACHMENT_NAME_PUSH_PIN, and the default.
	AttachmentNamePushPin = "PushPin"
	// AttachmentNameGraph is ATTACHMENT_NAME_GRAPH.
	AttachmentNameGraph = "Graph"
	// AttachmentNamePaperclip is ATTACHMENT_NAME_PAPERCLIP.
	AttachmentNamePaperclip = "Paperclip"
	// AttachmentNameTag is ATTACHMENT_NAME_TAG.
	AttachmentNameTag = "Tag"
)

// PDAnnotationFileAttachment attaches a file to the page.
type PDAnnotationFileAttachment struct {
	PDAnnotationMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationFileAttachment)(nil)

// NewPDAnnotationFileAttachment creates a new file attachment annotation.
func NewPDAnnotationFileAttachment() *PDAnnotationFileAttachment {
	a := &PDAnnotationFileAttachment{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeFileAttachment)
	return a
}

// NewPDAnnotationFileAttachmentOf creates one over the given dictionary.
func NewPDAnnotationFileAttachmentOf(field *cos.Dictionary) *PDAnnotationFileAttachment {
	a := &PDAnnotationFileAttachment{}
	a.InitAnnotationOf(field)
	return a
}

// File returns the /FS attached file.
func (a *PDAnnotationFileAttachment) File() (filespecification.PDFileSpecification, error) {
	return filespecification.CreateFS(a.AnnotationDictionary().GetDictionaryObject(cos.FS))
}

// SetFile sets the /FS attached file.
func (a *PDAnnotationFileAttachment) SetFile(file filespecification.PDFileSpecification) {
	setAnnotationItem(a.AnnotationDictionary(), cos.FS, file)
}

// AttachmentName returns the /Name icon, which defaults to PushPin.
func (a *PDAnnotationFileAttachment) AttachmentName() string {
	return a.AnnotationDictionary().GetNameAsString(cos.NameKey, AttachmentNamePushPin)
}

// SetAttachmentName sets the /Name icon.
func (a *PDAnnotationFileAttachment) SetAttachmentName(name string) {
	a.AnnotationDictionary().SetName(cos.NameKey, name)
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationFileAttachment) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationFileAttachment) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// The text annotation icon names, PDF 32000-1:2008 Table 172.
const (
	// TextNameComment is PDAnnotationText.NAME_COMMENT.
	TextNameComment = "Comment"
	// TextNameKey is NAME_KEY.
	TextNameKey = "Key"
	// TextNameNote is NAME_NOTE, and the default.
	TextNameNote = "Note"
	// TextNameHelp is NAME_HELP.
	TextNameHelp = "Help"
	// TextNameNewParagraph is NAME_NEW_PARAGRAPH.
	TextNameNewParagraph = "NewParagraph"
	// TextNameParagraph is NAME_PARAGRAPH.
	TextNameParagraph = "Paragraph"
	// TextNameInsert is NAME_INSERT.
	TextNameInsert = "Insert"
	// TextNameCircle is NAME_CIRCLE.
	TextNameCircle = "Circle"
	// TextNameCross is NAME_CROSS.
	TextNameCross = "Cross"
	// TextNameStar is NAME_STAR.
	TextNameStar = "Star"
	// TextNameCheck is NAME_CHECK.
	TextNameCheck = "Check"
	// TextNameRightArrow is NAME_RIGHT_ARROW.
	TextNameRightArrow = "RightArrow"
	// TextNameRightPointer is NAME_RIGHT_POINTER.
	TextNameRightPointer = "RightPointer"
	// TextNameUpArrow is NAME_UP_ARROW.
	TextNameUpArrow = "UpArrow"
	// TextNameUpLeftArrow is NAME_UP_LEFT_ARROW.
	TextNameUpLeftArrow = "UpLeftArrow"
	// TextNameCrossHairs is NAME_CROSS_HAIRS.
	TextNameCrossHairs = "CrossHairs"
)

// PDAnnotationText is a sticky note.
type PDAnnotationText struct {
	PDAnnotationMarkup
	customHandler
}

var _ PDAnnotation = (*PDAnnotationText)(nil)

// NewPDAnnotationText creates a new text annotation.
func NewPDAnnotationText() *PDAnnotationText {
	a := &PDAnnotationText{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeText)
	return a
}

// NewPDAnnotationTextOf creates one over the given dictionary.
func NewPDAnnotationTextOf(field *cos.Dictionary) *PDAnnotationText {
	a := &PDAnnotationText{}
	a.InitAnnotationOf(field)
	return a
}

// SetOpen sets whether the note is open.
func (a *PDAnnotationText) SetOpen(open bool) {
	a.AnnotationDictionary().SetBoolean(cos.Open, open)
}

// Open reports whether the note is open.
func (a *PDAnnotationText) Open() bool {
	return a.AnnotationDictionary().GetBoolean(cos.Open, false)
}

// SetName sets the /Name icon.
func (a *PDAnnotationText) SetName(name string) {
	a.AnnotationDictionary().SetName(cos.NameKey, name)
}

// Name returns the /Name icon, which defaults to Note.
func (a *PDAnnotationText) Name() string {
	return a.AnnotationDictionary().GetNameAsString(cos.NameKey, TextNameNote)
}

// State returns the /State of the note.
func (a *PDAnnotationText) State() string {
	return a.AnnotationDictionary().GetString(cos.State, "")
}

// SetState sets the /State of the note.
func (a *PDAnnotationText) SetState(state string) {
	a.AnnotationDictionary().SetString(cos.State, state)
}

// StateModel returns the /StateModel of the note.
func (a *PDAnnotationText) StateModel() string {
	return a.AnnotationDictionary().GetString(cos.StateModel, "")
}

// SetStateModel sets the /StateModel of the note.
func (a *PDAnnotationText) SetStateModel(stateModel string) {
	a.AnnotationDictionary().SetString(cos.StateModel, stateModel)
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationText) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationText) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}

// The link highlight modes, PDF 32000-1:2008 Table 173.
const (
	// HighlightModeNone is PDAnnotationLink.HIGHLIGHT_MODE_NONE.
	HighlightModeNone = "N"
	// HighlightModeInvert is HIGHLIGHT_MODE_INVERT, and the default.
	HighlightModeInvert = "I"
	// HighlightModeOutline is HIGHLIGHT_MODE_OUTLINE.
	HighlightModeOutline = "O"
	// HighlightModePush is HIGHLIGHT_MODE_PUSH.
	HighlightModePush = "P"
)

// PDAnnotationLink is a hyperlink.
type PDAnnotationLink struct {
	PDAnnotationBase
	customHandler
}

var _ PDAnnotation = (*PDAnnotationLink)(nil)

// NewPDAnnotationLink creates a new link annotation.
func NewPDAnnotationLink() *PDAnnotationLink {
	a := &PDAnnotationLink{}
	a.InitAnnotation()
	a.AnnotationDictionary().SetName(cos.Subtype, SubTypeLink)
	return a
}

// NewPDAnnotationLinkOf creates one over the given dictionary.
func NewPDAnnotationLinkOf(field *cos.Dictionary) *PDAnnotationLink {
	a := &PDAnnotationLink{}
	a.InitAnnotationOf(field)
	return a
}

// Action returns the /A action, or nil.
func (a *PDAnnotationLink) Action() action.Action {
	if act := a.AnnotationDictionary().GetCOSDictionary(cos.A); act != nil {
		return action.CreateAction(act)
	}
	return nil
}

// SetAction sets the /A action.
func (a *PDAnnotationLink) SetAction(act action.Action) {
	setAnnotationItem(a.AnnotationDictionary(), cos.A, act)
}

// SetQuadPoints sets the /QuadPoints that say where this annotation is
// activated.
func (a *PDAnnotationLink) SetQuadPoints(quadPoints []float32) {
	a.AnnotationDictionary().SetItem(cos.QuadPoints, cos.ArrayOfFloats(quadPoints))
}

// QuadPoints returns the /QuadPoints that say where this annotation is
// activated, or nil.
func (a *PDAnnotationLink) QuadPoints() []float32 {
	if array := a.AnnotationDictionary().GetCOSArray(cos.QuadPoints); array != nil {
		return array.ToFloatArray()
	}
	return nil
}

// SetBorderStyle sets the /BS border style.
func (a *PDAnnotationLink) SetBorderStyle(bs *PDBorderStyleDictionary) {
	setAnnotationItem(a.AnnotationDictionary(), cos.BS, bs)
}

// BorderStyle returns the /BS border style, or nil.
func (a *PDAnnotationLink) BorderStyle() *PDBorderStyleDictionary {
	if bs := a.AnnotationDictionary().GetCOSDictionary(cos.BS); bs != nil {
		return NewPDBorderStyleDictionaryOf(bs)
	}
	return nil
}

// Destination returns the /Dest destination.
func (a *PDAnnotationLink) Destination() (destination.PDDestination, error) {
	return destination.Create(a.AnnotationDictionary().GetDictionaryObject(cos.Dest))
}

// SetDestination sets the /Dest destination.
func (a *PDAnnotationLink) SetDestination(dest destination.PDDestination) {
	setAnnotationItem(a.AnnotationDictionary(), cos.Dest, dest)
}

// HighlightMode returns the /H highlight mode, which defaults to invert.
func (a *PDAnnotationLink) HighlightMode() string {
	return a.AnnotationDictionary().GetNameAsString(cos.H, HighlightModeInvert)
}

// SetHighlightMode sets the /H highlight mode.
func (a *PDAnnotationLink) SetHighlightMode(mode string) {
	a.AnnotationDictionary().SetName(cos.H, mode)
}

// SetPreviousURI sets the /PA previous URI action.
func (a *PDAnnotationLink) SetPreviousURI(pa *action.PDActionURI) {
	setAnnotationItem(a.AnnotationDictionary(), cos.PA, pa)
}

// PreviousURI returns the /PA previous URI action, or nil.
func (a *PDAnnotationLink) PreviousURI() *action.PDActionURI {
	if previousURI := a.AnnotationDictionary().GetCOSDictionary(cos.PA); previousURI != nil {
		return action.NewPDActionURIOf(previousURI)
	}
	return nil
}

// ConstructAppearances builds the appearance of this annotation.
func (a *PDAnnotationLink) ConstructAppearances() error {
	return a.constructAppearances(a, nil)
}

// ConstructAppearancesInDocument builds the appearance of this annotation, with
// the document its streams belong to.
func (a *PDAnnotationLink) ConstructAppearancesInDocument(document common.COSDocumentLike) error {
	return a.constructAppearances(a, document)
}
