package cos

import (
	"errors"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The Java suite has no TestCOSObject — COSObject is exercised only through the
// parser. Per migration/conventions/tdd.md these are written from
// pdfbox/src/main/java/org/apache/pdfbox/cos/COSObject.java.

// stubParser is an ICOSParser that hands back a fixed object and counts how
// often it was asked.
type stubParser struct {
	result Base
	err    error
	calls  int
}

func (p *stubParser) DereferenceObject(obj *Object) (Base, error) {
	p.calls++
	return p.result, p.err
}

func (p *stubParser) CreateRandomAccessReadView(start, length int64) (pdfio.RandomAccessRead, error) {
	return nil, errors.New("not implemented in the stub")
}

func TestObjectBaseContract(t *testing.T) {
	assertBaseContract(t, NewObject(GetPDFName("Type")))
}

func TestObjectAccept(t *testing.T) {
	assertVisits(t, NewObject(GetPDFName("Type")), "object")
}

func TestObjectDirect(t *testing.T) {
	// A COSObject built around an existing base is already dereferenced.
	inner := GetPDFName("Type")
	o := NewObject(inner)

	if !o.IsDereferenced() {
		t.Error("IsDereferenced() = false for an object built from a base")
	}
	if o.IsObjectNull() {
		t.Error("IsObjectNull() = true for an object built from a base")
	}
	if got := o.Object(); got != Base(inner) {
		t.Errorf("Object() = %v, want the base it was built from", got)
	}
}

func TestObjectLazyDereference(t *testing.T) {
	inner := GetPDFName("Resolved")
	parser := &stubParser{result: inner}
	key, err := NewObjectKey(12, 0)
	if err != nil {
		t.Fatalf("NewObjectKey: %v", err)
	}
	o := NewObjectRef(key, parser)

	if o.IsDereferenced() {
		t.Error("IsDereferenced() = true before the first Object() call")
	}
	if o.Key() != key {
		t.Error("Key() did not return the key the object was built with")
	}

	if got := o.Object(); got != Base(inner) {
		t.Errorf("Object() = %v, want the resolved base", got)
	}
	if !o.IsDereferenced() {
		t.Error("IsDereferenced() = false after Object()")
	}
	if parser.calls != 1 {
		t.Errorf("parser called %d times, want 1", parser.calls)
	}

	// a second call must not go back to the parser
	o.Object()
	if parser.calls != 1 {
		t.Errorf("parser called %d times after a second Object(), want 1", parser.calls)
	}
}

// TestObjectDereferenceFailure pins the behaviour when the parser fails: Java
// logs and returns whatever baseObject holds, which is null, and never retries.
func TestObjectDereferenceFailure(t *testing.T) {
	parser := &stubParser{err: errors.New("boom")}
	key, _ := NewObjectKey(7, 0)
	o := NewObjectRef(key, parser)

	if got := o.Object(); got != nil {
		t.Errorf("Object() = %v after a parser failure, want nil", got)
	}
	if !o.IsDereferenced() {
		t.Error("IsDereferenced() = false after a failed dereference; it must not be retried")
	}
	o.Object()
	if parser.calls != 1 {
		t.Errorf("parser called %d times, want 1 — a failed dereference must not retry", parser.calls)
	}
}

// TestObjectDereferenceRecursion covers the reason Java sets isDereferenced
// before calling the parser: an object whose resolution reaches itself must not
// recurse forever.
func TestObjectDereferenceRecursion(t *testing.T) {
	key, _ := NewObjectKey(3, 0)
	parser := &recursiveParser{}
	o := NewObjectRef(key, parser)
	parser.target = o

	// must terminate
	o.Object()
	if parser.calls != 1 {
		t.Errorf("parser called %d times, want 1", parser.calls)
	}
}

type recursiveParser struct {
	target *Object
	calls  int
}

func (p *recursiveParser) DereferenceObject(obj *Object) (Base, error) {
	p.calls++
	// re-entering while the first call is still in flight
	return p.target.Object(), nil
}

func (p *recursiveParser) CreateRandomAccessReadView(start, length int64) (pdfio.RandomAccessRead, error) {
	return nil, errors.New("not implemented in the stub")
}

func TestObjectSetToNull(t *testing.T) {
	o := NewObject(GetPDFName("Type"))
	o.SetToNull()
	if got := o.Object(); got != Base(NullObject) {
		t.Errorf("Object() = %v after SetToNull, want the null object", got)
	}
}

func TestObjectString(t *testing.T) {
	key, _ := NewObjectKey(12, 0)
	o := NewObjectRef(key, nil)
	// Java: toString returns "COSObject{" + getKey() + "}"
	if got, want := o.String(), "COSObject{12 0 R}"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestObjectIsObjectNull(t *testing.T) {
	key, _ := NewObjectKey(1, 0)
	o := NewObjectRef(key, nil)
	if !o.IsObjectNull() {
		t.Error("IsObjectNull() = false for an unresolved object with no parser")
	}
}
