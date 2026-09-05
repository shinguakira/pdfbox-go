package handlers

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// init records the handler of each annotation subtype.
//
// Java has no registry: each annotation's constructAppearances names its own
// handler class. The two packages would then reference each other, which Go
// forbids, so the annotation asks this map instead and this package fills it.
// The subtypes with no entry are the ones whose Java class has no handler
// either, and for those constructAppearances does nothing.
func init() {
	register := func(subtype string,
		factory func(annot annotation.PDAnnotation,
			document common.COSDocumentLike) annotation.PDAppearanceHandler) {
		annotation.DefaultAppearanceHandlers[subtype] = factory
	}

	register(annotation.SubTypeCaret, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDCaretAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeCircle, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDCircleAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeFileAttachment, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDFileAttachmentAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeHighlight, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDHighlightAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeInk, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDInkAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeLine, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDLineAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeLink, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDLinkAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypePolygon, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDPolygonAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypePolyLine, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDPolylineAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeSound, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDSoundAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeSquare, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDSquareAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeSquiggly, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDSquigglyAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeStrikeOut, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDStrikeoutAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeText, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDTextAppearanceHandlerInDocument(annot, document)
	})
	register(annotation.SubTypeUnderline, func(annot annotation.PDAnnotation,
		document common.COSDocumentLike) annotation.PDAppearanceHandler {
		return NewPDUnderlineAppearanceHandlerInDocument(annot, document)
	})
}
