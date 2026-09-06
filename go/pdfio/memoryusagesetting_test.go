package pdfio

// Written from org.apache.pdfbox.io.MemoryUsageSetting, which Java has no test
// for -- phase A4 of migration/tasks/track-scratchfile.md says to write one
// from the source in that case.
//
// Every expected value below is read off the private constructor, which
// normalises the four arguments before storing them, and off toString. The
// cases are the four setup methods and each branch of that normalisation.

import "testing"

func TestMemoryUsageSettingSetupMethods(t *testing.T) {
	for _, c := range []struct {
		name                       string
		setting                    *MemoryUsageSetting
		useMainMemory, useTempFile bool
		mainMemoryRestricted       bool
		storageRestricted          bool
		maxMainMemoryBytes         int64
		maxStorageBytes            int64
		text                       string
	}{
		{
			// setupMainMemoryOnly() is setupMainMemoryOnly(-1).
			name:               "MainMemoryOnly",
			setting:            SetupMainMemoryOnly(),
			useMainMemory:      true,
			maxMainMemoryBytes: -1,
			maxStorageBytes:    -1,
			text:               "Main memory only with no size restriction",
		},
		{
			// new MemoryUsageSetting(true, false, 1000, 1000): main memory is
			// used, so maxMainMemoryBytes stays; maxStorageBytes is positive so
			// it stays too; the last adjustment does not fire because
			// maxMainMemoryBytes is not greater than maxStorageBytes.
			name:                 "MainMemoryOnlyRestricted",
			setting:              SetupMainMemoryOnlyMax(1000),
			useMainMemory:        true,
			mainMemoryRestricted: true,
			storageRestricted:    true,
			maxMainMemoryBytes:   1000,
			maxStorageBytes:      1000,
			text:                 "Main memory only with max. of 1000 bytes",
		},
		{
			// new MemoryUsageSetting(true, false, 0, 0): main memory with a
			// zero maximum and no temp file, so maxMainMemoryBytes takes the
			// value of maxStorageBytes, which is -1 because 0 is not positive.
			name:               "MainMemoryOnlyZero",
			setting:            SetupMainMemoryOnlyMax(0),
			useMainMemory:      true,
			maxMainMemoryBytes: -1,
			maxStorageBytes:    -1,
			text:               "Main memory only with no size restriction",
		},
		{
			// setupTempFileOnly() is new MemoryUsageSetting(false, true, 0, -1):
			// useMainMemory is false and useTempFile true, so the first
			// adjustment leaves useMainMemory false and maxMainMemoryBytes
			// becomes -1.
			name:               "TempFileOnly",
			setting:            SetupTempFileOnly(),
			useTempFile:        true,
			maxMainMemoryBytes: -1,
			maxStorageBytes:    -1,
			text:               "Scratch file only with no size restriction",
		},
		{
			name:               "TempFileOnlyRestricted",
			setting:            SetupTempFileOnlyMax(2000),
			useTempFile:        true,
			storageRestricted:  true,
			maxMainMemoryBytes: -1,
			maxStorageBytes:    2000,
			text:               "Scratch file only with max. of 2000 bytes",
		},
		{
			// setupMixed(1000) is setupMixed(1000, -1): both flags on,
			// maxStorageBytes is not positive so it becomes -1, and the last
			// adjustment does not fire because maxStorageBytes is -1.
			name:                 "Mixed",
			setting:              SetupMixed(1000),
			useMainMemory:        true,
			useTempFile:          true,
			mainMemoryRestricted: true,
			maxMainMemoryBytes:   1000,
			maxStorageBytes:      -1,
			text:                 "Mixed mode with max. of 1000 main memory bytes and unrestricted scratch file size",
		},
		{
			// setupMixed(1000, 5000): both positive, and maxMainMemoryBytes is
			// not greater than maxStorageBytes, so both stay.
			name:                 "MixedWithStorage",
			setting:              SetupMixedMax(1000, 5000),
			useMainMemory:        true,
			useTempFile:          true,
			mainMemoryRestricted: true,
			storageRestricted:    true,
			maxMainMemoryBytes:   1000,
			maxStorageBytes:      5000,
			text:                 "Mixed mode with max. of 1000 main memory bytes and max. of 5000 storage bytes",
		},
		{
			// setupMixed(5000, 1000): maxMainMemoryBytes is greater than
			// maxStorageBytes, so maxStorageBytes takes the main memory value.
			// Java assigns locMaxStorageBytes = locMaxMainMemoryBytes, which
			// raises the storage limit rather than lowering the memory one --
			// the opposite of what the javadoc says.
			name:                 "MixedStorageBelowMainMemory",
			setting:              SetupMixedMax(5000, 1000),
			useMainMemory:        true,
			useTempFile:          true,
			mainMemoryRestricted: true,
			storageRestricted:    true,
			maxMainMemoryBytes:   5000,
			maxStorageBytes:      5000,
			text:                 "Mixed mode with max. of 5000 main memory bytes and max. of 5000 storage bytes",
		},
		{
			// setupMixed(0) is new MemoryUsageSetting(true, true, 0, -1): main
			// memory with a zero maximum and a temp file, so useMainMemory
			// becomes false. The zero maximum is left as it is, so this is not
			// quite the state setupTempFileOnly() gives -- that one leaves
			// maxMainMemoryBytes at -1 and so answers false from
			// isMainMemoryRestricted. ScratchFile reads both through
			// useMainMemory first and cannot tell them apart.
			name:                 "MixedZeroIsTempFileOnly",
			setting:              SetupMixed(0),
			useTempFile:          true,
			mainMemoryRestricted: true,
			maxMainMemoryBytes:   0,
			maxStorageBytes:      -1,
			text:                 "Scratch file only with no size restriction",
		},
		{
			// setupMixed(-1) is new MemoryUsageSetting(true, true, -1, -1) --
			// the javadoc says this is setupMainMemoryOnly(), and the flags say
			// otherwise: useTempFile stays true.
			name:               "MixedUnrestricted",
			setting:            SetupMixed(-1),
			useMainMemory:      true,
			useTempFile:        true,
			maxMainMemoryBytes: -1,
			maxStorageBytes:    -1,
			text:               "Mixed mode with max. of -1 main memory bytes and unrestricted scratch file size",
		},
		{
			// A maximum below -1 is brought back to -1.
			name:               "BelowMinusOne",
			setting:            SetupMainMemoryOnlyMax(-5),
			useMainMemory:      true,
			maxMainMemoryBytes: -1,
			maxStorageBytes:    -1,
			text:               "Main memory only with no size restriction",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.setting.UseMainMemory(); got != c.useMainMemory {
				t.Errorf("UseMainMemory() = %v, want %v", got, c.useMainMemory)
			}
			if got := c.setting.UseTempFile(); got != c.useTempFile {
				t.Errorf("UseTempFile() = %v, want %v", got, c.useTempFile)
			}
			if got := c.setting.IsMainMemoryRestricted(); got != c.mainMemoryRestricted {
				t.Errorf("IsMainMemoryRestricted() = %v, want %v", got,
					c.mainMemoryRestricted)
			}
			if got := c.setting.IsStorageRestricted(); got != c.storageRestricted {
				t.Errorf("IsStorageRestricted() = %v, want %v", got, c.storageRestricted)
			}
			if got := c.setting.MaxMainMemoryBytes(); got != c.maxMainMemoryBytes {
				t.Errorf("MaxMainMemoryBytes() = %d, want %d", got, c.maxMainMemoryBytes)
			}
			if got := c.setting.MaxStorageBytes(); got != c.maxStorageBytes {
				t.Errorf("MaxStorageBytes() = %d, want %d", got, c.maxStorageBytes)
			}
			if got := c.setting.String(); got != c.text {
				t.Errorf("String() = %q, want %q", got, c.text)
			}
		})
	}
}

// TestMemoryUsageSettingUseMainMemoryOrTempFile pins the invariant the two
// accessors document: where one is false the other is true.
func TestMemoryUsageSettingUseMainMemoryOrTempFile(t *testing.T) {
	for _, setting := range []*MemoryUsageSetting{
		SetupMainMemoryOnly(),
		SetupMainMemoryOnlyMax(0),
		SetupMainMemoryOnlyMax(1000),
		SetupTempFileOnly(),
		SetupTempFileOnlyMax(2000),
		SetupMixed(-1),
		SetupMixed(0),
		SetupMixed(1000),
		SetupMixedMax(1000, 5000),
	} {
		if !setting.UseMainMemory() && !setting.UseTempFile() {
			t.Errorf("%s uses neither main memory nor a temp file", setting)
		}
	}
}

// TestMemoryUsageSettingTempDir pins setTempDir, which returns this so that it
// chains onto a setup method.
func TestMemoryUsageSettingTempDir(t *testing.T) {
	setting := SetupTempFileOnly()
	if got := setting.TempDir(); got != "" {
		t.Errorf("TempDir() = %q on a fresh setting, want the empty string", got)
	}
	if returned := setting.SetTempDir("some/dir"); returned != setting {
		t.Error("SetTempDir returned a different setting, want the same one")
	}
	if got := setting.TempDir(); got != "some/dir" {
		t.Errorf("TempDir() = %q, want %q", got, "some/dir")
	}
}
