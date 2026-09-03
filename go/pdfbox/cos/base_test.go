package cos

import "testing"

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSBase.java.
//
// The Java version is an abstract test class that every COS type's test extends,
// inheriting testGetCOSObject and testIsSetDirect. Go has no test inheritance,
// so the shared contract becomes a helper each type's test calls.

// assertBaseContract runs the checks TestCOSBase applies to every COS object.
func assertBaseContract(t *testing.T, b Base) {
	t.Helper()

	// testGetCOSObject: the underlying object is itself
	if got := b.COSObject(); got != b {
		t.Errorf("COSObject() = %v, want the receiver itself", got)
	}

	// testIsSetDirect: getter and setter round-trip
	b.SetDirect(true)
	if !b.IsDirect() {
		t.Error("IsDirect() = false after SetDirect(true)")
	}
	b.SetDirect(false)
	if b.IsDirect() {
		t.Error("IsDirect() = true after SetDirect(false)")
	}
}

// assertBytesEqual ports the testByteArrays helper.
//
// The Java helper compares byteArr1.length against itself, so it never actually
// checks that the lengths match. That looks like a typo rather than intent, and
// the port compares the lengths properly.
func assertBytesEqual(t *testing.T, want, got []byte) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length = %d, want %d (got %q, want %q)", len(got), len(want), got, want)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("byte %d = %d, want %d (got %q, want %q)", i, got[i], want[i], got, want)
		}
	}
}

// recordingVisitor is a Visitor that records which method was dispatched to.
//
// The Java accept() tests drive a COSWriter and assert on the bytes it emits.
// COSWriter belongs to pdfwriter, which is not ported yet, so the accept tests
// here assert the double dispatch instead — that a type calls the right Visitor
// method. The byte-level assertions come back with pdfwriter; see
// migration/STATUS.md.
type recordingVisitor struct {
	visited string
	err     error
}

func (v *recordingVisitor) record(name string) error {
	v.visited = name
	return v.err
}

func (v *recordingVisitor) VisitArray(*Array) error           { return v.record("array") }
func (v *recordingVisitor) VisitBoolean(*Boolean) error       { return v.record("boolean") }
func (v *recordingVisitor) VisitDictionary(*Dictionary) error { return v.record("dictionary") }
func (v *recordingVisitor) VisitFloat(*Float) error           { return v.record("float") }
func (v *recordingVisitor) VisitInteger(*Integer) error       { return v.record("integer") }
func (v *recordingVisitor) VisitName(*Name) error             { return v.record("name") }
func (v *recordingVisitor) VisitNull(*Null) error             { return v.record("null") }
func (v *recordingVisitor) VisitObject(*Object) error         { return v.record("object") }
func (v *recordingVisitor) VisitStream(*Stream) error         { return v.record("stream") }
func (v *recordingVisitor) VisitStringObj(*StringObj) error   { return v.record("string") }

func assertVisits(t *testing.T, b Base, want string) {
	t.Helper()
	v := &recordingVisitor{}
	if err := b.Accept(v); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if v.visited != want {
		t.Errorf("Accept dispatched to %q, want %q", v.visited, want)
	}
}
