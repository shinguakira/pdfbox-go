package handlers

import (
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PDPolygonAppearanceHandler draws a polygon annotation.
//
// Port of PDPolygonAppearanceHandler.
type PDPolygonAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDPolygonAppearanceHandler builds a handler for the given annotation.
func NewPDPolygonAppearanceHandler(annot annotation.PDAnnotation) *PDPolygonAppearanceHandler {
	return NewPDPolygonAppearanceHandlerInDocument(annot, nil)
}

// NewPDPolygonAppearanceHandlerInDocument builds one whose streams belong to
// the given document.
func NewPDPolygonAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDPolygonAppearanceHandler {
	h := &PDPolygonAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// LineWidth returns the line width of the border. Java declares it
// package-private.
func (h *PDPolygonAppearanceHandler) LineWidth() float32 {
	return markupLineWidth(h.Annotation())
}

// GenerateNormalAppearance draws the polygon.
func (h *PDPolygonAppearanceHandler) GenerateNormalAppearance() error {
	annot, isPolygon := h.Annotation().(*annotation.PDAnnotationPolygon)
	if !isPolygon {
		panic("handlers: the annotation of a polygon handler is not a polygon")
	}
	lineWidth := h.LineWidth()
	rect := annot.Rectangle()
	if rect == nil {
		return nil
	}

	// Adjust rectangle even if not empty
	// CTAN-example-Annotations.pdf p2
	minX := float32(math.MaxFloat32)
	minY := float32(math.MaxFloat32)
	maxX := float32(smallestPositiveFloat32)
	maxY := float32(smallestPositiveFloat32)
	pathArray := pathArrayOf(annot)
	if pathArray == nil {
		return nil
	}
	for i := 0; i < len(pathArray); i++ {
		for j := 0; j < len(pathArray[i])/2; j++ {
			x := pathArray[i][j*2]
			y := pathArray[i][j*2+1]
			minX = minFloat32(minX, x)
			minY = minFloat32(minY, y)
			maxX = maxFloat32(maxX, x)
			maxY = maxFloat32(maxY, y)
		}
	}
	rect.SetLowerLeftX(minFloat32(minX-lineWidth, rect.LowerLeftX()))
	rect.SetLowerLeftY(minFloat32(minY-lineWidth, rect.LowerLeftY()))
	rect.SetUpperRightX(maxFloat32(maxX+lineWidth, rect.UpperRightX()))
	rect.SetUpperRightY(maxFloat32(maxY+lineWidth, rect.UpperRightY()))
	annot.SetRectangle(rect)

	contentStream, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: polygon appearance", slog.Any("error", err))
		return nil
	}
	defer contentStream.Close()

	if err := h.drawPolygon(annot, lineWidth, pathArray, contentStream); err != nil {
		slog.Error("handlers: polygon appearance", slog.Any("error", err))
	}
	return nil
}

// drawPolygon is the body of the try block Java writes.
func (h *PDPolygonAppearanceHandler) drawPolygon(annot *annotation.PDAnnotationPolygon,
	lineWidth float32, pathArray [][]float32,
	contentStream annotation.AppearanceContentStream) error {
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
		if err := border.createCloudyPolygon(pathArray); err != nil {
			return err
		}
		annot.SetRectangle(border.rectangle())
		appearanceStream := annot.NormalAppearanceStream()
		appearanceStream.SetBBox(border.bbox())
		appearanceStream.SetMatrix(border.matrix())
	} else {
		// Acrobat applies a padding to each side of the bbox so the line is
		// completely within the bbox.
		for i := 0; i < len(pathArray); i++ {
			pointsArray := pathArray[i]
			// first array shall be of size 2 and specify the moveto operator
			if i == 0 && len(pointsArray) == 2 {
				if err := contentStream.MoveTo(pointsArray[0], pointsArray[1]); err != nil {
					return err
				}
				continue
			}
			// entries of length 2 shall be treated as lineto operator
			switch len(pointsArray) {
			case 2:
				if err := contentStream.LineTo(pointsArray[0], pointsArray[1]); err != nil {
					return err
				}
			case 6:
				if err := contentStream.CurveTo(pointsArray[0], pointsArray[1],
					pointsArray[2], pointsArray[3],
					pointsArray[4], pointsArray[5]); err != nil {
					return err
				}
			}
		}
		if err := contentStream.ClosePath(); err != nil {
			return err
		}
	}
	return contentStream.DrawShape(lineWidth, hasStroke, hasBackground)
}

// pathArrayOf returns the path of a polygon annotation, from /Path where there
// is one and from /Vertices otherwise. Java declares it private.
func pathArrayOf(annot *annotation.PDAnnotationPolygon) [][]float32 {
	// PDF 2.0: Path takes priority over Vertices
	pathArray := annot.Path()
	if pathArray != nil {
		return pathArray
	}
	// convert PDF 1.* array to PDF 2.0 array
	verticesArray := annot.Vertices()
	if verticesArray == nil {
		return nil
	}
	points := len(verticesArray) / 2
	pathArray = make([][]float32, points)
	for i := 0; i < points; i++ {
		pathArray[i] = []float32{verticesArray[i*2], verticesArray[i*2+1]}
	}
	return pathArray
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated for
// a polygon annotation.
func (h *PDPolygonAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated for a
// polygon annotation.
func (h *PDPolygonAppearanceHandler) GenerateDownAppearance() error { return nil }

// PDPolylineAppearanceHandler draws a polyline annotation.
//
// Port of PDPolylineAppearanceHandler.
type PDPolylineAppearanceHandler struct {
	PDAbstractAppearanceHandler
}

// NewPDPolylineAppearanceHandler builds a handler for the given annotation.
func NewPDPolylineAppearanceHandler(annot annotation.PDAnnotation) *PDPolylineAppearanceHandler {
	return NewPDPolylineAppearanceHandlerInDocument(annot, nil)
}

// NewPDPolylineAppearanceHandlerInDocument builds one whose streams belong to
// the given document.
func NewPDPolylineAppearanceHandlerInDocument(annot annotation.PDAnnotation,
	document common.COSDocumentLike) *PDPolylineAppearanceHandler {
	h := &PDPolylineAppearanceHandler{}
	h.initAppearanceHandler(h, annot, document)
	return h
}

// GenerateNormalAppearance draws the polyline and its line endings.
func (h *PDPolylineAppearanceHandler) GenerateNormalAppearance() error {
	annot, isPolyline := h.Annotation().(*annotation.PDAnnotationPolyline)
	if !isPolyline {
		panic("handlers: the annotation of a polyline handler is not a polyline")
	}
	rect := annot.Rectangle()
	if rect == nil {
		return nil
	}
	pathsArray := annot.Vertices()
	if len(pathsArray) < 4 {
		return nil
	}
	ab := getAnnotationBorder(annot, annot.BorderStyle())
	lineColor := annot.Color()
	if lineColor == nil || len(lineColor.Components()) == 0 || ab.width == 0 {
		return nil
	}

	// Adjust rectangle even if not empty
	// CTAN-example-Annotations.pdf and pdf_commenting_new.pdf p11
	// TODO in a class structure this should be overridable
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
	// arrow length is 9 * width at about 30 degrees => 10 * width seems to be enough
	rect.SetLowerLeftX(minFloat32(minX-ab.width*10, rect.LowerLeftX()))
	rect.SetLowerLeftY(minFloat32(minY-ab.width*10, rect.LowerLeftY()))
	rect.SetUpperRightX(maxFloat32(maxX+ab.width*10, rect.UpperRightX()))
	rect.SetUpperRightY(maxFloat32(maxY+ab.width*10, rect.UpperRightY()))
	annot.SetRectangle(rect)

	cs, err := h.NormalAppearanceAsContentStream()
	if err != nil {
		slog.Error("handlers: polyline appearance", slog.Any("error", err))
		return nil
	}
	defer cs.Close()

	if err := h.drawPolyline(annot, ab, lineColor, pathsArray, cs); err != nil {
		slog.Error("handlers: polyline appearance", slog.Any("error", err))
	}
	return nil
}

// drawPolyline is the body of the try block Java writes.
func (h *PDPolylineAppearanceHandler) drawPolyline(annot *annotation.PDAnnotationPolyline,
	ab *annotationBorder, lineColor *color.PDColor, pathsArray []float32,
	cs annotation.AppearanceContentStream) error {
	hasBackground, err := cs.SetNonStrokingColorOnDemand(annot.InteriorColor())
	if err != nil {
		return err
	}
	if err := h.SetOpacity(cs, annot.ConstantOpacity()); err != nil {
		return err
	}
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

	startPointEndingStyle := annot.StartPointEndingStyle()
	endPointEndingStyle := annot.EndPointEndingStyle()

	for i := 0; i < len(pathsArray)/2; i++ {
		x := pathsArray[i*2]
		y := pathsArray[i*2+1]
		if i == 0 {
			if shortStyles[startPointEndingStyle] {
				// modify coordinate to shorten the segment
				// https://stackoverflow.com/questions/7740507/extend-a-line-segment-a-specific-distance
				x1 := pathsArray[2]
				y1 := pathsArray[3]
				length := float32(math.Sqrt(math.Pow(float64(x-x1), 2) +
					math.Pow(float64(y-y1), 2)))
				if length != 0 {
					x += (x1 - x) / length * ab.width
					y += (y1 - y) / length * ab.width
				}
			}
			if err := cs.MoveTo(x, y); err != nil {
				return err
			}
			continue
		}
		if i == len(pathsArray)/2-1 && shortStyles[endPointEndingStyle] {
			// modify coordinate to shorten the segment
			// https://stackoverflow.com/questions/7740507/extend-a-line-segment-a-specific-distance
			x0 := pathsArray[len(pathsArray)-4]
			y0 := pathsArray[len(pathsArray)-3]
			length := float32(math.Sqrt(math.Pow(float64(x0-x), 2) +
				math.Pow(float64(y0-y), 2)))
			if length != 0 {
				x -= (x - x0) / length * ab.width
				y -= (y - y0) / length * ab.width
			}
		}
		if err := cs.LineTo(x, y); err != nil {
			return err
		}
	}
	if err := cs.Stroke(); err != nil {
		return err
	}

	// do a transform so that first and last "arms" are imagined flat, like in line handler
	// the alternative would be to apply the transform to the LE shapes directly,
	// which would be more work and produce code difficult to understand

	// paint the styles here and after polyline draw, to avoid line crossing a filled shape
	if startPointEndingStyle != annotation.LENone {
		// check only needed to avoid q cm Q if LE_NONE
		x2 := pathsArray[2]
		y2 := pathsArray[3]
		x1 := pathsArray[0]
		y1 := pathsArray[1]
		if err := cs.SaveGraphicsState(); err != nil {
			return err
		}
		if angledStyles[startPointEndingStyle] {
			angle := math.Atan2(float64(y2-y1), float64(x2-x1))
			if err := cs.Transform(util.RotateInstance(angle, x1, y1)); err != nil {
				return err
			}
		} else if err := cs.Transform(util.TranslateInstance(x1, y1)); err != nil {
			return err
		}
		if err := h.DrawStyle(startPointEndingStyle, cs, 0, 0, ab.width,
			hasStroke, hasBackground, false); err != nil {
			return err
		}
		if err := cs.RestoreGraphicsState(); err != nil {
			return err
		}
	}

	if endPointEndingStyle != annotation.LENone {
		// check only needed to avoid q cm Q if LE_NONE
		x1 := pathsArray[len(pathsArray)-4]
		y1 := pathsArray[len(pathsArray)-3]
		x2 := pathsArray[len(pathsArray)-2]
		y2 := pathsArray[len(pathsArray)-1]
		// save / restore not needed because it's the last one
		if angledStyles[endPointEndingStyle] {
			angle := math.Atan2(float64(y2-y1), float64(x2-x1))
			if err := cs.Transform(util.RotateInstance(angle, x2, y2)); err != nil {
				return err
			}
		} else if err := cs.Transform(util.TranslateInstance(x2, y2)); err != nil {
			return err
		}
		return h.DrawStyle(endPointEndingStyle, cs, 0, 0, ab.width,
			hasStroke, hasBackground, true)
	}
	return nil
}

// GenerateRolloverAppearance does nothing: no rollover appearance generated for
// a polyline annotation.
func (h *PDPolylineAppearanceHandler) GenerateRolloverAppearance() error { return nil }

// GenerateDownAppearance does nothing: no down appearance generated for a
// polyline annotation.
func (h *PDPolylineAppearanceHandler) GenerateDownAppearance() error { return nil }
