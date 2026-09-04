package image

import "testing"

// TestSetPixelRefusesAShortSlice pins what the slice 6 feedback asked for.
//
// Java's WritableRaster.setPixel(int, int, int[]) reads numBands values out of
// the array and throws ArrayIndexOutOfBoundsException where there are fewer.
// The port used to stop at the shorter of the two, which left the remaining
// bands holding whatever the pixel had before -- a caller's mistake turned into
// a half written pixel rather than a failure.
func TestSetPixelRefusesAShortSlice(t *testing.T) {
	raster := NewInterleavedRaster(TypeByte, 2, 2, 3)

	defer func() {
		if recover() == nil {
			t.Error("SetPixel with fewer values than bands should panic")
		}
	}()
	raster.SetPixel(0, 0, []int{1, 2})
}

// TestSetDataElementsRefusesAShortSlice is the same for the byte form.
func TestSetDataElementsRefusesAShortSlice(t *testing.T) {
	raster := NewInterleavedRaster(TypeByte, 2, 2, 3)

	defer func() {
		if recover() == nil {
			t.Error("SetDataElements with fewer values than bands should panic")
		}
	}()
	raster.SetDataElements(0, 0, []byte{1, 2})
}

// TestSetPixelAcceptsALongerSlice checks the other side: Java reads only
// numBands values and ignores the rest, which is what lets the CIE colour
// spaces hand a three element array to a one band raster.
func TestSetPixelAcceptsALongerSlice(t *testing.T) {
	raster := NewInterleavedRaster(TypeByte, 2, 1, 1)
	raster.SetPixel(0, 0, []int{7, 99, 99})
	pixel := raster.GetPixel(0, 0, make([]int, 1))
	if pixel[0] != 7 {
		t.Errorf("the pixel is %v, want the first value only", pixel)
	}
}

// TestRasterRoundTrip covers the accessors PDFBox uses, which nothing else
// tests directly.
func TestRasterRoundTrip(t *testing.T) {
	raster := NewInterleavedRaster(TypeByte, 3, 2, 2)
	if raster.Width() != 3 || raster.Height() != 2 || raster.NumBands() != 2 {
		t.Fatalf("the raster is %dx%d of %d bands",
			raster.Width(), raster.Height(), raster.NumBands())
	}
	if raster.MinX() != 0 || raster.MinY() != 0 {
		t.Error("PDFBox builds every raster at the origin")
	}
	if raster.DataType() != TypeByte || raster.TransferType() != TypeByte {
		t.Error("the data type should be TypeByte")
	}
	if raster.NumDataElements() != 2 {
		t.Error("NumDataElements should be the band count")
	}

	raster.SetPixels(0, 0, 3, 2, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	got := raster.GetPixels(0, 0, 3, 2, make([]int, 12))
	for i, want := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12} {
		if got[i] != want {
			t.Fatalf("GetPixels = %v", got)
		}
	}

	raster.SetSamples(0, 0, 3, 2, 1, []int{20, 21, 22, 23, 24, 25})
	got = raster.GetPixels(0, 0, 3, 2, make([]int, 12))
	if got[1] != 20 || got[3] != 21 || got[11] != 25 {
		t.Errorf("SetSamples wrote %v", got)
	}

	raster.SetDataElements(1, 1, []byte{200, 201})
	elements := raster.GetDataElements(1, 1, make([]byte, 2))
	if elements[0] != 200 || elements[1] != 201 {
		t.Errorf("the data elements are %v", elements)
	}

	empty := raster.CreateCompatibleWritableRaster()
	if empty.Width() != 3 || empty.Height() != 2 || empty.NumBands() != 2 {
		t.Error("the compatible raster has the wrong shape")
	}
	if empty.GetPixel(1, 1, make([]int, 2))[0] != 0 {
		t.Error("the compatible raster should be empty")
	}
}

// TestBandedRasterRefusesMoreThanOneBand pins the port's own limit: PDFBox only
// ever asks for one band, where the banded and interleaved layouts are the
// same, so a caller asking for more would get a layout Java would not give it.
func TestBandedRasterRefusesMoreThanOneBand(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a banded raster of more than one band should be refused")
		}
	}()
	NewBandedRaster(TypeByte, 2, 2, 3)
}
