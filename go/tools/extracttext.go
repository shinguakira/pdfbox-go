package tools

// This is the main program that simply parses the pdf document and transforms
// it into text.
//
// Port of org.apache.pdfbox.tools.ExtractText, and of the three package-private
// classes declared beside it: AngleCollector, FilteredTextStripper,
// FilteredText2Markdown and NullWriter.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// stdEncoding is Java's STD_ENCODING.
const stdEncoding = "UTF-8"

// ExtractText is the `extracttext` command.
type ExtractText struct {
	streams
	mixinStandardHelpOptions

	alwaysNext    bool
	toConsole     bool
	debug         bool
	encoding      string
	endPage       int
	toHTML        bool
	toMD          bool
	ignoreBeads   bool
	password      string
	rotationMagic bool
	sort          bool
	startPage     int
	infile        string
	outfile       string
	addFileName   bool
	append        bool
}

var _ Command = (*ExtractText)(nil)

// NewExtractText returns the command with its option defaults.
func NewExtractText() *ExtractText {
	return &ExtractText{encoding: stdEncoding, endPage: math.MaxInt32, startPage: 1}
}

// Name is @Command(name = "extracttext").
func (e *ExtractText) Name() string { return "extracttext" }

// Header is @Command(header = ...).
func (e *ExtractText) Header() string { return "Extracts the text from a PDF document" }

// Flags declares the seventeen options, in the order the Java declares them.
func (e *ExtractText) Flags(set *flag.FlagSet) {
	set.BoolVar(&e.alwaysNext, "alwaysNext", false,
		"Process next page (if applicable) despite IOException (ignored when -html)")
	set.BoolVar(&e.toConsole, "console", false, "Send text to console instead of file")
	set.BoolVar(&e.debug, "debug", false,
		"Enables debug output about the time consumption of every stage")
	set.StringVar(&e.encoding, "encoding", stdEncoding,
		"UTF-8 or ISO-8859-1, UTF-16BE, UTF-16LE, etc. (default: UTF-8)")
	set.IntVar(&e.endPage, "endPage", math.MaxInt32,
		"The last page to extract (1 based, inclusive)")
	set.BoolVar(&e.toHTML, "html", false, "Output in HTML format instead of raw text")
	set.BoolVar(&e.toMD, "md", false, "Output in Markdown format instead of raw text")
	set.BoolVar(&e.ignoreBeads, "ignoreBeads", false, "Disables the separation by beads")
	set.StringVar(&e.password, "password", "",
		"the password for the PDF or certificate in keystore.")
	set.BoolVar(&e.rotationMagic, "rotationMagic", false,
		"Analyze each page for rotated/skewed text, rotate to 0° and extract separately "+
			"(slower, and ignored when -html)")
	set.BoolVar(&e.sort, "sort", false, "Sort the text before writing of every stage")
	set.IntVar(&e.startPage, "startPage", 1, "The first page to start extraction (1 based)")
	set.StringVar(&e.infile, "i", "", "the PDF file")
	set.StringVar(&e.infile, "input", "", "the PDF file")
	set.StringVar(&e.outfile, "o", "", "the exported text file")
	set.StringVar(&e.outfile, "output", "", "the exported text file")
	set.BoolVar(&e.addFileName, "addFileName", false, "Print PDF file name to the output text")
	set.BoolVar(&e.append, "append", false, "Use append mode for output file")
}

// validate is `required = true` on -i.
func (e *ExtractText) validate(set *flag.FlagSet) error {
	return requireSet(set, "--input=<infile>", "i", "input")
}

// Call starts the text extraction.
func (e *ExtractText) Call() int {
	// set file extension
	if e.toHTML && e.toMD {
		e.printlnErr("You can't set md and html at the same time")
		return 1
	}
	ext := ".txt"
	if e.toHTML {
		ext = ".html"
	}
	if e.toMD {
		ext = ".md"
	}

	if e.outfile == "" {
		e.outfile = removeExtension(absolutePath(e.infile)) + ext
	}

	if e.toHTML && e.encoding != stdEncoding {
		e.encoding = stdEncoding
		e.printlnOut("The encoding parameter is ignored when writing html output.")
	}

	if e.toConsole && e.encoding != "" {
		e.printlnOut("The encoding parameter is ignored when writing to the console.")
	}

	if err := e.extract(); err != nil {
		if errors.Is(err, errNoExtractPermission) {
			// Java returns 1 from inside the try, having already printed the
			// message; the sentinel carries that back out past the two
			// resources without printing twice.
			return 1
		}
		e.printlnErr("Error extracting text for document: " + err.Error())
		return 4
	}
	return ExitOK
}

// extract is the body of the try-with-resources, so that the two resources are
// released on every path.
func (e *ExtractText) extract() error {
	document, err := pdfbox.LoadPDFWithPassword(e.infile, e.password)
	if err != nil {
		return err
	}
	defer document.Close()

	output, closeOutput, err := e.createOutputWriter()
	if err != nil {
		return err
	}
	defer closeOutput()

	startTime := e.startProcessing("Loading PDF " + e.infile)

	if !document.CurrentAccessPermission().CanExtractContent() {
		e.printlnErr("You do not have permission to extract text")
		// Java returns 1 from call() here, before the writer is closed by the
		// try-with-resources. The port raises a sentinel the caller turns back
		// into 1, so that the two resources are still released.
		return errNoExtractPermission
	}

	e.stopProcessing("Time for loading: ", startTime)
	startTime = e.startProcessing("Starting text extraction")

	if e.addFileName {
		if _, err := io.WriteString(output, "PDF file: "+e.infile); err != nil {
			return err
		}
		if _, err := io.WriteString(output, lineSeparator()); err != nil {
			return err
		}
	}

	if e.debug {
		e.printlnErr("Writing to " + absolutePath(e.outfile))
	}

	stripper, writeText := e.newStripper()

	if e.toHTML {
		// HTML stripper can't work page by page because of startDocument()
		// callback
		stripper.SetSortByPosition(e.sort)
		stripper.SetShouldSeparateByBeads(!e.ignoreBeads)
		stripper.SetStartPage(e.startPage)
		stripper.SetEndPage(e.endPage)

		// Extract text for main document:
		if err := writeText(document, output); err != nil {
			return err
		}
	} else {
		stripper.SetSortByPosition(e.sort)
		stripper.SetShouldSeparateByBeads(!e.ignoreBeads)

		// Extract text for main document:
		if err := e.extractPages(e.startPage, min(e.endPage, document.NumberOfPages()),
			stripper, writeText, document, output); err != nil {
			return err
		}
	}

	// ... also for any embedded PDFs:
	if err := e.extractEmbedded(document, stripper, writeText, output); err != nil {
		return err
	}

	e.stopProcessing("Time for extraction: ", startTime)
	return nil
}

// errNoExtractPermission is the one failure Call answers 1 for rather than 4.
var errNoExtractPermission = errors.New("tools: no permission to extract text")

// writeTextFunc is a stripper's writeText, which differs by which stripper it
// belongs to.
type writeTextFunc func(document *pdmodel.PDDocument, output io.Writer) error

// newStripper builds the stripper the options ask for, and the writeText that
// reaches it.
func (e *ExtractText) newStripper() (*text.PDFTextStripper, writeTextFunc) {
	switch {
	case e.toHTML:
		html := NewPDFText2HTML()
		return html.PDFTextStripper, html.WriteText
	case e.toMD && e.rotationMagic:
		filtered := newFilteredText2Markdown()
		return filtered.PDFTextStripper, filtered.WriteText
	case e.toMD:
		md := NewPDFText2Markdown()
		return md.PDFTextStripper, md.WriteText
	case e.rotationMagic:
		filtered := newFilteredTextStripper()
		return filtered.PDFTextStripper, filtered.WriteText
	default:
		plain := text.NewPDFTextStripper()
		return plain, plain.WriteText
	}
}

// extractEmbedded writes the text of every embedded PDF the document carries.
func (e *ExtractText) extractEmbedded(document *pdmodel.PDDocument,
	stripper *text.PDFTextStripper, writeText writeTextFunc, output io.Writer) error {
	names := document.DocumentCatalog().Names()
	if names == nil {
		return nil
	}
	embeddedFiles := names.EmbeddedFiles()
	if embeddedFiles == nil {
		return nil
	}
	embeddedFileNames, err := embeddedFiles.Names()
	if err != nil {
		return err
	}
	if embeddedFileNames == nil {
		return nil
	}

	// Java walks a Map<String, ...>, whose order is the tree's; the port sorts
	// so that two runs of the same document write the same bytes.
	keys := make([]string, 0, len(embeddedFileNames))
	for key := range embeddedFileNames {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if e.debug {
			e.printlnErr("Processing embedded file " + key + ":")
		}
		spec := embeddedFileNames[key]
		file := spec.EmbeddedFile()
		if file == nil || file.Subtype() != "application/pdf" {
			continue
		}
		if e.debug {
			e.printlnErr("  is PDF (size=" + itoa(file.Size()) + ")")
		}
		if err := e.extractOneEmbedded(file, stripper, writeText, output); err != nil {
			return err
		}
	}
	return nil
}

// extractOneEmbedded is the inner try-with-resources of the embedded walk.
func (e *ExtractText) extractOneEmbedded(file embeddedPDF, stripper *text.PDFTextStripper,
	writeText writeTextFunc, output io.Writer) error {
	input, err := file.CreateInputStream()
	if err != nil {
		return err
	}
	data, err := io.ReadAll(input)
	if err != nil {
		return err
	}
	subDoc, err := pdfbox.LoadPDFFrom(pdfio.NewReadBufferBytes(data))
	if err != nil {
		return err
	}
	defer subDoc.Close()

	if e.toHTML {
		// will not really work because of HTML header + footer
		return writeText(subDoc, output)
	}
	return e.extractPages(1, subDoc.NumberOfPages(), stripper, writeText, subDoc, output)
}

// embeddedPDF is what the walk needs of a PDEmbeddedFile.
type embeddedPDF interface {
	CreateInputStream() (io.Reader, error)
}

// extractPages writes one page at a time, so that a failure on one page does
// not have to end the run.
func (e *ExtractText) extractPages(startPage, endPage int, stripper *text.PDFTextStripper,
	writeText writeTextFunc, document *pdmodel.PDDocument, output io.Writer) error {
	for p := startPage; p <= endPage; p++ {
		stripper.SetStartPage(p)
		stripper.SetEndPage(p)
		err := e.extractOnePage(p, stripper, writeText, document, output)
		if err != nil {
			if !e.alwaysNext {
				return err
			}
			slog.Error("tools: failed to process page", "page", p, "err", err)
		}
	}
	return nil
}

// extractOnePage is the body of the try in extractPages.
func (e *ExtractText) extractOnePage(p int, stripper *text.PDFTextStripper,
	writeText writeTextFunc, document *pdmodel.PDDocument, output io.Writer) error {
	if !e.rotationMagic {
		return writeText(document, output)
	}

	page := document.Page(p - 1)
	rotation := page.Rotation()
	page.SetRotation(0)

	angleCollector := newAngleCollector()
	angleCollector.SetStartPage(p)
	angleCollector.SetEndPage(p)
	if err := angleCollector.WriteText(document, nullWriter{}); err != nil {
		return err
	}
	// rotation magic
	for _, angle := range angleCollector.Angles() {
		// prepend a transformation
		// (we could skip these parts for angle 0, but it doesn't matter much)
		if err := prependRotation(document, page, angle); err != nil {
			return err
		}

		if err := writeText(document, output); err != nil {
			return err
		}

		// remove prepended transformation
		page.Dictionary().GetCOSArray(cos.Contents).RemoveAt(0)
	}
	page.SetRotation(rotation)
	return nil
}

// prependRotation is the try-with-resources that writes the transformation.
func prependRotation(document *pdmodel.PDDocument, page *pdmodel.PDPage, angle int) error {
	cs, err := pdmodel.NewPDPageContentStreamOfMode(document, page, pdmodel.Prepend, false, true)
	if err != nil {
		return err
	}
	if err := cs.Transform(util.RotateInstance(-radians(float64(angle)), 0, 0)); err != nil {
		cs.Close()
		return err
	}
	return cs.Close()
}

// createOutputWriter answers the writer the text goes to, and the close that
// releases it.
func (e *ExtractText) createOutputWriter() (io.Writer, func(), error) {
	if e.toConsole {
		// Java wraps System.out in a PrintWriter whose close() does nothing, so
		// that the console survives the try-with-resources.
		return e.out(), func() {}, nil
	}
	flags := os.O_WRONLY | os.O_CREATE
	if e.append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(e.outfile, flags, 0o666)
	if err != nil {
		return nil, nil, err
	}
	writer, err := encodingWriter(file, e.encoding)
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	return writer, func() { file.Close() }, nil
}

// startProcessing prints the message where -debug was given, and answers when
// it started.
func (e *ExtractText) startProcessing(message string) int64 {
	if e.debug {
		e.printlnErr(message)
	}
	return nowMillis()
}

// stopProcessing prints how long the stage took, where -debug was given.
func (e *ExtractText) stopProcessing(message string, startTime int64) {
	if e.debug {
		elapsedTime := float32(nowMillis()-startTime) / 1000
		e.printlnErr(message + formatFloat(elapsedTime) + " seconds")
	}
}

// getAngle is the static ExtractText.getAngle.
func getAngle(t *text.TextPosition) int {
	// should this become a part of TextPosition?
	m := t.TextMatrix().Clone()
	m.Concatenate(t.Font().FontMatrix())
	return int(javaRoundDouble(degrees(math.Atan2(float64(m.ShearY()), float64(m.ScaleY())))))
}

// angleCollector collects all angles while doing text extraction. Angles are in
// degrees and rounded to the closest integer (to avoid slight differences from
// floating point arithmetic resulting in similarly angled glyphs being treated
// separately). This class must be constructed for each page so that the angle
// set is initialized.
//
// Port of the package-private class AngleCollector.
type angleCollector struct {
	*text.PDFTextStripper

	// angles is Java's TreeSet<Integer>: sorted, and each angle once.
	angles map[int]bool
}

func newAngleCollector() *angleCollector {
	c := &angleCollector{PDFTextStripper: text.NewPDFTextStripper(), angles: map[int]bool{}}
	c.SetProcessTextPosition(c.ProcessTextPosition)
	return c
}

// Angles returns the angles seen, in order, which is what a TreeSet gives.
func (c *angleCollector) Angles() []int {
	angles := make([]int, 0, len(c.angles))
	for angle := range c.angles {
		angles = append(angles, angle)
	}
	sort.Ints(angles)
	return angles
}

// ProcessTextPosition records the angle of one glyph.
func (c *angleCollector) ProcessTextPosition(t *text.TextPosition) error {
	angle := getAngle(t)
	angle = (angle + 360) % 360
	c.angles[angle] = true
	return nil
}

// filteredTextStripper is a TextStripper that only processes glyphs that have
// angle 0.
type filteredTextStripper struct {
	*text.PDFTextStripper
}

func newFilteredTextStripper() *filteredTextStripper {
	s := &filteredTextStripper{PDFTextStripper: text.NewPDFTextStripper()}
	s.SetProcessTextPosition(s.processTextPosition)
	return s
}

func (s *filteredTextStripper) processTextPosition(t *text.TextPosition) error {
	if getAngle(t) == 0 {
		return s.PDFTextStripper.ProcessTextPosition(t)
	}
	return nil
}

// filteredText2Markdown is a PDFText2Markdown that only processes glyphs that
// have angle 0.
type filteredText2Markdown struct {
	*PDFText2Markdown
}

func newFilteredText2Markdown() *filteredText2Markdown {
	s := &filteredText2Markdown{PDFText2Markdown: NewPDFText2Markdown()}
	s.SetProcessTextPosition(s.processTextPosition)
	return s
}

func (s *filteredText2Markdown) processTextPosition(t *text.TextPosition) error {
	if getAngle(t) == 0 {
		return s.PDFTextStripper.ProcessTextPosition(t)
	}
	return nil
}

// nullWriter is Java's NullWriter: dummy output.
type nullWriter struct{}

func (nullWriter) Write(p []byte) (int, error) { return len(p), nil }

// removeExtension is commons-io FilenameUtils.removeExtension, which drops
// everything from the last dot of the last path element.
func removeExtension(filename string) string {
	base := filepath.Base(filename)
	dot := strings.LastIndex(base, ".")
	if dot < 0 {
		return filename
	}
	return filename[:len(filename)-(len(base)-dot)]
}

// lineSeparator is System.lineSeparator(), which -addFileName writes after the
// file name.
func lineSeparator() string { return text.LineSeparatorDefault }

// itoa is Integer.toString.
func itoa(value int) string { return strconv.Itoa(value) }

// formatFloat is what Java's string concatenation does with a float.
func formatFloat(value float32) string {
	return strconv.FormatFloat(float64(value), 'g', -1, 32)
}

// nowMillis is System.currentTimeMillis().
func nowMillis() int64 { return time.Now().UnixMilli() }

// radians is Math.toRadians.
func radians(degrees float64) float64 { return degrees * math.Pi / 180 }

// degrees is Math.toDegrees.
func degrees(radians float64) float64 { return radians * 180 / math.Pi }

// javaRoundDouble is Math.round(double): the closest long, ties going towards
// positive infinity, where Go's math.Round takes a tie away from zero.
func javaRoundDouble(value float64) int64 { return int64(math.Floor(value + 0.5)) }

// encodingWriter wraps the file in whatever the -encoding option asked for.
//
// Java hands the charset to an OutputStreamWriter, which encodes every write.
// The port supports the encodings the Java help names and refuses the rest by
// name rather than writing the wrong bytes silently.
func encodingWriter(file io.Writer, encoding string) (io.Writer, error) {
	switch strings.ToUpper(strings.ReplaceAll(encoding, "_", "-")) {
	case "", "UTF-8", "UTF8":
		return file, nil
	case "ISO-8859-1", "LATIN1", "LATIN-1":
		return charmapWriter{file, charmap.ISO8859_1}, nil
	case "UTF-16BE":
		return utf16Writer{file, unicode.BigEndian}, nil
	case "UTF-16LE":
		return utf16Writer{file, unicode.LittleEndian}, nil
	}
	return nil, fmt.Errorf("unsupported encoding: %s", encoding)
}

// charmapWriter encodes to a single-byte charset.
type charmapWriter struct {
	out io.Writer
	cm  *charmap.Charmap
}

func (w charmapWriter) Write(p []byte) (int, error) {
	encoded, err := w.cm.NewEncoder().Bytes(p)
	if err != nil {
		return 0, err
	}
	if _, err := w.out.Write(encoded); err != nil {
		return 0, err
	}
	return len(p), nil
}

// utf16Writer encodes to UTF-16 in the given byte order, with no byte order
// mark, which is what Java's "UTF-16BE" and "UTF-16LE" charsets do.
type utf16Writer struct {
	out       io.Writer
	byteOrder unicode.Endianness
}

func (w utf16Writer) Write(p []byte) (int, error) {
	encoder := unicode.UTF16(w.byteOrder, unicode.IgnoreBOM).NewEncoder()
	encoded, err := encoder.Bytes(p)
	if err != nil {
		return 0, err
	}
	if _, err := w.out.Write(encoded); err != nil {
		return 0, err
	}
	return len(p), nil
}
