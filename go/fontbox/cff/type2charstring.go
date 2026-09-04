package cff

// Type2CharString represents a Type 2 CharString by converting it into an
// equivalent Type 1 CharString.
//
// Port of org.apache.fontbox.cff.Type2CharString, which extends
// Type1CharString; the port embeds that one.
type Type2CharString struct {
	*Type1CharString

	defWidthX     float32
	nominalWidthX float32
	pathCount     int
	gid           int
}

// NewType2CharString builds the charstring, font being the parent CFF font,
// glyphName the glyph name (or CID as hex string), sequence the Type 2 char
// string sequence, defaultWidthX the default width and nomWidthX the nominal
// width.
func NewType2CharString(font Type1CharStringReader, fontName, glyphName string, gid int,
	sequence []any, defaultWidthX, nomWidthX int) *Type2CharString {
	c := &Type2CharString{
		Type1CharString: newType1CharString(font, fontName, glyphName),
		gid:             gid,
		defWidthX:       float32(defaultWidthX),
		nominalWidthX:   float32(nomWidthX),
	}
	c.convertType1ToType2(sequence)
	return c
}

// GID returns the GID (glyph id) of this charstring.
func (c *Type2CharString) GID() int { return c.gid }

// convertType1ToType2 converts a sequence of Type 2 commands into a sequence of
// Type 1 commands.
func (c *Type2CharString) convertType1ToType2(sequence []any) {
	c.pathCount = 0

	// PDFBOX-5987: the sequence contains several "num denom DIV" sequences whose
	// results are used for further operations. However the converter only
	// handles direct arguments properly, not arguments that are created at
	// runtime on the stack. It's not possible to fix this by just copying the
	// command codes because addAlternatingCurve / addCurve require switching the
	// sequence of arguments.
	// The solution below just replaces all "num denom DIV" sequences with its
	// result. If more files with even more complex sequences appear we will have
	// to get rid of the converter and implement a complete renderer like with
	// type1 charstrings.
	newSequence := make([]any, 0, len(sequence))
	for i := 0; i < len(sequence); i++ {
		if sequence[i] == any(CmdDIV) && i >= 2 {
			num := sequence[i-2]
			den := sequence[i-1]
			if isNumber(num) && isNumber(den) {
				f := numberFloat(num) / numberFloat(den)
				newSequence = newSequence[:len(newSequence)-2]
				newSequence = append(newSequence, f)
			} else {
				newSequence = append(newSequence, sequence[i]) // GIGO
			}
		} else {
			newSequence = append(newSequence, sequence[i])
		}
	}

	numbers := []any{}
	for _, obj := range newSequence {
		if command, ok := obj.(*CharStringCommand); ok {
			// Java clears the list and refills it from the result, which is
			// always empty; the port simply takes the result.
			numbers = c.convertType2Command(numbers, command)
		} else {
			numbers = append(numbers, obj)
		}
	}
}

func (c *Type2CharString) convertType2Command(numbers []any, command *CharStringCommand) []any {
	type2KeyWord := command.Type2KeyWord()
	if type2KeyWord == Type2KeyWordNone {
		c.addCommand(numbers, command)
		return nil
	}
	switch type2KeyWord {
	case Type2HSTEM, Type2HSTEMHM, Type2VSTEM, Type2VSTEMHM, Type2HINTMASK, Type2CNTRMASK:
		numbers = c.clearStack(numbers, len(numbers)%2 != 0)
		c.expandStemHints(numbers, type2KeyWord == Type2HSTEM || type2KeyWord == Type2HSTEMHM)
	case Type2HMOVETO, Type2VMOVETO:
		numbers = c.clearStack(numbers, len(numbers) > 1)
		c.markPath()
		c.addCommand(numbers, command)
	case Type2RLINETO:
		c.addCommandList(split(numbers, 2), command)
	case Type2HLINETO, Type2VLINETO:
		c.addAlternatingLine(numbers, type2KeyWord == Type2HLINETO)
	case Type2RRCURVETO:
		c.addCommandList(split(numbers, 6), command)
	case Type2ENDCHAR:
		numbers = c.clearStack(numbers, len(numbers) == 5 || len(numbers) == 1)
		c.closeCharString2Path()
		if len(numbers) == 4 {
			// deprecated "seac" operator
			numbers = append([]any{0}, numbers...)
			c.addCommand(numbers, GetInstance2(12, 6))
		} else {
			c.addCommand(numbers, command)
		}
	case Type2RMOVETO:
		numbers = c.clearStack(numbers, len(numbers) > 2)
		c.markPath()
		c.addCommand(numbers, command)
	case Type2HVCURVETO, Type2VHCURVETO:
		c.addAlternatingCurve(numbers, type2KeyWord == Type2HVCURVETO)
	case Type2HFLEX:
		if len(numbers) >= 7 {
			first := []any{numbers[0], 0, numbers[1], numbers[2], numbers[3], 0}
			second := []any{numbers[4], 0, numbers[5], -numberFloat(numbers[2]), numbers[6], 0}
			c.addCommandList([][]any{first, second}, CmdRRCURVETO)
		}
	case Type2FLEX:
		first := numbers[0:6]
		second := numbers[6:12]
		c.addCommandList([][]any{first, second}, CmdRRCURVETO)
	case Type2HFLEX1:
		if len(numbers) >= 9 {
			first := []any{numbers[0], numbers[1], numbers[2], numbers[3], numbers[4], 0}
			second := []any{numbers[5], 0, numbers[6], numbers[7], numbers[8], 0}
			c.addCommandList([][]any{first, second}, CmdRRCURVETO)
		}
	case Type2FLEX1:
		dx := 0
		dy := 0
		for i := 0; i < 5; i++ {
			dx += numberInt(numbers[i*2])
			dy += numberInt(numbers[i*2+1])
		}
		first := numbers[0:6]
		dxIsBigger := abs(dx) > abs(dy)
		fifth := any(-dx)
		sixth := numbers[10]
		if dxIsBigger {
			fifth = numbers[10]
			sixth = any(-dy)
		}
		second := []any{numbers[6], numbers[7], numbers[8], numbers[9], fifth, sixth}
		c.addCommandList([][]any{first, second}, CmdRRCURVETO)
	case Type2RCURVELINE:
		if len(numbers) >= 2 {
			c.addCommandList(split(numbers[0:len(numbers)-2], 6), CmdRRCURVETO)
			c.addCommand(numbers[len(numbers)-2:], CmdRLINETO)
		}
	case Type2RLINECURVE:
		if len(numbers) >= 6 {
			c.addCommandList(split(numbers[0:len(numbers)-6], 2), CmdRLINETO)
			c.addCommand(numbers[len(numbers)-6:], CmdRRCURVETO)
		}
	case Type2HHCURVETO, Type2VVCURVETO:
		c.addCurve(numbers, type2KeyWord == Type2HHCURVETO)
	default:
		c.addCommand(numbers, command)
	}
	return nil
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (c *Type2CharString) clearStack(numbers []any, flag bool) []any {
	if c.isSequenceEmpty() {
		if flag {
			c.addCommand([]any{0, numberFloat(numbers[0]) + c.nominalWidthX}, CmdHSBW)
			numbers = numbers[1:]
		} else {
			c.addCommand([]any{0, c.defWidthX}, CmdHSBW)
		}
	}
	return numbers
}

func (c *Type2CharString) expandStemHints(numbers []any, horizontal bool) {
	// TODO
}

func (c *Type2CharString) markPath() {
	if c.pathCount > 0 {
		c.closeCharString2Path()
	}
	c.pathCount++
}

func (c *Type2CharString) closeCharString2Path() {
	var command *CharStringCommand
	if c.pathCount > 0 {
		// Java casts the last entry to CharStringCommand, which throws where it
		// is a number; nothing puts a number last, since every addCommand ends
		// with its command.
		command, _ = c.lastSequenceEntry().(*CharStringCommand)
	}
	if command != nil && command.Type1KeyWord() != Type1CLOSEPATH {
		c.addCommand(nil, CmdCLOSEPATH)
	}
}

func (c *Type2CharString) addAlternatingLine(numbers []any, horizontal bool) {
	for len(numbers) != 0 {
		if horizontal {
			c.addCommand(numbers[0:1], CmdHLINETO)
		} else {
			c.addCommand(numbers[0:1], CmdVLINETO)
		}
		numbers = numbers[1:]
		horizontal = !horizontal
	}
}

func (c *Type2CharString) addAlternatingCurve(numbers []any, horizontal bool) {
	for len(numbers) >= 4 {
		last := len(numbers) == 5
		if horizontal {
			fifth := any(0)
			if last {
				fifth = numbers[4]
			}
			c.addCommand([]any{numbers[0], 0, numbers[1], numbers[2], fifth, numbers[3]},
				CmdRRCURVETO)
		} else {
			sixth := any(0)
			if last {
				sixth = numbers[4]
			}
			c.addCommand([]any{0, numbers[0], numbers[1], numbers[2], numbers[3], sixth},
				CmdRRCURVETO)
		}
		if last {
			numbers = numbers[5:]
		} else {
			numbers = numbers[4:]
		}
		horizontal = !horizontal
	}
}

func (c *Type2CharString) addCurve(numbers []any, horizontal bool) {
	for len(numbers) >= 4 {
		first := len(numbers)%4 == 1

		pick := func(withFirst, without int) any {
			if first {
				return numbers[withFirst]
			}
			return numbers[without]
		}
		if horizontal {
			second := any(0)
			if first {
				second = numbers[0]
			}
			c.addCommand([]any{pick(1, 0), second, pick(2, 1), pick(3, 2), pick(4, 3), 0},
				CmdRRCURVETO)
		} else {
			one := any(0)
			if first {
				one = numbers[0]
			}
			c.addCommand([]any{one, pick(1, 0), pick(2, 1), pick(3, 2), 0, pick(4, 3)},
				CmdRRCURVETO)
		}
		if first {
			numbers = numbers[5:]
		} else {
			numbers = numbers[4:]
		}
	}
}

func (c *Type2CharString) addCommandList(numbers [][]any, command *CharStringCommand) {
	for _, ns := range numbers {
		c.addCommand(ns, command)
	}
}

// split cuts the list into runs of the given size, dropping whatever does not
// fill a whole one.
func split(list []any, size int) [][]any {
	listSize := len(list) / size
	result := make([][]any, 0, listSize)
	for i := 0; i < listSize; i++ {
		result = append(result, list[i*size:(i+1)*size])
	}
	return result
}
