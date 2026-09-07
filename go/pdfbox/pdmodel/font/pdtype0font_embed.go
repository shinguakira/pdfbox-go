package font

// The embedding half of PDType0Font: the load factories that write a TrueType
// font into a document.
//
// Port of the load and loadVertical statics of
// org.apache.pdfbox.pdmodel.font.PDType0Font and of the private constructor
// they all funnel into. The reading half is in pdtype0font.go, which said this
// was waiting for a later slice; this is it.
//
// Java distinguishes nine overloads by their argument lists. Go names them: the
// two-argument form is the plain one, and the suffix says what the extra
// argument controls, which is the shape SetupMainMemoryOnly and
// SetupMainMemoryOnlyMax already use in this port.

import (
	"io"
	"os"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// LoadPDType0Font loads a TTF to be embedded into a document as a Type 0 font,
// subsetting it.
//
// Port of load(PDDocument, InputStream), which passes embedSubset true.
func LoadPDType0Font(doc embeddingDocument, input io.Reader) (*PDType0Font, error) {
	return LoadPDType0FontSubset(doc, input, true)
}

// LoadPDType0FontSubset loads a TTF to be embedded into a document as a Type 0
// font, subsetting it where embedSubset says to.
//
// Port of load(PDDocument, InputStream, boolean).
func LoadPDType0FontSubset(doc embeddingDocument, input io.Reader,
	embedSubset bool) (*PDType0Font, error) {
	source, err := pdfio.NewReadBufferFromReader(input)
	if err != nil {
		return nil, err
	}
	return loadPDType0FontFromSource(doc, source, embedSubset, false)
}

// LoadPDType0FontFile loads a TTF from a file to be embedded into a document as
// a Type 0 font, subsetting it.
//
// Port of load(PDDocument, File).
func LoadPDType0FontFile(doc embeddingDocument, path string) (*PDType0Font, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	source, err := pdfio.NewReadBufferFromReader(file)
	if err != nil {
		return nil, err
	}
	return loadPDType0FontFromSource(doc, source, true, false)
}

// LoadPDType0FontTTF loads an already parsed TTF to be embedded into a document
// as a Type 0 font.
//
// Port of load(PDDocument, TrueTypeFont, boolean), which does not close the
// font it is handed.
func LoadPDType0FontTTF(doc embeddingDocument, font *ttf.TrueTypeFont,
	embedSubset bool) (*PDType0Font, error) {
	return newEmbeddedPDType0Font(doc, font, embedSubset, false, false)
}

// LoadPDType0FontVertical loads a TTF to be embedded into a document as a
// vertical Type 0 font, subsetting it.
//
// Port of loadVertical(PDDocument, InputStream).
func LoadPDType0FontVertical(doc embeddingDocument, input io.Reader) (*PDType0Font, error) {
	return LoadPDType0FontVerticalSubset(doc, input, true)
}

// LoadPDType0FontVerticalSubset loads a TTF to be embedded into a document as a
// vertical Type 0 font, subsetting it where embedSubset says to.
//
// Port of loadVertical(PDDocument, InputStream, boolean).
func LoadPDType0FontVerticalSubset(doc embeddingDocument, input io.Reader,
	embedSubset bool) (*PDType0Font, error) {
	source, err := pdfio.NewReadBufferFromReader(input)
	if err != nil {
		return nil, err
	}
	return loadPDType0FontFromSource(doc, source, embedSubset, true)
}

// loadPDType0FontFromSource parses the font and embeds it.
//
// Port of load(PDDocument, RandomAccessRead, boolean, boolean), which is where
// the stream overloads meet and which closes the font it parsed.
func loadPDType0FontFromSource(doc embeddingDocument, source pdfio.RandomAccessRead,
	embedSubset, vertical bool) (*PDType0Font, error) {
	parsed, err := ttf.NewParser().Parse(source)
	if err != nil {
		return nil, err
	}
	return newEmbeddedPDType0Font(doc, parsed, embedSubset, true, vertical)
}

// newEmbeddedPDType0Font builds a Type 0 font around a font program that is to
// be written into the document.
//
// Port of the private PDType0Font(PDDocument, TrueTypeFont, boolean, boolean,
// boolean) constructor.
func newEmbeddedPDType0Font(doc embeddingDocument, font *ttf.TrueTypeFont,
	embedSubset, closeTTF, vertical bool) (*PDType0Font, error) {
	f := &PDType0Font{
		pdFont:    newPDFont(),
		noUnicode: map[int]bool{},
	}
	f.pdFont.self = f

	if vertical {
		font.EnableVerticalSubstitutions()
	}

	gsubData, err := font.GsubData()
	if err != nil {
		return nil, err
	}
	f.gsubData = gsubData

	// Java calls the no-argument getUnicodeCmapLookup, which is the strict one.
	lookup, err := font.UnicodeCmapLookup(true)
	if err != nil {
		return nil, err
	}
	f.cmapLookup = lookup

	embedder, err := newPDCIDFontType2Embedder(doc, f.dict, font, embedSubset, vertical)
	if err != nil {
		return nil, err
	}
	f.embedder = embedder

	descendant, err := embedder.CIDFont()
	if err != nil {
		return nil, err
	}
	f.descendantFont = descendant

	if err := f.readEncoding(); err != nil {
		return nil, err
	}
	f.fetchCMapUCS2()

	if closeTTF {
		if embedSubset {
			// the font has to stay open until the document is saved, because
			// the subset is not built until then
			f.ttf = font
			doc.RegisterTrueTypeFontForClosing(font)
		} else {
			// the TTF is fully loaded and it is safe to close the underlying
			// data source
			if err := font.Close(); err != nil {
				return nil, err
			}
		}
	}
	return f, nil
}
