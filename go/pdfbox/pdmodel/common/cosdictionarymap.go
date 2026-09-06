package common

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// COSDictionaryMap is a map that syncs its contents to a cos.Dictionary.
//
// Port of org.apache.pdfbox.pdmodel.common.COSDictionaryMap, which implements
// java.util.Map. Go has no Map interface, so the port is a struct carrying the
// same method set.
//
// The key type is String in every use Java makes of it --- put and remove both
// cast the key to String to build a COSName --- so the port fixes it there
// rather than carrying a type parameter that only one type can satisfy.
type COSDictionaryMap[V any] struct {
	m       *cos.Dictionary
	actuals map[string]V
	// keys keeps the insertion order, which Java's HashMap does not have but
	// which keySet and values must be deterministic over for a Go caller.
	keys []string
}

// NewCOSDictionaryMap creates a map over the given dictionary.
func NewCOSDictionaryMap[V any](actualsMap map[string]V, dicMap *cos.Dictionary) *COSDictionaryMap[V] {
	m := &COSDictionaryMap[V]{actuals: actualsMap, m: dicMap}
	for _, key := range dicMap.KeySet() {
		if _, ok := actualsMap[key.Name()]; ok {
			m.keys = append(m.keys, key.Name())
		}
	}
	return m
}

// Size returns the number of entries of the dictionary.
//
// Java returns the dictionary's size, not the map's; the two can differ where
// the map was built from a dictionary holding an entry the conversion skipped.
func (m *COSDictionaryMap[V]) Size() int { return m.m.Size() }

// IsEmpty reports whether the dictionary has no entries.
func (m *COSDictionaryMap[V]) IsEmpty() bool { return m.Size() == 0 }

// ContainsKey reports whether the map holds the given key.
func (m *COSDictionaryMap[V]) ContainsKey(key string) bool {
	_, ok := m.actuals[key]
	return ok
}

// Get returns the value under key, and whether it was there.
func (m *COSDictionaryMap[V]) Get(key string) (V, bool) {
	value, ok := m.actuals[key]
	return value, ok
}

// Put stores a value, writing its COS form into the dictionary.
//
// Java casts the value to COSObjectable and throws ClassCastException where it
// is not one, so the port panics.
func (m *COSDictionaryMap[V]) Put(key string, value V) (V, bool) {
	object := any(value).(COSObjectable)
	m.m.SetItem(cos.GetPDFName(key), object.COSObject())
	previous, existed := m.actuals[key]
	if !existed {
		m.keys = append(m.keys, key)
	}
	m.actuals[key] = value
	return previous, existed
}

// Remove drops the entry under key from the map and the dictionary.
func (m *COSDictionaryMap[V]) Remove(key string) (V, bool) {
	m.m.RemoveItem(cos.GetPDFName(key))
	previous, existed := m.actuals[key]
	if existed {
		delete(m.actuals, key)
		for i, k := range m.keys {
			if k == key {
				m.keys = append(m.keys[:i], m.keys[i+1:]...)
				break
			}
		}
	}
	return previous, existed
}

// PutAll is not implemented, as it is not in Java.
//
// Java throws UnsupportedOperationException, which is unchecked, so the port
// panics.
func (m *COSDictionaryMap[V]) PutAll(t map[string]V) {
	panic("Not yet implemented")
}

// Clear removes every entry.
func (m *COSDictionaryMap[V]) Clear() {
	m.m.Clear()
	clear(m.actuals)
	m.keys = nil
}

// KeySet returns the keys.
func (m *COSDictionaryMap[V]) KeySet() []string {
	out := make([]string, len(m.keys))
	copy(out, m.keys)
	return out
}

// Values returns the values.
func (m *COSDictionaryMap[V]) Values() []V {
	out := make([]V, 0, len(m.keys))
	for _, k := range m.keys {
		out = append(out, m.actuals[k])
	}
	return out
}

// Equals reports whether the other map is over an equal dictionary, which is
// what Java's equals compares.
func (m *COSDictionaryMap[V]) Equals(o any) bool {
	other, ok := o.(*COSDictionaryMap[V])
	return ok && other.m == m.m
}

// String returns the Java toString form, which is the map's.
func (m *COSDictionaryMap[V]) String() string {
	return fmt.Sprintf("%v", m.actuals)
}

// ConvertToDictionary builds a dictionary out of a map of COSObjectables.
//
// Port of the static convert(Map<String, ?>).
func ConvertToDictionary(someMap map[string]COSObjectable) *cos.Dictionary {
	dic := cos.NewDictionary()
	for _, name := range sortedKeys(someMap) {
		dic.SetItem(cos.GetPDFName(name), someMap[name].COSObject())
	}
	return dic
}

// ConvertBasicTypesToMap reads a dictionary of scalars into a map.
//
// Port of the static convertBasicTypesToMap(COSDictionary). The values are
// Java's Object: a String, an Integer, a name's String, a Float or a Boolean.
func ConvertBasicTypesToMap(m *cos.Dictionary) (*COSDictionaryMap[any], error) {
	if m == nil {
		return nil, nil
	}
	actualMap := map[string]any{}
	for _, key := range m.KeySet() {
		cosObj := m.GetDictionaryObject(key)
		var actualObject any
		switch value := cosObj.(type) {
		case *cos.StringObj:
			actualObject = value.Value()
		case *cos.Integer:
			actualObject = int(value.IntValue())
		case *cos.Name:
			actualObject = value.Name()
		case *cos.Float:
			actualObject = value.FloatValue()
		case *cos.Boolean:
			actualObject = value.Value()
		default:
			return nil, fmt.Errorf("Error:unknown type of object to convert:%v", cosObj)
		}
		actualMap[key.Name()] = actualObject
	}
	return NewCOSDictionaryMap(actualMap, m), nil
}
