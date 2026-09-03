// Package operator models the operators of a PDF content stream.
//
// Port of org.apache.pdfbox.contentstream.operator.
package operator

import (
	"fmt"
	"strings"
	"sync"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Operator is one content stream operator, such as Tj or re.
//
// Port of org.apache.pdfbox.contentstream.operator.Operator. Most operators are
// cached and shared, so Get returns the same value for the same name; the two
// inline image operators are the exception, because they carry data belonging
// to one occurrence.
type Operator struct {
	name            string
	imageData       []byte
	imageParameters *cos.Dictionary
}

var (
	operatorsMu sync.Mutex
	operators   = make(map[string]*Operator)
)

// Get returns the operator with the given name.
//
// Port of Operator.getOperator. It panics on a name starting with '/', where
// Java throws IllegalArgumentException from the constructor; use GetChecked
// where the name comes from parsed input rather than from a constant.
func Get(name string) *Operator {
	op, err := GetChecked(name)
	if err != nil {
		panic(err)
	}
	return op
}

// GetChecked returns the operator with the given name, reporting a name that
// cannot be one.
func GetChecked(name string) (*Operator, error) {
	if strings.HasPrefix(name, "/") {
		return nil, fmt.Errorf("operator: operators are not allowed to start with /: %q", name)
	}

	// BI and ID are never cached: each carries the parameters and data of one
	// inline image, so a shared instance would let one image overwrite another.
	if name == BeginInlineImageData || name == BeginInlineImage {
		return &Operator{name: name}, nil
	}

	operatorsMu.Lock()
	defer operatorsMu.Unlock()
	if op, ok := operators[name]; ok {
		return op, nil
	}
	op := &Operator{name: name}
	operators[name] = op
	return op, nil
}

// Name returns the operator's name.
func (o *Operator) Name() string { return o.name }

// ImageData returns the data of an inline image, for the ID operator.
func (o *Operator) ImageData() []byte { return o.imageData }

// SetImageData records the data of an inline image.
func (o *Operator) SetImageData(data []byte) { o.imageData = data }

// ImageParameters returns the dictionary of an inline image, for the BI
// operator.
func (o *Operator) ImageParameters() *cos.Dictionary { return o.imageParameters }

// SetImageParameters records the dictionary of an inline image.
func (o *Operator) SetImageParameters(params *cos.Dictionary) { o.imageParameters = params }

// String returns the Java toString form.
func (o *Operator) String() string { return "PDFOperator{" + o.name + "}" }
