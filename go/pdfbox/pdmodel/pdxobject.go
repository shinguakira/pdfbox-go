package pdmodel

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
)

// CreateXObject builds the XObject a stream holds: an image, a form, a
// transparency group, or a PostScript one.
//
// Port of the static PDXObject.createXObject(COSBase, PDResources). Java hangs
// it off PDXObject, in pdmodel/graphics; the port cannot, because that package
// is imported by the image and form packages this dispatches to. It lives here
// instead, where both are in reach, and pdmodel/graphics/pdxobject.go says so.
//
// Java's return type is PDXObject, the common base; the port has no interface
// over the four, so it returns what every caller of it needs, the objectable.
func CreateXObject(base cos.Base, resources *PDResources) (common.COSObjectable, error) {
	if base == nil {
		// TODO throw an exception?
		return nil, nil
	}
	stream, isStream := base.(*cos.Stream)
	if !isStream {
		return nil, fmt.Errorf("Unexpected object type: %T", base)
	}
	switch stream.GetNameAsString(cos.Subtype, "") {
	case cos.Image.Name():
		// A nil *PDResources would still be a non-nil interface, so the port
		// passes the untyped nil on rather than the pointer.
		var resourcesLike color.ResourcesLike
		if resources != nil {
			resourcesLike = resources
		}
		return image.NewPDImageXObject(common.NewPDStream(stream), resourcesLike), nil
	case cos.Form.Name():
		var cache form.CacheLike
		if resources != nil && resources.Cache() != nil {
			cache = resources.Cache()
		}
		group := stream.GetCOSDictionary(cos.Group)
		if group != nil && cos.Transparency == group.GetCOSName(cos.S) {
			return form.NewPDTransparencyGroupOfStreamCached(stream, cache), nil
		}
		return form.NewPDFormXObjectOfStreamCached(stream, cache), nil
	case cos.PS.Name():
		return graphics.NewPDPostScriptXObject(stream), nil
	default:
		return nil, fmt.Errorf("Invalid XObject Subtype: %s",
			stream.GetNameAsString(cos.Subtype, ""))
	}
}
