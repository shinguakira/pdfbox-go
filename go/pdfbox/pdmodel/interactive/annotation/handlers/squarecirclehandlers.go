package handlers

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// markupLineWidth returns the line width of a markup annotation: the border
// style's where there is one, the third entry of /Border otherwise, and 1
// where there is neither.
//
// Port of the package-private getLineWidth that the square, circle, line,
// polygon and polyline handlers each carry a copy of.
func markupLineWidth(annot annotation.PDAnnotation) float32 {
	markup, isMarkup := annot.(interface {
		BorderStyle() *annotation.PDBorderStyleDictionary
	})
	if isMarkup {
		if bs := markup.BorderStyle(); bs != nil {
			return bs.Width()
		}
	}
	borderCharacteristics := annot.Border()
	if borderCharacteristics.Size() >= 3 {
		if base, isNumber := borderCharacteristics.GetObject(2).(cos.Number); isNumber {
			return base.FloatValue()
		}
	}
	return 1
}

// PDSquareAppearanceHandler draws a square annotation.
//
// Port of PDSquareAppearanceHandler.
type PDSquareAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDSquareAppearanceHandler builds a handler for the given annotation.
func NewPDSquareAppearanceHandler(annot annotation.PDAnnotation) *PDSquareAppearanceHandler {
	return NewPDSquareAppearanceHandlerInDocument(annot, nil)
}

// NewPDSquareAppearanceHandlerInDocument builds one whose streams belong to the
// given document.
func NewPDSquareAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDSquareAppearanceHandler {
	h := &PDSquareAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// LineWidth returns the line width of the border. Java declares it
// package-private.
func (h *PDSquareAppearanceHandler) LineWidth() float32 {
	return markupLineWidth(h.Annotation())
}

// GenerateNormalAppearance draws the square.
func (h *PDSquareAppearanceHandler) GenerateNormalAppearance() error {
	lineWidth := h.LineWidth()
	annot, isSquare := h.Annotation().(*annotation.PDAnnotationSquare)
	if !isSquare {
		panic("handlers: the annotation of a square handler is not a square")
	}
	contentStream, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: square appearance", slog.Any("error", err))
		return nil
	}
	defer contentStream.Close()

	if err := h.drawSquare(annot, lineWidth, contentStream); err != nil {
		slog.Error("handlers: square appearance", slog.Any("error", err))
	}
	return nil
}

// drawSquare is the body of the try block Java writes.
func (h *PDSquareAppearanceHandler) drawSquare(annot *annotation.PDAnnotationSquare,
	lineWidth float32, contentStream annotation.AppearanceContentStream) error {
	hasStroke, err := contentStream.SetStrokingColorOnDemand(h.Color())
	if err != nil {
		return err
	}
	hasBackground, err := contentStream.SetNonStrokingColorOnDemand(annot.InteriorColor())
	if err != nil {
		return err
	}
	if err := h.SetOpacity(contentStream, annot.ConstantOpacity()); err != nil {
		return err
	}
	if err := contentStream.SetBorderLine(lineWidth, annot.BorderStyle(),
		annot.Border()); err != nil {
		return err
	}

	borderEffect := annot.BorderEffect()
	if borderEffect != nil && borderEffect.Style() == annotation.BorderEffectStyleCloudy {
		border := newCloudyBorder(contentStream, float64(borderEffect.Intensity()),
			float64(lineWidth), h.Rectangle())
		if err := border.createCloudyRectangle(annot.RectDifference()); err != nil {
			return err
		}
		annot.SetRectangle(border.rectangle())
		annot.SetRectDifference(border.rectDifference())
		appearanceStream := annot.NormalAppearanceStream()
		appearanceStream.SetBBox(border.bbox())
		appearanceStream.SetMatrix(border.matrix())
	} else {
		borderBox := h.HandleBorderBox(&annot.PDAnnotationSquareCircle, lineWidth)
		if err := contentStream.AddRect(borderBox.LowerLeftX(), borderBox.LowerLeftY(),
			borderBox.Width(), borderBox.Height()); err != nil {
			return err
		}
	}
	return contentStream.DrawShape(lineWidth, hasStroke, hasBackground)
}

// GenerateRolloverAppearance does nothing: Java leaves it to be implemented.
func (h *PDSquareAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: Java leaves it to be implemented.
func (h *PDSquareAppearanceHandler) GenerateDownAppearance() error { return nil }

// PDCircleAppearanceHandler draws a circle annotation.
//
// Port of PDCircleAppearanceHandler.
type PDCircleAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDCircleAppearanceHandler builds a handler for the given annotation.
func NewPDCircleAppearanceHandler(annot annotation.PDAnnotation) *PDCircleAppearanceHandler {
	return NewPDCircleAppearanceHandlerInDocument(annot, nil)
}

// NewPDCircleAppearanceHandlerInDocument builds one whose streams belong to the
// given document.
func NewPDCircleAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDCircleAppearanceHandler {
	h := &PDCircleAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// LineWidth returns the line width of the border. Java declares it
// package-private.
func (h *PDCircleAppearanceHandler) LineWidth() float32 {
	return markupLineWidth(h.Annotation())
}

// GenerateNormalAppearance draws the ellipse.
func (h *PDCircleAppearanceHandler) GenerateNormalAppearance() error {
	lineWidth := h.LineWidth()
	annot, isCircle := h.Annotation().(*annotation.PDAnnotationCircle)
	if !isCircle {
		panic("handlers: the annotation of a circle handler is not a circle")
	}
	contentStream, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: circle appearance", slog.Any("error", err))
		return nil
	}
	defer contentStream.Close()

	if err := h.drawCircle(annot, lineWidth, contentStream); err != nil {
		slog.Error("handlers: circle appearance", slog.Any("error", err))
	}
	return nil
}

// drawCircle is the body of the try block Java writes.
func (h *PDCircleAppearanceHandler) drawCircle(annot *annotation.PDAnnotationCircle,
	lineWidth float32, contentStream annotation.AppearanceContentStream) error {
	hasStroke, err := contentStream.SetStrokingColorOnDemand(h.Color())
	if err != nil {
		return err
	}
	hasBackground, err := contentStream.SetNonStrokingColorOnDemand(annot.InteriorColor())
	if err != nil {
		return err
	}
	if err := h.SetOpacity(contentStream, annot.ConstantOpacity()); err != nil {
		return err
	}
	if err := contentStream.SetBorderLine(lineWidth, annot.BorderStyle(),
		annot.Border()); err != nil {
		return err
	}

	borderEffect := annot.BorderEffect()
	if borderEffect != nil && borderEffect.Style() == annotation.BorderEffectStyleCloudy {
		border := newCloudyBorder(contentStream, float64(borderEffect.Intensity()),
			float64(lineWidth), h.Rectangle())
		if err := border.createCloudyEllipse(annot.RectDifference()); err != nil {
			return err
		}
		annot.SetRectangle(border.rectangle())
		annot.SetRectDifference(border.rectDifference())
		appearanceStream := annot.NormalAppearanceStream()
		appearanceStream.SetBBox(border.bbox())
		appearanceStream.SetMatrix(border.matrix())
	} else {
		// Acrobat applies a padding to each side of the bbox so the line is completely within
		// the bbox.
		borderBox := h.HandleBorderBox(&annot.PDAnnotationSquareCircle, lineWidth)

		// lower left corner
		x0 := borderBox.LowerLeftX()
		y0 := borderBox.LowerLeftY()
		// upper right corner
		x1 := borderBox.UpperRightX()
		y1 := borderBox.UpperRightY()
		// mid points
		xm := x0 + borderBox.Width()/2
		ym := y0 + borderBox.Height()/2
		// see http://spencermortensen.com/articles/bezier-circle/
		// the below number was calculated from sampling content streams
		// generated using Adobe Reader
		magic := float32(0.55555417)
		// control point offsets
		vOffset := borderBox.Height() / 2 * magic
		hOffset := borderBox.Width() / 2 * magic

		if err := contentStream.MoveTo(xm, y1); err != nil {
			return err
		}
		if err := contentStream.CurveTo(xm+hOffset, y1, x1, ym+vOffset, x1, ym); err != nil {
			return err
		}
		if err := contentStream.CurveTo(x1, ym-vOffset, xm+hOffset, y0, xm, y0); err != nil {
			return err
		}
		if err := contentStream.CurveTo(xm-hOffset, y0, x0, ym-vOffset, x0, ym); err != nil {
			return err
		}
		if err := contentStream.CurveTo(x0, ym+vOffset, xm-hOffset, y1, xm, y1); err != nil {
			return err
		}
		if err := contentStream.ClosePath(); err != nil {
			return err
		}
	}
	return contentStream.DrawShape(lineWidth, hasStroke, hasBackground)
}

// GenerateRolloverAppearance does nothing: Java leaves it to be implemented.
func (h *PDCircleAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: Java leaves it to be implemented.
func (h *PDCircleAppearanceHandler) GenerateDownAppearance() error { return nil }
