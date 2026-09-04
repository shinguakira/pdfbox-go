package cff

// CharStringCommand represents a CharStringCommand.
//
// Port of the org.apache.fontbox.cff.CharStringCommand enum. Go has no enum
// with fields, so each member is a package-level value and the commands are
// compared by pointer, as Java compares enum members by identity.
type CharStringCommand struct {
	type1KeyWord Type1KeyWord
	type2KeyWord Type2KeyWord
	value        int
	stringValue  string
}

// newCharStringCommand is the enum constructor.
func newCharStringCommand(name string, type1KeyWord Type1KeyWord, type2KeyWord Type2KeyWord,
	value int) *CharStringCommand {
	stringValue := name + "|"
	if value == 99 {
		stringValue = "unknown command|"
	}
	return &CharStringCommand{
		type1KeyWord: type1KeyWord,
		type2KeyWord: type2KeyWord,
		value:        value,
		stringValue:  stringValue,
	}
}

// The commands, in the order the Java enum declares them.
var (
	CmdHSTEM           = newCharStringCommand("HSTEM", Type1HSTEM, Type2HSTEM, 1)
	CmdVSTEM           = newCharStringCommand("VSTEM", Type1VSTEM, Type2VSTEM, 3)
	CmdVMOVETO         = newCharStringCommand("VMOVETO", Type1VMOVETO, Type2VMOVETO, 4)
	CmdRLINETO         = newCharStringCommand("RLINETO", Type1RLINETO, Type2RLINETO, 5)
	CmdHLINETO         = newCharStringCommand("HLINETO", Type1HLINETO, Type2HLINETO, 6)
	CmdVLINETO         = newCharStringCommand("VLINETO", Type1VLINETO, Type2VLINETO, 7)
	CmdRRCURVETO       = newCharStringCommand("RRCURVETO", Type1RRCURVETO, Type2RRCURVETO, 8)
	CmdCLOSEPATH       = newCharStringCommand("CLOSEPATH", Type1CLOSEPATH, Type2KeyWordNone, 9)
	CmdCALLSUBR        = newCharStringCommand("CALLSUBR", Type1CALLSUBR, Type2CALLSUBR, 10)
	CmdRET             = newCharStringCommand("RET", Type1RET, Type2RET, 11)
	CmdESCAPE          = newCharStringCommand("ESCAPE", Type1ESCAPE, Type2ESCAPE, 12)
	CmdHSBW            = newCharStringCommand("HSBW", Type1HSBW, Type2KeyWordNone, 13)
	CmdENDCHAR         = newCharStringCommand("ENDCHAR", Type1ENDCHAR, Type2ENDCHAR, 14)
	CmdHSTEMHM         = newCharStringCommand("HSTEMHM", Type1KeyWordNone, Type2HSTEMHM, 18)
	CmdHINTMASK        = newCharStringCommand("HINTMASK", Type1KeyWordNone, Type2HINTMASK, 19)
	CmdCNTRMASK        = newCharStringCommand("CNTRMASK", Type1KeyWordNone, Type2CNTRMASK, 20)
	CmdRMOVETO         = newCharStringCommand("RMOVETO", Type1RMOVETO, Type2RMOVETO, 21)
	CmdHMOVETO         = newCharStringCommand("HMOVETO", Type1HMOVETO, Type2HMOVETO, 22)
	CmdVSTEMHM         = newCharStringCommand("VSTEMHM", Type1KeyWordNone, Type2VSTEMHM, 23)
	CmdRCURVELINE      = newCharStringCommand("RCURVELINE", Type1KeyWordNone, Type2RCURVELINE, 24)
	CmdRLINECURVE      = newCharStringCommand("RLINECURVE", Type1KeyWordNone, Type2RLINECURVE, 25)
	CmdVVCURVETO       = newCharStringCommand("VVCURVETO", Type1KeyWordNone, Type2VVCURVETO, 26)
	CmdHHCURVETO       = newCharStringCommand("HHCURVETO", Type1KeyWordNone, Type2HHCURVETO, 27)
	CmdSHORTINT        = newCharStringCommand("SHORTINT", Type1KeyWordNone, Type2SHORTINT, 28)
	CmdCALLGSUBR       = newCharStringCommand("CALLGSUBR", Type1KeyWordNone, Type2CALLGSUBR, 29)
	CmdVHCURVETO       = newCharStringCommand("VHCURVETO", Type1VHCURVETO, Type2VHCURVETO, 30)
	CmdHVCURVETO       = newCharStringCommand("HVCURVETO", Type1HVCURVETO, Type2HVCURVETO, 31)
	CmdDOTSECTION      = newCharStringCommand("DOTSECTION", Type1DOTSECTION, Type2KeyWordNone, 192)
	CmdVSTEM3          = newCharStringCommand("VSTEM3", Type1VSTEM3, Type2KeyWordNone, 193)
	CmdHSTEM3          = newCharStringCommand("HSTEM3", Type1HSTEM3, Type2KeyWordNone, 194)
	CmdAND             = newCharStringCommand("AND", Type1KeyWordNone, Type2AND, 195)
	CmdOR              = newCharStringCommand("OR", Type1KeyWordNone, Type2OR, 196)
	CmdNOT             = newCharStringCommand("NOT", Type1KeyWordNone, Type2NOT, 197)
	CmdSEAC            = newCharStringCommand("SEAC", Type1SEAC, Type2KeyWordNone, 198)
	CmdSBW             = newCharStringCommand("SBW", Type1SBW, Type2KeyWordNone, 199)
	CmdABS             = newCharStringCommand("ABS", Type1KeyWordNone, Type2ABS, 201)
	CmdADD             = newCharStringCommand("ADD", Type1KeyWordNone, Type2ADD, 202)
	CmdSUB             = newCharStringCommand("SUB", Type1KeyWordNone, Type2SUB, 203)
	CmdDIV             = newCharStringCommand("DIV", Type1DIV, Type2DIV, 204)
	CmdNEG             = newCharStringCommand("NEG", Type1KeyWordNone, Type2NEG, 206)
	CmdEQ              = newCharStringCommand("EQ", Type1KeyWordNone, Type2EQ, 207)
	CmdCALLOTHERSUBR   = newCharStringCommand("CALLOTHERSUBR", Type1CALLOTHERSUBR, Type2KeyWordNone, 208)
	CmdPOP             = newCharStringCommand("POP", Type1POP, Type2KeyWordNone, 209)
	CmdDROP            = newCharStringCommand("DROP", Type1KeyWordNone, Type2DROP, 210)
	CmdPUT             = newCharStringCommand("PUT", Type1KeyWordNone, Type2PUT, 212)
	CmdGET             = newCharStringCommand("GET", Type1KeyWordNone, Type2GET, 213)
	CmdIFELSE          = newCharStringCommand("IFELSE", Type1KeyWordNone, Type2IFELSE, 214)
	CmdRANDOM          = newCharStringCommand("RANDOM", Type1KeyWordNone, Type2RANDOM, 215)
	CmdMUL             = newCharStringCommand("MUL", Type1KeyWordNone, Type2MUL, 216)
	CmdSQRT            = newCharStringCommand("SQRT", Type1KeyWordNone, Type2SQRT, 218)
	CmdDUP             = newCharStringCommand("DUP", Type1KeyWordNone, Type2DUP, 219)
	CmdEXCH            = newCharStringCommand("EXCH", Type1KeyWordNone, Type2EXCH, 220)
	CmdINDEX           = newCharStringCommand("INDEX", Type1KeyWordNone, Type2INDEX, 221)
	CmdROLL            = newCharStringCommand("ROLL", Type1KeyWordNone, Type2ROLL, 222)
	CmdSETCURRENTPOINT = newCharStringCommand("SETCURRENTPOINT", Type1SETCURRENTPOINT, Type2KeyWordNone, 225)
	CmdHFLEX           = newCharStringCommand("HFLEX", Type1KeyWordNone, Type2HFLEX, 226)
	CmdFLEX            = newCharStringCommand("FLEX", Type1KeyWordNone, Type2FLEX, 227)
	CmdHFLEX1          = newCharStringCommand("HFLEX1", Type1KeyWordNone, Type2HFLEX1, 228)
	CmdFLEX1           = newCharStringCommand("FLEX1", Type1KeyWordNone, Type2FLEX1, 229)
	CmdUNKNOWN         = newCharStringCommand("UNKNOWN", Type1KeyWordNone, Type2KeyWordNone, 99)
)

// charStringCommands is every command, in declaration order.
var charStringCommands = []*CharStringCommand{
	CmdHSTEM,
	CmdVSTEM,
	CmdVMOVETO,
	CmdRLINETO,
	CmdHLINETO,
	CmdVLINETO,
	CmdRRCURVETO,
	CmdCLOSEPATH,
	CmdCALLSUBR,
	CmdRET,
	CmdESCAPE,
	CmdHSBW,
	CmdENDCHAR,
	CmdHSTEMHM,
	CmdHINTMASK,
	CmdCNTRMASK,
	CmdRMOVETO,
	CmdHMOVETO,
	CmdVSTEMHM,
	CmdRCURVELINE,
	CmdRLINECURVE,
	CmdVVCURVETO,
	CmdHHCURVETO,
	CmdSHORTINT,
	CmdCALLGSUBR,
	CmdVHCURVETO,
	CmdHVCURVETO,
	CmdDOTSECTION,
	CmdVSTEM3,
	CmdHSTEM3,
	CmdAND,
	CmdOR,
	CmdNOT,
	CmdSEAC,
	CmdSBW,
	CmdABS,
	CmdADD,
	CmdSUB,
	CmdDIV,
	CmdNEG,
	CmdEQ,
	CmdCALLOTHERSUBR,
	CmdPOP,
	CmdDROP,
	CmdPUT,
	CmdGET,
	CmdIFELSE,
	CmdRANDOM,
	CmdMUL,
	CmdSQRT,
	CmdDUP,
	CmdEXCH,
	CmdINDEX,
	CmdROLL,
	CmdSETCURRENTPOINT,
	CmdHFLEX,
	CmdFLEX,
	CmdHFLEX1,
	CmdFLEX1,
	CmdUNKNOWN,
}

// commandsByValue indexes the commands by their value, as Java's static
// initialiser does.
var commandsByValue = func() []*CharStringCommand {
	maximum := 0
	for _, c := range charStringCommands {
		maximum = max(maximum, c.Value())
	}
	byValue := make([]*CharStringCommand, maximum+1)
	for _, c := range charStringCommands {
		byValue[c.Value()] = c
	}
	return byValue
}()

// Value returns the value the command is encoded as.
func (c *CharStringCommand) Value() int { return c.value }

// GetInstance gets the CharStringCommand represented by the given value.
func GetInstance(b0 int) *CharStringCommand {
	var c *CharStringCommand
	if b0 >= 0 && b0 < len(commandsByValue) {
		c = commandsByValue[b0]
	}
	if c != nil {
		return c
	}
	return CmdUNKNOWN
}

// GetInstance2 gets the CharStringCommand represented by the given two values.
func GetInstance2(b0, b1 int) *CharStringCommand {
	return GetInstance((b0 << 4) + b1)
}

// GetInstanceValues gets the CharStringCommand represented by the given array.
func GetInstanceValues(values []int) *CharStringCommand {
	switch len(values) {
	case 1:
		return GetInstance(values[0])
	case 2:
		return GetInstance2(values[0], values[1])
	default:
		return CmdUNKNOWN
	}
}

// Type1KeyWord returns the underlying type1 key word.
func (c *CharStringCommand) Type1KeyWord() Type1KeyWord { return c.type1KeyWord }

// Type2KeyWord returns the underlying type2 key word.
func (c *CharStringCommand) Type2KeyWord() Type2KeyWord { return c.type2KeyWord }

// String returns the command's name followed by a pipe.
func (c *CharStringCommand) String() string { return c.stringValue }
