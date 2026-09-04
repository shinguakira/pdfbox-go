// Package form holds the form XObject, which is a piece of content a page can
// draw more than once.
//
// Port of org.apache.pdfbox.pdmodel.graphics.form. PLAN.md puts this package in
// slice 9, with the rest of the drawing; slice 8 ports it because
// PDAppearanceStream extends PDFormXObject and every annotation appearance is
// one. PDTransparencyGroup, the third file, stays with slice 9: it exists for
// the renderer. See migration/STATUS.md.
package form

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// ResourcesLike is the resource dictionary of a form XObject, and CacheLike is
// the resource cache it reads through.
//
// Java names PDResources and ResourceCache, which live in pdmodel; pdmodel
// imports this package transitively through the annotations, so the dependency
// cannot run both ways. The port names what is used and takes the two
// constructors below, which pdmodel sets from its init.
type ResourcesLike interface {
	common.COSObjectable
}

// CacheLike is the resource cache a form XObject reads through.
type CacheLike any

// NewResourcesFromDictionary builds resources from their dictionary and a
// cache. pdmodel sets it.
var NewResourcesFromDictionary func(dict *cos.Dictionary, cache CacheLike) ResourcesLike

// NewEmptyResources builds empty resources. pdmodel sets it.
var NewEmptyResources func() ResourcesLike

// PDFormXObject is a form XObject: a self-contained piece of content.
//
// Port of PDFormXObject, which extends PDXObject and implements
// PDContentStream.
//
// Java's TODO above the class, that there are further form XObjects to
// implement and that its methods should then be final, is carried here too.
type PDFormXObject struct {
	graphics.PDXObject
	group *PDTransparencyGroupAttributes
	cache CacheLike
}

var _ common.COSObjectable = (*PDFormXObject)(nil)

// NewPDFormXObjectOfPDStream creates a form XObject over the given stream.
func NewPDFormXObjectOfPDStream(stream *common.PDStream) *PDFormXObject {
	return &PDFormXObject{PDXObject: graphics.NewPDXObjectOfPDStream(stream, cos.Form)}
}

// NewPDFormXObjectOfStream creates a form XObject over the given stream.
func NewPDFormXObjectOfStream(stream *cos.Stream) *PDFormXObject {
	return &PDFormXObject{PDXObject: graphics.NewPDXObjectOfStream(stream, cos.Form)}
}

// NewPDFormXObjectOfStreamCached creates one that reads its resources through
// the given cache.
func NewPDFormXObjectOfStreamCached(stream *cos.Stream, cache CacheLike) *PDFormXObject {
	return &PDFormXObject{
		PDXObject: graphics.NewPDXObjectOfStream(stream, cos.Form),
		cache:     cache,
	}
}

// NewPDFormXObject creates a new empty form XObject in the given document.
func NewPDFormXObject(document common.COSDocumentLike) *PDFormXObject {
	return NewPDFormXObjectOfStream(document.CreateStream())
}

// FormType returns the /FormType, which defaults to 1.
func (f *PDFormXObject) FormType() int {
	return f.Stream().GetIntDefault(cos.FormType, 1)
}

// SetFormType sets the /FormType.
func (f *PDFormXObject) SetFormType(formType int) {
	f.Stream().SetInt(cos.FormType, formType)
}

// Group returns the /Group transparency attributes, or nil.
func (f *PDFormXObject) Group() *PDTransparencyGroupAttributes {
	if f.group == nil {
		if dic := f.Stream().GetCOSDictionary(cos.Group); dic != nil {
			f.group = NewPDTransparencyGroupAttributesOf(dic)
		}
	}
	return f.group
}

// SetGroup sets the /Group transparency attributes.
func (f *PDFormXObject) SetGroup(group *PDTransparencyGroupAttributes) {
	f.group = group
	if group == nil {
		f.Stream().SetItem(cos.Group, nil)
		return
	}
	f.Stream().SetItem(cos.Group, group.COSObject())
}

// ContentStream returns the stream as a PDStream.
func (f *PDFormXObject) ContentStream() *common.PDStream {
	return common.NewPDStream(f.Stream())
}

// ContentsForRandomAccess returns the content of this form.
func (f *PDFormXObject) ContentsForRandomAccess() (pdfio.RandomAccessRead, error) {
	return f.Stream().CreateView()
}

// Resources returns the /Resources of this form, or nil where it has none.
func (f *PDFormXObject) Resources() ResourcesLike {
	if resources := f.Stream().GetCOSDictionary(cos.Resources); resources != nil {
		return NewResourcesFromDictionary(resources, f.cache)
	}
	if f.Stream().ContainsKey(cos.Resources) {
		// PDFBOX-4372 if the resource key exists but has nothing, return empty resources,
		// to avoid a self-reference (xobject form Fm0 contains "/Fm0 Do")
		// See also the mention of PDFBOX-1359 in PDFStreamEngine
		return NewEmptyResources()
	}
	return nil
}

// SetResources sets the /Resources of this form.
func (f *PDFormXObject) SetResources(resources ResourcesLike) {
	if resources == nil {
		f.Stream().SetItem(cos.Resources, nil)
		return
	}
	f.Stream().SetItem(cos.Resources, resources.COSObject())
}

// BBox returns the /BBox of this form, or nil.
func (f *PDFormXObject) BBox() *common.PDRectangle {
	if array := f.Stream().GetCOSArray(cos.BBox); array != nil {
		return common.NewPDRectangleOfCOSArray(array)
	}
	return nil
}

// SetBBox sets the /BBox of this form; nil removes it.
func (f *PDFormXObject) SetBBox(bbox *common.PDRectangle) {
	if bbox == nil {
		f.Stream().RemoveItem(cos.BBox)
	} else {
		f.Stream().SetItem(cos.BBox, bbox.COSArray())
	}
}

// Matrix returns the /Matrix of this form.
func (f *PDFormXObject) Matrix() *util.Matrix {
	return util.CreateMatrix(f.Stream().GetDictionaryObject(cos.Matrix))
}

// SetMatrix sets the /Matrix of this form.
//
// Java takes an AffineTransform and writes the six values getMatrix fills, in
// the order m00 m10 m01 m11 m02 m12, which is a PDF matrix's a b c d e f. The
// port takes a util.Matrix, which is what everything else here holds, and reads
// the same six.
func (f *PDFormXObject) SetMatrix(transform *util.Matrix) {
	matrix := cos.NewArray()
	for _, v := range []float32{
		transform.ScaleX(), transform.ShearY(),
		transform.ShearX(), transform.ScaleY(),
		transform.TranslateX(), transform.TranslateY(),
	} {
		matrix.Add(cos.NewFloat(v))
	}
	f.Stream().SetItem(cos.Matrix, matrix)
}

// StructParents returns the /StructParents entry.
func (f *PDFormXObject) StructParents() int {
	return f.Stream().GetInt(cos.StructParents)
}

// SetStructParents sets the /StructParents entry.
func (f *PDFormXObject) SetStructParents(structParent int) {
	f.Stream().SetInt(cos.StructParents, structParent)
}

// OptionalContent returns the /OC property list, or nil.
func (f *PDFormXObject) OptionalContent() markedcontent.PropertyList {
	if optionalContent := f.Stream().GetCOSDictionary(cos.OC); optionalContent != nil {
		return markedcontent.CreatePropertyList(optionalContent)
	}
	return nil
}

// SetOptionalContent sets the /OC property list.
func (f *PDFormXObject) SetOptionalContent(oc markedcontent.PropertyList) {
	if oc == nil {
		f.Stream().SetItem(cos.OC, nil)
		return
	}
	f.Stream().SetItem(cos.OC, oc.COSObject())
}

// PDTransparencyGroupAttributes is the /Group entry of a form XObject.
//
// Port of PDTransparencyGroupAttributes.
type PDTransparencyGroupAttributes struct {
	dictionary *cos.Dictionary
	colorSpace color.PDColorSpace
}

var _ common.COSObjectable = (*PDTransparencyGroupAttributes)(nil)

// NewPDTransparencyGroupAttributes creates a new transparency group.
func NewPDTransparencyGroupAttributes() *PDTransparencyGroupAttributes {
	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.S, cos.Transparency)
	return &PDTransparencyGroupAttributes{dictionary: dictionary}
}

// NewPDTransparencyGroupAttributesOf creates one over the given dictionary.
func NewPDTransparencyGroupAttributesOf(dic *cos.Dictionary) *PDTransparencyGroupAttributes {
	return &PDTransparencyGroupAttributes{dictionary: dic}
}

// COSObject returns the dictionary.
func (g *PDTransparencyGroupAttributes) COSObject() cos.Base { return g.dictionary }

// ColorSpace returns the /CS colour space of the group, or nil.
func (g *PDTransparencyGroupAttributes) ColorSpace(resources color.ResourcesLike) (color.PDColorSpace, error) {
	if g.colorSpace == nil && g.dictionary.ContainsKey(cos.CS) {
		space, err := color.CreateOfResources(g.dictionary.GetDictionaryObject(cos.CS), resources)
		if err != nil {
			return nil, err
		}
		g.colorSpace = space
	}
	return g.colorSpace, nil
}

// IsIsolated reports the /I entry.
func (g *PDTransparencyGroupAttributes) IsIsolated() bool {
	return g.dictionary.GetBoolean(cos.I, false)
}

// IsKnockout reports the /K entry.
func (g *PDTransparencyGroupAttributes) IsKnockout() bool {
	return g.dictionary.GetBoolean(cos.K, false)
}
