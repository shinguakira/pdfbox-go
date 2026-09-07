package tools

// This program will just save the loaded pdf without any changes. As PDFBox
// doesn't support writing compressed object streams those streams are stripped
// and will be gone in the resulting file. This is very helpful when trying to
// debug problems as it'll make it possible to easily look through a PDF using a
// text editor. It also exposes problems which stem from objects inside object
// streams overwriting other objects.
//
// Port of org.apache.pdfbox.tools.DecompressObjectstreams.

import (
	"flag"
	"fmt"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
)

// DecompressObjectstreams is the `DecompressObjectstreams` command.
//
// It is one of the two commands that is **not** mixinStandardHelpOptions: it
// declares -h/--help itself with usageHelp = true, and has no -V.
type DecompressObjectstreams struct {
	streams
	mixinUsageHelpOption

	infile  string
	outfile string
}

var _ Command = (*DecompressObjectstreams)(nil)

// NewDecompressObjectstreams returns the command.
func NewDecompressObjectstreams() *DecompressObjectstreams { return &DecompressObjectstreams{} }

// Name is @Command(name = "DecompressObjectstreams"), which keeps its capitals
// where every other command is lower case.
func (d *DecompressObjectstreams) Name() string { return "DecompressObjectstreams" }

// Header is @Command(header = ...).
func (d *DecompressObjectstreams) Header() string {
	return "Decompresses object streams in a PDF file."
}

// Flags declares -i/--input and -o/--output.
func (d *DecompressObjectstreams) Flags(set *flag.FlagSet) {
	set.StringVar(&d.infile, "i", "", "the PDF file to decompress")
	set.StringVar(&d.infile, "input", "", "the PDF file to decompress")
	set.StringVar(&d.outfile, "o", "",
		"the decompressed PDF file. If omitted the original file is overwritten.")
	set.StringVar(&d.outfile, "output", "",
		"the decompressed PDF file. If omitted the original file is overwritten.")
}

// validate is `required = true` on -i.
func (d *DecompressObjectstreams) validate(set *flag.FlagSet) error {
	return requireSet(set, "--input=<infile>", "i", "input")
}

// Call loads the document and saves it uncompressed.
func (d *DecompressObjectstreams) Call() int {
	doc, err := pdfbox.LoadPDF(d.infile)
	if err != nil {
		return d.reportIOError(err)
	}
	defer doc.Close()

	// overwrite inputfile if no outputfile was specified
	if d.outfile == "" {
		d.outfile = d.infile
	}

	if err := doc.SaveToFileOfParameters(d.outfile, compress.NoCompression); err != nil {
		return d.reportIOError(err)
	}
	return ExitOK
}

// reportIOError is the catch(IOException) every command of this module ends
// with, which prints the exception's simple name and message and answers 4.
//
// Go has no class name for an error, so the port prints the error itself where
// Java prints "[" + simpleName + "]: " + message. The exit code, which is the
// part a caller can act on, is Java's.
func (d *DecompressObjectstreams) reportIOError(err error) int {
	fmt.Fprintln(d.err(), "Error processing file: "+err.Error())
	return 4
}
