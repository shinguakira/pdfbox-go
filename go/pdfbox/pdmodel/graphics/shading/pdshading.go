// Package shading holds the seven shading types and the geometry they are
// built from.
//
// Port of org.apache.pdfbox.pdmodel.graphics.shading.
//
// The Java package has two halves. One is the model and the arithmetic: what
// the shading dictionary says, the functions it evaluates, and the triangles
// and patches a mesh shading decomposes into. The other is the binding to
// Java2D -- ShadingPaint and the seven *Paint classes, ShadingContext and the
// seven *Context classes, which implement java.awt.Paint and
// java.awt.PaintContext and hand pixels to a Raster.
//
// This package is the first half. The second is the raster backend that slice
// 9 defers behind an interface, so PDShading.toPaint is not here and neither
// are the eighteen classes that implement it. See migration/STATUS.md.
package shading

import (
	"errors"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// ErrPatchMeshNotPorted is what NewPDShading answers for a Coons or a
// tensor-product patch mesh, which are the two shading types this slice has not
// reached. It is named rather than anonymous so that a caller can tell "this
// port does not do that yet" from "this file says something I do not
// understand".
var ErrPatchMeshNotPorted = errors.New("shading: the patch mesh shadings are not ported yet")

// The seven shading types.
const (
	// ShadingType1 is function based shading.
	ShadingType1 = 1
	// ShadingType2 is axial shading.
	ShadingType2 = 2
	// ShadingType3 is radial shading.
	ShadingType3 = 3
	// ShadingType4 is free-form Gouraud-shaded triangle meshes.
	ShadingType4 = 4
	// ShadingType5 is lattice-form Gouraud-shaded triangle meshes.
	ShadingType5 = 5
	// ShadingType6 is Coons patch meshes.
	ShadingType6 = 6
	// ShadingType7 is tensor-product patch meshes.
	ShadingType7 = 7
)

// Shading is what a concrete shading supplies.
//
// PDShading is an abstract class in Java; the port splits it into this
// interface for the two abstract methods and the embedded struct below for the
// state and the concrete ones, which is what
// migration/conventions/java-to-go.md says to do.
//
// toPaint is the third abstract method there and is not here: it answers a
// java.awt.Paint, which is the raster backend this slice defers.
type Shading interface {
	common.COSObjectable

	// Dictionary returns the shading dictionary, which getCOSObject narrows to
	// in Java.
	Dictionary() *cos.Dictionary

	// ShadingType returns which of the seven this is.
	ShadingType() int

	// Bounds returns a rectangle around the areas of this shading, or nil
	// where the type does not support one.
	Bounds(xform *geom.AffineTransform, matrix *util.Matrix) (*geom.Rectangle2D, error)
}

// PDShading carries the state and the concrete methods of a shading resource.
//
// Port of the non-abstract half of PDShading.
type PDShading struct {
	// stream is the shading dictionary when it is a stream, which the four mesh
	// types read their vertices out of.
	//
	// Java needs no such field: COSStream extends COSDictionary there, so a
	// mesh type recovers the stream with an instanceof on the dictionary it was
	// built from. cos.Stream embeds cos.Dictionary rather than extending it, so
	// a *cos.Dictionary cannot be narrowed back and the stream is kept here.
	stream *cos.Stream

	dictionary    *cos.Dictionary
	background    *cos.Array
	bBox          *common.PDRectangle
	colorSpace    color.PDColorSpace
	function      function.PDFunction
	functionArray []function.PDFunction
}

// InitShading is the protected PDShading() constructor. A concrete shading
// calls it from its own constructor.
func (s *PDShading) InitShading() { s.dictionary = cos.NewDictionary() }

// InitShadingOf is the protected PDShading(COSDictionary) constructor.
func (s *PDShading) InitShadingOf(shadingDictionary *cos.Dictionary) {
	s.dictionary = shadingDictionary
}

// InitShadingOfStream is InitShadingOf for a shading held as a stream, which
// the four mesh types are. See the stream field.
func (s *PDShading) InitShadingOfStream(shadingStream *cos.Stream) {
	s.stream = shadingStream
	s.dictionary = &shadingStream.Dictionary
}

// Stream returns the shading stream, or nil where the shading is a plain
// dictionary. It is the instanceof COSStream the mesh types make.
func (s *PDShading) Stream() *cos.Stream { return s.stream }

// COSObject returns the shading dictionary.
func (s *PDShading) COSObject() cos.Base { return s.dictionary }

// Dictionary returns the shading dictionary, typed.
func (s *PDShading) Dictionary() *cos.Dictionary { return s.dictionary }

// Type returns the type of object that this is.
func (s *PDShading) Type() string { return cos.Shading.Name() }

// SetShadingType sets the shading type.
func (s *PDShading) SetShadingType(shadingType int) {
	s.dictionary.SetInt(cos.ShadingType, shadingType)
}

// SetBackground sets the background.
func (s *PDShading) SetBackground(newBackground *cos.Array) {
	s.background = newBackground
	s.dictionary.SetItem(cos.Background, newBackground)
}

// Background returns the background, or nil where there is none.
func (s *PDShading) Background() *cos.Array {
	if s.background == nil {
		s.background = s.dictionary.GetCOSArray(cos.Background)
	}
	return s.background
}

// BBox returns the /BBox: four numbers in the form coordinate system giving the
// coordinates of the left, bottom, right and top edges of the shading's
// bounding box, or nil where there is none.
func (s *PDShading) BBox() *common.PDRectangle {
	if s.bBox == nil {
		if array := s.dictionary.GetCOSArray(cos.BBox); array != nil {
			s.bBox = common.NewPDRectangleOfCOSArray(array)
		}
	}
	return s.bBox
}

// SetBBox sets the bounding box for this shading, and removes it for nil.
func (s *PDShading) SetBBox(newBBox *common.PDRectangle) {
	s.bBox = newBBox
	if s.bBox == nil {
		s.dictionary.RemoveItem(cos.BBox)
		return
	}
	s.dictionary.SetItem(cos.BBox, s.bBox.COSArray())
}

// Bounds returns nil, which is what the base class answers: a shading type
// that can bound itself overrides this.
func (s *PDShading) Bounds(*geom.AffineTransform, *util.Matrix) (*geom.Rectangle2D, error) {
	return nil, nil
}

// SetAntiAlias sets the /AntiAlias value.
func (s *PDShading) SetAntiAlias(antiAlias bool) {
	s.dictionary.SetBoolean(cos.AntiAlias, antiAlias)
}

// AntiAlias returns the /AntiAlias value, false where there is none.
func (s *PDShading) AntiAlias() bool {
	return s.dictionary.GetBoolean(cos.AntiAlias, false)
}

// ColorSpace returns the colour space of the shading, or nil where there is
// none.
func (s *PDShading) ColorSpace() (color.PDColorSpace, error) {
	if s.colorSpace == nil {
		colorSpaceDictionary := s.dictionary.GetDictionaryObject2(cos.CS, cos.ColorSpace)
		created, err := color.Create(colorSpaceDictionary)
		if err != nil {
			return nil, err
		}
		s.colorSpace = created
	}
	return s.colorSpace, nil
}

// SetColorSpace sets the colour space for the shading, and removes it for nil.
func (s *PDShading) SetColorSpace(colorSpace color.PDColorSpace) {
	s.colorSpace = colorSpace
	if colorSpace != nil {
		s.dictionary.SetItem(cos.ColorSpace, colorSpace.COSObject())
		return
	}
	s.dictionary.RemoveItem(cos.ColorSpace)
}

// SetFunction sets the function for the colour conversion.
func (s *PDShading) SetFunction(newFunction function.PDFunction) {
	s.functionArray = nil
	s.function = newFunction
	// Java's setItem(COSName, COSObjectable) writes a null for a null
	// objectable, which removes the entry.
	if newFunction == nil {
		s.dictionary.SetItem(cos.Function, nil)
		return
	}
	s.dictionary.SetItem(cos.Function, newFunction.COSObject())
}

// SetFunctionArray sets the /Function array for the colour conversion.
//
// Port of the setFunction(COSArray) overload; Go has no overloading.
func (s *PDShading) SetFunctionArray(newFunctions *cos.Array) {
	s.functionArray = nil
	s.function = nil
	s.dictionary.SetItem(cos.Function, newFunctions)
}

// Function returns the function used to convert the colour values, or nil
// where the dictionary names none.
func (s *PDShading) Function() (function.PDFunction, error) {
	if s.function == nil {
		dictionaryFunctionObject := s.dictionary.GetDictionaryObject(cos.Function)
		if dictionaryFunctionObject != nil {
			created, err := function.NewPDFunction(dictionaryFunctionObject)
			if err != nil {
				return nil, err
			}
			s.function = created
		}
	}
	return s.function, nil
}

// functionsArray provides the functions of the shading dictionary as a slice.
//
// Port of the private getFunctionsArray.
func (s *PDShading) functionsArray() ([]function.PDFunction, error) {
	if s.functionArray != nil {
		return s.functionArray, nil
	}
	switch functionObject := s.dictionary.GetDictionaryObject(cos.Function).(type) {
	case *cos.Dictionary:
		created, err := function.NewPDFunction(functionObject)
		if err != nil {
			return nil, err
		}
		s.functionArray = []function.PDFunction{created}
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so its instanceof lets one
		// through here; a type 0 or type 4 function is always a stream.
		created, err := function.NewPDFunction(functionObject)
		if err != nil {
			return nil, err
		}
		s.functionArray = []function.PDFunction{created}
	case *cos.Array:
		numberOfFunctions := functionObject.Size()
		s.functionArray = make([]function.PDFunction, numberOfFunctions)
		for i := 0; i < numberOfFunctions; i++ {
			// Java reads the entry with get rather than getObject, so an
			// indirect reference reaches PDFunction.create as a COSObject.
			created, err := function.NewPDFunction(functionObject.Get(i))
			if err != nil {
				return nil, err
			}
			s.functionArray[i] = created
		}
	default:
		return nil, fmt.Errorf("mandatory /Function element must be a dictionary or an array")
	}
	return s.functionArray, nil
}

// EvalFunction converts one input value using the functions of the shading
// dictionary.
func (s *PDShading) EvalFunction(inputValue float32) ([]float32, error) {
	return s.EvalFunctionOfInput([]float32{inputValue})
}

// EvalFunctionOfInput converts the input values using the functions of the
// shading dictionary.
//
// Port of the evalFunction(float[]) overload.
func (s *PDShading) EvalFunctionOfInput(input []float32) ([]float32, error) {
	functions, err := s.functionsArray()
	if err != nil {
		return nil, err
	}
	numberOfFunctions := len(functions)
	var returnValues []float32
	if numberOfFunctions == 1 {
		returnValues, err = functions[0].Eval(input)
		if err != nil {
			return nil, err
		}
	} else {
		returnValues = make([]float32, numberOfFunctions)
		for i := 0; i < numberOfFunctions; i++ {
			newValue, err := functions[i].Eval(input)
			if err != nil {
				return nil, err
			}
			returnValues[i] = newValue[0]
		}
	}
	// From the PDF spec:
	// "If the value returned by the function for a given colour component
	// is out of range, it shall be adjusted to the nearest valid value."
	for i := range returnValues {
		if returnValues[i] < 0 {
			returnValues[i] = 0
		} else if returnValues[i] > 1 {
			returnValues[i] = 1
		}
	}
	return returnValues, nil
}

// NewPDShading creates the right shading for the given dictionary.
//
// Port of the static PDShading.create. Java declares the parameter as a
// COSDictionary and a mesh shading arrives as the COSStream that extends one;
// the port takes a cos.Base, because a Go *cos.Dictionary cannot be narrowed
// back to the stream it came from. See the stream field.
func NewPDShading(shadingBase cos.Base) (Shading, error) {
	var shadingDictionary *cos.Dictionary
	var shadingStream *cos.Stream
	switch value := shadingBase.(type) {
	case *cos.Stream:
		shadingStream = value
		shadingDictionary = &value.Dictionary
	case *cos.Dictionary:
		shadingDictionary = value
	default:
		return nil, fmt.Errorf("Error: Unknown shading type %v", shadingBase)
	}
	shadingType := shadingDictionary.GetIntDefault(cos.ShadingType, 0)
	switch shadingType {
	case ShadingType1:
		return NewPDShadingType1(shadingDictionary), nil
	case ShadingType2:
		return NewPDShadingType2(shadingDictionary), nil
	case ShadingType3:
		return NewPDShadingType3(shadingDictionary), nil
	case ShadingType4:
		return newPDShadingType4(shadingDictionary, shadingStream), nil
	case ShadingType5:
		return newPDShadingType5(shadingDictionary, shadingStream), nil
	case ShadingType6, ShadingType7:
		// Java builds a PDShadingType6 or a PDShadingType7 here. Neither is
		// ported yet: both need the patch machinery -- Patch, CoonsPatch,
		// TensorPatch and CubicBezierCurve -- which is the rest of B7. The
		// error names the gap rather than letting the caller read a type 6
		// mesh as an unknown one. See migration/STATUS.md.
		return nil, fmt.Errorf("%w: shading type %d", ErrPatchMeshNotPorted, shadingType)
	}
	return nil, fmt.Errorf("Error: Unknown shading type %d", shadingType)
}
