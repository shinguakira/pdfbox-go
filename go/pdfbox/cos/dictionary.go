package cos

import (
	"strconv"
	"strings"
)

// Dictionary is a PDF dictionary: an ordered map from Name to COS object.
//
// Port of org.apache.pdfbox.cos.COSDictionary. Java backs it with a
// LinkedHashMap, so entries come back in insertion order and a dictionary is
// written out the way it was read. Go maps have no order, so the port keeps the
// key order in a slice alongside the map.
//
// Dictionaries compare by identity. Java achieves that by not overriding
// equals, which is what keeps a COSStream from comparing equal to a
// COSDictionary holding the same entries; comparing *Dictionary with == does
// the same here.
//
// Not yet ported: the date accessors, which need a date parser from
// pdfbox/util; getCOSStream, which needs Stream; the COSObjectable overloads,
// which need pdmodel; and the COSUpdateState for incremental saves. See
// migration/STATUS.md.
type Dictionary struct {
	object
	items map[*Name]Base
	// keys holds the insertion order of items.
	keys []*Name
}

var _ Base = (*Dictionary)(nil)

// NewDictionary returns an empty dictionary.
func NewDictionary() *Dictionary {
	return &Dictionary{items: make(map[*Name]Base)}
}

// NewDictionaryFrom returns a shallow copy of another dictionary, preserving
// key order.
//
// Port of COSDictionary(COSDictionary).
func NewDictionaryFrom(other *Dictionary) *Dictionary {
	d := NewDictionary()
	if other != nil {
		d.AddAll(other)
	}
	return d
}

// Size returns the number of entries.
func (d *Dictionary) Size() int { return len(d.items) }

// Clear removes every entry.
func (d *Dictionary) Clear() {
	d.items = make(map[*Name]Base)
	d.keys = d.keys[:0]
}

// SetItem stores value under key. A nil value removes the entry, which is what
// Java does when setItem is given null.
func (d *Dictionary) SetItem(key *Name, value Base) {
	if value == nil {
		d.RemoveItem(key)
		return
	}
	if _, exists := d.items[key]; !exists {
		d.keys = append(d.keys, key)
	}
	d.items[key] = value
}

// RemoveItem removes the entry under key, if any.
func (d *Dictionary) RemoveItem(key *Name) {
	if _, exists := d.items[key]; !exists {
		return
	}
	delete(d.items, key)
	for i, k := range d.keys {
		if k == key {
			d.keys = append(d.keys[:i], d.keys[i+1:]...)
			break
		}
	}
}

// GetItem returns the raw entry under key, which may be an indirect reference
// or nil. Use GetDictionaryObject to resolve references.
func (d *Dictionary) GetItem(key *Name) Base {
	return d.items[key]
}

// GetItem2 returns the raw entry under firstKey, falling back to secondKey.
//
// Port of getItem(COSName, COSName), which exists for the abbreviated keys that
// inline images use.
func (d *Dictionary) GetItem2(firstKey, secondKey *Name) Base {
	if v, ok := d.items[firstKey]; ok {
		return v
	}
	return d.items[secondKey]
}

// GetDictionaryObject returns the entry under key, resolving an indirect
// reference and mapping the null object to nil.
func (d *Dictionary) GetDictionaryObject(key *Name) Base {
	return resolve(d.items[key])
}

// GetDictionaryObject2 returns the entry under firstKey, falling back to
// secondKey, resolving references.
func (d *Dictionary) GetDictionaryObject2(firstKey, secondKey *Name) Base {
	if v, ok := d.items[firstKey]; ok {
		return resolve(v)
	}
	return resolve(d.items[secondKey])
}

// resolve dereferences an indirect reference and maps the null object to nil.
func resolve(value Base) Base {
	if ref, ok := value.(*Object); ok {
		value = ref.Object()
	}
	if _, isNull := value.(*Null); isNull {
		return nil
	}
	return value
}

// ContainsKey reports whether the dictionary holds an entry under key.
func (d *Dictionary) ContainsKey(key *Name) bool {
	_, ok := d.items[key]
	return ok
}

// ContainsValue reports whether any entry holds value, resolving indirect
// references.
func (d *Dictionary) ContainsValue(value Base) bool {
	return d.KeyForValue(value) != nil
}

// KeyForValue returns the first key whose entry holds value, or nil.
//
// Port of getKeyForValue. It matches against the raw entry and, for an indirect
// reference, against what it resolves to.
func (d *Dictionary) KeyForValue(value Base) *Name {
	for _, k := range d.keys {
		item := d.items[k]
		if cosEqual(item, value) {
			return k
		}
		if ref, ok := item.(*Object); ok && cosEqual(ref.Object(), value) {
			return k
		}
	}
	return nil
}

// KeySet returns the keys in insertion order.
func (d *Dictionary) KeySet() []*Name {
	out := make([]*Name, len(d.keys))
	copy(out, d.keys)
	return out
}

// Values returns the entries in key insertion order.
func (d *Dictionary) Values() []Base {
	out := make([]Base, 0, len(d.keys))
	for _, k := range d.keys {
		out = append(out, d.items[k])
	}
	return out
}

// All returns an iterator over the entries in insertion order, for use with
// range.
//
// Port of the forEach(BiConsumer) method and the entrySet view.
func (d *Dictionary) All(yield func(*Name, Base) bool) {
	for _, k := range d.keys {
		if !yield(k, d.items[k]) {
			return
		}
	}
}

// AddAll copies every entry of other into this dictionary.
func (d *Dictionary) AddAll(other *Dictionary) {
	if other == nil {
		return
	}
	for _, k := range other.keys {
		d.SetItem(k, other.items[k])
	}
}

// --- typed setters ---

// SetBoolean stores a boolean.
func (d *Dictionary) SetBoolean(key *Name, value bool) {
	d.SetItem(key, GetBoolean(value))
}

// SetName stores a name.
func (d *Dictionary) SetName(key *Name, value string) {
	d.SetItem(key, GetPDFName(value))
}

// SetString stores a string, removing the entry when the text is empty.
//
// Java stores null for a null argument; Go has no null string, so the empty
// string takes that role.
func (d *Dictionary) SetString(key *Name, value string) {
	if value == "" {
		d.RemoveItem(key)
		return
	}
	d.SetItem(key, NewStringObj(value))
}

// SetInt stores an integer.
func (d *Dictionary) SetInt(key *Name, value int) {
	d.SetLong(key, int64(value))
}

// SetLong stores a long integer.
func (d *Dictionary) SetLong(key *Name, value int64) {
	d.SetItem(key, GetInteger(value))
}

// SetFloat stores a real number.
func (d *Dictionary) SetFloat(key *Name, value float32) {
	d.SetItem(key, NewFloat(value))
}

// SetEmbeddedInt stores an integer in a sub-dictionary, creating it if needed.
func (d *Dictionary) SetEmbeddedInt(embedded, key *Name, value int) {
	sub := d.GetCOSDictionary(embedded)
	if sub == nil {
		sub = NewDictionary()
		d.SetItem(embedded, sub)
	}
	sub.SetInt(key, value)
}

// SetEmbeddedString stores a string in a sub-dictionary, creating it if needed.
func (d *Dictionary) SetEmbeddedString(embedded, key *Name, value string) {
	sub := d.GetCOSDictionary(embedded)
	if sub == nil {
		if value == "" {
			return
		}
		sub = NewDictionary()
		d.SetItem(embedded, sub)
	}
	sub.SetString(key, value)
}

// SetFlag sets or clears the given bits of an integer entry.
func (d *Dictionary) SetFlag(field *Name, bitFlag int, value bool) {
	current := d.GetIntDefault(field, 0)
	if value {
		current |= bitFlag
	} else {
		current &^= bitFlag
	}
	d.SetInt(field, current)
}

// --- typed getters ---

// GetCOSName returns the entry under key when it is a name, else nil.
func (d *Dictionary) GetCOSName(key *Name) *Name {
	if n, ok := d.GetDictionaryObject(key).(*Name); ok {
		return n
	}
	return nil
}

// GetCOSNameDefault returns the entry under key when it is a name, else
// defaultValue.
func (d *Dictionary) GetCOSNameDefault(key, defaultValue *Name) *Name {
	if n := d.GetCOSName(key); n != nil {
		return n
	}
	return defaultValue
}

// GetCOSObject returns the entry under key when it is an indirect reference,
// else nil. Unlike the other getters this does not resolve it.
func (d *Dictionary) GetCOSObject(key *Name) *Object {
	if o, ok := d.items[key].(*Object); ok {
		return o
	}
	return nil
}

// GetCOSDictionary returns the entry under key when it is a dictionary, else
// nil.
func (d *Dictionary) GetCOSDictionary(key *Name) *Dictionary {
	if sub, ok := d.GetDictionaryObject(key).(*Dictionary); ok {
		return sub
	}
	return nil
}

// GetCOSDictionary2 returns the entry under firstKey or secondKey when it is a
// dictionary, else nil.
func (d *Dictionary) GetCOSDictionary2(firstKey, secondKey *Name) *Dictionary {
	if sub, ok := d.GetDictionaryObject2(firstKey, secondKey).(*Dictionary); ok {
		return sub
	}
	return nil
}

// GetCOSArray returns the entry under key when it is an array, else nil.
func (d *Dictionary) GetCOSArray(key *Name) *Array {
	if a, ok := d.GetDictionaryObject(key).(*Array); ok {
		return a
	}
	return nil
}

// GetNameAsString returns the entry under key as a string when it is a name or
// a string, else defaultValue.
func (d *Dictionary) GetNameAsString(key *Name, defaultValue string) string {
	switch v := d.GetDictionaryObject(key).(type) {
	case *Name:
		return v.Name()
	case *StringObj:
		return v.Value()
	}
	return defaultValue
}

// GetString returns the entry under key when it is a string, else defaultValue.
func (d *Dictionary) GetString(key *Name, defaultValue string) string {
	if s, ok := d.GetDictionaryObject(key).(*StringObj); ok {
		return s.Value()
	}
	return defaultValue
}

// GetEmbeddedString returns a string from a sub-dictionary, or defaultValue.
func (d *Dictionary) GetEmbeddedString(embedded, key *Name, defaultValue string) string {
	if sub := d.GetCOSDictionary(embedded); sub != nil {
		return sub.GetString(key, defaultValue)
	}
	return defaultValue
}

// GetBoolean returns the entry under key when it is a boolean, else
// defaultValue.
func (d *Dictionary) GetBoolean(key *Name, defaultValue bool) bool {
	if b, ok := d.GetDictionaryObject(key).(*Boolean); ok {
		return b.Value()
	}
	return defaultValue
}

// GetBoolean2 returns the entry under firstKey or secondKey when it is a
// boolean, else defaultValue.
func (d *Dictionary) GetBoolean2(firstKey, secondKey *Name, defaultValue bool) bool {
	if b, ok := d.GetDictionaryObject2(firstKey, secondKey).(*Boolean); ok {
		return b.Value()
	}
	return defaultValue
}

// GetInt returns the entry under key as an int, or -1.
//
// Port of getInt(COSName), which defaults to -1.
func (d *Dictionary) GetInt(key *Name) int {
	return d.GetIntDefault(key, -1)
}

// GetIntDefault returns the entry under key as an int, or defaultValue.
func (d *Dictionary) GetIntDefault(key *Name, defaultValue int) int {
	if n, ok := d.GetDictionaryObject(key).(Number); ok {
		return n.IntValue()
	}
	return defaultValue
}

// GetInt2 returns the entry under firstKey or secondKey as an int, or
// defaultValue.
func (d *Dictionary) GetInt2(firstKey, secondKey *Name, defaultValue int) int {
	if n, ok := d.GetDictionaryObject2(firstKey, secondKey).(Number); ok {
		return n.IntValue()
	}
	return defaultValue
}

// GetEmbeddedInt returns an int from a sub-dictionary, or defaultValue.
func (d *Dictionary) GetEmbeddedInt(embedded, key *Name, defaultValue int) int {
	if sub := d.GetCOSDictionary(embedded); sub != nil {
		return sub.GetIntDefault(key, defaultValue)
	}
	return defaultValue
}

// GetLong returns the entry under key as an int64, or -1.
func (d *Dictionary) GetLong(key *Name) int64 {
	return d.GetLongDefault(key, -1)
}

// GetLongDefault returns the entry under key as an int64, or defaultValue.
func (d *Dictionary) GetLongDefault(key *Name, defaultValue int64) int64 {
	if n, ok := d.GetDictionaryObject(key).(Number); ok {
		return n.LongValue()
	}
	return defaultValue
}

// GetFloat returns the entry under key as a float32, or defaultValue.
func (d *Dictionary) GetFloat(key *Name, defaultValue float32) float32 {
	if n, ok := d.GetDictionaryObject(key).(Number); ok {
		return n.FloatValue()
	}
	return defaultValue
}

// GetFlag reports whether the given bits are set in an integer entry.
func (d *Dictionary) GetFlag(field *Name, bitFlag int) bool {
	return d.GetIntDefault(field, 0)&bitFlag == bitFlag
}

// ObjectFromPath walks a slash-separated path of dictionary keys and array
// indices, returning what it reaches.
//
// Port of getObjectFromPath. Java splits on "/" and reads a bracketed segment
// as an array index.
func (d *Dictionary) ObjectFromPath(objPath string) Base {
	var current Base = d
	for _, segment := range strings.Split(objPath, "/") {
		switch node := current.(type) {
		case *Array:
			idx, err := strconv.Atoi(strings.Trim(segment, "[]"))
			if err != nil || idx < 0 || idx >= node.Size() {
				return nil
			}
			current = node.GetObject(idx)
		case *Dictionary:
			current = node.GetDictionaryObject(GetPDFName(segment))
		default:
			return current
		}
	}
	return current
}

// COSObject returns the receiver.
func (d *Dictionary) COSObject() Base { return d }

// Accept dispatches to the visitor.
func (d *Dictionary) Accept(v Visitor) error { return v.VisitDictionary(d) }

// String returns the Java toString form.
func (d *Dictionary) String() string {
	var sb strings.Builder
	sb.WriteString("COSDictionary{")
	for i, k := range d.keys {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString("(")
		sb.WriteString(k.Name())
		sb.WriteString(":")
		if v := d.items[k]; v == nil {
			sb.WriteString("<null>")
		} else {
			sb.WriteString(baseString(v))
		}
		sb.WriteString(")")
	}
	sb.WriteString("}")
	return sb.String()
}
