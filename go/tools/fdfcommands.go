package tools

// The four commands that move AcroForm data in and out of a document.
//
// Port of org.apache.pdfbox.tools.ExportFDF, ImportFDF, ExportXFDF and
// ImportXFDF, which are four small classes with one shape between them.

import (
	"flag"
	"os"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// ExportFDF is the `exportfdf` command.
type ExportFDF struct {
	streams
	mixinStandardHelpOptions

	infile  string
	outfile string
}

var _ Command = (*ExportFDF)(nil)

// NewExportFDF returns the command.
func NewExportFDF() *ExportFDF { return &ExportFDF{} }

// Name is @Command(name = "exportfdf").
func (e *ExportFDF) Name() string { return "exportfdf" }

// Header is @Command(header = ...).
func (e *ExportFDF) Header() string { return "Exports AcroForm form data to FDF" }

// Flags declares -i and -o, both of which Java marks required.
func (e *ExportFDF) Flags(set *flag.FlagSet) {
	set.StringVar(&e.infile, "i", "", "the PDF file to export")
	set.StringVar(&e.infile, "input", "", "the PDF file to export")
	set.StringVar(&e.outfile, "o", "", "the FDF data file")
	set.StringVar(&e.outfile, "output", "", "the FDF data file")
}

func (e *ExportFDF) validate(set *flag.FlagSet) error {
	if err := requireSet(set, "--input=<infile>", "i", "input"); err != nil {
		return err
	}
	return requireSet(set, "--output=<outfile>", "o", "output")
}

// Call exports the form data.
func (e *ExportFDF) Call() int {
	return exportForm(&e.streams, e.infile, &e.outfile, ".fdf", "FDF",
		func(document *fdf.FDFDocument, out *os.File) error { return document.Save(out) })
}

// ExportXFDF is the `exportxfdf` command.
type ExportXFDF struct {
	streams
	mixinStandardHelpOptions

	infile  string
	outfile string
}

var _ Command = (*ExportXFDF)(nil)

// NewExportXFDF returns the command.
func NewExportXFDF() *ExportXFDF { return &ExportXFDF{} }

// Name is @Command(name = "exportxfdf").
func (e *ExportXFDF) Name() string { return "exportxfdf" }

// Header is @Command(header = ...).
func (e *ExportXFDF) Header() string { return "Exports AcroForm form data to XFDF" }

// Flags declares -i and -o.
func (e *ExportXFDF) Flags(set *flag.FlagSet) {
	set.StringVar(&e.infile, "i", "", "the PDF file to export")
	set.StringVar(&e.infile, "input", "", "the PDF file to export")
	set.StringVar(&e.outfile, "o", "", "the XFDF data file")
	set.StringVar(&e.outfile, "output", "", "the XFDF data file")
}

func (e *ExportXFDF) validate(set *flag.FlagSet) error {
	if err := requireSet(set, "--input=<infile>", "i", "input"); err != nil {
		return err
	}
	return requireSet(set, "--output=<outfile>", "o", "output")
}

// Call exports the form data as XFDF.
func (e *ExportXFDF) Call() int {
	return exportForm(&e.streams, e.infile, &e.outfile, ".xfdf", "XFDF",
		func(document *fdf.FDFDocument, out *os.File) error { return document.SaveXFDF(out) })
}

// exportForm is the body ExportFDF and ExportXFDF share.
func exportForm(s *streams, infile string, outfile *string, extension, what string,
	save func(*fdf.FDFDocument, *os.File) error) int {
	document, err := pdfbox.LoadPDF(infile)
	if err != nil {
		s.printlnErr("Error exporting " + what + " data: " + err.Error())
		return 4
	}
	defer document.Close()

	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())
	if acroForm == nil {
		s.printlnErr("Error: This PDF does not contain a form.")
		// ExportXFDF falls out of this branch in the Java and reaches the
		// `return 0` at the end of the method, so it reports the error and
		// exits successfully without writing a file; ExportFDF, which is the
		// same class twice over, answers 1. See migration/JAVA-BUGS.md 75.
		return 1
	}

	if *outfile == "" {
		*outfile = removeExtension(absolutePath(infile)) + extension
	}

	exported, err := form.ExportFDFDocument(acroForm)
	if err != nil {
		s.printlnErr("Error exporting " + what + " data: " + err.Error())
		return 4
	}
	defer exported.Close()

	out, err := os.Create(*outfile)
	if err != nil {
		s.printlnErr("Error exporting " + what + " data: " + err.Error())
		return 4
	}
	if err := save(exported, out); err != nil {
		out.Close()
		s.printlnErr("Error exporting " + what + " data: " + err.Error())
		return 4
	}
	if err := out.Close(); err != nil {
		s.printlnErr("Error exporting " + what + " data: " + err.Error())
		return 4
	}
	return ExitOK
}

// ImportFDF is the `importfdf` command.
type ImportFDF struct {
	streams
	mixinStandardHelpOptions

	infile  string
	outfile string
	fdffile string
}

var _ Command = (*ImportFDF)(nil)

// NewImportFDF returns the command.
func NewImportFDF() *ImportFDF { return &ImportFDF{} }

// Name is @Command(name = "importfdf").
func (i *ImportFDF) Name() string { return "importfdf" }

// Header is @Command(header = ...).
func (i *ImportFDF) Header() string { return "Imports AcroForm form data from FDF" }

// Flags declares -i, -o and --data.
func (i *ImportFDF) Flags(set *flag.FlagSet) {
	set.StringVar(&i.infile, "i", "", "the PDF file to import to")
	set.StringVar(&i.infile, "input", "", "the PDF file to import to")
	set.StringVar(&i.outfile, "o", "",
		"the PDF file to save to. If omitted the original file will be used")
	set.StringVar(&i.outfile, "output", "",
		"the PDF file to save to. If omitted the original file will be used")
	// Java declares only the long name for this one.
	set.StringVar(&i.fdffile, "data", "", "the FDF data file to import from")
}

func (i *ImportFDF) validate(set *flag.FlagSet) error {
	if err := requireSet(set, "--input=<infile>", "i", "input"); err != nil {
		return err
	}
	return requireSet(set, "--data=<fdffile>", "data")
}

// ImportFDFInto imports the FDF document into the PDF document.
//
// Port of the public importFDF(PDDocument, FDFDocument). It sets
// /NeedAppearances, which ImportXFDF's does not.
func (i *ImportFDF) ImportFDFInto(pdfDocument *pdmodel.PDDocument,
	fdfDocument *fdf.FDFDocument) error {
	acroForm := form.AcroFormOfCatalog(pdfDocument.DocumentCatalog())
	if acroForm == nil {
		return nil
	}
	acroForm.SetCacheFields(true)
	if err := form.ImportFDFDocument(acroForm, fdfDocument); err != nil {
		return err
	}
	acroForm.SetNeedAppearances(true)
	return nil
}

// Call imports the form data.
func (i *ImportFDF) Call() int {
	return importForm(&i.streams, i.infile, i.outfile, i.fdffile, "FDF",
		pdfbox.LoadFDF, i.ImportFDFInto)
}

// ImportXFDF is the `importxfdf` command.
type ImportXFDF struct {
	streams
	mixinStandardHelpOptions

	infile   string
	outfile  string
	xfdffile string
}

var _ Command = (*ImportXFDF)(nil)

// NewImportXFDF returns the command.
func NewImportXFDF() *ImportXFDF { return &ImportXFDF{} }

// Name is @Command(name = "importxfdf").
func (i *ImportXFDF) Name() string { return "importxfdf" }

// Header is @Command(header = ...).
func (i *ImportXFDF) Header() string { return "Imports AcroForm form data from XFDF" }

// Flags declares -i, -o and --data.
func (i *ImportXFDF) Flags(set *flag.FlagSet) {
	set.StringVar(&i.infile, "i", "", "the PDF file to import to")
	set.StringVar(&i.infile, "input", "", "the PDF file to import to")
	set.StringVar(&i.outfile, "o", "",
		"the PDF file to save to. If omitted the original file will be used")
	set.StringVar(&i.outfile, "output", "",
		"the PDF file to save to. If omitted the original file will be used")
	set.StringVar(&i.xfdffile, "data", "", "the XFDF data file to import from")
}

func (i *ImportXFDF) validate(set *flag.FlagSet) error {
	if err := requireSet(set, "--input=<infile>", "i", "input"); err != nil {
		return err
	}
	return requireSet(set, "--data=<xfdffile>", "data")
}

// ImportFDFInto imports the FDF document into the PDF document.
//
// Port of ImportXFDF's own importFDF(PDDocument, FDFDocument), which differs
// from ImportFDF's twice: it does not set /NeedAppearances, and it does not
// check for a null form before calling setCacheFields on it, so a document with
// no form raises NullPointerException. See migration/JAVA-BUGS.md.
func (i *ImportXFDF) ImportFDFInto(pdfDocument *pdmodel.PDDocument,
	fdfDocument *fdf.FDFDocument) error {
	acroForm := form.AcroFormOfCatalog(pdfDocument.DocumentCatalog())
	// Java dereferences acroForm here with no null check; the port reaches the
	// same nil dereference and panics, which is what an unchecked exception is
	// in this port.
	acroForm.SetCacheFields(true)
	return form.ImportFDFDocument(acroForm, fdfDocument)
}

// Call imports the form data.
func (i *ImportXFDF) Call() int {
	return importForm(&i.streams, i.infile, i.outfile, i.xfdffile, "XFDF",
		pdfbox.LoadXFDF, i.ImportFDFInto)
}

// importForm is the body ImportFDF and ImportXFDF share.
func importForm(s *streams, infile, outfile, datafile, what string,
	loadData func(string) (*fdf.FDFDocument, error),
	importInto func(*pdmodel.PDDocument, *fdf.FDFDocument) error) int {
	document, err := pdfbox.LoadPDF(infile)
	if err != nil {
		s.printlnErr("Error importing " + what + " data: " + err.Error())
		return 4
	}
	defer document.Close()

	data, err := loadData(datafile)
	if err != nil {
		s.printlnErr("Error importing " + what + " data: " + err.Error())
		return 4
	}
	defer data.Close()

	if err := importInto(document, data); err != nil {
		s.printlnErr("Error importing " + what + " data: " + err.Error())
		return 4
	}
	if outfile == "" {
		outfile = infile
	}
	if err := document.SaveToFile(outfile); err != nil {
		s.printlnErr("Error importing " + what + " data: " + err.Error())
		return 4
	}
	return ExitOK
}
