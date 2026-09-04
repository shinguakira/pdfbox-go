package cff

import (
	"errors"
	"fmt"
	"log/slog"
)

// The commands the Type 1 charstring parser watches for by value.
const (
	// 1-byte commands
	type1CallSubr = 10

	// 2-byte commands
	type1TwoByte       = 12
	type1CallOtherSubr = 16
	type1Pop           = 17
)

// Type1CharStringParser represents a converter for a mapping into a Type 1
// sequence.
//
// See "Adobe Type 1 Font Format, Adobe Systems (1999)".
//
// Port of org.apache.fontbox.cff.Type1CharStringParser.
type Type1CharStringParser struct {
	fontName     string
	currentGlyph string
}

// NewType1CharStringParser constructs a new Type1CharStringParser object.
func NewType1CharStringParser(fontName string) *Type1CharStringParser {
	return &Type1CharStringParser{fontName: fontName}
}

// Parse parses the given byte array and converts it to a Type1 sequence, subrs
// being the list of local subroutines and glyphName the name of the current
// glyph.
func (p *Type1CharStringParser) Parse(bytes []byte, subrs [][]byte, glyphName string) ([]any, error) {
	p.currentGlyph = glyphName
	return p.parse(bytes, subrs, []any{})
}

func (p *Type1CharStringParser) parse(bytes []byte, subrs [][]byte, sequence []any) ([]any, error) {
	input := NewDataInputByteArray(bytes)
	for {
		hasRemaining, err := input.HasRemaining()
		if err != nil {
			return nil, err
		}
		if !hasRemaining {
			break
		}
		b0, err := input.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}
		isCallOtherSubr := false
		if b0 == type1TwoByte {
			if isCallOtherSubr, err = peeksAt(input, 0, type1CallOtherSubr); err != nil {
				return nil, err
			}
		}
		switch {
		case b0 == type1CallSubr:
			sequence, err = p.processCallSubr(subrs, sequence)
		case isCallOtherSubr:
			sequence, err = p.processCallOtherSubr(input, sequence)
		case b0 >= 0 && b0 <= 31:
			var command *CharStringCommand
			command, err = p.readCommand(input, b0)
			if err == nil {
				sequence = append(sequence, command)
			}
		case b0 >= 32 && b0 <= 255:
			var number int
			number, err = p.readNumber(input, b0)
			if err == nil {
				sequence = append(sequence, number)
			}
		default:
			// Java throws IllegalArgumentException, which is unreachable: b0 is
			// an unsigned byte, so the two ranges above cover it.
			panic("cff: unreachable charstring byte")
		}
		if err != nil {
			return nil, err
		}
	}
	return sequence, nil
}

// peeksAt reports whether the byte at the given offset is the given value.
//
// Java's peekUnsignedByte throws past the end of the buffer, and the two
// conditions below sit where that exception would propagate out of the parse,
// so the error travels with the answer rather than being swallowed.
func peeksAt(input DataInput, offset, value int) (bool, error) {
	peeked, err := input.PeekUnsignedByte(offset)
	if err != nil {
		return false, err
	}
	return peeked == value, nil
}

func (p *Type1CharStringParser) processCallSubr(subrs [][]byte, sequence []any) ([]any, error) {
	// callsubr command
	obj := sequence[len(sequence)-1]
	sequence = sequence[:len(sequence)-1]
	if !isInteger(obj) {
		slog.Warn("Parameter for CALLSUBR is ignored, integer expected",
			"parameter", obj, "glyph", p.currentGlyph, "font", p.fontName)
		return sequence, nil
	}
	operand := obj.(int)

	if operand >= 0 && operand < len(subrs) {
		subrBytes := subrs[operand]
		var err error
		sequence, err = p.parse(subrBytes, subrs, sequence)
		if err != nil {
			return nil, err
		}
		lastItem := sequence[len(sequence)-1]
		if command, ok := lastItem.(*CharStringCommand); ok &&
			command.Type1KeyWord() == Type1RET {
			sequence = sequence[:len(sequence)-1] // remove "return" command
		}
		return sequence, nil
	}
	slog.Warn("CALLSUBR is ignored", "operand", operand, "subrs.size()", len(subrs),
		"glyph", p.currentGlyph, "font", p.fontName)
	// remove all parameters (there can be more than one)
	for len(sequence) != 0 && isInteger(sequence[len(sequence)-1]) {
		sequence = sequence[:len(sequence)-1]
	}
	return sequence, nil
}

func (p *Type1CharStringParser) processCallOtherSubr(input DataInput, sequence []any) ([]any, error) {
	// callothersubr command (needed in order to expand Subrs)
	if _, err := input.ReadSignedByte(); err != nil {
		return nil, err
	}

	// Java casts both to Integer and throws ClassCastException on anything
	// else, which is unchecked.
	othersubrNum := sequence[len(sequence)-1].(int)
	numArgs := sequence[len(sequence)-2].(int)
	sequence = sequence[:len(sequence)-2]

	// othersubrs 0-3 have their own semantics
	var results []int
	push := func(value int) { results = append([]int{value}, results...) }
	var err error
	switch othersubrNum {
	case 0:
		var value int
		if value, sequence, err = removeInteger(sequence); err != nil {
			return nil, err
		}
		push(value)
		if value, sequence, err = removeInteger(sequence); err != nil {
			return nil, err
		}
		push(value)
		sequence = sequence[:len(sequence)-1]
		// end flex
		sequence = append(sequence, 0)
		sequence = append(sequence, CmdCALLOTHERSUBR)
	case 1:
		// begin flex
		sequence = append(sequence, 1)
		sequence = append(sequence, CmdCALLOTHERSUBR)
	case 3:
		// allows hint replacement
		var value int
		if value, sequence, err = removeInteger(sequence); err != nil {
			return nil, err
		}
		push(value)
	default:
		// all remaining othersubrs use this fallback mechanism
		for i := 0; i < numArgs; i++ {
			var value int
			if value, sequence, err = removeInteger(sequence); err != nil {
				return nil, err
			}
			push(value)
		}
	}

	// pop must follow immediately
	for {
		isPop, err := peeksAt(input, 0, type1TwoByte)
		if err != nil {
			return nil, err
		}
		if isPop {
			if isPop, err = peeksAt(input, 1, type1Pop); err != nil {
				return nil, err
			}
		}
		if !isPop {
			break
		}
		if _, err := input.ReadSignedByte(); err != nil { // B0_POP
			return nil, err
		}
		if _, err := input.ReadSignedByte(); err != nil { // B1_POP
			return nil, err
		}
		sequence = append(sequence, results[0])
		results = results[1:]
	}

	if len(results) != 0 {
		slog.Warn("Value left on the PostScript stack",
			"glyph", p.currentGlyph, "font", p.fontName)
	}
	return sequence, nil
}

// removeInteger is a workaround for the fact that Type1CharStringParser assumes
// that subrs and othersubrs can be unrolled without executing the 'div'
// operator, which isn't true.
func removeInteger(sequence []any) (int, []any, error) {
	item := sequence[len(sequence)-1]
	sequence = sequence[:len(sequence)-1]
	if value, ok := item.(int); ok {
		return value, sequence, nil
	}
	command := item.(*CharStringCommand)

	// div
	if command.Type1KeyWord() == Type1DIV {
		a := sequence[len(sequence)-1].(int)
		b := sequence[len(sequence)-2].(int)
		sequence = sequence[:len(sequence)-2]
		return b / a, sequence, nil
	}
	return 0, sequence, fmt.Errorf("Unexpected char string command: %v", command.Type1KeyWord())
}

func (p *Type1CharStringParser) readCommand(input DataInput, b0 int) (*CharStringCommand, error) {
	if b0 == 12 {
		b1, err := input.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}
		return GetInstance2(b0, b1), nil
	}
	return GetInstance(b0), nil
}

func (p *Type1CharStringParser) readNumber(input DataInput, b0 int) (int, error) {
	switch {
	case b0 >= 32 && b0 <= 246:
		return b0 - 139, nil
	case b0 >= 247 && b0 <= 250:
		b1, err := input.ReadUnsignedByte()
		if err != nil {
			return 0, err
		}
		return (b0-247)*256 + b1 + 108, nil
	case b0 >= 251 && b0 <= 254:
		b1, err := input.ReadUnsignedByte()
		if err != nil {
			return 0, err
		}
		return -(b0-251)*256 - b1 - 108, nil
	case b0 == 255:
		value, err := input.ReadInt()
		if err != nil {
			return 0, err
		}
		return int(value), nil
	}
	return 0, errors.New("cff: unreachable charstring number")
}
