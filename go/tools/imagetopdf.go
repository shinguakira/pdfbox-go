package tools

// Creates a PDF document from images.
//
// Port of org.apache.pdfbox.tools.ImageToPDF.
//
// B8 asked how much of this is raster before starting, and the answer is none:
// it reads image files through PDImageXObject.createFromFile, which slice 6
// ported, and draws them with PDPageContentStream. Nothing here needs
// rendering.Backend -- that is ExtractImages, which writes images *out* through
// ImageIOUtil.

import (
	"flag"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
)

// ImageToPDF is the `imagetopdf` command.
type ImageToPDF struct {
	streams
	mixinStandardHelpOptions

	mediaBox *common.PDRectangle

	autoOrientation bool
	landscape       bool
	pageSize        string
	resize          bool
	infiles         repeatedFiles
	outfile         string
}

var _ Command = (*ImageToPDF)(nil)

// NewImageToPDF returns the command with its option defaults.
func NewImageToPDF() *ImageToPDF {
	return &ImageToPDF{mediaBox: common.Letter, pageSize: "Letter"}
}

// Name is @Command(name = "imagetopdf").
func (i *ImageToPDF) Name() string { return "imagetopdf" }

// Header is @Command(header = ...).
func (i *ImageToPDF) Header() string { return "Creates a PDF document from images" }

// Flags declares the six options.
func (i *ImageToPDF) Flags(set *flag.FlagSet) {
	set.BoolVar(&i.autoOrientation, "autoOrientation", false,
		"set orientation depending of image proportion")
	set.BoolVar(&i.landscape, "landscape", false, "set orientation to landscape")
	set.StringVar(&i.pageSize, "pageSize", "Letter",
		"the page size to use: Letter, Legal, A0, A1, A2, A3, A4, A5, A6 (default: Letter)")
	set.BoolVar(&i.resize, "resize", false, "resize to page size")
	// Java's -i is a File[]: picocli takes it more than once, or with several
	// values after it. `flag` takes it more than once, or comma-separated.
	set.Var(&i.infiles, "i", "the image files to convert")
	set.Var(&i.infiles, "input", "the image files to convert")
	set.StringVar(&i.outfile, "o", "", "the generated PDF file")
	set.StringVar(&i.outfile, "output", "", "the generated PDF file")
}

func (i *ImageToPDF) validate(set *flag.FlagSet) error {
	if err := requireSet(set, "--input=<image-file>", "i", "input"); err != nil {
		return err
	}
	return requireSet(set, "--output=<outfile>", "o", "output")
}

// Call writes one page per image.
func (i *ImageToPDF) Call() int {
	i.mediaBox = createRectangle(i.pageSize)

	if err := i.convert(); err != nil {
		i.printlnErr("Error converting image to PDF: " + err.Error())
		return 4
	}
	return ExitOK
}

// convert is the body of the try-with-resources.
func (i *ImageToPDF) convert() error {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()

	for _, imageFile := range i.infiles {
		pdImage, err := image.CreateFromFile(doc, absolutePath(imageFile))
		if err != nil {
			return err
		}
		actualMediaBox := i.mediaBox
		if (i.autoOrientation && pdImage.Width() > pdImage.Height()) || i.landscape {
			actualMediaBox = common.NewPDRectangleOfSize(
				i.mediaBox.Height(), i.mediaBox.Width())
		}
		page := pdmodel.NewPDPageOfSize(actualMediaBox)
		doc.AddPage(page)

		if err := i.drawOnePage(doc, page, pdImage, actualMediaBox); err != nil {
			return err
		}
	}
	return doc.SaveToFile(i.outfile)
}

// drawOnePage is the inner try-with-resources.
func (i *ImageToPDF) drawOnePage(doc *pdmodel.PDDocument, page *pdmodel.PDPage,
	pdImage *image.PDImageXObject, actualMediaBox *common.PDRectangle) error {
	contents, err := pdmodel.NewPDPageContentStream(doc, page)
	if err != nil {
		return err
	}
	if i.resize {
		err = contents.DrawImageSized(pdImage, 0, 0,
			actualMediaBox.Width(), actualMediaBox.Height())
	} else {
		err = contents.DrawImageSized(pdImage, 0, 0,
			float32(pdImage.Width()), float32(pdImage.Height()))
	}
	if err != nil {
		contents.Close()
		return err
	}
	return contents.Close()
}

// createRectangle answers the page size the name asks for, defaulting to
// Letter -- which is also what an unknown name answers, with Java's comment.
func createRectangle(paperSize string) *common.PDRectangle {
	switch {
	case strings.EqualFold(paperSize, "letter"):
		return common.Letter
	case strings.EqualFold(paperSize, "legal"):
		return common.Legal
	case strings.EqualFold(paperSize, "A0"):
		return common.A0
	case strings.EqualFold(paperSize, "A1"):
		return common.A1
	case strings.EqualFold(paperSize, "A2"):
		return common.A2
	case strings.EqualFold(paperSize, "A3"):
		return common.A3
	case strings.EqualFold(paperSize, "A4"):
		return common.A4
	case strings.EqualFold(paperSize, "A5"):
		return common.A5
	case strings.EqualFold(paperSize, "A6"):
		return common.A6
	default:
		// return default if wron size was specified
		return common.Letter
	}
}
