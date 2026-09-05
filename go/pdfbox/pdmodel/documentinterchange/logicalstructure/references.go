package logicalstructure

import (
	"log/slog"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// TypeObjectReference is the /Type of an object reference.
//
// Port of PDObjectReference.TYPE.
const TypeObjectReference = "OBJR"

// asDictionary reads a base as a dictionary, taking the dictionary of a stream.
//
// COSStream extends COSDictionary in Java, so an instanceof COSDictionary is
// true for a stream; the port's *cos.Stream embeds a Dictionary instead, so
// every such test needs this.
func asDictionary(base cos.Base) (*cos.Dictionary, bool) {
	switch value := base.(type) {
	case *cos.Stream:
		return &value.Dictionary, true
	case *cos.Dictionary:
		return value, true
	}
	return nil, false
}

// PDObjectReference points at a PDF object -- an annotation or an XObject --
// from the structure tree.
//
// Port of PDObjectReference.
type PDObjectReference struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDObjectReference)(nil)

// NewPDObjectReference builds an empty object reference.
func NewPDObjectReference() *PDObjectReference {
	dictionary := cos.NewDictionary()
	dictionary.SetName(cos.Type, TypeObjectReference)
	return &PDObjectReference{dictionary: dictionary}
}

// NewPDObjectReferenceOf builds one over the given dictionary.
func NewPDObjectReferenceOf(theDictionary *cos.Dictionary) *PDObjectReference {
	return &PDObjectReference{dictionary: theDictionary}
}

// COSObject returns the dictionary.
func (r *PDObjectReference) COSObject() cos.Base { return r.dictionary }

// Dictionary returns the dictionary, typed.
func (r *PDObjectReference) Dictionary() *cos.Dictionary { return r.dictionary }

// ReferencedObject returns the object this reference points at: an XObject, an
// annotation, or nil where it is neither or cannot be built.
func (r *PDObjectReference) ReferencedObject() common.COSObjectable {
	// Java reads the entry with getCOSDictionary, which a stream also answers,
	// since COSStream extends COSDictionary there.
	objBase := r.dictionary.GetDictionaryObject(cos.Obj)
	objDictionary, isDictionary := asDictionary(objBase)
	if !isDictionary {
		return nil
	}
	if _, isStream := objBase.(*cos.Stream); isStream {
		xobject, err := CreateXObject(objBase)
		if err != nil {
			// this can only happen if the target is an XObject.
			slog.Debug("logicalstructure: could not get the referenced object - returning null instead",
				slog.Any("error", err))
			return nil
		}
		if xobject != nil {
			return xobject
		}
	}
	annot, err := annotation.CreateAnnotation(objBase)
	if err != nil {
		slog.Debug("logicalstructure: could not get the referenced object - returning null instead",
			slog.Any("error", err))
		return nil
	}
	// COSName.TYPE is optional, so if annotation is of type unknown and
	// COSName.TYPE is not COSName.ANNOT it still may be an annotation.
	// TODO shall we return the annotation object instead of null?
	// what else can be the target of the object reference?
	_, isUnknown := annot.(*annotation.PDAnnotationUnknown)
	if !isUnknown || cos.Annot == objDictionary.GetCOSName(cos.Type) {
		return annot
	}
	return nil
}

// SetReferencedObjectAnnotation points this reference at an annotation.
func (r *PDObjectReference) SetReferencedObjectAnnotation(annot annotation.PDAnnotation) {
	if annot == nil {
		r.dictionary.SetItem(cos.Obj, nil)
		return
	}
	r.dictionary.SetItem(cos.Obj, annot.COSObject())
}

// SetReferencedObjectXObject points this reference at an XObject.
func (r *PDObjectReference) SetReferencedObjectXObject(xobject common.COSObjectable) {
	if xobject == nil {
		r.dictionary.SetItem(cos.Obj, nil)
		return
	}
	r.dictionary.SetItem(cos.Obj, xobject.COSObject())
}

// Page returns the page the referenced object is on, or nil.
func (r *PDObjectReference) Page() PageLike {
	if pageDict := r.dictionary.GetCOSDictionary(cos.Pg); pageDict != nil {
		return NewPageFromDictionary(pageDict)
	}
	return nil
}

// SetPage sets the page the referenced object is on.
func (r *PDObjectReference) SetPage(page PageLike) {
	if page == nil {
		r.dictionary.SetItem(cos.Pg, nil)
		return
	}
	r.dictionary.SetItem(cos.Pg, page.COSObject())
}

// TypeMarkedContentReference is the /Type of a marked content reference.
//
// Port of PDMarkedContentReference.TYPE.
const TypeMarkedContentReference = "MCR"

// PDMarkedContentReference points at a marked-content sequence from the
// structure tree.
//
// Port of PDMarkedContentReference.
type PDMarkedContentReference struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDMarkedContentReference)(nil)

// NewPDMarkedContentReference builds an empty marked content reference.
func NewPDMarkedContentReference() *PDMarkedContentReference {
	dictionary := cos.NewDictionary()
	dictionary.SetName(cos.Type, TypeMarkedContentReference)
	return &PDMarkedContentReference{dictionary: dictionary}
}

// NewPDMarkedContentReferenceOf builds one over the given dictionary.
func NewPDMarkedContentReferenceOf(dictionary *cos.Dictionary) *PDMarkedContentReference {
	return &PDMarkedContentReference{dictionary: dictionary}
}

// COSObject returns the dictionary.
func (r *PDMarkedContentReference) COSObject() cos.Base { return r.dictionary }

// Dictionary returns the dictionary, typed.
func (r *PDMarkedContentReference) Dictionary() *cos.Dictionary { return r.dictionary }

// Page returns the page the marked content is on, or nil.
func (r *PDMarkedContentReference) Page() PageLike {
	if pg := r.dictionary.GetCOSDictionary(cos.Pg); pg != nil {
		return NewPageFromDictionary(pg)
	}
	return nil
}

// SetPage sets the page the marked content is on.
func (r *PDMarkedContentReference) SetPage(page PageLike) {
	if page == nil {
		r.dictionary.SetItem(cos.Pg, nil)
		return
	}
	r.dictionary.SetItem(cos.Pg, page.COSObject())
}

// MCID returns the marked-content identifier.
func (r *PDMarkedContentReference) MCID() int {
	return r.dictionary.GetInt(cos.MCID)
}

// SetMCID sets the marked-content identifier.
//
// Java throws IllegalArgumentException for a negative one, which is unchecked,
// so the port panics.
func (r *PDMarkedContentReference) SetMCID(mcid int) {
	if mcid < 0 {
		panic("MCID is negative")
	}
	r.dictionary.SetInt(cos.MCID, mcid)
}

// String renders the reference the way Java's toString does.
func (r *PDMarkedContentReference) String() string {
	return "mcid=" + strconv.Itoa(r.MCID())
}
