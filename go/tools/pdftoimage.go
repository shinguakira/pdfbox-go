package tools

// Converts a PDF document to image(s).
//
// Port of org.apache.pdfbox.tools.PDFToImage. It was one of the nine commands
// `track/tools` could not build, and one of the two that were waiting for a
// rendering.Backend rather than for `tools/imageio`. `track/raster` brings
// one, so this is written against rendering/raster.

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
	"github.com/shinguakira/pdfbox-go/go/tools/imageio"
)

// defaultDPI is what Java falls back to where there is no screen to ask.
//
//	dpi = Toolkit.getDefaultToolkit().getScreenResolution();
//	catch (HeadlessException e) { dpi = 96; }
//
// There is no toolkit here and no screen resolution to read, so the fallback
// is the only arm: a command-line tool on a machine with no display would take
// it in Java too.
const defaultDPI = 96

// PDFToImage converts a PDF document to image(s).
type PDFToImage struct {
	streams
	mixinStandardHelpOptions

	password    string
	imageFormat string
	outputPrefix
	page        int
	startPage   int
	endPage     int
	color       string
	dpi         int
	quality     float64
	cropbox     string
	showTime    bool
	subsampling bool
	infile      string
}

// outputPrefix is the -prefix/-outputPrefix pair, which Java declares as one
// option with two names.
type outputPrefix struct {
	prefix string
}

var _ Command = (*PDFToImage)(nil)

// NewPDFToImage returns the command.
func NewPDFToImage() *PDFToImage { return &PDFToImage{} }

// Name is @Command(name = "render").
//
// Java's own name is "pdftoimage"; the dispatcher renames it, and
// notbuilt.go's row has carried the new name since `track/tools`.
func (p *PDFToImage) Name() string { return "render" }

// Header is @Command(header = ...).
func (p *PDFToImage) Header() string { return "Converts a PDF document to image(s)" }

// Flags declares the options.
func (p *PDFToImage) Flags(set *flag.FlagSet) {
	set.StringVar(&p.password, "password", "", "the password to decrypt the document")
	set.StringVar(&p.imageFormat, "format", "jpg", "the image file format (default: jpg)")
	set.StringVar(&p.prefix, "prefix", "", "the filename prefix for image files")
	set.StringVar(&p.prefix, "outputPrefix", "", "the filename prefix for image files")
	set.IntVar(&p.page, "page", -1, "the only page to extract (1-based)")
	set.IntVar(&p.startPage, "startPage", 1, "the first page to start extraction (1-based)")
	set.IntVar(&p.endPage, "endPage", maxPageNumber, "the last page to extract (inclusive)")
	set.StringVar(&p.color, "color", "RGB",
		"the color depth (valid: BILEVEL, GRAY, RGB, ARGB) (default: RGB)")
	set.IntVar(&p.dpi, "dpi", 0,
		"the DPI of the output image, default: screen resolution or 96 if unknown")
	set.IntVar(&p.dpi, "resolution", 0,
		"the DPI of the output image, default: screen resolution or 96 if unknown")
	set.Float64Var(&p.quality, "quality", -1,
		"the quality to be used when compressing the image (0 <= quality <= 1) "+
			"(default: 0 for PNG and 1 for the other formats)")
	set.StringVar(&p.cropbox, "cropbox", "", "the page area to export, as four numbers")
	set.BoolVar(&p.showTime, "time", false, "print timing information to stdout")
	set.BoolVar(&p.subsampling, "subsampling", false,
		"activate subsampling (for PDFs with huge images)")
	set.StringVar(&p.infile, "i", "", "the PDF files to convert.")
	set.StringVar(&p.infile, "input", "", "the PDF files to convert.")
}

// maxPageNumber is Java's `Integer.MAX_VALUE` default for -endPage.
const maxPageNumber = 1<<31 - 1

// Call runs the command.
func (p *PDFToImage) Call() int {
	if p.infile == "" {
		p.printlnErr("Error: Missing required option: '--input=<infile>'")
		return ExitUsage
	}
	if p.prefix == "" {
		absolute, err := filepath.Abs(p.infile)
		if err != nil {
			absolute = p.infile
		}
		p.prefix = strings.TrimSuffix(absolute, filepath.Ext(absolute))
	}

	if !imageio.CanWrite(p.imageFormat) {
		p.printlnErr("Error: Invalid image format " + p.imageFormat +
			" - supported formats: " + strings.Join(imageio.WriterFormatNames(), ", "))
		return 2
	}

	imageType, known := imageTypeOf(p.color)
	if !known {
		// picocli rejects an unknown enum value before call() ever runs, with
		// a usage error; the port's flag package cannot, so the check is here.
		p.printlnErr("Error: Invalid color " + p.color +
			" - valid values: BILEVEL, GRAY, RGB, ARGB")
		return ExitUsage
	}

	if p.quality < 0 {
		if p.imageFormat == "png" {
			p.quality = 0
		} else {
			p.quality = 1
		}
	}
	if p.dpi == 0 {
		p.dpi = defaultDPI
	}

	cropbox, ok := p.cropboxOf()
	if !ok {
		p.printlnErr("Error: -cropbox takes four numbers")
		return ExitUsage
	}

	document, err := pdfbox.LoadPDFWithPassword(p.infile, p.password)
	if err != nil {
		p.reportConvertError(err)
		return 4
	}
	defer document.Close()

	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())
	if acroForm != nil && acroForm.NeedAppearances() {
		if err := acroForm.RefreshAppearances(); err != nil {
			p.reportConvertError(err)
			return 4
		}
	}
	if cropbox != nil {
		changeCropBox(document, cropbox)
	}

	return p.render(document, imageType)
}

// render is the loop and the timing report.
func (p *PDFToImage) render(document *pdmodel.PDDocument, imageType rendering.ImageType) int {
	startTime := time.Now()

	// render the pages
	if p.page != -1 {
		p.startPage = p.page
		p.endPage = p.page
	}
	success := true
	if p.endPage > document.NumberOfPages() {
		p.endPage = document.NumberOfPages()
	}
	for i := p.startPage - 1; i < p.endPage; i++ {
		image, err := raster.RenderPageWithDPI(document, i, float32(p.dpi),
			imageType, p.subsampling)
		if err != nil {
			p.reportConvertError(err)
			return 4
		}
		fileName := fmt.Sprintf("%s-%d.%s", p.prefix, i+1, p.imageFormat)
		written, err := imageio.WriteImageToFileOfQuality(image, fileName, p.dpi,
			float32(p.quality))
		if err != nil {
			p.reportConvertError(err)
			return 4
		}
		success = success && written
	}

	// performance stats
	duration := time.Since(startTime)
	count := 1 + p.endPage - p.startPage
	if p.showTime {
		plural := "s"
		if count == 1 {
			plural = ""
		}
		p.printlnErr(fmt.Sprintf("Rendered %d page%s in %dms", count, plural,
			duration.Milliseconds()))
	}

	if !success {
		p.printlnErr("Error: no writer found for image format '" + p.imageFormat + "'")
		return 1
	}
	return ExitOK
}

// reportConvertError is Java's catch, which names the exception class.
func (p *PDFToImage) reportConvertError(err error) {
	p.printlnErr(fmt.Sprintf("Error converting document [%T]: %v", err, err))
}

// cropboxOf reads the four numbers of -cropbox, and answers nil where the
// option was not given.
//
// picocli parses `arity="4"` into an int[]; the port's flag package takes one
// string, so the four are split here.
func (p *PDFToImage) cropboxOf() (*common.PDRectangle, bool) {
	if p.cropbox == "" {
		return nil, true
	}
	fields := strings.FieldsFunc(p.cropbox, func(r rune) bool {
		return r == ' ' || r == ','
	})
	if len(fields) != 4 {
		return nil, false
	}
	var numbers [4]float32
	for i, field := range fields {
		var value float32
		if _, err := fmt.Sscanf(field, "%g", &value); err != nil {
			return nil, false
		}
		numbers[i] = value
	}
	rectangle := common.NewPDRectangle()
	rectangle.SetLowerLeftX(numbers[0])
	rectangle.SetLowerLeftY(numbers[1])
	rectangle.SetUpperRightX(numbers[2])
	rectangle.SetUpperRightY(numbers[3])
	return rectangle, true
}

// changeCropBox is the private static of the same name: every page gets the
// same crop box.
func changeCropBox(document *pdmodel.PDDocument, cropbox *common.PDRectangle) {
	for page := range document.Pages().All {
		rectangle := common.NewPDRectangle()
		rectangle.SetLowerLeftX(cropbox.LowerLeftX())
		rectangle.SetLowerLeftY(cropbox.LowerLeftY())
		rectangle.SetUpperRightX(cropbox.UpperRightX())
		rectangle.SetUpperRightY(cropbox.UpperRightY())
		page.SetCropBox(rectangle)
	}
}

// imageTypeOf is picocli's enum conversion for -color.
func imageTypeOf(name string) (rendering.ImageType, bool) {
	switch strings.ToUpper(name) {
	case "BILEVEL":
		return rendering.Binary, true
	case "GRAY":
		return rendering.Gray, true
	case "RGB":
		return rendering.RGB, true
	case "ARGB":
		return rendering.ARGB, true
	}
	return rendering.RGB, false
}
