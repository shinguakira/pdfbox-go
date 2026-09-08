package multipdf

// The /K tree cloner, and the annotation cloning that fills the maps it reads.
//
// Port of Splitter's private inner class KCloner and of processAnnotations.

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
)

// kCloner helps clone the /K tree.
//
// "It clones structure elements and fills the structure elements map. Pages are
// replaced with the help of the page map. Elements with pages that don't belong
// to the destination are removed from the clone."
//
// Java's is an inner class and reaches the splitter's fields directly; the port
// holds it.
type kCloner struct {
	splitter    *Splitter
	dstPageTree *pdmodel.PDPageTree
}

// createClone creates a clone of the source.
//
// `src` is a source dictionary or array; `dstParent` is for the /P entry, and
// is a parameter "because arrays don't keep a parent"; `currentPageDict` is
// "used to remember whether we have a page parent somewhere or not. Starts with
// null."
//
// It answers nil "if source is null or if there is no clone because it belongs
// to a different page or to no page."
func (c *kCloner) createClone(src cos.Base, dstParent cos.Base,
	currentPageDict *cos.Dictionary) cos.Base {
	switch value := src.(type) {
	case *cos.Array:
		return c.createArrayClone(value, dstParent, currentPageDict)
	case *cos.Dictionary:
		return c.createDictionaryClone(value, dstParent, currentPageDict)
	default:
		return src
	}
}

func (c *kCloner) createArrayClone(src *cos.Array, dstParent cos.Base,
	currentPageDict *cos.Dictionary) cos.Base {
	dst := cos.NewArray()
	for i := 0; i < src.Size(); i++ {
		base2 := src.Get(i)
		var rc cos.Base
		if reference, isReference := base2.(*cos.Object); isReference {
			rc = c.createClone(reference.Object(), dstParent, currentPageDict)
		} else {
			rc = c.createClone(base2, dstParent, currentPageDict)
		}
		// if this is null then they don't belong to the destination document
		if rc != nil {
			dst.Add(rc)
		}
	}
	if dst.Size() == 0 {
		return nil
	}
	return dst
}

func (c *kCloner) createDictionaryClone(srcDict *cos.Dictionary, dstParent cos.Base,
	currentPageDict *cos.Dictionary) cos.Base {
	if dstDict := c.splitter.structDictMap[srcDict]; dstDict != nil {
		return dstDict
	}
	srcPageDict := srcDict.GetCOSDictionary(cos.Pg)
	var dstPageDict *cos.Dictionary
	kid := srcDict.GetDictionaryObject(cos.K)
	kind := srcDict.GetCOSName(cos.Type)
	if srcPageDict != nil {
		dstPageDict = c.splitter.pageDictMap[srcPageDict]
		if dstPageDict != nil {
			if c.dstPageTree.IndexOf(pdmodel.NewPDPageOf(dstPageDict)) == -1 {
				return nil
			}
		} else {
			// PDFBOX-6009: "wrong" /Pg entry
			// quit if MCIDs because these need a /Pg entry
			// or if MCR/OBJR dicts
			if kind == cos.MCR || kind == cos.OBJR || hasMCIDs(kid) {
				return nil
			}
			// else keep this as an intermediate element for now
		}
	}

	// special handling for MCR items ("marked-content reference dictionary")
	if kind == cos.MCR && dstPageDict == nil && parentHasNoPage(dstParent) {
		// PAC: Pg entry of marked-content reference and of parent structure is null
		return nil
	}

	// Create and fill clone
	dstDict := cos.NewDictionary()
	c.splitter.structDictMap[srcDict] = dstDict
	for _, key := range srcDict.KeySet() {
		if key != cos.K && key != cos.Pg && key != cos.P {
			dstDict.SetItem(key, srcDict.GetItem(key))
		}
	}

	// special handling for OBJR items ("object reference dictionary")
	// see e.g. file 488300.pdf and Root/StructTreeRoot/K/K/[2]/K/[1]/K/[0]/Obj
	if kind == cos.OBJR {
		srcObj := srcDict.GetCOSDictionary(cos.Obj)
		dstObj := c.splitter.annotDictMap[srcObj]
		if dstObj != nil {
			// replace annotation with clone
			dstDict.SetItem(cos.Obj, dstObj)
		} else if srcObj != nil { // 079177.pdf
			removePossibleOrphanAnnotation(srcObj, srcDict, currentPageDict, dstDict)
		}
		if dstDict.Size() == 1 {
			return nil
		}
		if dstPageDict == nil && parentHasNoPage(dstParent) {
			// Pg entry of object reference dictionary and of parent structure is null
			return nil
		}
	}

	if kind != cos.OBJR && kind != cos.MCR {
		// /P not needed for OBJR or MCR items
		dstDict.SetItem(cos.P, dstParent)
	}

	if dstPageDict != nil {
		dstDict.SetItem(cos.Pg, dstPageDict)
	} else {
		// setItem with a null value removes the entry, which is what Java's
		// `dstDict.setItem(COSName.PG, dstPageDict)` does for a null page.
		dstDict.RemoveItem(cos.Pg)
	}

	// stack overflow here with 207658.pdf and 113484.pdf, too complex; works with -Xss50m
	nextPageDict := currentPageDict
	if dstPageDict != nil {
		nextPageDict = dstPageDict
	}
	cloneKid := c.createClone(kid, dstDict, nextPageDict)
	if cloneKid == nil && kid != nil {
		return nil // kids array wasn't empty, but is empty now => ignore
	}

	// removes orphan nodes, example:
	// Root/StructTreeRoot/K/[7]/K/[3]/K/[5]/K/[2] in 271459.pdf
	// decide about keeping source dictionaries with no /K and no /PG
	if dstPageDict == nil && cloneKid == nil && currentPageDict == nil {
		// if no parent page and no page here and no kids, assume this is an orphan
		return nil
	}
	if cloneKid != nil {
		dstDict.SetItem(cos.K, cloneKid)
	} else {
		dstDict.RemoveItem(cos.K)
	}
	if id := dstDict.GetString(cos.ID, ""); id != "" {
		c.splitter.idSet[id] = true
	}
	if s := dstDict.GetCOSName(cos.S); s != nil {
		c.splitter.roleSet[s] = true
	}
	return dstDict
}

// parentHasNoPage is Java's `dstParent instanceof COSDictionary &&
// ((COSDictionary) dstParent).getCOSDictionary(COSName.PG) == null`.
func parentHasNoPage(dstParent cos.Base) bool {
	parent, isDictionary := dstParent.(*cos.Dictionary)
	return isDictionary && parent.GetCOSDictionary(cos.Pg) == nil
}

func hasMCIDs(kid cos.Base) bool {
	if _, isInteger := kid.(*cos.Integer); isInteger {
		return true
	}
	if array, isArray := kid.(*cos.Array); isArray {
		for i := 0; i < array.Size(); i++ {
			if _, isInteger := array.GetObject(i).(*cos.Integer); isInteger {
				return true
			}
		}
	}
	return false
}

// removePossibleOrphanAnnotation is PDFBOX-5929: "Check whether this is an
// 'orphan' annotation that isn't in the page".
func removePossibleOrphanAnnotation(srcObj, srcDict, currentPageDict, dstDict *cos.Dictionary) {
	objType := srcObj.GetDictionaryObject(cos.Type)
	objSubtype := srcObj.GetDictionaryObject(cos.Subtype)
	if objType != cos.Base(cos.Annot) && objSubtype != cos.Base(cos.Link) {
		return
	}
	srcPageDict := srcDict.GetCOSDictionary(cos.Pg)
	if srcPageDict == nil {
		// /Pg entry is not always on this level
		srcPageDict = currentPageDict
	}
	if srcPageDict == nil {
		return
	}
	annotationArray := srcPageDict.GetCOSArray(cos.Annots)
	if annotationArray == nil || annotationArray.IndexOfObject(srcObj) == -1 {
		// Ideally the entire OBJR entry should be removed.
		// Removing the OBJ entry is done to avoid potential page orphans
		// from the annotation destination.
		slog.Warn("multipdf: An annotation OBJ that isn't in the page has been " +
			"removed from the structure tree")
		dstDict.RemoveItem(cos.Obj)
	}
}

// processAnnotations clones all annotations "because of changes possibly made,
// and because the structure tree is cloned".
func (s *Splitter) processAnnotations(imported *pdmodel.PDPage) error {
	annotations := imported.Annotations().ToSlice()
	if len(annotations) == 0 {
		return nil
	}
	clonedAnnotations := make([]annotation.PDAnnotation, 0, len(annotations))
	for _, ann := range annotations {
		// create a shallow clone
		clonedDict := cos.NewDictionaryFrom(ann.COSObject().(*cos.Dictionary))
		annotationClone, err := annotation.CreateAnnotation(clonedDict)
		if err != nil {
			return err
		}
		s.annotDictMap[ann.COSObject().(*cos.Dictionary)] = clonedDict
		clonedAnnotations = append(clonedAnnotations, annotationClone)

		if link, isLink := annotationClone.(*annotation.PDAnnotationLink); isLink {
			if err := s.fixLinkDestination(link, imported); err != nil {
				return err
			}
		}
		if _, isWidget := annotationClone.(*annotation.PDAnnotationWidget); isWidget &&
			clonedDict.ContainsKey(cos.Parent) {
			// remove non-terminal field /Parent reference, because this may
			// lead to orphan pages
			clonedDict.RemoveItem(cos.Parent)
		}
		if pageOf(ann) != nil {
			setPageOf(annotationClone, imported)
		}
	}
	// Second loop for markup and popup annotations, which reference annotations themselves
	for _, ann := range clonedAnnotations {
		s.fixMarkupAndPopup(ann, imported)
	}
	imported.SetAnnotations(clonedAnnotations)
	return nil
}

// fixLinkDestination is the body of Java's `if (annotationClone instanceof
// PDAnnotationLink)`: read the destination however it is reached, clone it, and
// remember it for fixDestinations.
func (s *Splitter) fixLinkDestination(link *annotation.PDAnnotationLink,
	imported *pdmodel.PDPage) error {
	srcDestination, err := link.Destination()
	if err != nil {
		slog.Warn("multipdf: Incorrect destination in link annotation is removed",
			"page", s.currentPageNumber+1, "error", err)
		link.SetDestination(nil)
		srcDestination = nil
	}
	var goToAction *action.PDActionGoTo
	if srcDestination == nil {
		if goTo, isGoTo := link.Action().(*action.PDActionGoTo); isGoTo {
			goToAction = goTo
			srcDestination, err = goTo.Destination()
			if err != nil {
				slog.Warn("multipdf: GoToAction with incorrect destination in link "+
					"annotation is removed", "page", s.currentPageNumber+1, "error", err)
				link.SetAction(nil)
				srcDestination = nil
			}
		}
	}
	if named, isNamed := srcDestination.(*destination.PDNamedDestination); isNamed {
		// we do not use the named destination anymore because names get
		// modified, e.g. 0xAD becomes 0, see file 410609.pdf where the name no
		// longer matches with the entry in the new name tree; plus the original
		// solution was 40 additional loc
		found, err := s.sourceDocument.DocumentCatalog().FindNamedDestinationPage(named)
		if err != nil {
			return err
		}
		srcDestination = nil
		if found != nil {
			srcDestination = found
		}
	}
	pageDestination, isPageDestination := srcDestination.(destination.PageDestination)
	if !isPageDestination {
		return nil
	}
	// preserve links to pages within the split result:
	// not fully possible here because we don't have the full target document
	// yet. However we're cloning as needed and remember what to do later.
	if pageDestination.Page() == nil {
		return nil
	}
	// clone destination
	clonedDestinationArray := cos.NewArrayOf(
		pageDestination.COSObject().(*cos.Array).ToList())
	cloned, err := destination.Create(clonedDestinationArray)
	if err != nil {
		return err
	}
	dstDestination, isPageDestination := cloned.(destination.PageDestination)
	if !isPageDestination {
		return nil
	}

	// remember the destination to adjust / remove page later
	s.destToFix = append(s.destToFix, destinationToFix{dstDestination, imported})

	if goToAction != nil {
		// if action is not null, then the destination came from an action,
		// thus clone action as well, then assign destination clone, then action
		clonedActionDict := cos.NewDictionaryFrom(goToAction.COSObject().(*cos.Dictionary))
		dstAction, isGoTo := action.CreateAction(clonedActionDict).(*action.PDActionGoTo)
		if !isGoTo {
			return nil
		}
		dstAction.SetDestination(dstDestination)
		link.SetAction(dstAction)
		return nil
	}
	// just assign destination clone
	link.SetDestination(dstDestination)
	return nil
}

// fixMarkupAndPopup is Java's second loop, which points a markup annotation and
// its popup at each other's clones.
func (s *Splitter) fixMarkupAndPopup(ann annotation.PDAnnotation, imported *pdmodel.PDPage) {
	if markup, isMarkup := asMarkup(ann); isMarkup {
		annotationPopup := markup.Popup()
		if annotationPopup != nil {
			s.fixPopupOfMarkup(markup, annotationPopup, imported)
		}
	}
	if popup, isPopup := ann.(*annotation.PDAnnotationPopup); isPopup {
		annotationMarkup := popup.Parent()
		if annotationMarkup == nil {
			return
		}
		// clonedMarkupDict will be null if markup annotation is an orphan (not
		// in annotation list)
		clonedMarkupDict := s.annotDictMap[annotationMarkup.COSObject().(*cos.Dictionary)]
		if clonedMarkupDict != nil {
			popup.COSObject().(*cos.Dictionary).SetItem(cos.Parent, clonedMarkupDict)
		} else {
			popup.COSObject().(*cos.Dictionary).RemoveItem(cos.Parent)
		}
	}
}

func (s *Splitter) fixPopupOfMarkup(markup *annotation.PDAnnotationMarkup,
	annotationPopup *annotation.PDAnnotationPopup, imported *pdmodel.PDPage) {
	clonedPopupDict := s.annotDictMap[annotationPopup.COSObject().(*cos.Dictionary)]
	if clonedPopupDict != nil {
		markup.COSObject().(*cos.Dictionary).SetItem(cos.Popup, clonedPopupDict)
		return
	}
	// orphan popup (not in annotation list); clone it and fix references
	clonedPopupDict = cos.NewDictionaryFrom(annotationPopup.COSObject().(*cos.Dictionary))
	s.annotDictMap[annotationPopup.COSObject().(*cos.Dictionary)] = clonedPopupDict
	annotationPopupClone := annotation.NewPDAnnotationPopupOf(clonedPopupDict)
	annotationPopupClone.SetParent(markup)
	markup.SetPopup(annotationPopupClone)
	if annotationPopupClone.Page() != nil {
		annotationPopupClone.SetPage(imported)
	}
}

// asMarkup narrows an annotation to a markup one.
//
// Java's `instanceof PDAnnotationMarkup` catches every subclass; the port has
// them embed *PDAnnotationMarkup, so the assertion is on the accessors that
// carry the popup rather than on the type.
func asMarkup(ann annotation.PDAnnotation) (*annotation.PDAnnotationMarkup, bool) {
	if markup, isMarkup := ann.(*annotation.PDAnnotationMarkup); isMarkup {
		return markup, true
	}
	if embedder, embeds := ann.(interface {
		MarkupAnnotation() *annotation.PDAnnotationMarkup
	}); embeds {
		return embedder.MarkupAnnotation(), true
	}
	return nil, false
}

// pageOf and setPageOf are `getPage()` and `setPage()`, which
// PDAnnotationBase has and the port's PDAnnotation interface does not.
func pageOf(ann annotation.PDAnnotation) annotation.PageLike {
	pager, hasPage := ann.(interface{ Page() annotation.PageLike })
	if !hasPage {
		return nil
	}
	return pager.Page()
}

func setPageOf(ann annotation.PDAnnotation, page annotation.PageLike) {
	if pager, hasPage := ann.(interface{ SetPage(annotation.PageLike) }); hasPage {
		pager.SetPage(page)
	}
}
