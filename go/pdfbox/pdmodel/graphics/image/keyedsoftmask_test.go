package image

import (
	"fmt"
	goimagecolor "image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// TestAKeyedPixelKeepsItsColourUnderASoftMask checks an image that has both a
// colour key /Mask and an /SMask.
//
// getImage keys the image first, into a TYPE_INT_ARGB image whose keyed-out
// pixel keeps its colour and has an alpha of 0. applyMask then replaces every
// alpha with the soft mask's, so that pixel comes back in its own colour. The
// Go version read the keyed image premultiplied on the way into the mask, and
// a pixel with an alpha of 0 premultiplies to black. Found by the review of the
// change that made the mask's result straight colour.
//
// Java does not keep the colour everywhere, and neither may the Go version.
// Where the image is smaller than the mask, applyMask scales it with
// scaleImage, which draws it onto a new transparent TYPE_INT_ARGB image; a
// pixel with an alpha of 0 draws nothing there, so it is black when the soft
// mask's alpha goes onto it.
//
// The expected values are PDFBox's, printed by getImage().getRGB for each
// pixel, left to right and top to bottom. The pixels are (200 100 50),
// (100 150 200), (30 60 90) and (250 240 230), and /Mask [100 100 150 150 200
// 200] keys out the second; the soft mask's samples are 128, 64, 255 and 32.
func TestAKeyedPixelKeepsItsColourUnderASoftMask(t *testing.T) {
	pixels := []int{200, 100, 50, 100, 150, 200, 30, 60, 90, 250, 240, 230}
	key := []int64{100, 100, 150, 150, 200, 200}
	softMask := []int{128, 64, 255, 32}

	for _, c := range []struct {
		name string
		make func(t *testing.T) *cos.Stream
		want []string
	}{
		{"the soft mask's size", func(t *testing.T) *cos.Stream {
			image := testImage(t, 2, 2, cos.DeviceRGB, pixels)
			image.SetItem(cos.Mask, integers(key))
			image.SetItem(cos.SMask, testImage(t, 2, 2, cos.DeviceGray, softMask))
			return image
		}, []string{"80c86432", "406496c8", "ff1e3c5a", "20faf0e6"}},

		{"smaller than the soft mask", func(t *testing.T) *cos.Stream {
			image := testImage(t, 2, 1, cos.DeviceRGB, pixels[:6])
			image.SetItem(cos.Mask, integers(key))
			image.SetItem(cos.SMask, testImage(t, 4, 2, cos.DeviceGray,
				[]int{200, 100, 50, 25, 10, 20, 30, 40}))
			return image
		}, []string{
			"c8c86432", "64c86432", "32000000", "19000000",
			"0ac86432", "14c86432", "1e000000", "28000000",
		}},

		// The /Matte arithmetic reads the colour the keyed pixel kept.
		{"a matte", func(t *testing.T) *cos.Stream {
			image := testImage(t, 2, 2, cos.DeviceRGB, pixels)
			image.SetItem(cos.Mask, integers(key))
			mask := testImage(t, 2, 2, cos.DeviceGray, softMask)
			matte := cos.NewArray()
			for _, v := range []float32{0.5, 0.25, 0.75} {
				matte.Add(cos.NewFloat(v))
			}
			mask.SetItem(cos.Matte, matte)
			image.SetItem(cos.SMask, mask)
			return image
		}, []string{"80ff8800", "4012ffe2", "ff1e3c5a", "20ffffff"}},

		{"no soft mask", func(t *testing.T) *cos.Stream {
			image := testImage(t, 2, 2, cos.DeviceRGB, pixels)
			image.SetItem(cos.Mask, integers(key))
			return image
		}, []string{"ffc86432", "006496c8", "ff1e3c5a", "fffaf0e6"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ximage := NewPDImageXObject(common.NewPDStream(c.make(t)), nil)
			out, err := ximage.Image()
			if err != nil {
				t.Fatalf("Image: %v", err)
			}
			bounds := out.Bounds()
			var got []string
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					got = append(got, argbOf(out.At(x, y)))
				}
			}
			if fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Errorf("the image reads back as %v, and PDFBox's getRGB answers %v", got, c.want)
			}
		})
	}
}

// testImage is an image XObject of 8 bit samples, unfiltered.
func testImage(t *testing.T, width, height int, space *cos.Name, samples []int) *cos.Stream {
	t.Helper()
	stream := cos.NewStream(nil)
	stream.SetItem(cos.Type, cos.XObject)
	stream.SetItem(cos.Subtype, cos.Image)
	stream.SetInt(cos.Width, width)
	stream.SetInt(cos.Height, height)
	stream.SetInt(cos.BitsPerComponent, 8)
	stream.SetItem(cos.ColorSpace, space)
	data := make([]byte, len(samples))
	for i, sample := range samples {
		data[i] = byte(sample)
	}
	writer, err := stream.CreateRawWriter()
	if err != nil {
		t.Fatalf("writing the samples: %v", err)
	}
	if _, err := writer.Write(data); err != nil {
		t.Fatalf("writing the samples: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writing the samples: %v", err)
	}
	return stream
}

// integers is a COSArray of the given integers.
func integers(values []int64) *cos.Array {
	array := cos.NewArray()
	for _, v := range values {
		array.Add(cos.GetInteger(v))
	}
	return array
}

// argbOf formats a pixel as getRGB answers it: straight colour, alpha first,
// in hex.
func argbOf(c goimagecolor.Color) string {
	n := goimagecolor.NRGBAModel.Convert(c).(goimagecolor.NRGBA)
	return fmt.Sprintf("%02x%02x%02x%02x", n.A, n.R, n.G, n.B)
}
