package pdmodel

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// The writer takes the document through an interface rather than the concrete
// type, so that pdfwriter does not import this package back; this is the check
// that the two still line up.
var _ pdfwriter.PDDocumentLike = (*PDDocument)(nil)

// DocumentId provides the document ID. This is not the trailer document ID but
// the time used to create it. Use cos.Document.DocumentID for the trailer
// document ID. Read PDFBOX-1613 for more details about the purpose.
//
// Port of getDocumentId(), whose Long is nullable.
func (d *PDDocument) DocumentId() *int64 { return d.documentId }

// SetDocumentId sets the document ID to the given value.
//
// Port of setDocumentId(Long); a nil pointer is Java's null.
func (d *PDDocument) SetDocumentId(docID *int64) { d.documentId = docID }

// SetVersion sets the version of the PDF specification the document claims.
//
// Port of setVersion(float).
func (d *PDDocument) SetVersion(newVersion float32) {
	currentVersion := d.Version()
	// nothing to do?
	if newVersion == currentVersion {
		return
	}
	// the version can't be downgraded
	if newVersion < currentVersion {
		slog.Error("pdmodel: It's not allowed to downgrade the version of a pdf.")
		return
	}
	// update the catalog version if the document version is >= 1.4
	if d.Document().Version() >= 1.4 {
		d.DocumentCatalog().SetVersion(strconv.FormatFloat(float64(newVersion), 'f', -1, 32))
	} else {
		// versions < 1.4f have a version header only
		d.Document().SetVersion(newVersion)
	}
}

// SaveToFile saves the document to a file using default compression.
//
// Don't use the input file as target as this will produce a corrupted file.
//
// If encryption has been activated, do not use the document after saving
// because the contents are now encrypted. The same applies if your file was
// created from parts of another file and that one is to be used after saving.
//
// Port of save(File) and save(String), which are the same method in Go.
func (d *PDDocument) SaveToFile(fileName string) error {
	return d.SaveToFileOfParameters(fileName, compress.DefaultCompression)
}

// SaveToFileOfParameters saves the document to a file using the given
// compression.
//
// Port of save(File, CompressParameters).
func (d *PDDocument) SaveToFileOfParameters(fileName string,
	compressParameters *compress.Parameters) error {
	if info, err := os.Stat(fileName); err == nil && info.Size() > 0 {
		slog.Warn("pdmodel: You are overwriting the existing file, this will produce a "+
			"corrupted file if you're also reading from it", "file", info.Name())
	}
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	bufferedOutputStream := bufio.NewWriter(file)
	if err := d.SaveOfParameters(bufferedOutputStream, compressParameters); err != nil {
		file.Close()
		return err
	}
	if err := bufferedOutputStream.Flush(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// Save saves the document to an output stream, using default compression.
//
// Don't use the input file as target as this will produce a corrupted file.
//
// If encryption has been activated, do not use the document after saving
// because the contents are now encrypted. The same applies if your file was
// created from parts of another file and that one is to be used after saving.
//
// Port of save(OutputStream).
func (d *PDDocument) Save(output io.Writer) error {
	return d.SaveOfParameters(output, compress.DefaultCompression)
}

// SaveOfParameters saves the document using the given compression.
//
// Port of save(OutputStream, CompressParameters).
func (d *PDDocument) SaveOfParameters(output io.Writer,
	compressParameters *compress.Parameters) error {
	if d.document.IsClosed() {
		return errors.New("Cannot save a document which has been closed")
	}

	// object stream compression requires a cross reference stream.
	d.document.SetIsXRefStream(compressParameters != nil &&
		compressParameters != compress.NoCompression)
	if err := d.subsetDesignatedFonts(); err != nil {
		return err
	}

	// save PDF
	writer := pdfwriter.NewCOSWriterOfParameters(output, compressParameters)
	return writer.Write(d)
}

// subsetDesignatedFonts subsets the fonts the document was told to subset.
//
// The set is filled by PDAbstractContentStream.SetFont, and only ever with a
// font that answers WillBeSubset. Subsetting itself is font embedding, which
// this port has not reached, so no font answers it yet and the walk is empty.
// See migration/STATUS.md.
func (d *PDDocument) subsetDesignatedFonts() error {
	// subset designated fonts
	for _, f := range d.fontsToSubset {
		if err := f.Subset(); err != nil {
			return err
		}
	}
	d.fontsToSubset = nil
	return nil
}

// SaveIncremental saves the PDF as an incremental update. This is only possible
// if the PDF was loaded from a file or a stream, not if the document was created
// in PDFBox itself. There must be a path of objects that have
// SetNeedToBeUpdated(true) from the document catalog to the objects you need to
// update.
//
// Port of saveIncremental(OutputStream).
func (d *PDDocument) SaveIncremental(output io.Writer) error {
	if err := d.subsetDesignatedFonts(); err != nil {
		return err
	}
	if d.pdfSource == nil {
		// Java throws IllegalStateException, which is unchecked.
		panic("document was not loaded from a file or a stream")
	}
	writer, err := pdfwriter.NewCOSWriterIncremental(output, d.pdfSource)
	if err != nil {
		return err
	}
	return writer.WriteSigned(d, d.signInterface)
}

// SaveIncrementalOfObjects saves the PDF as an incremental update, writing only
// the given dictionaries. This allows to include objects even if there is no
// path of objects that have SetNeedToBeUpdated(true), so the incremental update
// gets smaller. Only dictionaries are supported; if you need to update other
// object classes, then add their parent dictionary.
//
// Port of saveIncremental(OutputStream, Set<COSDictionary>).
func (d *PDDocument) SaveIncrementalOfObjects(output io.Writer,
	objectsToWrite []*cos.Dictionary) error {
	if err := d.subsetDesignatedFonts(); err != nil {
		return err
	}
	if d.pdfSource == nil {
		// Java throws IllegalStateException, which is unchecked.
		panic("document was not loaded from a file or a stream")
	}
	writer, err := pdfwriter.NewCOSWriterIncrementalOfObjects(output, d.pdfSource, objectsToWrite)
	if err != nil {
		return err
	}
	return writer.WriteSigned(d, d.signInterface)
}

// ImportPage imports a page from another document, cloning what it needs into
// this one, and returns the page as it now belongs here.
//
// Port of importPage(PDPage).
func (d *PDDocument) ImportPage(page *PDPage) (*PDPage, error) {
	// BEWARE: when making changes here, make sure that these changes don't mess with the code
	// in the splitter, and avoid making changes in the source document (as happened in PDFBOX-5809)
	importedPage := NewPDPageOfCache(cos.NewDictionaryFrom(page.Dictionary()), d.resourceCache)
	importedPage.Dictionary().RemoveItem(cos.Parent)
	contents, err := page.Contents()
	if err != nil {
		return nil, err
	}
	dest, err := common.NewPDStreamOfInput(d.document, contents, cos.FlateDecode)
	if err != nil {
		return nil, err
	}
	importedPage.SetContents(dest)
	// reset imported object keys to avoid overlapping object numbers
	importedPage.Dictionary().ResetImportedObjectKeys()
	d.AddPage(importedPage)
	importedPage.SetCropBox(common.NewPDRectangleOfCOSArray(page.CropBox().COSArray()))
	importedPage.SetMediaBox(common.NewPDRectangleOfCOSArray(page.MediaBox().COSArray()))
	importedPage.SetRotation(page.Rotation())
	if page.Resources() != nil && !page.Dictionary().ContainsKey(cos.Resources) {
		slog.Warn("pdmodel: inherited resources of source document are not imported to destination page")
		slog.Warn("pdmodel: call importedPage.SetResources(page.Resources()) to do this")
	}
	return importedPage, nil
}
