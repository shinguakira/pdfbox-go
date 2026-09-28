package raster

import (
	"fmt"
	goimage "image"
	goimagecolor "image/color"
	"strings"
	"testing"
)

// TestScaledInstanceIsJavas holds scaledInstance to what the JDK's
// getScaledInstance(w, h, SCALE_SMOOTH) answers for the three image types
// PDFBox draws: TYPE_INT_RGB, TYPE_INT_ARGB and TYPE_BYTE_GRAY.
//
// The expected values are the JDK's, printed through a PixelGrabber by a
// scratch program for a 7 by 5 image of each type made by probeImage's
// arithmetic. The grey case goes through getRGB's linear-to-sRGB conversion
// before it is averaged, which is why its greys are lighter than the samples.
func TestScaledInstanceIsJavas(t *testing.T) {
	for _, c := range []struct {
		kind string
		w, h int
		want string
	}{
		{"rgb", 3, 2, "ff232a1f ff783254 ffcc459d ff3ea45e ff92ad5e ffbbc072"},
		{"rgb", 2, 4, "ff320d26 ffb1218a ff3f4a2b ffbe5f8f ff4c8730 ffcb9c93 ff59c585 ff9ed949"},
		{"rgb", 5, 3, "ff0f150d ff3f172f ff731e52 ffa82882 ffd835ab ff21663c ff506929 " +
			"ff856f31 ffba7ab0 ffe98797 ff32b86a ff62ba72 ff97c173 ffcbcb54 ff8dd85b"},
		{"argb", 3, 2, "27383429 827f365a c8cc41a3 4649aa61 a298b062 8fcdbe74"},
		{"argb", 4, 3, "16261c11 5b5b1c44 9f972575 e4d435a6 2b306d41 706b6e1c " +
			"b4a8778e 84d779b7 403ebb6e 857cbe79 c9b9c85b 7bcdd355"},
		{"gray", 3, 2, "ff747474 ffbbbbbb ffd5d5d5 ffd5d5d5 ffb2b2b2 ff949494"},
	} {
		t.Run(fmt.Sprintf("%s to %dx%d", c.kind, c.w, c.h), func(t *testing.T) {
			scaled := scaledInstance(probeImage(c.kind), c.w, c.h)
			var got []string
			for y := 0; y < c.h; y++ {
				for x := 0; x < c.w; x++ {
					n := goimagecolor.NRGBAModel.Convert(scaled.At(x, y)).(goimagecolor.NRGBA)
					got = append(got, fmt.Sprintf("%02x%02x%02x%02x", n.A, n.R, n.G, n.B))
				}
			}
			if strings.Join(got, " ") != c.want {
				t.Errorf("got  %s\nwant %s", strings.Join(got, " "), c.want)
			}
		})
	}
}

// probeImage is the 7 by 5 image the Java was given, in the type the port
// holds each of Java's types in.
func probeImage(kind string) goimage.Image {
	bounds := goimage.Rect(0, 0, 7, 5)
	rgba := goimage.NewRGBA(bounds)
	nrgba := goimage.NewNRGBA(bounds)
	gray := goimage.NewGray(bounds)
	for y := 0; y < 5; y++ {
		for x := 0; x < 7; x++ {
			r, g, b := uint8(x*37+y*11), uint8(x*x+y*3*17), uint8((x^y)*29)
			rgba.SetRGBA(x, y, goimagecolor.RGBA{R: r, G: g, B: b, A: 0xFF})
			nrgba.SetNRGBA(x, y, goimagecolor.NRGBA{R: r, G: g, B: b, A: uint8(x*40 + y*13)})
			gray.SetGray(x, y, goimagecolor.Gray{Y: uint8(x*31 + y*47)})
		}
	}
	switch kind {
	case "argb":
		return nrgba
	case "gray":
		return gray
	}
	return rgba
}

// TestTheGreyTableIsJavas checks grayToSRGB against what the JDK's getRGB
// answers for a TYPE_BYTE_GRAY pixel, printed by a scratch program: the ends
// stay where they are, and the middle comes up the sRGB curve.
func TestTheGreyTableIsJavas(t *testing.T) {
	for grey, want := range map[int]uint8{0: 0, 64: 0x89, 128: 0xbc, 200: 0xe5, 255: 0xff} {
		if got := grayToSRGB[grey]; got != want {
			t.Errorf("grey %d reads as %d, and TYPE_BYTE_GRAY's getRGB answers %d", grey, got, want)
		}
	}
}
