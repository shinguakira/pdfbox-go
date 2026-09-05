package shading

// A3: PDFBox has no test for this package, so these are written from the Java
// source and from the shading sections of PDF32000_2008.pdf, not from running
// the port. Every expected value below is worked out by hand from the decode
// arithmetic, which is why the fixtures use eight bit fields and a decode range
// that maps one to one -- the numbers stay checkable.
//
// This is the comparison strategy A5 settles on: with no rasteriser there is no
// image to diff, so what is asserted is the arithmetic -- which triangles a mesh
// decomposes into, with what corners and what colours, and what rectangle
// bounds them.

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// TestNewPDShadingDispatch checks that each /ShadingType builds its own type,
// which is the switch of PDShading.create.
func TestNewPDShadingDispatch(t *testing.T) {
	for _, want := range []struct {
		shadingType int
		mesh        bool
	}{
		{ShadingType1, false},
		{ShadingType2, false},
		{ShadingType3, false},
		{ShadingType4, true},
		{ShadingType5, true},
		{ShadingType6, true},
		{ShadingType7, true},
	} {
		dict := cos.NewDictionary()
		dict.SetInt(cos.ShadingType, want.shadingType)
		created, err := NewPDShading(dict)
		if err != nil {
			t.Errorf("NewPDShading(type %d) = %v", want.shadingType, err)
			continue
		}
		if got := created.ShadingType(); got != want.shadingType {
			t.Errorf("ShadingType() = %d, want %d", got, want.shadingType)
		}
	}
}

// TestNewPDShadingRefusesAnUnknownType is the default arm of the same switch,
// which Java throws from.
func TestNewPDShadingRefusesAnUnknownType(t *testing.T) {
	dict := cos.NewDictionary()
	dict.SetInt(cos.ShadingType, 8)
	if _, err := NewPDShading(dict); err == nil {
		t.Error("NewPDShading(type 8) = nil error, want one")
	}
	// A dictionary with no /ShadingType reads as type 0, which is also unknown.
	if _, err := NewPDShading(cos.NewDictionary()); err == nil {
		t.Error("NewPDShading(no type) = nil error, want one")
	}
}

// TestEvalFunctionClampsToTheValidRange is the loop at the end of
// evalFunction: "If the value returned by the function for a given colour
// component is out of range, it shall be adjusted to the nearest valid value."
//
// The function is a type 2 exponential with N=1, so it interpolates linearly
// from C0 to C1; C0 and C1 are set outside 0..1 so that the clamping shows.
func TestEvalFunctionClampsToTheValidRange(t *testing.T) {
	exponential := cos.NewDictionary()
	exponential.SetInt(cos.FunctionType, 2)
	exponential.SetItem(cos.Domain, arrayOfFloats(0, 1))
	exponential.SetItem(cos.C0, arrayOfFloats(-2, 0.25))
	exponential.SetItem(cos.C1, arrayOfFloats(3, 0.75))
	exponential.SetInt(cos.N, 1)

	dict := cos.NewDictionary()
	dict.SetInt(cos.ShadingType, ShadingType2)
	dict.SetItem(cos.Function, exponential)
	created, err := NewPDShading(dict)
	if err != nil {
		t.Fatal(err)
	}
	shading := created.(*PDShadingType2)

	for _, c := range []struct {
		input float32
		want  []float32
	}{
		// At t=0 the function answers C0, whose first component is -2 and
		// clamps to 0; the second is already in range.
		{0, []float32{0, 0.25}},
		// At t=1 it answers C1, whose first component is 3 and clamps to 1.
		{1, []float32{1, 0.75}},
		// Halfway it answers (-2+3)/2 = 0.5 and (0.25+0.75)/2 = 0.5, neither
		// of which needs clamping.
		{0.5, []float32{0.5, 0.5}},
	} {
		got, err := shading.EvalFunction(c.input)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(c.want) {
			t.Fatalf("EvalFunction(%v) = %v, want %v", c.input, got, c.want)
		}
		for i := range got {
			if math.Abs(float64(got[i]-c.want[i])) > 1e-6 {
				t.Errorf("EvalFunction(%v)[%d] = %v, want %v", c.input, i, got[i], c.want[i])
			}
		}
	}
}

// TestFreeFormTriangleMesh reads a type 4 mesh holding one triangle.
//
// The fixture is built so the decode arithmetic is the identity: eight bit
// coordinates with a decode range of 0 to 255 make interpolate(x, 255, 0, 255)
// answer x, and eight bit colour components with a range of 0 to 1 make it
// answer c/255. The matrix and the transform are both the identity, so the
// corners are exactly the numbers written into the stream.
func TestFreeFormTriangleMesh(t *testing.T) {
	data := []byte{
		0 /* flag */, 0, 0 /* x, y */, 255, 0, 0, /* red */
		0 /* flag */, 100, 0 /* x, y */, 0, 255, 0, /* green */
		0 /* flag */, 0, 100 /* x, y */, 0, 0, 255, /* blue */
	}
	shading := meshShading(t, ShadingType4, data)

	triangles, err := shading.(*PDShadingType4).collectTriangles(
		geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	if len(triangles) != 1 {
		t.Fatalf("collectTriangles() = %d triangles, want 1", len(triangles))
	}
	assertTriangle(t, triangles[0],
		[3][2]float64{{0, 0}, {100, 0}, {0, 100}},
		[3][]float32{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}})

	bounds, err := shading.Bounds(geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	assertRectangle(t, "type 4 bounds", bounds, 0, 0, 100, 100)
}

// TestFreeFormTriangleMeshChainsOnFlagOne checks the second arm of the flag
// switch: a triangle with flag 1 keeps the second and third corners of the one
// before it and reads only its own third vertex.
func TestFreeFormTriangleMeshChainsOnFlagOne(t *testing.T) {
	data := []byte{
		0, 0, 0, 255, 0, 0,
		0, 100, 0, 255, 0, 0,
		0, 0, 100, 255, 0, 0,
		// flag 1: corners become previous[1], previous[2] and this vertex.
		1, 100, 100, 255, 0, 0,
	}
	shading := meshShading(t, ShadingType4, data)

	triangles, err := shading.(*PDShadingType4).collectTriangles(
		geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	if len(triangles) != 2 {
		t.Fatalf("collectTriangles() = %d triangles, want 2", len(triangles))
	}
	assertTriangle(t, triangles[1],
		[3][2]float64{{100, 0}, {0, 100}, {100, 100}},
		[3][]float32{{1, 0, 0}, {1, 0, 0}, {1, 0, 0}})
}

// TestLatticeFormTriangleMesh reads a type 5 mesh: two rows of two vertices,
// which is one cell and so two triangles.
func TestLatticeFormTriangleMesh(t *testing.T) {
	data := []byte{
		// row 0
		0, 0, 255, 0, 0,
		100, 0, 255, 0, 0,
		// row 1
		0, 100, 255, 0, 0,
		100, 100, 255, 0, 0,
	}
	dict, stream := meshDictionary(t, ShadingType5, data)
	dict.SetInt(cos.VerticesPerRow, 2)
	created, err := NewPDShading(stream)
	if err != nil {
		t.Fatal(err)
	}

	triangles, err := created.(*PDShadingType5).collectTriangles(
		geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	if len(triangles) != 2 {
		t.Fatalf("collectTriangles() = %d triangles, want 2", len(triangles))
	}
	// The cell is cut as (v1, v2, v3) and (v2, v3, v4), which is what
	// createShadedTriangleList does.
	assertTriangle(t, triangles[0],
		[3][2]float64{{0, 0}, {100, 0}, {0, 100}},
		[3][]float32{{1, 0, 0}, {1, 0, 0}, {1, 0, 0}})
	assertTriangle(t, triangles[1],
		[3][2]float64{{100, 0}, {0, 100}, {100, 100}},
		[3][]float32{{1, 0, 0}, {1, 0, 0}, {1, 0, 0}})

	bounds, err := created.Bounds(geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	assertRectangle(t, "type 5 bounds", bounds, 0, 0, 100, 100)
}

// TestLatticeFormNeedsTwoRows is the guard in collectTriangles: a lattice with
// fewer than two rows has no cells and so no triangles.
func TestLatticeFormNeedsTwoRows(t *testing.T) {
	data := []byte{
		0, 0, 255, 0, 0,
		100, 0, 255, 0, 0,
	}
	dict, stream := meshDictionary(t, ShadingType5, data)
	dict.SetInt(cos.VerticesPerRow, 2)
	created, err := NewPDShading(stream)
	if err != nil {
		t.Fatal(err)
	}
	triangles, err := created.(*PDShadingType5).collectTriangles(
		geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	if len(triangles) != 0 {
		t.Errorf("collectTriangles() = %d triangles, want none", len(triangles))
	}
}

// TestCoonsPatchMesh reads a type 6 mesh holding one square patch.
//
// The twelve control points run round the boundary from corner p0, and
// reshapeControlPoints reads the corners as p0, p3, p6 and p9. The four edges
// are straight, so the patch is the square (0,0) to (100,100) and every
// triangle of it must lie inside that square.
func TestCoonsPatchMesh(t *testing.T) {
	data := []byte{0} // free patch
	for _, p := range [][2]byte{
		{0, 0}, {0, 33}, {0, 67}, // p0 corner 1, controls up the left edge
		{0, 100},             // p3 corner 2
		{33, 100}, {67, 100}, // controls along the top
		{100, 100},           // p6 corner 3
		{100, 67}, {100, 33}, // controls down the right edge
		{100, 0},         // p9 corner 4
		{67, 0}, {33, 0}, // controls back along the bottom
	} {
		data = append(data, p[0], p[1])
	}
	for i := 0; i < 4; i++ {
		data = append(data, 255, 0, 0) // every corner red
	}
	shading := meshShading(t, ShadingType6, data)

	triangles, err := shading.(*PDShadingType6).collectTriangles(
		geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	if len(triangles) == 0 {
		t.Fatal("collectTriangles() = none, want the triangles of one patch")
	}
	for i, triangle := range triangles {
		for _, corner := range triangle.corner {
			if corner.X() < -0.5 || corner.X() > 100.5 ||
				corner.Y() < -0.5 || corner.Y() > 100.5 {
				t.Fatalf("triangle %d has a corner at (%g, %g), outside the patch",
					i, corner.X(), corner.Y())
			}
		}
		// Every corner colour is red, so bilinear interpolation of them is red
		// everywhere.
		for _, color := range triangle.color {
			if len(color) != 3 || color[0] != 1 || color[1] != 0 || color[2] != 0 {
				t.Fatalf("triangle %d has colour %v, want red", i, color)
			}
		}
	}

	bounds, err := shading.Bounds(geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	assertRectangle(t, "type 6 bounds", bounds, 0, 0, 100, 100)
}

// TestTensorPatchMesh reads a type 7 mesh: the same square, with the four inner
// control points a tensor patch adds.
//
// reshapeControlPoints puts tcp[12..15] at the inner positions, so those four
// are placed inside the square; the surface is then the same square.
func TestTensorPatchMesh(t *testing.T) {
	data := []byte{0} // free patch
	for _, p := range [][2]byte{
		{0, 0}, {0, 33}, {0, 67},
		{0, 100},
		{33, 100}, {67, 100},
		{100, 100},
		{100, 67}, {100, 33},
		{100, 0},
		{67, 0}, {33, 0},
		// the four inner control points
		{33, 33}, {33, 67}, {67, 67}, {67, 33},
	} {
		data = append(data, p[0], p[1])
	}
	for i := 0; i < 4; i++ {
		data = append(data, 255, 0, 0)
	}
	shading := meshShading(t, ShadingType7, data)

	bounds, err := shading.Bounds(geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	if bounds == nil {
		t.Fatal("Bounds() = nil, want the square the patch covers")
	}
	// The tensor surface stays within the convex hull of its control points,
	// which is the square.
	if bounds.X < -0.5 || bounds.Y < -0.5 ||
		bounds.MaxX() > 100.5 || bounds.MaxY() > 100.5 {
		t.Errorf("Bounds() = %v, want it inside the square", bounds)
	}
}

// TestMeshOfAPlainDictionaryHasNoTriangles is Java's
// `!(dict instanceof COSStream)` guard: a mesh shading whose dictionary is not
// a stream has no vertex data and so no triangles.
func TestMeshOfAPlainDictionaryHasNoTriangles(t *testing.T) {
	dict := cos.NewDictionary()
	dict.SetInt(cos.ShadingType, ShadingType4)
	created, err := NewPDShading(dict)
	if err != nil {
		t.Fatal(err)
	}
	triangles, err := created.(*PDShadingType4).collectTriangles(
		geom.NewIdentityTransform(), util.NewMatrix())
	if err != nil {
		t.Fatal(err)
	}
	if len(triangles) != 0 {
		t.Errorf("collectTriangles() = %d, want none", len(triangles))
	}
}

// TestCubicBezierCurveSampling checks the curve sampler at two levels. A level
// of n samples 2^n + 1 points, and the first and last are always the two end
// control points.
func TestCubicBezierCurveSampling(t *testing.T) {
	control := []geom.Point2D{
		geom.NewPointDouble(0, 0),
		geom.NewPointDouble(0, 100),
		geom.NewPointDouble(100, 100),
		geom.NewPointDouble(100, 0),
	}
	for _, c := range []struct{ level, want int }{{0, 2}, {1, 3}, {3, 9}} {
		curve := newCubicBezierCurve(control, c.level).CubicBezierCurve()
		if len(curve) != c.want {
			t.Errorf("level %d sampled %d points, want %d", c.level, len(curve), c.want)
			continue
		}
		if curve[0].X() != 0 || curve[0].Y() != 0 {
			t.Errorf("level %d starts at (%g, %g), want the first control point",
				c.level, curve[0].X(), curve[0].Y())
		}
		last := curve[len(curve)-1]
		if math.Abs(last.X()-100) > 1e-9 || math.Abs(last.Y()) > 1e-9 {
			t.Errorf("level %d ends at (%g, %g), want the last control point",
				c.level, last.X(), last.Y())
		}
	}
	// At level 1 the middle sample is the curve at t = 0.5, which for these
	// control points is (50, 75) by the cubic Bernstein weights 1/8, 3/8, 3/8,
	// 1/8: x = 3/8*0 + 3/8*100 + 1/8*100 = 50, y = 3/8*100 + 3/8*100 = 75.
	middle := newCubicBezierCurve(control, 1).CubicBezierCurve()[1]
	if math.Abs(middle.X()-50) > 1e-9 || math.Abs(middle.Y()-75) > 1e-9 {
		t.Errorf("the midpoint is (%g, %g), want (50, 75)", middle.X(), middle.Y())
	}
}

// TestBitReader checks the reader the mesh types decode through, including the
// bit offset the vertex padding depends on.
func TestBitReader(t *testing.T) {
	// 0xA5 is 1010 0101, 0x3C is 0011 1100.
	r := newBitReader(bytes.NewReader([]byte{0xA5, 0x3C}))
	for _, want := range []struct {
		bits  int
		value int64
	}{
		{4, 0xA},  // 1010
		{2, 0x1},  // 01
		{2, 0x1},  // 01
		{8, 0x3C}, // the whole second byte
	} {
		got, err := r.readBits(want.bits)
		if err != nil {
			t.Fatal(err)
		}
		if got != want.value {
			t.Errorf("readBits(%d) = %#x, want %#x", want.bits, got, want.value)
		}
	}
	if _, err := r.readBits(1); !errors.Is(err, io.EOF) {
		t.Errorf("readBits past the end = %v, want io.EOF", err)
	}

	// The offset is zero on a byte boundary and the count into the byte
	// otherwise, which is what ImageInputStream.getBitOffset answers.
	r = newBitReader(bytes.NewReader([]byte{0xFF, 0xFF}))
	if got := r.getBitOffset(); got != 0 {
		t.Errorf("getBitOffset() before any read = %d, want 0", got)
	}
	if _, err := r.readBits(3); err != nil {
		t.Fatal(err)
	}
	if got := r.getBitOffset(); got != 3 {
		t.Errorf("getBitOffset() after three bits = %d, want 3", got)
	}
	if _, err := r.readBits(5); err != nil {
		t.Fatal(err)
	}
	if got := r.getBitOffset(); got != 0 {
		t.Errorf("getBitOffset() back on a boundary = %d, want 0", got)
	}
}

// meshShading builds a mesh shading of the given type over the given stream
// data, and returns it.
func meshShading(t *testing.T, shadingType int, data []byte) Shading {
	t.Helper()
	_, stream := meshDictionary(t, shadingType, data)
	created, err := NewPDShading(stream)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

// meshDictionary builds the shading stream every mesh test above uses: eight
// bit fields, a decode range that maps coordinates one to one and colours onto
// zero to one, and DeviceRGB.
func meshDictionary(t *testing.T, shadingType int, data []byte) (*cos.Dictionary, *cos.Stream) {
	t.Helper()
	stream := cos.NewStream(filter.Provider{})
	w, err := stream.CreateWriter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	dict := &stream.Dictionary
	dict.SetInt(cos.ShadingType, shadingType)
	dict.SetItem(cos.ColorSpace, cos.DeviceRGB)
	dict.SetInt(cos.BitsPerCoordinate, 8)
	dict.SetInt(cos.BitsPerComponent, 8)
	dict.SetInt(cos.BitsPerFlag, 8)
	dict.SetItem(cos.Decode, arrayOfFloats(0, 255, 0, 255, 0, 1, 0, 1, 0, 1))
	return dict, stream
}

// arrayOfFloats is a COSArray of the given numbers.
func arrayOfFloats(values ...float32) *cos.Array {
	array := cos.NewArray()
	for _, v := range values {
		array.Add(cos.NewFloat(v))
	}
	return array
}

// assertTriangle checks a triangle's corners and colours.
func assertTriangle(t *testing.T, triangle *shadedTriangle,
	corners [3][2]float64, colors [3][]float32) {
	t.Helper()
	for i, want := range corners {
		got := triangle.corner[i]
		if math.Abs(got.X()-want[0]) > 1e-6 || math.Abs(got.Y()-want[1]) > 1e-6 {
			t.Errorf("corner %d = (%g, %g), want (%g, %g)",
				i, got.X(), got.Y(), want[0], want[1])
		}
	}
	for i, want := range colors {
		got := triangle.color[i]
		if len(got) != len(want) {
			t.Errorf("colour %d = %v, want %v", i, got, want)
			continue
		}
		for j := range want {
			if math.Abs(float64(got[j]-want[j])) > 1e-6 {
				t.Errorf("colour %d = %v, want %v", i, got, want)
				break
			}
		}
	}
}

// assertRectangle checks a bounding box.
func assertRectangle(t *testing.T, what string, got *geom.Rectangle2D, x, y, w, h float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want (%g,%g %gx%g)", what, x, y, w, h)
	}
	const tolerance = 1e-6
	if math.Abs(got.X-x) > tolerance || math.Abs(got.Y-y) > tolerance ||
		math.Abs(got.Width-w) > tolerance || math.Abs(got.Height-h) > tolerance {
		t.Errorf("%s = (%g,%g %gx%g), want (%g,%g %gx%g)",
			what, got.X, got.Y, got.Width, got.Height, x, y, w, h)
	}
}
