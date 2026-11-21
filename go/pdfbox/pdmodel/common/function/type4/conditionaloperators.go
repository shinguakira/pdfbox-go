package type4

// The conditional operators of the calculator language.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.ConditionalOperators.

type ifOperator struct{}

func (ifOperator) Execute(context *ExecutionContext) {
	proc := context.PopProcedure()
	condition := context.PopBool()
	if condition {
		proc.Execute(context)
	}
}

type ifElseOperator struct{}

func (ifElseOperator) Execute(context *ExecutionContext) {
	proc2 := context.PopProcedure()
	proc1 := context.PopProcedure()
	condition := context.PopBool()
	if condition {
		proc1.Execute(context)
	} else {
		proc2.Execute(context)
	}
}
