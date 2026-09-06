package handlers

import (
	"log/slog"
	"math"
	"regexp"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// colorPattern matches the colour of a default style string.
//
// Port of the private static COLOR_PATTERN of PDFreeTextAppearanceHandler. Java
// writes the quantifier as a possessive \s*+, which Go has no spelling for; a
// greedy \s* matches the same strings here, because what follows it -- a # and
// six hex digits -- can never match a space, so there is nothing to backtrack
// into.
var colorPattern = regexp.MustCompile(`color:\s*#([0-9a-fA-F]{6})`)

// AcroFormDefaultAppearance answers the /DA of the form of the given document,
// and the empty string where the document has no form.
//
// Java reads it through document.getDocumentCatalog().getAcroForm(); this
// package cannot name PDAcroForm, because pdmodel blank-imports this one for the
// handler registry and interactive/form imports pdmodel. That package fills this
// from its init, so a program that wants the free text handler to read the form
// has to link it in -- importing it anywhere is enough. Without it the handler
// behaves as Java does for a document that has no form.
var AcroFormDefaultAppearance func(document common.COSDocumentLike) string

// AcroFormDefaultResourcesFont answers the font the default resources of the
// form of the given document give for the given name, and nil where there is
// none. See AcroFormDefaultAppearance.
var AcroFormDefaultResourcesFont func(document common.COSDocumentLike,
	fontName *cos.Name) font.PDFont

// PDFreeTextAppearanceHandler draws a free text annotation.
//
// Port of PDFreeTextAppearanceHandler.
type PDFreeTextAppearanceHandler struct {
	PDAbstractAppearanceHandler

	fontSize float32
	fontName *cos.Name
}

// NewPDFreeTextAppearanceHandler builds a handler for the given annotation.
func NewPDFreeTextAppearanceHandler(annot annotation.PDAnnotation) *PDFreeTextAppearanceHandler {
	return NewPDFreeTextAppearanceHandlerInDocument(annot, nil)
}

// NewPDFreeTextAppearanceHandlerInDocument builds one whose streams belong to
// the given document.
func NewPDFreeTextAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDFreeTextAppearanceHandler {
	h := &PDFreeTextAppearanceHandler{fontSize: 10, fontName: cos.Helv}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance draws the free text.
func (h *PDFreeTextAppearanceHandler) GenerateNormalAppearance() error {
	annot, isFreeText := h.Annotation().(*annotation.PDAnnotationFreeText)
	if !isFreeText {
		panic("handlers: the annotation of a free text handler is not a free text")
	}
	var pathsArray []float32
	if annot.Intent() == annotation.ITFreeTextCallout {
		pathsArray = annot.Callout()
		if pathsArray == nil || len(pathsArray) != 4 && len(pathsArray) != 6 {
			pathsArray = []float32{}
		}
	} else {
		pathsArray = []float32{}
	}
	ab := getAnnotationBorder(annot, annot.BorderStyle())

	cs, err := h.NormalAppearanceAsContentStreamCompressed(true)
	if err != nil {
		slog.Error("handlers: free text appearance", slog.Any("error", err))
		return nil
	}
	defer cs.Close()

	if err := h.drawFreeText(annot, cs, ab, pathsArray); err != nil {
		slog.Error("handlers: free text appearance", slog.Any("error", err))
	}
	return nil
}

// drawFreeText is the body of the try block Java writes.
func (h *PDFreeTextAppearanceHandler) drawFreeText(annot *annotation.PDAnnotationFreeText,
	cs annotation.AppearanceContentStream, ab *annotationBorder, pathsArray []float32) error {
	// The fill color is the /C entry, there is no /IC entry defined
	hasBackground, err := cs.SetNonStrokingColorOnDemand(annot.Color())
	if err != nil {
		return err
	}
	if err := h.SetOpacity(cs, annot.ConstantOpacity()); err != nil {
		return err
	}

	// Adobe uses the last non stroking color from /DA as stroking color!
	// But if there is a color in /DS, then that one is used for text.
	strokingColor := h.extractNonStrokingColor(annot)
	hasStroke, err := cs.SetStrokingColorOnDemand(strokingColor)
	if err != nil {
		return err
	}
	textColor := strokingColor
	defaultStyleString := annot.DefaultStyleString()
	if defaultStyleString != "" {
		if m := colorPattern.FindStringSubmatch(defaultStyleString); m != nil {
			value, err := strconv.ParseInt(m[1], 16, 64)
			if err != nil {
				panic("For input string: " + m[1])
			}
			c := int(value)
			r := float32((c>>16)&0xFF) / 255
			g := float32((c>>8)&0xFF) / 255
			b := float32(c&0xFF) / 255
			textColor = color.NewPDColorOfComponents([]float32{r, g, b}, color.DeviceRGB)
		}
	}

	if ab.dashArray != nil {
		if err := cs.SetLineDashPattern(ab.dashArray, 0); err != nil {
			return err
		}
	}
	if err := cs.SetLineWidth(ab.width); err != nil {
		return err
	}

	lineEndingStyle := annot.LineEndingStyle()

	// draw callout line(s)
	// must be done before retangle paint to avoid a line cutting through cloud
	// see CTAN-example-Annotations.pdf
	for i := 0; i < len(pathsArray)/2; i++ {
		x := pathsArray[i*2]
		y := pathsArray[i*2+1]
		if i == 0 {
			if shortStyles[lineEndingStyle] {
				// modify coordinate to shorten the segment
				// https://stackoverflow.com/questions/7740507/extend-a-line-segment-a-specific-distance
				x1 := pathsArray[2]
				y1 := pathsArray[3]
				length := float32(math.Sqrt(
					math.Pow(float64(x-x1), 2) + math.Pow(float64(y-y1), 2)))
				if length != 0 {
					x += (x1 - x) / length * ab.width
					y += (y1 - y) / length * ab.width
				}
			}
			if err := cs.MoveTo(x, y); err != nil {
				return err
			}
		} else {
			if err := cs.LineTo(x, y); err != nil {
				return err
			}
		}
	}
	if len(pathsArray) > 0 {
		if err := cs.Stroke(); err != nil {
			return err
		}
	}

	// paint the styles here and after line(s) draw, to avoid line crossing a filled shape
	if annot.Intent() == annotation.ITFreeTextCallout &&
		// check only needed to avoid q cm Q if LE_NONE
		lineEndingStyle != annotation.LENone && len(pathsArray) >= 4 {
		x2 := pathsArray[2]
		y2 := pathsArray[3]
		x1 := pathsArray[0]
		y1 := pathsArray[1]
		if err := cs.SaveGraphicsState(); err != nil {
			return err
		}
		if angledStyles[lineEndingStyle] {
			// do a transform so that first "arm" is imagined flat,
			// like in line handler.
			// The alternative would be to apply the transform to the
			// LE shape coordinates directly, which would be more work
			// and produce code difficult to understand
			angle := math.Atan2(float64(y2-y1), float64(x2-x1))
			if err := cs.Transform(util.RotateInstance(angle, x1, y1)); err != nil {
				return err
			}
		} else {
			if err := cs.Transform(util.TranslateInstance(x1, y1)); err != nil {
				return err
			}
		}
		if err := h.DrawStyle(lineEndingStyle, cs, 0, 0, ab.width,
			hasStroke, hasBackground, false); err != nil {
			return err
		}
		if err := cs.RestoreGraphicsState(); err != nil {
			return err
		}
	}

	var borderBox *common.PDRectangle
	borderEffect := annot.BorderEffect()
	if borderEffect != nil && borderEffect.Style() == annotation.BorderEffectStyleCloudy {
		// Adobe draws the text with the original rectangle in mind.
		// but if there is an /RD, then writing area get smaller.
		// do this here because /RD is overwritten in a few lines
		borderBox = h.ApplyRectDifferences(h.Rectangle(), annot.RectDifferences())

		//TODO this segment was copied from square handler. Refactor?
		cloudyBorder := newCloudyBorder(cs, float64(borderEffect.Intensity()),
			float64(ab.width), h.Rectangle())
		if err := cloudyBorder.createCloudyRectangle(annot.RectDifference()); err != nil {
			return err
		}
		annot.SetRectangle(cloudyBorder.rectangle())
		annot.SetRectDifference(cloudyBorder.rectDifference())
		appearanceStream := annot.NormalAppearanceStream()
		appearanceStream.SetBBox(cloudyBorder.bbox())
		appearanceStream.SetMatrix(cloudyBorder.matrix())
	} else {
		// handle the border box
		//
		// There are two options. The handling is not part of the PDF specification but
		// implementation specific to Adobe Reader
		// - if /RD is set the border box is the /Rect entry inset by the respective
		//   border difference.
		// - if /RD is not set then we don't touch /RD etc because Adobe doesn't either.
		borderBox = h.ApplyRectDifferences(h.Rectangle(), annot.RectDifferences())
		annot.NormalAppearanceStream().SetBBox(borderBox)

		// note that borderBox is not modified
		paddedRectangle := h.PaddedRectangle(borderBox, ab.width/2)
		if err := cs.AddRect(paddedRectangle.LowerLeftX(), paddedRectangle.LowerLeftY(),
			paddedRectangle.Width(), paddedRectangle.Height()); err != nil {
			return err
		}
	}
	if err := cs.DrawShape(ab.width, hasStroke, hasBackground); err != nil {
		return err
	}

	// rotation is an undocumented feature, but Adobe uses it. Examples can be found
	// in pdf_commenting_new.pdf file, page 3.
	rotation := annot.AnnotationDictionary().GetIntDefault(cos.Rotate, 0)
	if err := cs.Transform(util.RotateInstance(toRadians(float64(rotation)), 0, 0)); err != nil {
		return err
	}
	var xOffset float32
	var yOffset float32
	width := borderBox.Width()
	if rotation == 90 || rotation == 270 {
		width = borderBox.Height()
	}
	// strategy to write formatted text is somewhat inspired by
	// AppearanceGeneratorHelper.insertGeneratedAppearance()
	var textFont font.PDFont
	var clipY float32
	clipWidth := width - ab.width*4
	clipHeight := borderBox.Height() - ab.width*4
	if rotation == 90 || rotation == 270 {
		clipHeight = borderBox.Width() - ab.width*4
	}
	h.extractFontDetails(annot)
	if h.document != nil && AcroFormDefaultResourcesFont != nil {
		// Try to get font from AcroForm default resources
		// Sample file: https://gitlab.freedesktop.org/poppler/poppler/issues/6
		if defaultResourcesFont := AcroFormDefaultResourcesFont(
			h.document, h.fontName); defaultResourcesFont != nil {
			textFont = defaultResourcesFont
		}
	}
	if textFont == nil {
		textFont, err = h.DefaultFont()
		if err != nil {
			return err
		}
	}

	// value used by Adobe, no idea where it comes from, actual font bbox max y is 0.931
	// gathered by creating an annotation with width 0.
	const yDelta = 0.7896
	switch rotation {
	case 180:
		xOffset = -borderBox.UpperRightX() + ab.width*2
		yOffset = -borderBox.LowerLeftY() - ab.width*2 - yDelta*h.fontSize
		clipY = -borderBox.UpperRightY() + ab.width*2
	case 90:
		xOffset = borderBox.LowerLeftY() + ab.width*2
		yOffset = -borderBox.LowerLeftX() - ab.width*2 - yDelta*h.fontSize
		clipY = -borderBox.UpperRightX() + ab.width*2
	case 270:
		xOffset = -borderBox.UpperRightY() + ab.width*2
		yOffset = borderBox.UpperRightX() - ab.width*2 - yDelta*h.fontSize
		clipY = borderBox.LowerLeftX() + ab.width*2
	default:
		xOffset = borderBox.LowerLeftX() + ab.width*2
		yOffset = borderBox.UpperRightY() - ab.width*2 - yDelta*h.fontSize
		clipY = borderBox.LowerLeftY() + ab.width*2
	}

	// clip writing area
	if err := cs.AddRect(xOffset, clipY, clipWidth, clipHeight); err != nil {
		return err
	}
	if err := cs.Clip(); err != nil {
		return err
	}

	annotationContents := annot.Contents()
	if annotationContents != "" {
		if err := cs.BeginText(); err != nil {
			return err
		}
		if err := cs.SetFont(textFont, h.fontSize); err != nil {
			return err
		}
		if err := cs.SetNonStrokingColorComponents(textColor.Components()); err != nil {
			return err
		}
		appearanceStyle := interactive.NewAppearanceStyle()
		appearanceStyle.SetFont(textFont)
		appearanceStyle.SetFontSize(h.fontSize)
		formatter := interactive.NewPlainTextFormatterBuilder(cs).
			Style(appearanceStyle).
			Text(interactive.NewPlainText(annotationContents)).
			Width(width-ab.width*4).
			WrapLines(true).
			InitialOffset(xOffset, yOffset).
			// Adobe ignores the /Q
			//.textAlign(annotation.getQ())
			Build()
		formatErr := formatter.Format()
		if err := cs.EndText(); err != nil {
			return err
		}
		if formatErr != nil {
			return formatErr
		}
	}

	if len(pathsArray) > 0 {
		rect := h.Rectangle()

		// Adjust rectangle
		// important to do this after the rectangle has been painted, because the
		// final rectangle will be bigger due to callout
		// CTAN-example-Annotations.pdf p1
		//TODO in a class structure this should be overridable
		minX := float32(math.MaxFloat32)
		minY := float32(math.MaxFloat32)
		maxX := float32(math.SmallestNonzeroFloat32)
		maxY := float32(math.SmallestNonzeroFloat32)
		for i := 0; i < len(pathsArray)/2; i++ {
			x := pathsArray[i*2]
			y := pathsArray[i*2+1]
			minX = min(minX, x)
			minY = min(minY, y)
			maxX = max(maxX, x)
			maxY = max(maxY, y)
		}
		// arrow length is 9 * width at about 30 degrees => 10 * width seems to be enough
		rect.SetLowerLeftX(min(minX-ab.width*10, rect.LowerLeftX()))
		rect.SetLowerLeftY(min(minY-ab.width*10, rect.LowerLeftY()))
		rect.SetUpperRightX(max(maxX+ab.width*10, rect.UpperRightX()))
		rect.SetUpperRightY(max(maxY+ab.width*10, rect.UpperRightY()))
		annot.SetRectangle(rect)

		// need to set the BBox too, because rectangle modification came later
		annot.NormalAppearanceStream().SetBBox(h.Rectangle())

		//TODO when callout is used, /RD should be so that the result is the writable part
	}
	return nil
}

// extractNonStrokingColor returns the last non-stroking colour of the /DA entry.
// Java declares it private.
func (h *PDFreeTextAppearanceHandler) extractNonStrokingColor(
	annot *annotation.PDAnnotationFreeText) *color.PDColor {
	// It could also work with a regular expression, but that should be written so that
	// "/LucidaConsole 13.94766 Tf .392 .585 .93 rg" does not produce "2 .585 .93 rg" as result
	// Another alternative might be to create a PDDocument and a PDPage with /DA content as /Content,
	// process the whole thing and then get the non stroking color.

	strokingColor := color.NewPDColorOfComponents([]float32{0}, color.DeviceGray)
	defaultAppearance := annot.DefaultAppearance()
	if defaultAppearance == "" {
		return strokingColor
	}

	// not sure if charset is correct, but we only need numbers and simple characters
	arguments, graphicOp, err := lastOperatorArguments(defaultAppearance, func(name string) bool {
		return name == operator.NonStrokingGray ||
			name == operator.NonStrokingRgb ||
			name == operator.NonStrokingCmyk
	})
	if err != nil {
		slog.Warn("handlers: problem parsing /DA, will use default black", slog.Any("err", err))
		return strokingColor
	}
	if graphicOp != nil {
		switch graphicOp.Name() {
		case operator.NonStrokingGray:
			strokingColor = color.NewPDColorOfCOSArray(arguments, color.DeviceGray)
		case operator.NonStrokingRgb:
			strokingColor = color.NewPDColorOfCOSArray(arguments, color.DeviceRGB)
		case operator.NonStrokingCmyk:
			strokingColor = color.NewPDColorOfCOSArray(arguments, color.DeviceCMYK)
		}
	}
	return strokingColor
}

// extractFontDetails reads the font name and size out of the /DA entry. Java
// declares it private.
//
// TODO extractNonStrokingColor and extractFontDetails might somehow be replaced
// with PDDefaultAppearanceString, which is quite similar.
func (h *PDFreeTextAppearanceHandler) extractFontDetails(
	annot *annotation.PDAnnotationFreeText) {
	defaultAppearance := annot.DefaultAppearance()
	if defaultAppearance == "" && h.document != nil && AcroFormDefaultAppearance != nil {
		defaultAppearance = AcroFormDefaultAppearance(h.document)
	}
	if defaultAppearance == "" {
		return
	}

	// not sure if charset is correct, but we only need numbers and simple characters
	fontArguments, _, err := lastOperatorArguments(defaultAppearance, func(name string) bool {
		return name == operator.SetFontAndSize
	})
	if err != nil {
		slog.Warn("handlers: problem parsing /DA, will use default 'Helv 10'",
			slog.Any("err", err))
		return
	}
	if fontArguments.Size() >= 2 {
		if base, isName := fontArguments.Get(0).(*cos.Name); isName {
			h.fontName = base
		}
		if base, isNumber := fontArguments.Get(1).(cos.Number); isNumber {
			h.fontSize = base.FloatValue()
		}
	}
}

// lastOperatorArguments parses the given content stream fragment and returns the
// operands of the last operator the given test accepts, along with that
// operator.
//
// Java writes the same loop twice, once in extractNonStrokingColor and once in
// extractFontDetails, differing only in the operator names it looks for; the
// port names the loop once. Where no operator matched, the arguments are the
// operands gathered since the last operator, which is what Java's
// `fontArguments` holds too.
func lastOperatorArguments(defaultAppearance string,
	wanted func(name string) bool) (*cos.Array, *operator.Operator, error) {
	parser, err := pdfparser.NewStreamTokenParser(encodeASCII(defaultAppearance))
	if err != nil {
		return cos.NewArray(), nil, err
	}
	arguments := cos.NewArray()
	matched := cos.NewArray()
	var matchedOp *operator.Operator
	for {
		token, err := parser.ParseNextToken()
		if err != nil {
			return cos.NewArray(), nil, err
		}
		if token == nil {
			break
		}
		if op, isOperator := token.(*operator.Operator); isOperator {
			if wanted(op.Name()) {
				matchedOp = op
				matched = arguments
			}
			arguments = cos.NewArray()
			continue
		}
		arguments.Add(token.(cos.Base))
	}
	if matchedOp == nil {
		// extractFontDetails reads the arguments even where no Tf was found:
		// its fontArguments starts empty and is only ever replaced, so an empty
		// array is what it holds.
		return matched, nil, nil
	}
	return matched, matchedOp, nil
}

// encodeASCII is String.getBytes(StandardCharsets.US_ASCII), which writes a
// question mark for anything outside ASCII.
func encodeASCII(text string) []byte {
	out := make([]byte, 0, len(text))
	for _, r := range text {
		if r > 0x7F {
			out = append(out, '?')
			continue
		}
		out = append(out, byte(r))
	}
	return out
}

// toRadians is java.lang.Math.toRadians.
func toRadians(angdeg float64) float64 { return angdeg / 180.0 * math.Pi }

// GenerateRolloverAppearance does nothing: Java leaves it to be implemented.
func (h *PDFreeTextAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: Java leaves it to be implemented.
func (h *PDFreeTextAppearanceHandler) GenerateDownAppearance() error { return nil }
