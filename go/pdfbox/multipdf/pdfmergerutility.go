package multipdf

// This class will take a list of pdf documents and merge them, saving the
// result in a new document.
//
// Port of org.apache.pdfbox.multipdf.PDFMergerUtility.
//
// Two things the Java signature carries that the port does not. The first is
// `StreamCacheCreateFunction`, which every `mergeDocuments` overload takes and
// hands to `new PDDocument(...)`: the port's `PDDocument` has no such
// parameter -- it is always memory-backed -- so the merge has nowhere to put
// one and the overloads collapse into one. The second is `CompressParameters`,
// which the port's `SaveOfParameters` does take, so that one is kept.

import (
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/outline"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// DocumentMergeMode is the mode to use when merging documents.
//
//   - OptimizeResourcesMode optimizes resource handling such as closing
//     documents early. **Not all document elements are merged** compared to
//     the PDFBoxLegacyMode. Currently supported are: page content and
//     resources.
//   - PDFBoxLegacyMode keeps all files open until the merge has been
//     completed. This is currently necessary to merge documents containing a
//     Structure Tree.
type DocumentMergeMode int

const (
	// PDFBoxLegacyDocumentMode is DocumentMergeMode.PDFBOX_LEGACY_MODE, and is
	// the default. Java's enum lists OPTIMIZE_RESOURCES_MODE first; the port
	// puts the default first so that the zero value is the default.
	PDFBoxLegacyDocumentMode DocumentMergeMode = iota

	// OptimizeResourcesMode is DocumentMergeMode.OPTIMIZE_RESOURCES_MODE.
	OptimizeResourcesMode
)

// AcroFormMergeMode is the mode to use when merging AcroForms between
// documents.
//
//   - JoinFormFieldsMode: fields with the same fully qualified name will be
//     merged into one with the widget annotations of the merged fields
//     becoming part of the same field.
//   - PDFBoxLegacyAcroFormMode: fields with the same fully qualified name will
//     be renamed and treated as independent. This mode was used in versions of
//     PDFBox up to 2.x.
type AcroFormMergeMode int

const (
	// PDFBoxLegacyAcroFormMode is AcroFormMergeMode.PDFBOX_LEGACY_MODE, and is
	// the default.
	PDFBoxLegacyAcroFormMode AcroFormMergeMode = iota

	// JoinFormFieldsMode is AcroFormMergeMode.JOIN_FORM_FIELDS_MODE.
	JoinFormFieldsMode
)

// PDFMergerUtility merges a list of PDF documents into one.
type PDFMergerUtility struct {
	sources                        []any
	destinationFileName            string
	destinationStream              io.Writer
	ignoreAcroFormErrors           bool
	destinationDocumentInformation *pdmodel.PDDocumentInformation
	destinationMetadata            *common.PDMetadata

	documentMergeMode DocumentMergeMode
	acroFormMergeMode AcroFormMergeMode

	nextFieldNum int
}

// NewPDFMergerUtility instantiates a new PDFMergerUtility.
func NewPDFMergerUtility() *PDFMergerUtility {
	// Java's `private int nextFieldNum = 1` is a field initialiser, not part of
	// the constructor.
	return &PDFMergerUtility{nextFieldNum: 1}
}

// AcroFormMergeMode returns the merge mode to be used for merging AcroForms
// between documents.
func (m *PDFMergerUtility) AcroFormMergeMode() AcroFormMergeMode { return m.acroFormMergeMode }

// SetAcroFormMergeMode sets the merge mode to be used for merging AcroForms
// between documents.
func (m *PDFMergerUtility) SetAcroFormMergeMode(mode AcroFormMergeMode) {
	m.acroFormMergeMode = mode
}

// DocumentMergeMode returns the merge mode to be used for merging documents.
func (m *PDFMergerUtility) DocumentMergeMode() DocumentMergeMode { return m.documentMergeMode }

// SetDocumentMergeMode sets the merge mode to be used for merging documents.
func (m *PDFMergerUtility) SetDocumentMergeMode(mode DocumentMergeMode) {
	m.documentMergeMode = mode
}

// DestinationFileName returns the name of the destination file.
func (m *PDFMergerUtility) DestinationFileName() string { return m.destinationFileName }

// SetDestinationFileName sets the name of the destination file.
func (m *PDFMergerUtility) SetDestinationFileName(destination string) {
	m.destinationFileName = destination
}

// DestinationStream returns the destination stream.
func (m *PDFMergerUtility) DestinationStream() io.Writer { return m.destinationStream }

// SetDestinationStream sets the destination stream.
func (m *PDFMergerUtility) SetDestinationStream(destStream io.Writer) {
	m.destinationStream = destStream
}

// DestinationDocumentInformation returns the document information that is to be
// set in MergeDocuments. The default is nil, which means that it is ignored.
func (m *PDFMergerUtility) DestinationDocumentInformation() *pdmodel.PDDocumentInformation {
	return m.destinationDocumentInformation
}

// SetDestinationDocumentInformation sets the document information that is to be
// set in MergeDocuments.
func (m *PDFMergerUtility) SetDestinationDocumentInformation(info *pdmodel.PDDocumentInformation) {
	m.destinationDocumentInformation = info
}

// DestinationMetadata returns the metadata that is to be set in MergeDocuments.
// The default is nil, which means that it is ignored.
func (m *PDFMergerUtility) DestinationMetadata() *common.PDMetadata {
	return m.destinationMetadata
}

// SetDestinationMetadata sets the metadata that is to be set in MergeDocuments.
func (m *PDFMergerUtility) SetDestinationMetadata(meta *common.PDMetadata) {
	m.destinationMetadata = meta
}

// AddSourceFile adds a source file to the list of files to merge.
//
// Port of addSource(String) and addSource(File), which are the same method
// twice: Java's first wraps the name in a File and calls the second, and
// neither of them checks anything despite the `throws FileNotFoundException`.
func (m *PDFMergerUtility) AddSourceFile(source string) error {
	m.sources = append(m.sources, source)
	return nil
}

// AddSource adds a source to the list of documents to merge.
//
// Port of addSource(RandomAccessRead).
func (m *PDFMergerUtility) AddSource(source pdfio.RandomAccessRead) {
	m.sources = append(m.sources, source)
}

// AddSources adds a list of sources to the list of documents to merge.
func (m *PDFMergerUtility) AddSources(sourcesList []pdfio.RandomAccessRead) {
	for _, source := range sourcesList {
		m.sources = append(m.sources, source)
	}
}

// IsIgnoreAcroFormErrors indicates if acroform errors are ignored or not.
func (m *PDFMergerUtility) IsIgnoreAcroFormErrors() bool { return m.ignoreAcroFormErrors }

// SetIgnoreAcroFormErrors sets whether acroform errors should be ignored.
func (m *PDFMergerUtility) SetIgnoreAcroFormErrors(value bool) { m.ignoreAcroFormErrors = value }

// MergeDocuments merges the list of source documents, saving the result in the
// destination file. The source list is not reset after merge. If you want to
// merge one document at a time, then it's better to use AppendDocument.
func (m *PDFMergerUtility) MergeDocuments() error {
	return m.MergeDocumentsOfParameters(compress.DefaultCompression)
}

// MergeDocumentsOfParameters is the same with the compression given.
//
// Port of mergeDocuments(StreamCacheCreateFunction, CompressParameters).
func (m *PDFMergerUtility) MergeDocumentsOfParameters(
	compressParameters *compress.Parameters) error {
	switch m.documentMergeMode {
	case PDFBoxLegacyDocumentMode:
		return m.legacyMergeDocuments(compressParameters)
	case OptimizeResourcesMode:
		return m.optimizedMergeDocuments(compressParameters)
	}
	return nil
}

// optimizedMergeDocuments merges page content and resources only, closing each
// source as soon as it has been read.
func (m *PDFMergerUtility) optimizedMergeDocuments(
	compressParameters *compress.Parameters) error {
	destinationDoc := pdmodel.NewPDDocument()
	defer destinationDoc.Close()

	cloner := NewPDFCloneUtility(destinationDoc)
	destinationPageTree := destinationDoc.Pages() // cache PageTree
	for _, sourceObject := range m.sources {
		if err := m.optimizedMergeSource(cloner, destinationPageTree, sourceObject); err != nil {
			return err
		}
	}
	return m.saveDestination(destinationDoc, compressParameters)
}

// optimizedMergeSource is the body of the loop, which Java wraps in a try with
// a finally that closes the source document however it leaves.
func (m *PDFMergerUtility) optimizedMergeSource(cloner *PDFCloneUtility,
	destinationPageTree *pdmodel.PDPageTree, sourceObject any) error {
	sourceDoc, err := loadSource(sourceObject)
	if err != nil {
		return err
	}
	defer closeQuietly(sourceDoc)

	for page := range sourceDoc.Pages().All {
		clonedPage, err := cloner.CloneDictionaryForNewDocument(page.Dictionary())
		if err != nil {
			return err
		}
		newPage := pdmodel.NewPDPageOf(clonedPage)
		newPage.SetCropBox(page.CropBox())
		newPage.SetMediaBox(page.MediaBox())
		newPage.SetRotation(page.Rotation())
		if err := cloneResources(cloner, page, newPage); err != nil {
			return err
		}
		destinationPageTree.Add(newPage)
	}
	return nil
}

// legacyMergeDocuments merges everything, keeping every source open until the
// merge is done.
func (m *PDFMergerUtility) legacyMergeDocuments(
	compressParameters *compress.Parameters) error {
	if len(m.sources) == 0 {
		return nil
	}
	// Make sure that:
	// - first Exception is kept
	// - all PDDocuments are closed
	// - all FileInputStreams are closed
	// - there's a way to see which errors occurred
	destinationDoc := pdmodel.NewPDDocument()
	defer destinationDoc.Close()

	for _, sourceObject := range m.sources {
		sourceDoc, err := loadSource(sourceObject)
		if err != nil {
			return err
		}
		err = m.AppendDocument(destinationDoc, sourceDoc)
		closeAndLogException(sourceDoc)
		if err != nil {
			return err
		}
	}

	// optionally set meta data
	if m.destinationDocumentInformation != nil {
		destinationDoc.SetDocumentInformation(m.destinationDocumentInformation)
	}
	if m.destinationMetadata != nil {
		destinationDoc.DocumentCatalog().SetMetadata(m.destinationMetadata)
	}
	return m.saveDestination(destinationDoc, compressParameters)
}

// saveDestination is the two-line tail both merges end with.
func (m *PDFMergerUtility) saveDestination(destinationDoc *pdmodel.PDDocument,
	compressParameters *compress.Parameters) error {
	if m.destinationStream == nil {
		return destinationDoc.SaveToFileOfParameters(m.destinationFileName, compressParameters)
	}
	return destinationDoc.SaveOfParameters(m.destinationStream, compressParameters)
}

// loadSource opens one entry of the source list, which is a file name or a
// RandomAccessRead.
func loadSource(sourceObject any) (*pdmodel.PDDocument, error) {
	switch source := sourceObject.(type) {
	case string:
		return loadPDFFile(source)
	case pdfio.RandomAccessRead:
		return loadPDFFrom(source)
	}
	return nil, fmt.Errorf("multipdf: a source of %T cannot be merged", sourceObject)
}

// cloneResources is the four lines both merges spend on a page's resources.
//
// "this is smart enough to just create references for resources that are used
// on multiple pages"
func cloneResources(cloner *PDFCloneUtility, page, newPage *pdmodel.PDPage) error {
	resources := page.Resources()
	if resources == nil {
		newPage.SetResources(pdmodel.NewPDResources())
		return nil
	}
	cloned, err := cloner.CloneDictionaryForNewDocument(resources.Dictionary())
	if err != nil {
		return err
	}
	newPage.SetResources(pdmodel.NewPDResourcesOf(cloned))
	return nil
}

// AppendDocument appends all pages from source to destination.
//
// The source "should not be a PDDocument that you created on the fly, it should
// be saved first, if it contains any fonts that are subset."
func (m *PDFMergerUtility) AppendDocument(destinationDoc, source *pdmodel.PDDocument) error {
	cloner := NewPDFCloneUtility(destinationDoc)
	if source.Document().IsClosed() {
		return errors.New("Error: source PDF is closed.")
	}
	if destinationDoc.Document().IsClosed() {
		return errors.New("Error: destination PDF is closed.")
	}

	srcCatalog := source.DocumentCatalog()
	if isDynamicXfa(form.AcroFormOfCatalog(srcCatalog)) {
		return errors.New("Error: can't merge source document containing dynamic XFA form content.")
	}

	destInfo := destinationDoc.DocumentInformation()
	srcInfo := source.DocumentInformation()
	if err := mergeInto(srcInfo.COSObject().(*cos.Dictionary),
		destInfo.COSObject().(*cos.Dictionary), cloner, nil); err != nil {
		return err
	}

	// use the highest version number for the resulting pdf
	if destinationDoc.Version() < source.Version() {
		destinationDoc.SetVersion(source.Version())
	}

	destCatalog := destinationDoc.DocumentCatalog()
	if err := m.mergeAcroForm(cloner, destCatalog, srcCatalog); err != nil {
		return err
	}
	if err := mergeThreads(cloner, destCatalog, srcCatalog); err != nil {
		return err
	}
	if err := mergeNames(cloner, destCatalog, srcCatalog); err != nil {
		return err
	}
	if err := mergeDests(cloner, destCatalog, srcCatalog); err != nil {
		return err
	}
	if err := mergeOutline(cloner, destCatalog, srcCatalog); err != nil {
		return err
	}
	mergePageMode(destCatalog, srcCatalog)
	if err := mergePageLabels(cloner, destinationDoc, destCatalog, srcCatalog); err != nil {
		return err
	}
	if err := mergeMetadata(cloner, destinationDoc, destCatalog, srcCatalog); err != nil {
		return err
	}
	if err := mergeOptionalContent(cloner, destCatalog, srcCatalog); err != nil {
		return err
	}
	if err := mergeOutputIntents(srcCatalog, destCatalog, cloner); err != nil {
		return err
	}
	return m.mergePagesAndStructureTree(cloner, destinationDoc, destCatalog, srcCatalog)
}

// mergeThreads is Java's /Threads block.
//
// Java reads the *destination* catalog for the array it clones as well as for
// the one it merges into, so the source's threads were never merged and the
// destination's were cloned into themselves -- doubling every merge. The
// clone is of the source, like every other block here. See
// migration/JAVA-BUGS.md 82.
func mergeThreads(cloner *PDFCloneUtility, destCatalog,
	srcCatalog *pdmodel.PDDocumentCatalog) error {
	destDictionary := destCatalog.COSObject().(*cos.Dictionary)
	srcDictionary := srcCatalog.COSObject().(*cos.Dictionary)
	destThreads := destDictionary.GetCOSArray(cos.Threads)

	// A nil *cos.Array in a cos.Base is not a nil cos.Base, so the guard
	// `cloneForNewDocument` opens with -- `if (base == null) return null` --
	// does not fire on one. Java reaches it because a Java null is a null
	// whatever its static type; the port has to not make the call.
	var srcThreads *cos.Array
	if toClone := srcDictionary.GetCOSArray(cos.Threads); toClone != nil {
		cloned, err := cloner.CloneForNewDocument(toClone)
		if err != nil {
			return err
		}
		srcThreads, _ = cloned.(*cos.Array)
	}

	if destThreads == nil {
		if srcThreads != nil {
			destDictionary.SetItem(cos.Threads, srcThreads)
		}
		return nil
	}
	if srcThreads != nil {
		destThreads.AddAll(srcThreads.ToList())
	}
	return nil
}

// mergeNames is Java's /Names block, and the /IDTree removal after it.
func mergeNames(cloner *PDFCloneUtility, destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) error {
	destNames := destCatalog.Names()
	srcNames := srcCatalog.Names()
	if srcNames != nil {
		if destNames == nil {
			cloned, err := cloner.CloneForNewDocument(srcNames.COSObject())
			if err != nil {
				return err
			}
			destCatalog.COSObject().(*cos.Dictionary).SetItem(cos.Names, cloned)
		} else if err := cloner.CloneMerge(srcNames, destNames); err != nil {
			return err
		}
	}
	if destNames != nil && destNames.COSObject().(*cos.Dictionary).ContainsKey(cos.IDTree) {
		// found in 001031.pdf from PDFBOX-4417 and doesn't belong there
		destNames.COSObject().(*cos.Dictionary).RemoveItem(cos.IDTree)
		slog.Warn("Removed /IDTree from /Names dictionary, doesn't belong there")
	}
	return nil
}

// mergeDests is Java's /Dests block.
func mergeDests(cloner *PDFCloneUtility, destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) error {
	srcDests := srcCatalog.Dests()
	if srcDests == nil {
		return nil
	}
	destDests := destCatalog.Dests()
	if destDests == nil {
		cloned, err := cloner.CloneForNewDocument(srcDests.COSObject())
		if err != nil {
			return err
		}
		destCatalog.COSObject().(*cos.Dictionary).SetItem(cos.Dests, cloned)
		return nil
	}
	return cloner.CloneMerge(srcDests, destDests)
}

// mergeOutline is Java's /Outlines block.
func mergeOutline(cloner *PDFCloneUtility, destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) error {
	srcOutline := srcCatalog.DocumentOutline()
	if srcOutline == nil {
		return nil
	}
	destOutline := destCatalog.DocumentOutline()
	if destOutline == nil || destOutline.FirstChild() == nil {
		cloned, err := cloner.CloneDictionaryForNewDocument(srcOutline.Dictionary())
		if err != nil {
			return err
		}
		destCatalog.SetDocumentOutline(outline.NewPDDocumentOutlineOf(cloned))
		return nil
	}

	// search last sibling for dest, because /Last entry is sometimes wrong
	visited := map[*cos.Dictionary]bool{}
	destLastOutlineItem := destOutline.FirstChild()
	for {
		if visited[destLastOutlineItem.Dictionary()] {
			slog.Warn("Outline ignored", "item", destLastOutlineItem.Dictionary())
			break // Cycle detected
		}
		visited[destLastOutlineItem.Dictionary()] = true
		outlineItem := destLastOutlineItem.NextSibling()
		if outlineItem == nil {
			break
		}
		destLastOutlineItem = outlineItem
	}
	for item := range srcOutline.Children() {
		// get each child, clone its dictionary, remove siblings info,
		// append outline item created from there
		clonedDict, err := cloner.CloneDictionaryForNewDocument(item.Dictionary())
		if err != nil {
			return err
		}
		clonedDict.RemoveItem(cos.Prev)
		clonedDict.RemoveItem(cos.Next)
		clonedItem := outline.NewPDOutlineItemOf(clonedDict)
		destLastOutlineItem.InsertSiblingAfter(clonedItem)
		destLastOutlineItem = destLastOutlineItem.NextSibling()
	}
	return nil
}

// mergePageMode is Java's /PageMode block: the destination takes the source's
// page mode where it has none of its own.
//
// Java asks whether the destination's page mode is null, and `getPageMode`
// answers USE_NONE for a document with no /PageMode and never null, so its
// branch is dead and a merge into an empty destination -- which is what
// `pdfbox merge` starts with -- loses the source's page mode. The question is
// whether the entry is there. The source is asked the same way, so a source
// with no /PageMode writes none rather than a UseNone that says nothing. See
// migration/JAVA-BUGS.md 84.
func mergePageMode(destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) {
	if destCatalog.COSObject().(*cos.Dictionary).ContainsKey(cos.PageMode) {
		return
	}
	if !srcCatalog.COSObject().(*cos.Dictionary).ContainsKey(cos.PageMode) {
		return
	}
	destCatalog.SetPageMode(srcCatalog.PageMode())
}

// mergePageLabels is Java's /PageLabels block.
func mergePageLabels(cloner *PDFCloneUtility, destinationDoc *pdmodel.PDDocument,
	destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) error {
	srcLabels := srcCatalog.COSObject().(*cos.Dictionary).GetCOSDictionary(cos.PageLabels)
	if srcLabels == nil {
		return nil
	}
	destPageCount := destinationDoc.NumberOfPages()
	var destNums *cos.Array
	destDictionary := destCatalog.COSObject().(*cos.Dictionary)
	destLabels := destDictionary.GetCOSDictionary(cos.PageLabels)
	if destLabels == nil {
		destLabels = cos.NewDictionary()
		destNums = cos.NewArray()
		destLabels.SetItem(cos.Nums, destNums)
		destDictionary.SetItem(cos.PageLabels, destLabels)
	} else {
		destNums = destLabels.GetCOSArray(cos.Nums)
	}
	srcNums := srcLabels.GetCOSArray(cos.Nums)
	if srcNums == nil {
		return nil
	}
	startSize := destNums.Size()
	for i := 0; i < srcNums.Size(); i += 2 {
		base := srcNums.GetObject(i)
		labelIndex, isNumber := base.(cos.Number)
		if !isNumber {
			slog.Error("page labels ignored, index should be a number",
				"index", i, "value", base)
			// remove what we added
			for destNums.Size() > startSize {
				destNums.RemoveAt(startSize)
			}
			break
		}
		labelIndexValue := int64(labelIndex.IntValue())
		destNums.Add(cos.GetInteger(labelIndexValue + int64(destPageCount)))
		cloned, err := cloner.CloneForNewDocument(srcNums.GetObject(i + 1))
		if err != nil {
			return err
		}
		destNums.Add(cloned)
	}
	return nil
}

// mergeMetadata is Java's /Metadata block.
func mergeMetadata(cloner *PDFCloneUtility, destinationDoc *pdmodel.PDDocument,
	destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) error {
	destMetadata := destCatalog.COSObject().(*cos.Dictionary).GetCOSStream(cos.Metadata)
	srcMetadata := srcCatalog.COSObject().(*cos.Dictionary).GetCOSStream(cos.Metadata)
	if destMetadata != nil || srcMetadata == nil {
		return nil
	}
	if err := copyMetadataStream(cloner, destinationDoc, destCatalog, srcMetadata); err != nil {
		// PDFBOX-4227 cleartext XMP stream with /Flate
		slog.Error("Metadata skipped because it could not be read", "error", err)
	}
	return nil
}

// copyMetadataStream is the body of Java's try, which it catches an IOException
// out of and logs.
func copyMetadataStream(cloner *PDFCloneUtility, destinationDoc *pdmodel.PDDocument,
	destCatalog *pdmodel.PDDocumentCatalog, srcMetadata *cos.Stream) error {
	input, err := srcMetadata.CreateReader()
	if err != nil {
		return err
	}
	newStream, err := common.NewPDStreamOfInput(destinationDoc, input, nil)
	if err != nil {
		return err
	}
	if err := mergeInto(&srcMetadata.Dictionary, &newStream.Stream().Dictionary,
		cloner, []*cos.Name{cos.Filter, cos.Length}); err != nil {
		return err
	}
	destCatalog.COSObject().(*cos.Dictionary).SetItem(cos.Metadata, newStream.COSObject())
	return nil
}

// mergeOptionalContent is Java's /OCProperties block.
func mergeOptionalContent(cloner *PDFCloneUtility,
	destCatalog, srcCatalog *pdmodel.PDDocumentCatalog) error {
	destDictionary := destCatalog.COSObject().(*cos.Dictionary)
	destOCP := destDictionary.GetCOSDictionary(cos.OCProperties)
	srcOCP := srcCatalog.COSObject().(*cos.Dictionary).GetCOSDictionary(cos.OCProperties)
	switch {
	case destOCP == nil && srcOCP != nil:
		cloned, err := cloner.CloneForNewDocument(srcOCP)
		if err != nil {
			return err
		}
		destDictionary.SetItem(cos.OCProperties, cloned)
	case destOCP != nil && srcOCP != nil:
		return cloner.CloneMerge(srcOCP, destOCP)
	}
	return nil
}

// mergeOutputIntents copies outputIntents to destination, but avoids duplicate
// OutputConditionIdentifier, except when it is missing or is named "Custom".
func mergeOutputIntents(srcCatalog, destCatalog *pdmodel.PDDocumentCatalog,
	cloner *PDFCloneUtility) error {
	srcOutputIntents := srcCatalog.OutputIntents()
	dstOutputIntents := destCatalog.OutputIntents()
	for _, srcOI := range srcOutputIntents {
		srcOCI := srcOI.OutputConditionIdentifier()
		if srcOCI != "" && srcOCI != "Custom" {
			// is that identifier already there?
			skip := false
			for _, dstOI := range dstOutputIntents {
				if srcOCI == dstOI.OutputConditionIdentifier() {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}
		cloned, err := cloner.CloneDictionaryForNewDocument(srcOI.Dictionary())
		if err != nil {
			return err
		}
		destCatalog.AddOutputIntent(color.NewPDOutputIntent(cloned))
		dstOutputIntents = append(dstOutputIntents, srcOI)
	}
	return nil
}
