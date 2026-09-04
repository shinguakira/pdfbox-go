package color

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDDeviceNAttributes are the attributes of a DeviceN colour space: which of
// its colorants are process colours and which are spot colours.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDDeviceNAttributes.
type PDDeviceNAttributes struct {
	dictionary *cos.Dictionary
}

// NewPDDeviceNAttributes returns empty attributes.
func NewPDDeviceNAttributes() *PDDeviceNAttributes {
	return &PDDeviceNAttributes{dictionary: cos.NewDictionary()}
}

// NewPDDeviceNAttributesOfDictionary wraps the given dictionary.
func NewPDDeviceNAttributesOfDictionary(attributes *cos.Dictionary) *PDDeviceNAttributes {
	return &PDDeviceNAttributes{dictionary: attributes}
}

// COSDictionary returns the dictionary below these attributes.
func (a *PDDeviceNAttributes) COSDictionary() *cos.Dictionary { return a.dictionary }

// Colorants returns the spot colour spaces, by colorant name.
//
// Java returns a COSDictionaryMap, a Map view that writes through to the
// dictionary; nothing in PDFBox writes through it, so the port returns a plain
// map and SetColorants writes the dictionary directly.
func (a *PDDeviceNAttributes) Colorants(resources ResourcesLike) (map[string]*PDSeparation, error) {
	actuals := map[string]*PDSeparation{}
	colorants := a.dictionary.GetCOSDictionary(cos.Colorants)
	if colorants == nil {
		colorants = cos.NewDictionary()
		a.dictionary.SetItem(cos.Colorants, colorants)
		return actuals, nil
	}
	for _, name := range colorants.KeySet() {
		value := colorants.GetDictionaryObject(name)
		space, err := CreateOfResources(value, resources)
		if err != nil {
			return nil, err
		}
		// Java casts to PDSeparation without a check, so a colorant that is not
		// one throws ClassCastException; the port panics for the same.
		actuals[name.Name()] = space.(*PDSeparation)
	}
	return actuals, nil
}

// Process returns the process colour attributes, or nil where there are none.
func (a *PDDeviceNAttributes) Process() *PDDeviceNProcess {
	process := a.dictionary.GetCOSDictionary(cos.Process)
	if process == nil {
		return nil
	}
	return NewPDDeviceNProcessOfDictionary(process)
}

// IsNChannel reports whether the subtype is NChannel.
func (a *PDDeviceNAttributes) IsNChannel() bool {
	return a.dictionary.GetNameAsString(cos.Subtype, "") == "NChannel"
}

// SetColorants sets the spot colour spaces.
func (a *PDDeviceNAttributes) SetColorants(colorants map[string]PDColorSpace) {
	var colorantDict cos.Base
	if colorants != nil {
		d := cos.NewDictionary()
		for name, space := range colorants {
			d.SetItem(cos.GetPDFName(name), space.COSObject())
		}
		colorantDict = d
	}
	a.dictionary.SetItem(cos.Colorants, colorantDict)
}

// String is Java's toString.
func (a *PDDeviceNAttributes) String() string {
	var sb strings.Builder
	sb.WriteString(a.dictionary.GetNameAsString(cos.Subtype, ""))
	sb.WriteByte('{')
	if process := a.Process(); process != nil {
		fmt.Fprintf(&sb, "%v ", process)
	}
	colorants, err := a.Colorants(nil)
	if err != nil {
		slog.Debug("color: couldn't get the colorants information - returning 'ERROR' instead",
			"err", err)
		sb.WriteString("ERROR")
	} else {
		sb.WriteString("Colorants{")
		for name, value := range colorants {
			fmt.Fprintf(&sb, "%q: %v ", name, value)
		}
		sb.WriteByte('}')
	}
	sb.WriteByte('}')
	return sb.String()
}

// PDDeviceNProcess is the process colour part of a DeviceN colour space.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDDeviceNProcess.
type PDDeviceNProcess struct {
	dictionary *cos.Dictionary
}

// NewPDDeviceNProcess returns an empty process.
func NewPDDeviceNProcess() *PDDeviceNProcess {
	return &PDDeviceNProcess{dictionary: cos.NewDictionary()}
}

// NewPDDeviceNProcessOfDictionary wraps the given dictionary.
func NewPDDeviceNProcessOfDictionary(attributes *cos.Dictionary) *PDDeviceNProcess {
	return &PDDeviceNProcess{dictionary: attributes}
}

// COSDictionary returns the dictionary below this process.
func (p *PDDeviceNProcess) COSDictionary() *cos.Dictionary { return p.dictionary }

// ColorSpace returns the process colour space, or nil where there is none.
func (p *PDDeviceNProcess) ColorSpace() (PDColorSpace, error) {
	cosColorSpace := p.dictionary.GetDictionaryObject(cos.ColorSpace)
	if cosColorSpace == nil {
		return nil, nil // TODO: return a default?
	}
	return Create(cosColorSpace)
}

// Components returns the names of the process components.
func (p *PDDeviceNProcess) Components() []string {
	cosComponents := p.dictionary.GetCOSArray(cos.Components)
	if cosComponents == nil {
		return []string{}
	}
	components := make([]string, 0, cosComponents.Size())
	for i := 0; i < cosComponents.Size(); i++ {
		// Java casts to COSName without a check, so a component that is not a
		// name throws ClassCastException; the port panics for the same.
		components = append(components, cosComponents.Get(i).(*cos.Name).Name())
	}
	return components
}

// String is Java's toString.
func (p *PDDeviceNProcess) String() string {
	var sb strings.Builder
	sb.WriteString("Process{")
	space, err := p.ColorSpace()
	if err != nil {
		slog.Debug("color: couldn't get the colorants information - returning 'ERROR' instead",
			"err", err)
		sb.WriteString("ERROR")
	} else {
		fmt.Fprintf(&sb, "%v", space)
		for _, component := range p.Components() {
			fmt.Fprintf(&sb, " %q", component)
		}
	}
	sb.WriteByte('}')
	return sb.String()
}
