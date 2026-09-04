package type4

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

// InstructionSequence is a parsed calculator program, or one procedure of one.
//
// Port of org.apache.pdfbox.pdmodel.common.function.type4.InstructionSequence.
// Java holds its instructions as Objects and tells a name from a value with
// instanceof String; the port holds the same four kinds the stack does, plus
// the operatorName type below for a name, so that a program which pushes a
// string could not be confused with one that names an operator.
type InstructionSequence struct {
	instructions []any
}

// operatorName is an instruction that names an operator, which Java holds as a
// bare String.
type operatorName string

// NewInstructionSequence returns an empty sequence.
func NewInstructionSequence() *InstructionSequence { return &InstructionSequence{} }

// AddName adds an operator name.
func (s *InstructionSequence) AddName(name string) {
	s.instructions = append(s.instructions, operatorName(name))
}

// AddInteger adds an integer literal.
func (s *InstructionSequence) AddInteger(value int32) {
	s.instructions = append(s.instructions, value)
}

// AddReal adds a real literal.
func (s *InstructionSequence) AddReal(value float32) {
	s.instructions = append(s.instructions, value)
}

// AddBoolean adds a boolean literal.
func (s *InstructionSequence) AddBoolean(value bool) {
	s.instructions = append(s.instructions, value)
}

// AddProc adds a nested procedure.
func (s *InstructionSequence) AddProc(child *InstructionSequence) {
	s.instructions = append(s.instructions, child)
}

// Execute runs the sequence against the given context.
func (s *InstructionSequence) Execute(context *ExecutionContext) {
	for _, o := range s.instructions {
		if name, ok := o.(operatorName); ok {
			cmd := context.Operators().GetOperator(string(name))
			if cmd != nil {
				cmd.Execute(context)
			} else {
				// Java throws UnsupportedOperationException, which is unchecked.
				panic(fmt.Sprintf("Unknown operator or name: %s", string(name)))
			}
		} else {
			context.Push(o)
		}
	}

	// Handles top-level procs that simply need to be executed
	for !context.IsEmpty() {
		nested, ok := context.Peek().(*InstructionSequence)
		if !ok {
			break
		}
		context.Pop()
		nested.Execute(context)
	}
}

// The patterns InstructionSequenceBuilder tells a number from a name with.
//
// Java compiles them with Pattern.compile and asks asMatchPredicate, which
// anchors at both ends; the Go patterns carry the anchors themselves.
var (
	matchesInteger = regexp.MustCompile(`^[\+\-]?\d+$`)
	matchesReal    = regexp.MustCompile(`^\-?\d*\.\d*([Ee]\-?\d+)?$`)
)

// instructionSequenceBuilder turns tokens into an InstructionSequence.
//
// Port of
// org.apache.pdfbox.pdmodel.common.function.type4.InstructionSequenceBuilder,
// which is a Parser.AbstractSyntaxHandler.
type instructionSequenceBuilder struct {
	mainSequence *InstructionSequence
	seqStack     []*InstructionSequence
}

var _ syntaxHandler = (*instructionSequenceBuilder)(nil)

func newInstructionSequenceBuilder() *instructionSequenceBuilder {
	main := NewInstructionSequence()
	return &instructionSequenceBuilder{
		mainSequence: main,
		seqStack:     []*InstructionSequence{main},
	}
}

// Parse turns the text of a type 4 function into a sequence of instructions.
//
// Port of the static InstructionSequenceBuilder.parse.
func Parse(text string) *InstructionSequence {
	builder := newInstructionSequenceBuilder()
	parse(text, builder)
	return builder.mainSequence
}

func (b *instructionSequenceBuilder) currentSequence() *InstructionSequence {
	return b.seqStack[len(b.seqStack)-1]
}

func (b *instructionSequenceBuilder) newLine(text string)    {}
func (b *instructionSequenceBuilder) whitespace(text string) {}
func (b *instructionSequenceBuilder) comment(text string)    {}

func (b *instructionSequenceBuilder) token(token string) {
	switch {
	case token == "{":
		child := NewInstructionSequence()
		b.currentSequence().AddProc(child)
		b.seqStack = append(b.seqStack, child)

	case token == "}":
		b.seqStack = b.seqStack[:len(b.seqStack)-1]

	default:
		if matchesInteger.MatchString(token) {
			b.currentSequence().AddInteger(ParseInt(token))
			return
		}

		if matchesReal.MatchString(token) {
			b.currentSequence().AddReal(ParseReal(token))
			return
		}

		// TODO Maybe implement radix numbers, such as 8#1777 or 16#FFFE

		b.currentSequence().AddName(token)
	}
}

// ParseInt reads an integer literal.
//
// Port of InstructionSequenceBuilder.parseInt, which is Integer.parseInt and
// throws NumberFormatException for a token that is not one; the pattern above
// is the only caller and has already checked the shape, except for a value too
// large to hold, which Java also rejects.
func ParseInt(token string) int32 {
	value, err := strconv.ParseInt(token, 10, 32)
	if err != nil {
		panic(fmt.Sprintf("For input string: \"%s\"", token))
	}
	return int32(value)
}

// ParseReal reads a real literal.
//
// Port of InstructionSequenceBuilder.parseReal, which is Float.parseFloat.
func ParseReal(token string) float32 {
	value, err := strconv.ParseFloat(token, 32)
	if err != nil {
		// Java's Float.parseFloat accepts "1." and ".5", which Go's does too;
		// what it also accepts and Go does not is a lone "." or "-.", which
		// the real pattern above lets through.
		if token == "." || token == "-." || token == "" {
			return 0
		}
		panic(fmt.Sprintf("For input string: \"%s\"", token))
	}
	return float32(value)
}

// javaFloatCompare is java.lang.Float.compare, which the eq operator uses: it
// orders NaN above every number and -0.0 below 0.0, where == says NaN equals
// nothing and the two zeroes are equal.
func javaFloatCompare(a, b float32) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	aBits := int32(math.Float32bits(a))
	bBits := int32(math.Float32bits(b))
	switch {
	case aBits == bBits:
		return 0
	case aBits < bBits:
		return -1
	default:
		return 1
	}
}
