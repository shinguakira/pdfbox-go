package pdfparser

// Port of org.apache.pdfbox.pdfparser.PDFObjectStreamParserTest.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// newObjectStream builds the object stream each case parses: /N objects whose
// pairs start at /First, and the objects themselves after that.
//
// Java writes `new COSStream()`, sets the two entries and writes the body
// through createOutputStream(); the Go stream needs a codec provider, and nil
// is right here because nothing is filtered.
func newObjectStream(t *testing.T, count int64, first int64, body string) *cos.Stream {
	t.Helper()
	stream := cos.NewStream(nil)
	stream.SetItem(cos.N, cos.GetInteger(count))
	stream.SetItem(cos.First, cos.GetInteger(first))

	writer, err := stream.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return stream
}

// parseAll is `new PDFObjectStreamParser(stream, doc).parseAllObjects()`. Java
// builds a fresh parser for each call because parsing consumes the source, and
// so does this.
func parseAll(t *testing.T, stream *cos.Stream, document *cos.Document) map[int64]cos.Base {
	t.Helper()
	parser, err := NewObjectStreamParser(stream, document)
	if err != nil {
		t.Fatalf("NewObjectStreamParser: %v", err)
	}
	objects, err := parser.ParseAllObjects()
	if err != nil {
		t.Fatalf("ParseAllObjects: %v", err)
	}
	return objects
}

// wantObject looks the key up the way Java looks up a COSObjectKey, which the
// port keys by the same number-and-generation hash.
func wantObject(t *testing.T, objects map[int64]cos.Base, num int64, want cos.Base) {
	t.Helper()
	key, err := cos.NewObjectKey(num, 0)
	if err != nil {
		t.Fatalf("NewObjectKey(%d): %v", num, err)
	}
	got, ok := objects[key.InternalHash()]
	if !ok {
		t.Errorf("object %d is missing", num)
		return
	}
	if got != want {
		t.Errorf("object %d = %v, want %v", num, got, want)
	}
}

// putXRefIndex is `xrefTable.put(new COSObjectKey(num, gen, index), -1L)`.
func putXRefIndex(t *testing.T, document *cos.Document, num int64, gen, index int) {
	t.Helper()
	key, err := cos.NewObjectKeyInStream(num, gen, index)
	if err != nil {
		t.Fatalf("NewObjectKeyInStream(%d, %d, %d): %v", num, gen, index, err)
	}
	document.PutXRefOffset(key, -1)
}

// TestOffsetParsing is testOffsetParsing.
func TestOffsetParsing(t *testing.T) {
	stream := newObjectStream(t, 2, 8, "4 0 6 5 true false")

	parser, err := NewObjectStreamParser(stream, nil)
	if err != nil {
		t.Fatalf("NewObjectStreamParser: %v", err)
	}
	objectNumbers, err := parser.ReadObjectNumbers()
	if err != nil {
		t.Fatalf("ReadObjectNumbers: %v", err)
	}
	if len(objectNumbers) != 2 {
		t.Fatalf("ReadObjectNumbers gave %d entries, want 2", len(objectNumbers))
	}
	if objectNumbers[4] != 0 {
		t.Errorf("offset of object 4 = %d, want 0", objectNumbers[4])
	}
	if objectNumbers[6] != 5 {
		t.Errorf("offset of object 6 = %d, want 5", objectNumbers[6])
	}

	for _, row := range []struct {
		number int64
		want   cos.Base
	}{
		{4, cos.True},
		{6, cos.False},
	} {
		parser, err := NewObjectStreamParser(stream, nil)
		if err != nil {
			t.Fatalf("NewObjectStreamParser: %v", err)
		}
		got, err := parser.ParseObject(row.number)
		if err != nil {
			t.Fatalf("ParseObject(%d): %v", row.number, err)
		}
		if got != row.want {
			t.Errorf("ParseObject(%d) = %v, want %v", row.number, got, row.want)
		}
	}
}

// TestParseAllObjects is testParseAllObjects.
func TestParseAllObjects(t *testing.T) {
	stream := newObjectStream(t, 2, 8, "6 0 4 5 true false")

	objects := parseAll(t, stream, nil)
	if len(objects) != 2 {
		t.Fatalf("ParseAllObjects gave %d objects, want 2", len(objects))
	}
	wantObject(t, objects, 6, cos.True)
	wantObject(t, objects, 4, cos.False)
}

// TestParseAllObjectsIndexed is testParseAllObjectsIndexed: object number 4 is
// used twice, so the index in the cross-reference key picks which one wins.
func TestParseAllObjectsIndexed(t *testing.T) {
	// use object number 4 for two objects
	stream := newObjectStream(t, 3, 13, "6 0 4 5 4 11 true false true")
	document := cos.NewDocument(nil)

	// select the second object from the stream for object number 4 by using 2
	// as the value for the index
	putXRefIndex(t, document, 6, 0, 0)
	putXRefIndex(t, document, 4, 0, 2)

	objects := parseAll(t, stream, document)
	if len(objects) != 2 {
		t.Fatalf("ParseAllObjects gave %d objects, want 2", len(objects))
	}
	wantObject(t, objects, 6, cos.True)
	wantObject(t, objects, 4, cos.True)

	// select the first object from the stream for object number 4 by using 1 as
	// the value for the index. Remove the old entry first to be sure it is
	// replaced -- Java's comment, and it is load-bearing: HashMap.put keeps the
	// key object it already has and only updates the value, so a put alone
	// leaves the index at 2. The port reproduces that, which is why this needs
	// the removal Java's test makes.
	key4, err := cos.NewObjectKey(4, 0)
	if err != nil {
		t.Fatalf("NewObjectKey(4): %v", err)
	}
	if !document.RemoveXRefOffset(key4) {
		t.Fatal("RemoveXRefOffset found nothing to remove for object 4")
	}
	putXRefIndex(t, document, 4, 0, 1)

	objects = parseAll(t, stream, document)
	if len(objects) != 2 {
		t.Fatalf("ParseAllObjects gave %d objects, want 2", len(objects))
	}
	wantObject(t, objects, 6, cos.True)
	wantObject(t, objects, 4, cos.False)
}

// TestParseAllObjectsSkipMalformedIndex is
// testParseAllObjectsSkipMalformedIndex: where every object number in the
// stream is distinct the index is not consulted at all, so indexes that match
// nothing do no harm.
func TestParseAllObjectsSkipMalformedIndex(t *testing.T) {
	stream := newObjectStream(t, 3, 13, "6 0 4 5 5 11 true false true")
	document := cos.NewDocument(nil)

	// an index for each object key which doesn't match the index within the
	// object stream
	putXRefIndex(t, document, 6, 0, 10)
	putXRefIndex(t, document, 4, 0, 11)
	putXRefIndex(t, document, 5, 0, 12)

	objects := parseAll(t, stream, document)
	if len(objects) != 3 {
		t.Fatalf("ParseAllObjects gave %d objects, want 3", len(objects))
	}
	wantObject(t, objects, 6, cos.True)
	wantObject(t, objects, 4, cos.False)
	wantObject(t, objects, 5, cos.True)
}

// TestParseAllObjectsUseMalformedIndex is
// testParseAllObjectsUseMalformedIndex: where an object number repeats, the
// index *is* consulted, and indexes that match nothing drop everything.
func TestParseAllObjectsUseMalformedIndex(t *testing.T) {
	stream := newObjectStream(t, 3, 13, "6 0 4 5 4 11 true false true")
	document := cos.NewDocument(nil)

	// two object keys only, as the stream uses one object number twice
	putXRefIndex(t, document, 6, 0, 10)
	putXRefIndex(t, document, 4, 0, 11)

	objects := parseAll(t, stream, document)
	if len(objects) != 0 {
		t.Errorf("ParseAllObjects gave %d objects, want none: the indexes match "+
			"no object in the stream", len(objects))
	}
}
