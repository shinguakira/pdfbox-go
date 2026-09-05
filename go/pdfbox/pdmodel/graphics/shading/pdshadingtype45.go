package shading

// The two triangle mesh shadings and the base they share.
//
// Port of PDTriangleBasedShadingType, PDShadingType4 and PDShadingType5.

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// pdTriangleBasedShadingType carries what the triangle mesh shadings share.
//
// Port of the package-private abstract PDTriangleBasedShadingType. The two
// abstract members reach the concrete shading through self, since Go embedding
// does not dispatch.
type pdTriangleBasedShadingType struct {
	PDShading

	self triangleBasedShading

	decode                  *cos.Array
	bitsPerCoordinate       int
	bitsPerColorComponent   int
	numberOfColorComponents int
}

// triangleBasedShading is the one abstract method of PDTriangleBasedShadingType.
type triangleBasedShading interface {
	// collectTriangles decomposes the mesh into triangles.
	collectTriangles(xform *geom.AffineTransform, matrix *util.Matrix) ([]*shadedTriangle, error)
}

// initTriangleBased is the PDTriangleBasedShadingType(COSDictionary)
// constructor. The three cached fields start at -1, as Java's do.
func (s *pdTriangleBasedShadingType) initTriangleBased(self triangleBasedShading,
	shadingDictionary *cos.Dictionary, shadingStream *cos.Stream) {
	s.self = self
	if shadingStream != nil {
		s.InitShadingOfStream(shadingStream)
	} else {
		s.InitShadingOf(shadingDictionary)
	}
	s.bitsPerCoordinate = -1
	s.bitsPerColorComponent = -1
	s.numberOfColorComponents = -1
}

// BitsPerComponent returns the /BitsPerComponent entry, -1 where there is none.
func (s *pdTriangleBasedShadingType) BitsPerComponent() int {
	if s.bitsPerColorComponent == -1 {
		s.bitsPerColorComponent = s.Dictionary().GetIntDefault(cos.BitsPerComponent, -1)
		slog.Debug("shading: bitsPerColorComponent", "value", s.bitsPerColorComponent)
	}
	return s.bitsPerColorComponent
}

// SetBitsPerComponent sets the /BitsPerComponent entry.
func (s *pdTriangleBasedShadingType) SetBitsPerComponent(bitsPerComponent int) {
	s.Dictionary().SetInt(cos.BitsPerComponent, bitsPerComponent)
	s.bitsPerColorComponent = bitsPerComponent
}

// BitsPerCoordinate returns the /BitsPerCoordinate entry, -1 where there is
// none.
func (s *pdTriangleBasedShadingType) BitsPerCoordinate() int {
	if s.bitsPerCoordinate == -1 {
		s.bitsPerCoordinate = s.Dictionary().GetIntDefault(cos.BitsPerCoordinate, -1)
		slog.Debug("shading: bitsPerCoordinate",
			"value", math.Pow(2, float64(s.bitsPerCoordinate))-1)
	}
	return s.bitsPerCoordinate
}

// SetBitsPerCoordinate sets the /BitsPerCoordinate entry.
func (s *pdTriangleBasedShadingType) SetBitsPerCoordinate(bitsPerCoordinate int) {
	s.Dictionary().SetInt(cos.BitsPerCoordinate, bitsPerCoordinate)
	s.bitsPerCoordinate = bitsPerCoordinate
}

// NumberOfColorComponents returns how many components a vertex colour has: one
// where the shading has a function, and the colour space's count otherwise.
func (s *pdTriangleBasedShadingType) NumberOfColorComponents() (int, error) {
	if s.numberOfColorComponents != -1 {
		return s.numberOfColorComponents, nil
	}
	shadingFunction, err := s.Function()
	if err != nil {
		return 0, err
	}
	if shadingFunction != nil {
		s.numberOfColorComponents = 1
	} else {
		colorSpace, err := s.ColorSpace()
		if err != nil {
			return 0, err
		}
		s.numberOfColorComponents = colorSpace.NumberOfComponents()
	}
	slog.Debug("shading: numberOfColorComponents", "value", s.numberOfColorComponents)
	return s.numberOfColorComponents, nil
}

// decodeValues is the private getDecodeValues.
func (s *pdTriangleBasedShadingType) decodeValues() *cos.Array {
	if s.decode == nil {
		s.decode = s.Dictionary().GetCOSArray(cos.Decode)
	}
	return s.decode
}

// SetDecodeValues sets the /Decode entry.
func (s *pdTriangleBasedShadingType) SetDecodeValues(decodeValues *cos.Array) {
	s.decode = decodeValues
	s.Dictionary().SetItem(cos.Decode, decodeValues)
}

// DecodeForParameter returns the decode range of one parameter, or nil where
// the /Decode array is too short to hold it.
func (s *pdTriangleBasedShadingType) DecodeForParameter(paramNum int) *common.PDRange {
	decodeValues := s.decodeValues()
	if decodeValues != nil && decodeValues.Size() >= paramNum*2+1 {
		return common.NewPDRangeOfIndex(decodeValues, paramNum)
	}
	return nil
}

// interpolate maps a value out of the source range onto the destination one.
func interpolate(src float32, srcMax int64, dstMin, dstMax float32) float32 {
	return dstMin + (src * (dstMax - dstMin) / float32(srcMax))
}

// readVertex reads one vertex: its coordinates, its colour, and the padding
// that takes the reader back onto a byte boundary.
func (s *pdTriangleBasedShadingType) readVertex(input *bitReader, maxSrcCoord, maxSrcColor int64,
	rangeX, rangeY *common.PDRange, colRangeTab []*common.PDRange,
	matrix *util.Matrix, xform *geom.AffineTransform) (*vertex, error) {
	if s.bitsPerCoordinate <= 0 || s.numberOfColorComponents <= 0 || s.bitsPerColorComponent <= 0 {
		return nil, errors.New("nothing to read, check bitsPerCoordinate, " +
			"numberOfColorComponents and bitsPerColorComponent")
	}
	colorComponentTab := make([]float32, s.numberOfColorComponents)
	x, err := input.readBits(s.bitsPerCoordinate)
	if err != nil {
		return nil, err
	}
	y, err := input.readBits(s.bitsPerCoordinate)
	if err != nil {
		return nil, err
	}
	dstX := interpolate(float32(x), maxSrcCoord, rangeX.Min(), rangeX.Max())
	dstY := interpolate(float32(y), maxSrcCoord, rangeY.Min(), rangeY.Max())
	slog.Debug("shading: coord", "x", fmt.Sprintf("%06X", x), "y", fmt.Sprintf("%06X", y),
		"dstX", dstX, "dstY", dstY)
	p := matrix.TransformPoint(dstX, dstY)
	transformed := xform.Transform(p, p)

	for n := 0; n < s.numberOfColorComponents; n++ {
		colorBits, err := input.readBits(s.bitsPerColorComponent)
		if err != nil {
			return nil, err
		}
		// Java narrows the value to an int before interpolating.
		colorValue := int32(colorBits)
		colorComponentTab[n] = interpolate(float32(colorValue), maxSrcColor,
			colRangeTab[n].Min(), colRangeTab[n].Max())
		slog.Debug("shading: color", "index", n, "raw", colorValue,
			"value", colorComponentTab[n])
	}
	// "Each set of vertex data shall occupy a whole number of bytes.
	// If the total number of bits required is not divisible by 8, the last data
	// byte for each vertex is padded at the end with extra bits, which shall be
	// ignored."
	if bitOffset := input.getBitOffset(); bitOffset != 0 {
		if _, err := input.readBits(8 - bitOffset); err != nil {
			return nil, err
		}
	}
	return newVertex(transformed, colorComponentTab), nil
}

// Bounds returns a rectangle around every triangle of the mesh.
func (s *pdTriangleBasedShadingType) Bounds(xform *geom.AffineTransform,
	matrix *util.Matrix) (*geom.Rectangle2D, error) {
	triangles, err := s.self.collectTriangles(xform, matrix)
	if err != nil {
		return nil, err
	}
	var bounds *geom.Rectangle2D
	for _, shadedTriangle := range triangles {
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
	if bounds == nil {
		// Speeds up files where triangles are empty, e.g. ghostscript file 690425
		return &geom.Rectangle2D{}, nil
	}
	return bounds, nil
}

// meshDecodeRanges reads the decode ranges every mesh shading needs, and
// reports false where the shading has none or they are degenerate -- which is
// where Java returns an empty triangle list.
func (s *pdTriangleBasedShadingType) meshDecodeRanges() (rangeX, rangeY *common.PDRange,
	colRange []*common.PDRange, ok bool, err error) {
	rangeX = s.DecodeForParameter(0)
	rangeY = s.DecodeForParameter(1)
	if rangeX == nil || rangeY == nil || rangeX.Min() == rangeX.Max() ||
		rangeY.Min() == rangeY.Max() {
		return nil, nil, nil, false, nil
	}
	numberOfColorComponents, err := s.NumberOfColorComponents()
	if err != nil {
		return nil, nil, nil, false, err
	}
	colRange = make([]*common.PDRange, numberOfColorComponents)
	for i := range colRange {
		colRange[i] = s.DecodeForParameter(2 + i)
		if colRange[i] == nil {
			return nil, nil, nil, false, errors.New("Range missing in shading /Decode entry")
		}
	}
	return rangeX, rangeY, colRange, true, nil
}

// meshReader returns a bit reader over the shading stream, and reports false
// where the shading is not a stream -- Java's `!(dict instanceof COSStream)`.
func (s *pdTriangleBasedShadingType) meshReader() (*bitReader, bool, error) {
	stream := s.Stream()
	if stream == nil {
		return nil, false, nil
	}
	// Java wraps the stream in a MemoryCacheImageInputStream, which does not
	// close what it wraps; the port reads it into a bit reader instead.
	reader, err := stream.CreateReader()
	if err != nil {
		return nil, false, err
	}
	return newBitReader(reader), true, nil
}

// maxSourceValues are the two ceilings a vertex is interpolated out of.
func (s *pdTriangleBasedShadingType) maxSourceValues() (maxSrcCoord, maxSrcColor int64) {
	return int64(math.Pow(2, float64(s.BitsPerCoordinate()))) - 1,
		int64(math.Pow(2, float64(s.BitsPerComponent()))) - 1
}

// PDShadingType4 is a free-form Gouraud-shaded triangle mesh.
//
// Port of PDShadingType4.
type PDShadingType4 struct {
	pdTriangleBasedShadingType
}

var _ Shading = (*PDShadingType4)(nil)

// NewPDShadingType4 creates a free-form triangle mesh shading over the given
// dictionary.
func NewPDShadingType4(shadingDictionary *cos.Dictionary) *PDShadingType4 {
	return newPDShadingType4(shadingDictionary, nil)
}

// newPDShadingType4 is NewPDShadingType4 with the stream the dictionary came
// from, where there was one.
func newPDShadingType4(shadingDictionary *cos.Dictionary, shadingStream *cos.Stream) *PDShadingType4 {
	s := &PDShadingType4{}
	s.initTriangleBased(s, shadingDictionary, shadingStream)
	return s
}

// ShadingType returns ShadingType4.
func (s *PDShadingType4) ShadingType() int { return ShadingType4 }

// BitsPerFlag returns the /BitsPerFlag entry, -1 where there is none.
func (s *PDShadingType4) BitsPerFlag() int {
	return s.Dictionary().GetIntDefault(cos.BitsPerFlag, -1)
}

// SetBitsPerFlag sets the /BitsPerFlag entry.
func (s *PDShadingType4) SetBitsPerFlag(bitsPerFlag int) {
	s.Dictionary().SetInt(cos.BitsPerFlag, bitsPerFlag)
}

// collectTriangles reads the mesh, whose vertices carry a flag saying how each
// triangle joins the one before it.
func (s *PDShadingType4) collectTriangles(xform *geom.AffineTransform,
	matrix *util.Matrix) ([]*shadedTriangle, error) {
	bitsPerFlag := s.BitsPerFlag()
	input, isStream, err := s.meshReader()
	if err != nil || !isStream {
		return nil, err
	}
	rangeX, rangeY, colRange, ok, err := s.meshDecodeRanges()
	if err != nil || !ok {
		return nil, err
	}
	list := []*shadedTriangle{}
	maxSrcCoord, maxSrcColor := s.maxSourceValues()

	flag := int64(0)
	if flag, err = input.readBits(bitsPerFlag); err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, err
		}
		slog.Error("shading: reading the first flag", slog.Any("error", err))
	}
	flag &= 3

	for eof := false; !eof; {
		readVertex := func() (*vertex, error) {
			return s.readVertex(input, maxSrcCoord, maxSrcColor, rangeX, rangeY,
				colRange, matrix, xform)
		}
		switch flag {
		case 0:
			p0, err := readVertex()
			if err != nil {
				if errors.Is(err, io.EOF) {
					eof = true
					continue
				}
				return nil, err
			}
			next, err := input.readBits(bitsPerFlag)
			if err != nil {
				if errors.Is(err, io.EOF) {
					eof = true
					continue
				}
				return nil, err
			}
			flag = next & 3
			if flag != 0 {
				slog.Error("shading: bad triangle", "flag", flag)
			}
			p1, err := readVertex()
			if err != nil {
				if errors.Is(err, io.EOF) {
					eof = true
					continue
				}
				return nil, err
			}
			// Java reads the next flag here without masking or assigning it,
			// then tests the flag it read before p1 -- so the second check
			// repeats the first rather than looking at this one.
			if _, err := input.readBits(bitsPerFlag); err != nil {
				if errors.Is(err, io.EOF) {
					eof = true
					continue
				}
				return nil, err
			}
			if flag != 0 {
				slog.Error("shading: bad triangle", "flag", flag)
			}
			p2, err := readVertex()
			if err != nil {
				if errors.Is(err, io.EOF) {
					eof = true
					continue
				}
				return nil, err
			}
			list = append(list, newShadedTriangle(
				[3]geom.Point2D{p0.point, p1.point, p2.point},
				[3][]float32{p0.color, p1.color, p2.color}))
			next, err = input.readBits(bitsPerFlag)
			if err != nil {
				if errors.Is(err, io.EOF) {
					eof = true
					continue
				}
				return nil, err
			}
			flag = next & 3
		case 1, 2:
			lastIndex := len(list) - 1
			if lastIndex < 0 {
				slog.Error("shading: broken data stream, aborting", "triangles", len(list))
				eof = true
				continue
			}
			preTri := list[lastIndex]
			p2, err := readVertex()
			if err != nil {
				if errors.Is(err, io.EOF) {
					eof = true
					continue
				}
				return nil, err
			}
			firstCorner, firstColor := preTri.corner[0], preTri.color[0]
			if flag == 1 {
				firstCorner, firstColor = preTri.corner[1], preTri.color[1]
			}
			list = append(list, newShadedTriangle(
				[3]geom.Point2D{firstCorner, preTri.corner[2], p2.point},
				[3][]float32{firstColor, preTri.color[2], p2.color}))
			next, err := input.readBits(bitsPerFlag)
			if err != nil {
				if errors.Is(err, io.EOF) {
					eof = true
					continue
				}
				return nil, err
			}
			flag = next & 3
		default:
			slog.Warn("shading: bad flag, aborting", "flag", flag)
			eof = true
		}
	}
	return list, nil
}

// PDShadingType5 is a lattice-form Gouraud-shaded triangle mesh.
//
// Port of PDShadingType5.
type PDShadingType5 struct {
	pdTriangleBasedShadingType
}

var _ Shading = (*PDShadingType5)(nil)

// NewPDShadingType5 creates a lattice-form triangle mesh shading over the given
// dictionary.
func NewPDShadingType5(shadingDictionary *cos.Dictionary) *PDShadingType5 {
	return newPDShadingType5(shadingDictionary, nil)
}

// newPDShadingType5 is NewPDShadingType5 with the stream the dictionary came
// from, where there was one.
func newPDShadingType5(shadingDictionary *cos.Dictionary, shadingStream *cos.Stream) *PDShadingType5 {
	s := &PDShadingType5{}
	s.initTriangleBased(s, shadingDictionary, shadingStream)
	return s
}

// ShadingType returns ShadingType5.
func (s *PDShadingType5) ShadingType() int { return ShadingType5 }

// VerticesPerRow returns the /VerticesPerRow entry, -1 where there is none.
func (s *PDShadingType5) VerticesPerRow() int {
	return s.Dictionary().GetIntDefault(cos.VerticesPerRow, -1)
}

// SetVerticesPerRow sets the /VerticesPerRow entry.
func (s *PDShadingType5) SetVerticesPerRow(verticesPerRow int) {
	s.Dictionary().SetInt(cos.VerticesPerRow, verticesPerRow)
}

// collectTriangles reads the lattice and cuts each cell of it into two
// triangles.
func (s *PDShadingType5) collectTriangles(xform *geom.AffineTransform,
	matrix *util.Matrix) ([]*shadedTriangle, error) {
	input, isStream, err := s.meshReader()
	if err != nil || !isStream {
		return nil, err
	}
	rangeX, rangeY, colRange, ok, err := s.meshDecodeRanges()
	if err != nil || !ok {
		return nil, err
	}
	numPerRow := s.VerticesPerRow()
	if numPerRow < 1 {
		// Java divides by this below; a lattice with no vertices per row has
		// no rows either, and the row check answers the empty list.
		return nil, nil
	}
	var vlist []*vertex
	maxSrcCoord, maxSrcColor := s.maxSourceValues()
	for {
		p, err := s.readVertex(input, maxSrcCoord, maxSrcColor, rangeX, rangeY, colRange,
			matrix, xform)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		vlist = append(vlist, p)
	}
	rowNum := len(vlist) / numPerRow
	if rowNum < 2 {
		// must have at least two rows; if not, return empty list
		return nil, nil
	}
	latticeArray := make([][]*vertex, rowNum)
	for i := 0; i < rowNum; i++ {
		latticeArray[i] = make([]*vertex, numPerRow)
		for j := 0; j < numPerRow; j++ {
			latticeArray[i][j] = vlist[i*numPerRow+j]
		}
	}
	return createShadedTriangleList(rowNum, numPerRow, latticeArray), nil
}

// createShadedTriangleList cuts each cell of the lattice into two triangles.
//
// Port of the private PDShadingType5.createShadedTriangleList.
func createShadedTriangleList(rowNum, numPerRow int, latticeArray [][]*vertex) []*shadedTriangle {
	list := make([]*shadedTriangle, 0, (rowNum-1)*(numPerRow-1))
	for i := 0; i < rowNum-1; i++ {
		for j := 0; j < numPerRow-1; j++ {
			vertex1 := latticeArray[i][j]
			vertex2 := latticeArray[i][j+1]
			vertex3 := latticeArray[i+1][j]
			vertex4 := latticeArray[i+1][j+1]
			list = append(list, newShadedTriangle(
				[3]geom.Point2D{vertex1.point, vertex2.point, vertex3.point},
				[3][]float32{vertex1.color, vertex2.color, vertex3.color}))
			list = append(list, newShadedTriangle(
				[3]geom.Point2D{vertex2.point, vertex3.point, vertex4.point},
				[3][]float32{vertex2.color, vertex3.color, vertex4.color}))
		}
	}
	return list
}
