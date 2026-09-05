package handlers

import (
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// lineFontSize is the size the caption of a line annotation is drawn at.
//
// Port of the package-private PDLineAppearanceHandler.FONT_SIZE.
const lineFontSize = 9

// PDLineAppearanceHandler draws a line annotation.
//
// Port of PDLineAppearanceHandler.
type PDLineAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDLineAppearanceHandler builds a handler for the given annotation.
func NewPDLineAppearanceHandler(annot annotation.PDAnnotation) *PDLineAppearanceHandler {
	return NewPDLineAppearanceHandlerInDocument(annot, nil)
}

// NewPDLineAppearanceHandlerInDocument builds one whose streams belong to the
// given document.
func NewPDLineAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDLineAppearanceHandler {
	h := &PDLineAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance draws the line, its leader lines, its caption and
// its endings.
func (h *PDLineAppearanceHandler) GenerateNormalAppearance() error {
	annot, isLine := h.Annotation().(*annotation.PDAnnotationLine)
	if !isLine {
		panic("handlers: the annotation of a line handler is not a line")
	}
	rect := annot.Rectangle()
	if rect == nil {
		return nil
	}
	pathsArray := annot.Line()
	if pathsArray == nil {
		return nil
	}
	ab := getAnnotationBorder(annot, annot.BorderStyle())
	lineColor := annot.Color()
	if lineColor == nil || len(lineColor.Components()) == 0 {
		return nil
	}
	ll := annot.LeaderLineLength()
	lle := annot.LeaderLineExtensionLength()
	llo := annot.LeaderLineOffsetLength()

	// Adjust rectangle even if not empty, see PLPDF.com-MarkupAnnotations.pdf
	minX := float32(math.MaxFloat32)
	minY := float32(math.MaxFloat32)
	maxX := float32(smallestPositiveFloat32)
	maxY := float32(smallestPositiveFloat32)
	for i := 0; i < len(pathsArray)/2; i++ {
		x := pathsArray[i*2]
		y := pathsArray[i*2+1]
		minX = minFloat32(minX, x)
		minY = minFloat32(minY, y)
		maxX = maxFloat32(maxX, x)
		maxY = maxFloat32(maxY, y)
	}

	// Leader lines
	if ll < 0 {
		// /LLO and /LLE go in the same direction as /LL
		llo = -llo
		lle = -lle
	}

	// observed with diagonal line of AnnotationSample.Standard.pdf
	// for line endings, very small widths must be treated as size 1.
	// However the border of the line ending shapes is not drawn.
	lineEndingSize := ab.width
	if ab.width < 1e-5 {
		lineEndingSize = 1
	}

	// add/subtract with, font height, and arrows
	// arrow length is 9 * width at about 30 degrees => 10 * width seems to be enough
	// but need to consider /LL, /LLE and /LLO too
	// TODO find better way to calculate padding
	max := maxFloat32(lineEndingSize*10, float32(math.Abs(float64(llo+ll+lle))))
	rect.SetLowerLeftX(minFloat32(minX-max, rect.LowerLeftX()))
	rect.SetLowerLeftY(minFloat32(minY-max, rect.LowerLeftY()))
	rect.SetUpperRightX(maxFloat32(maxX+max, rect.UpperRightX()))
	rect.SetUpperRightY(maxFloat32(maxY+max, rect.UpperRightY()))
	annot.SetRectangle(rect)

	cs, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: line appearance", slog.Any("error", err))
		return nil
	}
	defer cs.Close()

	if err := h.drawLine(annot, ab, lineColor, pathsArray, ll, lle, llo,
		lineEndingSize, cs); err != nil {
		slog.Error("handlers: line appearance", slog.Any("error", err))
	}
	return nil
}

// drawLine is the body of the try block Java writes.
func (h *PDLineAppearanceHandler) drawLine(annot *annotation.PDAnnotationLine,
	ab *annotationBorder, lineColor *color.PDColor, pathsArray []float32,
	ll, lle, llo, lineEndingSize float32,
	cs annotation.AppearanceContentStream) error {
	if err := h.SetOpacity(cs, annot.ConstantOpacity()); err != nil {
		return err
	}

	// Tested with Adobe Reader:
	// text is written first (TODO)
	// width 0 is used by Adobe as such (but results in a visible line in rendering)
	// empty color array results in an invisible line ("n" operator) but the rest is visible
	// empty content is like no caption
	hasStroke, err := cs.SetStrokingColorOnDemand(lineColor)
	if err != nil {
		return err
	}
	if ab.dashArray != nil {
		if err := cs.SetLineDashPattern(ab.dashArray, 0); err != nil {
			return err
		}
	}
	if err := cs.SetLineWidth(ab.width); err != nil {
		return err
	}

	x1 := pathsArray[0]
	y1 := pathsArray[1]
	x2 := pathsArray[2]
	y2 := pathsArray[3]

	// if there are leader lines, then the /L coordinates represent
	// the endpoints of the leader lines rather than the endpoints of the line itself.
	// so for us, llo + ll is the vertical offset for the line.
	y := llo + ll

	contents := annot.Contents()

	if err := cs.SaveGraphicsState(); err != nil {
		return err
	}
	angle := math.Atan2(float64(y2-y1), float64(x2-x1))
	if err := cs.Transform(util.RotateInstance(angle, x1, y1)); err != nil {
		return err
	}
	lineLength := float32(math.Sqrt(float64((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1))))

	// Leader lines
	if err := cs.MoveTo(0, llo); err != nil {
		return err
	}
	if err := cs.LineTo(0, llo+ll+lle); err != nil {
		return err
	}
	if err := cs.MoveTo(lineLength, llo); err != nil {
		return err
	}
	if err := cs.LineTo(lineLength, llo+ll+lle); err != nil {
		return err
	}

	startPointEndingStyle := annot.StartPointEndingStyle()
	endPointEndingStyle := annot.EndPointEndingStyle()

	if annot.HasCaption() && contents != "" {
		if err := h.drawCaptionedLine(annot, contents, lineLength, y, lineEndingSize,
			startPointEndingStyle, endPointEndingStyle, hasStroke, cs); err != nil {
			return err
		}
	} else {
		if shortStyles[startPointEndingStyle] {
			if err := cs.MoveTo(lineEndingSize, y); err != nil {
				return err
			}
		} else if err := cs.MoveTo(0, y); err != nil {
			return err
		}
		if shortStyles[endPointEndingStyle] {
			if err := cs.LineTo(lineLength-lineEndingSize, y); err != nil {
				return err
			}
		} else if err := cs.LineTo(lineLength, y); err != nil {
			return err
		}
		if err := cs.DrawShape(lineEndingSize, hasStroke, false); err != nil {
			return err
		}
	}
	if err := cs.RestoreGraphicsState(); err != nil {
		return err
	}

	// paint the styles here and not before showing the text, or the text would appear
	// with the interior color
	hasBackground, err := cs.SetNonStrokingColorOnDemand(annot.InteriorColor())
	if err != nil {
		return err
	}

	// observed with diagonal line of file AnnotationSample.Standard.pdf
	// when width is very small, the border of the line ending shapes
	// is not drawn.
	if ab.width < 1e-5 {
		hasStroke = false
	}

	// check for LE_NONE only needed to avoid q cm Q for that case
	if startPointEndingStyle != annotation.LENone {
		if err := cs.SaveGraphicsState(); err != nil {
			return err
		}
		if angledStyles[startPointEndingStyle] {
			if err := cs.Transform(util.RotateInstance(angle, x1, y1)); err != nil {
				return err
			}
			if err := h.DrawStyle(startPointEndingStyle, cs, 0, y, lineEndingSize,
				hasStroke, hasBackground, false); err != nil {
				return err
			}
		} else {
			// Support of non-angled styles is more difficult than in the other handlers
			// because the lines do not always go from (x1,y1) to (x2,y2) due to the leader lines
			// when the "y" value above is not 0.
			// We use the angle we already know and the distance y to translate to the new coordinate.
			xx1 := x1 - float32(float64(y)*math.Sin(angle))
			yy1 := y1 + float32(float64(y)*math.Cos(angle))
			if err := h.DrawStyle(startPointEndingStyle, cs, xx1, yy1, lineEndingSize,
				hasStroke, hasBackground, false); err != nil {
				return err
			}
		}
		if err := cs.RestoreGraphicsState(); err != nil {
			return err
		}
	}

	// check for LE_NONE only needed to avoid q cm Q for that case
	if endPointEndingStyle != annotation.LENone {
		// save / restore not needed because it's the last one
		if angledStyles[endPointEndingStyle] {
			if err := cs.Transform(util.RotateInstance(angle, x2, y2)); err != nil {
				return err
			}
			return h.DrawStyle(endPointEndingStyle, cs, 0, y, lineEndingSize,
				hasStroke, hasBackground, true)
		}
		// Support of non-angled styles is more difficult than in the other handlers
		// because the lines do not always go from (x1,y1) to (x2,y2) due to the leader lines
		// when the "y" value above is not 0.
		// We use the angle we already know and the distance y to translate to the new coordinate.
		xx2 := x2 - float32(float64(y)*math.Sin(angle))
		yy2 := y2 + float32(float64(y)*math.Cos(angle))
		return h.DrawStyle(endPointEndingStyle, cs, xx2, yy2, lineEndingSize,
			hasStroke, hasBackground, true)
	}
	return nil
}

// drawCaptionedLine draws the line in two halves with the caption between them,
// or under it.
func (h *PDLineAppearanceHandler) drawCaptionedLine(annot *annotation.PDAnnotationLine,
	contents string, lineLength, y, lineEndingSize float32,
	startPointEndingStyle, endPointEndingStyle string, hasStroke bool,
	cs annotation.AppearanceContentStream) error {
	// Note that Adobe places the text as a caption even if /CP is not set
	// when the text is so long that it would cross arrows, but we ignore this for now
	// and stick to the specification.

	captionFont, err := h.DefaultFont()
	if err != nil {
		return err
	}

	// TODO: support newlines!!!!!
	// see https://www.pdfill.com/example/pdf_commenting_new.pdf
	contentLength := float32(0)
	// TODO How to decide the size of the font?
	// 9 seems to be standard, but if the text doesn't fit, a scaling is done
	// see AnnotationSample.Standard.pdf, diagonal line
	width, err := captionFont.StringWidth(contents)
	if err != nil {
		// Adobe Reader displays placeholders instead
		slog.Error("handlers: line text can't be shown",
			slog.String("text", contents), slog.Any("error", err))
	} else {
		contentLength = width / 1000 * lineFontSize
	}

	xOffset := (lineLength - contentLength) / 2
	var yOffset float32
	captionPositioning := annot.CaptionPositioning()

	// draw the line horizontally, using the rotation CTM to get to correct final position
	// that's the easiest way to calculate the positions for the line before and after inline caption
	if shortStyles[startPointEndingStyle] {
		if err := cs.MoveTo(lineEndingSize, y); err != nil {
			return err
		}
	} else if err := cs.MoveTo(0, y); err != nil {
		return err
	}

	if captionPositioning == "Top" {
		// this arbitrary number is from Adobe
		yOffset = 1.908
	} else {
		// Inline
		// this arbitrary number is from Adobe
		yOffset = -2.6
		if err := cs.LineTo(xOffset-lineEndingSize, y); err != nil {
			return err
		}
		if err := cs.MoveTo(lineLength-xOffset+lineEndingSize, y); err != nil {
			return err
		}
	}

	if shortStyles[endPointEndingStyle] {
		if err := cs.LineTo(lineLength-lineEndingSize, y); err != nil {
			return err
		}
	} else if err := cs.LineTo(lineLength, y); err != nil {
		return err
	}
	if err := cs.DrawShape(lineEndingSize, hasStroke, false); err != nil {
		return err
	}

	// /CO entry (caption offset)
	captionHorizontalOffset := annot.CaptionHorizontalOffset()
	captionVerticalOffset := annot.CaptionVerticalOffset()

	// check contentLength so we don't show if there was trouble before
	if contentLength > 0 {
		if err := h.showCaption(captionFont, contents,
			xOffset+captionHorizontalOffset, y+yOffset+captionVerticalOffset, cs); err != nil {
			return err
		}
	}

	if captionVerticalOffset != 0 {
		// Adobe paints vertical bar to the caption
		if err := cs.MoveTo(0+lineLength/2, y); err != nil {
			return err
		}
		if err := cs.LineTo(0+lineLength/2, y+captionVerticalOffset); err != nil {
			return err
		}
		return cs.DrawShape(lineEndingSize, hasStroke, false)
	}
	return nil
}

// showCaption writes the caption text at the given offset.
func (h *PDLineAppearanceHandler) showCaption(captionFont font.PDFont, contents string,
	x, y float32, cs annotation.AppearanceContentStream) error {
	if err := cs.BeginText(); err != nil {
		return err
	}
	if err := cs.SetFont(captionFont, lineFontSize); err != nil {
		return err
	}
	if err := cs.NewLineAtOffset(x, y); err != nil {
		return err
	}
	if err := cs.ShowText(contents); err != nil {
		return err
	}
	return cs.EndText()
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated.
func (h *PDLineAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated.
func (h *PDLineAppearanceHandler) GenerateDownAppearance() error { return nil }
