package common

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// COSArrayList is a list that syncs its contents to a cos.Array.
//
// Port of org.apache.pdfbox.pdmodel.common.COSArrayList, which implements
// java.util.List. Go has no List interface, so the port is a struct carrying
// the same method set; the names follow the Java ones except where Go has an
// established spelling for the same thing.
//
// The element type is not constrained to COSObjectable, because Java's is not
// either: it special-cases String at every point it converts an element to a
// COSBase, and casts to COSObjectable otherwise. An element that is neither
// makes Java throw ClassCastException, so the port panics.
type COSArrayList[E any] struct {
	array  *cos.Array
	actual []E

	// isFiltered indicates that the list has been filtered
	// i.e. the number of entries in array and actual differ
	isFiltered bool

	parentDict *cos.Dictionary
	dictKey    *cos.Name
}

var _ COSObjectable = (*COSArrayList[COSObjectable])(nil)

// NewCOSArrayList is the default constructor.
func NewCOSArrayList[E any]() *COSArrayList[E] {
	return &COSArrayList[E]{array: cos.NewArray()}
}

// NewCOSArrayListOf creates the COSArrayList specifying the list and the
// backing cos.Array.
//
// Users of this constructor need to ensure that the entries in the list and the
// backing cos.Array are matching i.e. the COSObject of the list entry is
// included in the cos.Array.
//
// If the number of entries in the list and the cos.Array differ it is assumed
// that the list has been filtered. In that case the COSArrayList shall only be
// used for reading purposes and no longer for updating.
func NewCOSArrayListOf[E any](actualList []E, cosArray *cos.Array) *COSArrayList[E] {
	l := &COSArrayList[E]{actual: actualList, array: cosArray}
	// if the number of entries differs this may come from a filter being
	// applied at the PDModel level
	if len(l.actual) != l.array.Size() {
		l.isFiltered = true
	}
	return l
}

// NewCOSArrayListOfDictionary is the constructor to be used if the array
// doesn't exist, but is to be created and added to the parent dictionary as
// soon as the first element is added to the array.
func NewCOSArrayListOfDictionary[E any](dictionary *cos.Dictionary, dictionaryKey *cos.Name) *COSArrayList[E] {
	return &COSArrayList[E]{
		array:      cos.NewArray(),
		parentDict: dictionary,
		dictKey:    dictionaryKey,
	}
}

// NewCOSArrayListOfItem is a really special constructor. Sometimes the PDF spec
// says that a dictionary entry can either be a single item or an array of those
// items. But in the PDModel interface we really just want to always return a
// list. In the case were we get the list and never modify it we don't want to
// convert to COSArray and put one element, unless we append to the list. So
// here we are going to create this object with a single item instead of a list,
// but allow more items to be added and then converted to an array.
func NewCOSArrayListOfItem[E any](actualObject E, item cos.Base,
	dictionary *cos.Dictionary, dictionaryKey *cos.Name) *COSArrayList[E] {
	l := &COSArrayList[E]{array: cos.NewArray()}
	l.array.Add(item)
	l.actual = []E{actualObject}
	l.parentDict = dictionary
	l.dictKey = dictionaryKey
	return l
}

// COSObject returns the backing array, so that a COSArrayList can be stored in
// a dictionary the way Java's is.
//
// Java's COSArrayList is not a COSObjectable; COSDictionary.setItem takes one
// and every caller passes toList(). The port declares it so that the same call
// sites read the same way.
func (l *COSArrayList[E]) COSObject() cos.Base { return l.array }

// Size returns the number of elements.
func (l *COSArrayList[E]) Size() int { return len(l.actual) }

// IsEmpty reports whether there are no elements.
func (l *COSArrayList[E]) IsEmpty() bool { return len(l.actual) == 0 }

// Contains reports whether the list holds the given element.
func (l *COSArrayList[E]) Contains(o any) bool { return l.IndexOf(o) >= 0 }

// All is the iterator, which a Go caller ranges over.
func (l *COSArrayList[E]) All(yield func(E) bool) {
	for _, e := range l.actual {
		if !yield(e) {
			return
		}
	}
}

// ToSlice returns a copy of the elements, which is Java's toArray.
func (l *COSArrayList[E]) ToSlice() []E {
	out := make([]E, len(l.actual))
	copy(out, l.actual)
	return out
}

// Add appends an element.
func (l *COSArrayList[E]) Add(o E) bool {
	// when adding if there is a parentDict then change the item
	// in the dictionary from a single item to an array.
	if l.parentDict != nil {
		l.parentDict.SetItem(l.dictKey, l.array)
		// clear the parent dict so it doesn't happen again, there might be
		// a usecase for keeping the parentDict around but not now.
		l.parentDict = nil
	}
	// string is a special case because we can't subclass to be COSObjectable
	l.array.Add(toCOSBase(o))
	l.actual = append(l.actual, o)
	return true
}

// Remove removes the first element equal to o, reporting whether one was found.
//
// Java throws UnsupportedOperationException on a filtered list, which is
// unchecked, so the port panics. Every mutator below does the same.
func (l *COSArrayList[E]) Remove(o any) bool {
	if l.isFiltered {
		panic("removing entries from a filtered List is not permitted")
	}
	retval := true
	index := l.IndexOf(o)
	if index >= 0 {
		l.removeAt(index)
		l.array.RemoveAt(index)
	} else {
		retval = false
	}
	return retval
}

// ContainsAll reports whether every element of c is in the list.
func (l *COSArrayList[E]) ContainsAll(c []E) bool {
	for _, item := range c {
		if !l.Contains(item) {
			return false
		}
	}
	return true
}

// AddAll appends every element of c.
func (l *COSArrayList[E]) AddAll(c []E) bool {
	if l.isFiltered {
		panic("Adding to a filtered List is not permitted")
	}
	// when adding if there is a parentDict then change the item
	// in the dictionary from a single item to an array.
	if l.parentDict != nil && len(c) > 0 {
		l.parentDict.SetItem(l.dictKey, l.array)
		// clear the parent dict so it doesn't happen again, there might be
		// a usecase for keeping the parentDict around but not now.
		l.parentDict = nil
	}
	l.array.AddAll(l.toCOSObjectList(c))
	if len(c) == 0 {
		return false
	}
	l.actual = append(l.actual, c...)
	return true
}

// AddAllAt inserts every element of c at the given index.
func (l *COSArrayList[E]) AddAllAt(index int, c []E) bool {
	if l.isFiltered {
		panic("Inserting to a filtered List is not permitted")
	}
	// when adding if there is a parentDict then change the item
	// in the dictionary from a single item to an array.
	if l.parentDict != nil && len(c) > 0 {
		l.parentDict.SetItem(l.dictKey, l.array)
		// clear the parent dict so it doesn't happen again, there might be
		// a usecase for keeping the parentDict around but not now.
		l.parentDict = nil
	}
	converted := l.toCOSObjectList(c)
	for i, base := range converted {
		l.array.AddAt(index+i, base)
	}
	if len(c) == 0 {
		return false
	}
	l.actual = append(l.actual[:index], append(append([]E{}, c...), l.actual[index:]...)...)
	return true
}

// ConverterToCOSArray converts a list of COSObjectables to a cos.Array.
//
// Port of the static converterToCOSArray. Java takes a raw List and dispatches
// on the runtime type of each element, so the port takes a slice of any and
// does the same; a type it does not know makes Java throw
// IllegalArgumentException, which is unchecked, so the port panics.
func ConverterToCOSArray(cosObjectableList []any) *cos.Array {
	if cosObjectableList == nil {
		return nil
	}
	array := cos.NewArray()
	for _, next := range cosObjectableList {
		switch value := next.(type) {
		case string:
			array.Add(cos.NewStringObj(value))
		case int:
			array.Add(cos.GetInteger(int64(value)))
		case int64:
			array.Add(cos.GetInteger(value))
		case float32:
			array.Add(cos.NewFloat(value))
		case float64:
			array.Add(cos.NewFloat(float32(value)))
		case COSObjectable:
			array.Add(value.COSObject())
		case nil:
			array.Add(cos.NullObject)
		default:
			panic(fmt.Sprintf("Error: Don't know how to convert type to COSBase '%T'", next))
		}
	}
	return array
}

// ConverterFromCOSArrayList reuses the backing array of a COSArrayList rather
// than recreating it, which is the branch converterToCOSArray takes when it is
// handed one. Go cannot express that branch inside ConverterToCOSArray, because
// COSArrayList is generic and the argument there is not.
func ConverterFromCOSArrayList[E any](list *COSArrayList[E]) *cos.Array {
	if list == nil {
		return nil
	}
	return list.array
}

// toCOSObjectList converts the elements to their COS form.
func (l *COSArrayList[E]) toCOSObjectList(list []E) []cos.Base {
	cosObjects := make([]cos.Base, 0, len(list))
	for _, next := range list {
		cosObjects = append(cosObjects, toCOSBase(next))
	}
	return cosObjects
}

// toCOSBase is the String-or-COSObjectable dispatch Java repeats at every point
// it turns an element into a COSBase.
func toCOSBase(o any) cos.Base {
	if s, isString := o.(string); isString {
		return cos.NewStringObj(s)
	}
	// Java casts to COSObjectable and throws ClassCastException otherwise.
	return o.(COSObjectable).COSObject()
}

// RemoveAll removes every element of c, and every indirect reference to one.
func (l *COSArrayList[E]) RemoveAll(c []E) bool {
	for _, item := range c {
		itemCOSBase := any(item).(COSObjectable).COSObject()
		// remove all indirect objects too by dereferencing them
		// before doing the comparison
		for i := l.array.Size() - 1; i >= 0; i-- {
			if itemCOSBase == l.array.GetObject(i) {
				l.array.RemoveAt(i)
			}
		}
	}
	return l.actualRemoveAll(c)
}

// RetainAll removes every element not in c.
func (l *COSArrayList[E]) RetainAll(c []E) bool {
	for _, item := range c {
		itemCOSBase := any(item).(COSObjectable).COSObject()
		// remove all indirect objects too by dereferencing them
		// before doing the comparison
		for i := l.array.Size() - 1; i >= 0; i-- {
			if itemCOSBase != l.array.GetObject(i) {
				l.array.RemoveAt(i)
			}
		}
	}
	return l.actualRetainAll(c)
}

// actualRemoveAll is List.removeAll on the Go slice: it reports whether the
// list changed.
func (l *COSArrayList[E]) actualRemoveAll(c []E) bool {
	kept := l.actual[:0]
	changed := false
	for _, e := range l.actual {
		if containsAny(c, e) {
			changed = true
			continue
		}
		kept = append(kept, e)
	}
	l.actual = kept
	return changed
}

// actualRetainAll is List.retainAll on the Go slice.
func (l *COSArrayList[E]) actualRetainAll(c []E) bool {
	kept := l.actual[:0]
	changed := false
	for _, e := range l.actual {
		if containsAny(c, e) {
			kept = append(kept, e)
			continue
		}
		changed = true
	}
	l.actual = kept
	return changed
}

// containsAny is Collection.contains over a slice, comparing the way Java's
// Object.equals does for the types that reach here: by identity.
func containsAny[E any](list []E, want E) bool {
	for _, e := range list {
		if any(e) == any(want) {
			return true
		}
	}
	return false
}

// Clear removes every element.
func (l *COSArrayList[E]) Clear() {
	// when adding if there is a parentDict then change the item
	// in the dictionary from a single item to an array.
	if l.parentDict != nil {
		l.parentDict.SetItem(l.dictKey, nil)
	}
	l.actual = nil
	l.array.Clear()
}

// Get returns the element at index.
func (l *COSArrayList[E]) Get(index int) E { return l.actual[index] }

// Set replaces the element at index and returns the one that was there.
func (l *COSArrayList[E]) Set(index int, element E) E {
	if l.isFiltered {
		panic("Replacing an element in a filtered List is not permitted")
	}
	item := toCOSBase(element)
	if l.parentDict != nil && index == 0 {
		l.parentDict.SetItem(l.dictKey, item)
	}
	l.array.Set(index, item)
	previous := l.actual[index]
	l.actual[index] = element
	return previous
}

// AddAt inserts an element at index.
func (l *COSArrayList[E]) AddAt(index int, element E) {
	if l.isFiltered {
		panic("Adding an element in a filtered List is not permitted")
	}
	// when adding if there is a parentDict then change the item
	// in the dictionary from a single item to an array.
	if l.parentDict != nil {
		l.parentDict.SetItem(l.dictKey, l.array)
		// clear the parent dict so it doesn't happen again, there might be
		// a usecase for keeping the parentDict around but not now.
		l.parentDict = nil
	}
	l.actual = append(l.actual[:index], append([]E{element}, l.actual[index:]...)...)
	l.array.AddAt(index, toCOSBase(element))
}

// RemoveAt removes the element at index and returns it.
func (l *COSArrayList[E]) RemoveAt(index int) E {
	if l.isFiltered {
		panic("removing entries from a filtered List is not permitted")
	}
	l.array.RemoveAt(index)
	return l.removeAt(index)
}

// removeAt drops the element at index from the Go slice and returns it.
func (l *COSArrayList[E]) removeAt(index int) E {
	removed := l.actual[index]
	l.actual = append(l.actual[:index], l.actual[index+1:]...)
	return removed
}

// IndexOf returns the index of the first element equal to o, or -1.
func (l *COSArrayList[E]) IndexOf(o any) int {
	for i, e := range l.actual {
		if any(e) == o {
			return i
		}
	}
	return -1
}

// LastIndexOf returns the index of the last element equal to o, or -1.
func (l *COSArrayList[E]) LastIndexOf(o any) int {
	for i := len(l.actual) - 1; i >= 0; i-- {
		if any(l.actual[i]) == o {
			return i
		}
	}
	return -1
}

// SubList returns the elements between the two indices.
//
// Java returns a view onto the list; the port returns the Go slice, which is
// also a view, so a write through it is seen by both, as in Java.
func (l *COSArrayList[E]) SubList(fromIndex, toIndex int) []E {
	return l.actual[fromIndex:toIndex]
}

// String returns the Java toString form.
func (l *COSArrayList[E]) String() string {
	return "COSArrayList{" + l.array.String() + "}"
}

// ToList returns the underlying cos.Array.
func (l *COSArrayList[E]) ToList() *cos.Array { return l.array }
