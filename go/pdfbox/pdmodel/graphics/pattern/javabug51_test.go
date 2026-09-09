package pattern

// JAVA-BUGS 51: the `/Pattern` arm of `color.PDColorSpace.create` builds the
// underlying colour space of an uncoloured tiling pattern with the
// one-argument `create`, which passes no resources, though it hands the same
// resources to the `PDPattern` on the very same line.

import (
	"errors"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// namedResources is the smallest thing that answers a colour space by name,
// which is what a page's /ColorSpace dictionary is.
type namedResources struct{ spaces map[string]color.PDColorSpace }

func (r namedResources) HasColorSpace(name *cos.Name) bool {
	_, present := r.spaces[name.Name()]
	return present
}

func (r namedResources) ColorSpaceOfName(name *cos.Name, wasDefault bool) (color.PDColorSpace, error) {
	space, present := r.spaces[name.Name()]
	if !present {
		return nil, errors.New("color: no such colour space")
	}
	return space, nil
}

func (r namedResources) CachedColorSpace(object *cos.Object) color.PDColorSpace { return nil }

func (r namedResources) CacheColorSpace(object *cos.Object, space color.PDColorSpace) {}

// TestPatternResolvesItsUnderlyingSpaceThroughTheResources is the defect.
//
// `[/Pattern /CS1]` is an uncoloured tiling pattern whose underlying colour
// space is named in the page's /ColorSpace dictionary. The resources are in
// this method's hand -- the PDPattern built beside it is given them -- so the
// underlying space is resolved with them, which is what the three sibling
// recursions of this method (indexed, separation, DeviceN) all do. Java builds
// it with the one-argument create and throws MissingResourceException.
func TestPatternResolvesItsUnderlyingSpaceThroughTheResources(t *testing.T) {
	resources := namedResources{spaces: map[string]color.PDColorSpace{"CS1": color.DeviceRGB}}
	array := cos.NewArrayOf([]cos.Base{cos.Pattern, cos.GetPDFName("CS1")})

	space, err := color.CreateWithResources(array, resources, false)
	if err != nil {
		t.Fatalf("CreateWithResources: %v", err)
	}
	pattern, isPattern := space.(*PDPattern)
	if !isPattern {
		t.Fatalf("the colour space is %T, want a pattern", space)
	}
	if got := pattern.UnderlyingColorSpace(); got != color.DeviceRGB {
		t.Errorf("the underlying colour space is %v, want the color.DeviceRGB /CS1 names", got)
	}
}

// TestPatternWithAnInlineUnderlyingSpaceIsUnchanged keeps the case that
// already worked, where the underlying space carries itself.
func TestPatternWithAnInlineUnderlyingSpaceIsUnchanged(t *testing.T) {
	array := cos.NewArrayOf([]cos.Base{cos.Pattern, cos.DeviceGray})

	space, err := color.CreateWithResources(array, nil, false)
	if err != nil {
		t.Fatalf("CreateWithResources: %v", err)
	}
	pattern, isPattern := space.(*PDPattern)
	if !isPattern {
		t.Fatalf("the colour space is %T, want a pattern", space)
	}
	if got := pattern.UnderlyingColorSpace(); got != color.DeviceGray {
		t.Errorf("the underlying colour space is %v, want color.DeviceGray", got)
	}
}
