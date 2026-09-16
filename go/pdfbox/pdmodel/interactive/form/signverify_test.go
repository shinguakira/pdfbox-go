package form_test

// This file is not a port. PDFBox has no signing test in the pdfbox module --
// TestCreateSignature lives in the examples module and needs BouncyCastle --
// and the slice 8 plan asks for one thing about this path in particular:
// "digital signatures verify or they do not". A signing path that mostly works
// is worse than one that is absent, so these two tests check the only thing
// that can be checked without a CMS implementation, and check it exactly:
//
//   - the bytes the signer is handed are the bytes of the written file with
//     the /Contents hole cut out of it, and
//   - the /ByteRange the file carries describes that same cut.
//
// Together those say that a verifier reading the output would digest exactly
// what the signer digested. What they do not say is anything about the CMS
// blob itself; the port has no signature algorithm and does not pretend to.

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"

	_ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fixup"
)

// catalogFixture is pdfbox's src/test/resources/org/apache/pdfbox/pdmodel.
const catalogFixture = "../../../../../pdfbox/src/test/resources/org/apache/pdfbox/pdmodel/"

// recordingSigner stands in for a CMS signer: it keeps the bytes it was handed
// and answers a deterministic blob derived from them.
type recordingSigner struct {
	signed []byte
	size   int
}

func (s *recordingSigner) Sign(content io.Reader) ([]byte, error) {
	b, err := io.ReadAll(content)
	if err != nil {
		return nil, err
	}
	s.signed = b
	sum := sha256.Sum256(b)
	// Fill the reserved space so that the /Contents hole is exercised at a
	// realistic width; the writer must accept anything up to it.
	out := make([]byte, s.size)
	copy(out, sum[:])
	return out, nil
}

// TestSignedBytesAreTheFileWithTheContentsHoleCutOut signs a document and then
// checks the output against the bytes the signer saw.
func TestSignedBytesAreTheFileWithTheContentsHoleCutOut(t *testing.T) {
	source := filepath.Join(catalogFixture, "test.unc.pdf")
	doc, err := pdfbox.LoadPDF(source)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	signature := digitalsignature.NewPDSignature()
	signature.SetFilter(cos.AdobePPKLite)
	signature.SetSubFilter(cos.AdbePkcs7Detached)
	signature.SetName("Slice 8")
	signature.SetReason("D9")

	signer := &recordingSigner{size: 64}
	if err := form.AddSignature(doc, signature, signer); err != nil {
		t.Fatal(err)
	}

	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	if err := doc.SaveIncremental(out); err != nil {
		t.Fatal(err)
	}
	written := out.Bytes()

	if len(signer.signed) == 0 {
		t.Fatal("the signer was never called")
	}
	if !bytes.HasPrefix(written, original) {
		t.Fatal("the incremental save did not keep the original bytes")
	}

	// Read the output back and take the /ByteRange the file now carries.
	signed, err := pdfbox.LoadPDFBytes(written)
	if err != nil {
		t.Fatal(err)
	}
	defer signed.Close()
	dictionaries := form.SignatureDictionariesOfDocument(signed)
	if len(dictionaries) != 1 {
		t.Fatalf("SignatureDictionariesOfDocument = %d, want 1", len(dictionaries))
	}
	byteRange := dictionaries[0].ByteRange()
	if len(byteRange) != 4 {
		t.Fatalf("ByteRange = %v, want four numbers", byteRange)
	}

	// The two ranges must be adjacent and must together cover the whole file
	// apart from the /Contents hole.
	if byteRange[0] != 0 {
		t.Errorf("ByteRange[0] = %d, want 0", byteRange[0])
	}
	holeStart := byteRange[1]
	holeEnd := byteRange[2]
	if holeEnd <= holeStart {
		t.Fatalf("ByteRange leaves no hole: %v", byteRange)
	}
	if got, want := byteRange[2]+byteRange[3], len(written); got != want {
		t.Errorf("ByteRange covers up to %d, want the file length %d", got, want)
	}

	// The hole is the hex string of /Contents, brackets included.
	if written[holeStart] != '<' || written[holeEnd-1] != '>' {
		t.Errorf("the hole runs from %q to %q, want it to be the <...> of /Contents",
			written[holeStart], written[holeEnd-1])
	}

	// And the signer saw exactly the file minus the hole.
	want := append(append([]byte{}, written[:holeStart]...), written[holeEnd:]...)
	if !bytes.Equal(signer.signed, want) {
		t.Errorf("the signer saw %d bytes, want the %d outside the /Contents hole",
			len(signer.signed), len(want))
	}

	// The signature that came back is the one the signer produced.
	contents := dictionaries[0].Contents()
	sum := sha256.Sum256(signer.signed)
	if !bytes.HasPrefix(contents, sum[:]) {
		t.Error("/Contents does not hold what the signer answered")
	}

	// SignedContentOfBytes is the accessor a verifier uses; it must answer the
	// same bytes the signer saw.
	signedContent, err := dictionaries[0].SignedContentOfBytes(written)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(signedContent, signer.signed) {
		t.Errorf("SignedContentOfBytes = %d bytes, want the %d the signer saw",
			len(signedContent), len(signer.signed))
	}
}

// TestExternalSigningSignsTheSameBytes runs the other half of the path: the
// caller asks for the data to sign, signs it elsewhere and writes the result
// back.
func TestExternalSigningSignsTheSameBytes(t *testing.T) {
	source := filepath.Join(catalogFixture, "test.unc.pdf")
	doc, err := pdfbox.LoadPDF(source)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	signature := digitalsignature.NewPDSignature()
	signature.SetFilter(cos.AdobePPKLite)
	signature.SetSubFilter(cos.AdbePkcs7Detached)
	if err := form.AddSignature(doc, signature, nil); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	support, err := form.SaveIncrementalForExternalSigning(doc, out)
	if err != nil {
		t.Fatal(err)
	}
	dataToSign, err := support.Content()
	if err != nil {
		t.Fatal(err)
	}
	toSign, err := io.ReadAll(dataToSign)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(toSign)
	if err := support.SetSignature(sum[:]); err != nil {
		t.Fatal(err)
	}
	if err := support.Close(); err != nil {
		t.Fatal(err)
	}
	written := out.Bytes()

	signed, err := pdfbox.LoadPDFBytes(written)
	if err != nil {
		t.Fatal(err)
	}
	defer signed.Close()
	dictionaries := form.SignatureDictionariesOfDocument(signed)
	if len(dictionaries) != 1 {
		t.Fatalf("SignatureDictionariesOfDocument = %d, want 1", len(dictionaries))
	}
	signedContent, err := dictionaries[0].SignedContentOfBytes(written)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(signedContent, toSign) {
		t.Errorf("SignedContentOfBytes = %d bytes, want the %d handed to the signer",
			len(signedContent), len(toSign))
	}
	if got := dictionaries[0].Contents(); !bytes.HasPrefix(got, sum[:]) {
		t.Error("/Contents does not hold the signature that was written back")
	}
}

// TestExternalSigningSubsetsTheFontsItWasToldTo pins the first line of
// PDDocument.saveIncrementalForExternalSigning, `subsetDesignatedFonts()`.
//
// A caller may write content before signing, and PDAbstractContentStream.SetFont
// records every font that answers WillBeSubset in the document's set. Java
// subsets that set before it writes the increment the signer is handed; the port
// wrote the increment with the whole font in it, because this function is in
// this package and the method was unexported in pdmodel.
//
// What subsetting does to the file is add the six-character tag to the font's
// name -- `ABCDEF+LiberationSans` -- so the reloaded name is the assertion.
func TestExternalSigningSubsetsTheFontsItWasToldTo(t *testing.T) {
	doc, err := pdfbox.LoadPDF(filepath.Join(catalogFixture, "test.unc.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	ttf, err := os.Open(filepath.Join("..", "..", "..", "..", "..", "pdfbox", "src", "main",
		"resources", "org", "apache", "pdfbox", "resources", "ttf", "LiberationSans-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	defer ttf.Close()
	embedded, err := font.LoadPDType0Font(doc, ttf)
	if err != nil {
		t.Fatal(err)
	}
	if !embedded.WillBeSubset() {
		t.Fatal("the font was loaded without subsetting; there would be nothing to test")
	}

	page := doc.Page(0)
	content, err := pdmodel.NewPDPageContentStreamCompressed(doc, page, pdmodel.Append, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := content.BeginText(); err != nil {
		t.Fatal(err)
	}
	if err := content.SetFont(embedded, 12); err != nil {
		t.Fatal(err)
	}
	if err := content.NewLineAtOffset(75, 750); err != nil {
		t.Fatal(err)
	}
	if err := content.ShowText("Apache PDFBox"); err != nil {
		t.Fatal(err)
	}
	if err := content.EndText(); err != nil {
		t.Fatal(err)
	}
	if err := content.Close(); err != nil {
		t.Fatal(err)
	}
	page.Dictionary().SetNeedToBeUpdated(true)

	signature := digitalsignature.NewPDSignature()
	signature.SetFilter(cos.AdobePPKLite)
	signature.SetSubFilter(cos.AdbePkcs7Detached)
	if err := form.AddSignature(doc, signature, nil); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	support, err := form.SaveIncrementalForExternalSigning(doc, out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := support.Content(); err != nil {
		t.Fatal(err)
	}
	if err := support.SetSignature(make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
	if err := support.Close(); err != nil {
		t.Fatal(err)
	}

	signed, err := pdfbox.LoadPDFBytes(out.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer signed.Close()
	var liberation string
	for _, name := range signed.Page(0).Resources().FontNames() {
		reloaded, err := signed.Page(0).Resources().GetFont(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(reloaded.Name(), "LiberationSans") {
			liberation = reloaded.Name()
		}
	}
	if liberation == "" {
		t.Fatal("the signed page does not carry the font that was written into it")
	}
	if !strings.Contains(liberation, "+") {
		t.Errorf("the signed file carries %q, a font that was not subset; PDFBox subsets "+
			"the designated fonts before it writes the increment", liberation)
	}
}
