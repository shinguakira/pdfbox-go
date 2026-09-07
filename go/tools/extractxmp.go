package tools

// Extracts the XMP stream from a PDF document.
//
// Port of org.apache.pdfbox.tools.ExtractXMP.

import (
	"flag"
	"os"
	"strconv"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// ExtractXMP is the `extractxmp` command.
type ExtractXMP struct {
	streams
	mixinStandardHelpOptions

	page      int
	password  string
	toConsole bool
	infile    string
	outfile   string
}

var _ Command = (*ExtractXMP)(nil)

// NewExtractXMP returns the command.
func NewExtractXMP() *ExtractXMP { return &ExtractXMP{} }

// Name is @Command(name = "extractxmp").
func (e *ExtractXMP) Name() string { return "extractxmp" }

// Header is @Command(header = ...).
func (e *ExtractXMP) Header() string { return "Extracts the xmp stream from a PDF document" }

// Flags declares the five options.
func (e *ExtractXMP) Flags(set *flag.FlagSet) {
	set.IntVar(&e.page, "page", 0,
		"extract the XMP information from a specific page (1 based)")
	set.StringVar(&e.password, "password", "",
		"the password for the PDF or certificate in keystore.")
	set.BoolVar(&e.toConsole, "console", false, "Send text to console instead of file")
	set.StringVar(&e.infile, "i", "", "the PDF file")
	set.StringVar(&e.infile, "input", "", "the PDF file")
	set.StringVar(&e.outfile, "o", "", "the exported text file")
	set.StringVar(&e.outfile, "output", "", "the exported text file")
}

func (e *ExtractXMP) validate(set *flag.FlagSet) error {
	return requireSet(set, "--input=<infile>", "i", "input")
}

// Call writes the metadata out.
func (e *ExtractXMP) Call() int {
	if e.outfile == "" {
		e.outfile = removeExtension(absolutePath(e.infile)) + ".xml"
	}

	document, err := pdfbox.LoadPDFWithPassword(e.infile, e.password)
	if err != nil {
		// Java's message says "extracting text", which is a copy from
		// ExtractText; the port keeps it.
		e.printlnErr("Error extracting text for document: " + err.Error())
		return 4
	}
	defer document.Close()

	var meta *common.PDMetadata
	if e.page == 0 {
		meta = document.DocumentCatalog().Metadata()
	} else {
		if e.page > document.NumberOfPages() {
			e.printlnErr("Page " + strconv.Itoa(e.page) + " doesn't exist")
			return 1
		}
		meta = document.Page(e.page - 1).Metadata()
	}
	if meta == nil {
		e.printlnErr("No XMP metadata available")
		return 1
	}

	data, err := meta.ToByteArray()
	if err != nil {
		e.printlnErr("Error extracting text for document: " + err.Error())
		return 4
	}
	if e.toConsole {
		if _, err := e.out().Write(data); err != nil {
			e.printlnErr("Error extracting text for document: " + err.Error())
			return 4
		}
		return ExitOK
	}
	if err := os.WriteFile(e.outfile, data, 0o666); err != nil {
		e.printlnErr("Error extracting text for document: " + err.Error())
		return 4
	}
	return ExitOK
}
