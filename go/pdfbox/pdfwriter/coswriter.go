package pdfwriter

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser/xref"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The tokens a PDF file is built out of.
//
// These are COSWriter's public byte array constants in Java. Their single
// definition sits in pdfwriter/compress, because COSWriterObjectStream needs
// them and this package imports that one; see compress/tokens.go. The names
// here are the Java names, in the Java place.
var (
	// DictOpen is the opening token of a dictionary.
	DictOpen = compress.DictOpen
	// DictClose is the closing token of a dictionary.
	DictClose = compress.DictClose
	// Space is a space.
	Space = compress.Space
	// Comment is the comment token.
	Comment = compress.Comment
	// Version is the version of the PDF this writer produces.
	Version = compress.Version
	// Garbage is the four high bytes of the second header line, which tell a
	// transfer program the file is binary.
	Garbage = compress.Garbage
	// EOF is the end of file token.
	EOF = compress.EOF
	// Reference is the token of an indirect reference.
	Reference = compress.Reference
	// XRef is the cross-reference table token.
	XRef = compress.XRef
	// XRefFree marks a free cross-reference entry.
	XRefFree = compress.XRefFree
	// XRefUsed marks a used cross-reference entry.
	XRefUsed = compress.XRefUsed
	// Trailer is the trailer token.
	Trailer = compress.Trailer
	// StartXRef is the token before the cross-reference offset.
	StartXRef = compress.StartXRef
	// Obj opens an indirect object.
	Obj = compress.Obj
	// EndObj closes one.
	EndObj = compress.EndObj
	// ArrayOpen is the opening token of an array.
	ArrayOpen = compress.ArrayOpen
	// ArrayClose is the closing token of an array.
	ArrayClose = compress.ArrayClose
	// Stream opens stream data.
	Stream = compress.Stream
	// EndStream closes it.
	EndStream = compress.EndStream
)

// WriteString writes a COS string, choosing the literal or the hexadecimal
// form.
//
// Port of the static COSWriter.writeString(COSString, OutputStream).
func WriteString(str *cos.StringObj, output io.Writer) error {
	return compress.WriteString(str, output)
}

// WriteStringBytes writes the given bytes as a literal PDF string.
//
// Port of the static COSWriter.writeString(byte[], OutputStream).
func WriteStringBytes(b []byte, output io.Writer) error {
	return compress.WriteStringBytes(b, output)
}

// PDDocumentLike is what the writer asks of the document it is writing.
//
// Java takes a PDDocument, which imports pdfwriter for its save methods; the
// port names what the writer needs instead, the way slice 5 broke the same
// cycle between the security handlers and PDDocument.
type PDDocumentLike interface {
	compress.DocumentLike
	encryption.PDDocumentLike

	// Version returns the document version. Port of getVersion().
	Version() float32

	// SetVersion sets the document version. Port of setVersion(float).
	SetVersion(version float32)

	// IsAllSecurityToBeRemoved reports whether the save is to drop the
	// encryption.
	IsAllSecurityToBeRemoved() bool

	// DocumentId returns the caller's fixed document ID, or nil where the
	// writer is to derive one. Java's field is a nullable Long.
	DocumentId() *int64
}

// SignatureInterface signs the bytes of an incremental update.
//
// Java's is org.apache.pdfbox.pdmodel.interactive.digitalsignature.
// SignatureInterface, which slice 8 brings with the rest of the signature
// model; this stands for it so that the writer's signature path can be ported
// now. See migration/STATUS.md.
type SignatureInterface interface {
	// Sign returns the CMS signature of the given content.
	Sign(content io.Reader) ([]byte, error)
}

// errSigningNeedsSlice8 is what the writer reports where Java would build the
// data to sign. getDataToSign needs COSFilterInputStream, which lives in
// pdmodel/interactive/digitalsignature and arrives with slice 8; an externally
// created signature can still be written with WriteExternalSignature.
var errSigningNeedsSlice8 = errors.New(
	"pdfwriter: signing through a SignatureInterface needs COSFilterInputStream, which slice 8 brings")

// COSWriter writes a COS document out as a PDF file.
//
// Port of org.apache.pdfbox.pdfwriter.COSWriter, which is the visitor that
// walks the object graph and emits it.
type COSWriter struct {
	output io.Writer

	// standardOutput is the stream everything is written through, which counts
	// the bytes so that the cross-reference table can point at them.
	standardOutput *standardOutputStream

	// startxref is the byte position of the cross-reference table.
	startxref int64

	// number is the highest object number written so far.
	number int64

	// objectKeys maps an object to the key it is written under, and keyObject
	// back the other way.
	//
	// Java uses a Hashtable and a HashMap keyed on COSBase, which does not
	// override equals, so both are identity maps; a Go map keyed on the
	// interface compares the pointer inside it, which is the same test.
	// keyObject is keyed on the key's internal hash, because COSObjectKey does
	// override equals.
	objectKeys map[cos.Base]*cos.ObjectKey
	keyObject  map[int64]cos.Base

	xRefEntries []xref.Entry

	// objectsToWrite is the queue of objects still to be written.
	objectsToWrite []cos.Base

	// writtenObjects is what has been written already.
	writtenObjects map[cos.Base]bool

	// actualsAdded holds the resolved object behind every queue entry, so that
	// an object reached through two different references is queued once.
	actualsAdded map[cos.Base]bool

	currentObjectKey *cos.ObjectKey
	pdDocument       PDDocumentLike
	willEncrypt      bool

	// incrementalUpdate and the fields below it drive an incremental save.
	incrementalUpdate  bool
	reachedSignature   bool
	signatureOffset    int64
	signatureLength    int64
	byteRangeOffset    int64
	byteRangeLength    int64
	incrementalInput   pdfio.RandomAccessRead
	incrementalOutput  io.Writer
	incrementPart      []byte
	byteRangeArray     *cos.Array
	signatureInterface SignatureInterface

	compressParameters *compress.Parameters
	blockAddingObject  bool
}

var _ cos.Visitor = (*COSWriter)(nil)

// NewCOSWriter returns a writer that writes to the given stream, without
// compressing objects into object streams.
//
// Port of COSWriter(OutputStream), which passes a null CompressParameters.
func NewCOSWriter(outputStream io.Writer) *COSWriter {
	return NewCOSWriterOfParameters(outputStream, nil)
}

// NewCOSWriterOfParameters returns a writer that writes to the given stream
// with the given compression.
//
// Port of COSWriter(OutputStream, CompressParameters).
func NewCOSWriterOfParameters(outputStream io.Writer,
	compressParameters *compress.Parameters) *COSWriter {
	w := newCOSWriter()
	w.output = outputStream
	w.standardOutput = newStandardOutputStream(outputStream)
	w.compressParameters = compressParameters
	return w
}

// NewCOSWriterIncremental returns a writer that appends an update to the file
// the given source holds.
//
// Port of COSWriter(OutputStream, RandomAccessRead). Java writes the update
// into a buffer and copies the original in front of it at the end, so that the
// original bytes are not touched; the port does the same.
func NewCOSWriterIncremental(outputStream io.Writer,
	inputData pdfio.RandomAccessRead) (*COSWriter, error) {
	length, err := inputData.Length()
	if err != nil {
		return nil, err
	}
	w := newCOSWriter()
	// write to buffer instead of output
	buffer := &bytes.Buffer{}
	w.output = buffer
	w.standardOutput = newStandardOutputStreamAt(buffer, length)
	// disable compressed object streams
	w.compressParameters = compress.NoCompression
	w.incrementalInput = inputData
	w.incrementalOutput = outputStream
	w.incrementalUpdate = true
	return w, nil
}

// NewCOSWriterIncrementalOfObjects returns an incremental writer that writes
// only the given dictionaries.
//
// Port of COSWriter(OutputStream, RandomAccessRead, Set<COSDictionary>).
//
// Implementation notes / summary of April 2019 comments in PDFBOX-45: we allow
// only COSDictionary in objectsToWrite because other types, especially
// COSArray, are written directly. If we'd allow them with the current COSWriter
// implementation, they would be written twice, once directly and once
// indirectly as orphan.
func NewCOSWriterIncrementalOfObjects(outputStream io.Writer, inputData pdfio.RandomAccessRead,
	objectsToWrite []*cos.Dictionary) (*COSWriter, error) {
	w, err := NewCOSWriterIncremental(outputStream, inputData)
	if err != nil {
		return nil, err
	}
	for _, object := range objectsToWrite {
		w.objectsToWrite = append(w.objectsToWrite, object)
	}
	return w, nil
}

func newCOSWriter() *COSWriter {
	return &COSWriter{
		objectKeys:     map[cos.Base]*cos.ObjectKey{},
		keyObject:      map[int64]cos.Base{},
		writtenObjects: map[cos.Base]bool{},
		actualsAdded:   map[cos.Base]bool{},
	}
}

// IsCompress reports whether the writer compresses objects into object streams.
func (w *COSWriter) IsCompress() bool {
	return w.compressParameters != nil && w.compressParameters.IsCompress()
}

// prepareIncrement records the keys the document was read under, so that an
// incremental save writes an object back under the number it already had.
func (w *COSWriter) prepareIncrement() {
	cosDoc := w.pdDocument.Document()
	for cosObjectKey := range cosDoc.XRefTable() {
		if cosObjectKey == nil {
			continue
		}
		object := cosDoc.ObjectFromPool(cosObjectKey).Object()
		if object == nil {
			continue
		}
		if _, isNumber := object.(cos.Number); isNumber {
			continue
		}
		// FIXME see PDFBOX-4997: objectKeys is (theoretically) risky because a
		// COSName in different objects would appear only once. Rev 1092855
		// considered this but only for COSNumber.
		w.objectKeys[object] = cosObjectKey
		w.keyObject[cosObjectKey.InternalHash()] = object
	}
}

func (w *COSWriter) addXRefEntry(entry xref.Entry) {
	w.xRefEntries = append(w.xRefEntries, entry)
}

// XRefEntries returns the cross-reference entries collected so far.
func (w *COSWriter) XRefEntries() []xref.Entry { return w.xRefEntries }

// StartXRef returns the byte position of the cross-reference table.
func (w *COSWriter) StartXRef() int64 { return w.startxref }

func (w *COSWriter) setStartXRef(newStartxref int64) { w.startxref = newStartxref }

// doWriteBody writes every object of the document.
func (w *COSWriter) doWriteBody(doc *cos.Document) error {
	trailer := doc.Trailer()
	// get the COSObjects to preserve the origin object numbers
	root := trailer.GetItem(cos.Root)
	info := trailer.GetItem(cos.Info)
	encrypt := trailer.GetItem(cos.Encrypt)
	if root != nil {
		w.addObjectToWrite(root)
	}
	if info != nil {
		w.addObjectToWrite(info)
	}
	if err := w.doWriteObjects(); err != nil {
		return err
	}

	w.willEncrypt = false
	if encrypt != nil {
		w.addObjectToWrite(encrypt)
	}
	return w.doWriteObjects()
}

// doWriteBodyCompressed writes the compressed body of the document.
func (w *COSWriter) doWriteBodyCompressed(document *cos.Document) error {
	trailer := document.Trailer()
	encrypt := trailer.GetCOSDictionary(cos.Encrypt)
	w.blockAddingObject = true
	w.willEncrypt = encrypt != nil
	if !trailer.ContainsKey(cos.Root) {
		return nil
	}

	compressionPool, err := compress.NewCompressionPool(w.pdDocument, w.compressParameters)
	if err != nil {
		return err
	}
	// Append object stream entries to document.
	for _, key := range compressionPool.ObjectStreamObjects() {
		object := compressionPool.Object(key)
		w.writtenObjects[object] = true
		w.objectKeys[object] = key
		w.keyObject[key.InternalHash()] = object
	}
	// Append top level objects to document.
	for _, key := range compressionPool.TopLevelObjects() {
		object := compressionPool.Object(key)
		w.writtenObjects[object] = true
		w.objectKeys[object] = key
		w.keyObject[key.InternalHash()] = object
	}
	w.number = compressionPool.HighestXRefObjectNumber()
	for _, key := range compressionPool.TopLevelObjects() {
		w.currentObjectKey = key
		if err := w.DoWriteObjectOfKey(key, w.keyObject[key.InternalHash()]); err != nil {
			return err
		}
	}
	// Append object streams to document.
	for _, finalizedObjectStream := range compressionPool.CreateObjectStreams() {
		// Create new COSObject for object stream.
		stream, err := finalizedObjectStream.WriteObjectsToStream(document.CreateStream())
		if err != nil {
			return err
		}
		// Determine key for object stream.
		w.number++
		objectStreamKey, err := cos.NewObjectKey(w.number, 0)
		if err != nil {
			return err
		}
		// Create new COSObject for object stream.
		objectStream := cos.NewObjectWithKey(stream, objectStreamKey)
		// Add object stream entries to xref - stream.
		for i, key := range finalizedObjectStream.PreparedKeys() {
			entry := xref.NewObjectStreamReference(key, objectStreamKey, i)
			w.addXRefEntry(&entry)
		}
		// Include object stream in document.
		w.currentObjectKey = objectStreamKey
		if err := w.DoWriteObjectOfKey(objectStreamKey, objectStream); err != nil {
			return err
		}
	}
	w.willEncrypt = false
	if encrypt != nil {
		w.number++
		encryptKey, err := cos.NewObjectKey(w.number, 0)
		if err != nil {
			return err
		}
		w.currentObjectKey = encryptKey
		w.writtenObjects[encrypt] = true
		w.keyObject[encryptKey.InternalHash()] = encrypt
		w.objectKeys[encrypt] = encryptKey

		if err := w.DoWriteObjectOfKey(encryptKey, encrypt); err != nil {
			return err
		}
	}
	w.blockAddingObject = false
	return nil
}

func (w *COSWriter) doWriteObjects() error {
	for len(w.objectsToWrite) > 0 {
		object := w.objectsToWrite[0]
		w.objectsToWrite = w.objectsToWrite[1:]
		if err := w.DoWriteObject(object); err != nil {
			return err
		}
	}
	return nil
}

func (w *COSWriter) addObjectToWrite(object cos.Base) {
	if w.blockAddingObject {
		return
	}
	actual := object
	if reference, ok := actual.(*cos.Object); ok {
		actual = reference.Object()
	}

	if w.writtenObjects[object] || (actual != nil && w.actualsAdded[actual]) ||
		slices.Contains(w.objectsToWrite, object) {
		return
	}

	if actual != nil {
		if cosObjectKey, ok := w.objectKeys[actual]; ok {
			cosBase := w.keyObject[cosObjectKey.InternalHash()]
			if !isNeedToBeUpdated(object) && !isNeedToBeUpdated(cosBase) {
				return
			}
		}
	}

	w.objectsToWrite = append(w.objectsToWrite, object)
	if actual != nil {
		w.actualsAdded[actual] = true
	}
}

// DoWriteObjectOfKey writes one object under the given key.
//
// Port of doWriteObject(COSObjectKey, COSBase).
func (w *COSWriter) DoWriteObjectOfKey(key *cos.ObjectKey, obj cos.Base) error {
	// don't write missing objects to avoid broken xref tables
	if obj == nil {
		return nil
	}
	if reference, ok := obj.(*cos.Object); ok && reference.Object() == nil {
		return nil
	}

	// add a x ref entry
	entry := xref.NewNormalReference(w.standardOutput.Pos(), key, obj)
	w.addXRefEntry(&entry)

	// write the object
	if err := w.writeAll(
		[]byte(strconv.FormatInt(key.Number(), 10)), Space,
		[]byte(strconv.Itoa(key.Generation())), Space, Obj); err != nil {
		return err
	}
	if err := w.standardOutput.writeEOL(); err != nil {
		return err
	}
	if err := obj.Accept(w); err != nil {
		return err
	}
	if err := w.standardOutput.writeEOL(); err != nil {
		return err
	}
	if _, err := w.standardOutput.Write(EndObj); err != nil {
		return err
	}
	return w.standardOutput.writeEOL()
}

// isNeedToBeUpdated reports whether an object has been changed since it was
// read, which is what an incremental save writes back.
func isNeedToBeUpdated(base cos.Base) bool {
	if info, ok := base.(cos.UpdateInfo); ok {
		return info.IsNeedToBeUpdated()
	}
	return false
}

// DoWriteObject writes one object, giving it a key first.
//
// Port of doWriteObject(COSBase).
func (w *COSWriter) DoWriteObject(obj cos.Base) error {
	w.writtenObjects[obj] = true
	// find the physical reference
	key, err := w.getObjectKey(obj)
	if err != nil {
		return err
	}
	w.currentObjectKey = key
	return w.DoWriteObjectOfKey(key, obj)
}

func (w *COSWriter) doWriteHeader(doc *cos.Document) error {
	if w.IsCompress() {
		w.pdDocument.SetVersion(max(w.pdDocument.Version(), compress.MinimumSupportedVersion))
		doc.SetVersion(max(doc.Version(), compress.MinimumSupportedVersion))
	}

	// Java writes "%FDF-" for an FDF document; FDF is not ported, so this is
	// always a PDF header. See migration/STATUS.md.
	headerString := "%PDF-" + formatVersion(doc.Version())

	if _, err := w.standardOutput.Write([]byte(headerString)); err != nil {
		return err
	}
	if err := w.standardOutput.writeEOL(); err != nil {
		return err
	}
	if err := w.writeAll(Comment, Garbage); err != nil {
		return err
	}
	return w.standardOutput.writeEOL()
}

// formatVersion renders the version the way Java's string concatenation of a
// float does: the shortest form that round-trips, always with a fraction part.
func formatVersion(version float32) string {
	s := strconv.FormatFloat(float64(version), 'f', -1, 32)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func (w *COSWriter) doWriteTrailer(doc *cos.Document) error {
	if _, err := w.standardOutput.Write(Trailer); err != nil {
		return err
	}
	if err := w.standardOutput.writeEOL(); err != nil {
		return err
	}

	trailer := doc.Trailer()
	// Only need to stay, if an incremental update will be performed
	if !w.incrementalUpdate {
		// sort xref, needed only if object keys not regenerated
		slices.SortStableFunc(w.xRefEntries, xref.Compare)
		lastEntry := w.xRefEntries[len(w.xRefEntries)-1]
		trailer.SetLong(cos.Size, lastEntry.ReferencedKey().Number()+1)
		trailer.RemoveItem(cos.Prev)
	}
	if !doc.IsXRefStream() {
		trailer.SetLong(cos.Size, w.number+1)
		trailer.RemoveItem(cos.XRefStm)
	}
	// Remove a checksum if present
	trailer.RemoveItem(cos.DocChecksum)

	if idArray := trailer.GetCOSArray(cos.ID); idArray != nil {
		idArray.SetDirect(true)
	}
	return trailer.Accept(w)
}

// doWriteXRefInc writes the cross-reference data of an incremental update, or
// of a document whose cross-reference data is a stream.
func (w *COSWriter) doWriteXRefInc(doc *cos.Document) error {
	if !doc.IsXRefStream() || (doc.HasHybridXRef() && w.incrementalUpdate) {
		trailer := doc.Trailer()
		trailer.SetLong(cos.Prev, doc.StartXref())
		if err := w.doWriteXRefTable(); err != nil {
			return err
		}
		return w.doWriteTrailer(doc)
	}

	// the file uses XrefStreams, so we need to update
	// it with an xref stream. We create a new one and fill it
	// with data available here

	// create a new XRefStream object
	pdfxRefStream := pdfparser.NewXRefStream(doc)

	// add all entries from the incremental update.
	for _, entry := range w.XRefEntries() {
		pdfxRefStream.AddEntry(entry)
	}

	trailer := doc.Trailer()
	if w.incrementalUpdate {
		// use previous startXref value as new PREV value
		trailer.SetLong(cos.Prev, doc.StartXref())
	} else {
		trailer.RemoveItem(cos.Prev)
	}
	pdfxRefStream.AddTrailerInfo(trailer)
	// Pre-assign the object key for the xref stream so it can be
	// included in its own cross-reference data. Per PDF Reference
	// §7.5.8, the /Size value must be one greater than the highest
	// object number in the file, including the xref stream itself.
	w.number++
	xrefStreamKey, err := cos.NewObjectKey(w.number, 0)
	if err != nil {
		return err
	}
	xrefStreamOffset := w.standardOutput.Pos()
	w.setStartXRef(xrefStreamOffset)
	entry := xref.NewNormalReference(xrefStreamOffset, xrefStreamKey, nil)
	pdfxRefStream.AddEntry(&entry)
	pdfxRefStream.SetSize(w.number + 1)

	xrefStream, err := pdfxRefStream.Stream()
	if err != nil {
		return err
	}
	return w.DoWriteObjectOfKey(xrefStreamKey, xrefStream)
}

// doWriteXRefTable writes the "xref" table.
func (w *COSWriter) doWriteXRefTable() error {
	if !w.incrementalUpdate {
		// fill gaps with free entries
		w.fillGapsWithFreeEntries()
	} else {
		// add free entry with object number 0
		w.addXRefEntry(xref.NullEntry)
	}

	// Filter for NormalXReferences and FreeXReferences
	// sort xref, needed only if object keys not regenerated
	tmpXRefEntries := make([]xref.Entry, 0, len(w.xRefEntries))
	for _, e := range w.xRefEntries {
		switch e.Type() {
		case xref.TypeNormal, xref.TypeFree:
			tmpXRefEntries = append(tmpXRefEntries, e)
		}
	}
	slices.SortStableFunc(tmpXRefEntries, xref.Compare)

	// remember the position where x ref was written
	w.setStartXRef(w.standardOutput.Pos())
	if _, err := w.standardOutput.Write(XRef); err != nil {
		return err
	}
	if err := w.standardOutput.writeEOL(); err != nil {
		return err
	}

	// write start object number and object count for this x ref section
	// we assume starting from scratch
	xRefRanges := getXRefRanges(tmpXRefEntries)
	xRefLength := len(xRefRanges)
	x := 0
	j := 0
	if xRefLength%2 == 0 {
		for x < xRefLength {
			xRefRangeX1 := xRefRanges[x+1]
			if err := w.writeXRefRange(xRefRanges[x], xRefRangeX1); err != nil {
				return err
			}
			for i := int64(0); i < xRefRangeX1; i++ {
				if err := w.writeXRefEntry(tmpXRefEntries[j]); err != nil {
					return err
				}
				j++
			}
			x += 2
		}
	}
	return nil
}

func (w *COSWriter) fillGapsWithFreeEntries() {
	normalXReferences := make([]xref.Entry, 0, len(w.xRefEntries))
	for _, e := range w.xRefEntries {
		if e.Type() == xref.TypeNormal {
			normalXReferences = append(normalXReferences, e)
		}
	}
	slices.SortStableFunc(normalXReferences, xref.Compare)

	last := int64(0)
	var freeNumbers []int64
	for _, entry := range normalXReferences {
		nr := entry.ReferencedKey().Number()
		if nr != last {
			for i := last; i < nr; i++ {
				freeNumbers = append(freeNumbers, i)
			}
		}
		last = nr + 1
	}

	numberOfFreeNumbers := len(freeNumbers)
	if numberOfFreeNumbers == 0 {
		// no gaps found -> add free entry with object number 0
		w.addXRefEntry(xref.NullEntry)
		return
	}

	// add free entries for all but the last one
	for i := 0; i < numberOfFreeNumbers-1; i++ {
		entry := xref.NewFreeReference(mustObjectKey(freeNumbers[i], 65535), freeNumbers[i+1])
		w.addXRefEntry(&entry)
	}
	// add free entry for the last one referencing object 0 as next free one
	entry := xref.NewFreeReference(
		mustObjectKey(freeNumbers[numberOfFreeNumbers-1], 65535), 0)
	w.addXRefEntry(&entry)

	firstObjectNumber := freeNumbers[0]
	// add free entry for object number 0 if not already present
	if firstObjectNumber > 0 {
		zero := xref.NewFreeReference(mustObjectKey(0, 65535), firstObjectNumber)
		w.addXRefEntry(&zero)
	}
}

// mustObjectKey builds a key that cannot fail, which is what Java's
// `new COSObjectKey(n, 65535)` is.
func mustObjectKey(number int64, generation int) *cos.ObjectKey {
	key, err := cos.NewObjectKey(number, generation)
	if err != nil {
		panic(err)
	}
	return key
}

// doWriteIncrement writes the original file and then the update appended to it.
func (w *COSWriter) doWriteIncrement() error {
	// write existing PDF
	if err := pdfio.SeekTo(w.incrementalInput, 0); err != nil {
		return err
	}
	if _, err := io.Copy(w.incrementalOutput, pdfio.NewReader(w.incrementalInput)); err != nil {
		return err
	}
	// write the actual incremental update
	buffer, ok := w.output.(*bytes.Buffer)
	if !ok {
		return errors.New("pdfwriter: an incremental writer must buffer its output")
	}
	_, err := w.incrementalOutput.Write(buffer.Bytes())
	return err
}

// doWriteSignature fills in the /ByteRange the signature dictionary reserved,
// and signs the update where a SignatureInterface was given.
func (w *COSWriter) doWriteSignature() error {
	// calculate the ByteRange values
	inLength, err := w.incrementalInput.Length()
	if err != nil {
		return err
	}
	beforeLength := w.signatureOffset
	afterOffset := w.signatureOffset + w.signatureLength
	afterLength := w.standardOutput.Pos() - (inLength + w.signatureLength) -
		(w.signatureOffset - inLength)

	byteRange := "0 " + strconv.FormatInt(beforeLength, 10) + " " +
		strconv.FormatInt(afterOffset, 10) + " " + strconv.FormatInt(afterLength, 10) + "]"

	// Assign the values to the actual COSArray, so that the user can access it before closing
	w.byteRangeArray.Set(0, cos.GetInteger(0))
	w.byteRangeArray.Set(1, cos.GetInteger(beforeLength))
	w.byteRangeArray.Set(2, cos.GetInteger(afterOffset))
	w.byteRangeArray.Set(3, cos.GetInteger(afterLength))

	if int64(len(byteRange)) > w.byteRangeLength {
		return fmt.Errorf("Can't write new byteRange '%s' not enough space: byteRange.length(): %d,"+
			" byteRangeLength: %d, byteRangeOffset: %d, inLength: %d",
			byteRange, len(byteRange), w.byteRangeLength, w.byteRangeOffset, inLength)
	}

	// copy the new incremental data into a buffer (e.g. signature dict, trailer)
	byteOut, ok := w.output.(*bytes.Buffer)
	if !ok {
		return errors.New("pdfwriter: an incremental writer must buffer its output")
	}
	w.incrementPart = slices.Clone(byteOut.Bytes())

	// overwrite the reserve ByteRange in the buffer
	byteRangeBytes := []byte(byteRange)
	for i := int64(0); i < w.byteRangeLength; i++ {
		if i >= int64(len(byteRangeBytes)) {
			w.incrementPart[w.byteRangeOffset+i-inLength] = 0x20 // SPACE
		} else {
			w.incrementPart[w.byteRangeOffset+i-inLength] = byteRangeBytes[i]
		}
	}

	if w.signatureInterface != nil {
		// data to be signed
		//
		// Java builds it with COSFilterInputStream, which slice 8 brings; until
		// then a signature has to be made externally and written with
		// WriteExternalSignature.
		return errSigningNeedsSlice8
	}
	// else signature should be created externally and set via writeSignature()
	return nil
}

// WriteExternalSignature writes an externally created signature of the PDF data
// into the space the signature dictionary reserved.
func (w *COSWriter) WriteExternalSignature(cmsSignature []byte) error {
	if w.incrementPart == nil || w.incrementalInput == nil {
		// Java throws IllegalStateException, which is unchecked.
		panic("PDF not prepared for setting signature")
	}
	// org.apache.pdfbox.util.Hex.getBytes
	const digits = "0123456789ABCDEF"
	signatureBytes := make([]byte, 0, 2*len(cmsSignature))
	for _, b := range cmsSignature {
		signatureBytes = append(signatureBytes, digits[b>>4], digits[b&0x0F])
	}

	// subtract 2 bytes because of the enclosing "<>"
	if int64(len(signatureBytes)) > w.signatureLength-2 {
		return errors.New("Can't write signature, not enough space; " +
			"adjust it with SignatureOptions.setPreferredSignatureSize")
	}

	// overwrite the signature Contents in the buffer
	inLength, err := w.incrementalInput.Length()
	if err != nil {
		return err
	}
	incPartSigOffset := w.signatureOffset - inLength
	copy(w.incrementPart[incPartSigOffset+1:], signatureBytes)

	// write the data to the incremental output stream
	if err := pdfio.SeekTo(w.incrementalInput, 0); err != nil {
		return err
	}
	if _, err := io.Copy(w.incrementalOutput, pdfio.NewReader(w.incrementalInput)); err != nil {
		return err
	}
	if _, err := w.incrementalOutput.Write(w.incrementPart); err != nil {
		return err
	}

	// prevent further use
	w.incrementPart = nil
	return nil
}

func (w *COSWriter) writeXRefRange(x, y int64) error {
	if err := w.writeAll([]byte(strconv.FormatInt(x, 10)), Space,
		[]byte(strconv.FormatInt(y, 10))); err != nil {
		return err
	}
	return w.standardOutput.writeEOL()
}

func (w *COSWriter) writeXRefEntry(entry xref.Entry) error {
	offset := fmt.Sprintf("%010d", entry.SecondColumnValue())
	generation := fmt.Sprintf("%05d", entry.ThirdColumnValue())
	marker := XRefUsed
	if entry.Type() == xref.TypeFree {
		marker = XRefFree
	}
	if err := w.writeAll([]byte(offset), Space, []byte(generation), Space, marker); err != nil {
		return err
	}
	return w.standardOutput.writeCRLF()
}

// getXRefRanges returns the contiguous runs of object numbers, as pairs of a
// first number and a count.
//
// Port of getXRefRanges.
func getXRefRanges(xRefEntriesList []xref.Entry) []int64 {
	last := int64(-2)
	count := int64(1)

	var list []int64
	for _, entry := range xRefEntriesList {
		nr := entry.ReferencedKey().Number()
		if nr == last+1 {
			count++
			last = nr
		} else if last == -2 {
			last = nr
		} else {
			list = append(list, last-count+1, count)
			last = nr
			count = 1
		}
	}
	// If no new entry is found, we need to write out the last result
	if len(xRefEntriesList) > 0 {
		list = append(list, last-count+1, count)
	}
	return list
}

// getObjectKey returns the key an object is written under, giving it a new one
// where it has none.
func (w *COSWriter) getObjectKey(obj cos.Base) (*cos.ObjectKey, error) {
	key := obj.Key()
	var actual cos.Base
	if reference, ok := obj.(*cos.Object); ok {
		actual = reference.Object()
		if actual == nil {
			// the referenced object isn't there due to a malformed pdf
			// check if a key is present, otherwise create a new one
			if key == nil {
				w.number++
				var err error
				if key, err = cos.NewObjectKey(w.number, 0); err != nil {
					return nil, err
				}
			}
			w.objectKeys[obj] = key
			return key, nil
		}
	} else {
		actual = obj
	}

	actualKey, ok := w.objectKeys[actual]
	if !ok {
		w.number++
		var err error
		if actualKey, err = cos.NewObjectKey(w.number, 0); err != nil {
			return nil, err
		}
		w.objectKeys[actual] = actualKey
	}

	// check if the returned key and the origin key of the given object are the same
	if key == nil || !key.Equals(actualKey) {
		// update the object key given object/referenced object
		key = actualKey
		actual.SetKey(actualKey)
		if _, isReference := obj.(*cos.Object); isReference {
			// update the object key of the indirect object
			obj.SetKey(key)
			w.objectKeys[obj] = key
		}
	}
	return key, nil
}

// writeAll writes each of the given byte slices in turn.
func (w *COSWriter) writeAll(parts ...[]byte) error {
	for _, part := range parts {
		if _, err := w.standardOutput.Write(part); err != nil {
			return err
		}
	}
	return nil
}

// VisitArray writes an array.
func (w *COSWriter) VisitArray(array *cos.Array) error {
	count := 0
	if _, err := w.standardOutput.Write(ArrayOpen); err != nil {
		return err
	}
	for i := 0; i < array.Size(); i++ {
		current := array.Get(i)
		switch value := current.(type) {
		case *cos.Stream:
			// COSStream is a COSDictionary in Java, so it takes the same branch.
			if err := w.writeDictionary(value, &value.Dictionary); err != nil {
				return err
			}
		case *cos.Dictionary:
			if err := w.writeDictionary(value, value); err != nil {
				return err
			}
		case *cos.Array:
			if err := w.writeArray(value); err != nil {
				return err
			}
		case *cos.Object:
			w.addObjectToWrite(current)
			if err := w.WriteReference(current); err != nil {
				return err
			}
		case nil:
			if err := cos.NullObject.Accept(w); err != nil {
				return err
			}
		default:
			if err := current.Accept(w); err != nil {
				return err
			}
		}
		count++
		if i+1 < array.Size() {
			if count%10 == 0 {
				if err := w.standardOutput.writeEOL(); err != nil {
					return err
				}
			} else if _, err := w.standardOutput.Write(Space); err != nil {
				return err
			}
		}
	}
	if _, err := w.standardOutput.Write(ArrayClose); err != nil {
		return err
	}
	return w.standardOutput.writeEOL()
}

func (w *COSWriter) writeArray(array *cos.Array) error {
	if array.IsDirect() {
		return w.VisitArray(array)
	}
	w.addObjectToWrite(array)
	return w.WriteReference(array)
}

// writeDictionary is Java's writeDictionary(COSDictionary): object is what is
// queued and referenced, which for a stream is the stream, and dictionary is
// the entries written inline.
func (w *COSWriter) writeDictionary(object cos.Base, dictionary *cos.Dictionary) error {
	if object.IsDirect() {
		return w.VisitDictionary(dictionary)
	}
	w.addObjectToWrite(object)
	return w.WriteReference(object)
}

// VisitBoolean writes a boolean.
func (w *COSWriter) VisitBoolean(obj *cos.Boolean) error {
	return obj.WritePDF(w.standardOutput)
}

// VisitDictionary writes a dictionary.
func (w *COSWriter) VisitDictionary(obj *cos.Dictionary) error {
	if err := w.detectPossibleSignature(obj); err != nil {
		return err
	}
	if _, err := w.standardOutput.Write(DictOpen); err != nil {
		return err
	}
	if err := w.standardOutput.writeEOL(); err != nil {
		return err
	}

	for _, key := range obj.KeySet() {
		value := obj.GetItem(key)
		if value == nil {
			// then we won't write anything, there are a couple cases
			// were the value of an entry in the COSDictionary will
			// be a dangling reference that points to nothing
			// so we will just not write out the entry if that is the case
			continue
		}
		if err := key.Accept(w); err != nil {
			return err
		}
		if _, err := w.standardOutput.Write(Space); err != nil {
			return err
		}

		switch value := value.(type) {
		case *cos.Stream:
			// COSStream is a COSDictionary in Java, so it takes the same branch.
			w.markDirectXObjectAndResources(&value.Dictionary, key)
			if err := w.writeDictionary(value, &value.Dictionary); err != nil {
				return err
			}

		case *cos.Dictionary:
			w.markDirectXObjectAndResources(value, key)
			if err := w.writeDictionary(value, value); err != nil {
				return err
			}

		case *cos.Object:
			w.addObjectToWrite(value)
			if err := w.WriteReference(value); err != nil {
				return err
			}

		default:
			// If we reach the pdf signature, we need to determinate the
			// position of the content and byterange
			switch {
			case w.reachedSignature && key == cos.Contents:
				w.signatureOffset = w.standardOutput.Pos()
				if err := value.Accept(w); err != nil {
					return err
				}
				w.signatureLength = w.standardOutput.Pos() - w.signatureOffset

			case w.reachedSignature && key == cos.ByteRange:
				// Java casts without a check; the port asserts the same way and
				// panics where the entry is not an array.
				w.byteRangeArray = value.(*cos.Array)
				w.byteRangeOffset = w.standardOutput.Pos() + 1
				if err := value.Accept(w); err != nil {
					return err
				}
				w.byteRangeLength = w.standardOutput.Pos() - 1 - w.byteRangeOffset
				w.reachedSignature = false

			default:
				if array, isArray := value.(*cos.Array); isArray {
					if err := w.writeArray(array); err != nil {
						return err
					}
				} else if err := value.Accept(w); err != nil {
					return err
				}
			}
		}
		if err := w.standardOutput.writeEOL(); err != nil {
			return err
		}
	}

	if _, err := w.standardOutput.Write(DictClose); err != nil {
		return err
	}
	return w.standardOutput.writeEOL()
}

// markDirectXObjectAndResources is the non-incremental branch of
// visitFromDictionary's COSDictionary case.
func (w *COSWriter) markDirectXObjectAndResources(value *cos.Dictionary, key *cos.Name) {
	if w.incrementalUpdate {
		return
	}
	// write all XObjects as direct objects, this will save some size
	// PDFBOX-3684: but avoid dictionary that references itself
	if item := value.GetItem(cos.XObject); item != nil && key != cos.XObject {
		item.SetDirect(true)
	}
	if item := value.GetItem(cos.Resources); item != nil && key != cos.Resources {
		item.SetDirect(true)
	}
}

// detectPossibleSignature notices the signature dictionary of an incremental
// save, whose /Contents and /ByteRange the writer must record the position of.
func (w *COSWriter) detectPossibleSignature(obj *cos.Dictionary) error {
	if w.reachedSignature || !w.incrementalUpdate {
		return nil
	}
	itemType := obj.GetItem(cos.Type)
	if itemType != cos.Base(cos.Sig) && itemType != cos.Base(cos.DocTimeStamp) {
		return nil
	}
	byteRange := obj.GetCOSArray(cos.ByteRange)
	if byteRange == nil || byteRange.Size() != 4 {
		return nil
	}
	base2, ok := byteRange.Get(2).(*cos.Integer)
	if !ok {
		return nil
	}
	// PDFBOX-5521 avoid hitting "old" signatures
	length, err := w.incrementalInput.Length()
	if err != nil {
		return err
	}
	if base2.LongValue() > length {
		slog.Debug("pdfwriter: reached signature",
			"offset", w.standardOutput.Pos(), "byteRange", byteRange, "inputLength", length)
		w.reachedSignature = true
	}
	return nil
}

// VisitDocument writes the whole document.
func (w *COSWriter) VisitDocument(doc *cos.Document) error {
	if !w.incrementalUpdate {
		if err := w.doWriteHeader(doc); err != nil {
			return err
		}
	} else {
		// Sometimes the original file will be missing a newline at the end
		// In order to avoid having %%EOF the first object on the same line
		// as the %%EOF, we put a newline here. If there's already one at
		// the end of the file, an extra one won't hurt. PDFBOX-1051
		if err := w.standardOutput.writeCRLF(); err != nil {
			return err
		}
	}

	if w.IsCompress() {
		if err := w.doWriteBodyCompressed(doc); err != nil {
			return err
		}
	} else if err := w.doWriteBody(doc); err != nil {
		return err
	}

	if w.incrementalUpdate || doc.IsXRefStream() {
		if err := w.doWriteXRefInc(doc); err != nil {
			return err
		}
	} else {
		if err := w.doWriteXRefTable(); err != nil {
			return err
		}
		if err := w.doWriteTrailer(doc); err != nil {
			return err
		}
	}

	// write endof
	if _, err := w.standardOutput.Write(StartXRef); err != nil {
		return err
	}
	if err := w.standardOutput.writeEOL(); err != nil {
		return err
	}
	if _, err := w.standardOutput.Write(
		[]byte(strconv.FormatInt(w.StartXRef(), 10))); err != nil {
		return err
	}
	if err := w.standardOutput.writeEOL(); err != nil {
		return err
	}
	if _, err := w.standardOutput.Write(EOF); err != nil {
		return err
	}
	if err := w.standardOutput.writeEOL(); err != nil {
		return err
	}

	if w.incrementalUpdate {
		if w.signatureOffset == 0 || w.byteRangeOffset == 0 {
			return w.doWriteIncrement()
		}
		return w.doWriteSignature()
	}
	return nil
}

// VisitFloat writes a real.
func (w *COSWriter) VisitFloat(obj *cos.Float) error { return obj.WritePDF(w.standardOutput) }

// VisitInteger writes an integer.
func (w *COSWriter) VisitInteger(obj *cos.Integer) error { return obj.WritePDF(w.standardOutput) }

// VisitName writes a name.
func (w *COSWriter) VisitName(obj *cos.Name) error { return obj.WritePDF(w.standardOutput) }

// VisitNull writes the null token.
func (w *COSWriter) VisitNull(obj *cos.Null) error { return obj.WritePDF(w.standardOutput) }

// WriteReference writes an indirect reference to the given object.
func (w *COSWriter) WriteReference(obj cos.Base) error {
	key, err := w.getObjectKey(obj)
	if err != nil {
		return err
	}
	return w.writeAll(
		[]byte(strconv.FormatInt(key.Number(), 10)), Space,
		[]byte(strconv.Itoa(key.Generation())), Space, Reference)
}

// VisitStream writes a stream: its dictionary, then its data as stored.
func (w *COSWriter) VisitStream(obj *cos.Stream) error {
	if w.willEncrypt {
		handler, err := w.securityHandler()
		if err != nil {
			return err
		}
		if err := handler.EncryptStream(obj, w.currentObjectKey.Number(),
			w.currentObjectKey.Generation()); err != nil {
			return err
		}
	}

	// write the stream content
	if err := w.VisitDictionary(&obj.Dictionary); err != nil {
		return err
	}
	if _, err := w.standardOutput.Write(Stream); err != nil {
		return err
	}
	if err := w.standardOutput.writeCRLF(); err != nil {
		return err
	}
	if obj.HasData() {
		input, err := obj.CreateRawReader()
		if err != nil {
			return err
		}
		if _, err := io.Copy(w.standardOutput, input); err != nil {
			return err
		}
	}
	if err := w.standardOutput.writeCRLF(); err != nil {
		return err
	}
	if _, err := w.standardOutput.Write(EndStream); err != nil {
		return err
	}
	return w.standardOutput.writeEOL()
}

// VisitStringObj writes a string.
func (w *COSWriter) VisitStringObj(obj *cos.StringObj) error {
	if !w.willEncrypt {
		return WriteString(obj, w.standardOutput)
	}
	handler, err := w.securityHandler()
	if err != nil {
		return err
	}
	encrypted, err := handler.EncryptString(obj, w.currentObjectKey.Number(),
		w.currentObjectKey.Generation())
	if err != nil {
		return err
	}
	// Java casts the result to COSString without a check; the port asserts the
	// same way and panics where the handler returned something else.
	return WriteString(encrypted.(*cos.StringObj), w.standardOutput)
}

// securityHandlerLike is the half of a security handler the writer uses.
type securityHandlerLike interface {
	EncryptStream(stream *cos.Stream, objNum int64, genNum int) error
	EncryptString(str *cos.StringObj, objNum int64, genNum int) (cos.Base, error)
}

// securityHandler is pdDocument.getEncryption().getSecurityHandler(), which the
// two encrypting visit methods reach for.
func (w *COSWriter) securityHandler() (securityHandlerLike, error) {
	return w.pdDocument.Encryption().SecurityHandler()
}

// VisitObject writes what an indirect reference points at.
func (w *COSWriter) VisitObject(obj *cos.Object) error {
	base := obj.Object()
	if base == nil {
		return w.VisitNull(cos.NullObject)
	}
	return base.Accept(w)
}

// Write writes the given document out.
//
// Port of write(PDDocument).
func (w *COSWriter) Write(doc PDDocumentLike) error {
	return w.WriteSigned(doc, nil)
}

// WriteSigned writes the pdf document. If signature should be created
// externally, WriteExternalSignature should be invoked to set signature after
// calling this method.
//
// Port of write(PDDocument, SignatureInterface).
func (w *COSWriter) WriteSigned(doc PDDocumentLike, signInterface SignatureInterface) error {
	w.pdDocument = doc
	cosDoc := w.pdDocument.Document()
	trailer := cosDoc.Trailer()
	if w.incrementalUpdate {
		for _, base := range trailer.ToIncrement().Exclude(trailer).Objects() {
			w.objectsToWrite = append(w.objectsToWrite, base)
			if reference, ok := base.(*cos.Object); ok {
				w.actualsAdded[reference.Object()] = true
			} else {
				w.actualsAdded[base] = true
			}
		}
	}
	w.signatureInterface = signInterface
	w.number = w.pdDocument.Document().HighestXRefObjectNumber()
	if w.incrementalUpdate {
		w.prepareIncrement()
	}
	var idTime int64
	if documentID := w.pdDocument.DocumentId(); documentID == nil {
		idTime = time.Now().UnixMilli()
	} else {
		idTime = *documentID
	}

	// if the document says we should remove encryption, then we shouldn't encrypt
	if doc.IsAllSecurityToBeRemoved() {
		w.willEncrypt = false
		// also need to get rid of the "Encrypt" in the trailer so readers
		// don't try to decrypt a document which is not encrypted
		trailer.RemoveItem(cos.Encrypt)
	} else {
		if w.pdDocument.Encryption() != nil {
			if !w.incrementalUpdate {
				securityHandler, err := w.pdDocument.Encryption().SecurityHandler()
				if err != nil {
					return err
				}
				if !securityHandler.HasProtectionPolicy() {
					// Java throws IllegalStateException, which is unchecked.
					panic("PDF contains an encryption dictionary, please remove it with " +
						"setAllSecurityToBeRemoved() or set a protection policy with protect()")
				}
				if err := securityHandler.PrepareDocumentForEncryption(doc); err != nil {
					return err
				}
			}
			w.willEncrypt = true
		} else {
			w.willEncrypt = false
		}
	}

	var idArray *cos.Array
	missingID := true
	if array, ok := trailer.GetDictionaryObject(cos.ID).(*cos.Array); ok {
		idArray = array
		if idArray.Size() == 2 {
			missingID = false
		}
	} else {
		idArray = cos.NewArray()
	}
	if missingID || w.incrementalUpdate {
		digest := sha256.New()

		// algorithm says to use time/path/size/values in doc to generate the id.
		// we don't have path or size, so do the best we can
		digest.Write([]byte(strconv.FormatInt(idTime, 10)))

		if info := trailer.GetCOSDictionary(cos.Info); info != nil {
			for _, cosBase := range info.Values() {
				digest.Write([]byte(baseString(cosBase)))
			}
		}
		// reuse origin documentID if available as first value
		var firstID, secondID *cos.StringObj
		if missingID {
			firstID = cos.NewStringObjBytes(digest.Sum(nil))
			// it's ok to use the same ID for the second part if the ID is
			// created for the first time
			secondID = firstID
		} else {
			firstID = idArray.Get(0).(*cos.StringObj)
			secondID = cos.NewStringObjBytes(digest.Sum(nil))
		}
		idArray = cos.NewArray()
		idArray.Add(firstID)
		idArray.Add(secondID)
		trailer.SetItem(cos.ID, idArray)
	}
	if err := cosDoc.Accept(w); err != nil {
		return err
	}
	if !w.incrementalUpdate {
		cosDoc.SetHighestXRefObjectNumber(w.number)
	}
	return nil
}

// baseString is Java's COSBase.toString, which the document ID digest feeds on.
func baseString(base cos.Base) string {
	if s, ok := base.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%v", base)
}
