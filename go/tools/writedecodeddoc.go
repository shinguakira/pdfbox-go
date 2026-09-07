package tools

// Load document and write with all streams decoded.
//
// Port of org.apache.pdfbox.tools.WriteDecodedDoc.
//
// It is the only command that takes positional @Parameters rather than options:
// an input file at index 0, and an optional output file after it.

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// WriteDecodedDoc is the `writedecodeddoc` command.
type WriteDecodedDoc struct {
	streams
	mixinStandardHelpOptions

	password   string
	skipImages bool

	infile  string
	outfile string
}

var (
	_ Command    = (*WriteDecodedDoc)(nil)
	_ positional = (*WriteDecodedDoc)(nil)
)

// NewWriteDecodedDoc returns the command.
func NewWriteDecodedDoc() *WriteDecodedDoc { return &WriteDecodedDoc{} }

// Name is @Command(name = "writedecodeddoc").
func (w *WriteDecodedDoc) Name() string { return "writedecodeddoc" }

// Header is @Command(header = ...).
func (w *WriteDecodedDoc) Header() string { return "Writes a PDF document with all streams decoded" }

// Flags declares -password and -skipImages. The two files are positional.
func (w *WriteDecodedDoc) Flags(set *flag.FlagSet) {
	set.StringVar(&w.password, "password", "", "the password to decrypt the document")
	set.BoolVar(&w.skipImages, "skipImages", false, "don't uncompress images")
}

// setPositional takes `inputfile` at index 0 and the optional `outputfile`.
//
// picocli's arity on the first is the default 1, so leaving it out is a usage
// error, and on the second "0..1", so a third argument is one too.
func (w *WriteDecodedDoc) setPositional(args []string) error {
	switch len(args) {
	case 0:
		return &errMissingRequired{names: "<inputfile>"}
	case 1:
		w.infile = args[0]
		return nil
	case 2:
		w.infile, w.outfile = args[0], args[1]
		return nil
	default:
		return fmt.Errorf("Unmatched argument at index 2: '%s'", args[2])
	}
}

// Call reads the document, decodes every stream, and writes it back.
func (w *WriteDecodedDoc) Call() int {
	outputFilename := w.outfile
	if outputFilename == "" {
		outputFilename = calculateOutputFilename(absolutePath(w.infile))
	} else {
		outputFilename = absolutePath(outputFilename)
	}

	if err := w.doIt(absolutePath(w.infile), outputFilename, w.password, w.skipImages); err != nil {
		w.printlnErr("Error writing decoded PDF: " + err.Error())
		return 4
	}
	return ExitOK
}

// DoIt performs the document reading, decoding and writing.
//
// Port of the public doIt(String, String, String, boolean).
func (w *WriteDecodedDoc) DoIt(in, out, password string, skipImages bool) error {
	return w.doIt(in, out, password, skipImages)
}

func (w *WriteDecodedDoc) doIt(in, out, password string, skipImages bool) error {
	doc, err := pdfbox.LoadPDFWithPassword(in, password)
	if err != nil {
		return err
	}
	defer doc.Close()

	doc.SetAllSecurityToBeRemoved(true)
	cosDocument := doc.Document()
	for key := range cosDocument.XRefTable() {
		w.processObject(key, cosDocument.ObjectFromPool(key), skipImages)
	}
	doc.DocumentCatalog()
	doc.Document().SetIsXRefStream(false)
	return doc.SaveToFileOfParameters(out, compress.NoCompression)
}

// processObject decodes one stream in place.
//
// Java takes the COSObject and asks it for its key when it has to report a
// failure; the port is walking the cross-reference table, so it has the key
// already and is handed it.
func (w *WriteDecodedDoc) processObject(key *cos.ObjectKey, cosObject *cos.Object, skipImages bool) {
	if cosObject == nil {
		return
	}
	stream, isStream := cosObject.Object().(*cos.Stream)
	if !isStream {
		return
	}
	if skipImages && stream.GetItem(cos.Type) == cos.XObject &&
		stream.GetItem(cos.Subtype) == cos.Image {
		return
	}
	if err := decodeStreamInPlace(stream); err != nil {
		w.printlnErr("skip " + key.String() + " obj: " + err.Error())
	}
}

// decodeStreamInPlace reads the stream through its filters, drops /Filter, and
// writes the decoded bytes back.
func decodeStreamInPlace(stream *cos.Stream) error {
	data, err := common.NewPDStream(stream).ToByteArray()
	if err != nil {
		return err
	}
	stream.RemoveItem(cos.Filter)
	out, err := stream.CreateRawWriter()
	if err != nil {
		return err
	}
	if _, err := out.Write(data); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// calculateOutputFilename drops a trailing ".pdf", case-insensitively, and adds
// "_unc.pdf".
func calculateOutputFilename(filename string) string {
	outputFilename := filename
	if strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		outputFilename = filename[:len(filename)-4]
	}
	return outputFilename + "_unc.pdf"
}

// absolutePath is File.getAbsolutePath(), which Java calls on both files. It
// answers the path unchanged where it cannot be resolved, because Java's throws
// nothing here.
func absolutePath(path string) string {
	if absolute, err := filepath.Abs(path); err == nil {
		return absolute
	}
	return path
}
