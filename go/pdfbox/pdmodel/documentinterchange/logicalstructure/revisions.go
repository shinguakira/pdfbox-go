package logicalstructure

import (
	"fmt"
	"strings"
)

// Revisions is a list of objects and the revision number of each.
//
// Port of Revisions. Java's element type is unconstrained; the port asks for a
// comparable one, because setRevisionNumber looks an object up with
// List.indexOf, which is equals. That is reference equality for an attribute
// object (PDDictionaryWrapper.equals compares the wrapped dictionaries, which
// COSDictionary does not override) and value equality for a class name.
type Revisions[T comparable] struct {
	objects         []T
	revisionNumbers []int
}

// NewRevisions builds an empty list.
func NewRevisions[T comparable]() *Revisions[T] {
	return &Revisions[T]{}
}

// Object returns the object at index.
func (r *Revisions[T]) Object(index int) T { return r.objects[index] }

// RevisionNumber returns the revision number at index.
func (r *Revisions[T]) RevisionNumber(index int) int { return r.revisionNumbers[index] }

// AddObject appends an object with its revision number.
func (r *Revisions[T]) AddObject(object T, revisionNumber int) {
	r.objects = append(r.objects, object)
	r.revisionNumbers = append(r.revisionNumbers, revisionNumber)
}

// SetRevisionNumber sets the revision number of an object already in the list,
// and does nothing when it is not in it. Java declares it protected.
func (r *Revisions[T]) SetRevisionNumber(object T, revisionNumber int) {
	for i, candidate := range r.objects {
		if candidate == object {
			r.revisionNumbers[i] = revisionNumber
			return
		}
	}
}

// Size returns how many objects the list holds.
func (r *Revisions[T]) Size() int { return len(r.objects) }

// String renders the list the way Java's toString does.
func (r *Revisions[T]) String() string {
	parts := make([]string, len(r.objects))
	for i, object := range r.objects {
		parts[i] = fmt.Sprintf("object=%v, revisionNumber=%d", object, r.revisionNumbers[i])
	}
	return "{" + strings.Join(parts, "; ") + "}"
}
