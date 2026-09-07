package tools

// This is the main program that will take a pdf document and split it into a
// number of other documents.
//
// Port of org.apache.pdfbox.tools.PDFSplit.

import (
	"flag"
	"strconv"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// PDFSplit is the `pdfsplit` command.
type PDFSplit struct {
	streams
	mixinStandardHelpOptions

	password     string
	split        int
	startPage    int
	endPage      int
	outputPrefix string
	infile       string
}

var _ Command = (*PDFSplit)(nil)

// NewPDFSplit returns the command with its option defaults, which are all -1
// so that "unset" can be told from "set to 1".
func NewPDFSplit() *PDFSplit {
	return &PDFSplit{split: -1, startPage: -1, endPage: -1}
}

// Name is @Command(name = "pdfsplit").
func (p *PDFSplit) Name() string { return "pdfsplit" }

// Header is @Command(header = ...).
func (p *PDFSplit) Header() string {
	return "Splits a PDF document into number of new documents"
}

// Flags declares the six options.
func (p *PDFSplit) Flags(set *flag.FlagSet) {
	set.StringVar(&p.password, "password", "", "the password to decrypt the document.")
	set.IntVar(&p.split, "split", -1,
		"split after this many pages (default 1, if startPage and endPage are unset).")
	set.IntVar(&p.startPage, "startPage", -1, "start page.")
	set.IntVar(&p.endPage, "endPage", -1, "end page.")
	set.StringVar(&p.outputPrefix, "outputPrefix", "", "the filename prefix for split files.")
	set.StringVar(&p.infile, "i", "", "the PDF file to split")
	set.StringVar(&p.infile, "input", "", "the PDF file to split")
}

// validate is `required = true` on -i.
func (p *PDFSplit) validate(set *flag.FlagSet) error {
	return requireSet(set, "--input=<infile>", "i", "input")
}

// Call splits the document.
func (p *PDFSplit) Call() int {
	splitter := multipdf.NewSplitter()
	if p.outputPrefix == "" {
		p.outputPrefix = removeExtension(absolutePath(p.infile))
	}

	if err := p.split1(splitter); err != nil {
		p.printlnErr("Error splitting document: " + err.Error())
		return 4
	}
	return ExitOK
}

// split1 is the body of the try-with-resources. Java's finally closes every
// document the splitter produced, whether or not saving them worked.
func (p *PDFSplit) split1(splitter *multipdf.Splitter) error {
	document, err := pdfbox.LoadPDFWithPassword(p.infile, p.password)
	if err != nil {
		return err
	}
	defer document.Close()

	var documents []*pdmodel.PDDocument
	defer func() {
		for _, doc := range documents {
			// IOUtils.closeQuietly
			doc.Close()
		}
	}()

	startEndPageSet := false
	if p.startPage != -1 {
		splitter.SetStartPage(p.startPage)
		startEndPageSet = true
		if p.split == -1 {
			splitter.SetSplitAtPage(document.NumberOfPages())
		}
	}
	if p.endPage != -1 {
		splitter.SetEndPage(p.endPage)
		startEndPageSet = true
		if p.split == -1 {
			splitter.SetSplitAtPage(p.endPage)
		}
	}
	if p.split != -1 {
		splitter.SetSplitAtPage(p.split)
	} else if !startEndPageSet {
		splitter.SetSplitAtPage(1)
	}

	documents, err = splitter.Split(document)
	if err != nil {
		return err
	}
	for i, doc := range documents {
		// Java closes each one as it saves it, in a try-with-resources, and
		// then closes them all again in the finally; closing twice is harmless
		// on both sides.
		if err := doc.SaveToFile(p.outputPrefix + "-" + strconv.Itoa(i+1) + ".pdf"); err != nil {
			return err
		}
		if err := doc.Close(); err != nil {
			return err
		}
	}
	return nil
}
