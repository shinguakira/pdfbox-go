// Package resources holds the data files fontbox ships with.
//
// Java loads these off the classpath with getResourceAsStream, relative to the
// class asking for them -- CMapParser asks for "Identity-H" and gets
// /org/apache/fontbox/cmap/Identity-H. Go has no classpath, so they are
// embedded into the binary and opened by the same tail of the path.
//
// The files themselves are copies of the ones under
// fontbox/src/main/resources/org/apache/fontbox, byte for byte.
package resources

import (
	"embed"
	"io/fs"
)

//go:embed cmap
//go:embed unicode/Scripts.txt
var files embed.FS

// Open returns the named resource, the name being the path below
// /org/apache/fontbox -- "cmap/Identity-H", say.
func Open(name string) (fs.File, error) {
	return files.Open(name)
}

// Read returns the contents of the named resource.
func Read(name string) ([]byte, error) {
	return files.ReadFile(name)
}
