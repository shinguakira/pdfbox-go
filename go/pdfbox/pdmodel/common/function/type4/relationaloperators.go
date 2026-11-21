package type4

// The relational operators of the calculator language.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.RelationalOperators.

type eqOperator struct{}

func (eqOperator) Execute(context *ExecutionContext) {
	op2 := context.Pop()
	op1 := context.Pop()
	context.Push(operandsEqual(op1, op2))
}

// operandsEqual is Java's Eq.isEqual: two numbers compare as doubles since
// PDFBOX-6331, anything else by Object.equals.
func operandsEqual(op1, op2 any) bool {
	if isNumber(op1) && isNumber(op2) {
		return javaDoubleCompare(numberAsFloat64(op1), numberAsFloat64(op2)) == 0
	}
	// Java's Object.equals on a Boolean or an InstructionSequence: the first
	// compares by value, the second by identity, which Go's == does for a bool
	// and for a pointer respectively.
	return op1 == op2
}

func isNumber(value any) bool {
	switch value.(type) {
	case int32, float32:
		return true
	default:
		return false
	}
}

type neOperator struct{}

// Execute is Java's Ne, which extends Eq and negates isEqual.
func (neOperator) Execute(context *ExecutionContext) {
	op2 := context.Pop()
	op1 := context.Pop()
	context.Push(!operandsEqual(op1, op2))
}

// comparisonOperator is Java's AbstractNumberComparisonOperator, which casts
// both operands to Number and applies one comparison.
type comparisonOperator struct {
	compare func(num1, num2 float64) bool
}

func (o comparisonOperator) Execute(context *ExecutionContext) {
	op2 := context.Pop()
	op1 := context.Pop()
	// Java casts to Number, which throws ClassCastException for anything else;
	// numberAsFloat64 panics for the same.
	context.Push(o.compare(numberAsFloat64(op1), numberAsFloat64(op2)))
}

func geCompare(num1, num2 float64) bool { return num1 >= num2 }
func gtCompare(num1, num2 float64) bool { return num1 > num2 }
func leCompare(num1, num2 float64) bool { return num1 <= num2 }

// ltCompare narrows both operands back to float32, because PDFBOX-6331 moved
// Ge, Gt and Le onto doubleValue() and left Lt on floatValue(). So two
// integers that differ above 2^24 are not less than one another here, where gt
// can now tell them apart. The Java in this tree is the specification, and its
// Lt arm was not written; see migration/JAVA-BUGS.md 93.
func ltCompare(num1, num2 float64) bool { return float32(num1) < float32(num2) }
