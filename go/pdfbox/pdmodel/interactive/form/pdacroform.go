package form

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	graphicsform "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// The bits of the /SigFlags of a form. Java declares them private.
const (
	flagSignaturesExist = 1
	flagAppendOnly      = 1 << 1
)

// PDAcroForm is the interactive form of a document.
//
// Port of PDAcroForm, which Java declares final.
//
// importFDF and exportFDF are not here: they name FDFDocument, and pdmodel/fdf
// is not ported yet. They land with it. See migration/STATUS.md.
type PDAcroForm struct {
	document   *pdmodel.PDDocument
	dictionary *cos.Dictionary

	fieldCache       map[string]PDField
	scriptingHandler ScriptingHandler

	// directFontCache holds the fonts of the default resources, so that the
	// fields share one cache. Java holds each through a SoftReference, which Go
	// has none of; the port holds them outright, as PDResources already does.
	directFontCache map[*cos.Name]font.PDFont

	glyphLayoutProcessor pdmodel.GlyphLayoutProcessor
}

var _ common.COSObjectable = (*PDAcroForm)(nil)

// NewPDAcroForm creates an empty form for the given document.
func NewPDAcroForm(doc *pdmodel.PDDocument) *PDAcroForm {
	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.Fields, cos.NewArray())
	return &PDAcroForm{
		document:        doc,
		dictionary:      dictionary,
		directFontCache: map[*cos.Name]font.PDFont{},
	}
}

// NewPDAcroFormOf creates one over the given dictionary.
func NewPDAcroFormOf(doc *pdmodel.PDDocument, form *cos.Dictionary) *PDAcroForm {
	return &PDAcroForm{
		document:        doc,
		dictionary:      form,
		directFontCache: map[*cos.Name]font.PDFont{},
	}
}

// SetGlyphLayoutProcessor sets the processor the field appearances lay text out
// with.
func (a *PDAcroForm) SetGlyphLayoutProcessor(glyphLayoutProcessor pdmodel.GlyphLayoutProcessor) {
	a.glyphLayoutProcessor = glyphLayoutProcessor
}

// GlyphLayoutProcessor returns the processor the field appearances lay text out
// with.
func (a *PDAcroForm) GlyphLayoutProcessor() pdmodel.GlyphLayoutProcessor {
	return a.glyphLayoutProcessor
}

// Document returns the document the form belongs to. Java declares it
// package-private.
func (a *PDAcroForm) Document() *pdmodel.PDDocument { return a.document }

// COSObject returns the form dictionary.
func (a *PDAcroForm) COSObject() cos.Base { return a.dictionary }

// Dictionary returns the form dictionary, typed.
func (a *PDAcroForm) Dictionary() *cos.Dictionary { return a.dictionary }

// Flatten draws every field into the page it sits on and removes it.
func (a *PDAcroForm) Flatten() error {
	// for dynamic XFA forms there is no flatten as this would mean to do a rendering
	// from the XFA content into a static PDF.
	if a.XFAIsDynamic() {
		slog.Warn("form: flatten for a dynamix XFA form is not supported")
		return nil
	}
	fields := []PDField{}
	for field := range a.FieldTree().All() {
		fields = append(fields, field)
	}
	return a.FlattenFields(fields, false)
}

// FlattenFields draws the given fields into the pages they sit on and removes
// them.
//
// Java names this flatten(List, boolean), overloading flatten().
func (a *PDAcroForm) FlattenFields(fields []PDField, refreshAppearances bool) error {
	// Nothing to flatten if there are no fields provided
	if len(fields) == 0 {
		return nil
	}

	if !refreshAppearances && a.NeedAppearances() {
		slog.Warn("form: acroForm.getNeedAppearances() returns true, " +
			"visual field appearances may not have been set")
		slog.Warn("form: call acroForm.refreshAppearances() or " +
			"use the flatten() method with refreshAppearances parameter")
	}

	// for dynamic XFA forms there is no flatten as this would mean to do a rendering
	// from the XFA content into a static PDF.
	if a.XFAIsDynamic() {
		slog.Warn("form: flatten for a dynamix XFA form is not supported")
		return nil
	}

	// refresh the appearances if set
	if refreshAppearances {
		if err := a.RefreshAppearancesOf(fields); err != nil {
			return err
		}
	}

	pages := a.document.Pages()
	pagesWidgetsMap := a.buildPagesWidgetsMap(fields, pages)

	// preserve all non widget annotations
	for page := range pages.All {
		// get the widgets that are to be flattened for this page
		widgetsForPageMap := pagesWidgetsMap[page.Dictionary()]

		// indicates if the original content stream
		// has been wrapped in a q...Q pair.
		isContentStreamWrapped := false

		annotations := []annotation.PDAnnotation{}
		for _, annot := range page.Annotations().ToSlice() {
			if widgetsForPageMap == nil || !widgetsForPageMap[annot.AnnotationDictionary()] {
				annotations = append(annotations, annot)
				continue
			}
			if !isVisibleAnnotation(annot) {
				continue
			}
			wrapped, err := a.drawAnnotation(page, annot, isContentStreamWrapped)
			if err != nil {
				return err
			}
			isContentStreamWrapped = wrapped
		}
		page.SetAnnotations(annotations)
	}

	// remove the fields
	a.removeFields(fields)

	// remove XFA for hybrid forms
	a.dictionary.RemoveItem(cos.XFA)

	// Java removes /SigFlags where no signature is left, through
	// PDDocument.getSignatureDictionaries; that names PDSignature, which
	// pdmodel/interactive/digitalsignature brings. See migration/STATUS.md.
	return nil
}

// drawAnnotation draws one widget into the page, and reports whether the
// content stream is now wrapped.
func (a *PDAcroForm) drawAnnotation(page *pdmodel.PDPage, annot annotation.PDAnnotation,
	isContentStreamWrapped bool) (bool, error) {
	contentStream, err := pdmodel.NewPDPageContentStreamOfMode(a.document, page,
		pdmodel.Append, true, !isContentStreamWrapped)
	if err != nil {
		return isContentStreamWrapped, err
	}
	defer contentStream.Close()

	if a.glyphLayoutProcessor != nil {
		contentStream.SetGlyphLayoutProcessor(a.glyphLayoutProcessor)
	}

	appearanceStream := annot.NormalAppearanceStream()
	fieldObject := graphicsform.NewPDFormXObjectOfStream(appearanceStream.Stream())

	if err := contentStream.SaveGraphicsState(); err != nil {
		return true, err
	}

	// see https://stackoverflow.com/a/54091766/1729265 for an explanation
	// of the steps required
	// this will transform the appearance stream form object into the rectangle of the
	// annotation bbox and map the coordinate systems
	transformationMatrix := resolveTransformationMatrix(annot, appearanceStream)
	if err := contentStream.Transform(transformationMatrix); err != nil {
		return true, err
	}
	if err := contentStream.DrawForm(fieldObject); err != nil {
		return true, err
	}
	if err := contentStream.RestoreGraphicsState(); err != nil {
		return true, err
	}
	return true, nil
}

// isVisibleAnnotation reports whether the annotation would be seen. Java
// declares it private.
func isVisibleAnnotation(annot annotation.PDAnnotation) bool {
	if annot.IsInvisible() || annot.IsHidden() {
		return false
	}
	normalAppearanceStream := annot.NormalAppearanceStream()
	if normalAppearanceStream == nil {
		return false
	}
	bbox := normalAppearanceStream.BBox()
	return bbox != nil && bbox.Width() > 0 && bbox.Height() > 0
}

// RefreshAppearances rebuilds the appearance of every field.
func (a *PDAcroForm) RefreshAppearances() error {
	for field := range a.FieldTree().All() {
		if terminal, isTerminal := field.(terminalField); isTerminal {
			if err := terminal.constructAppearances(); err != nil {
				return err
			}
		}
	}
	return nil
}

// RefreshAppearancesOf rebuilds the appearance of the given fields.
//
// Java names this refreshAppearances(List), overloading refreshAppearances().
func (a *PDAcroForm) RefreshAppearancesOf(fields []PDField) error {
	for _, field := range fields {
		if terminal, isTerminal := field.(terminalField); isTerminal {
			if err := terminal.constructAppearances(); err != nil {
				return err
			}
		}
	}
	return nil
}

// Fields returns the top level fields of the form.
func (a *PDAcroForm) Fields() []PDField {
	cosFields := a.dictionary.GetCOSArray(cos.Fields)
	if cosFields == nil {
		return []PDField{}
	}
	pdFields := []PDField{}
	for i := 0; i < cosFields.Size(); i++ {
		if element, isDictionary := cosFields.GetObject(i).(*cos.Dictionary); isDictionary {
			if field := fieldFromDictionary(a, element, nil); field != nil {
				pdFields = append(pdFields, field)
			}
		}
	}
	return pdFields
}

// SetFields sets the top level fields of the form.
func (a *PDAcroForm) SetFields(fields []PDField) {
	array := cos.NewArray()
	for _, field := range fields {
		array.Add(field.COSObject())
	}
	a.dictionary.SetItem(cos.Fields, array)
}

// FieldTree returns a walk over every field of the form.
func (a *PDAcroForm) FieldTree() *PDFieldTree { return NewPDFieldTree(a) }

// SetCacheFields sets whether the fields are looked up through a cache.
func (a *PDAcroForm) SetCacheFields(cache bool) {
	if !cache {
		a.fieldCache = nil
		return
	}
	a.fieldCache = map[string]PDField{}
	for field := range a.FieldTree().All() {
		a.fieldCache[field.FullyQualifiedName()] = field
	}
}

// IsCachingFields reports whether the fields are looked up through a cache.
func (a *PDAcroForm) IsCachingFields() bool { return a.fieldCache != nil }

// Field returns the field with the given fully qualified name, or nil.
func (a *PDAcroForm) Field(fullyQualifiedName string) PDField {
	// get the field from the cache if there is one.
	if a.fieldCache != nil {
		return a.fieldCache[fullyQualifiedName]
	}
	// get the field from the field tree
	for field := range a.FieldTree().All() {
		if field.FullyQualifiedName() == fullyQualifiedName {
			return field
		}
	}
	return nil
}

// DefaultAppearance returns the /DA of the form.
func (a *PDAcroForm) DefaultAppearance() string { return a.dictionary.GetString(cos.DA, "") }

// SetDefaultAppearance sets the /DA of the form.
func (a *PDAcroForm) SetDefaultAppearance(daValue string) {
	a.dictionary.SetString(cos.DA, daValue)
}

// NeedAppearances reports the /NeedAppearances flag.
func (a *PDAcroForm) NeedAppearances() bool {
	return a.dictionary.GetBoolean(cos.NeedAppearances, false)
}

// SetNeedAppearances sets the /NeedAppearances flag.
func (a *PDAcroForm) SetNeedAppearances(value bool) {
	a.dictionary.SetBoolean(cos.NeedAppearances, value)
}

// DefaultResources returns the /DR resources of the form, or nil.
func (a *PDAcroForm) DefaultResources() *pdmodel.PDResources {
	if dr := a.dictionary.GetCOSDictionary(cos.DR); dr != nil {
		return pdmodel.NewPDResourcesOfCacheAndFontCache(dr, a.document.ResourceCache(),
			a.directFontCache)
	}
	return nil
}

// SetDefaultResources sets the /DR resources of the form.
func (a *PDAcroForm) SetDefaultResources(dr *pdmodel.PDResources) {
	if dr == nil {
		a.dictionary.SetItem(cos.DR, nil)
		return
	}
	a.dictionary.SetItem(cos.DR, dr.COSObject())
}

// HasXFA reports whether the form carries an XFA form.
func (a *PDAcroForm) HasXFA() bool { return a.dictionary.ContainsKey(cos.XFA) }

// XFAIsDynamic reports whether the XFA form is the only form there is.
func (a *PDAcroForm) XFAIsDynamic() bool { return a.HasXFA() && len(a.Fields()) == 0 }

// XFA returns the XFA form, or nil.
func (a *PDAcroForm) XFA() *PDXFAResource {
	if base := a.dictionary.GetDictionaryObject(cos.XFA); base != nil {
		return NewPDXFAResource(base)
	}
	return nil
}

// SetXFA sets the XFA form.
func (a *PDAcroForm) SetXFA(xfa *PDXFAResource) {
	if xfa == nil {
		a.dictionary.SetItem(cos.XFA, nil)
		return
	}
	a.dictionary.SetItem(cos.XFA, xfa.COSObject())
}

// Q returns the /Q quadding of the form.
func (a *PDAcroForm) Q() int { return a.dictionary.GetIntDefault(cos.Q, 0) }

// SetQ sets the /Q quadding of the form.
func (a *PDAcroForm) SetQ(q int) { a.dictionary.SetInt(cos.Q, q) }

// IsSignaturesExist reports the signatures exist flag of /SigFlags.
func (a *PDAcroForm) IsSignaturesExist() bool {
	return a.dictionary.GetFlag(cos.SigFlags, flagSignaturesExist)
}

// SetSignaturesExist sets the signatures exist flag of /SigFlags.
func (a *PDAcroForm) SetSignaturesExist(signaturesExist bool) {
	a.dictionary.SetFlag(cos.SigFlags, flagSignaturesExist, signaturesExist)
}

// IsAppendOnly reports the append only flag of /SigFlags.
func (a *PDAcroForm) IsAppendOnly() bool {
	return a.dictionary.GetFlag(cos.SigFlags, flagAppendOnly)
}

// SetAppendOnly sets the append only flag of /SigFlags.
func (a *PDAcroForm) SetAppendOnly(appendOnly bool) {
	a.dictionary.SetFlag(cos.SigFlags, flagAppendOnly, appendOnly)
}

// ScriptingHandler returns the handler the JavaScript actions run through.
func (a *PDAcroForm) ScriptingHandler() ScriptingHandler { return a.scriptingHandler }

// SetScriptingHandler sets the handler the JavaScript actions run through.
func (a *PDAcroForm) SetScriptingHandler(scriptingHandler ScriptingHandler) {
	a.scriptingHandler = scriptingHandler
}

// CalcOrder returns the /CO fields, in the order they are calculated.
func (a *PDAcroForm) CalcOrder() []PDField {
	co := a.dictionary.GetCOSArray(cos.CO)
	if co == nil {
		return []PDField{}
	}
	fields := []PDField{}
	if a.IsCachingFields() {
		for _, field := range a.fieldCache {
			fields = append(fields, field)
		}
	} else {
		for field := range a.FieldTree().All() {
			fields = append(fields, field)
		}
	}
	actuals := []PDField{}
	for i := 0; i < co.Size(); i++ {
		item := co.GetObject(i)
		for _, field := range fields {
			if field.COSObject() == item {
				actuals = append(actuals, field)
				break
			}
		}
	}
	return actuals
}

// SetCalcOrder sets the /CO fields.
func (a *PDAcroForm) SetCalcOrder(fields []PDField) {
	array := cos.NewArray()
	for _, field := range fields {
		array.Add(field.COSObject())
	}
	a.dictionary.SetItem(cos.CO, array)
}

// resolveTransformationMatrix returns the matrix that maps an appearance into
// the rectangle of its annotation. Java declares it private.
func resolveTransformationMatrix(annot annotation.PDAnnotation,
	appearanceStream *annotation.PDAppearanceStream) *util.Matrix {
	// 1st step transform appearance stream bbox with appearance stream matrix
	transformedAppearanceBox := transformedAppearanceBBox(appearanceStream)
	annotationRect := annot.Rectangle()

	// 2nd step caclulate matrix to transform calculated rectangle into the annotation Rect boundaries
	transformationMatrix := util.NewMatrix()
	transformationMatrix.Translate(
		annotationRect.LowerLeftX()-float32(transformedAppearanceBox.X),
		annotationRect.LowerLeftY()-float32(transformedAppearanceBox.Y))
	transformationMatrix.Scale(
		annotationRect.Width()/float32(transformedAppearanceBox.Width),
		annotationRect.Height()/float32(transformedAppearanceBox.Height))
	return transformationMatrix
}

// transformedAppearanceBBox returns the bounding box of the appearance, through
// its own matrix. Java declares it private.
func transformedAppearanceBBox(appearanceStream *annotation.PDAppearanceStream) *geom.Rectangle2D {
	appearanceStreamMatrix := appearanceStream.Matrix()
	appearanceStreamBBox := appearanceStream.BBox()
	transformedAppearanceBox := appearanceStreamBBox.Transform(appearanceStreamMatrix)
	return transformedAppearanceBox.Bounds2D()
}

// buildPagesWidgetsMap maps each page onto the widgets of the given fields that
// sit on it. Java declares it private.
func (a *PDAcroForm) buildPagesWidgetsMap(fields []PDField,
	pages *pdmodel.PDPageTree) map[*cos.Dictionary]map[*cos.Dictionary]bool {
	pagesAnnotationsMap := map[*cos.Dictionary]map[*cos.Dictionary]bool{}
	hasMissingPageRef := false

	for _, field := range fields {
		for _, widget := range field.Widgets() {
			page, _ := widget.Page().(*pdmodel.PDPage)
			if page != nil {
				fillPagesAnnotationMap(pagesAnnotationsMap, page, widget)
			} else {
				slog.Warn("form: missing /P entry (page reference) in a widget for field",
					slog.String("field", field.FullyQualifiedName()))
				hasMissingPageRef = true
			}
		}
	}

	if !hasMissingPageRef {
		return pagesAnnotationsMap
	}

	// If there is a widget with a missing page reference we need to build the map reverse i.e.
	// from the annotations to the widget.
	slog.Warn("form: there has been a widget with a missing page reference, " +
		"will check all page annotations")

	widgetDictionarySet := createWidgetDictionarySet(fields)
	for page := range pages.All {
		for _, annot := range page.Annotations().ToSlice() {
			if widgetDictionarySet[annot.AnnotationDictionary()] {
				if widget, isWidget := annot.(*annotation.PDAnnotationWidget); isWidget {
					fillPagesAnnotationMap(pagesAnnotationsMap, page, widget)
				}
			}
		}
	}
	return pagesAnnotationsMap
}

// createWidgetDictionarySet gathers the widget dictionaries of the given
// fields. Java declares it private.
func createWidgetDictionarySet(fields []PDField) map[*cos.Dictionary]bool {
	widgetDictionarySet := map[*cos.Dictionary]bool{}
	for _, field := range fields {
		for _, widget := range field.Widgets() {
			widgetDictionarySet[widget.AnnotationDictionary()] = true
		}
	}
	return widgetDictionarySet
}

// fillPagesAnnotationMap records one widget against its page. Java declares it
// private.
func fillPagesAnnotationMap(pagesAnnotationsMap map[*cos.Dictionary]map[*cos.Dictionary]bool,
	page *pdmodel.PDPage, widget *annotation.PDAnnotationWidget) {
	widgetsForPage := pagesAnnotationsMap[page.Dictionary()]
	if widgetsForPage == nil {
		widgetsForPage = map[*cos.Dictionary]bool{}
		pagesAnnotationsMap[page.Dictionary()] = widgetsForPage
	}
	widgetsForPage[widget.AnnotationDictionary()] = true
}

// removeFields takes the given fields out of the form or of their parents. Java
// declares it private.
func (a *PDAcroForm) removeFields(fields []PDField) {
	for _, field := range fields {
		var array *cos.Array
		if field.Parent() == nil {
			// if the field has no parent, assume it is at root level list, remove it from there
			array = a.dictionary.GetCOSArray(cos.Fields)
		} else {
			// if the field has a parent, then remove from the list there
			array = field.Parent().FieldDictionary().GetCOSArray(cos.Kids)
		}
		array.RemoveObject(field.COSObject())
	}
}
