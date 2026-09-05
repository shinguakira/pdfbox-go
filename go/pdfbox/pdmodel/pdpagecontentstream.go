package pdmodel

import (
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// AppendMode says whether a new content stream replaces the page's content,
// follows it or comes before it.
//
// Port of the nested enum PDPageContentStream.AppendMode.
type AppendMode int

const (
	// Overwrite replaces the content of the page.
	Overwrite AppendMode = iota

	// Append adds the new content after the existing content.
	Append

	// Prepend adds the new content before the existing content.
	Prepend
)

// IsOverwrite reports whether this mode replaces the content.
func (m AppendMode) IsOverwrite() bool { return m == Overwrite }

// IsPrepend reports whether this mode comes before the content.
func (m AppendMode) IsPrepend() bool { return m == Prepend }

// PDPageContentStream writes the content stream of a page.
//
// Port of PDPageContentStream, which Java declares final. The five deprecated
// appendRawCommands methods are not ported.
type PDPageContentStream struct {
	pdAbstractContentStream

	sourcePageHadContents bool
}

// NewPDPageContentStream writes into the given page, replacing its content.
func NewPDPageContentStream(document *PDDocument,
	sourcePage *PDPage) (*PDPageContentStream, error) {
	c, err := NewPDPageContentStreamOfMode(document, sourcePage, Overwrite, true, false)
	if err != nil {
		return nil, err
	}
	if c.sourcePageHadContents {
		slog.Warn("pdmodel: you are overwriting an existing content, you should use the append mode")
	}
	return c, nil
}

// NewPDPageContentStreamCompressed writes into the given page in the given
// mode, deflating the content where compress is true.
func NewPDPageContentStreamCompressed(document *PDDocument, sourcePage *PDPage,
	appendContent AppendMode, compress bool) (*PDPageContentStream, error) {
	return NewPDPageContentStreamOfMode(document, sourcePage, appendContent, compress, false)
}

// NewPDPageContentStreamOfMode writes into the given page in the given mode,
// resetContext saying whether the graphics state is saved before the existing
// content and restored after it.
func NewPDPageContentStreamOfMode(document *PDDocument, sourcePage *PDPage,
	appendContent AppendMode, compress bool, resetContext bool) (*PDPageContentStream, error) {
	resources := sourcePage.Resources()
	if resources == nil {
		resources = NewPDResources()
	}
	return newPDPageContentStream(document, sourcePage, appendContent, compress, resetContext,
		common.NewPDStreamOfDocument(document.Document()), resources)
}

// newPDPageContentStream is the private constructor the four public ones reach.
func newPDPageContentStream(document *PDDocument, sourcePage *PDPage,
	appendContent AppendMode, compress bool, resetContext bool, stream *common.PDStream,
	resources *PDResources) (*PDPageContentStream, error) {
	var filter *cos.Name
	if compress {
		filter = cos.FlateDecode
	}
	outputStream, err := stream.CreateOutputStreamOfFilter(filter)
	if err != nil {
		return nil, err
	}
	c := &PDPageContentStream{}
	c.initAbstractContentStream(document, outputStream, resources)

	// propagate resources to the page
	if sourcePage.Resources() == nil {
		sourcePage.SetResources(resources)
	}

	// If request specifies the need to append/prepend to the document
	if !appendContent.IsOverwrite() && sourcePage.HasContents() {
		// Add new stream to contents array
		contents := sourcePage.Dictionary().GetDictionaryObject(cos.Contents)
		array, isArray := contents.(*cos.Array)
		if !isArray {
			// Creates a new array and adds the current stream plus a new one to it
			array = cos.NewArray()
			array.Add(contents)
		}
		if appendContent.IsPrepend() {
			array.AddAt(0, stream.COSObject())
		} else {
			array.Add(stream.COSObject())
		}

		// save the initial/unmodified graphics context
		if resetContext {
			// create a new stream to prefix existing stream
			prefixStream := common.NewPDStreamOfDocument(document.Document())

			// save the pre-append graphics state
			prefixOut, err := prefixStream.CreateOutputStream()
			if err != nil {
				return nil, err
			}
			if _, err := prefixOut.Write([]byte{'q', '\n'}); err != nil {
				prefixOut.Close()
				return nil, err
			}
			if err := prefixOut.Close(); err != nil {
				return nil, err
			}

			// insert the new stream at the beginning
			array.AddAt(0, prefixStream.COSObject())
		}

		// Sets the compoundStream as page contents
		sourcePage.Dictionary().SetItem(cos.Contents, array)

		// restore the pre-append graphics state
		if resetContext {
			if err := c.RestoreGraphicsState(); err != nil {
				return nil, err
			}
		}
	} else {
		c.sourcePageHadContents = sourcePage.HasContents()
		sourcePage.SetContents(stream)
	}

	// configure NumberFormat
	c.setMaximumFractionDigits(5)
	return c, nil
}

// NewPDPageContentStreamOfAppearance writes into the given appearance stream.
func NewPDPageContentStreamOfAppearance(doc *PDDocument,
	appearance *annotation.PDAppearanceStream) (*PDPageContentStream, error) {
	outputStream, err := appearance.PDStream().CreateOutputStream()
	if err != nil {
		return nil, err
	}
	return NewPDPageContentStreamOfAppearanceOutput(doc, appearance, outputStream), nil
}

// NewPDPageContentStreamOfAppearanceOutput writes into the given output stream,
// with the resources of the given appearance.
func NewPDPageContentStreamOfAppearanceOutput(doc *PDDocument,
	appearance *annotation.PDAppearanceStream,
	outputStream io.WriteCloser) *PDPageContentStream {
	c := &PDPageContentStream{}
	resources, _ := appearance.Resources().(*PDResources)
	c.initAbstractContentStream(doc, outputStream, resources)
	return c
}
