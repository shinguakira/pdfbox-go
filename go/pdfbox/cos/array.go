package cos

import (
	"fmt"
	"strings"
)

// Array is an array of COS objects.
//
// Port of org.apache.pdfbox.cos.COSArray. Entries may be nil, which is what the
// Java list holds for an absent element, and may be an *Object standing for an
// indirect reference — see Get versus GetObject.
//
// Not yet ported: the COSObjectable-taking overloads, which need pdmodel. See
// migration/STATUS.md.
type Array struct {
	object
	updateInfoState
	objects []Base
}

var _ Base = (*Array)(nil)
var _ UpdateInfo = (*Array)(nil)

// UpdateState returns the current UpdateState of this Array.
func (a *Array) UpdateState() *UpdateState { return a.state(a) }

// IsNeedToBeUpdated gets the update state for the COSWriter.
func (a *Array) IsNeedToBeUpdated() bool { return a.UpdateState().IsUpdated() }

// SetNeedToBeUpdated sets the update state for the COSWriter.
func (a *Array) SetNeedToBeUpdated(flag bool) { a.UpdateState().updateTo(flag) }

// ToIncrement uses this Array as the base object of a new Increment.
func (a *Array) ToIncrement() *Increment { return a.UpdateState().toIncrement() }

// maybeWrap stores a dictionary or array that already has a key and is not
// direct as an indirect reference to it, so that the writer emits it once
// rather than inline at every use.
//
// Port of the private COSArray.maybeWrap.
func maybeWrap(object Base) Base {
	if object != nil && isWrappable(object) {
		return NewObjectWithKey(object, object.Key())
	}
	return object
}

// NewArray returns an empty array.
func NewArray() *Array {
	return &Array{}
}

// NewArrayOf returns an array holding the given objects. The slice is copied.
//
// Port of COSArray(List). Java throws NullPointerException for a null list; a
// nil slice here yields an empty array, which is what a Go caller means by it.
func NewArrayOf(objects []Base) *Array {
	a := &Array{objects: make([]Base, len(objects))}
	copy(a.objects, objects)
	return a
}

// ArrayOfFloats returns an array of floats.
//
// Port of the static COSArray.of(float...).
func ArrayOfFloats(values []float32) *Array {
	a := &Array{objects: make([]Base, 0, len(values))}
	for _, v := range values {
		a.objects = append(a.objects, NewFloat(v))
	}
	return a
}

// ArrayOfIntegers returns an array of integers.
//
// Port of the static COSArray.ofCOSIntegers.
func ArrayOfIntegers(values []int) *Array {
	a := &Array{objects: make([]Base, 0, len(values))}
	for _, v := range values {
		a.objects = append(a.objects, GetInteger(int64(v)))
	}
	return a
}

// ArrayOfNames returns an array of names.
//
// Port of the static COSArray.ofCOSNames.
func ArrayOfNames(values []string) *Array {
	a := &Array{objects: make([]Base, 0, len(values))}
	for _, v := range values {
		a.objects = append(a.objects, GetPDFName(v))
	}
	return a
}

// ArrayOfStrings returns an array of strings.
//
// Port of the static COSArray.ofCOSStrings.
func ArrayOfStrings(values []string) *Array {
	a := &Array{objects: make([]Base, 0, len(values))}
	for _, v := range values {
		a.objects = append(a.objects, NewStringObj(v))
	}
	return a
}

// Size returns the number of entries.
func (a *Array) Size() int { return len(a.objects) }

// IsEmpty reports whether the array has no entries.
func (a *Array) IsEmpty() bool { return len(a.objects) == 0 }

// Add appends an object.
func (a *Array) Add(object Base) {
	objectToAdd := maybeWrap(object)
	a.objects = append(a.objects, objectToAdd)
	a.UpdateState().updateChild(objectToAdd)
}

// AddAt inserts an object at the given index.
func (a *Array) AddAt(i int, object Base) {
	objectToAdd := maybeWrap(object)
	a.objects = append(a.objects, nil)
	copy(a.objects[i+1:], a.objects[i:])
	a.objects[i] = objectToAdd
	a.UpdateState().updateChild(objectToAdd)
}

// AddAll appends every object in the slice.
//
// Java updates only when List.addAll reports a change, which for an addAll is
// exactly when the collection was not empty.
func (a *Array) AddAll(objects []Base) {
	if len(objects) == 0 {
		return
	}
	a.objects = append(a.objects, objects...)
	a.UpdateState().updateChildren(objects)
}

// AddArray appends every entry of another array.
func (a *Array) AddArray(other *Array) {
	if other == nil {
		return
	}
	if len(other.objects) == 0 {
		return
	}
	a.objects = append(a.objects, other.objects...)
	a.UpdateState().updateChildren(other.ToList())
}

// Clear removes every entry.
func (a *Array) Clear() {
	a.objects = a.objects[:0]
	a.UpdateState().update()
}

// Get returns the raw entry at index, which may be nil or an indirect
// reference. Use GetObject to resolve references.
func (a *Array) Get(index int) Base {
	return a.objects[index]
}

// GetObject returns the entry at index, resolving an indirect reference and
// mapping the null object to nil.
//
// Port of getObject(int).
func (a *Array) GetObject(index int) Base {
	obj := a.objects[index]
	if ref, ok := obj.(*Object); ok {
		obj = ref.Object()
	}
	if _, isNull := obj.(*Null); isNull {
		return nil
	}
	return obj
}

// Set replaces the entry at index.
func (a *Array) Set(index int, object Base) {
	objectToAdd := maybeWrap(object)
	a.objects[index] = objectToAdd
	a.UpdateState().updateChild(objectToAdd)
}

// SetInt stores an integer at index.
//
// Java has its own set(int, int) rather than routing through set(int, COSBase),
// so it updates without a child.
func (a *Array) SetInt(index, value int) {
	a.objects[index] = GetInteger(int64(value))
	a.UpdateState().update()
}

// SetName stores a name at index.
func (a *Array) SetName(index int, name string) {
	a.Set(index, GetPDFName(name))
}

// SetString stores a string at index.
//
// Port of setString(int, String). Java stores null for a null argument; Go has
// no null string, and an empty one is a value rather than an absence — a caller
// wanting Java's null calls Set(index, nil).
func (a *Array) SetString(index int, text string) {
	a.Set(index, NewStringObj(text))
}

// GetInt returns the integer at index, or -1 when it is not a number.
//
// Port of getInt(int), which defaults to -1.
func (a *Array) GetInt(index int) int {
	return a.GetIntDefault(index, -1)
}

// GetIntDefault returns the integer at index, or defaultValue when the entry is
// out of range or is not a number.
func (a *Array) GetIntDefault(index, defaultValue int) int {
	if index >= a.Size() {
		return defaultValue
	}
	if n, ok := a.objects[index].(Number); ok {
		return n.IntValue()
	}
	return defaultValue
}

// GetName returns the name at index, or defaultValue when the entry is out of
// range or is not a name.
func (a *Array) GetName(index int, defaultValue string) string {
	if index >= a.Size() {
		return defaultValue
	}
	if n, ok := a.objects[index].(*Name); ok {
		return n.Name()
	}
	return defaultValue
}

// GetString returns the string at index, or defaultValue when the entry is out
// of range or is not a string.
func (a *Array) GetString(index int, defaultValue string) string {
	if index >= a.Size() {
		return defaultValue
	}
	if s, ok := a.objects[index].(*StringObj); ok {
		return s.Value()
	}
	return defaultValue
}

// RemoveAt removes and returns the entry at index.
func (a *Array) RemoveAt(index int) Base {
	removed := a.objects[index]
	a.objects = append(a.objects[:index], a.objects[index+1:]...)
	a.UpdateState().update()
	return removed
}

// Remove removes the first entry equal to object, reporting whether one was
// found. It compares raw entries only.
//
// Port of remove(COSBase).
func (a *Array) Remove(object Base) bool {
	for i, item := range a.objects {
		if cosEqual(item, object) {
			a.RemoveAt(i)
			return true
		}
	}
	return false
}

// RemoveObject removes the first entry equal to object, resolving indirect
// references while looking.
//
// Port of removeObject(COSBase).
func (a *Array) RemoveObject(object Base) bool {
	if i := a.IndexOfObject(object); i >= 0 {
		a.RemoveAt(i)
		return true
	}
	return false
}

// RemoveAll removes every entry equal to one in the slice.
func (a *Array) RemoveAll(objects []Base) {
	kept := a.objects[:0]
	for _, item := range a.objects {
		if !containsBase(objects, item) {
			kept = append(kept, item)
		}
	}
	a.objects = kept
	// Java updates whether or not anything was removed.
	a.UpdateState().update()
}

// RetainAll removes every entry not equal to one in the slice.
func (a *Array) RetainAll(objects []Base) {
	sizeBefore := len(a.objects)
	kept := a.objects[:0]
	for _, item := range a.objects {
		if containsBase(objects, item) {
			kept = append(kept, item)
		}
	}
	a.objects = kept
	// Java updates only when List.retainAll reports a change.
	if len(a.objects) != sizeBefore {
		a.UpdateState().update()
	}
}

// IndexOf returns the index of the first entry equal to object, or -1.
// It compares raw entries and does not resolve indirect references.
func (a *Array) IndexOf(object Base) int {
	for i, item := range a.objects {
		if cosEqual(item, object) {
			return i
		}
	}
	return -1
}

// IndexOfObject returns the index of the first entry equal to object,
// resolving indirect references while looking.
func (a *Array) IndexOfObject(object Base) int {
	for i, item := range a.objects {
		if cosEqual(item, object) {
			return i
		}
		if ref, ok := item.(*Object); ok {
			if resolved := ref.Object(); resolved != nil && cosEqual(resolved, object) {
				return i
			}
		}
	}
	return -1
}

// GrowToSize pads the array with nil entries until it holds size of them.
func (a *Array) GrowToSize(size int) {
	a.GrowToSizeWith(size, nil)
}

// GrowToSizeWith pads the array with the given object until it holds size
// entries. An array already that long is left alone.
func (a *Array) GrowToSizeWith(size int, object Base) {
	for a.Size() < size {
		a.Add(object)
	}
	a.UpdateState().update()
}

// ToList returns a copy of the entries.
func (a *Array) ToList() []Base {
	out := make([]Base, len(a.objects))
	copy(out, a.objects)
	return out
}

// ToFloatArray returns the numeric entries as floats. A nil or non-numeric
// entry becomes 0, as it does in Java.
//
// An indirect reference is resolved first: Java reads through getObject, and
// reading the raw entry instead would find a reference rather than a number and
// silently yield zero.
func (a *Array) ToFloatArray() []float32 {
	out := make([]float32, len(a.objects))
	for i := range a.objects {
		if n, ok := a.GetObject(i).(Number); ok {
			out[i] = n.FloatValue()
		}
	}
	return out
}

// SetFloatArray replaces the contents with the given floats.
func (a *Array) SetFloatArray(values []float32) {
	a.Clear()
	for _, v := range values {
		a.Add(NewFloat(v))
	}
}

// ToNameStringList returns the entries as name strings.
//
// Java casts each entry to COSName and calls getName on it, so an entry that is
// not a name raises ClassCastException and a null entry raises
// NullPointerException. Both are unchecked, so the port panics.
func (a *Array) ToNameStringList() []string {
	out := make([]string, len(a.objects))
	for i, item := range a.objects {
		n, isName := item.(*Name)
		if !isName {
			panic(fmt.Sprintf("cos: %T cannot be cast to COSName", item))
		}
		out[i] = n.Name()
	}
	return out
}

// ToStringStringList returns the entries as string values, with nil for an
// entry that is not a string.
func (a *Array) ToStringStringList() []*string {
	out := make([]*string, len(a.objects))
	for i, item := range a.objects {
		if s, ok := item.(*StringObj); ok {
			v := s.Value()
			out[i] = &v
		}
	}
	return out
}

// ToNumberFloatList returns the entries as floats, with nil for an entry that
// is not a number.
func (a *Array) ToNumberFloatList() []*float32 {
	out := make([]*float32, len(a.objects))
	for i := range a.objects {
		if n, ok := a.GetObject(i).(Number); ok {
			v := n.FloatValue()
			out[i] = &v
		}
	}
	return out
}

// ToNumberIntegerList returns the entries as ints, with nil for an entry that
// is not a number.
func (a *Array) ToNumberIntegerList() []*int {
	out := make([]*int, len(a.objects))
	for i := range a.objects {
		if n, ok := a.GetObject(i).(Number); ok {
			v := n.IntValue()
			out[i] = &v
		}
	}
	return out
}

// All returns an iterator over the raw entries, for use with range.
//
// Port of the Iterable<COSBase> the Java class implements.
func (a *Array) All(yield func(Base) bool) {
	for _, item := range a.objects {
		if !yield(item) {
			return
		}
	}
}

// COSObject returns the receiver.
func (a *Array) COSObject() Base { return a }

// Accept dispatches to the visitor.
func (a *Array) Accept(v Visitor) error { return v.VisitArray(a) }

// String returns the Java toString form.
func (a *Array) String() string {
	var sb strings.Builder
	sb.WriteString("COSArray{")
	for i, item := range a.objects {
		if i > 0 {
			sb.WriteString(", ")
		}
		if item == nil {
			sb.WriteString("null")
			continue
		}
		sb.WriteString(baseString(item))
	}
	sb.WriteString("}")
	return sb.String()
}

// Equal reports whether two COS objects are equal the way Java's equals does:
// a name, a number, a string or a boolean by value, everything else by
// identity, since COSDictionary, COSArray and COSStream do not override equals.
func Equal(a, b Base) bool { return cosEqual(a, b) }

// cosEqual compares two COS values by content where the type defines equality,
// and by identity otherwise.
//
// Java relies on each class overriding equals. Go has no such single hook, so
// this dispatches to the Equals method of the types that have one.
func cosEqual(a, b Base) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch x := a.(type) {
	case *Name:
		y, ok := b.(*Name)
		return ok && x.Equals(y)
	case *Integer:
		y, ok := b.(*Integer)
		return ok && x.Equals(y)
	case *Float:
		y, ok := b.(*Float)
		return ok && x.Equals(y)
	case *StringObj:
		y, ok := b.(*StringObj)
		return ok && x.Equals(y)
	default:
		return a == b
	}
}

func containsBase(objects []Base, target Base) bool {
	for _, o := range objects {
		if cosEqual(o, target) {
			return true
		}
	}
	return false
}

// baseString renders a value for String, using its own String method when it
// has one.
func baseString(b Base) string {
	if s, ok := b.(interface{ String() string }); ok {
		return s.String()
	}
	return "?"
}

// resetObjectKeys collects all indirect objects numbers within this array and
// all included structures, resetting them.
//
// Port of the protected COSArray.resetObjectKeys.
func (a *Array) resetObjectKeys(indirectObjects map[int64]bool) map[int64]bool {
	if indirectObjects == nil {
		return indirectObjects
	}
	if key := a.Key(); key != nil {
		// avoid endless recursions
		if indirectObjects[key.InternalHash()] {
			return indirectObjects
		}
		indirectObjects[key.InternalHash()] = true
		// reset key
		a.SetKey(nil)
	}
	for _, cosBase := range a.objects {
		if cosBase == nil {
			continue
		}
		var indirectObjectKey *ObjectKey
		if reference, ok := cosBase.(*Object); ok {
			indirectObjectKey = reference.Key()
		}
		if indirectObjectKey != nil {
			if indirectObjects[indirectObjectKey.InternalHash()] {
				continue
			}
			dereferencedObject := cosBase.(*Object).Object()
			// reset key
			cosBase.SetKey(nil)
			cosBase = dereferencedObject
		}
		switch value := cosBase.(type) {
		case *Stream:
			// COSStream is a COSDictionary in Java, so it takes the same branch.
			value.resetObjectKeys(indirectObjects)
		case *Dictionary:
			// descend to included dictionary to reset all included indirect objects
			value.resetObjectKeys(indirectObjects)
		case *Array:
			// descend to included array to reset all included indirect objects
			value.resetObjectKeys(indirectObjects)
		default:
			if indirectObjectKey != nil {
				// add key for all indirect objects other than COSDictionary/COSArray
				indirectObjects[indirectObjectKey.InternalHash()] = true
			}
		}
	}
	return indirectObjects
}
