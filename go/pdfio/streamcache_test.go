package pdfio

// Port of the stream cache factory cases of org.apache.pdfbox.io.TestIOUtils.
//
// Java asserts only that each factory answers a non-null function. A Go
// function value returned by a literal is never nil, so the assertion as
// written would prove nothing; the cases below call the function and check the
// cache it makes is the kind the factory names, which is what "non-null" is
// standing in for.

import "testing"

// TestCreateMemoryOnlyStreamCache is testCreateMemoryOnlyStreamCache.
func TestCreateMemoryOnlyStreamCache(t *testing.T) {
	create := MemoryOnlyStreamCache()
	if create == nil {
		t.Fatal("MemoryOnlyStreamCache() = nil, want a function")
	}

	cache, err := create()
	noError(t, "create", err)
	defer cache.Close()

	if _, ok := cache.(*MemoryStreamCache); !ok {
		t.Errorf("cache is %T, want *MemoryStreamCache", cache)
	}
}

// TestCreateTempFileOnlyStreamCache is testCreateTempFileOnlyStreamCache.
func TestCreateTempFileOnlyStreamCache(t *testing.T) {
	create := TempFileOnlyStreamCache()
	if create == nil {
		t.Fatal("TempFileOnlyStreamCache() = nil, want a function")
	}

	cache, err := create()
	noError(t, "create", err)
	defer cache.Close()

	scratchFile, ok := cache.(*ScratchFile)
	if !ok {
		t.Fatalf("cache is %T, want *ScratchFile", cache)
	}
	if scratchFile.inMemoryMaxPageCount != 0 {
		t.Errorf("inMemoryMaxPageCount = %d, want 0: the setting is temporary "+
			"files only", scratchFile.inMemoryMaxPageCount)
	}

	// The cache is usable, which is the point of handing one out.
	buffer, err := cache.CreateBuffer()
	noError(t, "CreateBuffer", err)
	writeAll(t, buffer, []byte("Apache PDFBox"))
	wantLength(t, buffer, 13)
	noError(t, "Close", buffer.Close())
}
