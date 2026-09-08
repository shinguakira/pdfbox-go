package font

// JAVA-BUGS 21: `createFSIgnored` builds an FSFontInfo with a null parent, and
// `getFont` dereferences the parent for its cache.

import "testing"

// TestIgnoredFontInfoHasItsParent is the defect.
//
// The other two call sites of the constructor pass `this`. This one passes
// null, so calling `Font()` on an entry the provider decided to ignore
// dereferences it — `parent.cache.GetFont(i)` is the first line of the method.
//
// The expected value is the parameter list's: an FSFontInfo belongs to the
// provider that made it.
func TestIgnoredFontInfoHasItsParent(t *testing.T) {
	provider := &fileSystemFontProvider{cache: NewFontCache()}
	info := provider.createFSIgnored("ignored.ttf", FontFormatTTF, "Ignored")

	if info.parent != provider {
		t.Fatal("an ignored entry has no parent; Font() on it dereferences null")
	}
	// And the method that needs it does not panic.
	if got := info.Font(); got != nil {
		t.Errorf("Font() answered %v for a file that is not there, want nil", got)
	}
}
