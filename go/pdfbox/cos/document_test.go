package cos

import (
	"errors"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/COSDocumentTest.java,
// plus tests written from COSDocument.java for the behaviour that file does not
// reach — it has only one test method.

// TestDocumentPDFBox6132 covers PDFBOX-6132: a nil key in the cross-reference
// table must not crash the lookups that walk it.
func TestDocumentPDFBox6132(t *testing.T) {
	d := NewDocument(nil)

	d.AddXRefTable(map[*ObjectKey]int64{nil: 10})

	if got := d.ObjectsByType(T); len(got) != 0 {
		t.Errorf("ObjectsByType(T) = %v, want empty", got)
	}
	if got := d.LinearizedDictionary(); got != nil {
		t.Errorf("LinearizedDictionary() = %v, want nil", got)
	}
}

func TestDocumentVersion(t *testing.T) {
	d := NewDocument(nil)

	// Java defaults the version to 1.4
	if got := d.Version(); got != 1.4 {
		t.Errorf("Version() = %v, want 1.4", got)
	}
	d.SetVersion(1.7)
	if got := d.Version(); got != 1.7 {
		t.Errorf("Version() = %v, want 1.7", got)
	}
}

func TestDocumentTrailer(t *testing.T) {
	d := NewDocument(nil)

	if got := d.Trailer(); got != nil {
		t.Errorf("Trailer() = %v, want nil before it is set", got)
	}
	trailer := NewDictionary()
	d.SetTrailer(trailer)
	if got := d.Trailer(); got != trailer {
		t.Errorf("Trailer() = %v, want the dictionary that was set", got)
	}
}

func TestDocumentObjectPool(t *testing.T) {
	d := NewDocument(nil)
	key, err := NewObjectKey(12, 0)
	if err != nil {
		t.Fatalf("NewObjectKey: %v", err)
	}

	obj := d.ObjectFromPool(key)
	if obj == nil {
		t.Fatal("ObjectFromPool returned nil for a valid key")
	}
	// asking twice must return the same proxy, so that a forward reference
	// resolved later is seen by every holder
	if again := d.ObjectFromPool(key); again != obj {
		t.Error("ObjectFromPool returned a different object for the same key")
	}

	// an equal key built separately must also hit the same entry
	same, _ := NewObjectKey(12, 0)
	if again := d.ObjectFromPool(same); again != obj {
		t.Error("ObjectFromPool returned a different object for an equal key")
	}

	if got := d.ObjectFromPool(nil); got != nil {
		t.Errorf("ObjectFromPool(nil) = %v, want nil", got)
	}
}

func TestDocumentEncryption(t *testing.T) {
	d := NewDocument(nil)
	d.SetTrailer(NewDictionary())

	if d.IsEncrypted() {
		t.Error("IsEncrypted() = true with no encryption dictionary")
	}

	enc := NewDictionary()
	d.SetEncryptionDictionary(enc)
	if !d.IsEncrypted() {
		t.Error("IsEncrypted() = false after setting an encryption dictionary")
	}
	if got := d.EncryptionDictionary(); got != enc {
		t.Errorf("EncryptionDictionary() = %v, want the dictionary that was set", got)
	}

	if d.IsDecrypted() {
		t.Error("IsDecrypted() = true before decryption")
	}
	d.SetDecrypted()
	if !d.IsDecrypted() {
		t.Error("IsDecrypted() = false after SetDecrypted")
	}
}

func TestDocumentDocumentID(t *testing.T) {
	d := NewDocument(nil)
	d.SetTrailer(NewDictionary())

	if got := d.DocumentID(); got != nil {
		t.Errorf("DocumentID() = %v, want nil", got)
	}
	id := NewArray()
	d.SetDocumentID(id)
	if got := d.DocumentID(); got != id {
		t.Errorf("DocumentID() = %v, want the array that was set", got)
	}
}

func TestDocumentXRefTable(t *testing.T) {
	d := NewDocument(nil)
	first, _ := NewObjectKey(1, 0)
	second, _ := NewObjectKey(2, 0)

	d.AddXRefTable(map[*ObjectKey]int64{first: 100})
	d.AddXRefTable(map[*ObjectKey]int64{second: 200})

	table := d.XRefTable()
	if len(table) != 2 {
		t.Fatalf("XRefTable() has %d entries, want 2", len(table))
	}
	if table[first] != 100 || table[second] != 200 {
		t.Errorf("XRefTable() = %v, want the two offsets that were added", table)
	}
}

func TestDocumentStartXrefAndFlags(t *testing.T) {
	d := NewDocument(nil)

	d.SetStartXref(1234)
	if got := d.StartXref(); got != 1234 {
		t.Errorf("StartXref() = %d, want 1234", got)
	}

	if d.IsXRefStream() {
		t.Error("IsXRefStream() = true by default")
	}
	d.SetIsXRefStream(true)
	if !d.IsXRefStream() {
		t.Error("IsXRefStream() = false after being set")
	}

	if d.HasHybridXRef() {
		t.Error("HasHybridXRef() = true by default")
	}
	d.SetHasHybridXRef()
	if !d.HasHybridXRef() {
		t.Error("HasHybridXRef() = false after being set")
	}

	d.SetHighestXRefObjectNumber(99)
	if got := d.HighestXRefObjectNumber(); got != 99 {
		t.Errorf("HighestXRefObjectNumber() = %d, want 99", got)
	}
}

func TestDocumentObjectsByType(t *testing.T) {
	d := NewDocument(nil)

	pageDict := NewDictionary()
	pageDict.SetItem(Type, Page)
	pagesDict := NewDictionary()
	pagesDict.SetItem(Type, Pages)

	pageKey, _ := NewObjectKey(1, 0)
	pagesKey, _ := NewObjectKey(2, 0)

	// seed the pool directly, as a parser would once it has read the objects
	d.ObjectFromPool(pageKey).baseObject = pageDict
	d.ObjectFromPool(pagesKey).baseObject = pagesDict
	d.AddXRefTable(map[*ObjectKey]int64{pageKey: 10, pagesKey: 20})

	pages := d.ObjectsByType(Page)
	if len(pages) != 1 {
		t.Fatalf("ObjectsByType(Page) returned %d objects, want 1", len(pages))
	}
	if pages[0].Object() != Base(pageDict) {
		t.Error("ObjectsByType(Page) returned the wrong object")
	}

	// the two-type form matches either
	both := d.ObjectsByType2(Page, Pages)
	if len(both) != 2 {
		t.Errorf("ObjectsByType2(Page, Pages) returned %d objects, want 2", len(both))
	}

	if got := d.ObjectsByType(Font); len(got) != 0 {
		t.Errorf("ObjectsByType(Font) = %v, want empty", got)
	}
}

func TestDocumentClose(t *testing.T) {
	d := NewDocument(nil)

	if d.IsClosed() {
		t.Error("IsClosed() = true before Close")
	}
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !d.IsClosed() {
		t.Error("IsClosed() = false after Close")
	}
	// closing twice must not be a problem
	if err := d.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestDocumentAccept(t *testing.T) {
	assertVisits(t, NewDocument(nil), "document")
}

func TestDocumentBaseContract(t *testing.T) {
	assertBaseContract(t, NewDocument(nil))
}

// rescanParser adds a cross-reference entry the first time it is asked to
// dereference anything, standing in for the brute force parser kicking in on a
// damaged file part-way through a scan.
type rescanParser struct {
	doc   *Document
	added bool
}

func (p *rescanParser) DereferenceObject(obj *Object) (Base, error) {
	if !p.added {
		p.added = true
		late, _ := NewObjectKey(99, 0)
		p.doc.AddXRefTable(map[*ObjectKey]int64{late: 500})
		lateDict := NewDictionary()
		lateDict.SetItem(Type, Page)
		p.doc.ObjectFromPool(late).baseObject = lateDict
	}
	return NullObject, nil
}

func (p *rescanParser) CreateRandomAccessReadView(start, length int64) (pdfio.RandomAccessRead, error) {
	return nil, errors.New("not implemented in the stub")
}

// TestDocumentObjectsByTypeRescans pins the second pass in getObjectsByType:
// dereferencing one of the initial keys can trigger damaged-file recovery that
// adds more entries, and Java scans those in the same call rather than leaving
// them for a later one. Page discovery would otherwise silently miss recovered
// objects.
func TestDocumentObjectsByTypeRescans(t *testing.T) {
	parser := &rescanParser{}
	d := NewDocument(parser)
	parser.doc = d

	// one key that resolves to nothing, but whose dereference adds another
	seed, _ := NewObjectKey(1, 0)
	d.AddXRefTable(map[*ObjectKey]int64{seed: 10})

	got := d.ObjectsByType(Page)
	if len(got) != 1 {
		t.Fatalf("ObjectsByType(Page) returned %d objects, want 1 — the key added "+
			"during the scan must be picked up in the same call", len(got))
	}
}
