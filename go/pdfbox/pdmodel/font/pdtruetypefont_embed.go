package font

// The embedding half of PDTrueTypeFont, and the embedder behind it.
//
// Port of org.apache.pdfbox.pdmodel.font.PDTrueTypeFontEmbedder, and of the
// load statics of PDTrueTypeFont and the private constructor they funnel into.
//
// This is the *simple* font path: one byte per character, an /Encoding, and a
// /Widths array. It never subsets -- its buildSubset throws
// UnsupportedOperationException with the comment "use PDType0Font instead" --
// so the whole font is written every time.

import (
	"io"
	"math"
	"os"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// pdTrueTypeFontEmbedder embeds a TrueType font as a simple font.
type pdTrueTypeFontEmbedder struct {
	*trueTypeEmbedder

	fontEncoding encoding.Encoding
}

// newPDTrueTypeFontEmbedder creates a new TrueType font embedder for the given
// TTF as a simple font.
func newPDTrueTypeFontEmbedder(document embeddingDocument, dict *cos.Dictionary,
	font *ttf.TrueTypeFont, enc encoding.Encoding) (*pdTrueTypeFontEmbedder, error) {
	base, err := newTrueTypeEmbedder(document, dict, font, false)
	if err != nil {
		return nil, err
	}
	e := &pdTrueTypeFontEmbedder{trueTypeEmbedder: base, fontEncoding: enc}
	base.buildSubsetFromStream = e.buildSubset

	dict.SetItem(cos.Subtype, cos.TrueType)

	glyphList := encoding.AdobeGlyphList()
	dict.SetItem(cos.Encoding, enc.COSObject())
	e.fontDescriptor.SetSymbolic(false)
	e.fontDescriptor.SetNonSymbolic(true)

	// add the font descriptor
	dict.SetItem(cos.FontDescriptor, e.fontDescriptor.COSObject())

	// set the glyph widths
	if err := e.setWidths(dict, glyphList); err != nil {
		return nil, err
	}
	return e, nil
}

// setWidths writes /FirstChar, /LastChar and /Widths.
func (e *pdTrueTypeFontEmbedder) setWidths(font *cos.Dictionary,
	glyphList *encoding.GlyphList) error {
	header, err := e.ttf.Header()
	if err != nil {
		return err
	}
	scaling := 1000 / float32(header.UnitsPerEm())
	hmtx, err := e.ttf.HorizontalMetrics()
	if err != nil {
		return err
	}

	codeToName := e.fontEncoding.CodeToNameMap()
	firstChar, lastChar := math.MaxInt32, math.MinInt32
	for code := range codeToName {
		if code < firstChar {
			firstChar = code
		}
		if code > lastChar {
			lastChar = code
		}
	}

	widths := make([]int, lastChar-firstChar+1)

	// a character code is mapped to a glyph name via the provided font
	// encoding; afterwards, the glyph name is translated to a glyph ID.
	for code, name := range codeToName {
		if code < firstChar || code > lastChar {
			continue
		}
		gid := 0
		if uni := glyphList.ToUnicode(name); uni != "" {
			charCode := firstCodePointOf(uni)
			gid = e.cmapLookup.GetGlyphID(charCode)
		}
		widths[code-firstChar] = int(math.Round(
			float64(float32(hmtx.AdvanceWidth(gid)) * scaling)))
	}

	font.SetInt(cos.FirstChar, firstChar)
	font.SetInt(cos.LastChar, lastChar)
	font.SetItem(cos.Widths, cos.ArrayOfIntegers(widths))
	return nil
}

// firstCodePointOf is String.codePointAt(0).
func firstCodePointOf(s string) int {
	for _, r := range s {
		return int(r)
	}
	return 0
}

// FontEncoding returns the encoding the font was embedded with.
func (e *pdTrueTypeFontEmbedder) FontEncoding() encoding.Encoding { return e.fontEncoding }

// buildSubset is not supported: a simple TrueType font is never subset.
//
// Java throws UnsupportedOperationException, which is unchecked, so the port
// panics. Its comment reads "use PDType0Font instead".
func (e *pdTrueTypeFontEmbedder) buildSubset(ttfSubset []byte, tag string,
	gidToCid map[int]int) error {
	panic("use PDType0Font instead")
}

// LoadPDTrueTypeFont loads a TTF to be embedded into a document as a simple
// font, with the given encoding.
//
// Port of load(PDDocument, InputStream, Encoding).
func LoadPDTrueTypeFont(doc embeddingDocument, input io.Reader,
	enc encoding.Encoding) (*PDTrueTypeFont, error) {
	source, err := pdfio.NewReadBufferFromReader(input)
	if err != nil {
		return nil, err
	}
	return loadPDTrueTypeFontFromSource(doc, source, enc)
}

// LoadPDTrueTypeFontFile loads a TTF from a file to be embedded into a document
// as a simple font.
//
// Port of load(PDDocument, File, Encoding).
func LoadPDTrueTypeFontFile(doc embeddingDocument, path string,
	enc encoding.Encoding) (*PDTrueTypeFont, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	source, err := pdfio.NewReadBufferFromReader(file)
	if err != nil {
		return nil, err
	}
	return loadPDTrueTypeFontFromSource(doc, source, enc)
}

// LoadPDTrueTypeFontTTF loads an already parsed TTF to be embedded into a
// document as a simple font.
//
// Port of load(PDDocument, TrueTypeFont, Encoding), which does not close the
// font it is handed.
func LoadPDTrueTypeFontTTF(doc embeddingDocument, font *ttf.TrueTypeFont,
	enc encoding.Encoding) (*PDTrueTypeFont, error) {
	return newEmbeddedPDTrueTypeFont(doc, font, enc, false)
}

// loadPDTrueTypeFontFromSource parses the font and embeds it.
//
// Port of load(PDDocument, RandomAccessRead, Encoding), which closes the font
// it parsed.
func loadPDTrueTypeFontFromSource(doc embeddingDocument, source pdfio.RandomAccessRead,
	enc encoding.Encoding) (*PDTrueTypeFont, error) {
	parsed, err := ttf.NewParser().Parse(source)
	if err != nil {
		return nil, err
	}
	return newEmbeddedPDTrueTypeFont(doc, parsed, enc, true)
}

// newEmbeddedPDTrueTypeFont builds a simple font around a font program that is
// to be written into the document.
//
// Port of the private PDTrueTypeFont(PDDocument, TrueTypeFont, Encoding,
// boolean) constructor.
func newEmbeddedPDTrueTypeFont(doc embeddingDocument, font *ttf.TrueTypeFont,
	enc encoding.Encoding, closeTTF bool) (*PDTrueTypeFont, error) {
	f := &PDTrueTypeFont{pdSimpleFont: pdSimpleFont{pdFont: newPDFont()}}
	f.pdFont.self = f

	embedder, err := newPDTrueTypeFontEmbedder(doc, f.dict, font, enc)
	if err != nil {
		return nil, err
	}

	f.encoding = enc
	f.ttf = font
	// OpenTypeFonts are not fully supported yet
	f.otf = nil
	f.setFontDescriptor(embedder.FontDescriptor())
	f.isEmbedded = true
	f.isDamaged = false
	f.glyphList = encoding.AdobeGlyphList()

	if closeTTF {
		// the TTF is fully loaded and it is safe to close the underlying data
		// source
		if err := font.Close(); err != nil {
			return nil, err
		}
	}
	return f, nil
}
