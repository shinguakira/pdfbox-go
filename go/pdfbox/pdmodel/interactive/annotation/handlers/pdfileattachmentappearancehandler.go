package handlers

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PDFileAttachmentAppearanceHandler draws a file attachment annotation.
//
// Port of PDFileAttachmentAppearanceHandler. The four icons are long literal
// paths, written here through a pathWriter so that each drawing statement is
// one line, as it is in Java.
type PDFileAttachmentAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDFileAttachmentAppearanceHandler builds a handler for the given
// annotation.
func NewPDFileAttachmentAppearanceHandler(
	annot annotation.PDAnnotation) *PDFileAttachmentAppearanceHandler {
	return NewPDFileAttachmentAppearanceHandlerInDocument(annot, nil)
}

// NewPDFileAttachmentAppearanceHandlerInDocument builds one whose streams
// belong to the given document.
func NewPDFileAttachmentAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDFileAttachmentAppearanceHandler {
	h := &PDFileAttachmentAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance draws the icon of the attachment.
func (h *PDFileAttachmentAppearanceHandler) GenerateNormalAppearance() error {
	annot, isFileAttachment := h.Annotation().(*annotation.PDAnnotationFileAttachment)
	if !isFileAttachment {
		panic("handlers: the annotation of a file attachment handler is not a file attachment")
	}
	rect := h.Rectangle()
	if rect == nil {
		return nil
	}
	contentStream, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: file attachment appearance", slog.Any("error", err))
		return nil
	}
	defer contentStream.Close()

	if err := h.drawAttachment(annot, rect, contentStream); err != nil {
		slog.Error("handlers: file attachment appearance", slog.Any("error", err))
	}
	return nil
}

// drawAttachment is the body of the try block Java writes.
func (h *PDFileAttachmentAppearanceHandler) drawAttachment(
	annot *annotation.PDAnnotationFileAttachment, rect *common.PDRectangle,
	contentStream annotation.AppearanceContentStream) error {
	if err := h.SetOpacity(contentStream, annot.ConstantOpacity()); err != nil {
		return err
	}

	// minimum code of PDTextAppearanceHandler.adjustRectAndBBox()
	const size = 18
	rect.SetUpperRightX(rect.LowerLeftX() + size)
	rect.SetLowerLeftY(rect.UpperRightY() - size)
	annot.SetRectangle(rect)
	annot.NormalAppearanceStream().SetBBox(common.NewPDRectangleOfSize(size, size))

	// test case: pdf_commenting_new.pdf page 7
	switch annot.AttachmentName() {
	case "Paperclip":
		return h.drawPaperclip(contentStream)
	case "Graph":
		return h.drawGraph(contentStream)
	case "Tag":
		return h.drawTag(contentStream)
	default:
		return h.drawPushPin(contentStream)
	}
}

// drawPaperclip draws the paperclip icon.
func (h *PDFileAttachmentAppearanceHandler) drawPaperclip(
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	p.moveTo(13.574, 9.301)
	p.lineTo(8.926, 13.949)
	p.curveTo(7.648, 15.227, 5.625, 15.227, 4.426, 13.949)
	p.curveTo(3.148, 12.676, 3.148, 10.648, 4.426, 9.449)
	p.lineTo(10.426, 3.449)
	p.curveTo(11.176, 2.773, 12.301, 2.773, 13.051, 3.449)
	p.curveTo(13.801, 4.199, 13.801, 5.398, 13.051, 6.074)
	p.lineTo(7.875, 11.25)
	p.curveTo(7.648, 11.477, 7.273, 11.477, 7.051, 11.25)
	p.curveTo(6.824, 11.023, 6.824, 10.648, 7.051, 10.426)
	p.lineTo(10.875, 6.602)
	p.curveTo(11.176, 6.301, 11.176, 5.852, 10.875, 5.551)
	p.curveTo(10.574, 5.25, 10.125, 5.25, 9.824, 5.551)
	p.lineTo(6, 9.449)
	p.curveTo(5.176, 10.273, 5.176, 11.551, 6, 12.375)
	p.curveTo(6.824, 13.125, 8.102, 13.125, 8.926, 12.375)
	p.lineTo(14.102, 7.199)
	p.curveTo(15.449, 5.852, 15.449, 3.75, 14.102, 2.398)
	p.curveTo(12.75, 1.051, 10.648, 1.051, 9.301, 2.398)
	p.lineTo(3.301, 8.398)
	p.curveTo(2.398, 9.301, 1.949, 10.5, 1.949, 11.699)
	p.curveTo(1.949, 14.324, 4.051, 16.352, 6.676, 16.352)
	p.curveTo(7.949, 16.352, 9.074, 15.824, 9.977, 15)
	p.lineTo(14.625, 10.352)
	p.curveTo(14.926, 10.051, 14.926, 9.602, 14.625, 9.301)
	p.curveTo(14.324, 9, 13.875, 9, 13.574, 9.301)
	p.closePath()
	p.fill()
	return p.err
}

// drawPushPin draws the push pin icon.
func (h *PDFileAttachmentAppearanceHandler) drawPushPin(
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	// ty 18 is from the caller, scale 0.022 is by trial and error
	p.transform(util.NewMatrixOf(0.022, 0, 0, -0.022, 0, 18))
	// Source: https://www.svgrepo.com/svg/269187/push-pin
	// License: CC0
	p.transform(util.TranslateInstance(586.47, 178.97))
	p.moveTo(0, 0)
	p.curveTo(13, 0, 23.43, -10.58, 23.43, -23.57)
	p.lineTo(23.43, -70.53)
	p.curveTo(23.43, -109.32, -8.19, -141.06, -47.03, -141.06)
	p.lineTo(-329.17, -141.06)
	p.curveTo(-368.17, -141.06, -399.79, -109.32, -399.79, -70.53)
	p.lineTo(-399.79, -23.57)
	p.curveTo(-399.79, -10.58, -389.19, 0.0, -376.19, 0)
	p.lineTo(-305.74, 0)
	p.lineTo(-305.74, 129.52)
	p.curveTo(-364.0, 168.47, -399.79, 234.67, -399.79, 305.36)
	p.curveTo(-399.79, 318.34, -389.19, 328.76, -376.19, 328.76)
	p.lineTo(-211.69, 328.76)
	p.lineTo(-211.69, 555.9)
	p.curveTo(-211.69, 568.88, -201.1, 579.3, -188.1, 579.3)
	p.curveTo(-175.1, 579.3, -164.67, 568.88, -164.67, 555.9)
	p.lineTo(-164.67, 328.76)
	p.lineTo(0, 328.76)
	p.curveTo(13.0, 328.76, 23.43, 318.34, 23.43, 305.36)
	p.curveTo(23.43, 234.67, -12.2, 168.47, -70.62, 129.52)
	p.lineTo(-70.62, 0)
	p.lineTo(0, 0)
	p.closePath()
	p.moveTo(-25.2, 281.79)
	p.lineTo(-351.0, 281.79)
	p.curveTo(-343.77, 232.42, -314.24, 188.18, -270.43, 162.86)
	p.curveTo(-263.21, 158.69, -258.71, 150.99, -258.71, 142.5)
	p.lineTo(-258.71, 0)
	p.lineTo(-117.64, 0)
	p.lineTo(-117.64, 142.5)
	p.curveTo(-117.64, 150.99, -113.15, 158.69, -105.77, 162.86)
	p.curveTo(-61.95, 188.18, -32.42, 232.42, -25.2, 281.79)
	p.closePath()
	p.moveTo(-352.76, -46.97)
	p.lineTo(-352.76, -70.53)
	p.curveTo(-352.76, -83.52, -342.17, -93.93, -329.17, -93.93)
	p.lineTo(-47.03, -93.93)
	p.curveTo(-34.03, -93.93, -23.59, -83.52, -23.59, -70.53)
	p.lineTo(-23.59, -46.97)
	p.lineTo(-352.76, -46.97)
	p.lineTo(-352.76, -46.97)
	p.closePath()
	p.fill()
	return p.err
}

// drawGraph draws the histogram icon.
func (h *PDFileAttachmentAppearanceHandler) drawGraph(
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	// ty 18 is from the caller, scale 0.022 is by trial and error
	p.transform(util.NewMatrixOf(0.022, 0, 0, -0.022, 0, 18))
	// Source: https://www.svgrepo.com/svg/339018/chart-histogram
	// Author: Carbon Design https://github.com/carbon-design-system/carbon
	// License: Apache
	p.transform(util.TranslateInstance(736.04, 907.89))
	p.moveTo(0, 0)
	p.lineTo(-675.23, 0)
	p.curveTo(-679.72, 0, -683.41, -3.53, -683.41, -8.01)
	p.lineTo(-683.41, -683.37)
	p.lineTo(-667.22, -683.37)
	p.lineTo(-667.22, -353.95)
	p.curveTo(-583.85, -357.8, -541.53, -419.99, -500.49, -480.27)
	p.curveTo(-459.93, -539.74, -418.09, -601.46, -337.61, -601.46)
	p.curveTo(-257.14, -601.46, -215.3, -539.74, -174.74, -480.27)
	p.curveTo(-132.58, -418.07, -88.81, -353.79, 0, -353.79)
	p.lineTo(0, -337.6)
	p.curveTo(-97.31, -337.6, -143.48, -405.41, -188.2, -471.13)
	p.curveTo(-228.12, -529.8, -265.8, -585.27, -337.61, -585.27)
	p.curveTo(-409.43, -585.27, -447.11, -529.8, -487.03, -471.13)
	p.curveTo(-530.47, -407.33, -575.36, -341.45, -667.22, -337.76)
	p.lineTo(-667.22, -16.19)
	p.lineTo(-615.76, -16.19)
	p.lineTo(-615.76, -255.68)
	p.curveTo(-615.76, -260.17, -612.23, -263.7, -607.74, -263.7)
	p.lineTo(-525.82, -263.7)
	p.lineTo(-525.82, -345.77)
	p.curveTo(-525.82, -350.26, -522.13, -353.79, -517.64, -353.79)
	p.lineTo(-435.73, -353.79)
	p.lineTo(-435.73, -458.31)
	p.curveTo(-435.73, -462.8, -432.2, -466.32, -427.71, -466.32)
	p.lineTo(-337.61, -466.32)
	p.curveTo(-333.13, -466.32, -329.6, -462.8, -329.6, -458.31)
	p.lineTo(-329.6, -421.28)
	p.lineTo(-247.68, -421.28)
	p.curveTo(-243.19, -421.28, -239.5, -417.75, -239.5, -413.26)
	p.lineTo(-239.5, -331.35)
	p.lineTo(-157.58, -331.35)
	p.curveTo(-153.1, -331.35, -149.41, -327.66, -149.41, -323.17)
	p.lineTo(-149.41, -218.81)
	p.lineTo(-67.49, -218.81)
	p.curveTo(-63.0, -218.81, -59.47, -215.13, -59.47, -210.64)
	p.lineTo(-59.47, -16.19)
	p.lineTo(0, -16.19)
	p.lineTo(0, 0)
	p.closePath()
	p.moveTo(-149.41, -16.19)
	p.lineTo(-75.67, -16.19)
	p.lineTo(-75.67, -202.62)
	p.lineTo(-149.41, -202.62)
	p.lineTo(-149.41, -16.19)
	p.closePath()
	p.moveTo(-239.5, -16.19)
	p.lineTo(-165.76, -16.19)
	p.lineTo(-165.76, -315.16)
	p.lineTo(-239.5, -315.16)
	p.lineTo(-239.5, -16.19)
	p.closePath()
	p.moveTo(-329.6, -16.19)
	p.lineTo(-255.7, -16.19)
	p.lineTo(-255.7, -405.09)
	p.lineTo(-329.6, -405.09)
	p.lineTo(-329.6, -16.19)
	p.closePath()
	p.moveTo(-419.53, -16.19)
	p.lineTo(-345.79, -16.19)
	p.lineTo(-345.79, -450.13)
	p.lineTo(-419.53, -450.13)
	p.lineTo(-419.53, -16.19)
	p.closePath()
	p.moveTo(-509.63, -16.19)
	p.lineTo(-435.73, -16.19)
	p.lineTo(-435.73, -337.6)
	p.lineTo(-509.63, -337.6)
	p.lineTo(-509.63, -16.19)
	p.closePath()
	p.moveTo(-599.56, -16.19)
	p.lineTo(-525.82, -16.19)
	p.lineTo(-525.82, -247.51)
	p.lineTo(-599.56, -247.51)
	p.lineTo(-599.56, -16.19)
	p.closePath()
	p.fill()
	return p.err
}

// drawTag draws the tag icon.
func (h *PDFileAttachmentAppearanceHandler) drawTag(
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	// ty 18 is from the caller, scale 0.022 is by trial and error
	p.transform(util.NewMatrixOf(0.022, 0, 0, -0.022, 0, 18))
	// Source: https://www.svgrepo.com/svg/29652/tag
	// License: CC0
	p.saveGraphicsState()
	p.transform(util.TranslateInstance(209.26, 128.32))
	p.moveTo(0, 0)
	p.curveTo(-44.73, 0, -80.64, 36.23, -80.64, 80.64)
	p.curveTo(-80.64, 125.2, -44.57, 161.27, 0, 161.27)
	p.curveTo(44.56, 161.27, 80.47, 125.04, 80.47, 80.64)
	p.curveTo(80.63, 36.07, 44.56, 0, 0, 0)
	p.closePath()
	p.moveTo(0, 132.74)
	p.curveTo(-28.7, 132.74, -52.1, 109.33, -52.1, 80.64)
	p.curveTo(-52.1, 51.94, -28.7, 28.54, 0, 28.54)
	p.curveTo(28.69, 28.54, 51.93, 51.94, 51.93, 80.64)
	p.curveTo(51.93, 109.33, 28.85, 132.74, 0, 132.74)
	p.closePath()
	p.fill()
	p.restoreGraphicsState()
	p.saveGraphicsState()
	p.transform(util.TranslateInstance(382.22, 79.91))
	p.moveTo(0, 0)
	p.curveTo(-14.58, -16.19, -35.1, -24.85, -57.22, -24.85)
	p.lineTo(-208.23, -26.45)
	p.curveTo(-240.45, -26.45, -271.23, -14.75, -293.35, 8.66)
	p.curveTo(-316.76, 30.78, -328.46, 61.56, -328.46, 93.78)
	p.lineTo(-327.02, 244.95)
	p.curveTo(-325.57, 265.47, -318.2, 285.98, -302.17, 302.18)
	p.lineTo(58.68, 663.02)
	p.lineTo(360.85, 360.69)
	p.lineTo(0, 0)
	p.lineTo(0, 0)
	p.closePath()
	p.moveTo(57.23, 621.82)
	p.lineTo(-283.09, 281.5)
	p.curveTo(-293.35, 271.24, -299.12, 258.09, -299.12, 243.34)
	p.lineTo(-300.57, 93.78)
	p.curveTo(-300.57, 70.38, -290.31, 46.81, -274.12, 29.34)
	p.curveTo(-256.64, 11.7, -233.08, 1.44, -208.23, 1.44)
	p.lineTo(-58.67, 2.89)
	p.curveTo(-44.08, 2.89, -30.77, 8.66, -20.51, 19.08)
	p.lineTo(319.81, 359.4)
	p.lineTo(57.23, 621.82)
	p.closePath()
	p.fill()
	p.restoreGraphicsState()
	return p.err
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated.
func (h *PDFileAttachmentAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated.
func (h *PDFileAttachmentAppearanceHandler) GenerateDownAppearance() error { return nil }
