package cos

import (
	"log/slog"
	"sort"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Document is the COS-level view of a PDF file: its trailer, its
// cross-reference table, and the pool of objects read from it.
//
// Port of org.apache.pdfbox.cos.COSDocument.
//
// Not yet ported: the COSDocumentState this carries in Java, which tracks
// whether the document has been changed for an incremental save. That is slice
// 7 work; see migration/STATUS.md.
type Document struct {
	object

	version float32
	trailer *Dictionary

	// objectPool holds one proxy per object key, so that a forward reference
	// resolved later is seen by everyone holding it. Keyed by the packed
	// number and generation rather than by *ObjectKey, since two equal keys
	// are distinct pointers — this is what Java gets from COSObjectKey
	// overriding equals and hashCode.
	objectPool map[int64]*Object

	// xrefTable maps an object key to its byte offset in the file, keyed the
	// same way, keeping the first key object seen for each entry.
	xrefTable map[int64]*xrefEntry

	// streams created by this document rather than read from a file, so that
	// they can be closed with it.
	streams []*Stream

	streamCache pdfio.StreamCache
	codecs      CodecProvider
	parser      Parser

	isDecrypted             bool
	startXref               int64
	isXRefStream            bool
	hasHybridXRef           bool
	highestXRefObjectNumber int64
	// nilKeyEntry holds the offset of an entry whose key was nil. Java keeps
	// such an entry in its map; the port cannot key the internal map by nil,
	// so it is held separately and re-joined by XRefTable.
	nilKeyEntry *int64
	closed      bool
}

type xrefEntry struct {
	key    *ObjectKey
	offset int64
}

var _ Base = (*Document)(nil)

// NewDocument returns an empty document.
//
// Port of the COSDocument() and COSDocument(ICOSParser) constructors. parser may
// be nil for a document being built rather than read.
func NewDocument(parser Parser) *Document {
	return NewDocumentWithCache(nil, nil, parser)
}

// NewDocumentWithCache returns an empty document that allocates stream buffers
// from the given cache and resolves filters through the given provider.
//
// Port of COSDocument(StreamCacheCreateFunction, ICOSParser). Java takes a
// factory function and calls it, falling back to a memory-only cache and
// logging if it fails; the port takes the cache itself, so there is nothing to
// fail here and no fallback to log.
func NewDocumentWithCache(cache pdfio.StreamCache, codecs CodecProvider, parser Parser) *Document {
	return &Document{
		version:     1.4,
		objectPool:  make(map[int64]*Object),
		xrefTable:   make(map[int64]*xrefEntry),
		streamCache: cache,
		codecs:      codecs,
		parser:      parser,
	}
}

// CreateStream returns a new stream owned by this document, which closes it.
//
// Port of createCOSStream().
func (d *Document) CreateStream() *Stream {
	s := NewStreamWithCache(d.streamCache, d.codecs)
	// Only streams created here are tracked; those read from a file are
	// reached through the object pool and closed with it.
	d.streams = append(d.streams, s)
	return s
}

// CreateStreamFromFile returns a stream whose data is read from the file this
// document was parsed from, carrying over the entries of dictionary.
//
// Port of createCOSStream(COSDictionary, long, long).
func (d *Document) CreateStreamFromFile(dictionary *Dictionary, startPosition, streamLength int64) (*Stream, error) {
	view, err := d.parser.CreateRandomAccessReadView(startPosition, streamLength)
	if err != nil {
		return nil, err
	}
	s, err := NewStreamFromView(d.streamCache, view, d.codecs)
	if err != nil {
		return nil, err
	}
	dictionary.All(func(k *Name, v Base) bool {
		s.SetItem(k, v)
		return true
	})
	s.SetKey(dictionary.Key())
	return s, nil
}

// Version returns the PDF version from the file header.
func (d *Document) Version() float32 { return d.version }

// SetVersion records the PDF version.
func (d *Document) SetVersion(version float32) { d.version = version }

// Trailer returns the trailer dictionary.
func (d *Document) Trailer() *Dictionary { return d.trailer }

// SetTrailer records the trailer dictionary.
func (d *Document) SetTrailer(trailer *Dictionary) { d.trailer = trailer }

// IsDecrypted reports whether the document has been decrypted.
func (d *Document) IsDecrypted() bool { return d.isDecrypted }

// SetDecrypted marks the document as decrypted.
func (d *Document) SetDecrypted() { d.isDecrypted = true }

// IsEncrypted reports whether the trailer names an encryption dictionary.
func (d *Document) IsEncrypted() bool {
	return d.trailer != nil && d.trailer.GetItem(Encrypt) != nil
}

// EncryptionDictionary returns the /Encrypt dictionary, or nil.
func (d *Document) EncryptionDictionary() *Dictionary {
	if d.trailer == nil {
		return nil
	}
	return d.trailer.GetCOSDictionary(Encrypt)
}

// SetEncryptionDictionary records the /Encrypt dictionary in the trailer.
func (d *Document) SetEncryptionDictionary(enc *Dictionary) {
	if d.trailer != nil {
		d.trailer.SetItem(Encrypt, enc)
	}
}

// DocumentID returns the /ID array from the trailer, or nil.
func (d *Document) DocumentID() *Array {
	if d.trailer == nil {
		return nil
	}
	return d.trailer.GetCOSArray(ID)
}

// SetDocumentID records the /ID array in the trailer.
func (d *Document) SetDocumentID(id *Array) {
	if d.trailer != nil {
		d.trailer.SetItem(ID, id)
	}
}

// HighestXRefObjectNumber returns the largest object number seen in the
// cross-reference data.
func (d *Document) HighestXRefObjectNumber() int64 { return d.highestXRefObjectNumber }

// SetHighestXRefObjectNumber records it.
func (d *Document) SetHighestXRefObjectNumber(n int64) { d.highestXRefObjectNumber = n }

// StartXref returns the byte offset of the cross-reference section.
func (d *Document) StartXref() int64 { return d.startXref }

// SetStartXref records it.
func (d *Document) SetStartXref(offset int64) { d.startXref = offset }

// IsXRefStream reports whether the cross-reference data is a stream rather than
// a table.
func (d *Document) IsXRefStream() bool { return d.isXRefStream }

// SetIsXRefStream records it.
func (d *Document) SetIsXRefStream(v bool) { d.isXRefStream = v }

// HasHybridXRef reports whether the file carries both a table and a stream.
func (d *Document) HasHybridXRef() bool { return d.hasHybridXRef }

// SetHasHybridXRef records it.
func (d *Document) SetHasHybridXRef() { d.hasHybridXRef = true }

// ObjectFromPool returns the proxy for an object key, creating it on first use.
//
// Port of getObjectFromPool. The proxy is created before the object it names
// has been read, which is how a forward reference is represented; resolving it
// later updates every holder, because they all share this one value.
func (d *Document) ObjectFromPool(key *ObjectKey) *Object {
	if key == nil {
		return nil
	}
	hash := key.InternalHash()
	if obj, ok := d.objectPool[hash]; ok {
		return obj
	}
	obj := NewObjectRef(key, d.parser)
	d.objectPool[hash] = obj
	return obj
}

// AddXRefTable merges cross-reference entries into the document.
//
// A nil key is kept, as Java keeps a null one, and every reader of the table
// checks for it — PDFBOX-6132 is the bug that arises when one does not. It
// would be tidier to drop it here, but that would change what XRefTable hands
// back and the Java is the reference.
func (d *Document) AddXRefTable(entries map[*ObjectKey]int64) {
	for key, offset := range entries {
		if key == nil {
			d.nilKeyEntry = &offset
			continue
		}
		hash := key.InternalHash()
		if existing, ok := d.xrefTable[hash]; ok {
			existing.offset = offset
			continue
		}
		d.xrefTable[hash] = &xrefEntry{key: key, offset: offset}
	}
}

// XRefTable returns the cross-reference entries, keyed by the first key object
// seen for each.
func (d *Document) XRefTable() map[*ObjectKey]int64 {
	out := make(map[*ObjectKey]int64, len(d.xrefTable))
	for _, e := range d.xrefTable {
		out[e.key] = e.offset
	}
	if d.nilKeyEntry != nil {
		out[nil] = *d.nilKeyEntry
	}
	return out
}

// LinearizedDictionary returns the linearization dictionary, or nil.
//
// Port of getLinearizedDictionary. It walks the objects with a positive offset
// in ascending offset order, because the linearization dictionary is required
// to be the first object in the file.
func (d *Document) LinearizedDictionary() *Dictionary {
	entries := make([]*xrefEntry, 0, len(d.xrefTable))
	for _, e := range d.xrefTable {
		if e.offset > 0 {
			entries = append(entries, e)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].offset < entries[j].offset })

	for _, e := range entries {
		obj := d.ObjectFromPool(e.key)
		if obj == nil {
			continue
		}
		if dict, ok := obj.Object().(*Dictionary); ok && dict.GetItem(Linearized) != nil {
			return dict
		}
	}
	return nil
}

// ObjectsByType returns every pooled object whose dictionary has the given
// /Type.
func (d *Document) ObjectsByType(typ *Name) []*Object {
	return d.ObjectsByType2(typ, nil)
}

// ObjectsByType2 returns every pooled object whose dictionary has either of the
// given types.
//
// Port of getObjectsByType(COSName, COSName). Dereferencing one of the initial
// keys can trigger the brute force parser on a damaged file, which adds more
// entries to the table while the scan is running. Java takes a second snapshot
// afterwards and scans whatever is new in the same call, so that page discovery
// does not silently miss the recovered objects; the port does the same.
//
// The keys are snapshotted rather than ranged over directly, because the pool
// and the table both grow during the scan and iterating a Go map while it is
// written to is undefined.
func (d *Document) ObjectsByType2(type1, type2 *Name) []*Object {
	scanned := make(map[int64]bool, len(d.xrefTable))
	out := d.scanForType(d.snapshotKeys(scanned), type1, type2, scanned)

	// Anything added while that ran is scanned too. This repeats rather than
	// running exactly twice, because recovery triggered by the second pass can
	// add more again; the scanned set guarantees it terminates.
	for {
		additional := d.snapshotKeys(scanned)
		if len(additional) == 0 {
			return out
		}
		out = append(out, d.scanForType(additional, type1, type2, scanned)...)
	}
}

// snapshotKeys returns the table keys not already in scanned, marking them.
func (d *Document) snapshotKeys(scanned map[int64]bool) []*ObjectKey {
	var keys []*ObjectKey
	for hash, e := range d.xrefTable {
		if scanned[hash] {
			continue
		}
		scanned[hash] = true
		keys = append(keys, e.key)
	}
	return keys
}

// scanForType returns the pooled objects among keys whose dictionary carries
// either type.
func (d *Document) scanForType(keys []*ObjectKey, type1, type2 *Name, scanned map[int64]bool) []*Object {
	var out []*Object
	for _, key := range keys {
		obj := d.ObjectFromPool(key)
		if obj == nil {
			continue
		}
		dict, ok := obj.Object().(*Dictionary)
		if !ok {
			continue
		}
		dictType := dict.GetCOSName(Type)
		if dictType == nil {
			continue
		}
		if dictType == type1 || (type2 != nil && dictType == type2) {
			out = append(out, obj)
		}
	}
	return out
}

// IsClosed reports whether Close has been called.
func (d *Document) IsClosed() bool { return d.closed }

// Close releases every stream the document owns and its stream cache.
//
// Port of close(). Java keeps the first exception, keeps closing everything
// else, logs each failure, and rethrows the first; the port does the same,
// logging through slog.
func (d *Document) Close() error {
	if d.closed {
		return nil
	}

	var firstErr error
	closeOne := func(c interface{ Close() error }, what string) {
		if err := c.Close(); err != nil {
			slog.Error("cos: closing "+what, "err", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	for _, obj := range d.objectPool {
		if obj.IsObjectNull() {
			continue
		}
		if s, ok := obj.Object().(*Stream); ok {
			closeOne(s, "stream")
		}
	}
	for _, s := range d.streams {
		closeOne(s, "stream")
	}
	if d.streamCache != nil {
		closeOne(d.streamCache, "stream cache")
	}

	d.closed = true
	return firstErr
}

// COSObject returns the receiver.
func (d *Document) COSObject() Base { return d }

// Accept dispatches to the visitor.
func (d *Document) Accept(v Visitor) error { return v.VisitDocument(d) }

// XRefOffset returns where the cross-reference table says the given object is,
// negative where it sits inside an object stream. The second result is false
// where the table does not mention it.
//
// Java writes document.getXrefTable().get(objKey), which is a lookup on the
// live map; XRefTable hands back a copy, so the lookup is a method of its own.
func (d *Document) XRefOffset(key *ObjectKey) (int64, bool) {
	if key == nil {
		if d.nilKeyEntry == nil {
			return 0, false
		}
		return *d.nilKeyEntry, true
	}
	entry, ok := d.xrefTable[key.InternalHash()]
	if !ok {
		return 0, false
	}
	return entry.offset, true
}

// PutXRefOffset records where the cross-reference table says an object is.
//
// Java writes document.getXrefTable().put(objKey, offset) in the one place the
// brute force search fills a gap; AddXRefTable is the bulk form.
func (d *Document) PutXRefOffset(key *ObjectKey, offset int64) {
	d.AddXRefTable(map[*ObjectKey]int64{key: offset})
}

// ClearXRefTable empties the cross-reference table.
//
// Java writes document.getXrefTable().clear() on the live map; XRefTable hands
// back a copy, so clearing it is a method of its own.
func (d *Document) ClearXRefTable() {
	clear(d.xrefTable)
	d.nilKeyEntry = nil
}
