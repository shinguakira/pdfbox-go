package raster

// The blend arithmetic, against PDFBox's own compositor.
//
// composite.go's blendInto is a port of
// BlendComposite.BlendCompositeContext.compose, and unlike the shapes this
// package draws, that class is real PDFBox and it is in the tree. So it is run
// rather than reasoned about: `testdata/BlendDrv.java` drives it the way
// PageDrawer does -- setComposite, setColor, fill -- over every blend mode and
// a set of colour pairs, and `testdata/blend.txt` is what came back.
//
// Regenerate it with the PDFBox classes and log4j-api on the class path:
//
//	javac -cp <classes> -d . BlendDrv.java
//	java -cp .;<classes>;<log4j-api.jar> BlendDrv > blend.txt
//
// Every expected value below is a line of that file.

import (
	goimagecolor "image/color"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// blendModesByName is the driver's names against the port's modes, in the
// order BlendDrv lists them.
var blendModesByName = map[string]*blend.BlendMode{
	"Normal":      blend.Normal,
	"NormalBlend": blend.Normal,
	"Multiply":    blend.Multiply,
	"Screen":      blend.Screen,
	"Overlay":     blend.Overlay,
	"Darken":      blend.Darken,
	"Lighten":     blend.Lighten,
	"ColorDodge":  blend.ColorDodge,
	"ColorBurn":   blend.ColorBurn,
	"HardLight":   blend.HardLight,
	"SoftLight":   blend.SoftLight,
	"Difference":  blend.Difference,
	"Exclusion":   blend.Exclusion,
	"Hue":         blend.Hue,
	"Saturation":  blend.Saturation,
	"Color":       blend.Color,
	"Luminosity":  blend.Luminosity,
}

// blendRow is one line of blend.txt.
type blendRow struct {
	line     int
	mode     string
	src      uint32
	dst      uint32
	constant float64
	want     uint32
}

// readBlendReference parses what BlendDrv printed.
func readBlendReference(t *testing.T) []blendRow {
	t.Helper()
	contents, err := os.ReadFile("testdata/blend.txt")
	if err != nil {
		t.Fatalf("reading the BlendComposite reference: %v", err)
	}
	var rows []blendRow
	for n, line := range strings.Split(strings.ReplaceAll(string(contents), "\r\n", "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 6 || fields[4] != "->" {
			t.Fatalf("blend.txt:%d: cannot read %q", n+1, line)
		}
		row := blendRow{line: n + 1, mode: fields[0]}
		for _, f := range []struct {
			text string
			into *uint32
		}{{fields[1], &row.src}, {fields[2], &row.dst}, {fields[5], &row.want}} {
			value, err := strconv.ParseUint(f.text, 16, 32)
			if err != nil {
				t.Fatalf("blend.txt:%d: %v", n+1, err)
			}
			*f.into = uint32(value)
		}
		if row.constant, err = strconv.ParseFloat(fields[3], 64); err != nil {
			t.Fatalf("blend.txt:%d: %v", n+1, err)
		}
		rows = append(rows, row)
	}
	return rows
}

// argb splits a packed colour the way the driver printed it: alpha first, and
// not premultiplied, which is what TYPE_INT_ARGB holds and what this backend's
// pixels hold too.
func argb(v uint32) (a, r, g, b uint8) {
	return uint8(v >> 24), uint8(v >> 16), uint8(v >> 8), uint8(v)
}

// composeRow puts one source over one destination the way the driver did.
func composeRow(t *testing.T, row blendRow, mode *blend.BlendMode) goimagecolor.RGBA {
	t.Helper()
	srcAlpha, srcRed, srcGreen, srcBlue := argb(row.src)
	dstAlpha, dstRed, dstGreen, dstBlue := argb(row.dst)

	i := NewImage(1, 1, rendering.ARGB)
	i.SetAntiAliasing(false)
	// The destination as the driver's BufferedImage held it.
	i.dst.SetRGBA(0, 0, goimagecolor.RGBA{R: dstRed, G: dstGreen, B: dstBlue, A: dstAlpha})
	i.SetComposite(mode, row.constant)
	i.SetPaint(rendering.ColorPaint{
		Red:   float32(srcRed) / 255,
		Green: float32(srcGreen) / 255,
		Blue:  float32(srcBlue) / 255,
		Alpha: float32(srcAlpha) / 255,
	})
	if err := i.Fill(rectangle(0, 0, 1, 1)); err != nil {
		t.Fatalf("blend.txt:%d: Fill: %v", row.line, err)
	}
	return i.dst.RGBAAt(0, 0)
}

// TestBlendsAsPDFBoxDoes fills one pixel over one pixel, once per line of the
// reference, and asks for the same answer.
//
// Every mode is held to it exactly, Normal included: the "NormalBlend" rows
// are BlendComposite over NORMAL, reached by reflection because getInstance
// will not hand one out, and they are what this backend's one code path has to
// produce. The "Normal" rows are the other thing, and they are separate --
// see TestAlphaCompositeRoundsSourceOverDifferently.
func TestBlendsAsPDFBoxDoes(t *testing.T) {
	for _, row := range readBlendReference(t) {
		if row.mode == "Normal" {
			continue
		}
		mode, known := blendModesByName[row.mode]
		if !known {
			t.Fatalf("blend.txt:%d: no mode named %q", row.line, row.mode)
		}
		got := composeRow(t, row, mode)
		wantAlpha, wantRed, wantGreen, wantBlue := argb(row.want)
		want := goimagecolor.RGBA{R: wantRed, G: wantGreen, B: wantBlue, A: wantAlpha}
		if got != want {
			t.Errorf("blend.txt:%d: %s %08x over %08x at %.2f is %02x%02x%02x%02x, "+
				"and PDFBox makes it %08x",
				row.line, row.mode, row.src, row.dst, row.constant,
				got.A, got.R, got.G, got.B, row.want)
		}
	}
}

// TestAlphaCompositeRoundsSourceOverDifferently is where the port and Java
// part, and it is a place Java parts from itself.
//
// BlendComposite.getInstance answers an AlphaComposite for NORMAL and
// COMPATIBLE rather than a BlendComposite, so `/BM /Normal` -- which is most
// of every document -- never reaches compose in Java at all. AlphaComposite is
// the JDK's own source-over, and it works on premultiplied bytes; compose
// works on components normalized to floats. On the same pixels the two do not
// always agree, and neither does this port, which has compose's arithmetic
// and only that.
//
// The disagreement is one unit of one channel, and only where the source and
// the destination are both partly transparent -- five of the twenty rows.
// Where either is opaque, which is the overwhelming majority of what a page
// composites, they agree exactly.
func TestAlphaCompositeRoundsSourceOverDifferently(t *testing.T) {
	const differingRows = 5
	differing := 0
	for _, row := range readBlendReference(t) {
		if row.mode != "Normal" {
			continue
		}
		got := composeRow(t, row, blend.Normal)
		wantAlpha, wantRed, wantGreen, wantBlue := argb(row.want)
		want := goimagecolor.RGBA{R: wantRed, G: wantGreen, B: wantBlue, A: wantAlpha}
		if got == want {
			continue
		}
		differing++
		for _, channel := range [][2]uint8{{got.A, want.A}, {got.R, want.R},
			{got.G, want.G}, {got.B, want.B}} {
			if delta := int(channel[0]) - int(channel[1]); delta < -1 || delta > 1 {
				t.Errorf("blend.txt:%d: %08x over %08x at %.2f is %02x%02x%02x%02x, "+
					"and AlphaComposite makes it %08x -- more than the one unit "+
					"the two roundings can differ by",
					row.line, row.src, row.dst, row.constant,
					got.A, got.R, got.G, got.B, row.want)
				break
			}
		}
	}
	if differing != differingRows {
		t.Errorf("%d of the source-over rows differ from AlphaComposite, and it was %d",
			differing, differingRows)
	}
}
