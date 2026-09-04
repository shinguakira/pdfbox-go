// Package encoding maps the character codes of a PDF font onto glyph names.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.
package encoding

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Encoding is a mapping from character codes to glyph names.
//
// Port of the abstract org.apache.pdfbox.pdmodel.font.encoding.Encoding. Java
// has the concrete encodings extend it; the port keeps the shared state in
// encodingBase, which each of them embeds, and puts what varies behind this
// interface.
type Encoding interface {
	// COSObject returns the COS object that stands for this encoding.
	COSObject() cos.Base

	// EncodingName returns the name of this encoding.
	EncodingName() string

	// CodeToNameMap returns the code to name mapping.
	CodeToNameMap() map[int]string

	// NameToCodeMap returns the name to code mapping.
	NameToCodeMap() map[string]int

	// ContainsName reports whether the encoding has the given glyph name.
	ContainsName(name string) bool

	// ContainsCode reports whether the encoding maps the given character code.
	ContainsCode(code int) bool

	// Name returns the glyph name the given character code maps to.
	Name(code int) string

	// base returns the shared part, which is how one encoding is built from
	// another. Java reaches the protected fields of the base encoding directly.
	base() *encodingBase
}

// GetInstance returns the encoding the given name stands for, or nil where the
// name is not one of the four the specification predefines.
func GetInstance(name *cos.Name) Encoding {
	switch {
	case cos.StandardEncoding.Equals(name):
		return StandardEncodingInstance
	case cos.WinAnsiEncoding.Equals(name):
		return WinAnsiEncodingInstance
	case cos.MacRomanEncoding.Equals(name):
		return MacRomanEncodingInstance
	case cos.MacExpertEncoding.Equals(name):
		return MacExpertEncodingInstance
	default:
		return nil
	}
}

// encodingBase is the state every encoding carries.
type encodingBase struct {
	codeToName map[int]string
	inverted   map[string]int
}

// newEncodingBase returns the two empty maps every encoding starts with.
func newEncodingBase() encodingBase {
	return encodingBase{
		codeToName: make(map[int]string, 250),
		inverted:   make(map[string]int, 250),
	}
}

// base returns the shared part of the encoding.
func (e *encodingBase) base() *encodingBase { return e }

// CodeToNameMap returns a copy of the code to name mapping. Java returns an
// unmodifiable view.
func (e *encodingBase) CodeToNameMap() map[int]string {
	out := make(map[int]string, len(e.codeToName))
	for code, name := range e.codeToName {
		out[code] = name
	}
	return out
}

// NameToCodeMap returns a copy of the name to code mapping. Java returns an
// unmodifiable view.
func (e *encodingBase) NameToCodeMap() map[string]int {
	out := make(map[string]int, len(e.inverted))
	for name, code := range e.inverted {
		out[name] = code
	}
	return out
}

// add records a character code and the glyph name it maps to. The reverse
// mapping keeps the first code a name was seen at.
func (e *encodingBase) add(code int, name string) {
	e.codeToName[code] = name
	if _, ok := e.inverted[name]; !ok {
		e.inverted[name] = code
	}
}

// overwrite records a character code and the glyph name it maps to, replacing
// whatever the code mapped to before.
func (e *encodingBase) overwrite(code int, name string) {
	// remove existing reverse mapping first
	if oldName, ok := e.codeToName[code]; ok {
		if oldCode, ok := e.inverted[oldName]; ok && oldCode == code {
			delete(e.inverted, oldName)
		}
	}
	e.inverted[name] = code
	e.codeToName[code] = name
}

// ContainsName reports whether the encoding has the given glyph name.
func (e *encodingBase) ContainsName(name string) bool {
	_, ok := e.inverted[name]
	return ok
}

// ContainsCode reports whether the encoding maps the given character code.
func (e *encodingBase) ContainsCode(code int) bool {
	_, ok := e.codeToName[code]
	return ok
}

// Name returns the glyph name the given character code maps to, or ".notdef"
// where the encoding does not map it.
func (e *encodingBase) Name(code int) string {
	if name, ok := e.codeToName[code]; ok {
		return name
	}
	return ".notdef"
}

// fromTable returns the state of an encoding built from a static table.
func fromTable(table []encodingEntry) encodingBase {
	e := newEncodingBase()
	for _, entry := range table {
		e.add(entry.code, entry.name)
	}
	return e
}
