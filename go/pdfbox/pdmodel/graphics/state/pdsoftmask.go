package state

// A soft mask, which is the /SMask of an extended graphics state.
//
// Port of PDSoftMask.

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// TransparencyGroupLike is the transparency group a soft mask paints through.
//
// Java's PDSoftMask.getGroup answers a
// graphics.form.PDTransparencyGroup, and builds it by asking
// PDXObject.createXObject and narrowing the result. Neither of those can be
// named here: graphics/form imports this package for the extended graphics
// state, and the factory lives in pdmodel, which imports both. So the mask
// names what it hands back and NewTransparencyGroup below is set from an
// importing package.
type TransparencyGroupLike interface {
	common.COSObjectable
}

// NewTransparencyGroup builds the transparency group of a soft mask from the
// /G entry and the cache the mask was read with. It is
// `PDXObject.createXObject(cosGroup, new PDResources(new COSDictionary(),
// resourceCache))` narrowed to a PDTransparencyGroup, and answers nil where
// that narrowing fails, which is what the instanceof of Java does.
//
// pdmodel sets it. Until it does, a soft mask has no group; see
// migration/STATUS.md.
var NewTransparencyGroup func(cosGroup cos.Base, resourceCache any) TransparencyGroupLike

// PDSoftMask is a soft mask: a transparency group whose luminosity or alpha
// becomes the mask through which paint reaches the page.
//
// Port of PDSoftMask, which Java declares final.
type PDSoftMask struct {
	dictionary    *cos.Dictionary
	resourceCache any

	subType          *cos.Name
	group            TransparencyGroupLike
	backdropColor    *cos.Array
	transferFunction function.PDFunction
	ctm              *util.Matrix
}

var _ common.COSObjectable = (*PDSoftMask)(nil)

// NewPDSoftMaskOf returns the soft mask the given object holds, and nil where
// it holds none.
//
// Port of the static create(COSBase), which answers null for /None, for any
// other name, and for anything that is not a dictionary.
func NewPDSoftMaskOf(dictionary cos.Base) *PDSoftMask {
	return NewPDSoftMaskOfCache(dictionary, nil)
}

// NewPDSoftMaskOfCache is NewPDSoftMaskOf with the resource cache the group is
// read through.
//
// Port of the static create(COSBase, ResourceCache).
func NewPDSoftMaskOfCache(dictionary cos.Base, resourceCache any) *PDSoftMask {
	switch value := dictionary.(type) {
	case *cos.Name:
		if value == cos.None {
			return nil
		}
		slog.Warn("state: invalid SMask", "smask", value)
		return nil
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so its instanceof lets one
		// through here.
		return NewPDSoftMask(&value.Dictionary, resourceCache)
	case *cos.Dictionary:
		return NewPDSoftMask(value, resourceCache)
	}
	slog.Warn("state: invalid SMask", "smask", dictionary)
	return nil
}

// NewPDSoftMask returns a soft mask over the given dictionary, read through the
// given cache.
//
// Port of the two constructors, which Java declares public.
func NewPDSoftMask(dictionary *cos.Dictionary, resourceCache any) *PDSoftMask {
	return &PDSoftMask{dictionary: dictionary, resourceCache: resourceCache}
}

// COSObject returns the soft mask dictionary.
func (m *PDSoftMask) COSObject() cos.Base { return m.dictionary }

// Dictionary returns the soft mask dictionary, typed.
func (m *PDSoftMask) Dictionary() *cos.Dictionary { return m.dictionary }

// SubType returns the /S entry, which says whether the mask is taken from the
// group's alpha or from its luminosity.
func (m *PDSoftMask) SubType() *cos.Name {
	if m.subType == nil {
		m.subType = m.dictionary.GetCOSName(cos.S)
	}
	return m.subType
}

// Group returns the transparency group of the mask, or nil where it has none.
func (m *PDSoftMask) Group() TransparencyGroupLike {
	if m.group == nil {
		if cosGroup := m.dictionary.GetDictionaryObject(cos.G); cosGroup != nil {
			if NewTransparencyGroup != nil {
				m.group = NewTransparencyGroup(cosGroup, m.resourceCache)
			}
		}
	}
	return m.group
}

// BackdropColor returns the /BC entry, the colour the group is composited
// against, or nil where there is none.
func (m *PDSoftMask) BackdropColor() *cos.Array {
	if m.backdropColor == nil {
		m.backdropColor = m.dictionary.GetCOSArray(cos.BC)
	}
	return m.backdropColor
}

// TransferFunction returns the /TR entry, which maps the mask value before it
// is used, or nil where there is none.
func (m *PDSoftMask) TransferFunction() (function.PDFunction, error) {
	if m.transferFunction == nil {
		if cosTF := m.dictionary.GetDictionaryObject(cos.TR); cosTF != nil {
			created, err := function.NewPDFunction(cosTF)
			if err != nil {
				return nil, err
			}
			m.transferFunction = created
		}
	}
	return m.transferFunction, nil
}

// SetInitialTransformationMatrix records the matrix in force when the mask was
// installed, which is what the mask is painted through.
//
// Java declares this package-private and sets it from PDExtendedGraphicsState;
// Go has no such level and the two are in one package here, so it is exported
// only as far as that.
func (m *PDSoftMask) SetInitialTransformationMatrix(ctm *util.Matrix) { m.ctm = ctm }

// InitialTransformationMatrix returns the matrix in force when the mask was
// installed.
func (m *PDSoftMask) InitialTransformationMatrix() *util.Matrix { return m.ctm }
