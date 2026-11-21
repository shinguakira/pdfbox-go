package type4

import "errors"

// The two Java exceptions PDFunctionType4.eval turns into an IOException since
// PDFBOX-6320: EmptyStackException, which java.util.Stack's pop and peek throw,
// and ClassCastException, which the casts the operators make on their operands
// throw. The port panics with errors wrapping these where the Java throws them,
// so that Eval can tell them from the unchecked failures it still lets through
// -- the IllegalArgumentException of `rangecheck` and of a negative `sqrt`, the
// UnsupportedOperationException of an unknown operator, and the
// NumberFormatException of a token that is not a number.
var (
	// ErrEmptyStack stands for Java's EmptyStackException.
	ErrEmptyStack = errors.New("empty stack")
	// ErrWrongType stands for Java's ClassCastException.
	ErrWrongType = errors.New("wrong type")
)
