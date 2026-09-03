// Package afm reads Adobe Font Metrics files, which carry the widths of the
// Standard 14 fonts.
//
// Port of org.apache.fontbox.afm.
//
// Java returns Collections.unmodifiableList from every list accessor. Go has no
// unmodifiable slice, so each accessor returns a copy: appending to what comes
// out cannot reach the object, which is the part of the Java contract that
// matters. A caller who only reads pays a copy for it.
package afm

import (
	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
)

// Ligature is a ligature of a character metric.
//
// Port of org.apache.fontbox.afm.Ligature.
type Ligature struct {
	successor string
	liga      string
}

// NewLigature returns the ligature reached from the given successor.
func NewLigature(successor, ligature string) *Ligature {
	return &Ligature{successor: successor, liga: ligature}
}

// Ligature returns the name of the ligature.
func (l *Ligature) Ligature() string { return l.liga }

// Successor returns the character that forms the ligature with this one.
func (l *Ligature) Successor() string { return l.successor }

// KernPair is a kerning pair: how much to move the second character of a pair
// when it follows the first.
//
// Port of org.apache.fontbox.afm.KernPair.
type KernPair struct {
	firstKernCharacter  string
	secondKernCharacter string
	x                   float32
	y                   float32
}

// NewKernPair returns the kerning of one pair of characters.
func NewKernPair(firstKernCharacter, secondKernCharacter string, x, y float32) *KernPair {
	return &KernPair{
		firstKernCharacter:  firstKernCharacter,
		secondKernCharacter: secondKernCharacter,
		x:                   x,
		y:                   y,
	}
}

// FirstKernCharacter returns the first character of the pair.
func (k *KernPair) FirstKernCharacter() string { return k.firstKernCharacter }

// SecondKernCharacter returns the second character of the pair.
func (k *KernPair) SecondKernCharacter() string { return k.secondKernCharacter }

// X returns the horizontal kerning.
func (k *KernPair) X() float32 { return k.x }

// Y returns the vertical kerning.
func (k *KernPair) Y() float32 { return k.y }

// TrackKern is a track kerning entry: how much to tighten or loosen a run of
// text, as a function of the point size it is set at.
//
// Port of org.apache.fontbox.afm.TrackKern.
type TrackKern struct {
	degree       int
	minPointSize float32
	minKern      float32
	maxPointSize float32
	maxKern      float32
}

// NewTrackKern returns one degree of track kerning.
func NewTrackKern(degree int, minPointSize, minKern, maxPointSize, maxKern float32) *TrackKern {
	return &TrackKern{
		degree:       degree,
		minPointSize: minPointSize,
		minKern:      minKern,
		maxPointSize: maxPointSize,
		maxKern:      maxKern,
	}
}

// Degree returns the degree of tightness or looseness.
func (t *TrackKern) Degree() int { return t.degree }

// MaxKern returns the kerning at the maximum point size.
func (t *TrackKern) MaxKern() float32 { return t.maxKern }

// MaxPointSize returns the point size the maximum kerning applies at.
func (t *TrackKern) MaxPointSize() float32 { return t.maxPointSize }

// MinKern returns the kerning at the minimum point size.
func (t *TrackKern) MinKern() float32 { return t.minKern }

// MinPointSize returns the point size the minimum kerning applies at.
func (t *TrackKern) MinPointSize() float32 { return t.minPointSize }

// CompositePart is one piece of a composite character, with the displacement it
// is drawn at.
//
// Port of org.apache.fontbox.afm.CompositePart.
type CompositePart struct {
	name          string
	xDisplacement int
	yDisplacement int
}

// NewCompositePart returns one part of a composite character.
func NewCompositePart(name string, xDisplacement, yDisplacement int) *CompositePart {
	return &CompositePart{name: name, xDisplacement: xDisplacement, yDisplacement: yDisplacement}
}

// Name returns the name of the part.
func (c *CompositePart) Name() string { return c.name }

// XDisplacement returns the horizontal displacement of the part.
func (c *CompositePart) XDisplacement() int { return c.xDisplacement }

// YDisplacement returns the vertical displacement of the part.
func (c *CompositePart) YDisplacement() int { return c.yDisplacement }

// Composite is a composite character, built from parts.
//
// Port of org.apache.fontbox.afm.Composite.
type Composite struct {
	name  string
	parts []*CompositePart
}

// NewComposite returns a composite character with no parts.
func NewComposite(name string) *Composite {
	return &Composite{name: name}
}

// Name returns the name of the composite character.
func (c *Composite) Name() string { return c.name }

// AddPart adds a part to this composite character.
func (c *Composite) AddPart(part *CompositePart) {
	c.parts = append(c.parts, part)
}

// Parts returns the parts of this composite character, as a copy.
func (c *Composite) Parts() []*CompositePart {
	return append([]*CompositePart{}, c.parts...)
}

// CharMetric is the metrics of one character.
//
// Port of org.apache.fontbox.afm.CharMetric.
type CharMetric struct {
	characterCode int
	wx            float32
	w0x           float32
	w1x           float32
	wy            float32
	w0y           float32
	w1y           float32
	w             []float32
	w0            []float32
	w1            []float32
	vv            []float32
	name          string
	boundingBox   *util.BoundingBox
	ligatures     []*Ligature
}

// NewCharMetric returns an empty character metric.
func NewCharMetric() *CharMetric { return &CharMetric{} }

// BoundingBox returns the bounding box of the character.
func (c *CharMetric) BoundingBox() *util.BoundingBox { return c.boundingBox }

// SetBoundingBox sets the bounding box of the character.
func (c *CharMetric) SetBoundingBox(bBox *util.BoundingBox) { c.boundingBox = bBox }

// CharacterCode returns the code of the character.
func (c *CharMetric) CharacterCode() int { return c.characterCode }

// SetCharacterCode sets the code of the character.
func (c *CharMetric) SetCharacterCode(cCode int) { c.characterCode = cCode }

// AddLigature adds a ligature this character forms.
func (c *CharMetric) AddLigature(ligature *Ligature) {
	c.ligatures = append(c.ligatures, ligature)
}

// Ligatures returns the ligatures this character forms, as a copy.
func (c *CharMetric) Ligatures() []*Ligature {
	return append([]*Ligature{}, c.ligatures...)
}

// Name returns the name of the character.
func (c *CharMetric) Name() string { return c.name }

// SetName sets the name of the character.
func (c *CharMetric) SetName(n string) { c.name = n }

// Vv returns the vector from the origin of writing direction 0 to that of
// direction 1.
func (c *CharMetric) Vv() []float32 { return c.vv }

// SetVv sets that vector.
func (c *CharMetric) SetVv(vvValue []float32) { c.vv = vvValue }

// W returns the width vector.
func (c *CharMetric) W() []float32 { return c.w }

// SetW sets the width vector.
func (c *CharMetric) SetW(wValue []float32) { c.w = wValue }

// W0 returns the width vector in writing direction 0.
func (c *CharMetric) W0() []float32 { return c.w0 }

// SetW0 sets the width vector in writing direction 0.
func (c *CharMetric) SetW0(w0Value []float32) { c.w0 = w0Value }

// W0x returns the horizontal width in writing direction 0.
func (c *CharMetric) W0x() float32 { return c.w0x }

// SetW0x sets the horizontal width in writing direction 0.
func (c *CharMetric) SetW0x(w0xValue float32) { c.w0x = w0xValue }

// W0y returns the vertical width in writing direction 0.
func (c *CharMetric) W0y() float32 { return c.w0y }

// SetW0y sets the vertical width in writing direction 0.
func (c *CharMetric) SetW0y(w0yValue float32) { c.w0y = w0yValue }

// W1 returns the width vector in writing direction 1.
func (c *CharMetric) W1() []float32 { return c.w1 }

// SetW1 sets the width vector in writing direction 1.
func (c *CharMetric) SetW1(w1Value []float32) { c.w1 = w1Value }

// W1x returns the horizontal width in writing direction 1.
func (c *CharMetric) W1x() float32 { return c.w1x }

// SetW1x sets the horizontal width in writing direction 1.
func (c *CharMetric) SetW1x(w1xValue float32) { c.w1x = w1xValue }

// W1y returns the vertical width in writing direction 1.
func (c *CharMetric) W1y() float32 { return c.w1y }

// SetW1y sets the vertical width in writing direction 1.
func (c *CharMetric) SetW1y(w1yValue float32) { c.w1y = w1yValue }

// Wx returns the horizontal width.
func (c *CharMetric) Wx() float32 { return c.wx }

// SetWx sets the horizontal width.
func (c *CharMetric) SetWx(wxValue float32) { c.wx = wxValue }

// Wy returns the vertical width.
func (c *CharMetric) Wy() float32 { return c.wy }

// SetWy sets the vertical width.
func (c *CharMetric) SetWy(wyValue float32) { c.wy = wyValue }
