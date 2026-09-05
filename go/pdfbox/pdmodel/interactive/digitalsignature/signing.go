package digitalsignature

import (
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// SignatureInterface provides an interface for the signing.
//
// Port of the interface SignatureInterface.
type SignatureInterface interface {
	// Sign creates a CMS signature of the given content, which is the byte
	// sequence the /ByteRange of the signature dictionary covers.
	Sign(content io.Reader) ([]byte, error)
}

// ExternalSigningSupport is the interface for external signature creation
// scenarios.
//
// Port of the interface ExternalSigningSupport.
type ExternalSigningSupport interface {
	// Content returns the PDF content to be signed.
	Content() (io.Reader, error)

	// SetSignature sets the CMS signature bytes as the value of the /Contents
	// entry of the signature dictionary.
	SetSignature(signature []byte) error
}

// COSWriterLike is the half of pdfwriter.COSWriter that SigningSupport uses.
//
// Java names COSWriter here; the port cannot, because pdfwriter imports this
// package for COSFilterInputStream, and Go forbids the cycle.
type COSWriterLike interface {
	// DataToSign returns the PDF content the signature covers.
	DataToSign() (io.Reader, error)

	// WriteExternalSignature writes the given CMS signature into the space the
	// signature dictionary reserved.
	WriteExternalSignature(cmsSignature []byte) error
}

// SigningSupport is the implementation of ExternalSigningSupport.
//
// Port of SigningSupport, which implements ExternalSigningSupport and Closeable.
type SigningSupport struct {
	cosWriter COSWriterLike
}

var _ ExternalSigningSupport = (*SigningSupport)(nil)

// NewSigningSupport returns the support writing through the given writer.
func NewSigningSupport(cosWriter COSWriterLike) *SigningSupport {
	return &SigningSupport{cosWriter: cosWriter}
}

// Content returns the PDF content to be signed.
func (s *SigningSupport) Content() (io.Reader, error) { return s.cosWriter.DataToSign() }

// SetSignature writes the CMS signature into the space the signature dictionary
// reserved.
func (s *SigningSupport) SetSignature(signature []byte) error {
	return s.cosWriter.WriteExternalSignature(signature)
}

// Close drops the writer, which is what Java does to make later use fail.
func (s *SigningSupport) Close() error {
	s.cosWriter = nil
	return nil
}

// DefaultSignatureSize is the space a signature is given when no other size is
// asked for.
//
// Port of SignatureOptions.DEFAULT_SIGNATURE_SIZE.
const DefaultSignatureSize = 0x2500

// VisibleSignatureProperties is the half of visible.PDVisibleSigProperties that
// SetVisualSignatureOfProperties uses.
//
// Java names PDVisibleSigProperties here; the port cannot, because that package
// names PDSignature, which lives here.
type VisibleSignatureProperties interface {
	// VisibleSignature returns the one page document holding the appearance.
	VisibleSignature() io.Reader
}

// SignatureOptions holds the parts of the signing that are not the signature
// itself: which page it goes on, how much space it needs, and the appearance it
// is drawn with.
//
// Port of SignatureOptions, which implements Closeable.
type SignatureOptions struct {
	visualSignature        *cos.Document
	preferredSignatureSize int
	pageNo                 int

	// pdfSource is the pdf to be read. This is done analog to PDDocument.
	pdfSource pdfio.RandomAccessRead
}

// NewSignatureOptions returns the options for a signature on the first page.
func NewSignatureOptions() *SignatureOptions {
	return &SignatureOptions{pageNo: 0}
}

// SetPage sets the 0-based page number the signature goes on.
func (o *SignatureOptions) SetPage(pageNo int) { o.pageNo = pageNo }

// Page returns the 0-based page number the signature goes on.
func (o *SignatureOptions) Page() int { return o.pageNo }

// SetVisualSignature reads the appearance of the signature out of the given one
// page document.
//
// Java has three overloads: File, InputStream and PDVisibleSigProperties. The
// first two both end in initFromRandomAccessRead, and the port takes the read
// itself, because a Go caller opens the file.
func (o *SignatureOptions) SetVisualSignature(rar pdfio.RandomAccessRead) error {
	return o.initFromRandomAccessRead(rar)
}

// initFromRandomAccessRead parses the appearance document. Java declares it
// private.
func (o *SignatureOptions) initFromRandomAccessRead(rar pdfio.RandomAccessRead) error {
	o.pdfSource = rar
	parser, err := pdfparser.NewPDFParser(o.pdfSource, nil, nil)
	if err != nil {
		return err
	}
	document, err := parser.Parse(false)
	if err != nil {
		return err
	}
	o.visualSignature = document
	return nil
}

// SetVisualSignatureOfProperties reads the appearance of the signature out of
// the given visible signature properties.
//
// Port of setVisualSignature(PDVisibleSigProperties).
func (o *SignatureOptions) SetVisualSignatureOfProperties(
	visSignatureProperties VisibleSignatureProperties) error {
	content, err := io.ReadAll(visSignatureProperties.VisibleSignature())
	if err != nil {
		return err
	}
	return o.initFromRandomAccessRead(pdfio.NewReadBufferBytes(content))
}

// VisualSignature returns the document holding the appearance of the signature,
// or nil where there is none.
func (o *SignatureOptions) VisualSignature() *cos.Document { return o.visualSignature }

// PreferredSignatureSize returns how much space the signature is given, or 0
// where no size was asked for.
func (o *SignatureOptions) PreferredSignatureSize() int { return o.preferredSignatureSize }

// SetPreferredSignatureSize sets how much space the signature is given, and
// ignores a size that is not positive.
func (o *SignatureOptions) SetPreferredSignatureSize(size int) {
	if size > 0 {
		o.preferredSignatureSize = size
	}
}

// Close closes the appearance document and the read it came from.
func (o *SignatureOptions) Close() error {
	var firstError error
	if o.visualSignature != nil {
		firstError = o.visualSignature.Close()
	}
	if o.pdfSource != nil {
		if err := o.pdfSource.Close(); err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
}
