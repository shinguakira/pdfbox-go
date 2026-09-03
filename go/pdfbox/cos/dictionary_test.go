package cos

import "testing"

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/COSDictionaryTest.java,
// plus tests written from COSDictionary.java for the behaviour that file does
// not reach — it has only one test method.
//
// testCOSDictionaryNotEqualsCOSStream needs COSStream, which is not ported; the
// identity semantics it guards are covered by TestDictionaryUsesIdentity below.

func TestDictionaryBaseContract(t *testing.T) {
	assertBaseContract(t, NewDictionary())
}

func TestDictionaryAccept(t *testing.T) {
	assertVisits(t, NewDictionary(), "dictionary")
}

func TestDictionarySetGetItem(t *testing.T) {
	d := NewDictionary()
	if got := d.Size(); got != 0 {
		t.Errorf("Size() = %d, want 0", got)
	}

	d.SetItem(Type, Page)
	if got := d.Size(); got != 1 {
		t.Errorf("Size() = %d, want 1", got)
	}
	if got := d.GetItem(Type); got != Base(Page) {
		t.Errorf("GetItem(Type) = %v, want Page", got)
	}
	if !d.ContainsKey(Type) {
		t.Error("ContainsKey(Type) = false")
	}
	if d.ContainsKey(Kids) {
		t.Error("ContainsKey(Kids) = true for an absent key")
	}
}

// TestDictionarySetItemNilRemoves pins that storing a nil value removes the
// entry, which is what Java does when setItem is given null.
func TestDictionarySetItemNilRemoves(t *testing.T) {
	d := NewDictionary()
	d.SetItem(Type, Page)
	d.SetItem(Type, nil)

	if d.ContainsKey(Type) {
		t.Error("ContainsKey(Type) = true after setting nil")
	}
	if got := d.Size(); got != 0 {
		t.Errorf("Size() = %d, want 0", got)
	}
}

// TestDictionaryPreservesInsertionOrder pins the LinkedHashMap behaviour the
// Java relies on: keys come back in the order they were first set, which is the
// order a dictionary is written back in.
func TestDictionaryPreservesInsertionOrder(t *testing.T) {
	d := NewDictionary()
	want := []*Name{Type, Kids, Count, Parent, Resources}
	for _, k := range want {
		d.SetItem(k, GetInteger(1))
	}

	got := d.KeySet()
	if len(got) != len(want) {
		t.Fatalf("KeySet() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("KeySet()[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// re-setting an existing key must not move it
	d.SetItem(Type, GetInteger(2))
	got = d.KeySet()
	if got[0] != Type {
		t.Errorf("KeySet()[0] = %v after re-setting Type, want Type", got[0])
	}
	if len(got) != len(want) {
		t.Errorf("KeySet() length = %d after re-setting, want %d", len(got), len(want))
	}

	// removing and re-adding moves the key to the end
	d.RemoveItem(Kids)
	d.SetItem(Kids, GetInteger(3))
	got = d.KeySet()
	if got[len(got)-1] != Kids {
		t.Errorf("KeySet() last = %v, want Kids after remove and re-add", got[len(got)-1])
	}
}

func TestDictionaryRemoveItem(t *testing.T) {
	d := NewDictionary()
	d.SetItem(Type, Page)
	d.RemoveItem(Type)

	if d.ContainsKey(Type) {
		t.Error("ContainsKey(Type) = true after RemoveItem")
	}
	if got := d.GetItem(Type); got != nil {
		t.Errorf("GetItem(Type) = %v after RemoveItem, want nil", got)
	}
	// removing an absent key is not an error
	d.RemoveItem(Kids)
}

func TestDictionaryClear(t *testing.T) {
	d := NewDictionary()
	d.SetItem(Type, Page)
	d.SetItem(Kids, NewArray())
	d.Clear()

	if got := d.Size(); got != 0 {
		t.Errorf("Size() = %d after Clear, want 0", got)
	}
	if got := len(d.KeySet()); got != 0 {
		t.Errorf("KeySet() length = %d after Clear, want 0", got)
	}
}

// TestDictionaryGetDictionaryObject pins the difference between GetItem and
// GetDictionaryObject: the latter resolves indirect references and maps the
// null object to nil.
func TestDictionaryGetDictionaryObject(t *testing.T) {
	inner := GetPDFName("Resolved")
	d := NewDictionary()
	d.SetItem(Type, NewObject(inner))
	d.SetItem(Kids, NewObject(NullObject))

	if _, ok := d.GetItem(Type).(*Object); !ok {
		t.Errorf("GetItem(Type) = %T, want the raw *Object", d.GetItem(Type))
	}
	if got := d.GetDictionaryObject(Type); got != Base(inner) {
		t.Errorf("GetDictionaryObject(Type) = %v, want the resolved name", got)
	}
	if got := d.GetDictionaryObject(Kids); got != nil {
		t.Errorf("GetDictionaryObject(Kids) = %v, want nil for the null object", got)
	}
	if got := d.GetDictionaryObject(Count); got != nil {
		t.Errorf("GetDictionaryObject of an absent key = %v, want nil", got)
	}
}

func TestDictionaryTypedAccessors(t *testing.T) {
	d := NewDictionary()

	d.SetName(Type, "Page")
	if got := d.GetNameAsString(Type, ""); got != "Page" {
		t.Errorf("GetNameAsString(Type) = %q, want Page", got)
	}
	if got := d.GetCOSName(Type); got != Page {
		t.Errorf("GetCOSName(Type) = %v, want Page", got)
	}

	d.SetInt(Count, 42)
	if got := d.GetInt(Count); got != 42 {
		t.Errorf("GetInt(Count) = %d, want 42", got)
	}
	if got := d.GetLong(Count); got != 42 {
		t.Errorf("GetLong(Count) = %d, want 42", got)
	}

	d.SetFloat(Width, 1.5)
	if got := d.GetFloat(Width, 0); got != 1.5 {
		t.Errorf("GetFloat(Width) = %v, want 1.5", got)
	}

	d.SetBoolean(Open, true)
	if got := d.GetBoolean(Open, false); !got {
		t.Error("GetBoolean(Open) = false, want true")
	}

	d.SetString(Title, "hello")
	if got := d.GetString(Title, ""); got != "hello" {
		t.Errorf("GetString(Title) = %q, want hello", got)
	}

	array := NewArray()
	d.SetItem(Kids, array)
	if got := d.GetCOSArray(Kids); got != array {
		t.Errorf("GetCOSArray(Kids) = %v, want the array", got)
	}

	sub := NewDictionary()
	d.SetItem(Resources, sub)
	if got := d.GetCOSDictionary(Resources); got != sub {
		t.Errorf("GetCOSDictionary(Resources) = %v, want the dictionary", got)
	}
}

func TestDictionaryAccessorDefaults(t *testing.T) {
	d := NewDictionary()

	// absent keys fall back
	if got := d.GetInt(Count); got != -1 {
		t.Errorf("GetInt of an absent key = %d, want -1", got)
	}
	if got := d.GetIntDefault(Count, 7); got != 7 {
		t.Errorf("GetIntDefault = %d, want 7", got)
	}
	if got := d.GetNameAsString(Type, "fallback"); got != "fallback" {
		t.Errorf("GetNameAsString = %q, want fallback", got)
	}
	if got := d.GetBoolean(Open, true); !got {
		t.Error("GetBoolean of an absent key did not return the default")
	}
	if got := d.GetCOSArray(Kids); got != nil {
		t.Errorf("GetCOSArray of an absent key = %v, want nil", got)
	}

	// a key holding the wrong type also falls back
	d.SetName(Count, "NotANumber")
	if got := d.GetIntDefault(Count, 7); got != 7 {
		t.Errorf("GetIntDefault over a name = %d, want 7", got)
	}
	if got := d.GetCOSArray(Count); got != nil {
		t.Errorf("GetCOSArray over a name = %v, want nil", got)
	}
}

// TestDictionaryTwoKeyFallback covers the getDictionaryObject(first, second)
// form, which tries an abbreviated key when the full one is absent.
func TestDictionaryTwoKeyFallback(t *testing.T) {
	d := NewDictionary()
	d.SetInt(Filter, 3)

	if got := d.GetDictionaryObject2(FL, Filter); got != Base(GetInteger(3)) {
		t.Errorf("GetDictionaryObject2 = %v, want 3 via the second key", got)
	}

	d.SetInt(FL, 9)
	if got := d.GetDictionaryObject2(FL, Filter); got != Base(GetInteger(9)) {
		t.Errorf("GetDictionaryObject2 = %v, want 9 via the first key", got)
	}
}

func TestDictionaryFlags(t *testing.T) {
	d := NewDictionary()
	d.SetFlag(Ff, 1<<2, true)
	if !d.GetFlag(Ff, 1<<2) {
		t.Error("GetFlag = false after setting the bit")
	}
	if d.GetFlag(Ff, 1<<3) {
		t.Error("GetFlag = true for a bit that was not set")
	}
	d.SetFlag(Ff, 1<<2, false)
	if d.GetFlag(Ff, 1<<2) {
		t.Error("GetFlag = true after clearing the bit")
	}
}

func TestDictionaryAddAll(t *testing.T) {
	src := NewDictionary()
	src.SetItem(Type, Page)
	src.SetItem(Count, GetInteger(1))

	dst := NewDictionary()
	dst.SetItem(Kids, NewArray())
	dst.AddAll(src)

	if got := dst.Size(); got != 3 {
		t.Errorf("Size() = %d, want 3", got)
	}
	if got := dst.GetItem(Type); got != Base(Page) {
		t.Errorf("GetItem(Type) = %v, want Page", got)
	}
}

func TestDictionaryCopy(t *testing.T) {
	src := NewDictionary()
	src.SetItem(Type, Page)

	cp := NewDictionaryFrom(src)
	if got := cp.GetItem(Type); got != Base(Page) {
		t.Errorf("GetItem(Type) = %v, want Page", got)
	}

	// the copy is independent
	cp.SetItem(Count, GetInteger(1))
	if src.ContainsKey(Count) {
		t.Error("mutating the copy changed the original")
	}
}

// TestDictionaryUsesIdentity covers what COSDictionaryTest guards: dictionaries
// compare by identity, not by content. Java gets this by not overriding equals,
// which is what keeps a COSStream from comparing equal to a COSDictionary
// holding the same entries.
func TestDictionaryUsesIdentity(t *testing.T) {
	a := NewDictionary()
	a.SetItem(BE, BE)
	b := NewDictionary()
	b.SetItem(BE, BE)

	if a == b {
		t.Error("two dictionaries with the same entries compare equal; they must compare by identity")
	}
	if a != a {
		t.Error("a dictionary does not compare equal to itself")
	}
}

func TestDictionaryKeyForValue(t *testing.T) {
	d := NewDictionary()
	d.SetItem(Type, Page)

	if got := d.KeyForValue(Page); got != Type {
		t.Errorf("KeyForValue(Page) = %v, want Type", got)
	}
	if got := d.KeyForValue(Kids); got != nil {
		t.Errorf("KeyForValue of an absent value = %v, want nil", got)
	}
	if !d.ContainsValue(Page) {
		t.Error("ContainsValue(Page) = false")
	}
}
