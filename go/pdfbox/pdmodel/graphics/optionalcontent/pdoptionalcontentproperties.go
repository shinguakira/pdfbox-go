package optionalcontent

import (
	"fmt"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// BaseState is the /BaseState entry of the /D dictionary.
//
// Port of the nested enum PDOptionalContentProperties.BaseState.
type BaseState int

const (
	// BaseStateON is the "ON" value.
	BaseStateON BaseState = iota
	// BaseStateOFF is the "OFF" value.
	BaseStateOFF
	// BaseStateUNCHANGED is the "Unchanged" value.
	BaseStateUNCHANGED
)

// Name returns the PDF name for the state.
func (s BaseState) Name() *cos.Name {
	switch s {
	case BaseStateOFF:
		return cos.OFF
	case BaseStateUNCHANGED:
		return cos.Unchanged
	}
	return cos.ON
}

// String returns the Java enum constant name, which is what its toString
// answers and what BaseStateValueOf takes back.
func (s BaseState) String() string {
	switch s {
	case BaseStateOFF:
		return "OFF"
	case BaseStateUNCHANGED:
		return "UNCHANGED"
	}
	return "ON"
}

// BaseStateValueOf returns the base state the given name stands for.
//
// Port of the static BaseState.valueOf(COSName), which upper-cases the name and
// looks up the enum constant; an unknown name makes Enum.valueOf throw
// IllegalArgumentException, so the port panics.
func BaseStateValueOf(state *cos.Name) BaseState {
	if state == nil {
		return BaseStateON
	}
	switch strings.ToUpper(state.Name()) {
	case "ON":
		return BaseStateON
	case "OFF":
		return BaseStateOFF
	case "UNCHANGED":
		return BaseStateUNCHANGED
	}
	panic(fmt.Sprintf("No enum constant BaseState.%s", strings.ToUpper(state.Name())))
}

// PDOptionalContentProperties is the optional content properties dictionary.
//
// Port of PDOptionalContentProperties. Since PDF 1.5.
type PDOptionalContentProperties struct {
	dict *cos.Dictionary
}

// NewPDOptionalContentProperties creates a new optional content properties
// dictionary.
func NewPDOptionalContentProperties() *PDOptionalContentProperties {
	p := &PDOptionalContentProperties{dict: cos.NewDictionary()}
	p.dict.SetItem(cos.OCGs, cos.NewArray())
	d := cos.NewDictionary()

	// Name optional but required for PDF/A-3
	d.SetString(cos.NameKey, "Top")

	p.dict.SetItem(cos.D, d)
	return p
}

// NewPDOptionalContentPropertiesOf creates a new instance based on a given
// dictionary.
func NewPDOptionalContentPropertiesOf(props *cos.Dictionary) *PDOptionalContentProperties {
	return &PDOptionalContentProperties{dict: props}
}

// COSObject returns the dictionary.
func (p *PDOptionalContentProperties) COSObject() cos.Base { return p.dict }

// Dictionary returns the dictionary, which is getCOSObject narrowed the way
// Java declares it.
func (p *PDOptionalContentProperties) Dictionary() *cos.Dictionary { return p.dict }

// ocgs returns the /OCGs array, never nil.
func (p *PDOptionalContentProperties) ocgs() *cos.Array {
	ocgs := p.dict.GetCOSArray(cos.OCGs)
	if ocgs == nil {
		ocgs = cos.NewArray()
		p.dict.SetItem(cos.OCGs, ocgs) // OCGs is required
	}
	return ocgs
}

// d returns the /D dictionary, never nil.
func (p *PDOptionalContentProperties) d() *cos.Dictionary {
	d := p.dict.GetCOSDictionary(cos.D)
	if d == nil {
		d = cos.NewDictionary()
		// Name optional but required for PDF/A-3
		d.SetString(cos.NameKey, "Top")
		// D is required
		p.dict.SetItem(cos.D, d)
	}
	return d
}

// Group returns the first optional content group of the given name, or nil
// where there is no such group.
func (p *PDOptionalContentProperties) Group(name string) *PDOptionalContentGroup {
	for _, o := range p.ocgs().ToList() {
		ocg := toDictionary(o)
		if ocg == nil {
			continue
		}
		// Java reads the name with getString, which answers null for a group
		// dictionary without a /Name, and then calls equals on it, so such a
		// group throws NullPointerException here. The port has no null string
		// and answers "", so the group is simply not matched. GroupNames says
		// the same thing.
		groupName := ocg.GetString(cos.NameKey, "")
		if groupName == name {
			return NewPDOptionalContentGroupOf(ocg)
		}
	}
	return nil
}

// AddGroup adds an optional content group.
func (p *PDOptionalContentProperties) AddGroup(ocg *PDOptionalContentGroup) {
	ocgs := p.ocgs()
	ocgs.Add(ocg.COSObject())

	// By default, add new group to the "Order" entry so it appears in the user interface
	order := p.d().GetCOSArray(cos.Order)
	if order == nil {
		order = cos.NewArray()
		p.d().SetItem(cos.Order, order)
	}
	order.Add(ocg.COSObject())
}

// OptionalContentGroups returns every optional content group.
func (p *PDOptionalContentProperties) OptionalContentGroups() []*PDOptionalContentGroup {
	coll := []*PDOptionalContentGroup{}
	for _, base := range p.ocgs().ToList() {
		if dictionary := toDictionary(base); dictionary != nil {
			coll = append(coll, NewPDOptionalContentGroupOf(dictionary))
		}
	}
	return coll
}

// BaseState returns the base state for optional content groups. /BaseState
// defaults to /ON, so this never answers a state the file did not choose.
func (p *PDOptionalContentProperties) BaseState() BaseState {
	return BaseStateValueOf(p.d().GetCOSNameDefault(cos.BaseState, cos.ON))
}

// SetBaseState sets the base state for optional content groups.
func (p *PDOptionalContentProperties) SetBaseState(state BaseState) {
	p.d().SetItem(cos.BaseState, state.Name())
}

// GroupNames lists all optional content group names.
//
// Java answers null for a group dictionary whose /Name is missing or of the
// wrong type, since getString returns null, and "" for an entry that is not a
// dictionary at all. The port has no null string, so both are "".
func (p *PDOptionalContentProperties) GroupNames() []string {
	ocgs := p.dict.GetCOSArray(cos.OCGs)
	if ocgs == nil {
		return []string{}
	}
	size := ocgs.Size()
	groups := make([]string, size)
	for i := 0; i < size; i++ {
		obj := ocgs.Get(i)
		ocg := toDictionary(obj)
		if ocg == nil {
			groups[i] = ""
		} else {
			groups[i] = ocg.GetString(cos.NameKey, "")
		}
	}
	return groups
}

// HasGroup says whether a particular optional content group is found in the
// PDF file.
func (p *PDOptionalContentProperties) HasGroup(groupName string) bool {
	for _, layer := range p.GroupNames() {
		if layer == groupName {
			return true
		}
	}
	return false
}

// IsGroupEnabledNamed says whether at least one optional content group with
// this name is enabled. There may be disabled optional content groups with this
// name even if this function returns true.
//
// Port of isGroupEnabled(String); Go has no overloading, so the two
// isGroupEnabled take different names.
func (p *PDOptionalContentProperties) IsGroupEnabledNamed(groupName string) bool {
	result := false
	for _, o := range p.ocgs().ToList() {
		ocg := toDictionary(o)
		if ocg == nil {
			continue
		}
		name := ocg.GetString(cos.NameKey, "")
		if groupName == name && p.IsGroupEnabled(NewPDOptionalContentGroupOf(ocg)) {
			result = true
		}
	}
	return result
}

// IsGroupEnabled says whether an optional content group is enabled.
//
// Java's two TODOs are not done here either: optional content configuration
// dictionaries, /OCProperties/Configs, are ignored, and BaseState.UNCHANGED is
// treated as enabled.
func (p *PDOptionalContentProperties) IsGroupEnabled(group *PDOptionalContentGroup) bool {
	// TODO handle Optional Content Configuration Dictionaries,
	// i.e. OCProperties/Configs

	baseState := p.BaseState()
	enabled := baseState != BaseStateOFF
	// TODO What to do with BaseState.Unchanged?

	if group == nil {
		return enabled
	}

	d := p.d()
	// Java compares the dereferenced entry against the group's dictionary with
	// ==, so it is dictionary identity, not equality; the port compares the
	// pointers, which is the same thing.
	if on := d.GetCOSArray(cos.ON); on != nil {
		for _, o := range on.ToList() {
			if toDictionary(o) == group.PropertyDictionary() {
				return true
			}
		}
	}

	if off := d.GetCOSArray(cos.OFF); off != nil {
		for _, o := range off.ToList() {
			if toDictionary(o) == group.PropertyDictionary() {
				return false
			}
		}
	}

	return enabled
}

// toDictionary is the private toDictionary, which dereferences an indirect
// object and answers nil for anything that is not a dictionary.
func toDictionary(o cos.Base) *cos.Dictionary {
	base := o
	if object, isObject := o.(*cos.Object); isObject {
		base = object.Object()
	}
	if dictionary, isDictionary := base.(*cos.Dictionary); isDictionary {
		return dictionary
	}
	return nil
}

// SetGroupEnabledNamed enables or disables all optional content groups with the
// given name, and returns whether at least one group with this name already had
// an on or off setting.
//
// Port of setGroupEnabled(String, boolean); see IsGroupEnabledNamed for the
// name.
func (p *PDOptionalContentProperties) SetGroupEnabledNamed(groupName string, enable bool) bool {
	result := false
	for _, o := range p.ocgs().ToList() {
		ocg := toDictionary(o)
		if ocg == nil {
			continue
		}
		name := ocg.GetString(cos.NameKey, "")
		if groupName == name && p.SetGroupEnabled(NewPDOptionalContentGroupOf(ocg), enable) {
			result = true
		}
	}
	return result
}

// SetGroupEnabled enables or disables an optional content group, and returns
// whether the group already had an on or off setting.
func (p *PDOptionalContentProperties) SetGroupEnabled(group *PDOptionalContentGroup, enable bool) bool {
	d := p.d()
	on := d.GetCOSArray(cos.ON)
	if on == nil {
		on = cos.NewArray()
		d.SetItem(cos.ON, on)
	}

	off := d.GetCOSArray(cos.OFF)
	if off == nil {
		off = cos.NewArray()
		d.SetItem(cos.OFF, off)
	}

	found := false
	if enable {
		// Java walks the array it is removing from, and breaks out of the loop
		// as soon as it removes; the port walks the snapshot ToList answers,
		// which is the same walk because of that break.
		for _, o := range off.ToList() {
			if toDictionary(o) == group.PropertyDictionary() {
				// enable group
				off.Remove(o)
				on.Add(o)
				found = true
				break
			}
		}
	} else {
		for _, o := range on.ToList() {
			if toDictionary(o) == group.PropertyDictionary() {
				// disable group
				on.Remove(o)
				off.Add(o)
				found = true
				break
			}
		}
	}
	if !found {
		if enable {
			on.Add(group.COSObject())
		} else {
			off.Add(group.COSObject())
		}
	}
	return found
}
