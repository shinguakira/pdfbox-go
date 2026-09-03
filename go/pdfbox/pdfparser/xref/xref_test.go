package xref

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from the sources in
// pdfbox/src/main/java/org/apache/pdfbox/pdfparser/xref/. The Java suite has no
// tests for this package, so per migration/conventions/tdd.md these are written
// first from the source.

func testKey(t *testing.T, num int64, gen int) *cos.ObjectKey {
	t.Helper()
	k, err := cos.NewObjectKey(num, gen)
	if err != nil {
		t.Fatalf("NewObjectKey(%d, %d): %v", num, gen, err)
	}
	return k
}

func TestTypeNumericValue(t *testing.T) {
	// The numbers are the type column of a cross-reference stream.
	cases := []struct {
		typ  Type
		want int
	}{
		{TypeFree, 0},
		{TypeNormal, 1},
		{TypeObjectStream, 2},
	}
	for _, c := range cases {
		if got := c.typ.NumericValue(); got != c.want {
			t.Errorf("%v.NumericValue() = %d, want %d", c.typ, got, c.want)
		}
	}
}

func TestFreeReference(t *testing.T) {
	key := testKey(t, 3, 2)
	ref := NewFreeReference(key, 7)

	if got := ref.Type(); got != TypeFree {
		t.Errorf("Type() = %v, want TypeFree", got)
	}
	// column one is the type number
	if got := ref.FirstColumnValue(); got != 0 {
		t.Errorf("FirstColumnValue() = %d, want 0", got)
	}
	// column two is the next free object
	if got := ref.SecondColumnValue(); got != 7 {
		t.Errorf("SecondColumnValue() = %d, want 7", got)
	}
	// column three is the generation
	if got := ref.ThirdColumnValue(); got != 2 {
		t.Errorf("ThirdColumnValue() = %d, want 2", got)
	}
	if got := ref.ReferencedKey(); got != key {
		t.Errorf("ReferencedKey() = %v, want the key it was built with", got)
	}
}

// TestNullEntry pins the head of the free list every PDF carries: object 0,
// generation 65535, pointing at 0.
func TestNullEntry(t *testing.T) {
	if got := NullEntry.ReferencedKey().Number(); got != 0 {
		t.Errorf("NullEntry object number = %d, want 0", got)
	}
	if got := NullEntry.ReferencedKey().Generation(); got != 65535 {
		t.Errorf("NullEntry generation = %d, want 65535", got)
	}
	if got := NullEntry.SecondColumnValue(); got != 0 {
		t.Errorf("NullEntry next free object = %d, want 0", got)
	}
}

func TestNormalReference(t *testing.T) {
	key := testKey(t, 12, 0)
	ref := NewNormalReference(1234, key, cos.GetPDFName("Anything"))

	if got := ref.Type(); got != TypeNormal {
		t.Errorf("Type() = %v, want TypeNormal", got)
	}
	if got := ref.FirstColumnValue(); got != 1 {
		t.Errorf("FirstColumnValue() = %d, want 1", got)
	}
	// column two is the byte offset
	if got := ref.SecondColumnValue(); got != 1234 {
		t.Errorf("SecondColumnValue() = %d, want 1234", got)
	}
	// column three is the generation
	if got := ref.ThirdColumnValue(); got != 0 {
		t.Errorf("ThirdColumnValue() = %d, want 0", got)
	}
	if got := ref.ByteOffset(); got != 1234 {
		t.Errorf("ByteOffset() = %d, want 1234", got)
	}
	if ref.IsObjectStream() {
		t.Error("IsObjectStream() = true for a plain object")
	}
}

// TestNormalReferenceDetectsObjectStream covers the check the constructor does:
// a stream whose /Type is /ObjStm is an object stream, and the writer needs to
// know because those objects are numbered differently.
func TestNormalReferenceDetectsObjectStream(t *testing.T) {
	stream := cos.NewStream(nil)
	defer stream.Close()
	stream.SetItem(cos.Type, cos.ObjStm)

	ref := NewNormalReference(10, testKey(t, 1, 0), stream)
	if !ref.IsObjectStream() {
		t.Error("IsObjectStream() = false for a stream typed /ObjStm")
	}

	// a stream of another type is not
	other := cos.NewStream(nil)
	defer other.Close()
	other.SetItem(cos.Type, cos.XRef)
	if NewNormalReference(10, testKey(t, 2, 0), other).IsObjectStream() {
		t.Error("IsObjectStream() = true for a stream that is not /ObjStm")
	}
}

// TestNormalReferenceThroughIndirect covers the constructor resolving an
// indirect reference before looking at the type.
func TestNormalReferenceThroughIndirect(t *testing.T) {
	stream := cos.NewStream(nil)
	defer stream.Close()
	stream.SetItem(cos.Type, cos.ObjStm)

	ref := NewNormalReference(10, testKey(t, 1, 0), cos.NewObject(stream))
	if !ref.IsObjectStream() {
		t.Error("IsObjectStream() = false for an indirect reference to an /ObjStm stream")
	}
}

func TestObjectStreamReference(t *testing.T) {
	key := testKey(t, 5, 0)
	container := testKey(t, 9, 0)
	ref := NewObjectStreamReference(key, container, 3)

	if got := ref.Type(); got != TypeObjectStream {
		t.Errorf("Type() = %v, want TypeObjectStream", got)
	}
	if got := ref.FirstColumnValue(); got != 2 {
		t.Errorf("FirstColumnValue() = %d, want 2", got)
	}
	// column two is the object number of the containing stream
	if got := ref.SecondColumnValue(); got != 9 {
		t.Errorf("SecondColumnValue() = %d, want 9", got)
	}
	// column three is the index within that stream
	if got := ref.ThirdColumnValue(); got != 3 {
		t.Errorf("ThirdColumnValue() = %d, want 3", got)
	}
}

func TestCompare(t *testing.T) {
	low := NewNormalReference(0, testKey(t, 1, 0), nil)
	high := NewNormalReference(0, testKey(t, 2, 0), nil)

	if got := Compare(low, high); got >= 0 {
		t.Errorf("Compare(1, 2) = %d, want negative", got)
	}
	if got := Compare(high, low); got <= 0 {
		t.Errorf("Compare(2, 1) = %d, want positive", got)
	}
	if got := Compare(low, low); got != 0 {
		t.Errorf("Compare(x, x) = %d, want 0", got)
	}

	// Java sorts an entry with no key first, and a nil other entry last.
	noKey := NewNormalReference(0, nil, nil)
	if got := Compare(noKey, low); got != -1 {
		t.Errorf("Compare(keyless, x) = %d, want -1", got)
	}
	if got := Compare(low, nil); got != 1 {
		t.Errorf("Compare(x, nil) = %d, want 1", got)
	}
	if got := Compare(low, noKey); got != 1 {
		t.Errorf("Compare(x, keyless) = %d, want 1", got)
	}
}
