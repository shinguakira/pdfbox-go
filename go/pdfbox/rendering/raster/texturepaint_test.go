package raster

// java.awt.TexturePaintContext, against the JDK's own.
//
// testdata/texturepaint.txt is what testdata/TexturePaintDrv.java prints: the
// pixels the JDK's TexturePaintContext answers for small textures, transforms
// and rectangles, and the rectangles Java2D asks a paint's context for while it
// fills or strokes a shape. Both matter, because the context walks the texture
// from the corner of each rectangle and not from each pixel. See the driver for
// how to regenerate it.

import (
	"fmt"
	goimage "image"
	goimagecolor "image/color"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// textureKind is the BufferedImage type a texture case is made as.
type textureKind int

const (
	rgbTexture  textureKind = iota // TYPE_INT_RGB
	argbTexture                    // TYPE_INT_ARGB
	greyTexture                    // TYPE_BYTE_GRAY
)

// textureRGB, textureAlpha and textureGrey are the driver's rgbAt, alphaAt and
// greyAt.
func textureRGB(x, y int) (uint8, uint8, uint8) {
	return uint8((x*37 + y*11) & 0xff), uint8((x*x + y*3) & 0xff), uint8(((x ^ y) * 9) & 0xff)
}

func textureAlpha(x, y int) uint8 {
	if (x+2*y)%7 == 0 {
		return 0
	}
	return uint8((x*29 + y*53) & 0xff)
}

func textureGrey(x, y int) uint8 {
	if (x/3+y/2)%2 == 0 {
		return 0
	}
	return 255
}

// textureOf is the driver's texture as the straight colour a tile holds.
func textureOf(kind textureKind, width, height int) *goimage.NRGBA {
	tile := goimage.NewNRGBA(goimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b := textureRGB(x, y)
			switch kind {
			case rgbTexture:
				tile.SetNRGBA(x, y, goimagecolor.NRGBA{R: r, G: g, B: b, A: 0xFF})
			case argbTexture:
				tile.SetNRGBA(x, y, goimagecolor.NRGBA{R: r, G: g, B: b, A: textureAlpha(x, y)})
			case greyTexture:
				v := textureGrey(x, y)
				tile.SetNRGBA(x, y, goimagecolor.NRGBA{R: v, G: v, B: v, A: 0xFF})
			}
		}
	}
	return tile
}

// floatRect is a Rectangle2D.Float: its values are floats, read as doubles.
func floatRect(x, y, w, h float32) *geom.Rectangle2D {
	return geom.NewRectangle2D(float64(x), float64(y), float64(w), float64(h))
}

// textureCase is one "texture" case of the driver.
type textureCase struct {
	kind          textureKind
	width, height int
	anchor        *geom.Rectangle2D
	xform         *geom.AffineTransform
	filter        bool
}

var (
	imageXform = geom.NewAffineTransform(1.5, 0, 0, 1.5, 60, 60)
	flipXform  = geom.NewAffineTransform(1, 0, 0, -1, 0, 60)
	skewXform  = geom.NewAffineTransform(1.2, 0.5, -0.4, 1.3, 20.25, 7.5)
	skewAnchor = floatRect(2.5, -3.25, 11.2, 15.6)
	tileAnchor = floatRect(0, 0, 13.7, 13.7)
)

var textureCases = map[string]textureCase{
	"imageNearest":  {rgbTexture, 40, 30, floatRect(0, 0, 40, 30), imageXform, false},
	"imageFiltered": {rgbTexture, 40, 30, floatRect(0, 0, 40, 30), imageXform, true},
	"patternRows":   {argbTexture, 13, 13, tileAnchor, flipXform, true},
	"patternTiles":  {argbTexture, 13, 13, tileAnchor, flipXform, true},
	"skewFiltered":  {argbTexture, 16, 12, skewAnchor, skewXform, true},
	"skewNearest":   {argbTexture, 16, 12, skewAnchor, skewXform, false},
	"shrinkFiltered": {rgbTexture, 40, 30, geom.NewRectangle2D(0, 0, 26.5, 19.25),
		geom.NewAffineTransform(1, 0, 0, 1, 3, 4), true},
	"greyNearest":  {greyTexture, 20, 10, floatRect(0, 0, 30, 15), geom.NewIdentityTransform(), false},
	"greyFiltered": {greyTexture, 20, 10, floatRect(0, 0, 30, 15), geom.NewIdentityTransform(), true},
}

// textureRequest is one rectangle a texture case asked for, and what the JDK
// answered, ARGB.
type textureRequest struct {
	x, y, w, h int
	pixels     [][]uint32
}

// textureReference reads the driver's output.
func textureReference(t *testing.T) (map[string][]textureRequest, map[string][][4]int,
	map[string][]uint8) {
	t.Helper()
	contents, err := os.ReadFile("testdata/texturepaint.txt")
	if err != nil {
		t.Fatalf("reading the TexturePaint reference: %v", err)
	}
	textures := map[string][]textureRequest{}
	requests := map[string][][4]int{}
	tables := map[string][]uint8{}
	var kind, name string
	ints := func(fields []string) []int {
		values := make([]int, len(fields))
		for k, field := range fields {
			value, err := strconv.Atoi(field)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			values[k] = value
		}
		return values
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(contents), "\r\n", "\n"), "\n") {
		fields := strings.Fields(line)
		switch {
		case len(fields) == 0:
		case fields[0] == "###":
			kind, name = fields[1], fields[2]
			if kind == "requests" {
				requests[name] = [][4]int{}
			}
		case kind == "texture" && fields[0] == "request":
			v := ints(fields[1:])
			textures[name] = append(textures[name], textureRequest{x: v[0], y: v[1], w: v[2], h: v[3]})
		case kind == "texture":
			request := &textures[name][len(textures[name])-1]
			row := make([]uint32, len(line)/8)
			for k := range row {
				value, err := strconv.ParseUint(line[k*8:k*8+8], 16, 32)
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				row[k] = uint32(value)
			}
			request.pixels = append(request.pixels, row)
		case kind == "requests":
			v := ints(fields)
			requests[name] = append(requests[name], [4]int{v[0], v[1], v[2], v[3]})
		case kind == "table":
			for k := 0; k+2 <= len(line); k += 2 {
				value, err := strconv.ParseUint(line[k:k+2], 16, 8)
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				tables[name] = append(tables[name], uint8(value))
			}
		}
	}
	return textures, requests, tables
}

// argbOf is what a context answered, as the JDK prints it. A pixel with no
// alpha paints nothing, whatever colour it carries, so it prints as zero here
// and is compared as zero.
func argbOf(c goimagecolor.NRGBA, painted bool) uint32 {
	if !painted || c.A == 0 {
		return 0
	}
	return uint32(c.A)<<24 | uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
}

// TestATexturePaintContextAnswersWhatTheJDKsDoes asks the port's context for
// the rectangles the driver asked the JDK's for, and compares every pixel.
func TestATexturePaintContextAnswersWhatTheJDKsDoes(t *testing.T) {
	textures, _, _ := textureReference(t)
	for name, c := range textureCases {
		t.Run(name, func(t *testing.T) {
			want, found := textures[name]
			if !found {
				t.Fatalf("the reference has no texture case named %q", name)
			}
			context := newTextureContext(textureOf(c.kind, c.width, c.height),
				c.anchor, c.xform, c.filter)
			context.grey = c.kind == greyTexture
			differing, reported := 0, 0
			for _, q := range want {
				context.request(q.x, q.y)
				for j := 0; j < q.h; j++ {
					for i := 0; i < q.w; i++ {
						wanted := q.pixels[j][i]
						if wanted>>24 == 0 {
							wanted = 0
						}
						got := argbOf(context.colorAt(q.x+i, q.y+j))
						if got == wanted {
							continue
						}
						differing++
						if reported < 5 {
							reported++
							t.Errorf("request (%d,%d): pixel (%d,%d) is %08x, want the JDK's %08x",
								q.x, q.y, q.x+i, q.y+j, got, wanted)
						}
					}
				}
			}
			if differing > 0 {
				t.Errorf("%d pixels are not the JDK's", differing)
			}
		})
	}
}

// requestCase is one "requests" case of the driver: a page, its hints, a
// transform, a clip set before the transform, and one shape filled or, with
// a stroke, stroked.
type requestCase struct {
	width, height int
	antiAliasing  bool
	transform     *geom.AffineTransform
	clip          geom.Shape
	shape         geom.Shape
	stroke        *rendering.Stroke
}

func requestCases() map[string]requestCase {
	quad := geom.NewPathDouble()
	quad.MoveTo(10.3, 5.2)
	quad.LineTo(70.6, 12.9)
	quad.LineTo(55.5, 50.4)
	quad.LineTo(4.8, 33.3)
	quad.ClosePath()
	line := geom.NewPathDouble()
	line.MoveTo(10.2, 20.7)
	line.LineTo(90.4, 55.1)
	// new BasicStroke(3.5f): a square cap and a miter join, limit 10.
	stroke := &rendering.Stroke{LineWidth: 3.5, LineCap: 2, LineJoin: 0, MiterLimit: 10}
	identity := geom.NewIdentityTransform()
	ellipse := geom.NewEllipse2D(10, 10, 60, 40)
	return map[string]requestCase{
		"imageTiles": {200, 170, true, imageXform, nil, floatRect(0, 0, 40, 30), nil},
		"imageFraction": {200, 170, true, geom.NewAffineTransform(1.3, 0, 0, 1.7, 10.4, 20.95),
			nil, floatRect(0, 0, 40, 30), nil},
		"imageNoAntiAlias": {200, 170, false, geom.NewAffineTransform(1.5, 0, 0, 1.5, 60.7, 60.2),
			nil, floatRect(0, 0, 40, 30), nil},
		"patternNoAntiAlias": {120, 60, false, flipXform, nil,
			geom.NewAreaOfShape(floatRect(10, 10, 100, 40)), nil},
		"patternAntiAlias": {120, 60, true, flipXform, nil, geom.NewAreaOfShape(quad), nil},
		"clipRectangle": {120, 90, true, identity, geom.NewRectangle2D(20, 15, 50, 40),
			floatRect(0, 0, 100, 80), nil},
		"clipShapeNoAntiAlias": {100, 60, false, identity, ellipse,
			geom.NewAreaOfShape(floatRect(0, 0, 100, 60)), nil},
		"clipShapeAntiAlias": {100, 60, true, identity, ellipse, floatRect(0, 0, 100, 60), nil},
		"strokeAntiAlias":    {100, 70, true, identity, nil, line, stroke},
		"strokeNoAntiAlias":  {100, 70, false, identity, nil, line, stroke},
		"bigTile": {120, 90, true, identity, nil,
			geom.NewRectangle2D(5.5, 3.25, 100, 70), nil},
	}
}

// requestRecorder is a paint that paints nothing and remembers, for every
// pixel it is asked for, the corner of the rectangle it was asked within.
type requestRecorder struct {
	current goimage.Point
	corners map[goimage.Point]goimage.Point
}

func (r *requestRecorder) request(x, y int) { r.current = goimage.Pt(x, y) }

func (r *requestRecorder) colorAt(x, y int) (goimagecolor.NRGBA, bool) {
	r.corners[goimage.Pt(x, y)] = r.current
	return goimagecolor.NRGBA{}, false
}

// TestTheRectanglesAreTheOnesJava2DAsksFor fills and strokes the driver's
// shapes and checks, pixel by pixel, that each is asked for within the
// rectangle Java2D asked for it in: the same corner, so the same walk.
//
// A pixel the port covers that no rectangle of Java2D's holds is a pixel
// Java2D does not paint, and is counted apart: that is a difference of
// coverage, not of where the walk starts.
func TestTheRectanglesAreTheOnesJava2DAsksFor(t *testing.T) {
	_, reference, _ := textureReference(t)
	for name, c := range requestCases() {
		t.Run(name, func(t *testing.T) {
			want, found := reference[name]
			if !found {
				t.Fatalf("the reference has no requests case named %q", name)
			}
			i := NewImage(c.width, c.height, rendering.RGB)
			i.SetAntiAliasing(c.antiAliasing)
			if c.clip != nil {
				i.SetClip(geom.NewAreaOfShape(c.clip))
			}
			i.SetTransform(c.transform)
			recorder := &requestRecorder{corners: map[goimage.Point]goimage.Point{}}
			if c.stroke != nil {
				i.SetStroke(c.stroke)
				mask, within, requests := i.strokeParts(c.shape)
				noError(t, i.composeSource(mask, within, requests, recorder, 1))
			} else {
				mask, within, requests := i.fillParts(c.shape)
				noError(t, i.composeSource(mask, within, requests, recorder, 1))
			}
			if len(recorder.corners) == 0 {
				t.Fatal("nothing was asked for")
			}
			outside, wrong := 0, 0
			var firstWrong []string
			for pixel, corner := range recorder.corners {
				var held *[4]int
				for k := range want {
					q := &want[k]
					if pixel.In(goimage.Rect(q[0], q[1], q[0]+q[2], q[1]+q[3])) {
						held = q
						break
					}
				}
				if held == nil {
					outside++
					continue
				}
				if corner != goimage.Pt(held[0], held[1]) {
					wrong++
					if wrong <= 5 {
						firstWrong = append(firstWrong, fmt.Sprintf(
							"pixel %v is asked for from %v, and Java2D asks for it in %v",
							pixel, corner, *held))
					}
				}
			}
			pinned := requestDifferences[name]
			if wrong != pinned.wrong {
				if pinned.wrong == 0 {
					for _, message := range firstWrong {
						t.Error(message)
					}
				}
				t.Errorf("%d of %d pixels are asked for from another corner; it was %d",
					wrong, len(recorder.corners), pinned.wrong)
			}
			if outside != pinned.outside {
				t.Errorf("%d pixels are covered where Java2D asks for nothing; it was %d",
					outside, pinned.outside)
			}
		})
	}
}

// requestDifferences is what each case is pinned at: pixels asked for from
// another corner than Java2D's, and pixels the port covers where Java2D asks
// for nothing. Seven of the eleven are Java2D's exactly. The other four:
//
//	imageFraction            0 and 53: row 20. The rectangle begins 0.95 of
//	                         the way down it, below the last of Marlin's eight
//	                         subpixel rows there, so Marlin covers none of it;
//	                         the port's coverage is the area, 5% of each pixel.
//	imageNoAntiAlias         0 and 0. It was 0 and 45, column 120: renderRect
//	                         truncates the rectangle's corners, 60.7 to 120.7,
//	                         to [60, 120), where the port covered the pixels
//	                         whose centres it held, [61, 121). The port fills
//	                         with antialiasing off the way Java2D does now (see
//	                         fillrule.go), and under this scale of 1.5 the
//	                         default pen is wide, so both sides round the
//	                         corners to the nearest quarter and land on the
//	                         same pixels as the truncated box.
//	clipShapeAntiAlias     760 and 10
//	clipShapeNoAntiAlias  1308 and 123: the clip is an ellipse, and the corners
//	                         follow the clip's pixels. Java2D's clip is a Region
//	                         ShapeSpanIterator makes, which flattens a curve to
//	                         within a pixel and holds a pixel whose centre is
//	                         inside; its rows of this ellipse are a pixel or so
//	                         narrower than the ellipse, and at the top and the
//	                         bottom much narrower. The port's clip is the
//	                         curve's own antialiased coverage, read here as
//	                         inside where it covers half a pixel or more. A
//	                         span that begins a pixel apart puts every pixel of
//	                         the row in another rectangle.
var requestDifferences = map[string]struct{ wrong, outside int }{
	"imageFraction":        {0, 53},
	"imageNoAntiAlias":     {0, 0},
	"clipShapeAntiAlias":   {760, 10},
	"clipShapeNoAntiAlias": {1308, 123},
}

func noError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// TestTheGreyTablesAreTheJDKs checks both of TYPE_BYTE_GRAY's tables, every
// entry, against what the driver printed of its colour model.
func TestTheGreyTablesAreTheJDKs(t *testing.T) {
	_, _, tables := textureReference(t)
	for name, table := range map[string]*[256]uint8{
		"greyToSRGB": &grayToSRGB,
		"sRGBToGrey": &sRGBToGrey,
	} {
		want := tables[name]
		if len(want) != len(table) {
			t.Fatalf("the reference's %s has %d entries", name, len(want))
		}
		for k := range want {
			if table[k] != want[k] {
				t.Errorf("%s[%d] is %d, and the JDK's is %d", name, k, table[k], want[k])
			}
		}
	}
}
