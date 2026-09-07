package tools

// This will read an encrypted document and decrypt it either using a password
// or a certificate.
//
// Port of org.apache.pdfbox.tools.Decrypt.

import (
	"flag"
	"io"
	"os"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
)

// Decrypt is the `decrypt` command.
type Decrypt struct {
	streams
	mixinStandardHelpOptions

	alias    string
	keyStore string
	password string
	infile   string
	outfile  string
}

var _ Command = (*Decrypt)(nil)

// NewDecrypt returns the command.
func NewDecrypt() *Decrypt { return &Decrypt{} }

// Name is @Command(name = "decrypt").
func (d *Decrypt) Name() string { return "decrypt" }

// Header is @Command(header = ...).
func (d *Decrypt) Header() string { return "Decrypts a PDF document" }

// Flags declares the five options.
func (d *Decrypt) Flags(set *flag.FlagSet) {
	set.StringVar(&d.alias, "alias", "", "the alias to the certificate in the keystore.")
	set.StringVar(&d.keyStore, "keyStore", "",
		"the path to the keystore that holds the certificate to decrypt the document. "+
			"This is only required if the document is encrypted with a certificate, "+
			"otherwise only the password is required.")
	set.StringVar(&d.password, "password", "",
		"the password for the PDF or certificate in keystore.")
	set.StringVar(&d.infile, "i", "", "the PDF file to decrypt")
	set.StringVar(&d.infile, "input", "", "the PDF file to decrypt")
	set.StringVar(&d.outfile, "o", "",
		"the decrypted PDF file. If omitted the original file is overwritten.")
	set.StringVar(&d.outfile, "output", "",
		"the decrypted PDF file. If omitted the original file is overwritten.")
}

// validate is `required = true` on -i.
func (d *Decrypt) validate(set *flag.FlagSet) error {
	return requireSet(set, "--input=<infile>", "i", "input")
}

// Call decrypts the document.
func (d *Decrypt) Call() int {
	// Java opens the keystore in the try-with-resources, so a null path is a
	// null stream rather than an error.
	var keyStoreStream io.Reader
	if d.keyStore != "" {
		file, err := os.Open(d.keyStore)
		if err != nil {
			d.printlnErr("Error decrypting document: " + err.Error())
			return 4
		}
		defer file.Close()
		keyStoreStream = file
	}

	document, err := pdfbox.LoadPDFWithKeyStore(d.infile, d.password, keyStoreStream, d.alias)
	if err != nil {
		d.printlnErr("Error decrypting document: " + err.Error())
		return 4
	}
	defer document.Close()

	// overwrite inputfile if no outputfile was specified
	if d.outfile == "" {
		d.outfile = d.infile
	}

	if !document.IsEncrypted() {
		d.printlnErr("Error: Document is not encrypted.")
		return 1
	}
	if !document.CurrentAccessPermission().IsOwnerPermission() {
		d.printlnErr("Error: You are only allowed to decrypt a document with the owner password.")
		return 1
	}

	document.SetAllSecurityToBeRemoved(true)
	if err := document.SaveToFile(d.outfile); err != nil {
		d.printlnErr("Error decrypting document: " + err.Error())
		return 4
	}
	return ExitOK
}
