package operator

import (
	"errors"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// ErrMissingOperand reports a PDF operator that is missing required operands.
//
// Port of org.apache.pdfbox.contentstream.operator.MissingOperandException.
var ErrMissingOperand = errors.New("operator has too few operands")

// MissingOperand returns the error for an operator that was given too few
// operands.
func MissingOperand(op *Operator, operands []cos.Base) error {
	return fmt.Errorf("%w: operator %s has too few operands: %v",
		ErrMissingOperand, op.Name(), operands)
}

// ErrEmptyGraphicsStack reports a restore executed when the graphics stack is
// empty.
//
// Port of
// org.apache.pdfbox.contentstream.operator.state.EmptyGraphicsStackException.
// Java keeps it in the state subpackage, where both the operator that raises it
// and the engine that catches it can see it. In Go that would be a cycle — the
// engine's package would import the operator package, which imports the
// engine's — so it lives here, in the package both sides already import.
var ErrEmptyGraphicsStack = errors.New("cannot execute restore, the graphics stack is empty")
