package pdmodel

// The constructors graphics/state needs from this package.
//
// A soft mask's /G entry is a transparency group, which
// PDXObject.createXObject builds; that factory is here, because it names both
// the image and the form XObject, and graphics/state sits below it. So the mask
// names what it hands back and this init fills the constructor in, which is the
// device migration/conventions/java-to-go.md describes.

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
)

func init() {
	state.NewTransparencyGroup = func(cosGroup cos.Base,
		resourceCache any) state.TransparencyGroupLike {
		cache, _ := resourceCache.(ResourceCache)
		resources := NewPDResourcesOfCache(cos.NewDictionary(), cache)
		x, err := CreateXObject(cosGroup, resources)
		if err != nil {
			// Java lets the IOException out of getGroup; the port's caller
			// answers a group or nothing, so the failure is logged here and
			// the mask has no group, which is what an XObject of the wrong
			// kind gives on both sides.
			slog.Debug("pdmodel: reading the transparency group of a soft mask",
				slog.Any("error", err))
			return nil
		}
		group, isGroup := x.(*form.PDTransparencyGroup)
		if !isGroup {
			return nil
		}
		return group
	}
}
