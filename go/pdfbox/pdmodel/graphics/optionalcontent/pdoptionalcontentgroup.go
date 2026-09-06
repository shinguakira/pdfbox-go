// Package optionalcontent holds the optional content groups a PDF can switch
// on and off.
//
// Port of org.apache.pdfbox.pdmodel.graphics.optionalcontent.
package optionalcontent

import (
	"fmt"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
)

func init() {
	// Java's PDPropertyList.create names these two types directly, which Go
	// cannot do without a cycle; see markedcontent/pdpropertylist.go.
	markedcontent.RegisterPropertyList(cos.OCG,
		func(dict *cos.Dictionary) markedcontent.PropertyList {
			return NewPDOptionalContentGroupOf(dict)
		})
	markedcontent.RegisterPropertyList(cos.OCMD,
		func(dict *cos.Dictionary) markedcontent.PropertyList {
			return NewPDOptionalContentMembershipDictionaryOf(dict)
		})
}

// PDOptionalContentGroup is an optional content group, which a viewer can show
// or hide.
//
// Port of PDOptionalContentGroup, which extends PDPropertyList.
type PDOptionalContentGroup struct {
	markedcontent.PDPropertyList
}

var _ markedcontent.PropertyList = (*PDOptionalContentGroup)(nil)

// NewPDOptionalContentGroup creates a group with the given name.
func NewPDOptionalContentGroup(name string) *PDOptionalContentGroup {
	g := &PDOptionalContentGroup{PDPropertyList: *markedcontent.NewPDPropertyList()}
	g.Dict.SetItem(cos.Type, cos.OCG)
	g.SetName(name)
	return g
}

// NewPDOptionalContentGroupOf creates a group over the given dictionary.
//
// Java throws IllegalArgumentException where the /Type is not /OCG, which is
// unchecked, so the port panics.
func NewPDOptionalContentGroupOf(dict *cos.Dictionary) *PDOptionalContentGroup {
	if dict.GetDictionaryObject(cos.Type) != cos.Base(cos.OCG) {
		panic(fmt.Sprintf("Provided dictionary is not of type '%v'", cos.OCG))
	}
	return &PDOptionalContentGroup{PDPropertyList: *markedcontent.NewPDPropertyListOf(dict)}
}

// RenderState is whether a group is on or off for a given destination.
//
// Port of the nested enum PDOptionalContentGroup.RenderState.
type RenderState int

const (
	// RenderStateUnset stands for Java's null, which every caller of
	// GetRenderState compares against.
	RenderStateUnset RenderState = iota
	// RenderStateOn is /ON.
	RenderStateOn
	// RenderStateOff is /OFF.
	RenderStateOff
)

// Name returns the COS name of the state.
func (s RenderState) Name() *cos.Name {
	switch s {
	case RenderStateOn:
		return cos.ON
	case RenderStateOff:
		return cos.OFF
	}
	return nil
}

// RenderStateValueOf returns the state the given name stands for.
//
// Port of the static RenderState.valueOf(COSName), which upper-cases the name
// and looks up the enum constant; an unknown name makes Enum.valueOf throw
// IllegalArgumentException, so the port panics.
func RenderStateValueOf(state *cos.Name) RenderState {
	if state == nil {
		return RenderStateUnset
	}
	switch strings.ToUpper(state.Name()) {
	case "ON":
		return RenderStateOn
	case "OFF":
		return RenderStateOff
	}
	panic(fmt.Sprintf("No enum constant RenderState.%s", strings.ToUpper(state.Name())))
}

// Name returns the /Name of the group.
func (g *PDOptionalContentGroup) Name() string {
	return g.Dict.GetString(cos.NameKey, "")
}

// SetName sets the /Name of the group.
func (g *PDOptionalContentGroup) SetName(name string) {
	g.Dict.SetString(cos.NameKey, name)
}

// RenderStateFor returns the state of this group for the given destination, or
// RenderStateUnset where the group does not say.
//
// Port of getRenderState(RenderDestination). The /Intent support Java marks
// with a TODO is not here either.
func (g *PDOptionalContentGroup) RenderStateFor(destination RenderDestination) RenderState {
	var state *cos.Name
	usage := g.Dict.GetCOSDictionary(cos.Usage)
	if usage != nil {
		if destination == Print {
			if print := usage.GetCOSDictionary(cos.Print); print != nil {
				state = print.GetCOSName(cos.PrintState)
			}
		} else if destination == View {
			if view := usage.GetCOSDictionary(cos.View); view != nil {
				state = view.GetCOSName(cos.ViewState)
			}
		}
		// Fallback to export
		if state == nil {
			if export := usage.GetCOSDictionary(cos.Export); export != nil {
				state = export.GetCOSName(cos.ExportState)
			}
		}
	}
	if state == nil {
		return RenderStateUnset
	}
	return RenderStateValueOf(state)
}

// String returns the Java toString form.
//
// Java's is `super.toString() + " (" + getName() + ")"`, and PDPropertyList
// does not override toString, so the first half is Object's
// `class@hashcode`. The port uses the type name, since Go has no such default
// and a hash of the pointer would say nothing.
func (g *PDOptionalContentGroup) String() string {
	return "PDOptionalContentGroup (" + g.Name() + ")"
}
