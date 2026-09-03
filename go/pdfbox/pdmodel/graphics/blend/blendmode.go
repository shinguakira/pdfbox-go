// Package blend holds the blend modes that say how paint is combined with what
// is already on the page.
//
// Port of org.apache.pdfbox.pdmodel.graphics.blend.
package blend

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// BlendChannelFunction is the blend operation of a separable blend mode, which
// works on one channel at a time.
type BlendChannelFunction func(src, dest float32) float32

// BlendFunction is the blend operation of a non-separable blend mode, which
// needs all three channels at once. The result is written into result.
type BlendFunction func(src, dest, result []float32)

// Functions for the blend operation of separable blend modes.
var (
	fNormal BlendChannelFunction = func(src, dest float32) float32 { return src }

	fMultiply BlendChannelFunction = func(src, dest float32) float32 { return src * dest }

	fScreen BlendChannelFunction = func(src, dest float32) float32 { return src + dest - src*dest }

	fOverlay BlendChannelFunction = func(src, dest float32) float32 {
		if dest <= 0.5 {
			return 2 * dest * src
		}
		return 2*(src+dest-src*dest) - 1
	}

	fDarken BlendChannelFunction = func(src, dest float32) float32 { return min(src, dest) }

	fLighten BlendChannelFunction = func(src, dest float32) float32 { return max(src, dest) }

	fColorDodge BlendChannelFunction = func(src, dest float32) float32 {
		// See PDF 2.0 specification
		if dest == 0 {
			return 0
		}
		if dest >= 1-src {
			return 1
		}
		return dest / (1 - src)
	}

	fColorBurn BlendChannelFunction = func(src, dest float32) float32 {
		// See PDF 2.0 specification
		if dest == 1 {
			return 1
		}
		if 1-dest >= src {
			return 0
		}
		return 1 - (1-dest)/src
	}

	fHardLight BlendChannelFunction = func(src, dest float32) float32 {
		if src <= 0.5 {
			return 2 * dest * src
		}
		return 2*(src+dest-src*dest) - 1
	}

	fSoftLight BlendChannelFunction = func(src, dest float32) float32 {
		if src <= 0.5 {
			return dest - (1-2*src)*dest*(1-dest)
		}
		var d float32
		if dest <= 0.25 {
			d = ((16*dest-12)*dest + 4) * dest
		} else {
			d = float32(math.Sqrt(float64(dest)))
		}
		return dest + (2*src-1)*(d-dest)
	}

	fDifference BlendChannelFunction = func(src, dest float32) float32 {
		return float32(math.Abs(float64(dest - src)))
	}

	fExclusion BlendChannelFunction = func(src, dest float32) float32 {
		return dest + src - 2*dest*src
	}
)

// Functions for the blend operation of non-separable blend modes.
var (
	fHue BlendFunction = func(src, dest, result []float32) {
		temp := make([]float32, 3)
		getSaturationRGB(dest, src, temp)
		getLuminosityRGB(dest, temp, result)
	}

	fSaturation BlendFunction = getSaturationRGB

	fColor BlendFunction = func(src, dest, result []float32) {
		getLuminosityRGB(dest, src, result)
	}

	fLuminosity BlendFunction = getLuminosityRGB
)

// Separable blend modes as defined in the PDF specification.
var (
	Normal     = newBlendMode(cos.Normal, fNormal, nil)
	Compatible = Normal
	Multiply   = newBlendMode(cos.Multiply, fMultiply, nil)
	Screen     = newBlendMode(cos.Screen, fScreen, nil)
	Overlay    = newBlendMode(cos.Overlay, fOverlay, nil)
	Darken     = newBlendMode(cos.Darken, fDarken, nil)
	Lighten    = newBlendMode(cos.Lighten, fLighten, nil)
	ColorDodge = newBlendMode(cos.ColorDodge, fColorDodge, nil)
	ColorBurn  = newBlendMode(cos.ColorBurn, fColorBurn, nil)
	HardLight  = newBlendMode(cos.HardLight, fHardLight, nil)
	SoftLight  = newBlendMode(cos.SoftLight, fSoftLight, nil)
	Difference = newBlendMode(cos.Difference, fDifference, nil)
	Exclusion  = newBlendMode(cos.Exclusion, fExclusion, nil)
)

// Non-separable blend modes as defined in the PDF specification.
var (
	Hue        = newBlendMode(cos.Hue, nil, fHue)
	Saturation = newBlendMode(cos.Saturation, nil, fSaturation)
	Color      = newBlendMode(cos.Color, nil, fColor)
	Luminosity = newBlendMode(cos.Luminosity, nil, fLuminosity)
)

var blendModes = map[*cos.Name]*BlendMode{
	cos.Normal: Normal,
	// Compatible should not be used
	cos.Compatible: Normal,
	cos.Multiply:   Multiply,
	cos.Screen:     Screen,
	cos.Overlay:    Overlay,
	cos.Darken:     Darken,
	cos.Lighten:    Lighten,
	cos.ColorDodge: ColorDodge,
	cos.ColorBurn:  ColorBurn,
	cos.HardLight:  HardLight,
	cos.SoftLight:  SoftLight,
	cos.Difference: Difference,
	cos.Exclusion:  Exclusion,
	cos.Hue:        Hue,
	cos.Saturation: Saturation,
	cos.Luminosity: Luminosity,
	cos.Color:      Color,
}

// BlendMode says how paint is combined with what is already on the page.
//
// Port of org.apache.pdfbox.pdmodel.graphics.blend.BlendMode. As in Java the
// set of modes is fixed and each is a single shared value, so two modes are
// equal exactly when they are the same value.
type BlendMode struct {
	name         *cos.Name
	blendChannel BlendChannelFunction
	blend        BlendFunction
	isSeparable  bool
}

func newBlendMode(name *cos.Name, blendChannel BlendChannelFunction, blend BlendFunction) *BlendMode {
	return &BlendMode{
		name:         name,
		blendChannel: blendChannel,
		blend:        blend,
		isSeparable:  blendChannel != nil,
	}
}

// COSName returns the blend mode name from the BM object.
func (b *BlendMode) COSName() *cos.Name { return b.name }

// IsSeparableBlendMode reports whether the blend mode works on one channel at a
// time.
func (b *BlendMode) IsSeparableBlendMode() bool { return b.isSeparable }

// BlendChannelFunction returns the blend channel function, only available for
// separable blend modes.
func (b *BlendMode) BlendChannelFunction() BlendChannelFunction { return b.blendChannel }

// BlendFunction returns the blend function, only available for non-separable
// blend modes.
func (b *BlendMode) BlendFunction() BlendFunction { return b.blend }

// GetInstance determines the blend mode from the BM entry in the COS ExtGState,
// which is a name or an array of them. An entry naming nothing known gives
// Normal.
func GetInstance(cosBlendMode cos.Base) *BlendMode {
	var result *BlendMode
	switch value := cosBlendMode.(type) {
	case *cos.Name:
		result = blendModes[value]
	case *cos.Array:
		for i := 0; i < value.Size(); i++ {
			name, ok := value.GetObject(i).(*cos.Name)
			if !ok {
				continue
			}
			if result = blendModes[name]; result != nil {
				break
			}
		}
	}
	if result != nil {
		return result
	}
	return Normal
}

// get255Value scales a component to the 0..255 the blend arithmetic works in.
func get255Value(val float32) int {
	if val >= 1.0 {
		return 255
	}
	return int(math.Floor(float64(val) * 255.0))
}

func getSaturationRGB(srcValues, dstValues, result []float32) {
	rd := get255Value(dstValues[0])
	gd := get255Value(dstValues[1])
	bd := get255Value(dstValues[2])

	minb := min(rd, gd, bd)
	maxb := max(rd, gd, bd)
	if minb == maxb {
		// backdrop has zero saturation, avoid divide by 0
		result[0] = float32(gd) / 255.0
		result[1] = float32(gd) / 255.0
		result[2] = float32(gd) / 255.0
		return
	}

	rs := get255Value(srcValues[0])
	gs := get255Value(srcValues[1])
	bs := get255Value(srcValues[2])

	mins := min(rs, gs, bs)
	maxs := max(rs, gs, bs)

	scale := ((maxs - mins) << 16) / (maxb - minb)
	y := (rd*77 + gd*151 + bd*28 + 0x80) >> 8
	r := y + ((((rd - y) * scale) + 0x8000) >> 16)
	g := y + ((((gd - y) * scale) + 0x8000) >> 16)
	b := y + ((((bd - y) * scale) + 0x8000) >> 16)

	if ((r | g | b) & 0x100) == 0x100 {
		var scalemin, scalemax int

		lowest := min(r, g, b)
		highest := max(r, g, b)

		if lowest < 0 {
			scalemin = (y << 16) / (y - lowest)
		} else {
			scalemin = 0x10000
		}

		if highest > 255 {
			scalemax = ((255 - y) << 16) / (highest - y)
		} else {
			scalemax = 0x10000
		}

		scale = min(scalemin, scalemax)
		r = y + (((r-y)*scale + 0x8000) >> 16)
		g = y + (((g-y)*scale + 0x8000) >> 16)
		b = y + (((b-y)*scale + 0x8000) >> 16)
	}
	result[0] = float32(r) / 255.0
	result[1] = float32(g) / 255.0
	result[2] = float32(b) / 255.0
}

func getLuminosityRGB(srcValues, dstValues, result []float32) {
	rd := get255Value(dstValues[0])
	gd := get255Value(dstValues[1])
	bd := get255Value(dstValues[2])
	rs := get255Value(srcValues[0])
	gs := get255Value(srcValues[1])
	bs := get255Value(srcValues[2])
	delta := ((rs-rd)*77 + (gs-gd)*151 + (bs-bd)*28 + 0x80) >> 8
	r := rd + delta
	g := gd + delta
	b := bd + delta

	if ((r | g | b) & 0x100) == 0x100 {
		var scale int
		y := (rs*77 + gs*151 + bs*28 + 0x80) >> 8
		if delta > 0 {
			highest := max(r, g, b)
			if highest == y {
				scale = 0
			} else {
				scale = ((255 - y) << 16) / (highest - y)
			}
		} else {
			lowest := min(r, g, b)
			if y == lowest {
				scale = 0
			} else {
				scale = (y << 16) / (y - lowest)
			}
		}
		r = y + (((r-y)*scale + 0x8000) >> 16)
		g = y + (((g-y)*scale + 0x8000) >> 16)
		b = y + (((b-y)*scale + 0x8000) >> 16)
	}
	result[0] = float32(r) / 255.0
	result[1] = float32(g) / 255.0
	result[2] = float32(b) / 255.0
}

// String returns the Java toString form.
func (b *BlendMode) String() string {
	separable := "false"
	if b.isSeparable {
		separable = "true"
	}
	return "BlendMode{name=" + b.name.Name() + ", isSeparable=" + separable + "}"
}
