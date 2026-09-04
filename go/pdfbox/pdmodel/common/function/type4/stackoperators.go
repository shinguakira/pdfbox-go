package type4

import "fmt"

// The stack operators of the calculator language.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.StackOperators.

type copyOperator struct{}

func (copyOperator) Execute(context *ExecutionContext) {
	n := int(numberAsInt(context.Pop()))
	if n > 0 {
		size := context.Size()
		// Need to copy to a new list to avoid ConcurrentModificationException
		duplicate := append([]any(nil), context.Stack()[size-n:size]...)
		context.AddAll(duplicate)
	}
}

type dupOperator struct{}

func (dupOperator) Execute(context *ExecutionContext) { context.Push(context.Peek()) }

type exchOperator struct{}

func (exchOperator) Execute(context *ExecutionContext) {
	any2 := context.Pop()
	any1 := context.Pop()
	context.Push(any2)
	context.Push(any1)
}

type indexOperator struct{}

func (indexOperator) Execute(context *ExecutionContext) {
	n := int(numberAsInt(context.Pop()))
	if n < 0 {
		// Java throws IllegalArgumentException, which is unchecked.
		panic(fmt.Sprintf("rangecheck: %d", n))
	}
	size := context.Size()
	context.Push(context.Get(size - n - 1))
}

type popOperator struct{}

func (popOperator) Execute(context *ExecutionContext) { context.Pop() }

type rollOperator struct{}

func (rollOperator) Execute(context *ExecutionContext) {
	j := int(numberAsInt(context.Pop()))
	n := int(numberAsInt(context.Pop()))
	if j == 0 {
		return // Nothing to do
	}
	if n < 0 {
		// Java throws IllegalArgumentException, which is unchecked.
		panic(fmt.Sprintf("rangecheck: %d", n))
	}
	var rolled []any
	var moved []any
	if j < 0 {
		// negative roll
		n1 := n + j
		for i := 0; i < n1; i++ {
			moved = append([]any{context.Pop()}, moved...)
		}
		for i := j; i < 0; i++ {
			rolled = append([]any{context.Pop()}, rolled...)
		}
		context.AddAll(moved)
		context.AddAll(rolled)
	} else {
		// positive roll
		n1 := n - j
		for i := j; i > 0; i-- {
			rolled = append([]any{context.Pop()}, rolled...)
		}
		for i := 0; i < n1; i++ {
			moved = append([]any{context.Pop()}, moved...)
		}
		context.AddAll(rolled)
		context.AddAll(moved)
	}
}
