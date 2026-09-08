package tools

// Extracts the images from a PDF file.
//
// Port of org.apache.pdfbox.tools.ExtractImages. It was one of the nine
// commands `track/tools` could not build, and the only one of them that was
// waiting for `tools/imageio` rather than for a rasteriser: it never imports
// `rendering`. It walks the content stream with `PDFGraphicsStreamEngine`,
// which slice 9 ported, and writes what it finds.

import (
	"flag"
	"fmt"
	goimage "image"
	gocolor "image/color"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	colorpr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/color"
	graphicspr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/graphics"
	markedcontentpr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/markedcontent"
	statepr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/state"
	textpr "github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/text"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	pdfont "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/tools/imageio"
)

// jpegFilters is Java's `private static final List<String> JPEG`, the filter
// names a JPEG stream can carry.
var jpegFilters = []string{
	cos.DCTDecode.Name(),
	cos.DCT.Name(),
}

// ExtractImages extracts the images from a PDF file.
type ExtractImages struct {
	streams
	mixinStandardHelpOptions

	password       string
	prefix         string
	useDirectJPEG  bool
	noColorConvert bool
	infile         string

	// seen is the set of image streams already written, so that an image used
	// twice on a page is written once.
	seen map[*cos.Stream]bool

	// imageCounter numbers the files, from one.
	imageCounter int
}

var _ Command = (*ExtractImages)(nil)

// NewExtractImages returns the command.
func NewExtractImages() *ExtractImages {
	return &ExtractImages{seen: map[*cos.Stream]bool{}, imageCounter: 1}
}

// Name is @Command(name = "export:images").
func (e *ExtractImages) Name() string { return "export:images" }

// Header is @Command(header = ...).
func (e *ExtractImages) Header() string { return "Extracts the images from a PDF file." }

// Flags declares the options.
func (e *ExtractImages) Flags(set *flag.FlagSet) {

	set.StringVar(&e.password, "password", "",
		"the password for the PDF or certificate in keystore.")
	set.StringVar(&e.prefix, "prefix", "", "the image prefix (default to pdf name).")
	set.BoolVar(&e.useDirectJPEG, "useDirectJPEG", false,
		"Forces the direct extraction of JPEG/JPX images regardless of colorspace or masking.")
	set.BoolVar(&e.noColorConvert, "noColorConvert", false,
		"Images are extracted with their original colorspace if possible.")
	set.StringVar(&e.infile, "i", "", "the PDF file")
	set.StringVar(&e.infile, "input", "", "the PDF file")
}

// Call runs the command.
func (e *ExtractImages) Call() int {
	if e.infile == "" {
		e.printlnErr("Error: Missing required option: '--input=<infile>'")
		return ExitUsage
	}

	document, err := pdfbox.LoadPDFWithPassword(e.infile, e.password)
	if err != nil {
		e.reportExtractError(err)
		return 4
	}
	defer document.Close()

	if !document.CurrentAccessPermission().CanExtractContent() {
		e.printlnErr("You do not have permission to extract images")
		return 1
	}

	if e.prefix == "" {
		absolute, err := filepath.Abs(e.infile)
		if err != nil {
			absolute = e.infile
		}
		e.prefix = strings.TrimSuffix(absolute, filepath.Ext(absolute))
	}

	for page := range document.Pages().All {
		engine := newImageGraphicsEngine(e, page)
		if err := engine.run(); err != nil {
			e.reportExtractError(err)
			return 4
		}
	}
	return ExitOK
}

// reportExtractError is Java's catch, which names the exception class.
func (e *ExtractImages) reportExtractError(err error) {
	e.printlnErr(fmt.Sprintf("Error extracting images [%T]: %v", err, err))
}

// imageGraphicsEngine walks a page for the images it draws.
//
// Port of the inner class ImageGraphicsEngine. Every path operator is empty in
// Java -- "add special handling if needed" -- except the three that fill, which
// look at the colour in case it is a tiling pattern with images inside it.
type imageGraphicsEngine struct {
	*contentstream.PDFGraphicsStreamEngine

	command *ExtractImages
}

var _ contentstream.GraphicsStreamEngineOverrides = (*imageGraphicsEngine)(nil)

func newImageGraphicsEngine(command *ExtractImages,
	page *pdmodel.PDPage) *imageGraphicsEngine {
	engine := &imageGraphicsEngine{
		PDFGraphicsStreamEngine: contentstream.NewPDFGraphicsStreamEngine(page),
		command:                 command,
	}
	engine.SetOverrides(engine)
	addAllOperators(engine.PDFGraphicsStreamEngine)
	return engine
}

// addAllOperators registers every operator PDFGraphicsStreamEngine's
// constructor registers.
//
// Java names all sixty in that constructor, so an `ImageGraphicsEngine` gets
// them by calling `super(page)`. The port's constructor registers none: every
// processor holds the engine, so the operator packages import `contentstream`
// and it cannot import them back, and the concrete engine registers them
// instead. This is that list, and it is the same one `rendering` has.
func addAllOperators(engine *contentstream.PDFGraphicsStreamEngine) {
	statepr.AddAll(engine.PDFStreamEngine)
	textpr.AddAll(engine.PDFStreamEngine)
	markedcontentpr.AddSequenceOperators(engine.PDFStreamEngine)
	colorpr.AddAll(engine.PDFStreamEngine)
	graphicspr.AddAll(engine)
}

// run processes the page and then the soft masks of its graphics states.
func (e *imageGraphicsEngine) run() error {
	page := e.Page()
	if err := e.ProcessPage(page); err != nil {
		return err
	}
	resources := page.Resources()
	if resources == nil {
		return nil
	}
	for _, name := range resources.ExtGStateNames() {
		extGState := resources.GetExtGState(name)
		if extGState == nil {
			// can happen if key exists but no value
			continue
		}
		softMask := extGState.SoftMask()
		if softMask == nil {
			continue
		}
		group, isGroup := softMask.Group().(*form.PDTransparencyGroup)
		if !isGroup || group == nil {
			continue
		}
		// PDFBOX-4327: without this line NPEs will occur
		extGState.CopyIntoGraphicsState(e.GraphicsState())
		if err := e.ProcessSoftMask(group); err != nil {
			return err
		}
	}
	return nil
}

// DrawImage writes the image out, once per stream.
func (e *imageGraphicsEngine) DrawImage(pdImage image.PDImage) error {
	if xobject, isXObject := pdImage.(*image.PDImageXObject); isXObject {
		if pdImage.IsStencil() {
			if err := e.processColor(e.GraphicsState().NonStrokingColor()); err != nil {
				return err
			}
		}
		stream, isStream := xobject.COSObject().(*cos.Stream)
		if isStream && e.command.seen[stream] {
			// skip duplicate image
			return nil
		}
		if isStream {
			e.command.seen[stream] = true
		}
	}

	// save image
	name := fmt.Sprintf("%s-%d", e.command.prefix, e.command.imageCounter)
	e.command.imageCounter++

	return e.command.write2file(pdImage, name, e.command.useDirectJPEG,
		e.command.noColorConvert)
}

// ShowGlyph processes the colour a glyph is painted in, which may be a tiling
// pattern with images inside it.
//
// Java overrides showGlyph and does not call super, so no glyph is drawn: the
// method is here for the colour and nothing else. `PDFGraphicsStreamEngine`
// leaves it alone, so without this a page whose *text* is filled with a
// patterned colour loses the images in that pattern.
func (e *imageGraphicsEngine) ShowGlyph(textRenderingMatrix *util.Matrix,
	f pdfont.PDFont, code int, displacement util.Vector) error {
	graphicsState := e.GraphicsState()
	renderingMode := graphicsState.TextState().RenderingMode()
	if renderingMode.IsFill() {
		if err := e.processColor(graphicsState.NonStrokingColor()); err != nil {
			return err
		}
	}
	if renderingMode.IsStroke() {
		if err := e.processColor(graphicsState.StrokingColor()); err != nil {
			return err
		}
	}
	return nil
}

// processColor finds out if it is a tiling pattern, then processes that one.
func (e *imageGraphicsEngine) processColor(pdColor *color.PDColor) error {
	if pdColor == nil {
		return nil
	}
	patternSpace, isPattern := pdColor.ColorSpace().(*pattern.PDPattern)
	if !isPattern {
		return nil
	}
	abstractPattern, err := patternSpace.Pattern(pdColor)
	if err != nil || abstractPattern == nil {
		return err
	}
	if tiling, isTiling := abstractPattern.(*pattern.PDTilingPattern); isTiling {
		return e.ProcessTilingPattern(tiling, nil, nil)
	}
	return nil
}

// The path operators, every one of which Java leaves empty except the three
// that fill.

func (e *imageGraphicsEngine) AppendRectangle(p0, p1, p2, p3 geom.Point2D) error { return nil }
func (e *imageGraphicsEngine) Clip(windingRule int) error                        { return nil }
func (e *imageGraphicsEngine) MoveTo(x, y float32) error                         { return nil }
func (e *imageGraphicsEngine) LineTo(x, y float32) error                         { return nil }
func (e *imageGraphicsEngine) CurveTo(x1, y1, x2, y2, x3, y3 float32) error      { return nil }
func (e *imageGraphicsEngine) ClosePath() error                                  { return nil }
func (e *imageGraphicsEngine) EndPath() error                                    { return nil }
func (e *imageGraphicsEngine) ShadingFill(shadingName *cos.Name) error           { return nil }

// CurrentPoint answers the origin, which is what Java's `new Point2D.Float()`
// is.
func (e *imageGraphicsEngine) CurrentPoint() (geom.Point2D, error) {
	return geom.NewPointFloat(0, 0), nil
}

func (e *imageGraphicsEngine) StrokePath() error {
	return e.processColor(e.GraphicsState().StrokingColor())
}

func (e *imageGraphicsEngine) FillPath(windingRule int) error {
	return e.processColor(e.GraphicsState().NonStrokingColor())
}

func (e *imageGraphicsEngine) FillAndStrokePath(windingRule int) error {
	return e.processColor(e.GraphicsState().NonStrokingColor())
}

// write2file writes the image to a file with the filename prefix and an
// appropriate suffix, which is set from the image's compression in the PDF.
func (e *ExtractImages) write2file(pdImage image.PDImage, prefix string,
	directJPEG, noColorConvert bool) error {
	suffix := pdImage.Suffix()
	switch suffix {
	case "", "jb2":
		suffix = "png"
	case "jpx":
		// use jp2 suffix for file because jpx not known by windows
		suffix = "jp2"
	}

	hasMasks, err := e.hasMasks(pdImage)
	if err != nil {
		return err
	}
	if hasMasks {
		// TIKA-3040, PDFBOX-4771: can't save ARGB as JPEG
		suffix = "png"
	}

	if noColorConvert {
		// We write the raw image if in any way possible.
		// But we have no alpha information here.
		raw, err := pdImage.RawImage()
		if err != nil {
			return err
		}
		if raw != nil {
			suffix = "png"
			if channelsOf(raw) > 3 {
				// More than 3 channels: That's likely CMYK. We use tiff here.
				suffix = "tiff"
			}
			return e.writeTo(prefix+"."+suffix, func(output io.Writer) error {
				_, err := imageio.WriteImage(raw, suffix, output)
				return err
			})
		}
	}

	return e.writeTo(prefix+"."+suffix, func(output io.Writer) error {
		return e.writeImageBody(pdImage, suffix, directJPEG, output)
	})
}

// writeImageBody is the body of Java's try-with-resources over the output
// stream: which of the four ways to write this image.
func (e *ExtractImages) writeImageBody(pdImage image.PDImage, suffix string,
	directJPEG bool, output io.Writer) error {
	colorSpace, err := pdImage.ColorSpace()
	if err != nil {
		return err
	}
	isGrayOrRGB := colorSpace != nil &&
		(colorSpace.Name() == color.DeviceGray.Name() ||
			colorSpace.Name() == color.DeviceRGB.Name())

	switch suffix {
	case "jpg":
		if directJPEG || isGrayOrRGB {
			// RGB or Gray colorspace: get and write the unmodified JPEG stream
			return copyEncodedStream(pdImage, jpegFilters, output)
		}
		// for CMYK and other "unusual" colorspaces, the JPEG will be converted
		return e.convertAndWrite(pdImage, suffix, output)

	case "jp2":
		if directJPEG || isGrayOrRGB {
			// RGB or Gray colorspace: get and write the unmodified JPEG2000
			// stream
			return copyEncodedStream(pdImage, []string{cos.JPXDecode.Name()}, output)
		}
		// for CMYK and other "unusual" colorspaces, the image will be converted
		//
		// Java asks ImageIOUtil for a "jpeg2000" writer, which is there only
		// with the JAI Image I/O Tools on the class path and is not there here
		// either: Go has no JPEG 2000 encoder and this port has no JPX support
		// at all; see migration/STATUS.md. The call is made anyway, because
		// what Java does when the writer is missing is log two lines and answer
		// false, leaving the file it has already created empty -- and that is
		// what this does.
		return e.convertAndWriteAs(pdImage, "jpeg2000", output)

	case "tiff":
		if colorSpace == color.DeviceGray {
			decoded, err := pdImage.Image()
			if err != nil {
				return err
			}
			if decoded == nil {
				return nil
			}
			// CCITT compressed images can have a different colorspace, but this
			// one is B/W. This is a bitonal image, so copy it to one bit per
			// pixel, which is what makes a compressed TIFF possible.
			_, err = imageio.WriteImage(asBitonal(decoded), suffix, output)
			return err
		}
		return e.convertAndWrite(pdImage, suffix, output)
	}

	return e.convertAndWrite(pdImage, suffix, output)
}

// convertAndWrite decodes the image and writes it in the format its suffix
// names.
func (e *ExtractImages) convertAndWrite(pdImage image.PDImage, suffix string,
	output io.Writer) error {
	return e.convertAndWriteAs(pdImage, suffix, output)
}

// convertAndWriteAs is the same with the format named separately, which the
// JPEG 2000 arm needs: its suffix is "jp2" and the format it asks for is
// "jpeg2000".
func (e *ExtractImages) convertAndWriteAs(pdImage image.PDImage, formatName string,
	output io.Writer) error {
	decoded, err := pdImage.Image()
	if err != nil {
		return err
	}
	if decoded == nil {
		return nil
	}
	_, err = imageio.WriteImage(decoded, formatName, output)
	return err
}

// writeTo creates the file, announces it the way Java does, and hands the
// writer to the caller.
func (e *ExtractImages) writeTo(path string, write func(io.Writer) error) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	e.printlnOut("Writing image: " + path)
	return write(file)
}

// hasMasks reports whether the image carries a mask or a soft mask, which is
// what stops it being written as a JPEG.
func (e *ExtractImages) hasMasks(pdImage image.PDImage) (bool, error) {
	xobject, isXObject := pdImage.(*image.PDImageXObject)
	if !isXObject {
		return false, nil
	}
	return xobject.Mask() != nil || xobject.SoftMask() != nil, nil
}

// channelsOf answers how many channels a raw image carries, which is Java's
// `image.getRaster().getNumDataElements()` -- the bands of the raster the
// colour space wrapped, not the channels of a Go pixel type.
//
// It is asked for one thing only: whether there are more than three, which
// means the image is likely CMYK and has to go into a TIFF rather than a PNG.
// A three-band RGB raster is three, and the alpha of a Go `image.RGBA` is not
// a fourth: "we have no alpha information here", says the caller.
//
// Only the first arm can be reached today. `PDColorSpace.ToRawImage` answers
// nil in every implementation this port has but `PDDeviceGray`'s and the
// `PDSeparation` that delegates to it, each of which answers an `image.Gray`;
// the rest are recorded in migration/STATUS.md as deferred for want of an ICC
// engine or of an indexed colour model. So `noColorConvert` writes a PNG or
// writes nothing, and the TIFF arm waits on those.
func channelsOf(img goimage.Image) int {
	switch img.(type) {
	case *goimage.Gray, *goimage.Gray16, *goimage.Alpha:
		return 1
	case *goimage.CMYK:
		return 4
	}
	return 3
}

// copyEncodedStream writes the image's stream out with the given filters left
// on it, which is Java's `pdImage.createInputStream(JPEG)` copied to the file.
//
// It is what makes `-useDirectJPEG` direct: the bytes in the PDF are the bytes
// in the file, with nothing decoded and nothing encoded again.
func copyEncodedStream(pdImage image.PDImage, stopFilters []string,
	output io.Writer) error {
	reader, err := pdImage.CreateInputStreamStopping(stopFilters)
	if err != nil {
		return err
	}
	if closer, isCloser := reader.(io.Closer); isCloser {
		defer closer.Close()
	}
	_, err = io.Copy(output, reader)
	return err
}

// asBitonal copies an image to one bit per pixel.
//
// Port of the loop Java writes out by hand -- "copy image the old-fashioned way
// - ColorConvertOp is slower!" -- into a TYPE_BYTE_BINARY image, which it does
// so that a compressed TIFF is written. This port's TIFF writer is not
// compressed, and the depth still matters: see migration/STATUS.md.
func asBitonal(img goimage.Image) goimage.Image {
	bounds := img.Bounds()
	out := goimage.NewGray(goimage.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			value := byte(0)
			if isWhitePixel(img, bounds.Min.X+x, bounds.Min.Y+y) {
				value = 0xFF
			}
			out.SetGray(x, y, gocolor.Gray{Y: value})
		}
	}
	return out
}

// isWhitePixel reports whether a pixel is nearer white than black.
func isWhitePixel(img goimage.Image, x, y int) bool {
	grey := gocolor.GrayModel.Convert(img.At(x, y)).(gocolor.Gray)
	return grey.Y >= 128
}
