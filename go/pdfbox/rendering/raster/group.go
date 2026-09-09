package raster

// Transparency groups and soft masks.
//
// Port of GroupGraphics and of what PageDrawer.processTransparencyGroup does
// around it: the group's contents are drawn onto a surface of their own and
// composited as one, so the alpha constant and the blend mode apply to the
// group rather than to each thing inside it.
//
// **The backdrop is the hard part, and it is Java's design that is copied.** A
// group that is neither isolated nor a knockout and uses a blend mode has to
// see what is behind it, so its contents are drawn onto a copy of the
// destination. But then the backdrop is in the result twice, once from the
// copy and once from compositing the group back. Java draws the group a second
// time onto a transparent surface to learn the alpha its own contents have,
// and then removes the backdrop with
//
//	C = Cn + (Cn - C0) * (alpha0 / alphagn - alpha0)
//
// which is `GroupGraphics.removeBackdrop`. The port keeps both surfaces the
// same way.

import (
	goimage "image"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// groupFrame is one open transparency group.
type groupFrame struct {
	// saved is the surface the group is composited back onto.
	saved *goimage.RGBA

	// backdrop is the copy of that surface the group was started from, and is
	// nil for an isolated group. alphaOnly is the second surface, drawn in
	// parallel, that says what alpha the group's own contents have.
	backdrop  *goimage.RGBA
	alphaOnly *goimage.RGBA

	// The state PushGroup takes away and PopGroup gives back: a group is
	// composited with the alpha and the mode that were in force when it began,
	// and its contents are drawn with neither.
	blendMode     *blend.BlendMode
	alphaConstant float64
	isSoftMask    bool

	// clip is where the group may be composited, which is the intersection of
	// its bounding box with the clip in force.
	clip *goimage.Alpha
}

// PushGroup begins a transparency group over the given box in user space.
func (i *Image) PushGroup(bbox *common.PDRectangle, isSoftMask, needsBackdrop bool,
	backdropColor *color.PDColor) error {
	bounds := i.dst.Bounds()
	frame := groupFrame{
		saved:         i.dst,
		blendMode:     i.blendMode,
		alphaConstant: i.alphaConstant,
		isSoftMask:    isSoftMask,
		clip:          i.groupClip(bbox),
	}

	group := goimage.NewRGBA(bounds)
	if needsBackdrop {
		frame.backdrop = goimage.NewRGBA(bounds)
		copy(frame.backdrop.Pix, i.dst.Pix)
		copy(group.Pix, i.dst.Pix)
		frame.alphaOnly = goimage.NewRGBA(bounds)
	}

	i.groups = append(i.groups, frame)
	i.dst = group
	i.secondary = frame.alphaOnly
	// The contents of a group are drawn as themselves; the alpha and the mode
	// belong to the group as a whole and are applied when it is composited.
	i.blendMode = nil
	i.alphaConstant = 1
	i.clipMask = nil
	return nil
}

// groupClip is where a group may be composited: its bounding box in device
// space, narrowed by the clip in force.
func (i *Image) groupClip(bbox *common.PDRectangle) *goimage.Alpha {
	bounds := i.dst.Bounds()
	clip := i.clipCoverage()
	if bbox == nil {
		return clip
	}
	box := coverageOf(geom.NewRectangle2D(
		float64(bbox.LowerLeftX()), float64(bbox.LowerLeftY()),
		float64(bbox.Width()), float64(bbox.Height())),
		i.transform, bounds.Dx(), bounds.Dy(), true)
	if clip == nil {
		return box
	}
	for index := range box.Pix {
		box.Pix[index] = uint8(int(box.Pix[index]) * int(clip.Pix[index]) / 255)
	}
	return box
}

// PopGroup composites the group PushGroup began and ends it.
func (i *Image) PopGroup() error {
	if len(i.groups) == 0 {
		return errNoGroup
	}
	frame := i.groups[len(i.groups)-1]
	i.groups = i.groups[:len(i.groups)-1]

	group := i.dst
	i.dst = frame.saved
	i.blendMode = frame.blendMode
	i.alphaConstant = frame.alphaConstant
	i.secondary = nil
	i.clipMask = nil
	if len(i.groups) > 0 {
		i.secondary = i.groups[len(i.groups)-1].alphaOnly
	}

	if frame.backdrop != nil {
		removeBackdrop(group, frame.alphaOnly, frame.backdrop)
	}

	bounds := i.dst.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := group.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			alpha := float64(c.A) / 255 * frame.alphaConstant
			if frame.clip != nil {
				alpha *= float64(frame.clip.AlphaAt(x, y).A) / 255
			}
			if alpha == 0 {
				continue
			}
			i.blendPixel(x, y, unpremultiply(c), alpha)
		}
	}
	return nil
}

// removeBackdrop is GroupGraphics.removeBackdrop: the group was drawn onto a
// copy of the backdrop, alphaOnly says what alpha its own contents have, and
// this takes the backdrop back out.
func removeBackdrop(group, alphaOnly, backdrop *goimage.RGBA) {
	for index := 0; index < len(group.Pix); index += 4 {
		// alphagn is the total alpha of the group contents excluding backdrop.
		alphagn := int(alphaOnly.Pix[index+3])
		if alphagn == 0 {
			// Avoid division by 0 and set the result to fully transparent.
			group.Pix[index], group.Pix[index+1] = 0, 0
			group.Pix[index+2], group.Pix[index+3] = 0, 0
			continue
		}
		alpha0 := float32(backdrop.Pix[index+3])
		alphaFactor := alpha0/float32(alphagn) - alpha0/255

		for k := 0; k < 3; k++ {
			cn := float32(group.Pix[index+k])
			c0 := float32(backdrop.Pix[index+k])
			group.Pix[index+k] = clampToByteValue(cn + (cn-c0)*alphaFactor)
		}
		group.Pix[index+3] = uint8(alphagn)
	}
}

// clampToByteValue rounds a 0..255 value and holds it in range, which is
// backdropRemoval's own `Math.round` and clamp.
func clampToByteValue(v float32) uint8 {
	rounded := int(v + 0.5)
	if v < 0 {
		rounded = int(v - 0.5)
	}
	switch {
	case rounded < 0:
		return 0
	case rounded > 255:
		return 255
	}
	return uint8(rounded)
}
