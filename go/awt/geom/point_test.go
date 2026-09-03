package geom

import "testing"

// Written from the java.awt.geom.Point2D specification. There is no PDFBox test
// for this class — it belongs to the JDK — so per migration/conventions/tdd.md
// the expected values come from the documented behaviour, not from this port.

func TestPointFloatStoresSinglePrecision(t *testing.T) {
	// 0.1 is not representable in binary, and a float32 rounds it further than
	// a float64 does. Point2D.Float stores floats, so reading the coordinate
	// back gives the float32 value widened, not the double that was set.
	p := NewPointFloat(0, 0)
	p.SetLocation(0.1, 0.1)

	if got, want := p.X(), float64(float32(0.1)); got != want {
		t.Errorf("X() = %v, want %v", got, want)
	}
	if got, want := p.XFloat(), float32(0.1); got != want {
		t.Errorf("XFloat() = %v, want %v", got, want)
	}
}

func TestPointDoubleStoresDoublePrecision(t *testing.T) {
	p := NewPointDouble(0, 0)
	p.SetLocation(0.1, 0.2)

	if got := p.X(); got != 0.1 {
		t.Errorf("X() = %v, want 0.1", got)
	}
	if got := p.Y(); got != 0.2 {
		t.Errorf("Y() = %v, want 0.2", got)
	}
}

func TestPointSatisfiesPoint2D(t *testing.T) {
	var points = []Point2D{NewPointFloat(3, 4), NewPointDouble(3, 4)}
	for _, p := range points {
		if p.X() != 3 || p.Y() != 4 {
			t.Errorf("%v = (%v, %v), want (3, 4)", p, p.X(), p.Y())
		}
	}
}

func TestDistance(t *testing.T) {
	if got := DistanceSq(1, 1, 4, 5); got != 25 {
		t.Errorf("DistanceSq = %v, want 25", got)
	}
	if got := Distance(1, 1, 4, 5); got != 5 {
		t.Errorf("Distance = %v, want 5", got)
	}
	if got := NewPointDouble(1, 1).Distance(NewPointFloat(4, 5)); got != 5 {
		t.Errorf("Distance = %v, want 5", got)
	}
}

func TestPointString(t *testing.T) {
	if got, want := NewPointFloat(1, 2).String(), "Point2D.Float[1.0, 2.0]"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got, want := NewPointDouble(1, 2).String(), "Point2D.Double[1.0, 2.0]"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
