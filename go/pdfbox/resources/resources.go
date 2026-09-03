// Package resources holds the data files the library ships with.
//
// Java loads these off the classpath with getResourceAsStream, from
// /org/apache/pdfbox/resources. Go has no classpath, so they are embedded into
// the binary and opened by the same tail of the path.
//
// The files themselves are copies of the ones under
// pdfbox/src/main/resources/org/apache/pdfbox/resources, byte for byte.
package resources

import (
	"embed"
	"io/fs"
)

//go:embed glyphlist/glyphlist.txt glyphlist/zapfdingbats.txt glyphlist/additional.txt
var files embed.FS

// Open returns the named resource, the name being the path below
// /org/apache/pdfbox/resources -- "glyphlist/glyphlist.txt", say.
func Open(name string) (fs.File, error) {
	return files.Open(name)
}

// Read returns the contents of the named resource.
func Read(name string) ([]byte, error) {
	return files.ReadFile(name)
}
