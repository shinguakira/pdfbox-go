package multipdf

// The half of Splitter that carries the structure tree, the annotations and
// the destinations across.
//
// Slice 7 ported the rest and left these seven methods and the KCloner inner
// class out, because "each of them works on pdmodel/interactive and
// pdmodel/documentinterchange/logicalstructure ... and slice 8 brings that
// whole subtree". Slice 8 and slice 9 are merged, so the deferral has stopped
// being true; `PDFMergerUtilityTest` has eight cases that split a tagged
// document and check what came with it, and none of them can be ported without
// this. See migration/STATUS.md.

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
)

// destinationToFix is one entry of Java's `Map<PDPageDestination, PDPage>`: a
// cloned destination, and the imported page whose link carries it.
//
// The port keeps a slice rather than a map because a Go map cannot be keyed on
// an interface whose implementations are not comparable, and nothing here needs
// to look an entry up.
type destinationToFix struct {
	destination destination.PageDestination
	sourcePage  *pdmodel.PDPage
}

// fixDestinations replaces the page destinations, where the source and
// destination pages are in the target document.
//
// "This must be called after all pages (and its annotations) are processed."
func (s *Splitter) fixDestinations(destinationDocument *pdmodel.PDDocument) {
	pageTree := destinationDocument.Pages()
	for _, entry := range s.destToFix {
		pageDestination := entry.destination
		// Find whether source page is inside or outside
		if pageTree.IndexOf(entry.sourcePage) < 0 {
			continue
		}
		srcPage, isPage := pageDestination.Page().(*pdmodel.PDPage)
		if !isPage {
			continue
		}
		dstPageDict := s.pageDictMap[srcPage.Dictionary()]
		if dstPageDict == nil {
			pageDestination.SetPage(nil)
			continue
		}
		dstPage := pdmodel.NewPDPageOf(dstPageDict)
		// Find whether destination page is inside or outside
		if pageTree.IndexOf(dstPage) >= 0 {
			pageDestination.SetPage(dstPage)
		} else {
			pageDestination.SetPage(nil)
		}
	}
}

// cloneStructureTree clones the structure tree from the source to the given
// destination document.
//
// "This must be called after all pages are processed."
func (s *Splitter) cloneStructureTree(destinationDocument *pdmodel.PDDocument) error {
	srcStructureTreeRoot := s.sourceDocument.DocumentCatalog().StructureTreeRoot()
	if srcStructureTreeRoot == nil {
		return nil
	}
	s.structDictMap = map[*cos.Dictionary]*cos.Dictionary{}
	dstStructureTreeRoot := logicalstructure.NewPDStructureTreeRoot()
	dstPageTree := destinationDocument.Pages()

	// clone /K, also fills dictMap
	k1 := srcStructureTreeRoot.K()
	cloner := &kCloner{splitter: s, dstPageTree: dstPageTree}
	k2 := cloner.createClone(k1, dstStructureTreeRoot.COSObject(), nil)
	dstStructureTreeRoot.SetK(k2)

	// transfer ParentTree using the map because the dictionaries are all found
	// in the /K structure.
	srcNumberTreeAsMap, err := GetNumberTreeAsMap(srcStructureTreeRoot.ParentTree())
	if err != nil {
		return err
	}
	dstNumberTreeAsMap := map[int]common.COSObjectable{}
	for p := 0; p < dstPageTree.Count(); p++ {
		page := dstPageTree.Get(p)
		if sp1 := page.StructParents(); sp1 != -1 {
			s.cloneTreeElement(srcNumberTreeAsMap, dstNumberTreeAsMap, sp1)
		}
		for ann := range page.Annotations().All {
			if sp2 := structParentOf(ann); sp2 != -1 {
				s.cloneTreeElement(srcNumberTreeAsMap, dstNumberTreeAsMap, sp2)
			}
			if normal := normalAppearanceStreamOf(ann); normal != nil {
				if err := s.processResources(resourcesOfAppearance(normal),
					srcNumberTreeAsMap, dstNumberTreeAsMap,
					map[*cos.Dictionary]bool{}); err != nil {
					return err
				}
			}
		}
		if err := s.processResources(page.Resources(), srcNumberTreeAsMap,
			dstNumberTreeAsMap, map[*cos.Dictionary]bool{}); err != nil {
			return err
		}
	}
	dstNumberTreeNode := common.NewPDNumberTreeNode(logicalstructure.ParentTreeValueConverter)
	dstNumberTreeNode.SetNumbers(dstNumberTreeAsMap)
	dstStructureTreeRoot.SetParentTree(dstNumberTreeNode)
	if upperLimit := dstNumberTreeNode.UpperLimit(); upperLimit != nil {
		dstStructureTreeRoot.SetParentTreeNextKey(*upperLimit + 1)
	}
	dstStructureTreeRoot.SetClassMap(srcStructureTreeRoot.ClassMap())
	s.cloneRoleMap(srcStructureTreeRoot, dstStructureTreeRoot)
	if err := s.cloneIDTree(srcStructureTreeRoot, dstStructureTreeRoot); err != nil {
		return err
	}

	destinationDocument.DocumentCatalog().SetStructureTreeRoot(dstStructureTreeRoot)
	return nil
}

// cloneIDTree carries over the /IDTree entries whose structure elements were
// cloned.
func (s *Splitter) cloneIDTree(srcStructTree, destStructTree *logicalstructure.PDStructureTreeRoot) error {
	srcIDTree := srcStructTree.IDTree()
	if srcIDTree == nil {
		return nil
	}
	srcIDTreeAsMap, err := GetIDTreeAsMap(srcIDTree)
	if err != nil {
		return err
	}
	destNames := map[string]*logicalstructure.PDStructureElement{}
	for key, val := range srcIDTreeAsMap {
		if !s.idSet[key] || val == nil {
			continue
		}
		if dstDict := s.structDictMap[val.COSObject().(*cos.Dictionary)]; dstDict != nil {
			destNames[key] = logicalstructure.NewPDStructureElementOf(dstDict)
		}
	}
	destIDTree := pdmodel.NewPDStructureElementNameTreeNode()
	destIDTree.SetNames(destNames)
	destStructTree.SetIDTree(destIDTree)
	// See comment at the end of PDFMergerUtility.mergeIDTree()
	return nil
}

// cloneRoleMap carries over the /RoleMap entries whose structure types were
// used.
//
// "needed because getRoleMap() and setRoleMap() habe different map types?!"
func (s *Splitter) cloneRoleMap(srcStructTree, destStructTree *logicalstructure.PDStructureTreeRoot) {
	srcDict := srcStructTree.COSObject().(*cos.Dictionary).GetCOSDictionary(cos.RoleMap)
	if srcDict == nil {
		return
	}
	dstDict := cos.NewDictionary()
	for _, key := range srcDict.KeySet() {
		if s.roleSet[key] {
			dstDict.SetItem(key, srcDict.GetItem(key))
		}
	}
	destStructTree.COSObject().(*cos.Dictionary).SetItem(cos.RoleMap, dstDict)
}

// cloneTreeElement clones a /ParentTree element using the map, so that
// structure elements are replaced.
func (s *Splitter) cloneTreeElement(srcNumberTreeAsMap, dstNumberTreeAsMap map[int]common.COSObjectable,
	sp int) {
	srcObj := srcNumberTreeAsMap[sp] // this is a PDParentTreeValue class
	if srcObj == nil {
		return
	}
	var dstObj cos.Base
	switch actualSrcObj := srcObj.COSObject().(type) {
	case *cos.Array:
		// structure element or array: create a clone of the array
		dstArray := cos.NewArray()
		for i := 0; i < actualSrcObj.Size(); i++ {
			srcElement, isDictionary := actualSrcObj.GetObject(i).(*cos.Dictionary)
			if !isDictionary {
				dstArray.Add(nil)
				continue
			}
			// may be null
			if cloned := s.structDictMap[srcElement]; cloned != nil {
				dstArray.Add(cloned)
			} else {
				dstArray.Add(nil)
			}
		}
		dstObj = dstArray
	case *cos.Dictionary:
		// get the clone from the map
		if cloned := s.structDictMap[actualSrcObj]; cloned != nil {
			dstObj = cloned
		} else {
			// 164421.pdf, structure tree is weird.
			// also 250052.pdf, 250198.pdf, 257012.pdf, 271459.pdf (multiple),
			// 670045.pdf (multiple)
			// In 71459.pdf annotations on page 1 have StructParent numbers
			// that point to structure elements in the /ParentTree that point to
			// a different page.
			slog.Warn("multipdf: ParentTree index dictionary not found in /K", "index", sp)
		}
	default:
		slog.Warn("multipdf: tree element neither dictionary nor array",
			"type", typeNameOf(srcObj.COSObject()))
	}
	if dstObj != nil {
		converted, err := logicalstructure.ParentTreeValueConverter(dstObj)
		if err != nil {
			slog.Warn("multipdf: ParentTree element could not be converted", "index", sp)
			return
		}
		dstNumberTreeAsMap[sp] = converted
	}
}

// processResources looks for /StructParent and /StructParents and adds them to
// the destination tree.
func (s *Splitter) processResources(res *pdmodel.PDResources,
	srcNumberTreeAsMap, dstNumberTreeAsMap map[int]common.COSObjectable,
	visited map[*cos.Dictionary]bool) error {
	if res == nil {
		return nil
	}
	if visited[res.Dictionary()] {
		// avoid endless recursion, e.g. with 002874.pdf
		return nil
	}
	visited[res.Dictionary()] = true

	for _, name := range res.XObjectNames() {
		xObject, err := res.GetXObject(name)
		if err != nil {
			return err
		}
		sp2 := -1
		switch object := xObject.(type) {
		case *form.PDFormXObject:
			sp2 = object.StructParents()
			if err := s.processResources(resourcesOfForm(object), srcNumberTreeAsMap,
				dstNumberTreeAsMap, visited); err != nil {
				return err
			}
		case *image.PDImageXObject:
			sp2 = object.StructParent()
		}
		if sp2 != -1 {
			s.cloneTreeElement(srcNumberTreeAsMap, dstNumberTreeAsMap, sp2)
		}
	}
	return nil
}

// resourcesOfForm narrows a form's resources, which the port declares as an
// interface so that graphics/form does not import pdmodel.
func resourcesOfForm(object *form.PDFormXObject) *pdmodel.PDResources {
	resources, isResources := object.Resources().(*pdmodel.PDResources)
	if !isResources {
		return nil
	}
	return resources
}

// structParentOf is `ann.getStructParent()`, which PDAnnotationBase has and the
// port's PDAnnotation interface does not.
func structParentOf(ann annotation.PDAnnotation) int {
	return ann.COSObject().(*cos.Dictionary).GetIntDefault(cos.StructParent, -1)
}

// normalAppearanceStreamOf is `ann.getNormalAppearanceStream()`.
func normalAppearanceStreamOf(ann annotation.PDAnnotation) *annotation.PDAppearanceStream {
	return ann.NormalAppearanceStream()
}

// typeNameOf is getClass().getSimpleName() for a log line.
func typeNameOf(base cos.Base) string {
	if base == nil {
		return "(null)"
	}
	return simpleName(base)
}

// resourcesOfAppearance narrows an appearance stream's resources, the way
// resourcesOfForm narrows a form's.
func resourcesOfAppearance(stream *annotation.PDAppearanceStream) *pdmodel.PDResources {
	resources, isResources := stream.Resources().(*pdmodel.PDResources)
	if !isResources {
		return nil
	}
	return resources
}
