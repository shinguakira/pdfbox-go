package pdfio

// JAVA-BUGS 62: `ScratchFile.markPagesAsFree` walks `for (aIdx = off; aIdx <
// count; aIdx++)`, so it visits `count - off` entries rather than `count`, and
// `ScratchFileBuffer.clear` -- which passes an offset of 1 -- never frees the
// buffer's last page.

import "testing"

// TestClearFreesEveryPage is the defect.
//
// The expected behaviour is that a cleared buffer can be refilled: the pages
// are a fixed allowance, `Clear` hands them all back, and nothing about
// clearing says one is kept. Java's loop stops `off` entries early, so every
// clear leaks a page; measured against JDK 17 before the fix, a scratch file
// of three pages with three pages written and then cleared fails its second
// filling with "Maximum allowed scratch file memory exceeded."
func TestClearFreesEveryPage(t *testing.T) {
	// three pages of main memory and no more
	scratchFile, err := NewScratchFile(SetupMainMemoryOnlyMax(3 * scratchPageSize))
	noError(t, "NewScratchFile", err)
	defer scratchFile.Close()

	buffer := createBuffer(t, scratchFile)
	page := make([]byte, scratchPageSize)
	for i := 0; i < 3; i++ {
		writeAll(t, buffer, page)
	}

	noError(t, "Clear", buffer.Clear())

	// all three pages are back, so the buffer fills again
	for i := 0; i < 3; i++ {
		if _, err := buffer.Write(page); err != nil {
			t.Fatalf("writing page %d after Clear: %v -- Clear did not free every page",
				i+1, err)
		}
	}
	length, err := buffer.Length()
	noError(t, "Length", err)
	if length != 3*scratchPageSize {
		t.Errorf("Length() = %d, want %d", length, 3*scratchPageSize)
	}
}
