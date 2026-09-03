package cos

import "testing"

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/cos/UnmodifiableCOSDictionaryTest.java.
//
// Every test there asserts that a mutating call throws
// UnsupportedOperationException. The port makes those calls a compile error
// instead — ReadOnlyDictionary has no mutating methods — so there is nothing
// left to assert at run time. What remains testable is that the read side
// works and that the view tracks the dictionary it came from.
//
// The compile-time guarantee is checked by the var _ ReadOnlyDictionary
// assertion in unmodifiable.go together with the interface definition: if a
// mutating method were ever added to the interface, that assertion still
// compiles, but nothing can call it without the concrete type.

func TestReadOnlyDictionaryReads(t *testing.T) {
	d := NewDictionary()
	d.SetItem(Type, Page)
	d.SetInt(Count, 3)

	ro := d.AsReadOnly()

	if got := ro.Size(); got != 2 {
		t.Errorf("Size() = %d, want 2", got)
	}
	if !ro.ContainsKey(Type) {
		t.Error("ContainsKey(Type) = false")
	}
	if got := ro.GetItem(Type); got != Base(Page) {
		t.Errorf("GetItem(Type) = %v, want Page", got)
	}
	if got := ro.GetInt(Count); got != 3 {
		t.Errorf("GetInt(Count) = %d, want 3", got)
	}
	if got := ro.KeySet(); len(got) != 2 || got[0] != Type {
		t.Errorf("KeySet() = %v, want [Type Count]", got)
	}
}

// TestReadOnlyDictionarySharesStorage pins that the view is a window onto the
// original, not a copy — the same as Java wrapping the live map.
func TestReadOnlyDictionarySharesStorage(t *testing.T) {
	d := NewDictionary()
	ro := d.AsReadOnly()

	d.SetItem(Type, Page)
	if !ro.ContainsKey(Type) {
		t.Error("a write through the original is not visible in the read-only view")
	}
}
