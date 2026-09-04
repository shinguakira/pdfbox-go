// Package type4 is the PostScript calculator a type 4 function is written in.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.
//
// The operand stack holds four kinds of value, which the Java holds as Object
// and tests with instanceof: int32 for Java's Integer, float32 for its Float,
// bool for its Boolean and *InstructionSequence for a procedure. The port uses
// those exact Go types so that every instanceof becomes a type switch on the
// same distinction -- an int and a real behave differently in a dozen
// operators, and collapsing them would change the arithmetic.
package type4

// Operator is one operator of the calculator language.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.Operator.
type Operator interface {
	// Execute runs the operator against the given context.
	Execute(context *ExecutionContext)
}

// Operators is the set of operators the calculator knows.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.Operators.
type Operators struct {
	operators map[string]Operator
}

// NewOperators returns the operator set of the PDF calculator language.
func NewOperators() *Operators {
	return &Operators{operators: map[string]Operator{
		"add":      addOperator{},
		"abs":      absOperator{},
		"atan":     atanOperator{},
		"ceiling":  ceilingOperator{},
		"cos":      cosOperator{},
		"cvi":      cviOperator{},
		"cvr":      cvrOperator{},
		"div":      divOperator{},
		"exp":      expOperator{},
		"floor":    floorOperator{},
		"idiv":     idivOperator{},
		"ln":       lnOperator{},
		"log":      logOperator{},
		"mod":      modOperator{},
		"mul":      mulOperator{},
		"neg":      negOperator{},
		"round":    roundOperator{},
		"sin":      sinOperator{},
		"sqrt":     sqrtOperator{},
		"sub":      subOperator{},
		"truncate": truncateOperator{},

		"and":      logicalOperator{andLogic{}},
		"bitshift": bitshiftOperator{},
		"eq":       eqOperator{},
		"false":    falseOperator{},
		"ge":       comparisonOperator{geCompare},
		"gt":       comparisonOperator{gtCompare},
		"le":       comparisonOperator{leCompare},
		"lt":       comparisonOperator{ltCompare},
		"ne":       neOperator{},
		"not":      notOperator{},
		"or":       logicalOperator{orLogic{}},
		"true":     trueOperator{},
		"xor":      logicalOperator{xorLogic{}},

		"if":     ifOperator{},
		"ifelse": ifElseOperator{},

		"copy":  copyOperator{},
		"dup":   dupOperator{},
		"exch":  exchOperator{},
		"index": indexOperator{},
		"pop":   popOperator{},
		"roll":  rollOperator{},
	}}
}

// GetOperator returns the operator of the given name, or nil where there is
// none.
func (o *Operators) GetOperator(operatorName string) Operator {
	return o.operators[operatorName]
}
