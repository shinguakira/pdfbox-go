package encoding

import (
	"fmt"
	"sort"

	"github.com/shinguakira/pdfbox-go/go/fontbox/afm"
	fbencoding "github.com/shinguakira/pdfbox-go/go/fontbox/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// BuiltInEncoding is the encoding a font carries inside itself.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.BuiltInEncoding.
type BuiltInEncoding struct {
	encodingBase
}

var _ Encoding = (*BuiltInEncoding)(nil)

// NewBuiltInEncoding returns the encoding the given code to name mapping
// describes.
func NewBuiltInEncoding(codeToName map[int]string) *BuiltInEncoding {
	e := &BuiltInEncoding{encodingBase: newEncodingBase()}
	// Java walks a HashMap, whose order is unspecified, and add keeps the first
	// code a name is seen at; the port walks the codes in order, so that the
	// same mapping always gives the same encoding.
	codes := make([]int, 0, len(codeToName))
	for code := range codeToName {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	for _, code := range codes {
		e.add(code, codeToName[code])
	}
	return e
}

// COSObject panics: a built-in encoding cannot be written out.
func (e *BuiltInEncoding) COSObject() cos.Base {
	panic("Built-in encodings cannot be serialized")
}

// EncodingName returns the name of this encoding.
func (e *BuiltInEncoding) EncodingName() string { return "built-in (TTF)" }

// Type1Encoding is the encoding a Type 1 font carries inside itself.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.Type1Encoding.
type Type1Encoding struct {
	encodingBase
}

var _ Encoding = (*Type1Encoding)(nil)

// Type1EncodingFromFontBox returns the encoding an already-parsed Type 1 font
// carries.
func Type1EncodingFromFontBox(encoding *fbencoding.Encoding) *Type1Encoding {
	// todo: could optimise this by looking for specific subclasses
	codeToName := encoding.CodeToNameMap()
	enc := NewType1Encoding()
	// Java walks a HashMap, whose order is unspecified; the port walks the
	// codes in order, for the reason NewBuiltInEncoding gives.
	codes := make([]int, 0, len(codeToName))
	for code := range codeToName {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	for _, code := range codes {
		enc.add(code, codeToName[code])
	}
	return enc
}

// NewType1Encoding returns an empty Type 1 encoding.
func NewType1Encoding() *Type1Encoding {
	return &Type1Encoding{encodingBase: newEncodingBase()}
}

// NewType1EncodingFromMetrics returns the encoding the character metrics of an
// AFM file describe.
func NewType1EncodingFromMetrics(fontMetrics *afm.FontMetrics) *Type1Encoding {
	e := NewType1Encoding()
	for _, nextMetric := range fontMetrics.CharMetrics() {
		e.add(nextMetric.CharacterCode(), nextMetric.Name())
	}
	return e
}

// COSObject returns nothing: a Type 1 encoding has no name of its own.
func (e *Type1Encoding) COSObject() cos.Base { return nil }

// EncodingName returns the name of this encoding.
func (e *Type1Encoding) EncodingName() string { return "built-in (Type 1)" }

// DictionaryEncoding is an encoding written out in the font dictionary: a base
// encoding, and the differences from it.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.DictionaryEncoding.
type DictionaryEncoding struct {
	encodingBase

	encoding     *cos.Dictionary
	baseEncoding Encoding
	differences  map[int]string
}

var _ Encoding = (*DictionaryEncoding)(nil)

// NewDictionaryEncodingFromDifferences returns an encoding that writes itself
// out as the given base encoding and differences.
//
// It fails where the base encoding is not one of the four the specification
// predefines; Java throws IllegalArgumentException.
func NewDictionaryEncodingFromDifferences(baseEncoding *cos.Name, differences *cos.Array) (*DictionaryEncoding, error) {
	e := &DictionaryEncoding{
		encodingBase: newEncodingBase(),
		differences:  map[int]string{},
	}
	e.encoding = cos.NewDictionary()
	e.encoding.SetItem(cos.NameKey, cos.Encoding)
	e.encoding.SetItem(cos.Differences, differences)
	if baseEncoding != cos.StandardEncoding {
		e.encoding.SetItem(cos.BaseEncoding, baseEncoding)
	}
	e.baseEncoding = GetInstance(baseEncoding)

	if e.baseEncoding == nil {
		return nil, fmt.Errorf("encoding: Invalid encoding: %v", baseEncoding)
	}
	e.copyFrom(e.baseEncoding)
	e.applyDifferences()
	return e, nil
}

// NewDictionaryEncodingForType3 returns the encoding of a Type 3 font, which
// carries its whole encoding in the differences.
func NewDictionaryEncodingForType3(fontEncoding *cos.Dictionary) *DictionaryEncoding {
	e := &DictionaryEncoding{
		encodingBase: newEncodingBase(),
		encoding:     fontEncoding,
		differences:  map[int]string{},
	}
	name := e.encoding.GetCOSName(cos.BaseEncoding)
	if name != nil {
		e.baseEncoding = GetInstance(name) // nil when the name is invalid
		if e.baseEncoding != nil {
			// PDFBOX-5963
			// PDF Specification: "Differences array shall specify the complete
			// character encoding for this font" but other viewers read it, thus
			// we do too.
			e.copyFrom(e.baseEncoding)
		}
	}
	e.applyDifferences()
	return e
}

// NewDictionaryEncodingWithBuiltIn returns the encoding of a simple font: the
// base encoding the dictionary names, falling back to the standard encoding for
// a non-symbolic font and to the built-in encoding for a symbolic one.
//
// It fails where a symbolic font has no built-in encoding; Java throws
// IllegalArgumentException, and says that indicates a bug in PDFBox.
func NewDictionaryEncodingWithBuiltIn(fontEncoding *cos.Dictionary, isNonSymbolic bool, builtIn Encoding) (*DictionaryEncoding, error) {
	e := &DictionaryEncoding{
		encodingBase: newEncodingBase(),
		encoding:     fontEncoding,
		differences:  map[int]string{},
	}

	var base Encoding
	if e.encoding.ContainsKey(cos.BaseEncoding) {
		name := e.encoding.GetCOSName(cos.BaseEncoding)
		base = GetInstance(name) // nil when the name is invalid
	}

	if base == nil {
		if isNonSymbolic {
			// Otherwise, for a nonsymbolic font, it is StandardEncoding
			base = StandardEncodingInstance
		} else {
			// and for a symbolic font, it is the font's built-in encoding.
			if builtIn != nil {
				base = builtIn
			} else {
				// triggering this error indicates a bug in PDFBox. Every font
				// should always have a built-in encoding, if not, we parsed it
				// incorrectly.
				return nil, fmt.Errorf("encoding: Symbolic fonts must have a built-in encoding")
			}
		}
	}
	e.baseEncoding = base
	e.copyFrom(e.baseEncoding)
	e.applyDifferences()
	return e, nil
}

// copyFrom takes the whole mapping of another encoding, which the differences
// then change.
func (e *DictionaryEncoding) copyFrom(other Encoding) {
	base := other.base()
	for code, name := range base.codeToName {
		e.codeToName[code] = name
	}
	for name, code := range base.inverted {
		e.inverted[name] = code
	}
}

// applyDifferences reads the differences array, each run of names following the
// code the run starts at.
func (e *DictionaryEncoding) applyDifferences() {
	// now replace with the differences
	diffArray := e.encoding.GetCOSArray(cos.Differences)
	if diffArray == nil {
		return
	}
	currentIndex := -1
	for i := 0; i < diffArray.Size(); i++ {
		next := diffArray.GetObject(i)
		switch value := next.(type) {
		case cos.Number:
			currentIndex = value.IntValue()
		case *cos.Name:
			e.overwrite(currentIndex, value.Name())
			e.differences[currentIndex] = value.Name()
			currentIndex++
		}
	}
}

// BaseEncoding returns the encoding the differences are applied to, or nil
// where a Type 3 font gave none.
func (e *DictionaryEncoding) BaseEncoding() Encoding { return e.baseEncoding }

// Differences returns the codes the encoding changes, and what it changes them
// to.
func (e *DictionaryEncoding) Differences() map[int]string { return e.differences }

// COSObject returns the dictionary the encoding was read from.
func (e *DictionaryEncoding) COSObject() cos.Base { return e.encoding }

// EncodingName returns the name of this encoding.
func (e *DictionaryEncoding) EncodingName() string {
	if e.baseEncoding == nil {
		// In type 3 the /Differences array shall specify the complete character
		// encoding
		return "differences"
	}
	return e.baseEncoding.EncodingName() + " with differences"
}
