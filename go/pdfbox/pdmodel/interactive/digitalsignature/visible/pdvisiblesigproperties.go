package visible

import (
	"io"
	"os"
	"strconv"
)

// itoa is Integer.toString, which the port needs where Java concatenates an int
// into a string.
func itoa(value int) string { return strconv.Itoa(value) }

// openFile opens the file at the given path for reading, which is the
// BufferedInputStream over a FileInputStream of Java.
func openFile(path string) (io.ReadCloser, error) { return os.Open(path) }

// PDVisibleSigProperties gathers what a visible signature is made of, and
// builds the one page document holding its appearance.
//
// Port of PDVisibleSigProperties.
type PDVisibleSigProperties struct {
	signerName        string
	signerLocation    string
	signatureReason   string
	visualSignEnabled bool
	page              int
	preferredSize     int
	visibleSignature  io.Reader

	pdVisibleSignature *PDVisibleSignDesigner
}

// BuildSignature builds the one page document holding the appearance.
func (p *PDVisibleSigProperties) BuildSignature() error {
	builder := NewPDVisibleSigBuilder()
	creator := NewPDFTemplateCreator(builder)
	visibleSignature, err := creator.BuildPDF(p.PdVisibleSignature())
	if err != nil {
		return err
	}
	p.SetVisibleSignature(visibleSignature)
	return nil
}

// SignerName returns the name of the signer.
func (p *PDVisibleSigProperties) SignerName() string { return p.signerName }

// SignerNameOf sets the name of the signer.
func (p *PDVisibleSigProperties) SignerNameOf(signerName string) *PDVisibleSigProperties {
	p.signerName = signerName
	return p
}

// SignerLocation returns where the signer is.
func (p *PDVisibleSigProperties) SignerLocation() string { return p.signerLocation }

// SignerLocationOf sets where the signer is.
func (p *PDVisibleSigProperties) SignerLocationOf(signerLocation string) *PDVisibleSigProperties {
	p.signerLocation = signerLocation
	return p
}

// SignatureReason returns the reason for the signing.
func (p *PDVisibleSigProperties) SignatureReason() string { return p.signatureReason }

// SignatureReasonOf sets the reason for the signing.
func (p *PDVisibleSigProperties) SignatureReasonOf(signatureReason string) *PDVisibleSigProperties {
	p.signatureReason = signatureReason
	return p
}

// Page returns the 1-based page the signature goes on.
func (p *PDVisibleSigProperties) Page() int { return p.page }

// PageOf sets the 1-based page the signature goes on.
func (p *PDVisibleSigProperties) PageOf(page int) *PDVisibleSigProperties {
	p.page = page
	return p
}

// PreferredSize returns how much space the signature is given.
func (p *PDVisibleSigProperties) PreferredSize() int { return p.preferredSize }

// PreferredSizeOf sets how much space the signature is given.
func (p *PDVisibleSigProperties) PreferredSizeOf(preferredSize int) *PDVisibleSigProperties {
	p.preferredSize = preferredSize
	return p
}

// IsVisualSignEnabled reports whether the signature is drawn.
func (p *PDVisibleSigProperties) IsVisualSignEnabled() bool { return p.visualSignEnabled }

// VisualSignEnabledOf sets whether the signature is drawn.
func (p *PDVisibleSigProperties) VisualSignEnabledOf(
	visualSignEnabled bool) *PDVisibleSigProperties {
	p.visualSignEnabled = visualSignEnabled
	return p
}

// PdVisibleSignature returns the designer of the appearance.
func (p *PDVisibleSigProperties) PdVisibleSignature() *PDVisibleSignDesigner {
	return p.pdVisibleSignature
}

// SetPdVisibleSignature sets the designer of the appearance.
func (p *PDVisibleSigProperties) SetPdVisibleSignature(
	pdVisibleSignature *PDVisibleSignDesigner) *PDVisibleSigProperties {
	p.pdVisibleSignature = pdVisibleSignature
	return p
}

// VisibleSignature returns the one page document holding the appearance.
func (p *PDVisibleSigProperties) VisibleSignature() io.Reader { return p.visibleSignature }

// SetVisibleSignature sets the one page document holding the appearance.
func (p *PDVisibleSigProperties) SetVisibleSignature(visibleSignature io.Reader) {
	p.visibleSignature = visibleSignature
}
