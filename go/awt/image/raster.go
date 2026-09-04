// Package image holds the raster a PDF image decodes into, and the sample
// types it can hold.
//
// Stands for the part of java.awt.image PDFBox uses: WritableRaster,
// DataBuffer and the two Raster factory methods. Go has no equivalent -- the
// standard library's image package models pictures with a fixed colour model,
// where a PDF image is samples in a colour space the document names -- so the
// port supplies the raster and turns it into an image.Image only at the end,
// where the colour space converts it to RGB.
package image

// The sample types a raster can hold.
//
// Port of the DataBuffer.TYPE_ constants PDFBox asks for; it uses only these
// two, and asks for TYPE_USHORT where an image carries 16 bits per component.
const (
	TypeByte = iota
	TypeUShort
)

// Raster holds the samples of an image: width by height pixels of numBands
// components each.
//
// Java has an interleaved raster and a banded one, and PDFBox builds both; the
// banded ones it builds all have a single band, where the two layouts are the
// same, so the port stores interleaved throughout and says so rather than
// carrying a sample model.
//
// Samples are held as uint16 whatever the data type, because a TYPE_BYTE
// raster holds 0 to 255 and a TYPE_USHORT one 0 to 65535, and Java's
// getPixel/setPixel pass both as int.
type Raster struct {
	width    int
	height   int
	numBands int
	dataType int
	samples  []uint16
}

// NewInterleavedRaster returns a raster with the samples of each pixel
// adjacent.
//
// Port of Raster.createInterleavedRaster(int, int, int, int, Point).
func NewInterleavedRaster(dataType, width, height, numBands int) *Raster {
	return &Raster{
		width:    width,
		height:   height,
		numBands: numBands,
		dataType: dataType,
		samples:  make([]uint16, width*height*numBands),
	}
}

// NewBandedRaster returns a raster with the samples of each band together.
//
// Port of Raster.createBandedRaster(int, int, int, int, Point). PDFBox only
// ever asks for one band, so this is the interleaved layout; a caller asking
// for more would get a layout Java would not give it, and the port refuses
// rather than answering wrongly.
func NewBandedRaster(dataType, width, height, numBands int) *Raster {
	if numBands != 1 {
		panic("image: a banded raster with more than one band is not ported")
	}
	return NewInterleavedRaster(dataType, width, height, numBands)
}

// Width returns the width in pixels.
func (r *Raster) Width() int { return r.width }

// Height returns the height in pixels.
func (r *Raster) Height() int { return r.height }

// NumBands returns how many samples one pixel has.
func (r *Raster) NumBands() int { return r.numBands }

// NumDataElements returns how many values one pixel takes in the transfer
// type, which for these rasters is the band count.
func (r *Raster) NumDataElements() int { return r.numBands }

// TransferType returns the data type of the samples.
func (r *Raster) TransferType() int { return r.dataType }

// DataType returns the data type of the samples, which is Java's
// getDataBuffer().getDataType().
func (r *Raster) DataType() int { return r.dataType }

// MinX returns the x coordinate of the first pixel, which is always 0 here:
// PDFBox builds every raster at the origin.
func (r *Raster) MinX() int { return 0 }

// MinY returns the y coordinate of the first pixel.
func (r *Raster) MinY() int { return 0 }

// Samples returns the sample array itself, which is Java's
// getDataBuffer().getData().
func (r *Raster) Samples() []uint16 { return r.samples }

func (r *Raster) offset(x, y int) int { return (y*r.width + x) * r.numBands }

// GetPixel writes the samples of one pixel into out and returns it.
//
// Java's getPixel(int, int, int[]) allocates when handed null; the port takes
// the destination and leaves the allocating to the caller, because every
// caller in PDFBox reuses one array across a loop.
func (r *Raster) GetPixel(x, y int, out []int) []int {
	base := r.offset(x, y)
	for b := 0; b < r.numBands; b++ {
		out[b] = int(r.samples[base+b])
	}
	return out
}

// SetPixel sets the samples of one pixel.
//
// Java reads numBands values out of the array and throws
// ArrayIndexOutOfBoundsException where there are fewer, so a caller that hands
// over too few fails rather than half writing the pixel; the port panics for
// the same. A longer array is fine in both -- the CIE colour spaces pass a
// three element one to a single band raster.
func (r *Raster) SetPixel(x, y int, values []int) {
	if len(values) < r.numBands {
		panic("image: SetPixel was given fewer values than the raster has bands")
	}
	base := r.offset(x, y)
	for b := 0; b < r.numBands; b++ {
		r.samples[base+b] = uint16(values[b])
	}
}

// SetDataElements sets the samples of one pixel from a byte per band, which is
// what PDFBox passes for a TYPE_BYTE raster.
//
// Port of setDataElements(int, int, Object) for a byte[] argument.
func (r *Raster) SetDataElements(x, y int, values []byte) {
	if len(values) < r.numBands {
		panic("image: SetDataElements was given fewer values than the raster has bands")
	}
	base := r.offset(x, y)
	for b := 0; b < r.numBands; b++ {
		r.samples[base+b] = uint16(values[b])
	}
}

// GetDataElements reads the samples of one pixel as a byte per band.
func (r *Raster) GetDataElements(x, y int, out []byte) []byte {
	base := r.offset(x, y)
	for b := 0; b < r.numBands; b++ {
		out[b] = byte(r.samples[base+b])
	}
	return out
}

// GetPixels writes the samples of a rectangle of pixels into out, pixel by
// pixel and band by band within each.
//
// Port of getPixels(int, int, int, int, int[]).
func (r *Raster) GetPixels(x, y, w, h int, out []int) []int {
	i := 0
	for row := y; row < y+h; row++ {
		base := r.offset(x, row)
		for j := 0; j < w*r.numBands; j++ {
			out[i] = int(r.samples[base+j])
			i++
		}
	}
	return out
}

// SetPixels sets the samples of a rectangle of pixels.
func (r *Raster) SetPixels(x, y, w, h int, values []int) {
	i := 0
	for row := y; row < y+h; row++ {
		base := r.offset(x, row)
		for j := 0; j < w*r.numBands; j++ {
			r.samples[base+j] = uint16(values[i])
			i++
		}
	}
}

// SetSamples sets one band of a rectangle of pixels.
//
// Port of setSamples(int, int, int, int, int, int[]).
func (r *Raster) SetSamples(x, y, w, h, band int, values []int) {
	i := 0
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			r.samples[r.offset(col, row)+band] = uint16(values[i])
			i++
		}
	}
}

// CreateCompatibleWritableRaster returns an empty raster of the same shape.
func (r *Raster) CreateCompatibleWritableRaster() *Raster {
	return NewInterleavedRaster(r.dataType, r.width, r.height, r.numBands)
}
