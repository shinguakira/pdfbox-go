package rendering

// Whether an optional content group is visible.
//
// Port of PageDrawer.isHiddenOCG, isHiddenOCMD and the four visibility
// expression methods. They are pure decisions about the document, so they are
// ported whole; nothing here draws.

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"
)

// isHiddenOCG reports whether the given property list names an optional content
// group that is off at this destination.
func (d *PageDrawer) isHiddenOCG(propertyList markedcontent.PropertyList) bool {
	switch p := propertyList.(type) {
	case *optionalcontent.PDOptionalContentGroup:
		printState := p.RenderStateFor(d.destination)
		if printState == optionalcontent.RenderStateUnset {
			if !d.renderer.IsGroupEnabled(p) {
				return true
			}
		} else if printState == optionalcontent.RenderStateOff {
			return true
		}
	case *optionalcontent.PDOptionalContentMembershipDictionary:
		return d.isHiddenOCMD(p)
	}
	return false
}

// isHiddenOCMD reports whether a membership dictionary is hidden, by its
// visibility expression where it has one and by its policy otherwise.
func (d *PageDrawer) isHiddenOCMD(ocmd *optionalcontent.PDOptionalContentMembershipDictionary) bool {
	veArray := ocmd.PropertyDictionary().GetCOSArray(cos.VE)
	if veArray != nil && veArray.Size() > 0 {
		return d.isHiddenVisibilityExpression(veArray)
	}
	oCGs := ocmd.OCGs()
	if len(oCGs) == 0 {
		return false
	}
	visibles := make([]bool, 0, len(oCGs))
	for _, prop := range oCGs {
		visibles = append(visibles, !d.isHiddenOCG(prop))
	}
	visibilityPolicy := ocmd.VisibilityPolicy()

	// visible if any of the entries in OCGs are OFF
	if visibilityPolicy == cos.AnyOff {
		return allTrue(visibles)
	}

	// visible only if all of the entries in OCGs are ON
	if visibilityPolicy == cos.AllOn {
		return anyFalse(visibles)
	}

	// visible only if all of the entries in OCGs are OFF
	if visibilityPolicy == cos.AllOff {
		return anyTrue(visibles)
	}

	// visible if any of the entries in OCGs are ON
	// AnyOn is default
	return !anyTrue(visibles)
}

// allTrue is Java's `visibles.stream().allMatch(v -> v)`.
func allTrue(visibles []bool) bool {
	for _, v := range visibles {
		if !v {
			return false
		}
	}
	return true
}

// anyTrue is Java's `visibles.stream().anyMatch(v -> v)`, and its negation is
// noneMatch.
func anyTrue(visibles []bool) bool {
	for _, v := range visibles {
		if v {
			return true
		}
	}
	return false
}

// anyFalse is Java's `visibles.stream().anyMatch(v -> !v)`.
func anyFalse(visibles []bool) bool {
	for _, v := range visibles {
		if !v {
			return true
		}
	}
	return false
}

// isHiddenVisibilityExpression evaluates a /VE array, which is a prefix
// expression over optional content groups.
func (d *PageDrawer) isHiddenVisibilityExpression(veArray *cos.Array) bool {
	if veArray.Size() == 0 {
		return false
	}
	op := veArray.GetName(0, "")
	switch op {
	case "And":
		return d.isHiddenAndVisibilityExpression(veArray)
	case "Or":
		return d.isHiddenOrVisibilityExpression(veArray)
	case "Not":
		return d.isHiddenNotVisibilityExpression(veArray)
	default:
		return false
	}
}

// isHiddenAndVisibilityExpression: hidden if at least one isn't visible.
func (d *PageDrawer) isHiddenAndVisibilityExpression(veArray *cos.Array) bool {
	for i := 1; i < veArray.Size(); i++ {
		switch base := veArray.GetObject(i).(type) {
		case *cos.Array:
			// Another VE
			if d.isHiddenVisibilityExpression(base) {
				return true
			}
		case *cos.Dictionary:
			// Another OCG
			if d.isHiddenOCG(markedcontent.CreatePropertyList(base)) {
				return true
			}
		}
	}
	return false
}

// isHiddenOrVisibilityExpression: hidden only if all are hidden.
func (d *PageDrawer) isHiddenOrVisibilityExpression(veArray *cos.Array) bool {
	for i := 1; i < veArray.Size(); i++ {
		switch base := veArray.GetObject(i).(type) {
		case *cos.Array:
			// Another VE
			if !d.isHiddenVisibilityExpression(base) {
				return false
			}
		case *cos.Dictionary:
			// Another OCG
			if !d.isHiddenOCG(markedcontent.CreatePropertyList(base)) {
				return false
			}
		}
	}
	return true
}

// isHiddenNotVisibilityExpression: the negation of its single operand.
func (d *PageDrawer) isHiddenNotVisibilityExpression(veArray *cos.Array) bool {
	if veArray.Size() != 2 {
		return false
	}
	switch base := veArray.GetObject(1).(type) {
	case *cos.Array:
		// Another VE
		return !d.isHiddenVisibilityExpression(base)
	case *cos.Dictionary:
		// Another OCG
		return !d.isHiddenOCG(markedcontent.CreatePropertyList(base))
	}
	return false
}
