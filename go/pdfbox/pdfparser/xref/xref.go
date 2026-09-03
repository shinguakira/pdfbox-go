// Package xref models the entries of a PDF cross-reference table or stream.
//
// Port of org.apache.pdfbox.pdfparser.xref. An entry says where to find one
// indirect object: at a byte offset, inside an object stream, or nowhere
// because it is free.
package xref

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Type is the kind of a cross-reference entry.
//
// Port of the enum XReferenceType. The numeric values are the first column of a
// cross-reference stream, so they are part of the file format and cannot be
// renumbered.
type Type int

const (
	// TypeFree marks an object that is not in use.
	TypeFree Type = 0
	// TypeNormal marks an object stored at a byte offset.
	TypeNormal Type = 1
	// TypeObjectStream marks an object stored inside an object stream.
	TypeObjectStream Type = 2
)

// NumericValue returns the value written in the type column.
func (t Type) NumericValue() int { return int(t) }

// String returns the Java enum constant name.
func (t Type) String() string {
	switch t {
	case TypeFree:
		return "FREE"
	case TypeNormal:
		return "NORMAL"
	case TypeObjectStream:
		return "OBJECT_STREAM_ENTRY"
	}
	return fmt.Sprintf("Type(%d)", int(t))
}

// Entry is one cross-reference entry.
//
// Port of the interface XReferenceEntry. The three column values are what a
// cross-reference stream stores; their meaning depends on the type.
type Entry interface {
	// Type returns the kind of entry.
	Type() Type

	// ReferencedKey returns the object this entry is about.
	ReferencedKey() *cos.ObjectKey

	// FirstColumnValue returns the type number.
	FirstColumnValue() int64

	// SecondColumnValue returns the byte offset, the next free object, or the
	// object number of the containing stream, by type.
	SecondColumnValue() int64

	// ThirdColumnValue returns the generation, or the index within the
	// containing object stream.
	ThirdColumnValue() int64
}

// base carries what every entry shares.
//
// Port of the abstract class AbstractXReference, which Java subclasses; the
// port embeds this struct instead.
type base struct {
	typ Type
}

// Type returns the kind of entry.
func (b base) Type() Type { return b.typ }

// FirstColumnValue returns the type number.
func (b base) FirstColumnValue() int64 { return int64(b.typ.NumericValue()) }

// Compare orders entries by the object they reference.
//
// Port of AbstractXReference.compareTo, which handles two null cases: an entry
// whose own key is nil returns -1, and a nil or keyless argument returns 1. In
// ascending order both put the keyless entry first.
//
// Java's compareTo is an instance method, so its receiver can never be null —
// calling it on one throws. The a == nil branch here has no Java counterpart
// and is defensive only; it sorts a nil receiver first, consistent with the
// keyless case.
func Compare(a, b Entry) int {
	if a == nil || a.ReferencedKey() == nil {
		return -1
	}
	if b == nil || b.ReferencedKey() == nil {
		return 1
	}
	return a.ReferencedKey().Compare(b.ReferencedKey())
}

// FreeReference is an entry for an object that is not in use.
//
// Port of FreeXReference.
type FreeReference struct {
	base
	key            *cos.ObjectKey
	nextFreeObject int64
}

var _ Entry = FreeReference{}

// NewFreeReference returns a free entry pointing at the next free object.
func NewFreeReference(key *cos.ObjectKey, nextFreeObject int64) FreeReference {
	return FreeReference{
		base:           base{typ: TypeFree},
		key:            key,
		nextFreeObject: nextFreeObject,
	}
}

// NullEntry is the head of the free list that every PDF carries: object 0,
// generation 65535, pointing at object 0.
var NullEntry = NewFreeReference(mustKey(0, 65535), 0)

// mustKey builds a key that is known to be valid, for the package-level
// NullEntry. Java writes new COSObjectKey(0, 65535) inline, where the
// constructor cannot fail for these arguments.
func mustKey(num int64, gen int) *cos.ObjectKey {
	key, err := cos.NewObjectKey(num, gen)
	if err != nil {
		panic("xref: " + err.Error())
	}
	return key
}

// ReferencedKey returns the object this entry frees.
func (f FreeReference) ReferencedKey() *cos.ObjectKey { return f.key }

// SecondColumnValue returns the next free object number.
func (f FreeReference) SecondColumnValue() int64 { return f.nextFreeObject }

// ThirdColumnValue returns the generation of the freed object.
func (f FreeReference) ThirdColumnValue() int64 {
	if f.key == nil {
		return 0
	}
	return int64(f.key.Generation())
}

// String returns the Java toString form.
func (f FreeReference) String() string {
	return fmt.Sprintf("FreeReference{key=%v, nextFreeObject=%d, type=%v}",
		f.key, f.nextFreeObject, f.typ)
}

// NormalReference is an entry for an object stored at a byte offset.
//
// Port of NormalXReference.
type NormalReference struct {
	base
	byteOffset   int64
	key          *cos.ObjectKey
	object       cos.Base
	objectStream bool
}

var _ Entry = NormalReference{}

// NewNormalReference returns an entry for an object at the given offset.
//
// The object is inspected to see whether it is an object stream, which the
// writer needs to know because the objects inside one are numbered differently.
// An indirect reference is resolved first, as Java does.
func NewNormalReference(byteOffset int64, key *cos.ObjectKey, object cos.Base) NormalReference {
	resolved := object
	if ref, ok := object.(*cos.Object); ok {
		resolved = ref.Object()
	}

	isObjectStream := false
	if stream, ok := resolved.(*cos.Stream); ok {
		isObjectStream = stream.GetCOSName(cos.Type) == cos.ObjStm
	}

	return NormalReference{
		base:         base{typ: TypeNormal},
		byteOffset:   byteOffset,
		key:          key,
		object:       object,
		objectStream: isObjectStream,
	}
}

// ByteOffset returns where the object starts in the file.
func (n NormalReference) ByteOffset() int64 { return n.byteOffset }

// Object returns the object this entry points at.
func (n NormalReference) Object() cos.Base { return n.object }

// IsObjectStream reports whether the referenced object is an object stream.
func (n NormalReference) IsObjectStream() bool { return n.objectStream }

// ReferencedKey returns the object this entry locates.
func (n NormalReference) ReferencedKey() *cos.ObjectKey { return n.key }

// SecondColumnValue returns the byte offset.
func (n NormalReference) SecondColumnValue() int64 { return n.byteOffset }

// ThirdColumnValue returns the generation.
func (n NormalReference) ThirdColumnValue() int64 {
	if n.key == nil {
		return 0
	}
	return int64(n.key.Generation())
}

// String returns the Java toString form.
func (n NormalReference) String() string {
	return fmt.Sprintf("NormalReference{key=%v, byteOffset=%d, type=%v}",
		n.key, n.byteOffset, n.typ)
}

// ObjectStreamReference is an entry for an object stored inside an object
// stream.
//
// Port of ObjectStreamXReference.
type ObjectStreamReference struct {
	base
	key               *cos.ObjectKey
	objectStreamKey   *cos.ObjectKey
	objectStreamIndex int
}

var _ Entry = ObjectStreamReference{}

// NewObjectStreamReference returns an entry for an object at the given index of
// the object stream named by objectStreamKey.
func NewObjectStreamReference(key, objectStreamKey *cos.ObjectKey, index int) ObjectStreamReference {
	return ObjectStreamReference{
		base:              base{typ: TypeObjectStream},
		key:               key,
		objectStreamKey:   objectStreamKey,
		objectStreamIndex: index,
	}
}

// ReferencedKey returns the object this entry locates.
func (o ObjectStreamReference) ReferencedKey() *cos.ObjectKey { return o.key }

// ObjectStreamKey returns the object stream that contains it.
func (o ObjectStreamReference) ObjectStreamKey() *cos.ObjectKey { return o.objectStreamKey }

// ObjectStreamIndex returns the index within that stream.
func (o ObjectStreamReference) ObjectStreamIndex() int { return o.objectStreamIndex }

// SecondColumnValue returns the object number of the containing stream.
func (o ObjectStreamReference) SecondColumnValue() int64 {
	if o.objectStreamKey == nil {
		return 0
	}
	return o.objectStreamKey.Number()
}

// ThirdColumnValue returns the index within the containing stream.
func (o ObjectStreamReference) ThirdColumnValue() int64 { return int64(o.objectStreamIndex) }

// String returns the Java toString form.
func (o ObjectStreamReference) String() string {
	return fmt.Sprintf("ObjectStreamReference{key=%v, objectStream=%v, index=%d, type=%v}",
		o.key, o.objectStreamKey, o.objectStreamIndex, o.typ)
}
