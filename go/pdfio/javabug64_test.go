package pdfio

// JAVA-BUGS 64: `MemoryUsageSetting.setupMixed`'s javadoc says -1 is the same
// as `setupMainMemoryOnly()` and 0 the same as `setupTempFileOnly()`. Neither
// holds. The entry is kept -- the arithmetic is the Java's and only the
// javadoc was wrong, and the port's comment already says so -- and these
// tests hold the four settings apart, so the record is checkable rather than
// asserted.

import "testing"

// TestMixedIsNotMainMemoryOnly and its sibling below are read off the running
// Java, `io` compiled against JDK 17:
//
//	setupMixed(-1)         mainMem=true  tempFile=true  memRestricted=false maxMem=-1 maxStore=-1
//	setupMainMemoryOnly()  mainMem=true  tempFile=false memRestricted=false maxMem=-1 maxStore=-1
//	setupMixed(0)          mainMem=false tempFile=true  memRestricted=true  maxMem=0  maxStore=-1
//	setupTempFileOnly()    mainMem=false tempFile=true  memRestricted=false maxMem=-1 maxStore=-1
func TestMixedIsNotMainMemoryOnly(t *testing.T) {
	mixed := SetupMixed(-1)
	mainMemory := SetupMainMemoryOnly()

	if !mixed.UseTempFile() {
		t.Error("SetupMixed(-1).UseTempFile() = false, want true")
	}
	if mainMemory.UseTempFile() {
		t.Error("SetupMainMemoryOnly().UseTempFile() = true, want false")
	}
	// the difference the javadoc denies
	if mixed.UseTempFile() == mainMemory.UseTempFile() {
		t.Error("the two settings agree on UseTempFile, which the javadoc claims " +
			"and the constructor does not do")
	}
	// and everything else about them is the same, which is why the claim looks
	// plausible
	if mixed.UseMainMemory() != mainMemory.UseMainMemory() ||
		mixed.MaxMainMemoryBytes() != mainMemory.MaxMainMemoryBytes() ||
		mixed.MaxStorageBytes() != mainMemory.MaxStorageBytes() {
		t.Errorf("the two settings differ elsewhere: %v against %v", mixed, mainMemory)
	}
}

// TestMixedOfZeroIsNotTempFileOnly is the second claim.
func TestMixedOfZeroIsNotTempFileOnly(t *testing.T) {
	mixed := SetupMixed(0)
	tempFile := SetupTempFileOnly()

	if got := mixed.MaxMainMemoryBytes(); got != 0 {
		t.Errorf("SetupMixed(0).MaxMainMemoryBytes() = %d, want 0", got)
	}
	if !mixed.IsMainMemoryRestricted() {
		t.Error("SetupMixed(0).IsMainMemoryRestricted() = false, want true")
	}
	if got := tempFile.MaxMainMemoryBytes(); got != -1 {
		t.Errorf("SetupTempFileOnly().MaxMainMemoryBytes() = %d, want -1", got)
	}
	if tempFile.IsMainMemoryRestricted() {
		t.Error("SetupTempFileOnly().IsMainMemoryRestricted() = true, want false")
	}
	// both agree that main memory is off and a temporary file is on, which is
	// as far as the claim goes
	if mixed.UseMainMemory() || tempFile.UseMainMemory() {
		t.Error("one of the two settings uses main memory")
	}
	if !mixed.UseTempFile() || !tempFile.UseTempFile() {
		t.Error("one of the two settings does not use a temporary file")
	}
}
