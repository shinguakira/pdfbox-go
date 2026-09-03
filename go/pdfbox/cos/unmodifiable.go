package cos

// ReadOnlyDictionary is the read-only view of a Dictionary.
//
// Port of org.apache.pdfbox.cos.UnmodifiableCOSDictionary, but not a literal
// one. Java wraps the backing map in Collections.unmodifiableMap and returns it
// as a COSDictionary, so a caller can pass it anywhere a dictionary goes and
// discovers the restriction only when a mutation throws
// UnsupportedOperationException at run time.
//
// Go can express the restriction in the type. AsReadOnly returns this
// interface, which simply has no mutating methods, so a write is a compile
// error rather than a panic. That is a deliberate improvement on the original:
// the Java design cannot fail earlier than run time, and this one cannot fail
// later than compile time.
//
// The consequence is that a read-only dictionary is not assignable to
// *Dictionary. Where a Java call site passes an unmodifiable dictionary into
// something typed COSDictionary, the Go port has to take this interface
// instead.
type ReadOnlyDictionary interface {
	Size() int
	ContainsKey(key *Name) bool
	ContainsValue(value Base) bool
	KeyForValue(value Base) *Name
	KeySet() []*Name
	Values() []Base
	All(yield func(*Name, Base) bool)

	GetItem(key *Name) Base
	GetItem2(firstKey, secondKey *Name) Base
	GetDictionaryObject(key *Name) Base
	GetDictionaryObject2(firstKey, secondKey *Name) Base
	ObjectFromPath(objPath string) Base

	GetCOSName(key *Name) *Name
	GetCOSNameDefault(key, defaultValue *Name) *Name
	GetCOSObject(key *Name) *Object
	GetCOSDictionary(key *Name) *Dictionary
	GetCOSDictionary2(firstKey, secondKey *Name) *Dictionary
	GetCOSArray(key *Name) *Array

	GetNameAsString(key *Name, defaultValue string) string
	GetString(key *Name, defaultValue string) string
	GetEmbeddedString(embedded, key *Name, defaultValue string) string
	GetBoolean(key *Name, defaultValue bool) bool
	GetBoolean2(firstKey, secondKey *Name, defaultValue bool) bool
	GetInt(key *Name) int
	GetIntDefault(key *Name, defaultValue int) int
	GetInt2(firstKey, secondKey *Name, defaultValue int) int
	GetEmbeddedInt(embedded, key *Name, defaultValue int) int
	GetLong(key *Name) int64
	GetLongDefault(key *Name, defaultValue int64) int64
	GetFloat(key *Name, defaultValue float32) float32
	GetFlag(field *Name, bitFlag int) bool

	String() string
}

// *Dictionary satisfies the read-only view, so AsReadOnly needs no wrapper
// type: it just narrows the static type.
var _ ReadOnlyDictionary = (*Dictionary)(nil)

// AsReadOnly returns a view of the dictionary that cannot be modified.
//
// Port of COSDictionary.asUnmodifiableDictionary. The returned value shares
// storage with the dictionary it came from, exactly as the Java wrapper does —
// it prevents writes through this reference, not writes by whoever still holds
// the original.
func (d *Dictionary) AsReadOnly() ReadOnlyDictionary {
	return d
}
