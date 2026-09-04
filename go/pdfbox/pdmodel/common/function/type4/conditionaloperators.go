package type4

// The conditional operators of the calculator language.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.ConditionalOperators.

type ifOperator struct{}

func (ifOperator) Execute(context *ExecutionContext) {
	proc := context.Pop().(*InstructionSequence)
	condition := context.Pop().(bool)
	if condition {
		proc.Execute(context)
	}
}

type ifElseOperator struct{}

func (ifElseOperator) Execute(context *ExecutionContext) {
	proc2 := context.Pop().(*InstructionSequence)
	proc1 := context.Pop().(*InstructionSequence)
	condition := context.Pop().(bool)
	if condition {
		proc1.Execute(context)
	} else {
		proc2.Execute(context)
	}
}
