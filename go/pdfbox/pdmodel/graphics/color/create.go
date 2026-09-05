package color

import (
	"errors"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// ResourcesLike is what a colour space asks of the resource dictionary it was
// found in.
//
// Java's PDColorSpace.create takes a PDResources, and PDResources.getColorSpace
// calls back into create. In Go that is an import cycle, so the colour spaces
// name what they need and pdmodel.PDResources satisfies it -- the same shape
// slice 5 used to break the cycle between the security handlers and PDDocument.
type ResourcesLike interface {
	// HasColorSpace reports whether the given colour space name is in these
	// resources.
	HasColorSpace(name *cos.Name) bool

	// ColorSpaceOfName returns the colour space of the given name.
	ColorSpaceOfName(name *cos.Name, wasDefault bool) (PDColorSpace, error)

	// CachedColorSpace returns the colour space cached for the given indirect
	// object, or nil where there is none or there is no cache.
	CachedColorSpace(object *cos.Object) PDColorSpace

	// CacheColorSpace records a colour space against its indirect object.
	CacheColorSpace(object *cos.Object, space PDColorSpace)
}

// ErrMissingResource is returned where a colour space names a resource that is
// not there.
//
// Port of java.util.MissingResourceException, which PDColorSpace.create throws
// and which is unchecked; the port returns it, because every caller of create
// already handles an error and a panic would be harder to recover from than
// the IOException beside it.
var ErrMissingResource = errors.New("color: missing resource")

// ErrColorSpaceNotPorted is returned for the two colour spaces slice 6 leaves
// to slice 9: /Pattern, which builds pattern dictionaries that only rendering
// reads, and the JPX colour space, which only the JPX filter constructs and
// which needs a JPEG 2000 decoder Go has not got.
var ErrColorSpaceNotPorted = errors.New("color: colour space is not ported yet")

// Create returns the colour space the given COS object names.
//
// Port of PDColorSpace.create(COSBase). Abbreviated device color names are not
// supported here, please replace them first.
func Create(colorSpace cos.Base) (PDColorSpace, error) {
	return CreateWithResources(colorSpace, nil, false)
}

// CreateOfResources returns the colour space the given COS object names,
// resolving a bare name through the given resources.
//
// Port of PDColorSpace.create(COSBase, PDResources).
func CreateOfResources(colorSpace cos.Base, resources ResourcesLike) (PDColorSpace, error) {
	return CreateWithResources(colorSpace, resources, false)
}

// CreateWithResources is Java's three argument create, which is for PDFBox
// internal use only; others should use CreateOfResources.
//
// wasDefault says whether the current colour space was reached through a
// default colour space, which stops the lookup recursing into itself.
func CreateWithResources(colorSpace cos.Base, resources ResourcesLike,
	wasDefault bool) (PDColorSpace, error) {
	switch space := colorSpace.(type) {
	case *cos.Object:
		return createFromCOSObject(space, resources)

	case *cos.Name:
		return createFromName(space, resources, wasDefault)

	case *cos.Array:
		return createFromArray(space, resources, wasDefault)

	case *cos.Dictionary:
		if !space.ContainsKey(cos.ColorSpace) {
			break
		}
		// PDFBOX-4833: dictionary with /ColorSpace entry
		base := space.GetDictionaryObject(cos.ColorSpace)
		if base == cos.Base(space) {
			// PDFBOX-5315
			return nil, fmt.Errorf("Recursion in colorspace: %v points to itself",
				space.GetItem(cos.ColorSpace))
		}
		return CreateWithResources(base, resources, wasDefault)
	}
	return nil, fmt.Errorf("Expected a name or array but got: %v", colorSpace)
}

func createFromName(name *cos.Name, resources ResourcesLike,
	wasDefault bool) (PDColorSpace, error) {
	// default color spaces
	if resources != nil {
		var defaultName *cos.Name
		switch {
		case name == cos.DeviceCMYK && resources.HasColorSpace(cos.DefaultCMYK):
			defaultName = cos.DefaultCMYK
		case name == cos.DeviceRGB && resources.HasColorSpace(cos.DefaultRGB):
			defaultName = cos.DefaultRGB
		case name == cos.DeviceGray && resources.HasColorSpace(cos.DefaultGray):
			defaultName = cos.DefaultGray
		}

		if defaultName != nil && resources.HasColorSpace(defaultName) && !wasDefault {
			return resources.ColorSpaceOfName(defaultName, true)
		}
	}

	// built-in color spaces
	switch name {
	case cos.DeviceCMYK:
		return DeviceCMYK, nil
	case cos.DeviceRGB:
		return DeviceRGB, nil
	case cos.DeviceGray:
		return DeviceGray, nil
	case cos.Pattern:
		return patternColorSpaceOf(resources, nil)
	}
	if resources != nil {
		if !resources.HasColorSpace(name) {
			return nil, fmt.Errorf("%w: Missing color space: %s", ErrMissingResource, name.Name())
		}
		return resources.ColorSpaceOfName(name, false)
	}
	return nil, fmt.Errorf("%w: Unknown color space: %s", ErrMissingResource, name.Name())
}

func createFromArray(array *cos.Array, resources ResourcesLike,
	wasDefault bool) (PDColorSpace, error) {
	if array.IsEmpty() {
		return nil, errors.New("Colorspace array is empty")
	}
	base := array.GetObject(0)
	name, ok := base.(*cos.Name)
	if !ok {
		return nil, errors.New("First element in colorspace array must be a name")
	}

	// TODO cache these returned color spaces?

	switch name {
	case cos.CalGray:
		return NewPDCalGrayOfArray(array), nil
	case cos.CalRGB:
		return NewPDCalRGBOfArray(array), nil
	case cos.DeviceN:
		return NewPDDeviceNOfArray(array, resources)
	case cos.Indexed:
		return NewPDIndexedOfArray(array, resources)
	case cos.Separation:
		return NewPDSeparationOfArray(array, resources)
	case cos.ICCBased:
		return NewPDICCBased(array, resources)
	case cos.Lab:
		return NewPDLabOfArray(array), nil
	case cos.Pattern:
		if array.Size() == 1 {
			return patternColorSpaceOf(resources, nil)
		}
		// Java reads the entry with get rather than getObject, so an indirect
		// underlying colour space reaches create as a COSObject.
		underlying, err := Create(array.Get(1))
		if err != nil {
			return nil, err
		}
		return patternColorSpaceOf(resources, underlying)
	case cos.DeviceCMYK, cos.DeviceRGB, cos.DeviceGray:
		// not allowed in an array, but we sometimes encounter these regardless
		return createFromName(name, resources, wasDefault)
	}
	return nil, fmt.Errorf("Invalid color space kind: %s", name.Name())
}

func createFromCOSObject(colorSpace *cos.Object, resources ResourcesLike) (PDColorSpace, error) {
	if resources == nil {
		return CreateWithResources(colorSpace.Object(), nil, false)
	}
	if cached := resources.CachedColorSpace(colorSpace); cached != nil {
		return cached, nil
	}
	cs, err := CreateOfResources(colorSpace.Object(), resources)
	if err != nil {
		return nil, err
	}
	if cs != nil {
		resources.CacheColorSpace(colorSpace, cs)
	}
	return cs, nil
}

// NewPatternColorSpace builds the /Pattern colour space. graphics/pattern sets
// it from its init.
//
// Java's PDPattern lives in this package and imports
// graphics.pattern.PDAbstractPattern and pdmodel.PDResources. Go forbids that
// direction -- pattern imports pdmodel, which imports this package -- so the
// class lives in graphics/pattern and reaches Create through here, which is
// the registry device migration/conventions/java-to-go.md describes.
//
// resources is the PDResources the pattern is looked up in, which Create is
// handed as a ResourcesLike; underlying is the colour space of an uncoloured
// tiling pattern, and nil for a coloured one.
var NewPatternColorSpace func(resources ResourcesLike, underlying PDColorSpace) PDColorSpace

// patternColorSpaceOf is the two /Pattern arms of Create, which answer the
// registered colour space where graphics/pattern is linked in and say so
// where it is not.
func patternColorSpaceOf(resources ResourcesLike, underlying PDColorSpace) (PDColorSpace, error) {
	if NewPatternColorSpace == nil {
		return nil, fmt.Errorf("%w: /Pattern", ErrColorSpaceNotPorted)
	}
	return NewPatternColorSpace(resources, underlying), nil
}
