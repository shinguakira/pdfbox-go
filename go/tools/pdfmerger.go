package tools

// This is the main program that will take a list of pdf documents and merge
// them, saving the result in a new document.
//
// Port of org.apache.pdfbox.tools.PDFMerger.

import (
	"flag"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
)

// PDFMerger merges multiple PDF documents into one.
type PDFMerger struct {
	streams
	mixinStandardHelpOptions

	infiles repeatedFiles
	outfile string
}

var _ Command = (*PDFMerger)(nil)

// NewPDFMerger returns the command.
func NewPDFMerger() *PDFMerger { return &PDFMerger{} }

// Name is @Command(name = "merge").
func (m *PDFMerger) Name() string { return "merge" }

// Header is @Command(header = ...).
func (m *PDFMerger) Header() string { return "Merges multiple PDF documents into one" }

// Flags declares the two options, both of which are required.
func (m *PDFMerger) Flags(set *flag.FlagSet) {
	set.Var(&m.infiles, "i", "the PDF files to merge.")
	set.Var(&m.infiles, "input", "the PDF files to merge.")
	set.StringVar(&m.outfile, "o", "", "the merged PDF file.")
	set.StringVar(&m.outfile, "output", "", "the merged PDF file.")
}

// validate is `required = true` on both options.
func (m *PDFMerger) validate(set *flag.FlagSet) error {
	if err := requireSet(set, "--input=<infile>", "i", "input"); err != nil {
		return err
	}
	return requireSet(set, "--output=<outfile>", "o", "output")
}

// Call merges the documents.
func (m *PDFMerger) Call() int {
	merger := multipdf.NewPDFMergerUtility()
	if err := m.merge(merger); err != nil {
		m.printlnErr(fmt.Sprintf("Error merging documents [%T]: %v", err, err))
		return 4
	}
	return ExitOK
}

// merge is the body of Java's try.
func (m *PDFMerger) merge(merger *multipdf.PDFMergerUtility) error {
	for _, infile := range m.infiles {
		if err := merger.AddSourceFile(infile); err != nil {
			return err
		}
	}
	merger.SetDestinationFileName(absolutePath(m.outfile))
	return merger.MergeDocuments()
}
