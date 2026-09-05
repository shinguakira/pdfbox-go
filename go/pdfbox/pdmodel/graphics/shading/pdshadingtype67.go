package shading

// The two patch mesh shadings and the base they share.
//
// Port of PDMeshBasedShadingType, PDShadingType6 and PDShadingType7.

import (
	"errors"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// pdMeshBasedShadingType carries what the two patch mesh shadings share.
//
// Port of the package-private abstract PDMeshBasedShadingType, which extends
// PDShadingType4 for its flag reading and its vertex decoding.
type pdMeshBasedShadingType struct {
	PDShadingType4

	self meshBasedShading
}

// meshBasedShading is the one abstract method of PDMeshBasedShadingType that
// is not getBounds.
type meshBasedShading interface {
	// generatePatch builds the patch this shading type makes of the control
	// points and corner colours just read.
	generatePatch(points []geom.Point2D, color [][]float32) patchWithTriangles
}

// patchWithTriangles is what generatePatch answers: a patch, read for the
// triangles it decomposed into and for the edge a following patch inherits.
type patchWithTriangles interface {
	patchShape

	triangles() []*shadedTriangle
	flag1Color() [][]float32
	flag2Color() [][]float32
	flag3Color() [][]float32
}

// triangles returns the triangles the patch decomposed into.
func (p *patch) triangles() []*shadedTriangle { return p.listOfTriangles }

// flag1Color, flag2Color and flag3Color expose the inherited colours under the
// names patchWithTriangles asks for; Java reaches the protected methods
// directly through the Patch reference.
func (p *patch) flag1Color() [][]float32 { return p.getFlag1Color() }
func (p *patch) flag2Color() [][]float32 { return p.getFlag2Color() }
func (p *patch) flag3Color() [][]float32 { return p.getFlag3Color() }

// initMeshBased is the PDMeshBasedShadingType(COSDictionary) constructor.
func (s *pdMeshBasedShadingType) initMeshBased(self meshBasedShading,
	triangleSelf triangleBasedShading, shadingDictionary *cos.Dictionary,
	shadingStream *cos.Stream) {
	s.self = self
	s.initTriangleBased(triangleSelf, shadingDictionary, shadingStream)
}

// collectPatches reads the mesh as a run of patches, each carrying a flag that
// says which edge of the one before it the next one continues from.
func (s *pdMeshBasedShadingType) collectPatches(xform *geom.AffineTransform,
	matrix *util.Matrix, controlPoints int) ([]patchWithTriangles, error) {
	input, isStream, err := s.meshReader()
	if err != nil || !isStream {
		return nil, err
	}
	rangeX, rangeY, colRange, ok, err := s.meshDecodeRanges()
	if err != nil || !ok {
		return nil, err
	}
	bitsPerFlag := s.BitsPerFlag()
	list := []patchWithTriangles{}
	maxSrcCoord, maxSrcColor := s.maxSourceValues()

	implicitEdge := make([]geom.Point2D, 4)
	implicitCornerColor := [][]float32{
		make([]float32, len(colRange)),
		make([]float32, len(colRange)),
	}
	flag, err := input.readBits(bitsPerFlag)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, err
		}
		slog.Error("shading: reading the first flag", slog.Any("error", err))
		return list, nil
	}
	flag &= 3

	for eof := false; !eof; {
		isFree := flag == 0
		current, err := s.readPatch(input, isFree, implicitEdge, implicitCornerColor,
			maxSrcCoord, maxSrcColor, rangeX, rangeY, colRange, matrix, xform, controlPoints)
		if err != nil {
			if errors.Is(err, io.EOF) {
				eof = true
				continue
			}
			return nil, err
		}
		if current == nil {
			break
		}
		list = append(list, current)
		next, err := input.readBits(bitsPerFlag)
		if err != nil {
			if errors.Is(err, io.EOF) {
				eof = true
				continue
			}
			return nil, err
		}
		flag = next & 3
		switch flag {
		case 0:
		case 1:
			implicitEdge = current.getFlag1Edge()
			implicitCornerColor = current.flag1Color()
		case 2:
			implicitEdge = current.getFlag2Edge()
			implicitCornerColor = current.flag2Color()
		case 3:
			implicitEdge = current.getFlag3Edge()
			implicitCornerColor = current.flag3Color()
		default:
			slog.Warn("shading: bad flag", "flag", flag)
		}
	}
	return list, nil
}

// readPatch reads one patch: the control points it does not inherit, and the
// corner colours it does not inherit.
//
// It answers nil where the stream ends part way through, which is what Java's
// catch of EOFException does.
func (s *pdMeshBasedShadingType) readPatch(input *bitReader, isFree bool,
	implicitEdge []geom.Point2D, implicitCornerColor [][]float32,
	maxSrcCoord, maxSrcColor int64, rangeX, rangeY *common.PDRange,
	colRange []*common.PDRange, matrix *util.Matrix, xform *geom.AffineTransform,
	controlPoints int) (patchWithTriangles, error) {
	numberOfColorComponents, err := s.NumberOfColorComponents()
	if err != nil {
		return nil, err
	}
	color := make([][]float32, 4)
	for i := range color {
		color[i] = make([]float32, numberOfColorComponents)
	}
	points := make([]geom.Point2D, controlPoints)
	pStart := 4
	cStart := 2
	if isFree {
		pStart = 0
		cStart = 0
	} else {
		points[0] = implicitEdge[0]
		points[1] = implicitEdge[1]
		points[2] = implicitEdge[2]
		points[3] = implicitEdge[3]
		for i := 0; i < numberOfColorComponents; i++ {
			color[0][i] = implicitCornerColor[0][i]
			color[1][i] = implicitCornerColor[1][i]
		}
	}

	bitsPerCoordinate := s.BitsPerCoordinate()
	for i := pStart; i < controlPoints; i++ {
		x, err := input.readBits(bitsPerCoordinate)
		if err != nil {
			return endOfPatch(err)
		}
		y, err := input.readBits(bitsPerCoordinate)
		if err != nil {
			return endOfPatch(err)
		}
		px := interpolate(float32(x), maxSrcCoord, rangeX.Min(), rangeX.Max())
		py := interpolate(float32(y), maxSrcCoord, rangeY.Min(), rangeY.Max())
		p := matrix.TransformPoint(px, py)
		points[i] = xform.Transform(p, p)
	}
	bitsPerComponent := s.BitsPerComponent()
	for i := cStart; i < 4; i++ {
		for j := 0; j < numberOfColorComponents; j++ {
			c, err := input.readBits(bitsPerComponent)
			if err != nil {
				return endOfPatch(err)
			}
			color[i][j] = interpolate(float32(c), maxSrcColor,
				colRange[j].Min(), colRange[j].Max())
		}
	}
	return s.self.generatePatch(points, color), nil
}

// endOfPatch turns the end of the stream into the nil patch Java answers, and
// leaves any other failure alone.
//
// Java catches EOFException inside readPatch and returns null, which
// collectPatches reads as the end of the mesh rather than as a failure.
func endOfPatch(err error) (patchWithTriangles, error) {
	if errors.Is(err, io.EOF) {
		slog.Debug("shading: EOF", slog.Any("error", err))
		return nil, nil
	}
	return nil, err
}

// boundsOfPatches returns a rectangle around every triangle of every patch.
//
// Port of the package-private getBounds(AffineTransform, Matrix, int).
func (s *pdMeshBasedShadingType) boundsOfPatches(xform *geom.AffineTransform,
	matrix *util.Matrix, controlPoints int) (*geom.Rectangle2D, error) {
	patches, err := s.collectPatches(xform, matrix, controlPoints)
	if err != nil {
		return nil, err
	}
	var bounds *geom.Rectangle2D
	for _, p := range patches {
		for _, shadedTriangle := range p.triangles() {
			if bounds == nil {
				bounds = &geom.Rectangle2D{
					X: shadedTriangle.corner[0].X(),
					Y: shadedTriangle.corner[0].Y(),
				}
			}
			bounds.Add(shadedTriangle.corner[0].X(), shadedTriangle.corner[0].Y())
			bounds.Add(shadedTriangle.corner[1].X(), shadedTriangle.corner[1].Y())
			bounds.Add(shadedTriangle.corner[2].X(), shadedTriangle.corner[2].Y())
		}
	}
	// Java answers null where there were no patches, unlike the triangle based
	// shadings, which answer an empty rectangle.
	return bounds, nil
}

// collectTriangles returns every triangle of every patch, which is what a patch
// mesh contributes to the triangle machinery it inherits.
//
// Java does not override collectTriangles in PDMeshBasedShadingType: it
// overrides getBounds instead and reaches the patches directly. The port
// declares it because the triangle based type asks for it through an interface,
// and it answers the same triangles the bounds are taken from.
func (s *pdMeshBasedShadingType) collectTrianglesOfPatches(xform *geom.AffineTransform,
	matrix *util.Matrix, controlPoints int) ([]*shadedTriangle, error) {
	patches, err := s.collectPatches(xform, matrix, controlPoints)
	if err != nil {
		return nil, err
	}
	var list []*shadedTriangle
	for _, p := range patches {
		list = append(list, p.triangles()...)
	}
	return list, nil
}

// coonsControlPoints and tensorControlPoints are how many control points a
// free-form patch of each type carries.
const (
	coonsControlPoints  = 12
	tensorControlPoints = 16
)

// PDShadingType6 is a Coons patch mesh.
//
// Port of PDShadingType6.
type PDShadingType6 struct {
	pdMeshBasedShadingType
}

var _ Shading = (*PDShadingType6)(nil)

// NewPDShadingType6 creates a Coons patch mesh shading over the given
// dictionary.
func NewPDShadingType6(shadingDictionary *cos.Dictionary) *PDShadingType6 {
	return newPDShadingType6(shadingDictionary, nil)
}

// newPDShadingType6 is NewPDShadingType6 with the stream the dictionary came
// from, where there was one.
func newPDShadingType6(shadingDictionary *cos.Dictionary, shadingStream *cos.Stream) *PDShadingType6 {
	s := &PDShadingType6{}
	s.initMeshBased(s, s, shadingDictionary, shadingStream)
	return s
}

// ShadingType returns ShadingType6.
func (s *PDShadingType6) ShadingType() int { return ShadingType6 }

// generatePatch builds a Coons patch.
func (s *PDShadingType6) generatePatch(points []geom.Point2D, color [][]float32) patchWithTriangles {
	return newCoonsPatch(points, color)
}

// collectTriangles returns every triangle of every Coons patch.
func (s *PDShadingType6) collectTriangles(xform *geom.AffineTransform,
	matrix *util.Matrix) ([]*shadedTriangle, error) {
	return s.collectTrianglesOfPatches(xform, matrix, coonsControlPoints)
}

// Bounds returns a rectangle around every patch of the mesh.
func (s *PDShadingType6) Bounds(xform *geom.AffineTransform,
	matrix *util.Matrix) (*geom.Rectangle2D, error) {
	return s.boundsOfPatches(xform, matrix, coonsControlPoints)
}

// PDShadingType7 is a tensor-product patch mesh.
//
// Port of PDShadingType7.
type PDShadingType7 struct {
	pdMeshBasedShadingType
}

var _ Shading = (*PDShadingType7)(nil)

// NewPDShadingType7 creates a tensor-product patch mesh shading over the given
// dictionary.
func NewPDShadingType7(shadingDictionary *cos.Dictionary) *PDShadingType7 {
	return newPDShadingType7(shadingDictionary, nil)
}

// newPDShadingType7 is NewPDShadingType7 with the stream the dictionary came
// from, where there was one.
func newPDShadingType7(shadingDictionary *cos.Dictionary, shadingStream *cos.Stream) *PDShadingType7 {
	s := &PDShadingType7{}
	s.initMeshBased(s, s, shadingDictionary, shadingStream)
	return s
}

// ShadingType returns ShadingType7.
func (s *PDShadingType7) ShadingType() int { return ShadingType7 }

// generatePatch builds a tensor-product patch.
func (s *PDShadingType7) generatePatch(points []geom.Point2D, color [][]float32) patchWithTriangles {
	return newTensorPatch(points, color)
}

// collectTriangles returns every triangle of every tensor patch.
func (s *PDShadingType7) collectTriangles(xform *geom.AffineTransform,
	matrix *util.Matrix) ([]*shadedTriangle, error) {
	return s.collectTrianglesOfPatches(xform, matrix, tensorControlPoints)
}

// Bounds returns a rectangle around every patch of the mesh.
func (s *PDShadingType7) Bounds(xform *geom.AffineTransform,
	matrix *util.Matrix) (*geom.Rectangle2D, error) {
	return s.boundsOfPatches(xform, matrix, tensorControlPoints)
}
