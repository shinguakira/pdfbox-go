package handlers

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// PDLinkAppearanceHandler draws a link annotation.
//
// Port of PDLinkAppearanceHandler.
type PDLinkAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDLinkAppearanceHandler builds a handler for the given annotation.
func NewPDLinkAppearanceHandler(annot annotation.PDAnnotation) *PDLinkAppearanceHandler {
	return NewPDLinkAppearanceHandlerInDocument(annot, nil)
}

// NewPDLinkAppearanceHandlerInDocument builds one whose streams belong to the
// given document.
func NewPDLinkAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDLinkAppearanceHandler {
	h := &PDLinkAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// LineWidth returns the line width of the border. Java declares it
// package-private.
func (h *PDLinkAppearanceHandler) LineWidth() float32 {
	return markupLineWidth(h.Annotation())
}

// GenerateNormalAppearance draws the border of the link.
func (h *PDLinkAppearanceHandler) GenerateNormalAppearance() error {
	annot, isLink := h.Annotation().(*annotation.PDAnnotationLink)
	if !isLink {
		panic("handlers: the annotation of a link handler is not a link")
	}
	rect := annot.Rectangle()
	if rect == nil {
		// 660402-p1-AnnotationEmptyRect.pdf has /Rect entry with 0 elements
		return nil
	}

	// Adobe doesn't generate an appearance for a link annotation
	lineWidth := h.LineWidth()
	contentStream, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: link appearance", slog.Any("error", err))
		return nil
	}
	defer contentStream.Close()

	if err := h.drawLink(annot, rect, lineWidth, contentStream); err != nil {
		slog.Error("handlers: link appearance", slog.Any("error", err))
	}
	return nil
}

// drawLink is the body of the try block Java writes.
func (h *PDLinkAppearanceHandler) drawLink(annot *annotation.PDAnnotationLink,
	rect *common.PDRectangle, lineWidth float32,
	contentStream annotation.AppearanceContentStream) error {
	linkColor := annot.Color()
	if linkColor == nil {
		// spec is unclear, but black is what Adobe does
		linkColor = color.NewPDColorOfComponents([]float32{0}, color.DeviceGray)
	}
	hasStroke, err := contentStream.SetStrokingColorOnDemand(linkColor)
	if err != nil {
		return err
	}
	if err := contentStream.SetBorderLine(lineWidth, annot.BorderStyle(),
		annot.Border()); err != nil {
		return err
	}

	// Acrobat applies a padding to each side of the bbox so the line is completely within
	// the bbox.
	borderEdge := h.PaddedRectangle(h.Rectangle(), lineWidth/2)

	pathsArray := annot.QuadPoints()
	if pathsArray != nil {
		// QuadPoints shall be ignored if any coordinate in the array lies outside
		// the region specified by Rect.
		for i := 0; i < len(pathsArray)/2; i++ {
			if !rect.Contains(pathsArray[i*2], pathsArray[i*2+1]) {
				slog.Warn("handlers: at least one /QuadPoints entry is outside of the rectangle, "+
					"/QuadPoints are ignored and /Rect is used instead",
					slog.Float64("x", float64(pathsArray[i*2])),
					slog.Float64("y", float64(pathsArray[i*2+1])),
					slog.Any("rect", rect))
				pathsArray = nil
				break
			}
		}
	}
	if pathsArray == nil {
		// Convert rectangle coordinates as if it was a /QuadPoints entry
		pathsArray = make([]float32, 8)
		pathsArray[0] = borderEdge.LowerLeftX()
		pathsArray[1] = borderEdge.LowerLeftY()
		pathsArray[2] = borderEdge.UpperRightX()
		pathsArray[3] = pathsArray[1]
		pathsArray[4] = pathsArray[2]
		pathsArray[5] = borderEdge.UpperRightY()
		pathsArray[6] = pathsArray[0]
		pathsArray[7] = pathsArray[5]
	}

	underlined := false
	if len(pathsArray) >= 8 {
		if borderStyleDic := annot.BorderStyle(); borderStyleDic != nil {
			underlined = borderStyleDic.Style() == annotation.BorderStyleUnderline
		}
	}

	of := 0
	for of+7 < len(pathsArray) {
		if err := contentStream.MoveTo(pathsArray[of], pathsArray[of+1]); err != nil {
			return err
		}
		if err := contentStream.LineTo(pathsArray[of+2], pathsArray[of+3]); err != nil {
			return err
		}
		if !underlined {
			if err := contentStream.LineTo(pathsArray[of+4], pathsArray[of+5]); err != nil {
				return err
			}
			if err := contentStream.LineTo(pathsArray[of+6], pathsArray[of+7]); err != nil {
				return err
			}
			if err := contentStream.ClosePath(); err != nil {
				return err
			}
		}
		of += 8
	}
	return contentStream.DrawShape(lineWidth, hasStroke, false)
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated for
// a link annotation.
func (h *PDLinkAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated for a link
// annotation.
func (h *PDLinkAppearanceHandler) GenerateDownAppearance() error { return nil }
