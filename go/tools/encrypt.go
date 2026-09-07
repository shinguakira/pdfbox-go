package tools

// This will read an unencrypted document and encrypt it either using a password
// or a certificate. While encrypting the document permissions can be set which
// will allow/disallow certain functionality.
//
// Port of org.apache.pdfbox.tools.Encrypt.

import (
	"flag"
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
		// Public key encryption needs an X.509 certificate and the CMS
		// enveloping around it. The port's PublicKeySecurityHandler reports
		// that its encryption half is not ported; see migration/STATUS.md.
		return errPublicKeyEncryptNotPorted
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

// errPublicKeyEncryptNotPorted is what -certFile answers.
var errPublicKeyEncryptNotPorted = &notPortedError{
	what: "encrypting with a certificate (-certFile)",
	why:  "PublicKeySecurityHandler's encryption half is not ported",
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
