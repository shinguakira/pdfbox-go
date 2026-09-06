package pdfio

// How much of a stream is held in memory before it spills to a scratch file.
//
// Port of org.apache.pdfbox.io.MemoryUsageSetting.

import "fmt"

// unrestricted is the maximum that means "no limit".
//
// Java writes -1 for it throughout and says so in every javadoc.
const unrestricted = -1

// MemoryUsageSetting says how memory and temporary files are used for
// buffering streams.
//
// Port of MemoryUsageSetting, which is final and immutable but for the
// temporary directory. Build one with a Setup function; the zero value is not
// one of the settings Java can produce.
type MemoryUsageSetting struct {
	useMainMemory bool
	useTempFile   bool

	// maxMainMemoryBytes is the most main memory to use; -1 is unrestricted.
	maxMainMemoryBytes int64

	// maxStorageBytes is the most main memory and temporary files may take
	// together; -1 is unrestricted.
	maxStorageBytes int64

	tempDir string
}

// newMemoryUsageSetting is the private constructor every setup method goes
// through, which checks and adjusts its arguments so that the setting is
// consistent.
func newMemoryUsageSetting(useMainMemory, useTempFile bool,
	maxMainMemoryBytes, maxStorageBytes int64) *MemoryUsageSetting {
	// do some checks; adjust values as needed to get consistent setting
	locUseMainMemory := !useTempFile || useMainMemory
	locMaxMainMemoryBytes := int64(unrestricted)
	if useMainMemory {
		locMaxMainMemoryBytes = maxMainMemoryBytes
	}
	locMaxStorageBytes := int64(unrestricted)
	if maxStorageBytes > 0 {
		locMaxStorageBytes = maxStorageBytes
	}

	if locMaxMainMemoryBytes < unrestricted {
		locMaxMainMemoryBytes = unrestricted
	}

	if locUseMainMemory && locMaxMainMemoryBytes == 0 {
		if useTempFile {
			locUseMainMemory = false
		} else {
			locMaxMainMemoryBytes = locMaxStorageBytes
		}
	}

	if locUseMainMemory && locMaxStorageBytes > unrestricted &&
		(locMaxMainMemoryBytes == unrestricted ||
			locMaxMainMemoryBytes > locMaxStorageBytes) {
		locMaxStorageBytes = locMaxMainMemoryBytes
	}

	return &MemoryUsageSetting{
		useMainMemory:      locUseMainMemory,
		useTempFile:        useTempFile,
		maxMainMemoryBytes: locMaxMainMemoryBytes,
		maxStorageBytes:    locMaxStorageBytes,
	}
}

// SetupMainMemoryOnly returns a setting that uses only main memory, with no
// restriction on size.
//
// Port of setupMainMemoryOnly().
func SetupMainMemoryOnly() *MemoryUsageSetting {
	return SetupMainMemoryOnlyMax(unrestricted)
}

// SetupMainMemoryOnlyMax returns a setting that uses only main memory, with the
// given maximum; -1 is no restriction, and 0 is read as no restriction too.
//
// Port of setupMainMemoryOnly(long).
func SetupMainMemoryOnlyMax(maxMainMemoryBytes int64) *MemoryUsageSetting {
	return newMemoryUsageSetting(true, false, maxMainMemoryBytes, maxMainMemoryBytes)
}

// SetupTempFileOnly returns a setting that uses only temporary files, with no
// restriction on size.
//
// Port of setupTempFileOnly().
func SetupTempFileOnly() *MemoryUsageSetting {
	return SetupTempFileOnlyMax(unrestricted)
}

// SetupTempFileOnlyMax returns a setting that uses only temporary files, with
// the given maximum size for them together; -1 is no restriction, and 0 is read
// as no restriction too.
//
// Port of setupTempFileOnly(long).
func SetupTempFileOnlyMax(maxStorageBytes int64) *MemoryUsageSetting {
	return newMemoryUsageSetting(false, true, 0, maxStorageBytes)
}

// SetupMixed returns a setting that uses a portion of main memory and spills to
// temporary files once it is exceeded.
//
// Port of setupMixed(long). Java's javadoc says -1 is the same as
// setupMainMemoryOnly and 0 the same as setupTempFileOnly; only the second
// holds -- see migration/JAVA-BUGS.md.
func SetupMixed(maxMainMemoryBytes int64) *MemoryUsageSetting {
	return SetupMixedMax(maxMainMemoryBytes, unrestricted)
}

// SetupMixedMax returns a setting that uses a portion of main memory and spills
// to temporary files once it is exceeded, with a maximum for the two together.
//
// Port of setupMixed(long, long).
func SetupMixedMax(maxMainMemoryBytes, maxStorageBytes int64) *MemoryUsageSetting {
	return newMemoryUsageSetting(true, true, maxMainMemoryBytes, maxStorageBytes)
}

// SetTempDir sets the directory temporary files are made in, and returns this
// setting so that it chains onto a setup function.
//
// Port of setTempDir(File).
func (m *MemoryUsageSetting) SetTempDir(tempDir string) *MemoryUsageSetting {
	m.tempDir = tempDir
	return m
}

// UseMainMemory reports whether main memory is to be used.
//
// Where this is false, UseTempFile is true.
func (m *MemoryUsageSetting) UseMainMemory() bool { return m.useMainMemory }

// UseTempFile reports whether a temporary file is to be used.
//
// Where this is false, UseMainMemory is true.
func (m *MemoryUsageSetting) UseTempFile() bool { return m.useTempFile }

// IsMainMemoryRestricted reports whether main memory is restricted to a
// specific number of bytes.
func (m *MemoryUsageSetting) IsMainMemoryRestricted() bool {
	return m.maxMainMemoryBytes >= 0
}

// IsStorageRestricted reports whether the storage size is restricted to a
// specific number of bytes.
func (m *MemoryUsageSetting) IsStorageRestricted() bool { return m.maxStorageBytes > 0 }

// MaxMainMemoryBytes returns the most main memory to be used, -1 being
// unrestricted.
func (m *MemoryUsageSetting) MaxMainMemoryBytes() int64 { return m.maxMainMemoryBytes }

// MaxStorageBytes returns the most main memory and temporary files may take
// together, -1 being unrestricted.
func (m *MemoryUsageSetting) MaxStorageBytes() int64 { return m.maxStorageBytes }

// TempDir returns the directory temporary files are made in, and the empty
// string where none was set -- which is Java's null.
func (m *MemoryUsageSetting) TempDir() string { return m.tempDir }

// StreamCache returns a function that makes a scratch file with this setting.
//
// Port of the public final streamCache field, which Java declares as a lambda
// over the same setting.
func (m *MemoryUsageSetting) StreamCache() StreamCacheFunc {
	return func() (StreamCache, error) { return NewScratchFile(m) }
}

// String returns the Java toString form.
func (m *MemoryUsageSetting) String() string {
	if !m.useMainMemory {
		if m.IsStorageRestricted() {
			return fmt.Sprintf("Scratch file only with max. of %d bytes", m.maxStorageBytes)
		}
		return "Scratch file only with no size restriction"
	}
	if !m.useTempFile {
		if m.IsMainMemoryRestricted() {
			return fmt.Sprintf("Main memory only with max. of %d bytes",
				m.maxMainMemoryBytes)
		}
		return "Main memory only with no size restriction"
	}
	if m.IsStorageRestricted() {
		return fmt.Sprintf(
			"Mixed mode with max. of %d main memory bytes and max. of %d storage bytes",
			m.maxMainMemoryBytes, m.maxStorageBytes)
	}
	return fmt.Sprintf(
		"Mixed mode with max. of %d main memory bytes and unrestricted scratch file size",
		m.maxMainMemoryBytes)
}
