package raster

// A0's evidence. `track/raster` builds its backend on
// github.com/srwiley/rasterx, and this is the measurement the choice was made
// on: rasterx answers the whole of the stroke model a PDF graphics state can
// ask for -- the three joins of `j`, the three caps of `J`, the miter limit of
// `M` and the dash pattern of `d`.
//
// It is kept because it also pins the one trap in the API, which cost an hour
// to find: SetStroke takes a gap function, and passing one explicitly
// overrides the join mode's own. `Round` renders as `Bevel` unless the gap is
// left nil. See the comment on strokeGapFunc.

import (
	"image"
	"testing"

	"github.com/srwiley/rasterx"
	"golang.org/x/image/math/fixed"
)

// cornerStroke strokes a right angle -- east from (0,20) to (20,20), then
// north to (20,0) -- eight units wide, and answers the image.
//
// The centreline turns at (20,20) and the half width is 4, so the two segments
// cover everything but the square from (20,20) to (23,23). That square is the
// outside of the turn, and what fills it is the join and nothing else, which
// is what makes it the place to look.
func cornerStroke(t *testing.T, join rasterx.JoinMode, cap rasterx.CapFunc,
	dashes []float64) *image.RGBA {
	t.Helper()
	const size = 32
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	dasher := rasterx.NewDasher(size, size, rasterx.NewScannerGV(size, size, dst, dst.Bounds()))
	// The gap function is nil on purpose: rasterx picks RoundGap for a Round
	// join and FlatGap otherwise, and passing one here would defeat that.
	dasher.SetStroke(fixed.Int26_6(8*64), fixed.Int26_6(10*64),
		cap, nil, nil, join, dashes, 0)
	dasher.Start(rasterx.ToFixedP(0, 20))
	dasher.Line(rasterx.ToFixedP(20, 20))
	dasher.Line(rasterx.ToFixedP(20, 0))
	dasher.Stop(false)
	dasher.Draw()
	return dst
}

// alphaAt is the coverage rasterx laid down at a point, 0 for none.
func alphaAt(img *image.RGBA, x, y int) uint32 {
	_, _, _, a := img.At(x, y).RGBA()
	return a
}

// TestRasterxDrawsTheThreeJoins is the discriminator. At (22,22) the miter is
// solid, the round join is partly covered by its arc, and the bevel's chord
// has already cut the corner away; at (23,23), the far tip, only the miter
// reaches.
func TestRasterxDrawsTheThreeJoins(t *testing.T) {
	miter := cornerStroke(t, rasterx.Miter, rasterx.ButtCap, nil)
	round := cornerStroke(t, rasterx.Round, rasterx.ButtCap, nil)
	bevel := cornerStroke(t, rasterx.Bevel, rasterx.ButtCap, nil)

	if got := alphaAt(miter, 22, 22); got != 0xFFFF {
		t.Errorf("miter at (22,22) = %#04x, want full coverage", got)
	}
	if got := alphaAt(miter, 23, 23); got == 0 {
		t.Error("miter does not reach (23,23), the tip of the corner")
	}

	if got := alphaAt(round, 22, 22); got == 0 {
		t.Error("the round join leaves (22,22) blank, which its arc covers")
	}
	if got := alphaAt(round, 23, 23); got != 0 {
		t.Errorf("the round join reaches (23,23) = %#04x, which is outside its arc", got)
	}

	if got := alphaAt(bevel, 22, 22); got != 0 {
		t.Errorf("the bevel reaches (22,22) = %#04x, which its chord cuts off", got)
	}
}

// TestRasterxDashes is the `d` operator: four on, four off along the first
// leg, so the run from x=4 to x=8 is blank.
func TestRasterxDashes(t *testing.T) {
	dashed := cornerStroke(t, rasterx.Miter, rasterx.ButtCap, []float64{4, 4})

	if got := alphaAt(dashed, 2, 20); got == 0 {
		t.Error("the first dash is missing")
	}
	if got := alphaAt(dashed, 6, 20); got != 0 {
		t.Errorf("the first gap has ink at (6,20) = %#04x", got)
	}
	if got := alphaAt(dashed, 10, 20); got == 0 {
		t.Error("the second dash is missing")
	}
}

// TestRasterxDrawsTheThreeCaps is `J`. The line runs east from (8,20) to
// (24,20), so the cap is what covers anything left of x=8, and the half width
// is 4.
//
// Measured, not assumed: a butt cap stops dead at x=8; a square cap fills the
// whole rectangle back to x=4, corners included; a round cap fills its
// semicircle, which reaches x=4 on the centreline and leaves the corner at
// (4,16) outside.
func TestRasterxDrawsTheThreeCaps(t *testing.T) {
	const size = 32
	stroke := func(cap rasterx.CapFunc) *image.RGBA {
		dst := image.NewRGBA(image.Rect(0, 0, size, size))
		d := rasterx.NewDasher(size, size, rasterx.NewScannerGV(size, size, dst, dst.Bounds()))
		d.SetStroke(fixed.Int26_6(8*64), fixed.Int26_6(10*64), cap, nil, nil, rasterx.Miter, nil, 0)
		d.Start(rasterx.ToFixedP(8, 20))
		d.Line(rasterx.ToFixedP(24, 20))
		d.Stop(false)
		d.Draw()
		return dst
	}

	butt, square, round := stroke(rasterx.ButtCap), stroke(rasterx.SquareCap), stroke(rasterx.RoundCap)

	if got := alphaAt(butt, 6, 20); got != 0 {
		t.Errorf("the butt cap runs past the end of the line: (6,20) = %#04x", got)
	}
	if got := alphaAt(square, 4, 16); got == 0 {
		t.Error("the square cap does not fill its corner at (4,16)")
	}
	if got := alphaAt(round, 4, 16); got != 0 {
		t.Errorf("the round cap reaches its corner (4,16) = %#04x, outside its "+
			"semicircle", got)
	}
	if got := alphaAt(round, 4, 20); got == 0 {
		t.Error("the round cap does not reach (4,20), the far point of its arc")
	}
}
