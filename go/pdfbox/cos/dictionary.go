package cos

import (
	"fmt"
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
// The date accessors are not methods here: they need DateConverter, and
// pdfbox/util imports this package for Matrix, so they read as functions over a
// dictionary in util/dictionarydate.go. Not yet ported: getCOSStream, which
// needs Stream, and the COSObjectable overloads, which need pdmodel. See
// migration/STATUS.md.
type Dictionary struct {
	object
	updateInfoState
	items map[*Name]Base
	// keys holds the insertion order of items.
	keys []*Name
}

var _ UpdateInfo = (*Dictionary)(nil)

// UpdateState returns the current UpdateState of this Dictionary.
func (d *Dictionary) UpdateState() *UpdateState { return d.state(d) }

// IsNeedToBeUpdated gets the update state for the COSWriter.
func (d *Dictionary) IsNeedToBeUpdated() bool { return d.UpdateState().IsUpdated() }

// SetNeedToBeUpdated sets the update state for the COSWriter.
func (d *Dictionary) SetNeedToBeUpdated(flag bool) { d.UpdateState().updateTo(flag) }

// ToIncrement uses this Dictionary as the base object of a new Increment.
func (d *Dictionary) ToIncrement() *Increment { return d.UpdateState().toIncrement() }

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
	d.UpdateState().update()
}

// SetItem stores value under key. A nil value removes the entry, which is what
// Java does when setItem is given null.
//
// A dictionary or array that already has a key and is not direct is stored as
// an indirect reference to it, so that the writer emits it once rather than
// inline at every use.
func (d *Dictionary) SetItem(key *Name, value Base) {
	if value == nil {
		d.RemoveItem(key)
		return
	}
	if isWrappable(value) {
		cosObject := NewObjectWithKey(value, value.Key())
		d.putItem(key, cosObject)
		d.UpdateState().updateChild(cosObject)
		return
	}
	d.putItem(key, value)
	d.UpdateState().updateChild(value)
}

// putItem is the items.put(key, value) of setItem, which keeps the key order
// alongside the map.
func (d *Dictionary) putItem(key *Name, value Base) {
	if _, exists := d.items[key]; !exists {
		d.keys = append(d.keys, key)
	}
	d.items[key] = value
}

// isWrappable reports whether Java would store the given value as a COSObject
// rather than directly, which is the condition setItem and COSArray.maybeWrap
// share.
func isWrappable(value Base) bool {
	switch value.(type) {
	case *Dictionary, *Array, *Stream:
		// COSStream is a COSDictionary in Java, so it takes the same branch.
		return !value.IsDirect() && value.Key() != nil
	}
	return false
}

// RemoveItem removes the entry under key, if any.
//
// Java calls update() whether or not the key was there, because Map.remove of
// an absent key is a no-op but the update is not conditional on it; the port
// does the same, so an incremental save writes the same objects.
func (d *Dictionary) RemoveItem(key *Name) {
	if _, exists := d.items[key]; exists {
		delete(d.items, key)
		for i, k := range d.keys {
			if k == key {
				d.keys = append(d.keys[:i], d.keys[i+1:]...)
				break
			}
		}
	}
	d.UpdateState().update()
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

// ContainsValue reports whether any entry holds value.
//
// Port of containsValue, which is deliberately not KeyForValue in disguise: it
// unwraps an indirect reference given as the argument, where KeyForValue
// unwraps the indirect references stored in the dictionary. So a dictionary
// holding a reference to x does not contain x by this test, and PDResources.add
// carries an extra search of its own because of it.
func (d *Dictionary) ContainsValue(value Base) bool {
	if d.containsRawValue(value) {
		return true
	}
	if ref, ok := value.(*Object); ok {
		return d.containsRawValue(ref.Object())
	}
	return false
}

// containsRawValue reports whether any entry equals value as it stands.
func (d *Dictionary) containsRawValue(value Base) bool {
	for _, item := range d.items {
		if cosEqual(item, value) {
			return true
		}
	}
	return false
}

// KeyForValue returns the first key whose entry holds value, or nil.
//
// Port of getKeyForValue. It matches against the raw entry and, for an indirect
// reference that resolves to something, against what it resolves to.
func (d *Dictionary) KeyForValue(value Base) *Name {
	for _, k := range d.keys {
		item := d.items[k]
		if cosEqual(item, value) {
			return k
		}
		if ref, ok := item.(*Object); ok && !ref.IsObjectNull() && cosEqual(ref.Object(), value) {
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
//
// Port of addAll(COSDictionary), which is `items.putAll(dict.items)` and
// nothing else. It deliberately does not go through setItem: the raw entries
// are copied, so a value is not turned into an indirect reference and neither
// dictionary is marked as needing an update. The copy constructor is this
// method, so `new COSDictionary(other)` copies raw too.
//
// A nil other is Java's null, which throws NullPointerException; the port
// returns, which is what slice 1 chose for every such argument.
func (d *Dictionary) AddAll(other *Dictionary) {
	if other == nil {
		return
	}
	for _, k := range other.keys {
		d.putItem(k, other.items[k])
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

// SetString stores a string.
//
// Java removes the entry for a null argument. Go has no null string, and an
// empty one is a value rather than an absence — a caller wanting Java's null
// calls RemoveItem.
func (d *Dictionary) SetString(key *Name, value string) {
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
//
// Java skips creating the sub-dictionary only for a null value, which Go has no
// way to express here; every string, empty or not, creates it.
func (d *Dictionary) SetEmbeddedString(embedded, key *Name, value string) {
	sub := d.GetCOSDictionary(embedded)
	if sub == nil {
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
//
// Port of COSDictionary.toString, which delegates to getDictionaryString with a
// list of the objects it has already been through. That list is not decoration:
// a dictionary may hold itself, directly or through an array, and without it the
// walk does not end. PDFBOX-5315 is one such document.
func (d *Dictionary) String() string {
	return dictionaryString(d, nil)
}

// dictionaryString is Java's private getDictionaryString.
//
// Java identifies a repeat with `objs.contains(base)`, which for these types is
// identity, because none of them overrides equals; the port compares the
// interface values, which for a pointer is the same test. Java then prints
// `hash:` and the identity hash code, which Go has no equivalent of, so the
// port prints the pointer instead and says as much here.
func dictionaryString(base Base, objs []Base) string {
	if base == nil {
		return "null"
	}
	for _, seen := range objs {
		if seen == base {
			// avoid endless recursion
			return fmt.Sprintf("hash:%p", base)
		}
	}

	switch value := base.(type) {
	case *Stream:
		objs = append(objs, base)
		var sb strings.Builder
		sb.WriteString(dictionaryEntriesString(&value.Dictionary, objs))
		// Java appends the hash of the raw stream data; the port appends the
		// length instead, because Arrays.hashCode of the bytes is a Java
		// specific number and nothing reads it.
		length, _ := value.Length()
		fmt.Fprintf(&sb, "COSStream{%d}", length)
		return sb.String()

	case *Dictionary:
		objs = append(objs, base)
		return dictionaryEntriesString(value, objs)

	case *Array:
		objs = append(objs, base)
		var sb strings.Builder
		sb.WriteString("COSArray{")
		for i := 0; i < value.Size(); i++ {
			sb.WriteString(dictionaryString(value.Get(i), objs))
			sb.WriteString(";")
		}
		sb.WriteString("}")
		return sb.String()

	case *Object:
		objs = append(objs, base)
		inner := value.Object()
		if inner == nil {
			inner = NullObject
		}
		return "COSObject{" + dictionaryString(inner, objs) + "}"
	}
	return baseString(base)
}

func dictionaryEntriesString(d *Dictionary, objs []Base) string {
	var sb strings.Builder
	sb.WriteString("COSDictionary{")
	for _, k := range d.keys {
		sb.WriteString(k.String())
		sb.WriteString(":")
		sb.WriteString(dictionaryString(d.items[k], objs))
		sb.WriteString(";")
	}
	sb.WriteString("}")
	return sb.String()
}

// GetCOSStream returns the entry under key as a stream, or nil where it is not
// one.
//
// Port of getCOSStream(COSName), which slice 1 left out because it names
// COSStream and the two files are in one package only by convention; slice 6
// needs it for the /Mask and /SMask of an image.
func (d *Dictionary) GetCOSStream(key *Name) *Stream {
	if stream, ok := d.GetDictionaryObject(key).(*Stream); ok {
		return stream
	}
	return nil
}

// ResetImportedObjectKeys resets all object keys to avoid overlapping numbers
// when saving the new pdf.
func (d *Dictionary) ResetImportedObjectKeys() {
	clear(d.resetObjectKeys(map[int64]bool{}))
}

// resetObjectKeys collects all indirect objects numbers within this dictionary
// and all included dictionaries. It is used to avoid overlapping object numbers
// when importing an existing page to another pdf.
//
// Expert use only. You might run into an endless recursion if choosing a wrong
// starting point.
//
// Java's collection is of COSObjectKey, which overrides equals; the port keys
// the set on the key's internal hash, as everything else that holds keys does.
// A nil map is Java's null argument, which it returns unchanged.
func (d *Dictionary) resetObjectKeys(indirectObjects map[int64]bool) map[int64]bool {
	if indirectObjects == nil {
		return indirectObjects
	}
	if key := d.Key(); key != nil {
		// avoid endless recursions
		if indirectObjects[key.InternalHash()] {
			return indirectObjects
		}
		indirectObjects[key.InternalHash()] = true
		// reset object key
		d.SetKey(nil)
	}
	for _, entryKey := range d.KeySet() {
		cosBase := d.items[entryKey]
		var indirectObjectKey *ObjectKey
		if reference, ok := cosBase.(*Object); ok {
			indirectObjectKey = reference.Key()
		}
		if indirectObjectKey != nil {
			// avoid endless recursions
			if indirectObjects[indirectObjectKey.InternalHash()] {
				continue
			}
			// dereference object first
			dereferenced := cosBase.(*Object).Object()
			// reset object key
			cosBase.SetKey(nil)
			cosBase = dereferenced
		}
		switch value := cosBase.(type) {
		case *Stream:
			// COSStream is a COSDictionary in Java, so it takes the same branch.
			// descend to included dictionary to reset all included indirect objects
			// skip PARENT and P references to avoid recursions
			if entryKey != Parent && entryKey != P {
				value.resetObjectKeys(indirectObjects)
			}
		case *Dictionary:
			// descend to included dictionary to reset all included indirect objects
			// skip PARENT and P references to avoid recursions
			if entryKey != Parent && entryKey != P {
				value.resetObjectKeys(indirectObjects)
			}
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
