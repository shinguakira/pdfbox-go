package pdfbox

import (
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The overloads of org.apache.pdfbox.Loader.loadPDF that take a password, and
// the two that take a PKCS#12 keystore with it.
//
// Java has sixteen overloads across the three sources it reads from; Go has no
// overloading, so each source keeps its base name and the extra arguments name
// the variant.

// LoadPDFWithPassword opens the PDF file at the given path with the given
// password.
//
// Port of loadPDF(File, String).
func LoadPDFWithPassword(path, password string) (*pdmodel.PDDocument, error) {
	source, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		return nil, err
	}
	document, err := LoadPDFFromWithKeyStore(source, password, nil, "")
	if err != nil {
		source.Close()
		return nil, err
	}
	return document, nil
}

// LoadPDFBytesWithPassword opens the PDF the given bytes hold with the given
// password.
//
// Port of loadPDF(byte[], String).
func LoadPDFBytesWithPassword(input []byte, password string) (*pdmodel.PDDocument, error) {
	return LoadPDFFromWithKeyStore(pdfio.NewReadBufferBytes(input), password, nil, "")
}

// LoadPDFFromWithPassword opens the PDF the given source holds with the given
// password.
//
// Port of loadPDF(RandomAccessRead, String).
func LoadPDFFromWithPassword(source pdfio.RandomAccessRead,
	password string) (*pdmodel.PDDocument, error) {
	return LoadPDFFromWithKeyStore(source, password, nil, "")
}

// LoadPDFWithKeyStore opens the PDF file at the given path, which is encrypted
// for the holder of a certificate in the given PKCS#12 keystore.
//
// Port of loadPDF(File, String, InputStream, String).
func LoadPDFWithKeyStore(path, password string, keyStore io.Reader,
	alias string) (*pdmodel.PDDocument, error) {
	source, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		return nil, err
	}
	document, err := LoadPDFFromWithKeyStore(source, password, keyStore, alias)
	if err != nil {
		source.Close()
		return nil, err
	}
	return document, nil
}

// LoadPDFFromWithKeyStore opens the PDF the given source holds.
//
// Where keyStore is nil the password is the document password and the standard
// security handler takes it. Where keyStore is given, the same password opens
// the PKCS#12 keystore instead, and the certificate the alias names decrypts
// the document: Java reads the store with KeyStore.load(keyStore,
// password.toCharArray()) and then hands the string on to
// PublicKeyDecryptionMaterial as the private key password, so the one argument
// carries both. Java's javadoc says only "password to be used for decryption".
//
// Port of loadPDF(RandomAccessRead, String, InputStream, String).
func LoadPDFFromWithKeyStore(source pdfio.RandomAccessRead, password string, keyStore io.Reader,
	alias string) (*pdmodel.PDDocument, error) {
	parser, err := pdfparser.NewPDFParserWithPassword(source, password, keyStore, alias, nil,
		filter.Provider{})
	if err != nil {
		return nil, err
	}
	document, err := parser.Parse(true)
	if err != nil {
		return nil, err
	}
	pdDocument := pdmodel.NewPDDocumentOf(document, source)
	// Java does these two in PDFParser.parse, which returns the PDDocument; the
	// port's parser returns the COS document and the loader wraps it, so they
	// happen here.
	pdDocument.SetAccessPermission(parser.AccessPermission())
	pdDocument.SetEncryptionDictionary(parser.Encryption())
	return pdDocument, nil
}
