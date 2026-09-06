package handlers

import (
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// PDSoundAppearanceHandler draws a sound annotation.
//
// Port of PDSoundAppearanceHandler, whose three methods Java leaves as TODO.
type PDSoundAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDSoundAppearanceHandler builds a handler for the given annotation.
func NewPDSoundAppearanceHandler(annot annotation.PDAnnotation) *PDSoundAppearanceHandler {
	return NewPDSoundAppearanceHandlerInDocument(annot, nil)
}

// NewPDSoundAppearanceHandlerInDocument builds one whose streams belong to the
// given document.
func NewPDSoundAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDSoundAppearanceHandler {
	h := &PDSoundAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance does nothing: Java leaves it to be implemented.
func (h *PDSoundAppearanceHandler) GenerateNormalAppearance() error {
	// TODO to be implemented
	return nil
}

// GenerateRolloverAppearance does nothing: Java leaves it to be implemented.
func (h *PDSoundAppearanceHandler) GenerateRolloverAppearance() error {
	// TODO to be implemented
	return nil
}

// GenerateDownAppearance does nothing: Java leaves it to be implemented.
func (h *PDSoundAppearanceHandler) GenerateDownAppearance() error {
	// TODO to be implemented
	return nil
}

// PDCaretAppearanceHandler draws a caret annotation.
//
// Port of PDCaretAppearanceHandler.
type PDCaretAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDCaretAppearanceHandler builds a handler for the given annotation.
func NewPDCaretAppearanceHandler(annot annotation.PDAnnotation) *PDCaretAppearanceHandler {
	return NewPDCaretAppearanceHandlerInDocument(annot, nil)
}

// NewPDCaretAppearanceHandlerInDocument builds one whose streams belong to the
// given document.
func NewPDCaretAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDCaretAppearanceHandler {
	h := &PDCaretAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance draws the caret.
func (h *PDCaretAppearanceHandler) GenerateNormalAppearance() error {
	annot, isCaret := h.Annotation().(*annotation.PDAnnotationCaret)
	if !isCaret {
		panic("handlers: the annotation of a caret handler is not a caret")
	}
	contentStream, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: caret appearance", slog.Any("error", err))
		return nil
	}
	defer contentStream.Close()

	if err := h.drawCaret(annot, contentStream); err != nil {
		slog.Error("handlers: caret appearance", slog.Any("error", err))
	}
	return nil
}

// drawCaret is the body of the try block Java writes.
func (h *PDCaretAppearanceHandler) drawCaret(annot *annotation.PDAnnotationCaret,
	contentStream annotation.AppearanceContentStream) error {
	annotationColor := h.Color()
	if err := contentStream.SetStrokingColor(annotationColor); err != nil {
		return err
	}
	if err := contentStream.SetNonStrokingColor(annotationColor); err != nil {
		return err
	}
	if err := h.SetOpacity(contentStream, annot.ConstantOpacity()); err != nil {
		return err
	}

	rect := h.Rectangle()
	rectWidth := rect.Width()
	rectHeight := rect.Height()
	bbox := common.NewPDRectangleOfSize(rectWidth, rectHeight)
	pdAppearanceStream := annot.NormalAppearanceStream()
	if !annot.AnnotationDictionary().ContainsKey(cos.RD) {
		// Adobe creates the /RD entry with a number that is decided
		// by dividing the height by 10, with a maximum result of 5.
		// That number is then used to enlarge the bbox and the rectangle and added to the
		// translation values in the matrix and also used for the line width
		// (not here because it has no effect, see comment near fill() ).
		// The curves are based on the original rectangle.
		rd := float32(math.Min(float64(rectHeight/10), 5))
		annot.SetRectDifferences(rd)
		bbox = common.NewPDRectangleOf(-rd, -rd, rectWidth+2*rd, rectHeight+2*rd)
		pdAppearanceStream.SetMatrix(pdAppearanceStream.Matrix())
		rect2 := common.NewPDRectangleOf(rect.LowerLeftX()-rd, rect.LowerLeftY()-rd,
			rectWidth+2*rd, rectHeight+2*rd)
		annot.SetRectangle(rect2)
	}
	pdAppearanceStream.SetBBox(bbox)

	halfX := rectWidth / 2
	halfY := rectHeight / 2
	if err := contentStream.MoveTo(0, 0); err != nil {
		return err
	}
	if err := contentStream.CurveTo(halfX, 0, halfX, halfY, halfX, rectHeight); err != nil {
		return err
	}
	if err := contentStream.CurveTo(halfX, halfY, halfX, 0, rectWidth, 0); err != nil {
		return err
	}
	if err := contentStream.ClosePath(); err != nil {
		return err
	}
	// Adobe has an additional stroke, but it has no effect
	// because fill "consumes" the path.
	return contentStream.Fill()
}

// GenerateRolloverAppearance does nothing: Java leaves it to be implemented.
func (h *PDCaretAppearanceHandler) GenerateRolloverAppearance() error {
	// TODO to be implemented
	return nil
}

// GenerateDownAppearance does nothing: Java leaves it to be implemented.
func (h *PDCaretAppearanceHandler) GenerateDownAppearance() error {
	// TODO to be implemented
	return nil
}

// PDInkAppearanceHandler draws an ink annotation.
//
// Port of PDInkAppearanceHandler.
type PDInkAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDInkAppearanceHandler builds a handler for the given annotation.
func NewPDInkAppearanceHandler(annot annotation.PDAnnotation) *PDInkAppearanceHandler {
	return NewPDInkAppearanceHandlerInDocument(annot, nil)
}

// NewPDInkAppearanceHandlerInDocument builds one whose streams belong to the
// given document.
func NewPDInkAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDInkAppearanceHandler {
	h := &PDInkAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance draws the ink paths.
func (h *PDInkAppearanceHandler) GenerateNormalAppearance() error {
	ink, isInk := h.Annotation().(*annotation.PDAnnotationInk)
	if !isInk {
		panic("handlers: the annotation of an ink handler is not an ink")
	}
	inkColor := ink.Color()
	if inkColor == nil || len(inkColor.Components()) == 0 {
		return nil
	}

	// PDF spec does not mention /Border for ink annotations, but it is used if /BS is not available
	ab := getAnnotationBorder(ink, ink.BorderStyle())
	if ab.width == 0 {
		return nil
	}

	// Adjust rectangle even if not empty
	// file from PDF.js issue 13447
	// TODO in a class structure this should be overridable
	minX := float32(math.MaxFloat32)
	minY := float32(math.MaxFloat32)
	maxX := float32(smallestPositiveFloat32)
	maxY := float32(smallestPositiveFloat32)
	for _, pathArray := range ink.InkList() {
		nPoints := len(pathArray) / 2
		for i := 0; i < nPoints; i++ {
			x := pathArray[i*2]
			y := pathArray[i*2+1]
			minX = minFloat32(minX, x)
			minY = minFloat32(minY, y)
			maxX = maxFloat32(maxX, x)
			maxY = maxFloat32(maxY, y)
		}
	}

	rect := ink.Rectangle()
	if rect == nil {
		return nil
	}
	rect.SetLowerLeftX(minFloat32(minX-ab.width*2, rect.LowerLeftX()))
	rect.SetLowerLeftY(minFloat32(minY-ab.width*2, rect.LowerLeftY()))
	rect.SetUpperRightX(maxFloat32(maxX+ab.width*2, rect.UpperRightX()))
	rect.SetUpperRightY(maxFloat32(maxY+ab.width*2, rect.UpperRightY()))
	ink.SetRectangle(rect)

	cs, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: ink appearance", slog.Any("error", err))
		return nil
	}
	defer cs.Close()

	if err := h.drawInk(ink, ab, inkColor, cs); err != nil {
		slog.Error("handlers: ink appearance", slog.Any("error", err))
	}
	return nil
}

// drawInk is the body of the try block Java writes.
func (h *PDInkAppearanceHandler) drawInk(ink *annotation.PDAnnotationInk, ab *annotationBorder,
	inkColor *color.PDColor, cs annotation.AppearanceContentStream) error {
	if err := h.SetOpacity(cs, ink.ConstantOpacity()); err != nil {
		return err
	}
	if err := cs.SetStrokingColor(inkColor); err != nil {
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
	for _, pathArray := range ink.InkList() {
		nPoints := len(pathArray) / 2

		// "When drawn, the points shall be connected by straight lines or curves
		// in an implementation-dependent way" - we do lines.
		for i := 0; i < nPoints; i++ {
			x := pathArray[i*2]
			y := pathArray[i*2+1]
			var err error
			if i == 0 {
				err = cs.MoveTo(x, y)
			} else {
				err = cs.LineTo(x, y)
			}
			if err != nil {
				return err
			}
		}
		if err := cs.Stroke(); err != nil {
			return err
		}
	}
	return nil
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated.
func (h *PDInkAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated.
func (h *PDInkAppearanceHandler) GenerateDownAppearance() error { return nil }

// smallestPositiveFloat32 is Java's Float.MIN_VALUE, the smallest positive
// value a float can hold. Java starts its maximum search from it, which leaves
// the maximum wrong for a wholly negative path; the port keeps that.
const smallestPositiveFloat32 = 1.4e-45

// minFloat32 is Math.min for a float.
func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// maxFloat32 is Math.max for a float.
func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
