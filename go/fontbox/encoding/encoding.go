// Package encoding maps character codes to glyph names.
//
// Port of org.apache.fontbox.encoding.
package encoding

// encodingEntry is one row of a built-in encoding table: an octal character
// code and the glyph name it stands for.
//
// Java holds these as Object[][] with two named column indexes, which Go has no
// reason to reproduce.
type encodingEntry struct {
	code int
	name string
}

// NotDef is the name of the glyph a code with no mapping stands for.
const NotDef = ".notdef"

// Encoding maps character codes to glyph names and back.
//
// Port of the abstract org.apache.fontbox.encoding.Encoding. Java uses
// inheritance to give the encodings below these methods; the port embeds this
// struct instead.
type Encoding struct {
	codeToName map[int]string
	nameToCode map[string]int
}

// NewEncoding returns an encoding with nothing mapped.
func NewEncoding() *Encoding {
	return &Encoding{
		codeToName: map[int]string{},
		nameToCode: map[string]int{},
	}
}

// AddCharacterEncoding maps one code to one name.
func (e *Encoding) AddCharacterEncoding(code int, name string) {
	e.codeToName[code] = name
	e.nameToCode[name] = code
}

// Code returns the code the named glyph is at, and whether it is mapped at all.
//
// Java returns a boxed Integer and null for an unmapped name; the port returns
// the comma-ok that Go maps already give, since a caller has to test either way.
func (e *Encoding) Code(name string) (int, bool) {
	code, ok := e.nameToCode[name]
	return code, ok
}

// Name returns the glyph name at the given code, or .notdef where the encoding
// does not map it.
func (e *Encoding) Name(code int) string {
	if name, ok := e.codeToName[code]; ok {
		return name
	}
	return NotDef
}

// CodeToNameMap returns the code-to-name mapping, as a copy.
//
// Java returns Collections.unmodifiableMap; Go has no unmodifiable map, so the
// copy stands in for it.
func (e *Encoding) CodeToNameMap() map[int]string {
	out := make(map[int]string, len(e.codeToName))
	for code, name := range e.codeToName {
		out[code] = name
	}
	return out
}

// newFromTable returns an encoding holding the given table.
func newFromTable(table []encodingEntry) *Encoding {
	e := NewEncoding()
	for _, entry := range table {
		e.AddCharacterEncoding(entry.code, entry.name)
	}
	return e
}

// NewBuiltInEncoding returns the encoding a font carries within itself.
//
// Port of org.apache.fontbox.encoding.BuiltInEncoding.
func NewBuiltInEncoding(codeToName map[int]string) *Encoding {
	e := NewEncoding()
	for code, name := range codeToName {
		e.AddCharacterEncoding(code, name)
	}
	return e
}

// StandardEncoding is the Adobe standard encoding.
//
// Port of org.apache.fontbox.encoding.StandardEncoding. Java exposes both a
// public constructor and an INSTANCE; nothing constructs a second one, so the
// port has only the instance.
var StandardEncoding = newFromTable(standardEncodingTable)

// MacRomanEncoding is the Mac OS Roman encoding.
//
// Port of org.apache.fontbox.encoding.MacRomanEncoding.
var MacRomanEncoding = newFromTable(macRomanEncodingTable)
