package handlers

import (
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// supportedTextNames are the /Name values this handler draws.
//
// Port of the private SUPPORTED_NAMES.
var supportedTextNames = map[string]bool{
	annotation.TextNameNote:         true,
	annotation.TextNameInsert:       true,
	annotation.TextNameCross:        true,
	annotation.TextNameHelp:         true,
	annotation.TextNameCircle:       true,
	annotation.TextNameParagraph:    true,
	annotation.TextNameNewParagraph: true,
	annotation.TextNameCheck:        true,
	annotation.TextNameStar:         true,
	annotation.TextNameRightArrow:   true,
	annotation.TextNameRightPointer: true,
	annotation.TextNameCrossHairs:   true,
	annotation.TextNameUpArrow:      true,
	annotation.TextNameUpLeftArrow:  true,
	annotation.TextNameComment:      true,
	annotation.TextNameKey:          true,
}

// PDTextAppearanceHandler draws a text annotation, which is a sticky note with
// one of sixteen icons.
//
// Port of PDTextAppearanceHandler. The icons are long literal paths, written
// here through a pathWriter so that each drawing statement is one line, as it
// is in Java.
type PDTextAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDTextAppearanceHandler builds a handler for the given annotation.
func NewPDTextAppearanceHandler(annot annotation.PDAnnotation) *PDTextAppearanceHandler {
	return NewPDTextAppearanceHandlerInDocument(annot, nil)
}

// NewPDTextAppearanceHandlerInDocument builds one whose streams belong to the
// given document.
func NewPDTextAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDTextAppearanceHandler {
	h := &PDTextAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance draws the icon the annotation names.
func (h *PDTextAppearanceHandler) GenerateNormalAppearance() error {
	annot, isText := h.Annotation().(*annotation.PDAnnotationText)
	if !isText {
		panic("handlers: the annotation of a text handler is not a text annotation")
	}
	if !supportedTextNames[annot.Name()] {
		return nil
	}
	contentStream, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: text appearance", slog.Any("error", err))
		return nil
	}
	defer contentStream.Close()

	if err := h.drawIcon(annot, contentStream); err != nil {
		slog.Error("handlers: text appearance", slog.Any("error", err))
	}
	return nil
}

// drawIcon is the body of the try block Java writes.
func (h *PDTextAppearanceHandler) drawIcon(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	bgColor := h.Color()
	if bgColor == nil {
		// White is used by Adobe when /C entry is missing
		if err := contentStream.SetNonStrokingColorGray(1); err != nil {
			return err
		}
	} else if err := contentStream.SetNonStrokingColor(bgColor); err != nil {
		return err
	}

	// stroking color is always black which is the PDF default
	if err := h.SetOpacity(contentStream, annot.ConstantOpacity()); err != nil {
		return err
	}

	switch annot.Name() {
	case annotation.TextNameNote:
		return h.drawNote(annot, contentStream)
	case annotation.TextNameCross:
		return h.drawZapf(annot, contentStream, 19, 0, "a22") // 0x2716
	case annotation.TextNameCircle:
		return h.drawCircles(annot, contentStream)
	case annotation.TextNameInsert:
		return h.drawInsert(annot, contentStream)
	case annotation.TextNameHelp:
		return h.drawHelp(annot, contentStream)
	case annotation.TextNameParagraph:
		return h.drawParagraph(annot, contentStream)
	case annotation.TextNameNewParagraph:
		return h.drawNewParagraph(annot, contentStream)
	case annotation.TextNameStar:
		return h.drawZapf(annot, contentStream, 19, 0, "a35") // 0x2605
	case annotation.TextNameCheck:
		return h.drawZapf(annot, contentStream, 19, 50, "a20") // 0x2714
	case annotation.TextNameRightArrow:
		return h.drawRightArrow(annot, contentStream)
	case annotation.TextNameRightPointer:
		return h.drawZapf(annot, contentStream, 17, 50, "a174") // 0x27A4
	case annotation.TextNameCrossHairs:
		return h.drawCrossHairs(annot, contentStream)
	case annotation.TextNameUpArrow:
		return h.drawUpArrow(annot, contentStream)
	case annotation.TextNameUpLeftArrow:
		return h.drawUpLeftArrow(annot, contentStream)
	case annotation.TextNameComment:
		return h.drawComment(annot, contentStream)
	case annotation.TextNameKey:
		return h.drawKey(annot, contentStream)
	}
	return nil
}

// adjustRectAndBBox sets the rectangle and the bounding box to the fixed size
// of an icon, and returns the box. Java declares it private.
func (h *PDTextAppearanceHandler) adjustRectAndBBox(annot *annotation.PDAnnotationText,
	width, height float32) *common.PDRectangle {
	// For /Note (other types have different values):
	// Adobe takes the left upper bound as anchor, and adjusts the rectangle to 18 x 20.
	// Observed with files 007071.pdf, 038785.pdf, 038787.pdf,
	// but not with 047745.pdf p133 and 084374.pdf p48, both have the NoZoom flag.
	// there the BBox is also set to fixed values, but the rectangle is left untouched.
	// When no flags are there, Adobe sets /F 24 = NoZoom NoRotate.
	rect := h.Rectangle()
	if !annot.IsNoZoom() {
		rect.SetUpperRightX(rect.LowerLeftX() + width)
		rect.SetLowerLeftY(rect.UpperRightY() - height)
		annot.SetRectangle(rect)
	}
	if !annot.AnnotationDictionary().ContainsKey(cos.F) {
		// We set these flags because Adobe does so, but PDFBox does not support
		// them when rendering.
		annot.SetNoRotate(true)
		annot.SetNoZoom(true)
	}
	bbox := common.NewPDRectangleOfSize(width, height)
	annot.NormalAppearanceStream().SetBBox(bbox)
	return bbox
}

// drawZapf draws one Zapf Dingbats glyph as the icon. Java declares it private.
func (h *PDTextAppearanceHandler) drawZapf(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream, by, ty int, glyphName string) error {
	bbox := h.adjustRectAndBBox(annot, 20, float32(by))
	min := minFloat32(bbox.Width(), bbox.Height())

	p := newPathWriter(contentStream)
	p.setMiterLimit(4)
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.setLineWidth(0.59) // value from Adobe

	zapf, err := font.NewPDType1FontStandard14(font.ZapfDingbatsFontName)
	if err != nil {
		return err
	}
	fontMatrix, err := zapf.FontBoxFont().FontMatrix()
	if err != nil {
		return err
	}
	xScale := fontMatrix[0]
	yScale := fontMatrix[3]
	p.transform(util.ScaleInstance(xScale*min/0.8, yScale*min/0.8))
	p.transform(util.TranslateInstance(0, float32(ty)))

	// we get the shape of a Zapf Dingbats glyph and use that one.
	// Adobe uses a different font (which one?), or created the shape from scratch.
	path, err := font.GetGlyphPath(font.ZapfDingbatsFontName, glyphName)
	if err != nil {
		return err
	}
	if err := h.addPath(p, path); err != nil {
		return err
	}
	p.fillAndStroke()
	return p.err
}

// addPath writes the segments of a glyph path into the content stream. Java
// declares it private.
func (h *PDTextAppearanceHandler) addPath(p *pathWriter, path *geom.Path2D) error {
	curX := 0.0
	curY := 0.0
	it := path.PathIterator(geom.NewIdentityTransform())
	coords := make([]float64, 6)
	for !it.IsDone() {
		switch it.CurrentSegment(coords) {
		case geom.SegClose:
			p.closePath()
		case geom.SegCubicTo:
			p.curveTo(float32(coords[0]), float32(coords[1]), float32(coords[2]),
				float32(coords[3]), float32(coords[4]), float32(coords[5]))
			curX = coords[4]
			curY = coords[5]
		case geom.SegQuadTo:
			// Convert quadratic Bezier curve to cubic
			// https://fontforge.github.io/bezier.html
			// CP1 = QP0 + 2/3 *(QP1-QP0)
			// CP2 = QP2 + 2/3 *(QP1-QP2)
			cp1x := curX + 2.0/3.0*(coords[0]-curX)
			cp1y := curY + 2.0/3.0*(coords[1]-curY)
			cp2x := coords[2] + 2.0/3.0*(coords[0]-coords[2])
			cp2y := coords[3] + 2.0/3.0*(coords[1]-coords[3])
			p.curveTo(float32(cp1x), float32(cp1y),
				float32(cp2x), float32(cp2y),
				float32(coords[2]), float32(coords[3]))
			curX = coords[2]
			curY = coords[3]
		case geom.SegLineTo:
			p.lineTo(float32(coords[0]), float32(coords[1]))
			curX = coords[0]
			curY = coords[1]
		case geom.SegMoveTo:
			p.moveTo(float32(coords[0]), float32(coords[1]))
			curX = coords[0]
			curY = coords[1]
		}
		it.Next()
	}
	return p.err
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated.
func (h *PDTextAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated.
func (h *PDTextAppearanceHandler) GenerateDownAppearance() error { return nil }

// drawNote draws the icon.
func (h *PDTextAppearanceHandler) drawNote(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	bbox := h.adjustRectAndBBox(annot, 18, 20)
	p.setMiterLimit(4)
	// get round edge the easy way. Adobe uses 4 lines with 4 arcs of radius 0.785 which is bigger.
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.setLineWidth(0.61) // value from Adobe
	width := bbox.Width()
	height := bbox.Height()
	p.addRect(1, 1, width-2, height-2)
	p.moveTo(width/4, height/7*2)
	p.lineTo(width*3/4-1, height/7*2)
	p.moveTo(width/4, height/7*3)
	p.lineTo(width*3/4-1, height/7*3)
	p.moveTo(width/4, height/7*4)
	p.lineTo(width*3/4-1, height/7*4)
	p.moveTo(width/4, height/7*5)
	p.lineTo(width*3/4-1, height/7*5)
	p.fillAndStroke()
	return p.err
}

// drawCircles draws the icon.
func (h *PDTextAppearanceHandler) drawCircles(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	bbox := h.adjustRectAndBBox(annot, 20, 20)
	// strategy used by Adobe:
	// 1) add small circle in white using /ca /CA 0.6 and width 1
	// 2) fill
	// 3) add small circle in one direction
	// 4) add large circle in other direction
	// 5) stroke + fill
	// with square width 20 small r = 6.36, large r = 9.756
	smallR := float32(6.36)
	largeR := float32(9.756)
	// adjustments because the bottom of the circle is flat
	p.transform(util.ScaleInstance(0.95, 0.95))
	p.transform(util.TranslateInstance(0, 0.5))
	p.setMiterLimit(4)
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.saveGraphicsState()
	p.setLineWidth(1)
	alphaConstant := float32(0.6)
	gs := state.NewPDExtendedGraphicsState()
	gs.SetAlphaSourceFlag(false)
	gs.SetStrokingAlphaConstant(&alphaConstant)
	gs.SetNonStrokingAlphaConstant(&alphaConstant)
	gs.SetBlendMode(blend.Normal)
	p.setGraphicsStateParameters(gs)
	p.setNonStrokingColorGray(1)
	width := bbox.Width() / 2
	height := bbox.Height() / 2
	p.circle(&h.PDAbstractAppearanceHandler, width, height, smallR)
	p.fill()
	p.restoreGraphicsState()
	p.setLineWidth(0.59) // value from Adobe
	p.circle(&h.PDAbstractAppearanceHandler, width, height, smallR)
	p.circle2(&h.PDAbstractAppearanceHandler, width, height, largeR)
	p.fillAndStroke()
	return p.err
}

// drawInsert draws the icon.
func (h *PDTextAppearanceHandler) drawInsert(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	bbox := h.adjustRectAndBBox(annot, 17, 20)
	p.setMiterLimit(4)
	p.setLineJoinStyle(0)
	p.setLineCapStyle(0)
	p.setLineWidth(0.59) // value from Adobe
	p.moveTo(bbox.Width()/2-1, bbox.Height()-2)
	p.lineTo(1, 1)
	p.lineTo(bbox.Width()-2, 1)
	p.closeAndFillAndStroke()
	return p.err
}

// drawHelp draws the icon.
func (h *PDTextAppearanceHandler) drawHelp(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	bbox := h.adjustRectAndBBox(annot, 20, 20)
	min := minFloat32(bbox.Width(), bbox.Height())
	p.setMiterLimit(4)
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.setLineWidth(0.59) // value from Adobe
	// Adobe first fills a white circle with CA ca 0.6, so do we
	p.saveGraphicsState()
	p.setLineWidth(1)
	alphaConstant := float32(0.6)
	gs := state.NewPDExtendedGraphicsState()
	gs.SetAlphaSourceFlag(false)
	gs.SetStrokingAlphaConstant(&alphaConstant)
	gs.SetNonStrokingAlphaConstant(&alphaConstant)
	gs.SetBlendMode(blend.Normal)
	p.setGraphicsStateParameters(gs)
	p.setNonStrokingColorGray(1)
	p.circle2(&h.PDAbstractAppearanceHandler, min/2, min/2, min/2-1)
	p.fill()
	p.restoreGraphicsState()
	p.saveGraphicsState()
	// rescale so that "?" fits into circle and move "?" to circle center
	// values gathered by trial and error
	p.transform(util.ScaleInstance(0.001*min/2.25, 0.001*min/2.25))
	p.transform(util.TranslateInstance(500, 375))
	// we get the shape of an Helvetica bold "?" and use that one.
	// Adobe uses a different font (which one?), or created the shape from scratch.
	path, err := font.GetGlyphPath(font.HelveticaBold, "question")
	if err != nil {
		return err
	}
	if err := h.addPath(p, path); err != nil {
		return err
	}
	p.restoreGraphicsState()
	// draw the outer circle counterclockwise to fill area between circle and "?"
	p.circle2(&h.PDAbstractAppearanceHandler, min/2, min/2, min/2-1)
	p.fillAndStroke()
	return p.err
}

// drawParagraph draws the icon.
func (h *PDTextAppearanceHandler) drawParagraph(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	bbox := h.adjustRectAndBBox(annot, 20, 20)
	min := minFloat32(bbox.Width(), bbox.Height())
	p.setMiterLimit(4)
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.setLineWidth(0.59) // value from Adobe
	// Adobe first fills a white circle with CA ca 0.6, so do we
	p.saveGraphicsState()
	p.setLineWidth(1)
	alphaConstant := float32(0.6)
	gs := state.NewPDExtendedGraphicsState()
	gs.SetAlphaSourceFlag(false)
	gs.SetStrokingAlphaConstant(&alphaConstant)
	gs.SetNonStrokingAlphaConstant(&alphaConstant)
	gs.SetBlendMode(blend.Normal)
	p.setGraphicsStateParameters(gs)
	p.setNonStrokingColorGray(1)
	p.circle2(&h.PDAbstractAppearanceHandler, min/2, min/2, min/2-1)
	p.fill()
	p.restoreGraphicsState()
	p.saveGraphicsState()
	// rescale so that "?" fits into circle and move "?" to circle center
	// values gathered by trial and error
	p.transform(util.ScaleInstance(0.001*min/3, 0.001*min/3))
	p.transform(util.TranslateInstance(850, 900))
	// we get the shape of an Helvetica "?" and use that one.
	// Adobe uses a different font (which one?), or created the shape from scratch.
	path, err := font.GetGlyphPath(font.Helvetica, "paragraph")
	if err != nil {
		return err
	}
	if err := h.addPath(p, path); err != nil {
		return err
	}
	p.restoreGraphicsState()
	p.fillAndStroke()
	p.circle(&h.PDAbstractAppearanceHandler, min/2, min/2, min/2-1)
	p.stroke()
	return p.err
}

// drawNewParagraph draws the icon.
func (h *PDTextAppearanceHandler) drawNewParagraph(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	h.adjustRectAndBBox(annot, 13, 20)
	p.setMiterLimit(4)
	p.setLineJoinStyle(0)
	p.setLineCapStyle(0)
	p.setLineWidth(0.59) // value from Adobe
	// small triangle (values from Adobe)
	p.moveTo(6.4995, 20)
	p.lineTo(0.295, 7.287)
	p.lineTo(12.705, 7.287)
	p.closeAndFillAndStroke()
	// rescale and translate so that "NP" fits below the triangle
	// values gathered by trial and error
	p.transform(util.ScaleInstance(0.001*4, 0.001*4))
	p.transform(util.TranslateInstance(200, 0))
	glyphN, err := font.GetGlyphPath(font.HelveticaBold, "N")
	if err != nil {
		return err
	}
	if err := h.addPath(p, glyphN); err != nil {
		return err
	}
	p.transform(util.TranslateInstance(1300, 0))
	glyphP, err := font.GetGlyphPath(font.HelveticaBold, "P")
	if err != nil {
		return err
	}
	if err := h.addPath(p, glyphP); err != nil {
		return err
	}
	p.fill()
	return p.err
}

// drawCrossHairs draws the icon.
func (h *PDTextAppearanceHandler) drawCrossHairs(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	symbol, err := font.NewPDType1FontStandard14(font.SymbolFontName)
	if err != nil {
		return err
	}
	fontMatrix, err := symbol.FontBoxFont().FontMatrix()
	if err != nil {
		return err
	}
	xScale := fontMatrix[0]
	yScale := fontMatrix[3]
	bbox := h.adjustRectAndBBox(annot, 20, 20)
	min := minFloat32(bbox.Width(), bbox.Height())
	p.setMiterLimit(4)
	p.setLineJoinStyle(0)
	p.setLineCapStyle(0)
	p.setLineWidth(0.61) // value from Adobe
	p.transform(util.ScaleInstance(xScale*min*1.3333, yScale*min*1.3333))
	p.transform(util.TranslateInstance(0, 50))
	// we get the shape of a Symbol crosshair (0x2295) and use that one.
	// Adobe uses a different font (which one?), or created the shape from scratch.
	path, err := font.GetGlyphPath(font.SymbolFontName, "circleplus")
	if err != nil {
		return err
	}
	if err := h.addPath(p, path); err != nil {
		return err
	}
	p.fillAndStroke()
	return p.err
}

// drawUpArrow draws the icon.
func (h *PDTextAppearanceHandler) drawUpArrow(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	h.adjustRectAndBBox(annot, 17, 20)
	p.setMiterLimit(4)
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.setLineWidth(0.59) // value from Adobe
	p.moveTo(1, 7)
	p.lineTo(5, 7)
	p.lineTo(5, 1)
	p.lineTo(12, 1)
	p.lineTo(12, 7)
	p.lineTo(16, 7)
	p.lineTo(8.5, 19)
	p.closeAndFillAndStroke()
	return p.err
}

// drawUpLeftArrow draws the icon.
func (h *PDTextAppearanceHandler) drawUpLeftArrow(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	h.adjustRectAndBBox(annot, 17, 17)
	p.setMiterLimit(4)
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.setLineWidth(0.59) // value from Adobe
	p.transform(util.RotateInstance(45*math.Pi/180, 8, -4))
	p.moveTo(1, 7)
	p.lineTo(5, 7)
	p.lineTo(5, 1)
	p.lineTo(12, 1)
	p.lineTo(12, 7)
	p.lineTo(16, 7)
	p.lineTo(8.5, 19)
	p.closeAndFillAndStroke()
	return p.err
}

// drawRightArrow draws the icon.
func (h *PDTextAppearanceHandler) drawRightArrow(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	bbox := h.adjustRectAndBBox(annot, 20, 20)
	min := minFloat32(bbox.Width(), bbox.Height())
	p.setMiterLimit(4)
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.setLineWidth(0.59) // value from Adobe
	// Adobe first fills a white circle with CA ca 0.6, so do we
	p.saveGraphicsState()
	p.setLineWidth(1)
	alphaConstant := float32(0.6)
	gs := state.NewPDExtendedGraphicsState()
	gs.SetAlphaSourceFlag(false)
	gs.SetStrokingAlphaConstant(&alphaConstant)
	gs.SetNonStrokingAlphaConstant(&alphaConstant)
	gs.SetBlendMode(blend.Normal)
	p.setGraphicsStateParameters(gs)
	p.setNonStrokingColorGray(1)
	p.circle2(&h.PDAbstractAppearanceHandler, min/2, min/2, min/2-1)
	p.fill()
	p.restoreGraphicsState()
	p.saveGraphicsState()
	p.moveTo(8, 17.5)
	p.lineTo(8, 13.5)
	p.lineTo(3, 13.5)
	p.lineTo(3, 6.5)
	p.lineTo(8, 6.5)
	p.lineTo(8, 2.5)
	p.lineTo(18, 10)
	p.closePath()
	p.restoreGraphicsState()
	// surprisingly, this one not counterclockwise.
	p.circle(&h.PDAbstractAppearanceHandler, min/2, min/2, min/2-1)
	p.fillAndStroke()
	return p.err
}

// drawComment draws the icon.
func (h *PDTextAppearanceHandler) drawComment(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	h.adjustRectAndBBox(annot, 18, 18)
	p.setMiterLimit(4)
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.setLineWidth(200)
	// Adobe first fills a white rectangle with CA ca 0.6, so do we
	p.saveGraphicsState()
	p.setLineWidth(1)
	alphaConstant := float32(0.6)
	gs := state.NewPDExtendedGraphicsState()
	gs.SetAlphaSourceFlag(false)
	gs.SetStrokingAlphaConstant(&alphaConstant)
	gs.SetNonStrokingAlphaConstant(&alphaConstant)
	gs.SetBlendMode(blend.Normal)
	p.setGraphicsStateParameters(gs)
	p.setNonStrokingColorGray(1)
	p.addRect(0.3, 0.3, 18-0.6, 18-0.6)
	p.fill()
	p.restoreGraphicsState()
	p.transform(util.ScaleInstance(0.003, 0.003))
	p.transform(util.TranslateInstance(500, -300))
	// outer shape was gathered from Font Awesome by "printing" comment.svg
	// into a PDF and looking at the content stream
	p.moveTo(2549, 5269)
	p.curveTo(1307, 5269, 300, 4451, 300, 3441)
	p.curveTo(300, 3023, 474, 2640, 764, 2331)
	p.curveTo(633, 1985, 361, 1691, 357, 1688)
	p.curveTo(299, 1626, 283, 1537, 316, 1459)
	p.curveTo(350, 1382, 426, 1332, 510, 1332)
	p.curveTo(1051, 1332, 1477, 1558, 1733, 1739)
	p.curveTo(1987, 1659, 2261, 1613, 2549, 1613)
	p.curveTo(3792, 1613, 4799, 2431, 4799, 3441)
	p.curveTo(4799, 4451, 3792, 5269, 2549, 5269)
	p.closePath()
	// can't use addRect: if we did that, we wouldn't get the donut effect
	p.moveTo(0.3/0.003-500, 0.3/0.003+300)
	p.lineTo(0.3/0.003-500, 0.3/0.003+300+17.4/0.003)
	p.lineTo(0.3/0.003-500+17.4/0.003, 0.3/0.003+300+17.4/0.003)
	p.lineTo(0.3/0.003-500+17.4/0.003, 0.3/0.003+300)
	p.closeAndFillAndStroke()
	return p.err
}

// drawKey draws the icon.
func (h *PDTextAppearanceHandler) drawKey(annot *annotation.PDAnnotationText,
	contentStream annotation.AppearanceContentStream) error {
	p := newPathWriter(contentStream)
	h.adjustRectAndBBox(annot, 13, 18)
	p.setMiterLimit(4)
	p.setLineJoinStyle(1)
	p.setLineCapStyle(0)
	p.setLineWidth(200)
	p.transform(util.ScaleInstance(0.003, 0.003))
	p.transform(util.RotateInstance(45*math.Pi/180, 2500, -800))
	// shape was gathered from Font Awesome by "printing" key.svg into a PDF
	// and looking at the content stream
	p.moveTo(4799, 4004)
	p.curveTo(4799, 3149, 4107, 2457, 3253, 2457)
	p.curveTo(3154, 2457, 3058, 2466, 2964, 2484)
	p.lineTo(2753, 2246)
	p.curveTo(2713, 2201, 2656, 2175, 2595, 2175)
	p.lineTo(2268, 2175)
	p.lineTo(2268, 1824)
	p.curveTo(2268, 1707, 2174, 1613, 2057, 1613)
	p.lineTo(1706, 1613)
	p.lineTo(1706, 1261)
	p.curveTo(1706, 1145, 1611, 1050, 1495, 1050)
	p.lineTo(510, 1050)
	p.curveTo(394, 1050, 300, 1145, 300, 1261)
	p.lineTo(300, 1947)
	p.curveTo(300, 2003, 322, 2057, 361, 2097)
	p.lineTo(1783, 3519)
	p.curveTo(1733, 3671, 1706, 3834, 1706, 4004)
	p.curveTo(1706, 4858, 2398, 5550, 3253, 5550)
	p.curveTo(4109, 5550, 4799, 4860, 4799, 4004)
	p.closePath()
	p.moveTo(3253, 4425)
	p.curveTo(3253, 4192, 3441, 4004, 3674, 4004)
	p.curveTo(3907, 4004, 4096, 4192, 4096, 4425)
	p.curveTo(4096, 4658, 3907, 4847, 3674, 4847)
	p.curveTo(3441, 4847, 3253, 4658, 3253, 4425)
	p.fillAndStroke()
	return p.err
}
