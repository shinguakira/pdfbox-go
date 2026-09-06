package pdmodel_test

// Written from org.apache.pdfbox.pdmodel.ResourceCacheFactory,
// ResourceCacheCreateFunction and DefaultResourceCacheCreateImpl, which Java
// has no test for.
//
// The three classes were unported until track/test-backfill, which found them
// while backfilling the cases around them. They are the process-wide override
// point PDDocument reads its cache from, and without them a document could
// neither be given a different cache nor be told to keep none.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// restoreFactory puts the default back, so a case that changes it cannot leak
// into the next.
func restoreFactory(t *testing.T) {
	t.Helper()
	previous := pdmodel.GetResourceCacheCreateFunction()
	t.Cleanup(func() { pdmodel.SetResourceCacheCreateFunction(previous) })
}

// TestCreateResourceCacheDefault checks the static initialiser's effect: with
// nothing set, every document gets a DefaultResourceCache.
func TestCreateResourceCacheDefault(t *testing.T) {
	if pdmodel.GetResourceCacheCreateFunction() == nil {
		t.Fatal("GetResourceCacheCreateFunction() = nil before anything set it, " +
			"want the default")
	}
	cache := pdmodel.CreateResourceCache()
	if cache == nil {
		t.Fatal("CreateResourceCache() = nil, want a default cache")
	}
	if _, ok := cache.(*pdmodel.DefaultResourceCache); !ok {
		t.Errorf("CreateResourceCache() is %T, want *DefaultResourceCache", cache)
	}

	document := pdmodel.NewPDDocument()
	defer document.Close()
	if document.ResourceCache() == nil {
		t.Error("a new document has no resource cache, want the default")
	}
}

// TestSetResourceCacheCreateFunctionToNilDisablesCaching is the behaviour
// ResourceCacheFactory's own javadoc names: "Setting the function to null
// disables the cache."
func TestSetResourceCacheCreateFunctionToNilDisablesCaching(t *testing.T) {
	restoreFactory(t)

	pdmodel.SetResourceCacheCreateFunction(nil)
	if pdmodel.CreateResourceCache() != nil {
		t.Error("CreateResourceCache() returned a cache after the function was " +
			"set to nil, want none")
	}

	document := pdmodel.NewPDDocument()
	defer document.Close()
	if document.ResourceCache() != nil {
		t.Error("a new document has a resource cache with caching disabled, " +
			"want none")
	}
}

// TestSetResourceCacheCreateFunctionIsUsed checks a caller's own function is
// the one every later document goes through.
func TestSetResourceCacheCreateFunctionIsUsed(t *testing.T) {
	restoreFactory(t)

	calls := 0
	pdmodel.SetResourceCacheCreateFunction(func() pdmodel.ResourceCache {
		calls++
		return pdmodel.NewDefaultResourceCacheStable(false)
	})

	if pdmodel.CreateResourceCache() == nil {
		t.Fatal("CreateResourceCache() = nil, want the cache the function makes")
	}
	if calls != 1 {
		t.Errorf("the function was called %d times, want 1", calls)
	}

	document := pdmodel.NewPDDocument()
	defer document.Close()
	if document.ResourceCache() == nil {
		t.Error("a new document has no resource cache, want the one the " +
			"function makes")
	}
	if calls != 2 {
		t.Errorf("the function was called %d times after opening a document, "+
			"want 2", calls)
	}
}
