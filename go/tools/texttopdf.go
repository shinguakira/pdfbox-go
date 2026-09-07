package tools

// This will take a text file and output a pdf with some formatting.
//
// Port of org.apache.pdfbox.tools.TextToPDF.

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

const (
	fontScale               = 1000
	defaultFontSize         = float32(10)
	defaultLineHeightFactor = float32(1.05)
	defaultMargin           = float32(40)
)

// pageSizes is the private nested enum PageSizes.
var pageSizes = map[string]*common.PDRectangle{
	"LETTER": common.Letter,
	"LEGAL":  common.Legal,
	"A0":     common.A0,
	"A1":     common.A1,
	"A2":     common.A2,
	"A3":     common.A3,
	"A4":     common.A4,
	"A5":     common.A5,
	"A6":     common.A6,
}

// TextToPDF is the `texttopdf` command.
type TextToPDF struct {
	streams
	mixinStandardHelpOptions

	mediaBox *common.PDRectangle
	font     font.PDFont

	fontSize     float64
	lineSpacing  float64
	landscape    bool
	pageSize     string
	charset      string
	margins      marginList
	standardFont string
	ttf          string
	infile       string
	outfile      string

	leftMargin   float32
	rightMargin  float32
	topMargin    float32
	bottomMargin float32
}

var _ Command = (*TextToPDF)(nil)

// NewTextToPDF returns the command with its option defaults.
func NewTextToPDF() *TextToPDF {
	return &TextToPDF{
		mediaBox:     common.Letter,
		fontSize:     float64(defaultFontSize),
		lineSpacing:  float64(defaultLineHeightFactor),
		pageSize:     "LETTER",
		charset:      "UTF-8",
		margins:      marginList{defaultMargin, defaultMargin, defaultMargin, defaultMargin},
		standardFont: string(font.Helvetica),
		leftMargin:   defaultMargin,
		rightMargin:  defaultMargin,
		topMargin:    defaultMargin,
		bottomMargin: defaultMargin,
	}
}

// Name is @Command(name = "texttopdf").
func (t *TextToPDF) Name() string { return "texttopdf" }

// Header is @Command(header = ...).
func (t *TextToPDF) Header() string { return "Creates a PDF document from text" }

// Flags declares the ten options.
func (t *TextToPDF) Flags(set *flag.FlagSet) {
	set.Float64Var(&t.fontSize, "fontSize", float64(defaultFontSize),
		"the size of the font to use (default: 10)")
	set.Float64Var(&t.lineSpacing, "lineSpacing", float64(defaultLineHeightFactor),
		"the factor of the font size for the line height (default: 1.05)")
	set.BoolVar(&t.landscape, "landscape", false, "set orientation to landscape")
	set.StringVar(&t.pageSize, "pageSize", "LETTER",
		"the page size to use. Candidates: LETTER, LEGAL, A0..A6 (default: LETTER)")
	set.StringVar(&t.charset, "charset", "UTF-8", "the charset to use. (default: UTF-8)")
	set.Var(&t.margins, "margins", "Left Right Top Bottom margins (default: 40 40 40 40)")
	set.StringVar(&t.standardFont, "standardFont", string(font.Helvetica),
		"the font to use for the text. Either this or -ttf should be specified but not both.")
	set.StringVar(&t.ttf, "ttf", "",
		"the TTF font to use for the text. Either this or -standardFont should be "+
			"specified but not both.")
	set.StringVar(&t.infile, "i", "", "the text file to convert")
	set.StringVar(&t.infile, "input", "", "the text file to convert")
	set.StringVar(&t.outfile, "o", "", "the generated PDF file")
	set.StringVar(&t.outfile, "output", "", "the generated PDF file")
}

// validate is `required = true` on -i and -o.
func (t *TextToPDF) validate(set *flag.FlagSet) error {
	if err := requireSet(set, "--input=<infile>", "i", "input"); err != nil {
		return err
	}
	return requireSet(set, "--output=<outfile>", "o", "output")
}

// Call reads the text file and writes the PDF.
func (t *TextToPDF) Call() int {
	if err := t.convert(); err != nil {
		t.printlnErr("Error converting text to PDF: " + err.Error())
		return 4
	}
	return ExitOK
}

// convert is the body of the try-with-resources.
func (t *TextToPDF) convert() error {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()

	if t.ttf != "" {
		embedded, err := font.LoadPDType0FontFile(doc, t.ttf)
		if err != nil {
			return err
		}
		t.font = embedded
	} else {
		standard, err := font.NewPDType1FontStandard14(font.FontName(t.standardFont))
		if err != nil {
			return err
		}
		t.font = standard
	}
	// Java routes every option through its setter, and one of them validates.
	if err := t.SetLineSpacing(float32(t.lineSpacing)); err != nil {
		return err
	}
	size, ok := pageSizes[strings.ToUpper(t.pageSize)]
	if !ok {
		return fmt.Errorf("unknown page size: %s", t.pageSize)
	}
	t.mediaBox = size
	t.leftMargin = t.margins[0]
	t.rightMargin = t.margins[1]
	t.topMargin = t.margins[2]
	t.bottomMargin = t.margins[3]

	file, err := os.Open(t.infile)
	if err != nil {
		return err
	}
	defer file.Close()

	reader, err := t.decodingReader(file)
	if err != nil {
		return err
	}
	if err := t.CreatePDFFromText(doc, reader); err != nil {
		return err
	}
	return doc.SaveToFile(t.outfile)
}

// decodingReader applies -charset, and skips a UTF-8 byte order mark where
// there is one, which is what Java's mark/read/reset does.
func (t *TextToPDF) decodingReader(file io.Reader) (io.Reader, error) {
	buffered := bufio.NewReader(file)
	if strings.EqualFold(strings.ReplaceAll(t.charset, "_", "-"), "UTF-8") ||
		strings.EqualFold(t.charset, "UTF8") {
		const readLimit = 3
		firstBytes, err := buffered.Peek(readLimit)
		if err != nil && len(firstBytes) == readLimit {
			return nil, err
		}
		if len(firstBytes) == readLimit &&
			firstBytes[0] == 0xEF && firstBytes[1] == 0xBB && firstBytes[2] == 0xBF {
			if _, err := buffered.Discard(readLimit); err != nil {
				return nil, err
			}
		}
		return buffered, nil
	}
	return decodingReaderFor(buffered, t.charset)
}

// CreatePDFFromText lays the text out on pages of the document.
//
// Port of the public createPDFFromText(PDDocument, Reader).
func (t *TextToPDF) CreatePDFFromText(doc *pdmodel.PDDocument, text io.Reader) error {
	if t.font == nil {
		standard, err := font.NewPDType1FontStandard14(font.FontName(t.standardFont))
		if err != nil {
			return err
		}
		t.font = standard
	}
	box, err := t.font.BoundingBox()
	if err != nil {
		return err
	}
	fontHeight := box.Height() / fontScale
	actualMediaBox := t.mediaBox
	if t.landscape {
		actualMediaBox = common.NewPDRectangleOfSize(t.mediaBox.Height(), t.mediaBox.Width())
	}
	fontSize := float32(t.fontSize)
	lineHeight := fontHeight * fontSize * float32(t.lineSpacing)

	// BufferedReader.readLine splits on "\n", "\r" or "\r\n", drops the ending,
	// and has no limit on how long a line may be. bufio.Scanner would impose
	// one -- and bufio.ScanLines would leave a lone "\r" inside a line -- so the
	// port reads lines itself.
	data := newJavaLineReader(text)

	page := pdmodel.NewPDPageOfSize(actualMediaBox)
	var contentStream *pdmodel.PDPageContentStream
	y := float32(-1)
	maxStringLength := page.MediaBox().Width() - t.leftMargin - t.rightMargin
	textIsEmpty := true
	var nextLineToDraw strings.Builder

	newPage := func() error {
		page = pdmodel.NewPDPageOfSize(actualMediaBox)
		doc.AddPage(page)
		if contentStream != nil {
			if err := contentStream.EndText(); err != nil {
				return err
			}
			if err := contentStream.Close(); err != nil {
				return err
			}
		}
		stream, err := pdmodel.NewPDPageContentStream(doc, page)
		if err != nil {
			return err
		}
		contentStream = stream
		if err := contentStream.SetFont(t.font, fontSize); err != nil {
			return err
		}
		if err := contentStream.BeginText(); err != nil {
			return err
		}
		y = page.MediaBox().Height() - t.topMargin
		y += lineHeight - fontHeight*fontSize // adjust for lineSpacing != 1
		return contentStream.NewLineAtOffset(t.leftMargin, y)
	}

	for {
		nextLine, more, err := data.ReadLine()
		if err != nil {
			return err
		}
		if !more {
			break
		}
		textIsEmpty = false
		// Java's split(" ", -1) keeps every empty, leading and trailing.
		lineWords := strings.Split(nextLine, " ")
		lineIndex := 0
		for lineIndex < len(lineWords) {
			nextLineToDraw.Reset()
			addSpace := false
			lengthIfUsingNextWord := float32(0)
			ff := false
			for {
				var word1, word2 string
				word := lineWords[lineIndex]
				indexFF := strings.IndexByte(word, '\f')
				if indexFF == -1 {
					word1 = word
				} else {
					ff = true
					word1 = word[:indexFF]
					word2 = word[indexFF+1:]
				}
				if word1 != "" || !ff {
					if addSpace {
						nextLineToDraw.WriteByte(' ')
					} else {
						addSpace = true
					}
					nextLineToDraw.WriteString(word1)
				}
				if !ff || word2 == "" {
					lineIndex++
				} else {
					lineWords[lineIndex] = word2
				}
				if ff {
					break
				}
				if lineIndex < len(lineWords) {
					nextWord := lineWords[lineIndex]
					if indexFF = strings.IndexByte(nextWord, '\f'); indexFF != -1 {
						nextWord = nextWord[:indexFF]
					}
					lineWithNextWord := nextLineToDraw.String() + " " + nextWord
					width, err := t.font.StringWidth(lineWithNextWord)
					if err != nil {
						return err
					}
					lengthIfUsingNextWord = (width / fontScale) * fontSize
				}
				if !(lineIndex < len(lineWords) && lengthIfUsingNextWord < maxStringLength) {
					break
				}
			}

			if y-lineHeight < t.bottomMargin {
				if err := newPage(); err != nil {
					return err
				}
			}
			if contentStream == nil {
				return fmt.Errorf("Error:Expected non-null content stream.")
			}
			if err := contentStream.NewLineAtOffset(0, -lineHeight); err != nil {
				return err
			}
			y -= lineHeight
			if err := contentStream.ShowText(nextLineToDraw.String()); err != nil {
				return err
			}
			if ff {
				if err := newPage(); err != nil {
					return err
				}
			}
		}
	}
	if textIsEmpty {
		doc.AddPage(page)
	}
	if contentStream != nil {
		if err := contentStream.EndText(); err != nil {
			return err
		}
		return contentStream.Close()
	}
	return nil
}

// SetFont sets the font the text is drawn with.
func (t *TextToPDF) SetFont(aFont font.PDFont) { t.font = aFont }

// Font returns the font the text is drawn with.
func (t *TextToPDF) Font() font.PDFont { return t.font }

// marginList is picocli's `arity = "0..4"` float array: up to four numbers
// after one option. `flag` gives one value per option, so the port takes them
// comma-separated as well as repeated, and says so in the help.
type marginList [4]float32

func (m *marginList) String() string {
	parts := make([]string, len(m))
	for i, value := range m {
		parts[i] = strconv.FormatFloat(float64(value), 'g', -1, 32)
	}
	return strings.Join(parts, ",")
}

func (m *marginList) Set(value string) error {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })
	if len(fields) == 0 || len(fields) > 4 {
		return fmt.Errorf("margins takes one to four numbers, not %d", len(fields))
	}
	for i, field := range fields {
		parsed, err := strconv.ParseFloat(field, 32)
		if err != nil {
			return err
		}
		m[i] = float32(parsed)
	}
	return nil
}

// scanJavaLines splits on "\n", "\r" or "\r\n" and drops the ending, which is
// BufferedReader.readLine. bufio.ScanLines drops only "\n" and "\r\n", so a
// file with old-style Mac endings would come back as one line.
func scanJavaLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\n' {
			return i + 1, data[:i], nil
		}
		if b == '\r' {
			if i+1 < len(data) {
				if data[i+1] == '\n' {
					return i + 2, data[:i], nil
				}
				return i + 1, data[:i], nil
			}
			if atEOF {
				return i + 1, data[:i], nil
			}
			// A trailing "\r" may still be the first half of "\r\n".
			return 0, nil, nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// decodingReaderFor applies a charset other than UTF-8 to the input, which is
// what Java's InputStreamReader does.
func decodingReaderFor(input io.Reader, charset string) (io.Reader, error) {
	switch strings.ToUpper(strings.ReplaceAll(charset, "_", "-")) {
	case "ISO-8859-1", "LATIN1", "LATIN-1":
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	case "UTF-16BE":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder().Reader(input), nil
	case "UTF-16LE":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder().Reader(input), nil
	case "UTF-16":
		return unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder().Reader(input), nil
	}
	return nil, fmt.Errorf("unsupported charset: %s", charset)
}

// SetMediaBox sets the page size the text is laid out on.
func (t *TextToPDF) SetMediaBox(mediaBox *common.PDRectangle) { t.mediaBox = mediaBox }

// SetLineSpacing sets the factor of the font size for the line height.
//
// Port of setLineSpacing(float), the one setter of this class that validates.
// Java throws IllegalArgumentException, which is unchecked, so the port panics
// -- and picocli answers ExitCode.SOFTWARE for it, which Execute does too.
func (t *TextToPDF) SetLineSpacing(lineSpacing float32) error {
	if lineSpacing <= 0 {
		panic("line spacing must be positive: " +
			strconv.FormatFloat(float64(lineSpacing), 'g', -1, 32))
	}
	t.lineSpacing = float64(lineSpacing)
	return nil
}

// javaLineReader is BufferedReader.readLine: it splits on "\n", "\r" or
// "\r\n", drops the ending, and has no limit on how long a line may be.
//
// bufio.Scanner cannot do the job twice over: its ScanLines leaves a lone "\r"
// inside a line, and every Scanner has a maximum token size, which readLine
// does not.
type javaLineReader struct{ input *bufio.Reader }

func newJavaLineReader(input io.Reader) *javaLineReader {
	return &javaLineReader{input: bufio.NewReader(input)}
}

// ReadLine answers the next line, and whether there was one. The line is
// returned without its ending, as readLine does.
func (r *javaLineReader) ReadLine() (string, bool, error) {
	var line strings.Builder
	sawAny := false
	for {
		b, err := r.input.ReadByte()
		if err == io.EOF {
			// readLine answers null only where nothing at all was read; a last
			// line with no ending is still a line.
			return line.String(), sawAny, nil
		}
		if err != nil {
			return "", false, err
		}
		sawAny = true
		switch b {
		case '\n':
			return line.String(), true, nil
		case '\r':
			// "\r\n" is one ending; a lone "\r" is another.
			next, err := r.input.ReadByte()
			if err == nil && next != '\n' {
				if unreadErr := r.input.UnreadByte(); unreadErr != nil {
					return "", false, unreadErr
				}
			} else if err != nil && err != io.EOF {
				return "", false, err
			}
			return line.String(), true, nil
		default:
			line.WriteByte(b)
		}
	}
}
