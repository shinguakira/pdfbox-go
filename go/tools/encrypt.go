package tools

// This will read an unencrypted document and encrypt it either using a password
// or a certificate. While encrypting the document permissions can be set which
// will allow/disallow certain functionality.
//
// Port of org.apache.pdfbox.tools.Encrypt.

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"strings"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// Encrypt is the `encrypt` command.
type Encrypt struct {
	streams
	mixinStandardHelpOptions

	ownerPassword string
	userPassword  string
	certFileList  repeatedFiles

	canAssembleDocument        bool
	canExtractContent          bool
	canExtractForAccessibility bool
	canFillInForm              bool
	canModify                  bool
	canModifyAnnotations       bool
	canPrint                   bool
	canPrintFaithful           bool

	keyLength int

	infile  string
	outfile string
}

var _ Command = (*Encrypt)(nil)

// NewEncrypt returns the command with its option defaults, every permission
// allowed and a 256-bit key.
func NewEncrypt() *Encrypt {
	return &Encrypt{
		canAssembleDocument:        true,
		canExtractContent:          true,
		canExtractForAccessibility: true,
		canFillInForm:              true,
		canModify:                  true,
		canModifyAnnotations:       true,
		canPrint:                   true,
		canPrintFaithful:           true,
		keyLength:                  256,
	}
}

// Name is @Command(name = "encrypt").
func (e *Encrypt) Name() string { return "encrypt" }

// Header is @Command(header = ...).
func (e *Encrypt) Header() string { return "Encrypts a PDF document" }

// Flags declares the thirteen options.
func (e *Encrypt) Flags(set *flag.FlagSet) {
	set.StringVar(&e.ownerPassword, "O", "",
		"set the owner password (ignored if certFile is set)")
	set.StringVar(&e.userPassword, "U", "",
		"set the user password (ignored if certFile is set)")
	set.Var(&e.certFileList, "certFile",
		"Path to X.509 certificate (repeat both if needed)")
	set.BoolVar(&e.canAssembleDocument, "canAssemble", true,
		"set the assemble permission (default: true)")
	set.BoolVar(&e.canExtractContent, "canExtractContent", true,
		"set the extraction permission (default: true)")
	set.BoolVar(&e.canExtractForAccessibility, "canExtractForAccessibility", true,
		"set the extraction permission (default: true)")
	set.BoolVar(&e.canFillInForm, "canFillInForm", true,
		"set the form fill in permission (default: true)")
	set.BoolVar(&e.canModify, "canModify", true, "set the modify permission (default: true)")
	set.BoolVar(&e.canModifyAnnotations, "canModifyAnnotations", true,
		"set the modify annots permission (default: true)")
	set.BoolVar(&e.canPrint, "canPrint", true, "set the print permission (default: true)")
	set.BoolVar(&e.canPrintFaithful, "canPrintFaithful", true,
		"set the print faithful permission (default: true)")
	set.IntVar(&e.keyLength, "keyLength", 256,
		"Key length in bits (valid values: 40, 128 or 256) (default: 256)")
	set.StringVar(&e.infile, "i", "", "the PDF file to encrypt")
	set.StringVar(&e.infile, "input", "", "the PDF file to encrypt")
	set.StringVar(&e.outfile, "o", "",
		"the encrypted PDF file. If omitted the original file is overwritten.")
	set.StringVar(&e.outfile, "output", "",
		"the encrypted PDF file. If omitted the original file is overwritten.")
}

// validate is `required = true` on -i.
func (e *Encrypt) validate(set *flag.FlagSet) error {
	return requireSet(set, "--input=<infile>", "i", "input")
}

// Call encrypts the document.
func (e *Encrypt) Call() int {
	ap := encryption.NewAccessPermission()
	ap.SetCanAssembleDocument(e.canAssembleDocument)
	ap.SetCanExtractContent(e.canExtractContent)
	ap.SetCanExtractForAccessibility(e.canExtractForAccessibility)
	ap.SetCanFillInForm(e.canFillInForm)
	ap.SetCanModify(e.canModify)
	ap.SetCanModifyAnnotations(e.canModifyAnnotations)
	ap.SetCanPrint(e.canPrint)
	ap.SetCanPrintFaithful(e.canPrintFaithful)

	if e.outfile == "" {
		e.outfile = e.infile
	}

	if err := e.encrypt(ap); err != nil {
		e.printlnErr("Error encrypting PDF: " + err.Error())
		return 4
	}
	return ExitOK
}

// encrypt is the body of the try-with-resources.
func (e *Encrypt) encrypt(ap *encryption.AccessPermission) error {
	document, err := pdfbox.LoadPDF(e.infile)
	if err != nil {
		return err
	}
	defer document.Close()

	if document.IsEncrypted() {
		e.printlnErr("Error: Document is already encrypted.")
		// Java prints and falls through to `return 0`, so an already-encrypted
		// document is not a failure.
		return nil
	}

	if len(form.SignatureDictionariesOfDocument(document)) != 0 {
		e.printlnErr("Warning: Document contains signatures which will be invalidated " +
			"by encryption.")
	}

	if len(e.certFileList) != 0 {
		ppp := encryption.NewPublicKeyProtectionPolicy()
		// One recipient for every certificate, which is Java's -- and it is
		// Java's bug: the object is built once, outside the loop, and the loop
		// overwrites its certificate and adds the same object again. With two
		// -certFile options the policy holds two references to one recipient
		// carrying the second certificate, and the first is lost. Ported as
		// written; see migration/JAVA-BUGS.md.
		recip := &encryption.PublicKeyRecipient{}
		recip.SetPermission(ap)

		for _, certFile := range e.certFileList {
			certificate, err := readX509Certificate(certFile)
			if err != nil {
				return err
			}
			recip.SetX509(certificate)
			ppp.AddRecipient(recip)
		}

		if err := ppp.SetEncryptionKeyLength(e.keyLength); err != nil {
			return err
		}
		if err := document.Protect(ppp); err != nil {
			return err
		}
		return document.SaveToFile(e.outfile)
	}

	spp := encryption.NewStandardProtectionPolicy(e.ownerPassword, e.userPassword, ap)
	if err := spp.SetEncryptionKeyLength(e.keyLength); err != nil {
		return err
	}
	if err := document.Protect(spp); err != nil {
		return err
	}
	return document.SaveToFile(e.outfile)
}

// notPortedError says what a command cannot do and what it waits for.
type notPortedError struct{ what, why string }

func (e *notPortedError) Error() string { return e.what + " is not supported: " + e.why }

// repeatedFiles is a picocli option that may be given more than once, which
// `flag` has no shape for: Var with an appending Set is the standard way.
type repeatedFiles []string

func (f *repeatedFiles) String() string { return strings.Join(*f, ",") }

func (f *repeatedFiles) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// readX509Certificate reads a certificate file, in either of the two shapes
// Java's CertificateFactory takes: DER, or the PEM that wraps it.
func readX509Certificate(path string) (*x509.Certificate, error) {
	der, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tools: reading the certificate %s: %w", path, err)
	}
	if block, _ := pem.Decode(der); block != nil {
		der = block.Bytes
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("tools: reading the certificate %s: %w", path, err)
	}
	return certificate, nil
}
