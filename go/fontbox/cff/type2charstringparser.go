package cff

// The commands the Type 2 charstring parser watches for by value.
var (
	// 1-byte commands
	type2CallSubr  = CmdCALLSUBR.Value()
	type2CallGSubr = CmdCALLGSUBR.Value()

	// not yet supported commands
	type2HintMask = CmdHINTMASK.Value()
	type2CntrMask = CmdCNTRMASK.Value()
)

// Type2CharStringParser represents a converter for a mapping into a
// Type2-sequence.
//
// Port of org.apache.fontbox.cff.Type2CharStringParser.
type Type2CharStringParser struct {
	fontName string
}

// NewType2CharStringParser constructs a new parser for a Type 1-equivalent
// font.
func NewType2CharStringParser(fontName string) *Type2CharStringParser {
	return &Type2CharStringParser{fontName: fontName}
}

// glyphData is the state one glyph's parse carries.
type glyphData struct {
	sequence   []any
	hstemCount int
	vstemCount int
}

// Parse parses the given byte array and converts it to a Type2 sequence,
// globalSubrIndex holding all global subroutines and localSubrIndex all local
// ones.
func (p *Type2CharStringParser) Parse(bytes []byte, globalSubrIndex, localSubrIndex [][]byte) ([]any, error) {
	data := &glyphData{sequence: []any{}}
	if err := p.parseSequence(bytes, globalSubrIndex, localSubrIndex, data); err != nil {
		return nil, err
	}
	return data.sequence, nil
}

func (p *Type2CharStringParser) parseSequence(bytes []byte, globalSubrIndex,
	localSubrIndex [][]byte, data *glyphData) error {
	input := NewDataInputByteArray(bytes)

	for {
		hasRemaining, err := input.HasRemaining()
		if err != nil {
			return err
		}
		if !hasRemaining {
			return nil
		}
		b0, err := input.ReadUnsignedByte()
		if err != nil {
			return err
		}
		switch {
		case b0 == type2CallSubr:
			err = p.processCallSubr(globalSubrIndex, localSubrIndex, data)
		case b0 == type2CallGSubr:
			err = p.processCallGSubr(globalSubrIndex, localSubrIndex, data)
		case b0 == type2HintMask || b0 == type2CntrMask:
			data.vstemCount += countNumbers(data.sequence) / 2
			maskLength := getMaskLength(data.hstemCount, data.vstemCount)
			// drop the following bytes representing the mask as long as we
			// don't support HINTMASK and CNTRMASK
			for i := 0; i < maskLength; i++ {
				if _, err = input.ReadUnsignedByte(); err != nil {
					return err
				}
			}
			data.sequence = append(data.sequence, GetInstance(b0))
		case (b0 >= 0 && b0 <= 18) || (b0 >= 21 && b0 <= 27) || (b0 >= 29 && b0 <= 31):
			var command *CharStringCommand
			command, err = p.readCommand(b0, input, data)
			if err == nil {
				data.sequence = append(data.sequence, command)
			}
		case b0 == 28 || (b0 >= 32 && b0 <= 255):
			var number any
			number, err = p.readNumber(b0, input)
			if err == nil {
				data.sequence = append(data.sequence, number)
			}
		default:
			// Java throws IllegalArgumentException; b0 is an unsigned byte, so
			// only 19, 20 and 28 fall outside the ranges above and 28 is
			// covered, which leaves the two mask commands the case above takes.
			panic("cff: unreachable charstring byte")
		}
		if err != nil {
			return err
		}
	}
}

func (p *Type2CharStringParser) getSubrBytes(subrIndex [][]byte, data *glyphData) []byte {
	// Java casts the last entry to Integer and throws on anything else.
	operand := data.sequence[len(data.sequence)-1].(int)
	data.sequence = data.sequence[:len(data.sequence)-1]
	subrNumber := calculateSubrNumber(operand, len(subrIndex))
	if subrNumber < len(subrIndex) {
		return subrIndex[subrNumber]
	}
	return nil
}

func (p *Type2CharStringParser) processCallSubr(globalSubrIndex, localSubrIndex [][]byte,
	data *glyphData) error {
	if len(localSubrIndex) > 0 {
		subrBytes := p.getSubrBytes(localSubrIndex, data)
		return p.processSubr(globalSubrIndex, localSubrIndex, subrBytes, data)
	}
	return nil
}

func (p *Type2CharStringParser) processCallGSubr(globalSubrIndex, localSubrIndex [][]byte,
	data *glyphData) error {
	if len(globalSubrIndex) > 0 {
		subrBytes := p.getSubrBytes(globalSubrIndex, data)
		return p.processSubr(globalSubrIndex, localSubrIndex, subrBytes, data)
	}
	return nil
}

func (p *Type2CharStringParser) processSubr(globalSubrIndex, localSubrIndex [][]byte,
	subrBytes []byte, data *glyphData) error {
	if err := p.parseSequence(subrBytes, globalSubrIndex, localSubrIndex, data); err != nil {
		return err
	}
	lastItem := data.sequence[len(data.sequence)-1]
	if command, ok := lastItem.(*CharStringCommand); ok && command.Type2KeyWord() == Type2RET {
		// remove "return" command
		data.sequence = data.sequence[:len(data.sequence)-1]
	}
	return nil
}

func calculateSubrNumber(operand, subrIndexlength int) int {
	if subrIndexlength < 1240 {
		return 107 + operand
	}
	if subrIndexlength < 33900 {
		return 1131 + operand
	}
	return 32768 + operand
}

func (p *Type2CharStringParser) readCommand(b0 int, input DataInput,
	data *glyphData) (*CharStringCommand, error) {
	switch b0 {
	case 1, 18:
		data.hstemCount += countNumbers(data.sequence) / 2
		return GetInstance(b0), nil
	case 3, 23:
		data.vstemCount += countNumbers(data.sequence) / 2
		return GetInstance(b0), nil
	case 12:
		b1, err := input.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}
		return GetInstance2(b0, b1), nil
	default:
		return GetInstance(b0), nil
	}
}

func (p *Type2CharStringParser) readNumber(b0 int, input DataInput) (any, error) {
	if b0 == 28 {
		value, err := input.ReadShort()
		if err != nil {
			return nil, err
		}
		return int(value), nil
	}
	if b0 >= 32 && b0 <= 246 {
		return b0 - 139, nil
	}
	if b0 >= 247 && b0 <= 250 {
		b1, err := input.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}
		return (b0-247)*256 + b1 + 108, nil
	}
	if b0 >= 251 && b0 <= 254 {
		b1, err := input.ReadUnsignedByte()
		if err != nil {
			return nil, err
		}
		return -(b0-251)*256 - b1 - 108, nil
	}
	if b0 == 255 {
		value, err := input.ReadShort()
		if err != nil {
			return nil, err
		}
		// The lower bytes are representing the digits after the decimal point
		low, err := input.ReadUnsignedShort()
		if err != nil {
			return nil, err
		}
		fraction := float64(low) / 65535
		// Java adds a short to a double, which widens to a Double.
		return float64(value) + fraction, nil
	}
	panic("cff: unreachable charstring number")
}

func getMaskLength(hstemCount, vstemCount int) int {
	hintCount := hstemCount + vstemCount
	length := hintCount / 8
	if hintCount%8 > 0 {
		length++
	}
	return length
}

// countNumbers counts the numbers at the tail of the sequence.
func countNumbers(sequence []any) int {
	count := 0
	for i := len(sequence) - 1; i > -1; i-- {
		if !isNumber(sequence[i]) {
			return count
		}
		count++
	}
	return count
}

// String returns the font name, as Java's toString does.
func (p *Type2CharStringParser) String() string { return p.fontName }
