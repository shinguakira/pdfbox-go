package pdmodel

// The rest of PDDocumentCatalog: everything that names a type from
// pdmodel/interactive, pdmodel/documentinterchange or
// graphics/optionalcontent. The catalogue itself, its two constructors, the
// page tree, the version and the AcroForm hooks are in pddocument.go, next to
// PDDocument.
//
// getAcroForm and setAcroForm are not here and cannot be: they name PDAcroForm,
// which lives in interactive/form, and that package imports this one. They are
// form.AcroFormOfCatalog, form.AcroFormOfCatalogFixup and
// form.SetAcroFormOfCatalog.

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/outline"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/pagenavigation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/viewerpreferences"
)

// ViewerPreferences returns the viewer preferences associated with this
// document, or nil where they do not exist.
func (c *PDDocumentCatalog) ViewerPreferences() *viewerpreferences.PDViewerPreferences {
	viewerPref := c.root.GetCOSDictionary(cos.ViewerPreferences)
	if viewerPref == nil {
		return nil
	}
	return viewerpreferences.NewPDViewerPreferencesOf(viewerPref)
}

// SetViewerPreferences sets the viewer preferences.
func (c *PDDocumentCatalog) SetViewerPreferences(prefs *viewerpreferences.PDViewerPreferences) {
	c.root.SetItem(cos.ViewerPreferences, common.COSObjectOrNil(prefs))
}

// DocumentOutline returns the outline associated with this document, or nil
// where it does not exist.
func (c *PDDocumentCatalog) DocumentOutline() *outline.PDDocumentOutline {
	outlineDict := c.root.GetCOSDictionary(cos.Outlines)
	if outlineDict == nil {
		return nil
	}
	return outline.NewPDDocumentOutlineOf(outlineDict)
}

// SetDocumentOutline sets the document outlines.
func (c *PDDocumentCatalog) SetDocumentOutline(outlines *outline.PDDocumentOutline) {
	c.root.SetItem(cos.Outlines, common.COSObjectOrNil(outlines))
}

// Threads returns the document's article threads.
//
// Java builds an empty /Threads array into the catalogue where it has none, so
// reading the threads of a document without any writes the entry.
func (c *PDDocumentCatalog) Threads() *common.COSArrayList[*pagenavigation.PDThread] {
	array := c.root.GetCOSArray(cos.Threads)
	if array == nil {
		array = cos.NewArray()
		array.SetDirect(false)
		c.root.SetItem(cos.Threads, array)
	}
	pdObjects := make([]*pagenavigation.PDThread, 0, array.Size())
	for i := 0; i < array.Size(); i++ {
		// Java casts the entry to COSDictionary without checking, so an array
		// holding anything else throws ClassCastException; the port's type
		// assertion panics, which is that cast.
		pdObjects = append(pdObjects,
			pagenavigation.NewPDThreadOf(array.GetObject(i).(*cos.Dictionary)))
	}
	return common.NewCOSArrayListOf(pdObjects, array)
}

// SetThreads sets the list of threads for this pdf document, and removes them
// for a nil list.
func (c *PDDocumentCatalog) SetThreads(threads []*pagenavigation.PDThread) {
	if threads == nil {
		c.root.RemoveItem(cos.Threads)
		return
	}
	threadsArray := common.NewCOSArrayOfObjectables(threads)
	threadsArray.SetDirect(false)
	c.root.SetItem(cos.Threads, threadsArray)
}

// Metadata returns the metadata that is part of the document catalog, or nil
// where there is none.
func (c *PDDocumentCatalog) Metadata() *common.PDMetadata {
	metaObj := c.root.GetCOSStream(cos.Metadata)
	if metaObj == nil {
		return nil
	}
	return common.NewPDMetadataOfStream(metaObj)
}

// SetMetadata sets the metadata for this object, which may be nil.
func (c *PDDocumentCatalog) SetMetadata(meta *common.PDMetadata) {
	c.root.SetItem(cos.Metadata, common.COSObjectOrNil(meta))
}

// SetOpenAction sets the Document Open Action for this object.
func (c *PDDocumentCatalog) SetOpenAction(openAction common.PDDestinationOrAction) {
	c.root.SetItem(cos.OpenAction, common.COSObjectOrNil(openAction))
}

// OpenAction returns the action to perform when the document is opened, or nil
// where there is none.
func (c *PDDocumentCatalog) OpenAction() (common.PDDestinationOrAction, error) {
	switch openAction := c.root.GetDictionaryObject(cos.OpenAction).(type) {
	case *cos.Dictionary:
		return action.CreateAction(openAction), nil
	case *cos.Array:
		return destination.Create(openAction)
	default:
		return nil, nil
	}
}

// Actions returns the additional actions for this document.
//
// Java adds the dictionary to the catalogue when it is absent, so reading the
// actions of a document that has none writes an empty /AA to it. The port
// carries that, the same way PDPage.Actions does.
func (c *PDDocumentCatalog) Actions() *action.PDDocumentCatalogAdditionalActions {
	addAction := c.root.GetCOSDictionary(cos.AA)
	if addAction == nil {
		addAction = cos.NewDictionary()
		c.root.SetItem(cos.AA, addAction)
	}
	return action.NewPDDocumentCatalogAdditionalActionsOf(addAction)
}

// SetActions sets the additional actions for the document.
func (c *PDDocumentCatalog) SetActions(actions *action.PDDocumentCatalogAdditionalActions) {
	c.root.SetItem(cos.AA, common.COSObjectOrNil(actions))
}

// Names returns the names dictionary for this document, or nil where none
// exists.
func (c *PDDocumentCatalog) Names() *PDDocumentNameDictionary {
	names := c.root.GetCOSDictionary(cos.Names)
	if names == nil {
		return nil
	}
	return NewPDDocumentNameDictionaryOf(c, names)
}

// Dests returns the named destinations dictionary for this document, or nil
// where none exists.
func (c *PDDocumentCatalog) Dests() *PDDocumentNameDestinationDictionary {
	dests := c.root.GetCOSDictionary(cos.Dests)
	if dests == nil {
		return nil
	}
	return NewPDDocumentNameDestinationDictionary(dests)
}

// FindNamedDestinationPage finds the page destination a named destination
// stands for, or nil where it is not found.
func (c *PDDocumentCatalog) FindNamedDestinationPage(
	namedDest *destination.PDNamedDestination) (destination.PageDestination, error) {
	var pageDestination destination.PageDestination
	namesDict := c.Names()
	if namesDict != nil {
		destsTree := namesDict.Dests()
		if destsTree != nil {
			value, err := destsTree.Value(namedDest.NamedDestination())
			if err != nil {
				return nil, err
			}
			pageDestination = value
		}
	}
	if pageDestination == nil {
		// Look up /Dests dictionary from catalog
		nameDestDict := c.Dests()
		if nameDestDict != nil {
			name := namedDest.NamedDestination()
			found, err := nameDestDict.Destination(name)
			if err != nil {
				return nil, err
			}
			if found == nil {
				return nil, nil
			}
			// Java casts to PDPageDestination without checking, so a /Dests
			// entry that is a named destination throws ClassCastException; the
			// port's type assertion panics, which is that cast.
			pageDestination = found.(destination.PageDestination)
		}
	}
	return pageDestination, nil
}

// SetNames sets the names dictionary for the document.
func (c *PDDocumentCatalog) SetNames(names *PDDocumentNameDictionary) {
	c.root.SetItem(cos.Names, common.COSObjectOrNil(names))
}

// MarkInfo returns info about the document's usage of tagged features, or nil
// where there is no information.
func (c *PDDocumentCatalog) MarkInfo() *logicalstructure.PDMarkInfo {
	dic := c.root.GetCOSDictionary(cos.MarkInfo)
	if dic == nil {
		return nil
	}
	return logicalstructure.NewPDMarkInfoOf(dic)
}

// SetMarkInfo sets information about the document's usage of tagged features.
func (c *PDDocumentCatalog) SetMarkInfo(markInfo *logicalstructure.PDMarkInfo) {
	c.root.SetItem(cos.MarkInfo, common.COSObjectOrNil(markInfo))
}

// OutputIntents returns the list of output intents defined in the document,
// never nil.
func (c *PDDocumentCatalog) OutputIntents() []*color.PDOutputIntent {
	array := c.root.GetCOSArray(cos.OutputIntents)
	if array == nil {
		return []*color.PDOutputIntent{}
	}
	retval := make([]*color.PDOutputIntent, 0, array.Size())
	for _, cosBase := range array.ToList() {
		if object, isObject := cosBase.(*cos.Object); isObject {
			cosBase = object.Object()
		}
		// Java casts the entry to COSDictionary without checking; the port's
		// type assertion panics, which is that cast.
		retval = append(retval, color.NewPDOutputIntent(cosBase.(*cos.Dictionary)))
	}
	return retval
}

// AddOutputIntent adds an output intent to the list. If there is no output
// intent, the list is created and the first element added.
func (c *PDDocumentCatalog) AddOutputIntent(outputIntent *color.PDOutputIntent) {
	array := c.root.GetCOSArray(cos.OutputIntents)
	if array == nil {
		array = cos.NewArray()
		c.root.SetItem(cos.OutputIntents, array)
	}
	array.Add(outputIntent.COSObject())
}

// SetOutputIntents replaces the list of output intents of the document; an
// empty list removes them all.
func (c *PDDocumentCatalog) SetOutputIntents(outputIntents []*color.PDOutputIntent) {
	array := cos.NewArray()
	for _, intent := range outputIntents {
		array.Add(intent.COSObject())
	}
	c.root.SetItem(cos.OutputIntents, array)
}

// PageMode returns the page display mode, PageModeUseNone where it is not
// present or is not one of the modes.
func (c *PDDocumentCatalog) PageMode() PageMode {
	mode := c.root.GetNameAsString(cos.PageMode, "")
	if mode == "" {
		// Java tests the name for null; a /PageMode of the empty name reaches
		// fromString there and is rejected, which lands on the same answer.
		return PageModeUseNone
	}
	pageMode, err := PageModeFromString(mode)
	if err != nil {
		// LOG.debug("Invalid PageMode used ... - setting to PageMode.USE_NONE")
		return PageModeUseNone
	}
	return pageMode
}

// SetPageMode sets the page mode.
func (c *PDDocumentCatalog) SetPageMode(mode PageMode) {
	c.root.SetName(cos.PageMode, mode.StringValue())
}

// PageLayout returns the page layout, PageLayoutSinglePage where it is not
// present or is not one of the layouts.
func (c *PDDocumentCatalog) PageLayout() PageLayout {
	mode := c.root.GetNameAsString(cos.PageLayout, "")
	if mode != "" {
		if pageLayout, err := PageLayoutFromString(mode); err == nil {
			return pageLayout
		}
		// LOG.warn("Invalid PageLayout used ... - returning PageLayout.SINGLE_PAGE")
	}
	return PageLayoutSinglePage
}

// SetPageLayout sets the page layout.
func (c *PDDocumentCatalog) SetPageLayout(layout PageLayout) {
	c.root.SetName(cos.PageLayout, layout.StringValue())
}

// URI returns the document-level URI, or nil where there is none.
func (c *PDDocumentCatalog) URI() *action.PDURIDictionary {
	uri := c.root.GetCOSDictionary(cos.URI)
	if uri == nil {
		return nil
	}
	return action.NewPDURIDictionaryOf(uri)
}

// SetURI sets the document level URI.
func (c *PDDocumentCatalog) SetURI(uri *action.PDURIDictionary) {
	c.root.SetItem(cos.URI, common.COSObjectOrNil(uri))
}

// StructureTreeRoot returns the document's structure tree root, or nil where
// none exists.
func (c *PDDocumentCatalog) StructureTreeRoot() *logicalstructure.PDStructureTreeRoot {
	dict := c.root.GetCOSDictionary(cos.StructTreeRoot)
	if dict == nil {
		return nil
	}
	return logicalstructure.NewPDStructureTreeRootOf(dict)
}

// SetStructureTreeRoot sets the document's structure tree root.
func (c *PDDocumentCatalog) SetStructureTreeRoot(treeRoot *logicalstructure.PDStructureTreeRoot) {
	c.root.SetItem(cos.StructTreeRoot, common.COSObjectOrNil(treeRoot))
}

// Language returns the language for the document, or "" where there is none.
func (c *PDDocumentCatalog) Language() string {
	return c.root.GetString(cos.Lang, "")
}

// SetLanguage sets the language for the document.
func (c *PDDocumentCatalog) SetLanguage(language string) {
	c.root.SetString(cos.Lang, language)
}

// PageLabels returns the page labels descriptor of the document, or nil where
// there is none.
func (c *PDDocumentCatalog) PageLabels() (*common.PDPageLabels, error) {
	dict := c.root.GetCOSDictionary(cos.PageLabels)
	if dict == nil {
		return nil, nil
	}
	return common.NewPDPageLabelsOf(c.document, dict)
}

// SetPageLabels sets the page label descriptor for the document.
func (c *PDDocumentCatalog) SetPageLabels(labels *common.PDPageLabels) {
	c.root.SetItem(cos.PageLabels, common.COSObjectOrNil(labels))
}

// OCProperties returns the optional content properties dictionary associated
// with this document, or nil where it is not present.
func (c *PDDocumentCatalog) OCProperties() *optionalcontent.PDOptionalContentProperties {
	dict := c.root.GetCOSDictionary(cos.OCProperties)
	if dict == nil {
		return nil
	}
	return optionalcontent.NewPDOptionalContentPropertiesOf(dict)
}

// SetOCProperties sets the optional content properties dictionary. The document
// version is incremented to 1.5 if lower.
func (c *PDDocumentCatalog) SetOCProperties(
	ocProperties *optionalcontent.PDOptionalContentProperties) {
	c.root.SetItem(cos.OCProperties, common.COSObjectOrNil(ocProperties))

	// optional content groups require PDF 1.5
	if ocProperties != nil && c.document.Version() < 1.5 {
		c.document.SetVersion(1.5)
	}
}
