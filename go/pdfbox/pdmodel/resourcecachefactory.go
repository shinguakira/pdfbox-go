package pdmodel

import "sync"

// ResourceCacheCreateFunction makes a resource cache.
//
// Port of the functional interface
// org.apache.pdfbox.pdmodel.ResourceCacheCreateFunction, which Go writes as a
// function type.
type ResourceCacheCreateFunction func() ResourceCache

// NewDefaultResourceCacheCreateFunction returns the function that makes a
// DefaultResourceCache with the stable object cache enabled.
//
// Port of DefaultResourceCacheCreateImpl, whose no-argument constructor passes
// true. Java needs a class because the interface is nominal; Go needs only the
// function.
func NewDefaultResourceCacheCreateFunction() ResourceCacheCreateFunction {
	return NewDefaultResourceCacheCreateFunctionStable(true)
}

// NewDefaultResourceCacheCreateFunctionStable returns the function that makes a
// DefaultResourceCache, with the stable object cache enabled or not.
//
// Port of DefaultResourceCacheCreateImpl(boolean).
func NewDefaultResourceCacheCreateFunctionStable(enableStableCache bool) ResourceCacheCreateFunction {
	return func() ResourceCache {
		return NewDefaultResourceCacheStable(enableStableCache)
	}
}

// resourceCacheFactory holds the function every new document's cache is made
// with.
//
// Java is a class of statics with a static initialiser that installs the
// default. Go has no static initialiser for a class, so the value is a package
// variable set here, and the mutex is because Java's field is written by
// setResourceCacheCreateFunction from whatever thread calls it while documents
// are being opened on others -- Java's is unsynchronised and racy, which is a
// hazard the port does not need to reproduce to be faithful about what the
// function does.
var resourceCacheFactory = struct {
	mu     sync.RWMutex
	create ResourceCacheCreateFunction
}{create: NewDefaultResourceCacheCreateFunction()}

// SetResourceCacheCreateFunction makes every later document use the given
// function for its resource cache. Caching is disabled where it is nil.
//
// Port of the static ResourceCacheFactory.setResourceCacheCreateFunction.
func SetResourceCacheCreateFunction(function ResourceCacheCreateFunction) {
	resourceCacheFactory.mu.Lock()
	defer resourceCacheFactory.mu.Unlock()
	resourceCacheFactory.create = function
}

// GetResourceCacheCreateFunction returns the function in use, which is nil
// where caching has been disabled.
//
// Port of the static ResourceCacheFactory.getResourceCacheCreateFunction.
func GetResourceCacheCreateFunction() ResourceCacheCreateFunction {
	resourceCacheFactory.mu.RLock()
	defer resourceCacheFactory.mu.RUnlock()
	return resourceCacheFactory.create
}

// CreateResourceCache makes a resource cache with the function in use, and
// answers nil where that function is nil.
//
// Port of the static ResourceCacheFactory.createResourceCache. A nil cache is
// how caching is turned off, and every reader of PDDocument.ResourceCache
// already checks for one.
func CreateResourceCache() ResourceCache {
	create := GetResourceCacheCreateFunction()
	if create == nil {
		return nil
	}
	return create()
}
