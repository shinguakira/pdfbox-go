package rendering

// Rendering a soft mask's transparency group.
//
// Port of the isSoftMask arm of PageDrawer's TransparencyGroup inner class,
// and of TransparencyGroup.getOrigin. Java builds a BufferedImage here and
// hands it to a SoftMask paint; the port works out the same surface and lets a
// Backend make it, because what a mask is held in is the Backend's business
// and every decision on the way there is the drawer's.
//
// What is not here is what applySoftMaskToPaint does with the result: turning
// the group into a grey raster, rescaling it, and multiplying a paint's alpha
// by it. Those are pixels, and they are the Backend's -- see
// rendering/raster/softmask.go.

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
)

// SoftMaskLayer is a soft mask's transparency group, rendered.
type SoftMaskLayer struct {
	// Backend is what the group was drawn onto, made by the caller at the size
	// this decided on.
	Backend Backend

	// OriginX and OriginY are where the layer's first pixel sits on the
	// surface the masked paint will be drawn onto. Port of
	// TransparencyGroup.getOrigin.
	OriginX, OriginY float64

	// BackdropColor is the mask's /BC, kept only where
	// applySoftMaskToPaint would keep it: a Luminosity mask whose colour
	// space says the array is the right length. PDFBOX-5795.
	BackdropColor *color.PDColor

	// Luminosity says which of the two subtypes this is, which decides
	// whether the mask reads the group's luminosity or its alpha.
	Luminosity bool

	// Gray is TransparencyGroup.isGray of the group's colour space, and it
	// decides what the group was drawn into: Java makes a grey-plus-alpha
	// image for a grey group and an ARGB one for anything else, so a
	// Luminosity mask over a grey group never converts a colour at all.
	Gray bool

	// Adjust is what PageDrawer.adjustImage would redraw the mask through,
	// and AdjustWidth and AdjustHeight the size it would redraw it to. It is
	// nil where that method answers the mask unchanged, which is every page
	// whose transform is a plain scale.
	Adjust                    *geom.AffineTransform
	AdjustWidth, AdjustHeight int
}

// DrawSoftMask renders a soft mask's transparency group and answers the layer.
//
// newBackend is asked for a surface of the size this works out, transparent to
// begin with; the group is drawn onto it and it comes back in the layer.
//
// A nil layer with no error is an empty group, which is not an error: "Adobe
// Reader ignores empty softmasks instead of using bc color" -- see
// applySoftMaskToPaint, which answers the unmasked paint in that case.
func (d *PageDrawer) DrawSoftMask(softMask *state.PDSoftMask,
	newBackend func(width, height int) Backend) (*SoftMaskLayer, error) {
	groupLike := softMask.Group()
	if groupLike == nil {
		return nil, nil
	}
	group, isGroup := groupLike.(*form.PDTransparencyGroup)
	if !isGroup {
		// NewTransparencyGroup builds nothing else; a mask that answers
		// something else has no group as far as this is concerned.
		return nil, nil
	}
	layer := &SoftMaskLayer{Luminosity: softMask.SubType() == cos.Luminosity}
	groupColorSpace, err := groupColorSpaceOf(group)
	if err != nil {
		return nil, err
	}
	layer.Gray = isGray(groupColorSpace)
	if layer.Luminosity {
		// "If the subtype is Luminosity, the transparency group XObject G
		// shall be composited with a fully opaque backdrop whose colour is
		// everywhere defined by the soft-mask dictionary's BC entry."
		if backdropColorArray := softMask.BackdropColor(); backdropColorArray != nil {
			// PDFBOX-5795
			if groupColorSpace != nil &&
				groupColorSpace.NumberOfComponents() == backdropColorArray.Size() {
				layer.BackdropColor = color.NewPDColorOfCOSArray(backdropColorArray,
					groupColorSpace)
			}
		}
	}

	// get the CTM x Form Matrix transform
	ctm := softMask.InitialTransformationMatrix()
	bbox := d.transparencyGroupBox(group, ctm)
	if bbox == nil {
		// group is empty, don't bother
		return nil, nil
	}

	// apply the underlying device's DPI transform
	scaleX := float64(d.xformScalingFactorX)
	scaleY := float64(d.xformScalingFactorY)
	clipRect := geom.NewRectangle2D(float64(bbox.LowerLeftX()), float64(bbox.LowerLeftY()),
		float64(bbox.Width()), float64(bbox.Height()))
	bounds := transformedBounds(geom.NewAffineTransform(scaleX, 0, 0, scaleY, 0, 0), clipRect)

	minX := int(math.Floor(bounds.X))
	minY := int(math.Floor(bounds.Y))
	maxX := int(math.Floor(bounds.X+bounds.Width)) + 1
	maxY := int(math.Floor(bounds.Y+bounds.Height)) + 1
	width := maxX - minX
	height := maxY - minY
	if width <= 0 || height <= 0 {
		return nil, nil
	}

	backend := newBackend(width, height)
	if layer.BackdropColor != nil {
		// "If the subtype is Luminosity, the transparency group XObject G
		// shall be composited with a fully opaque backdrop whose colour is
		// everywhere defined by the soft-mask dictionary's BC entry."
		//
		// Java sets it as the background and clears, which replaces the
		// pixels; over a surface that is still transparent, painting it
		// opaque comes to the same thing.
		rgb, err := layer.BackdropColor.ToRGB()
		if err != nil {
			return nil, err
		}
		backend.SetTransform(geom.NewAffineTransform(1, 0, 0, 1, 0, 0))
		backend.SetComposite(nil, 1)
		backend.SetPaint(ColorPaint{
			Red:   float32((rgb>>16)&0xFF) / 255,
			Green: float32((rgb>>8)&0xFF) / 255,
			Blue:  float32(rgb&0xFF) / 255,
			Alpha: 1,
		})
		if err := backend.Fill(geom.NewRectangle2D(0, 0,
			float64(width), float64(height))); err != nil {
			return nil, err
		}
	}

	// The transform the group is drawn through, in the order Java builds it:
	//
	//	g.translate(0, image.getHeight());
	//	g.scale(1, -1);
	//	g.transform(xform);                            // the DPI scale
	//	g.translate(-clipRect.getX(), -clipRect.getY());
	//
	// A soft mask is never a knockout of the page, so there is no backdrop to
	// draw first and no GroupGraphics to wrap it in.
	at := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)
	at.Translate(0, float64(height))
	at.Scale(1, -1)
	at.Scale(scaleX, scaleY)
	at.Translate(-clipRect.X, -clipRect.Y)
	backend.SetTransform(at)

	savedFlipTG := d.flipTG
	d.flipTG = false
	savedPageSize := d.pageSize
	d.pageSize = common.NewPDRectangleOf(float32(float64(minX)/scaleX), float32(float64(minY)/scaleY),
		float32(bounds.Width/scaleX), float32(bounds.Height/scaleY))
	savedClipWindingRule := d.clipWindingRule
	d.clipWindingRule = -1
	savedLinePath := d.linePath
	d.linePath = geom.NewPathFloat()
	savedInitialClip := d.initialClip
	d.initialClip = nil
	savedLastClips, savedHasLastClips := d.lastClips, d.hasLastClips
	savedBackend := d.backend
	d.backend = backend
	d.setRenderingHints()

	err = d.ProcessSoftMask(group)

	// Java restores these in a finally, so they go back on the error path too.
	d.backend = savedBackend
	d.lastClips, d.hasLastClips = savedLastClips, savedHasLastClips
	d.initialClip = savedInitialClip
	d.clipWindingRule = savedClipWindingRule
	d.linePath = savedLinePath
	d.pageSize = savedPageSize
	d.flipTG = savedFlipTG
	if err != nil {
		return nil, err
	}

	layer.Backend = backend
	layer.OriginX, layer.OriginY = d.softMaskOrigin(minX, minY, width, height)
	layer.Adjust, layer.AdjustWidth, layer.AdjustHeight = d.softMaskAdjust(width, height)
	return layer, nil
}

// softMaskOrigin is TransparencyGroup.getOrigin: where the layer's first pixel
// lands on the surface the masked paint is drawn onto.
//
// It reads flipTG as it is now, not as it was while the group was drawn, which
// is what Java does -- the constructor puts flipTG back before getOrigin is
// ever called.
func (d *PageDrawer) softMaskOrigin(minX, minY, width, height int) (x, y float64) {
	scaleX := float64(d.xformScalingFactorX)
	scaleY := float64(d.xformScalingFactorY)

	var r *geom.Rectangle2D
	if d.flipTG {
		// Fixes PDFBOX-5966 and PDFBOX-5251, but not pdfium 1317, which has
		// similar PDF code.
		r = geom.NewRectangle2D(float64(minX), float64(minY), float64(width), float64(height))
	} else {
		// y-axis flip
		r = geom.NewRectangle2D(
			float64(minX)-float64(d.pageSize.LowerLeftX())*scaleX,
			(float64(d.pageSize.LowerLeftY())+float64(d.pageSize.Height()))*scaleY-
				float64(minY)-float64(height),
			float64(width), float64(height))
	}

	// Apply the underlying device's DPI transform, which "adjusts the
	// rectangle to the rotated image to put the soft mask at the correct
	// position".
	adjusted := d.xform.Clone()
	adjusted.Scale(1/scaleX, 1/scaleY)
	box := transformedBounds(adjusted, r)
	return box.X, box.Y
}

// transformedBounds is `at.createTransformedShape(r).getBounds2D()` for a
// rectangle: the box around its four transformed corners.
func transformedBounds(at *geom.AffineTransform, r *geom.Rectangle2D) *geom.Rectangle2D {
	corners := []float64{
		r.X, r.Y,
		r.X + r.Width, r.Y,
		r.X + r.Width, r.Y + r.Height,
		r.X, r.Y + r.Height,
	}
	at.TransformDoubles(corners, 0, corners, 0, 4)
	minX, minY := corners[0], corners[1]
	maxX, maxY := minX, minY
	for i := 2; i < len(corners); i += 2 {
		minX = math.Min(minX, corners[i])
		maxX = math.Max(maxX, corners[i])
		minY = math.Min(minY, corners[i+1])
		maxY = math.Max(maxY, corners[i+1])
	}
	return geom.NewRectangle2D(minX, minY, maxX-minX, maxY-minY)
}

// softMaskAdjust is PageDrawer.adjustImage, up to the redraw: the transform
// the mask is put back through and the size it comes out at.
//
//	AffineTransform at = new AffineTransform(xform);
//	at.scale(1.0 / xformScalingFactorX, 1.0 / xformScalingFactorY);
//	Rectangle2D transformedBounds = at.createTransformedShape(originalBounds)...
//	at.preConcatenate(getTranslateInstance(-minX, -minY));
//
// A nil transform means the method would have answered the mask unchanged,
// which is the case Java tests for: the size did not move and the transform is
// the identity. Every page whose transform is a plain scale is that case.
func (d *PageDrawer) softMaskAdjust(width, height int) (*geom.AffineTransform, int, int) {
	at := d.xform.Clone()
	at.Scale(1/float64(d.xformScalingFactorX), 1/float64(d.xformScalingFactorY))

	bounds := transformedBounds(at,
		geom.NewRectangle2D(0, 0, float64(width), float64(height)))
	at.PreConcatenate(geom.NewAffineTransform(1, 0, 0, 1, -bounds.X, -bounds.Y))

	adjustedWidth := int(math.Ceil(bounds.Width))
	adjustedHeight := int(math.Ceil(bounds.Height))
	if adjustedWidth == width && adjustedHeight == height && at.IsIdentity() {
		return nil, width, height
	}
	return at, adjustedWidth, adjustedHeight
}

// groupColorSpaceOf is `form.getGroup().getColorSpace(form.getResources())`.
//
// The narrowing is Go's, not Java's: graphics/form cannot import pdmodel, so
// the resources it hands back are typed more narrowly than the colour package
// asks for. What is behind them is always a *pdmodel.PDResources.
func groupColorSpaceOf(group *form.PDTransparencyGroup) (color.PDColorSpace, error) {
	resources, _ := group.Resources().(color.ResourcesLike)
	return group.Group().ColorSpace(resources)
}

// isGray is TransparencyGroup.isGray: a grey colour space, or an ICC-based one
// whose alternate is grey.
func isGray(colorSpace color.PDColorSpace) bool {
	switch c := colorSpace.(type) {
	case *color.PDDeviceGray:
		return true
	case *color.PDICCBased:
		// Java catches an IOException from getAlternateColorSpace and answers
		// false; the port reads the alternate that was parsed with the space,
		// so there is nothing left to fail here.
		_, gray := c.AlternateColorSpace().(*color.PDDeviceGray)
		return gray
	}
	return false
}
