package type4

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
		panic("type4: pop from an empty stack")
	}
	value := c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
	return value
}

// Peek returns the top value without taking it off.
func (c *ExecutionContext) Peek() any {
	if len(c.stack) == 0 {
		panic("type4: peek at an empty stack")
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
		panic("type4: the value is not a number")
	}
}

// PopInt takes an integer off the stack.
func (c *ExecutionContext) PopInt() int32 {
	value := c.Pop()
	i, ok := value.(int32)
	if !ok {
		panic("type4: the value is not an integer")
	}
	return i
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
		panic("type4: the value is not a number")
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
		panic("type4: the value is not a number")
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
		panic("type4: the value is not a number")
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
		panic("type4: the value is not a number")
	}
}
