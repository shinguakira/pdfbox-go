package cos

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Parser resolves indirect references and opens views onto the underlying file.
//
// Port of org.apache.pdfbox.cos.ICOSParser. It exists so that cos does not
// depend on pdfparser: an Object holds a Parser and calls back into it the
// first time it is dereferenced.
type Parser interface {
	// DereferenceObject resolves an indirect reference to the object it names.
	DereferenceObject(obj *Object) (Base, error)

	// CreateRandomAccessReadView returns a view onto the file backing the
	// document.
	//
	// Java declares this as returning RandomAccessReadView, the concrete type.
	// The port returns the interface, which is what pdfio.CreateView produces
	// and what every caller here needs.
	CreateRandomAccessReadView(start, length int64) (pdfio.RandomAccessRead, error)
}

// Object is an indirect reference to another COS object.
//
// Port of org.apache.pdfbox.cos.COSObject. The referenced object is resolved
// lazily, the first time Object is called.
//
// Not yet ported: the COSUpdateState this carries in Java, which tracks whether
// the object has changed for an incremental save. That is slice 7 work; see
// migration/STATUS.md.
type Object struct {
	object
	baseObject     Base
	parser         Parser
	isDereferenced bool
}

var _ Base = (*Object)(nil)

// NewObject wraps an already-resolved object.
//
// Port of COSObject(COSBase).
func NewObject(base Base) *Object {
	return &Object{baseObject: base, isDereferenced: true}
}

// NewObjectWithKey wraps an already-resolved object and records its key.
//
// Port of COSObject(COSBase, COSObjectKey).
func NewObjectWithKey(base Base, key *ObjectKey) *Object {
	o := &Object{baseObject: base, isDereferenced: true}
	o.SetKey(key)
	return o
}

// NewObjectLazy wraps an object that may still need resolving through parser.
//
// Port of COSObject(COSBase, ICOSParser). A nil base means the object has not
// been read yet.
func NewObjectLazy(base Base, parser Parser) *Object {
	return &Object{
		baseObject:     base,
		parser:         parser,
		isDereferenced: base != nil,
	}
}

// NewObjectRef creates an unresolved reference to the object named by key.
//
// Port of COSObject(COSObjectKey, ICOSParser).
func NewObjectRef(key *ObjectKey, parser Parser) *Object {
	o := &Object{parser: parser}
	o.SetKey(key)
	return o
}

// IsObjectNull reports whether the referenced object is absent.
func (o *Object) IsObjectNull() bool { return o.baseObject == nil }

// IsDereferenced reports whether resolution has been attempted.
func (o *Object) IsDereferenced() bool { return o.isDereferenced }

// Object returns the referenced object, resolving it through the parser on the
// first call.
//
// Java swallows the IOException here, logs it, and returns whatever baseObject
// holds — the method has no throws clause and every caller assumes it cannot
// fail. The port keeps that contract rather than changing the signature of a
// method the whole document model calls; the error is logged through slog.
func (o *Object) Object() Base {
	if !o.isDereferenced && o.parser != nil {
		// Marked before the call, so that an object whose resolution reaches
		// itself terminates instead of recursing.
		o.isDereferenced = true
		parser := o.parser
		o.parser = nil

		base, err := parser.DereferenceObject(o)
		if err != nil {
			slog.Error("cos: cannot dereference object", "key", o.Key(), "err", err)
		} else {
			o.baseObject = base
		}
	}
	return o.baseObject
}

// SetToNull replaces the referenced object with the null object and drops the
// parser, so it is never resolved.
func (o *Object) SetToNull() {
	o.baseObject = NullObject
	o.parser = nil
	o.isDereferenced = true
}

// COSObject returns the receiver.
//
// Note that this is the Base method, which returns this reference. To reach the
// object it refers to, call Object.
func (o *Object) COSObject() Base { return o }

// Accept dispatches to the visitor.
func (o *Object) Accept(v Visitor) error { return v.VisitObject(o) }

// String returns the Java toString form.
func (o *Object) String() string {
	if key := o.Key(); key != nil {
		return "COSObject{" + key.String() + "}"
	}
	return "COSObject{<nil>}"
}
