package multipdf

// The half of PDFMergerUtility that moves the pages and the logical structure
// hierarchy across, and the small helpers the whole class shares.

import (
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/viewerpreferences"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// mergePagesAndStructureTree is the tail of appendDocument: work out whether
// the structure trees can be merged, move the pages across, and then move the
// tree.
func (m *PDFMergerUtility) mergePagesAndStructureTree(cloner *PDFCloneUtility,
	destinationDoc *pdmodel.PDDocument,
	destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) error {
	// merge logical structure hierarchy
	mergeStructTree := false
	destParentTreeNextKey := -1
	var srcNumberTreeAsMap map[int]common.COSObjectable
	var destNumberTreeAsMap map[int]common.COSObjectable
	srcStructTree := srcCatalog.StructureTreeRoot()
	destStructTree := destCatalog.StructureTreeRoot()

	if destStructTree == nil && srcStructTree != nil {
		// create a dummy structure tree in the destination, so that the source
		// tree is cloned. (We can't just copy the tree reference due to PDFBOX-3999)
		destStructTree = logicalstructure.NewPDStructureTreeRoot()
		destCatalog.SetStructureTreeRoot(destStructTree)
		destStructTree.SetParentTree(
			common.NewPDNumberTreeNode(logicalstructure.ParentTreeValueConverter))
		// PDFBOX-4429: remove bogus StructParent(s)
		for page := range destCatalog.Pages().All {
			page.Dictionary().RemoveItem(cos.StructParents)
			for ann := range page.Annotations().All {
				ann.COSObject().(*cos.Dictionary).RemoveItem(cos.StructParent)
			}
		}
	}

	if destStructTree != nil {
		destParentTree := destStructTree.ParentTree()
		destParentTreeNextKey = destStructTree.ParentTreeNextKey()
		if destParentTree != nil {
			var err error
			destNumberTreeAsMap, err = GetNumberTreeAsMap(destParentTree)
			if err != nil {
				return err
			}
			if destParentTreeNextKey < 0 {
				if len(destNumberTreeAsMap) == 0 {
					destParentTreeNextKey = 0
				} else {
					destParentTreeNextKey = maxKey(destNumberTreeAsMap) + 1
				}
			}
			if destParentTreeNextKey >= 0 && srcStructTree != nil {
				if srcParentTree := srcStructTree.ParentTree(); srcParentTree != nil {
					srcNumberTreeAsMap, err = GetNumberTreeAsMap(srcParentTree)
					if err != nil {
						return err
					}
					if len(srcNumberTreeAsMap) > 0 {
						mergeStructTree = true
					}
				}
			}
		}
	}

	objMapping := map[*cos.Dictionary]*cos.Dictionary{}
	destinationPageTree := destinationDoc.Pages() // cache PageTree
	for page := range srcCatalog.Pages().All {
		if err := m.appendPage(cloner, destinationPageTree, page, objMapping,
			mergeStructTree, destParentTreeNextKey); err != nil {
			return err
		}
	}
	if err := mergeOpenAction(srcCatalog, destCatalog, cloner); err != nil {
		return err
	}
	if !mergeStructTree {
		return nil
	}

	if err := updatePageReferencesOfMap(cloner, srcNumberTreeAsMap, objMapping); err != nil {
		return err
	}
	maxSrcKey := -1
	for srcKey, value := range srcNumberTreeAsMap {
		if srcKey > maxSrcKey {
			maxSrcKey = srcKey
		}
		if value == nil {
			continue
		}
		cloned, err := cloner.CloneForNewDocument(value.COSObject())
		if err != nil {
			return err
		}
		converted, err := logicalstructure.ParentTreeValueConverter(cloned)
		if err != nil {
			return err
		}
		destNumberTreeAsMap[destParentTreeNextKey+srcKey] = converted
	}
	destParentTreeNextKey += maxSrcKey + 1
	newParentTreeNode := common.NewPDNumberTreeNode(logicalstructure.ParentTreeValueConverter)

	// Note that all elements are stored flatly. This could become a problem for
	// large files when these are opened in a viewer that uses the tagging
	// information. If this happens, then PDNumberTreeNode should be improved
	// with a convenience method that stores the map into a B+Tree, see
	// https://en.wikipedia.org/wiki/B+_tree
	newParentTreeNode.SetNumbers(destNumberTreeAsMap)

	destStructTree.SetParentTree(newParentTreeNode)
	destStructTree.SetParentTreeNextKey(destParentTreeNextKey)

	if err := mergeKEntries(cloner, srcStructTree, destStructTree); err != nil {
		return err
	}
	if err := mergeRoleMap(srcStructTree, destStructTree, cloner); err != nil {
		return err
	}
	if err := mergeIDTree(cloner, srcStructTree, destStructTree); err != nil {
		return err
	}
	mergeMarkInfo(destCatalog, srcCatalog)
	mergeLanguage(destCatalog, srcCatalog)
	return mergeViewerPreferences(destCatalog, srcCatalog, cloner)
}

// appendPage is the body of appendDocument's page loop.
func (m *PDFMergerUtility) appendPage(cloner *PDFCloneUtility,
	destinationPageTree *pdmodel.PDPageTree, page *pdmodel.PDPage,
	objMapping map[*cos.Dictionary]*cos.Dictionary,
	mergeStructTree bool, destParentTreeNextKey int) error {
	clonedPage, err := cloner.CloneDictionaryForNewDocument(page.Dictionary())
	if err != nil {
		return err
	}
	newPage := pdmodel.NewPDPageOf(clonedPage)
	if !mergeStructTree {
		// PDFBOX-4429: remove bogus StructParent(s)
		newPage.Dictionary().RemoveItem(cos.StructParents)
		for ann := range newPage.Annotations().All {
			ann.COSObject().(*cos.Dictionary).RemoveItem(cos.StructParent)
		}
	}
	newPage.SetCropBox(page.CropBox())
	newPage.SetMediaBox(page.MediaBox())
	newPage.SetRotation(page.Rotation())
	if err := cloneResources(cloner, page, newPage); err != nil {
		return err
	}
	if mergeStructTree {
		// add the value of the destination ParentTreeNextKey to every source
		// element StructParent(s) value so that these don't overlap with the
		// existing values
		if err := updateStructParentEntries(newPage, destParentTreeNextKey); err != nil {
			return err
		}
		objMapping[page.Dictionary()] = newPage.Dictionary()
		oldAnnots := page.Annotations().ToSlice()
		newAnnots := newPage.Annotations().ToSlice()
		for i := 0; i < len(oldAnnots) && i < len(newAnnots); i++ {
			objMapping[oldAnnots[i].COSObject().(*cos.Dictionary)] =
				newAnnots[i].COSObject().(*cos.Dictionary)
		}
		// TODO update mapping for XObjects
	}
	destinationPageTree.Add(newPage)
	return nil
}

// mergeOpenAction moves the source's /OpenAction across where the destination
// has none, dropping it where its page did not survive the merge.
func mergeOpenAction(srcCatalog, dstCatalog *pdmodel.PDDocumentCatalog,
	cloner *PDFCloneUtility) error {
	// Java reads both inside one try and catches an IOException out of either,
	// so the *second* is never assigned when the first throws: both locals stay
	// null and the block below does nothing. The port has to say that, because
	// two calls that each answer an error leave both values in hand.
	dstOpenAction, dstErr := dstCatalog.OpenAction()
	var srcOpenAction common.PDDestinationOrAction
	var srcErr error
	if dstErr == nil {
		srcOpenAction, srcErr = srcCatalog.OpenAction()
	}
	if dstErr != nil || srcErr != nil {
		// PDFBOX-4223
		slog.Error("Invalid OpenAction ignored", "error", firstError(dstErr, srcErr))
		if dstErr != nil {
			dstOpenAction = nil
		}
		srcOpenAction = nil
	}
	if dstOpenAction != nil || srcOpenAction == nil {
		return nil
	}

	clonedOpenActionBase, err := cloner.CloneForNewDocument(srcOpenAction.COSObject())
	if err != nil {
		return err
	}
	var openActionDestination destination.PDDestination
	switch cloned := clonedOpenActionBase.(type) {
	case *cos.Dictionary:
		clonedAction := action.CreateAction(cloned)
		if goTo, isGoTo := clonedAction.(*action.PDActionGoTo); isGoTo {
			openActionDestination, err = goTo.Destination()
			if err != nil {
				return err
			}
		}
		dstCatalog.SetOpenAction(clonedAction)
	case *cos.Array:
		openActionDestination, err = destination.Create(cloned)
		if err != nil {
			return err
		}
		dstCatalog.SetOpenAction(openActionDestination)
	}

	pageDestination, isPageDestination := openActionDestination.(destination.PageDestination)
	if !isPageDestination {
		return nil
	}
	page := pageDestination.Page()
	pdPage, isPage := page.(*pdmodel.PDPage)
	if isPage && dstCatalog.Pages().IndexOf(pdPage) == -1 {
		slog.Warn("OpenAction entry ignored because destination page doesn't exist")
		dstCatalog.SetOpenAction(nil)
	}
	return nil
}

// mergeViewerPreferences merges the two /ViewerPreferences dictionaries, and
// then ORs the six booleans.
func mergeViewerPreferences(destCatalog, srcCatalog *pdmodel.PDDocumentCatalog,
	cloner *PDFCloneUtility) error {
	srcViewerPreferences := srcCatalog.ViewerPreferences()
	if srcViewerPreferences == nil {
		return nil
	}
	destViewerPreferences := destCatalog.ViewerPreferences()
	if destViewerPreferences == nil {
		destViewerPreferences = viewerpreferences.NewPDViewerPreferences()
		destCatalog.SetViewerPreferences(destViewerPreferences)
	}
	if err := mergeInto(srcViewerPreferences.Dictionary(),
		destViewerPreferences.Dictionary(), cloner, nil); err != nil {
		return err
	}

	// check the booleans - set to true if one is set and true
	if srcViewerPreferences.HideToolbar() || destViewerPreferences.HideToolbar() {
		destViewerPreferences.SetHideToolbar(true)
	}
	if srcViewerPreferences.HideMenubar() || destViewerPreferences.HideMenubar() {
		destViewerPreferences.SetHideMenubar(true)
	}
	if srcViewerPreferences.HideWindowUI() || destViewerPreferences.HideWindowUI() {
		destViewerPreferences.SetHideWindowUI(true)
	}
	if srcViewerPreferences.FitWindow() || destViewerPreferences.FitWindow() {
		destViewerPreferences.SetFitWindow(true)
	}
	if srcViewerPreferences.CenterWindow() || destViewerPreferences.CenterWindow() {
		destViewerPreferences.SetCenterWindow(true)
	}
	if srcViewerPreferences.DisplayDocTitle() || destViewerPreferences.DisplayDocTitle() {
		destViewerPreferences.SetDisplayDocTitle(true)
	}
	return nil
}

// mergeLanguage takes the source's /Lang where the destination has none.
func mergeLanguage(destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) {
	if destCatalog.Language() == "" {
		if srcLanguage := srcCatalog.Language(); srcLanguage != "" {
			destCatalog.SetLanguage(srcLanguage)
		}
	}
}

// mergeMarkInfo marks the destination and ORs the two flags.
//
// The second of those two lines calls the wrong setter in the Java, so
// /UserProperties is never written and /Suspect is written twice; see
// migration/JAVA-BUGS.md.
func mergeMarkInfo(destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) {
	destMark := destCatalog.MarkInfo()
	srcMark := srcCatalog.MarkInfo()
	if destMark == nil {
		destMark = logicalstructure.NewPDMarkInfo()
	}
	if srcMark == nil {
		srcMark = logicalstructure.NewPDMarkInfo()
	}
	destMark.SetMarked(true)
	destMark.SetSuspect(srcMark.IsSuspect() || destMark.IsSuspect())
	destMark.SetSuspect(srcMark.UsesUserProperties() || destMark.UsesUserProperties())
	destCatalog.SetMarkInfo(destMark)
}

// mergeKEntries puts the source's structure elements under the destination's,
// making a /Document over both where there is no place to insert them.
func mergeKEntries(cloner *PDFCloneUtility,
	srcStructTree, destStructTree *logicalstructure.PDStructureTreeRoot) error {
	srcKEntry := srcStructTree.K()
	srcKArray := cos.NewArray()
	clonedSrcKEntry, err := cloner.CloneForNewDocument(srcKEntry)
	if err != nil {
		return err
	}
	switch cloned := clonedSrcKEntry.(type) {
	case *cos.Array:
		srcKArray.AddAll(cloned.ToList())
	case *cos.Dictionary:
		srcKArray.Add(cloned)
	}
	if srcKArray.Size() == 0 {
		return nil
	}

	dstKArray := cos.NewArray()
	switch dstKEntry := destStructTree.K().(type) {
	case *cos.Array:
		dstKArray.AddAll(dstKEntry.ToList())
	case *cos.Dictionary:
		dstKArray.Add(dstKEntry)
	}

	if dstKArray.Size() == 1 {
		if topKDict, isDictionary := dstKArray.GetObject(0).(*cos.Dictionary); isDictionary {
			// Only one element in the destination. If it is a /Document and its
			// children are /Document or /Part, then we can insert there
			if topKDict.GetCOSName(cos.S) == cos.DocumentName {
				kLevelOneArray := topKDict.GetCOSArray(cos.K)
				if kLevelOneArray != nil && hasOnlyDocumentsOrParts(kLevelOneArray) {
					// insert src elements at level 1
					kLevelOneArray.AddAll(srcKArray.ToList())
					updateParentEntry(kLevelOneArray, topKDict, cos.Part)
					return nil
				}
			}
		}
	}

	if dstKArray.Size() == 0 {
		updateParentEntry(srcKArray, destStructTree.COSObject().(*cos.Dictionary), nil)
		destStructTree.SetK(srcKArray)
		return nil
	}

	// whatever this is, merge this under a new /Document element
	dstKArray.AddAll(srcKArray.ToList())
	kLevelZeroDict := cos.NewDictionary()
	// If it is all Document, then make it all Part
	var newStructureType *cos.Name
	if hasOnlyDocumentsOrParts(dstKArray) {
		newStructureType = cos.Part
	}
	updateParentEntry(dstKArray, kLevelZeroDict, newStructureType)
	kLevelZeroDict.SetItem(cos.K, dstKArray)
	kLevelZeroDict.SetItem(cos.P, destStructTree.COSObject())
	kLevelZeroDict.SetItem(cos.S, cos.DocumentName)
	destStructTree.SetK(kLevelZeroDict)
	return nil
}

func hasOnlyDocumentsOrParts(kLevelOneArray *cos.Array) bool {
	for i := 0; i < kLevelOneArray.Size(); i++ {
		dict, isDictionary := kLevelOneArray.GetObject(i).(*cos.Dictionary)
		if !isDictionary {
			return false
		}
		sEntry := dict.GetCOSName(cos.S)
		if sEntry != cos.DocumentName && sEntry != cos.Part {
			return false
		}
	}
	return true
}

// updateParentEntry updates the P reference to the new parent dictionary, and
// the /S to the new structure type where one is given.
func updateParentEntry(kArray *cos.Array, newParent *cos.Dictionary,
	newStructureType *cos.Name) {
	for i := 0; i < kArray.Size(); i++ {
		dictEntry, isDictionary := kArray.GetObject(i).(*cos.Dictionary)
		if !isDictionary {
			continue
		}
		dictEntry.SetItem(cos.P, newParent)
		if newStructureType != nil {
			dictEntry.SetItem(cos.S, newStructureType)
		}
	}
}

// mergeIDTree merges the two /IDTree name trees, keeping the destination's
// entry where a name is in both.
func mergeIDTree(cloner *PDFCloneUtility,
	srcStructTree, destStructTree *logicalstructure.PDStructureTreeRoot) error {
	if srcStructTree == nil {
		return nil
	}
	srcIDTree := srcStructTree.IDTree()
	if srcIDTree == nil {
		return nil
	}
	destIDTree := destStructTree.IDTree()
	if destIDTree == nil {
		destIDTree = pdmodel.NewPDStructureElementNameTreeNode()
	}
	srcNames, err := GetIDTreeAsMap(srcIDTree)
	if err != nil {
		return err
	}
	destNames, err := GetIDTreeAsMap(destIDTree)
	if err != nil {
		return err
	}
	for key, value := range srcNames {
		if _, found := destNames[key]; found {
			slog.Warn("key already exists in destination IDTree", "key", key)
			continue
		}
		if value == nil {
			continue
		}
		cloned, err := cloner.CloneDictionaryForNewDocument(value.COSObject().(*cos.Dictionary))
		if err != nil {
			return err
		}
		destNames[key] = logicalstructure.NewPDStructureElementOf(cloned)
	}
	newIDTree := pdmodel.NewPDStructureElementNameTreeNode()
	newIDTree.SetNames(destNames)
	destStructTree.SetIDTree(newIDTree)
	// Note that all elements are stored flatly. This could become a problem for
	// large files when these are opened in a viewer that uses the tagging
	// information. If this happens, then PDNameTreeNode should be improved with
	// a convenience method that stores the map into a B+Tree, see
	// https://en.wikipedia.org/wiki/B+_tree
	return nil
}

// GetIDTreeAsMap flattens a name tree of structure elements.
//
// "PDNameTreeNode.getNames() only brings one level, this is why we need this."
// Java's method is package-private and static, and its own test calls it; the
// port exports it for the same reason.
func GetIDTreeAsMap(idTree common.NameTreeNode[*logicalstructure.PDStructureElement]) (
	map[string]*logicalstructure.PDStructureElement, error) {
	names := map[string]*logicalstructure.PDStructureElement{}
	if idTree == nil {
		return names, nil
	}
	own, err := idTree.Names()
	if err != nil {
		return nil, err
	}
	// must copy because the map is read only
	for key, value := range own {
		names[key] = value
	}
	kids := idTree.Kids()
	if kids != nil {
		for kid := range kids.All {
			kidNames, err := GetIDTreeAsMap(kid)
			if err != nil {
				return nil, err
			}
			for key, value := range kidNames {
				names[key] = value
			}
		}
	}
	return names, nil
}

// GetNumberTreeAsMap flattens a number tree.
//
// "PDNumberTreeNode.getNumbers() only brings one level, this is why we need
// this."
func GetNumberTreeAsMap(tree *common.PDNumberTreeNode) (map[int]common.COSObjectable, error) {
	numbers := map[int]common.COSObjectable{}
	if tree == nil {
		return numbers, nil
	}
	own, err := tree.Numbers()
	if err != nil {
		return nil, err
	}
	// must copy because the map is read only
	for key, value := range own {
		numbers[key] = value
	}
	kids := tree.Kids()
	if kids != nil {
		for i := 0; i < kids.Size(); i++ {
			kidNumbers, err := GetNumberTreeAsMap(kids.Get(i))
			if err != nil {
				return nil, err
			}
			for key, value := range kidNumbers {
				numbers[key] = value
			}
		}
	}
	return numbers, nil
}

// mergeRoleMap merges the two /RoleMap dictionaries, keeping the destination's
// entry where a key is in both and differs.
func mergeRoleMap(srcStructTree, destStructTree *logicalstructure.PDStructureTreeRoot,
	cloner *PDFCloneUtility) error {
	srcDict := srcStructTree.COSObject().(*cos.Dictionary).GetCOSDictionary(cos.RoleMap)
	if srcDict == nil {
		return nil
	}
	destDictionary := destStructTree.COSObject().(*cos.Dictionary)
	destDict := destDictionary.GetCOSDictionary(cos.RoleMap)
	if destDict == nil {
		cloned, err := cloner.CloneForNewDocument(srcDict)
		if err != nil {
			return err
		}
		destDictionary.SetItem(cos.RoleMap, cloned)
		return nil
	}
	for _, key := range srcDict.KeySet() {
		value := srcDict.GetItem(key)
		destValue := destDict.GetDictionaryObject(key)
		if destValue != nil && cos.Equal(destValue, value) {
			// already exists, but identical
			continue
		}
		if destDict.ContainsKey(key) {
			slog.Warn("key already exists in destination RoleMap", "key", key.Name())
			continue
		}
		cloned, err := cloner.CloneForNewDocument(value)
		if err != nil {
			return err
		}
		destDict.SetItem(key, cloned)
	}
	return nil
}

// mergeAcroForm merges the source form into the destination form.
func (m *PDFMergerUtility) mergeAcroForm(cloner *PDFCloneUtility,
	destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) error {
	err := m.mergeAcroFormBody(cloner, destCatalog, srcCatalog)
	if err != nil && !m.ignoreAcroFormErrors {
		// if we are not ignoring exceptions, we'll re-throw this
		return err
	}
	return nil
}

// mergeAcroFormBody is the body of Java's try.
func (m *PDFMergerUtility) mergeAcroFormBody(cloner *PDFCloneUtility,
	destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) error {
	destAcroForm := form.AcroFormOfCatalog(destCatalog)
	srcAcroForm := form.AcroFormOfCatalog(srcCatalog)

	if destAcroForm == nil && srcAcroForm != nil {
		cloned, err := cloner.CloneForNewDocument(srcAcroForm.COSObject())
		if err != nil {
			return err
		}
		destCatalog.COSObject().(*cos.Dictionary).SetItem(cos.AcroForm, cloned)
		return nil
	}
	if srcAcroForm == nil {
		return nil
	}
	switch m.acroFormMergeMode {
	case PDFBoxLegacyAcroFormMode:
		return m.acroFormLegacyMode(cloner, destAcroForm, srcAcroForm)
	case JoinFormFieldsMode:
		return m.acroFormJoinFieldsMode(cloner, destAcroForm, srcAcroForm)
	}
	return nil
}

// acroFormJoinFieldsMode merges the contents of the source form into the
// destination form for the destination file.
//
// Java's body is a single call to acroFormLegacyMode: the mode is declared and
// selected for, and the two modes do the same thing.
func (m *PDFMergerUtility) acroFormJoinFieldsMode(cloner *PDFCloneUtility,
	destAcroForm, srcAcroForm *form.PDAcroForm) error {
	return m.acroFormLegacyMode(cloner, destAcroForm, srcAcroForm)
}

// dummyFieldPrefix is the name a field gets when its own collides with one in
// the destination.
const dummyFieldPrefix = "dummyFieldName"

// dummyFieldSuffix is Java's `suffix.matches("\\d+")`.
var dummyFieldSuffix = regexp.MustCompile(`^\d+$`)

// acroFormLegacyMode merges the contents of the source form into the
// destination form, renaming a field whose fully qualified name is taken.
func (m *PDFMergerUtility) acroFormLegacyMode(cloner *PDFCloneUtility,
	destAcroForm, srcAcroForm *form.PDAcroForm) error {
	srcFields := srcAcroForm.Fields()
	if len(srcFields) == 0 {
		return nil
	}

	// if a form is merged multiple times using PDFBox the newly generated
	// fields starting with dummyFieldName may already exist. We need to
	// determine the last unique number used and increment that.
	for destField := range destAcroForm.FieldTree().All() {
		fieldName := destField.PartialName()
		if !strings.HasPrefix(fieldName, dummyFieldPrefix) {
			continue
		}
		suffix := fieldName[len(dummyFieldPrefix):]
		if !dummyFieldSuffix.MatchString(suffix) {
			continue
		}
		number, err := strconv.Atoi(suffix)
		if err != nil {
			continue
		}
		if number+1 > m.nextFieldNum {
			m.nextFieldNum = number + 1
		}
	}

	// get the destinations root fields. Could be that the entry doesn't exist
	// or is of wrong type
	destFields, _ := destAcroForm.COSObject().(*cos.Dictionary).GetItem(cos.Fields).(*cos.Array)
	if destFields == nil {
		destFields = cos.NewArray()
	}

	for _, srcField := range srcFields {
		dstField, err := cloner.CloneDictionaryForNewDocument(
			srcField.COSObject().(*cos.Dictionary))
		if err != nil {
			return err
		}
		// if the form already has a field with this name then we need to rename
		// this field to prevent merge conflicts.
		existing := destAcroForm.Field(srcField.FullyQualifiedName())
		if existing != nil {
			dstField.SetString(cos.T, dummyFieldPrefix+strconv.Itoa(m.nextFieldNum))
			m.nextFieldNum++
		}
		destFields.Add(dstField)
	}
	destAcroForm.COSObject().(*cos.Dictionary).SetItem(cos.Fields, destFields)
	return nil
}

// updatePageReferencesOfMap updates the Pg and Obj references of every value of
// a flattened number tree.
func updatePageReferencesOfMap(cloner *PDFCloneUtility,
	numberTreeAsMap map[int]common.COSObjectable,
	objMapping map[*cos.Dictionary]*cos.Dictionary) error {
	for _, obj := range numberTreeAsMap {
		if obj == nil {
			continue
		}
		switch base := obj.COSObject().(type) {
		case *cos.Array:
			if err := updatePageReferencesOfArray(cloner, base, objMapping); err != nil {
				return err
			}
		case *cos.Dictionary:
			if err := updatePageReferences(cloner, base, objMapping); err != nil {
				return err
			}
		}
	}
	return nil
}

// updatePageReferences updates the Pg and Obj references to the new (merged)
// page.
func updatePageReferences(cloner *PDFCloneUtility, parentTreeEntry *cos.Dictionary,
	objMapping map[*cos.Dictionary]*cos.Dictionary) error {
	pageDict := parentTreeEntry.GetCOSDictionary(cos.Pg)
	if mapped, found := objMapping[pageDict]; found {
		parentTreeEntry.SetItem(cos.Pg, mapped)
	}
	objDict := parentTreeEntry.GetCOSDictionary(cos.Obj)
	if objDict != nil {
		if mapped, found := objMapping[objDict]; found {
			parentTreeEntry.SetItem(cos.Obj, mapped)
		} else {
			// PDFBOX-3999: clone objects that are not in mapping to make sure
			// that these don't remain attached to the source document
			slog.Debug("clone potential orphan object in structure tree",
				"type", objDict.GetNameAsString(cos.Type, ""),
				"subtype", objDict.GetNameAsString(cos.Subtype, ""),
				"T", objDict.GetNameAsString(cos.T, ""))
			cloned, err := cloner.CloneForNewDocument(objDict)
			if err != nil {
				return err
			}
			parentTreeEntry.SetItem(cos.Obj, cloned)
		}
	}
	switch kSubEntry := parentTreeEntry.GetDictionaryObject(cos.K).(type) {
	case *cos.Array:
		return updatePageReferencesOfArray(cloner, kSubEntry, objMapping)
	case *cos.Dictionary:
		return updatePageReferences(cloner, kSubEntry, objMapping)
	}
	return nil
}

func updatePageReferencesOfArray(cloner *PDFCloneUtility, parentTreeEntry *cos.Array,
	objMapping map[*cos.Dictionary]*cos.Dictionary) error {
	for i := 0; i < parentTreeEntry.Size(); i++ {
		switch subEntry := parentTreeEntry.GetObject(i).(type) {
		case *cos.Array:
			if err := updatePageReferencesOfArray(cloner, subEntry, objMapping); err != nil {
				return err
			}
		case *cos.Dictionary:
			if err := updatePageReferences(cloner, subEntry, objMapping); err != nil {
				return err
			}
		}
	}
	return nil
}

// updateStructParentEntries updates the StructParents and StructParent values
// in a PDPage by the given offset.
func updateStructParentEntries(page *pdmodel.PDPage, structParentOffset int) error {
	if structParents := page.StructParents(); structParents >= 0 {
		page.SetStructParents(structParents + structParentOffset)
	}
	annots := page.Annotations().ToSlice()
	newannots := make([]annotation.PDAnnotation, 0, len(annots))
	for _, annot := range annots {
		// The port's PDAnnotation interface does not carry StructParent, which
		// PDAnnotationBase has; the entry is the same one either way.
		dictionary := annot.COSObject().(*cos.Dictionary)
		if structParent := dictionary.GetInt(cos.StructParent); structParent >= 0 {
			dictionary.SetInt(cos.StructParent, structParent+structParentOffset)
		}
		newannots = append(newannots, annot)
	}
	page.SetAnnotations(newannots)
	return nil
}

// isDynamicXfa tests for dynamic XFA content.
func isDynamicXfa(acroForm *form.PDAcroForm) bool {
	return acroForm != nil && acroForm.XFAIsDynamic()
}

// mergeInto adds all of the source dictionary's keys and values to the
// destination, but only if they are not in an exclusion list and if they don't
// already exist. If a key already exists in the destination then nothing is
// changed.
func mergeInto(src, dst *cos.Dictionary, cloner *PDFCloneUtility, exclude []*cos.Name) error {
	for _, key := range src.KeySet() {
		if containsName(exclude, key) || dst.ContainsKey(key) {
			continue
		}
		cloned, err := cloner.CloneForNewDocument(src.GetItem(key))
		if err != nil {
			return err
		}
		dst.SetItem(key, cloned)
	}
	return nil
}

func containsName(names []*cos.Name, name *cos.Name) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}

// maxKey is Java's Collections.max over a key set.
func maxKey(numbers map[int]common.COSObjectable) int {
	max := 0
	first := true
	for key := range numbers {
		if first || key > max {
			max = key
			first = false
		}
	}
	return max
}

// firstError answers the first of two errors that is not nil.
func firstError(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// loadPDFFile and loadPDFFrom are Loader.loadPDF, which multipdf is the only
// caller of outside the tools.
func loadPDFFile(path string) (*pdmodel.PDDocument, error) { return pdfbox.LoadPDF(path) }

func loadPDFFrom(source pdfio.RandomAccessRead) (*pdmodel.PDDocument, error) {
	return pdfbox.LoadPDFFrom(source)
}

// closeQuietly is IOUtils.closeQuietly, which swallows whatever close answers.
func closeQuietly(doc *pdmodel.PDDocument) {
	if doc != nil {
		_ = doc.Close()
	}
}

// closeAndLogException is IOUtils.closeAndLogException, which logs it instead.
func closeAndLogException(doc *pdmodel.PDDocument) {
	if doc == nil {
		return
	}
	if err := doc.Close(); err != nil {
		slog.Warn("multipdf: closing a PDDocument", "error", err)
	}
}
