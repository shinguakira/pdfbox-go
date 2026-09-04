package type4

import "math"

// The arithmetic operators of the calculator language.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.ArithmeticOperators,
// whose operators are static inner classes; each is a type here, named after
// the class it ports.

// floatToInt32 narrows a float to an int the way a Java (int) cast does:
// towards zero, with anything past the range clamped to it and NaN going to
// zero.
func floatToInt32(value float32) int32 {
	switch {
	case value != value: // NaN
		return 0
	case value >= math.MaxInt32:
		return math.MaxInt32
	case value <= math.MinInt32:
		return math.MinInt32
	}
	return int32(value)
}

type absOperator struct{}

func (absOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	if i, ok := num.(int32); ok {
		// Java's Math.abs(Integer.MIN_VALUE) is Integer.MIN_VALUE.
		if i < 0 {
			i = -i
		}
		context.Push(i)
	} else {
		context.Push(float32(math.Abs(float64(numberAsFloat(num)))))
	}
}

type addOperator struct{}

func (addOperator) Execute(context *ExecutionContext) {
	num2 := context.PopNumber()
	num1 := context.PopNumber()
	_, isInt1 := num1.(int32)
	_, isInt2 := num2.(int32)
	if isInt1 && isInt2 {
		sum := numberAsLong(num1) + numberAsLong(num2)
		if sum < math.MinInt32 || sum > math.MaxInt32 {
			context.Push(float32(sum))
		} else {
			context.Push(int32(sum))
		}
	} else {
		sum := numberAsFloat(num1) + numberAsFloat(num2)
		context.Push(sum)
	}
}

type atanOperator struct{}

func (atanOperator) Execute(context *ExecutionContext) {
	den := context.PopReal()
	num := context.PopReal()
	atan := float32(math.Atan2(float64(num), float64(den)))
	atan = float32(math.Mod(float64(atan)*180/math.Pi, 360))
	if atan < 0 {
		atan = atan + 360
	}
	context.Push(atan)
}

type ceilingOperator struct{}

func (ceilingOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	if _, ok := num.(int32); ok {
		context.Push(num)
	} else {
		context.Push(float32(math.Ceil(numberAsDouble(num))))
	}
}

type cosOperator struct{}

func (cosOperator) Execute(context *ExecutionContext) {
	angle := context.PopReal()
	cos := float32(math.Cos(float64(angle) * math.Pi / 180))
	context.Push(cos)
}

type cviOperator struct{}

func (cviOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	context.Push(numberAsInt(num))
}

type cvrOperator struct{}

func (cvrOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	context.Push(numberAsFloat(num))
}

type divOperator struct{}

func (divOperator) Execute(context *ExecutionContext) {
	num2 := context.PopNumber()
	num1 := context.PopNumber()
	context.Push(numberAsFloat(num1) / numberAsFloat(num2))
}

type expOperator struct{}

func (expOperator) Execute(context *ExecutionContext) {
	exp := context.PopNumber()
	base := context.PopNumber()
	value := math.Pow(numberAsDouble(base), numberAsDouble(exp))
	context.Push(float32(value))
}

type floorOperator struct{}

func (floorOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	if _, ok := num.(int32); ok {
		context.Push(num)
	} else {
		context.Push(float32(math.Floor(numberAsDouble(num))))
	}
}

type idivOperator struct{}

func (idivOperator) Execute(context *ExecutionContext) {
	num2 := context.PopInt()
	num1 := context.PopInt()
	context.Push(num1 / num2)
}

type lnOperator struct{}

func (lnOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	context.Push(float32(math.Log(numberAsDouble(num))))
}

type logOperator struct{}

func (logOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	context.Push(float32(math.Log10(numberAsDouble(num))))
}

type modOperator struct{}

func (modOperator) Execute(context *ExecutionContext) {
	int2 := context.PopInt()
	int1 := context.PopInt()
	context.Push(int1 % int2)
}

type mulOperator struct{}

func (mulOperator) Execute(context *ExecutionContext) {
	num2 := context.PopNumber()
	num1 := context.PopNumber()
	_, isInt1 := num1.(int32)
	_, isInt2 := num2.(int32)
	if isInt1 && isInt2 {
		result := numberAsLong(num1) * numberAsLong(num2)
		if result >= math.MinInt32 && result <= math.MaxInt32 {
			context.Push(int32(result))
		} else {
			context.Push(float32(result))
		}
	} else {
		result := numberAsDouble(num1) * numberAsDouble(num2)
		context.Push(float32(result))
	}
}

type negOperator struct{}

func (negOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	if v, ok := num.(int32); ok {
		if v == math.MinInt32 {
			context.Push(-numberAsFloat(num))
		} else {
			context.Push(-v)
		}
	} else {
		context.Push(-numberAsFloat(num))
	}
}

type roundOperator struct{}

func (roundOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	if _, ok := num.(int32); ok {
		context.Push(numberAsInt(num))
	} else {
		context.Push(float32(javaRound(numberAsDouble(num))))
	}
}

// javaRound is java.lang.Math.round(double), which is floor(x + 0.5) rather
// than Go's math.Round, and so takes a half towards positive infinity.
func javaRound(value float64) int64 {
	if math.IsNaN(value) {
		return 0
	}
	rounded := math.Floor(value + 0.5)
	switch {
	case rounded >= math.MaxInt64:
		return math.MaxInt64
	case rounded <= math.MinInt64:
		return math.MinInt64
	}
	return int64(rounded)
}

type sinOperator struct{}

func (sinOperator) Execute(context *ExecutionContext) {
	angle := context.PopReal()
	sin := float32(math.Sin(float64(angle) * math.Pi / 180))
	context.Push(sin)
}

type sqrtOperator struct{}

func (sqrtOperator) Execute(context *ExecutionContext) {
	num := context.PopReal()
	if num < 0 {
		// Java throws IllegalArgumentException, which is unchecked.
		panic("argument must be nonnegative")
	}
	context.Push(float32(math.Sqrt(float64(num))))
}

type subOperator struct{}

func (subOperator) Execute(context *ExecutionContext) {
	num2 := context.PopNumber()
	num1 := context.PopNumber()
	_, isInt1 := num1.(int32)
	_, isInt2 := num2.(int32)
	if isInt1 && isInt2 {
		result := numberAsLong(num1) - numberAsLong(num2)
		if result < math.MinInt32 || result > math.MaxInt32 {
			context.Push(float32(result))
		} else {
			context.Push(int32(result))
		}
	} else {
		result := numberAsFloat(num1) - numberAsFloat(num2)
		context.Push(result)
	}
}

type truncateOperator struct{}

func (truncateOperator) Execute(context *ExecutionContext) {
	num := context.PopNumber()
	if _, ok := num.(int32); ok {
		context.Push(numberAsInt(num))
	} else {
		context.Push(float32(floatToInt32(numberAsFloat(num))))
	}
}
