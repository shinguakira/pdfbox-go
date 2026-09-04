package type4

// The boolean and bitwise operators of the calculator language.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.BitwiseOperators.

// logicalOperator is Java's AbstractLogicalOperator, which reads two operands
// and applies one rule to booleans and another to integers.
type logicalOperator struct {
	logic binaryLogic
}

// binaryLogic is the pair of abstract methods AbstractLogicalOperator declares.
type binaryLogic interface {
	applyForBoolean(bool1, bool2 bool) bool
	applyForInteger(int1, int2 int32) int32
}

func (o logicalOperator) Execute(context *ExecutionContext) {
	op2 := context.Pop()
	op1 := context.Pop()
	bool1, isBool1 := op1.(bool)
	bool2, isBool2 := op2.(bool)
	if isBool1 && isBool2 {
		context.Push(o.logic.applyForBoolean(bool1, bool2))
		return
	}
	int1, isInt1 := op1.(int32)
	int2, isInt2 := op2.(int32)
	if isInt1 && isInt2 {
		context.Push(o.logic.applyForInteger(int1, int2))
		return
	}
	// Java throws ClassCastException, which is unchecked.
	panic("Operands must be bool/bool or int/int")
}

type andLogic struct{}

func (andLogic) applyForBoolean(bool1, bool2 bool) bool { return bool1 && bool2 }
func (andLogic) applyForInteger(int1, int2 int32) int32 { return int1 & int2 }

type orLogic struct{}

func (orLogic) applyForBoolean(bool1, bool2 bool) bool { return bool1 || bool2 }
func (orLogic) applyForInteger(int1, int2 int32) int32 { return int1 | int2 }

type xorLogic struct{}

func (xorLogic) applyForBoolean(bool1, bool2 bool) bool { return bool1 != bool2 }
func (xorLogic) applyForInteger(int1, int2 int32) int32 { return int1 ^ int2 }

type bitshiftOperator struct{}

func (bitshiftOperator) Execute(context *ExecutionContext) {
	shift := context.Pop().(int32)
	int1 := context.Pop().(int32)
	if shift < 0 {
		// Java's >> is arithmetic, and Go's is too for a signed left operand.
		context.Push(int1 >> uint(-shift))
	} else {
		context.Push(int1 << uint(shift))
	}
}

type falseOperator struct{}

func (falseOperator) Execute(context *ExecutionContext) { context.Push(false) }

type trueOperator struct{}

func (trueOperator) Execute(context *ExecutionContext) { context.Push(true) }

type notOperator struct{}

func (notOperator) Execute(context *ExecutionContext) {
	op1 := context.Pop()
	switch v := op1.(type) {
	case bool:
		context.Push(!v)
	case int32:
		// Java writes -int1 here, which is the arithmetic negation and not the
		// bitwise complement PostScript's `not` is defined as. It is what
		// PDFBox does, so it is what the port does. See migration/JAVA-BUGS.md.
		context.Push(-v)
	default:
		// Java throws ClassCastException, which is unchecked.
		panic("Operand must be bool or int")
	}
}
