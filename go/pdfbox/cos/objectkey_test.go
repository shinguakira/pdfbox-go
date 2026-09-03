package cos

import "testing"

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/COSObjectKeyTest.java.
//
// testPDFBox5742 is not ported here: it loads a PDF, renders both pages, splits
// the document and compares renderings. That needs the parser, the writer,
// multipdf and a renderer, none of which exist yet. It is a slice 7/9 test, not
// a cos test — recorded in migration/STATUS.md.

func TestObjectKeyInputValues(t *testing.T) {
	// Java: negative object and generation numbers throw IllegalArgumentException
	if _, err := NewObjectKey(-1, 0); err == nil {
		t.Error("NewObjectKey(-1, 0) succeeded, want an error")
	}
	if _, err := NewObjectKey(1, -1); err == nil {
		t.Error("NewObjectKey(1, -1) succeeded, want an error")
	}
}

// mustKey builds a key, failing the test if the Java constructor would have
// thrown.
func mustKey(t *testing.T, num int64, gen int) *ObjectKey {
	t.Helper()
	k, err := NewObjectKey(num, gen)
	if err != nil {
		t.Fatalf("NewObjectKey(%d, %d): %v", num, gen, err)
	}
	return k
}

func TestObjectKeyCompareToEqual(t *testing.T) {
	objectUnderTest := mustKey(t, 1, 0)
	other := mustKey(t, 1, 0)

	if got := objectUnderTest.Compare(other); got != 0 {
		t.Errorf("Compare = %d, want 0", got)
	}
}

func TestObjectKeyCompareToNotEqual(t *testing.T) {
	objectUnderTest := mustKey(t, 1, 0)
	other := mustKey(t, 9_999_999, 0)

	if got := objectUnderTest.Compare(other); got != -1 {
		t.Errorf("Compare = %d, want -1", got)
	}
	if got := other.Compare(objectUnderTest); got != 1 {
		t.Errorf("Compare = %d, want 1", got)
	}
}

func TestObjectKeyEquals(t *testing.T) {
	// Java equals compares only numberAndGeneration, so the stream index is
	// deliberately not part of equality.
	if !mustKey(t, 100, 0).Equals(mustKey(t, 100, 0)) {
		t.Error("(100,0) != (100,0)")
	}
	if mustKey(t, 100, 0).Equals(mustKey(t, 101, 0)) {
		t.Error("(100,0) == (101,0), want them distinct")
	}
}

func TestObjectKeyInternalRepresentation(t *testing.T) {
	// The number and generation are packed into one int64: generation in the
	// low 16 bits, number above. These cases check the packing round-trips.
	cases := []struct {
		num int64
		gen int
	}{
		{100, 0},
		{200, 4},
		{200000, 0},
		{87654321, 123},
	}
	for _, c := range cases {
		key := mustKey(t, c.num, c.gen)
		if got := key.Number(); got != c.num {
			t.Errorf("Number() = %d, want %d", got, c.num)
		}
		if got := key.Generation(); got != c.gen {
			t.Errorf("Generation() = %d, want %d", got, c.gen)
		}
	}
}

func TestObjectKeySortingOrder(t *testing.T) {
	// Comparison is by object number first; generation breaks ties.
	key40 := mustKey(t, 4, 0)
	key41 := mustKey(t, 4, 1)
	key50 := mustKey(t, 5, 0)

	if got := key40.Compare(key40); got != 0 {
		t.Errorf("key40.Compare(key40) = %d, want 0", got)
	}
	if got := key41.Compare(key41); got != 0 {
		t.Errorf("key41.Compare(key41) = %d, want 0", got)
	}
	if got := key40.Compare(key41); got != -1 {
		t.Errorf("key40.Compare(key41) = %d, want -1", got)
	}
	if got := key40.Compare(key50); got != -1 {
		t.Errorf("key40.Compare(key50) = %d, want -1", got)
	}
	if got := key41.Compare(key50); got != -1 {
		t.Errorf("key41.Compare(key50) = %d, want -1", got)
	}
}

func TestObjectKeyInternalHash(t *testing.T) {
	// Java checkHashCode asserts on Object.hashCode. Go has no hashCode, and
	// the value it hashes is the packed number-and-generation, which is
	// exposed directly as InternalHash. These are the same three cases.

	// same object number
	if mustKey(t, 100, 0).InternalHash() != mustKey(t, 100, 0).InternalHash() {
		t.Error("(100,0) and (100,0) have different internal hashes")
	}
	// different object numbers, same generation
	if mustKey(t, 100, 0).InternalHash() == mustKey(t, 200, 0).InternalHash() {
		t.Error("(100,0) and (200,0) share an internal hash")
	}
	// different numbers and generations whose sums are equal
	if mustKey(t, 100, 0).InternalHash() == mustKey(t, 99, 1).InternalHash() {
		t.Error("(100,0) and (99,1) share an internal hash")
	}
}

func TestObjectKeyString(t *testing.T) {
	// Java: toString returns "<number> <generation> R"
	if got := mustKey(t, 12, 0).String(); got != "12 0 R" {
		t.Errorf("String() = %q, want %q", got, "12 0 R")
	}
	if got := mustKey(t, 87654321, 123).String(); got != "87654321 123 R" {
		t.Errorf("String() = %q, want %q", got, "87654321 123 R")
	}
}

func TestObjectKeyStreamIndex(t *testing.T) {
	// The two-argument constructor defaults the stream index to -1; the
	// three-argument one sets it. The index is not part of equality.
	plain := mustKey(t, 5, 0)
	if got := plain.StreamIndex(); got != -1 {
		t.Errorf("StreamIndex() = %d, want -1", got)
	}

	inStream, err := NewObjectKeyInStream(5, 0, 3)
	if err != nil {
		t.Fatalf("NewObjectKeyInStream: %v", err)
	}
	if got := inStream.StreamIndex(); got != 3 {
		t.Errorf("StreamIndex() = %d, want 3", got)
	}
	if !plain.Equals(inStream) {
		t.Error("keys differing only in stream index compare unequal; Java equality ignores it")
	}
}
