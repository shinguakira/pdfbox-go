package color

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDGamma is a gamma value for each of red, green and blue.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDGamma.
type PDGamma struct {
	values *cos.Array
}

var _ common.COSObjectable = (*PDGamma)(nil)

// NewPDGamma returns 0 0 0.
func NewPDGamma() *PDGamma {
	values := cos.NewArray()
	values.Add(cos.FloatZero)
	values.Add(cos.FloatZero)
	values.Add(cos.FloatZero)
	return &PDGamma{values: values}
}

// NewPDGammaOfArray wraps the given array.
func NewPDGammaOfArray(array *cos.Array) *PDGamma { return &PDGamma{values: array} }

// COSObject returns the array below this value.
func (g *PDGamma) COSObject() cos.Base { return g.values }

// COSArray returns the array below this value.
func (g *PDGamma) COSArray() *cos.Array { return g.values }

// R returns the red gamma.
func (g *PDGamma) R() float32 { return g.values.Get(0).(cos.Number).FloatValue() }

// SetR sets the red gamma.
func (g *PDGamma) SetR(r float32) { g.values.Set(0, cos.NewFloat(r)) }

// G returns the green gamma.
func (g *PDGamma) G() float32 { return g.values.Get(1).(cos.Number).FloatValue() }

// SetG sets the green gamma.
func (g *PDGamma) SetG(value float32) { g.values.Set(1, cos.NewFloat(value)) }

// B returns the blue gamma.
func (g *PDGamma) B() float32 { return g.values.Get(2).(cos.Number).FloatValue() }

// SetB sets the blue gamma.
func (g *PDGamma) SetB(b float32) { g.values.Set(2, cos.NewFloat(b)) }
