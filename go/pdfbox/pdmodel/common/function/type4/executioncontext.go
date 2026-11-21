package type4

import "fmt"

// ExecutionContext is the stack a calculator function runs against, and the
// operators it may use.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.ExecutionContext.
// Java's java.util.Stack is a Vector, so its bottom is index 0 and its top the
// last element; the Go slice below is the same way round, which the stack
// operators depend on.
type ExecutionContext struct {
	operators *Operators
	stack     []any
}

// NewExecutionContext returns a context over an empty stack.
func NewExecutionContext(operatorSet *Operators) *ExecutionContext {
	return &ExecutionContext{operators: operatorSet}
}

// Stack returns the operand stack.
func (c *ExecutionContext) Stack() []any { return c.stack }

// Operators returns the operator set.
func (c *ExecutionContext) Operators() *Operators { return c.operators }

// Push puts a value on the stack.
func (c *ExecutionContext) Push(value any) { c.stack = append(c.stack, value) }

// Pop takes the top value off the stack.
//
// Java's Stack.pop throws EmptyStackException on an empty stack, which is
// unchecked; the port panics for the same.
func (c *ExecutionContext) Pop() any {
	if len(c.stack) == 0 {
		panic(fmt.Errorf("type4: %w: pop", ErrEmptyStack))
	}
	value := c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
	return value
}

// Peek returns the top value without taking it off.
func (c *ExecutionContext) Peek() any {
	if len(c.stack) == 0 {
		panic(fmt.Errorf("type4: %w: peek", ErrEmptyStack))
	}
	return c.stack[len(c.stack)-1]
}

// Size returns how many values are on the stack.
func (c *ExecutionContext) Size() int { return len(c.stack) }

// IsEmpty reports whether the stack is empty.
func (c *ExecutionContext) IsEmpty() bool { return len(c.stack) == 0 }

// Get returns the value at the given index from the bottom of the stack.
func (c *ExecutionContext) Get(index int) any { return c.stack[index] }

// AddAll pushes each value in order, which is Java's Stack.addAll.
func (c *ExecutionContext) AddAll(values []any) { c.stack = append(c.stack, values...) }

// PopNumber takes a number off the stack.
//
// Java casts to Number and throws ClassCastException for anything else; the
// port panics.
func (c *ExecutionContext) PopNumber() any {
	value := c.Pop()
	switch value.(type) {
	case int32, float32:
		return value
	default:
		panic(fmt.Errorf("type4: %w: the value is not a number", ErrWrongType))
	}
}

// PopInt takes an integer off the stack.
func (c *ExecutionContext) PopInt() int32 {
	value := c.Pop()
	i, ok := value.(int32)
	if !ok {
		panic(fmt.Errorf("type4: %w: the value is not an integer", ErrWrongType))
	}
	return i
}

// PopBool takes a boolean off the stack.
//
// Java casts to Boolean and throws ClassCastException for anything else; the
// port raises ErrWrongType, which PDFunctionType4.eval takes (PDFBOX-6320). A
// plain Go type assertion here would panic with *runtime.TypeAssertionError,
// which wraps nothing and so would escape that boundary — a hole the Java
// cannot have, since catch (ClassCastException) takes every failed cast.
func (c *ExecutionContext) PopBool() bool {
	value := c.Pop()
	b, isBool := value.(bool)
	if !isBool {
		panic(fmt.Errorf("type4: %w: the value is not a boolean", ErrWrongType))
	}
	return b
}

// PopProcedure takes a procedure off the stack.
//
// Java casts to InstructionSequence, and the same reasoning as PopBool applies.
func (c *ExecutionContext) PopProcedure() *InstructionSequence {
	value := c.Pop()
	procedure, isProcedure := value.(*InstructionSequence)
	if !isProcedure {
		panic(fmt.Errorf("type4: %w: the value is not a procedure", ErrWrongType))
	}
	return procedure
}

// PopReal takes a number off the stack as a float.
func (c *ExecutionContext) PopReal() float32 { return numberAsFloat(c.PopNumber()) }

// numberAsFloat is Java's Number.floatValue.
func numberAsFloat(value any) float32 {
	switch v := value.(type) {
	case int32:
		return float32(v)
	case float32:
		return v
	default:
		panic(fmt.Errorf("type4: %w: the value is not a number", ErrWrongType))
	}
}

// numberAsFloat64 is Java's Number.doubleValue. PDFBOX-6331 moved the eq, ge,
// gt and le operators onto it: an int32 widens to float64 exactly, where
// float32 rounds every integer above 2^24.
func numberAsFloat64(value any) float64 {
	switch v := value.(type) {
	case int32:
		return float64(v)
	case float32:
		return float64(v)
	default:
		panic(fmt.Errorf("type4: %w: the value is not a number", ErrWrongType))
	}
}

// numberAsInt is Java's Number.intValue, which truncates a float toward zero
// and saturates at the int bounds.
func numberAsInt(value any) int32 {
	switch v := value.(type) {
	case int32:
		return v
	case float32:
		return floatToInt32(v)
	default:
		panic(fmt.Errorf("type4: %w: the value is not a number", ErrWrongType))
	}
}

// numberAsLong is Java's Number.longValue.
func numberAsLong(value any) int64 {
	switch v := value.(type) {
	case int32:
		return int64(v)
	case float32:
		return int64(v)
	default:
		panic(fmt.Errorf("type4: %w: the value is not a number", ErrWrongType))
	}
}

// numberAsDouble is Java's Number.doubleValue.
func numberAsDouble(value any) float64 {
	switch v := value.(type) {
	case int32:
		return float64(v)
	case float32:
		return float64(v)
	default:
		panic(fmt.Errorf("type4: %w: the value is not a number", ErrWrongType))
	}
}
