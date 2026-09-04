package cff

// Type1KeyWord is the enum of all valid type1 key words.
//
// Port of org.apache.fontbox.cff.CharStringCommand.Type1KeyWord. Java leaves
// the field null on a command that has no type 1 key word; the zero value here
// stands for that absence.
type Type1KeyWord int

const (
	// Type1KeyWordNone is where Java has null.
	Type1KeyWordNone Type1KeyWord = iota
	Type1HSTEM
	Type1VSTEM
	Type1VMOVETO
	Type1RLINETO
	Type1HLINETO
	Type1VLINETO
	Type1RRCURVETO
	Type1CLOSEPATH
	Type1CALLSUBR
	Type1RET
	Type1ESCAPE
	Type1HSBW
	Type1ENDCHAR
	Type1RMOVETO
	Type1HMOVETO
	Type1VHCURVETO
	Type1HVCURVETO
	Type1DOTSECTION
	Type1VSTEM3
	Type1HSTEM3
	Type1SEAC
	Type1SBW
	Type1DIV
	Type1CALLOTHERSUBR
	Type1POP
	Type1SETCURRENTPOINT
)

// String names the key word.
func (k Type1KeyWord) String() string {
	if k <= Type1KeyWordNone || int(k) >= len(type1KeyWordNames) {
		return "none"
	}
	return type1KeyWordNames[k]
}

var type1KeyWordNames = []string{
	"none",
	"HSTEM",
	"VSTEM",
	"VMOVETO",
	"RLINETO",
	"HLINETO",
	"VLINETO",
	"RRCURVETO",
	"CLOSEPATH",
	"CALLSUBR",
	"RET",
	"ESCAPE",
	"HSBW",
	"ENDCHAR",
	"RMOVETO",
	"HMOVETO",
	"VHCURVETO",
	"HVCURVETO",
	"DOTSECTION",
	"VSTEM3",
	"HSTEM3",
	"SEAC",
	"SBW",
	"DIV",
	"CALLOTHERSUBR",
	"POP",
	"SETCURRENTPOINT",
}

// Type2KeyWord is the enum of all valid type2 key words.
//
// Port of org.apache.fontbox.cff.CharStringCommand.Type2KeyWord. Java leaves
// the field null on a command that has no type 2 key word; the zero value here
// stands for that absence.
type Type2KeyWord int

const (
	// Type2KeyWordNone is where Java has null.
	Type2KeyWordNone Type2KeyWord = iota
	Type2HSTEM
	Type2VSTEM
	Type2VMOVETO
	Type2RLINETO
	Type2HLINETO
	Type2VLINETO
	Type2RRCURVETO
	Type2CALLSUBR
	Type2RET
	Type2ESCAPE
	Type2ENDCHAR
	Type2HSTEMHM
	Type2HINTMASK
	Type2CNTRMASK
	Type2RMOVETO
	Type2HMOVETO
	Type2VSTEMHM
	Type2RCURVELINE
	Type2RLINECURVE
	Type2VVCURVETO
	Type2HHCURVETO
	Type2SHORTINT
	Type2CALLGSUBR
	Type2VHCURVETO
	Type2HVCURVETO
	Type2AND
	Type2OR
	Type2NOT
	Type2ABS
	Type2ADD
	Type2SUB
	Type2DIV
	Type2NEG
	Type2EQ
	Type2DROP
	Type2PUT
	Type2GET
	Type2IFELSE
	Type2RANDOM
	Type2MUL
	Type2SQRT
	Type2DUP
	Type2EXCH
	Type2INDEX
	Type2ROLL
	Type2HFLEX
	Type2FLEX
	Type2HFLEX1
	Type2FLEX1
)

// String names the key word.
func (k Type2KeyWord) String() string {
	if k <= Type2KeyWordNone || int(k) >= len(type2KeyWordNames) {
		return "none"
	}
	return type2KeyWordNames[k]
}

var type2KeyWordNames = []string{
	"none",
	"HSTEM",
	"VSTEM",
	"VMOVETO",
	"RLINETO",
	"HLINETO",
	"VLINETO",
	"RRCURVETO",
	"CALLSUBR",
	"RET",
	"ESCAPE",
	"ENDCHAR",
	"HSTEMHM",
	"HINTMASK",
	"CNTRMASK",
	"RMOVETO",
	"HMOVETO",
	"VSTEMHM",
	"RCURVELINE",
	"RLINECURVE",
	"VVCURVETO",
	"HHCURVETO",
	"SHORTINT",
	"CALLGSUBR",
	"VHCURVETO",
	"HVCURVETO",
	"AND",
	"OR",
	"NOT",
	"ABS",
	"ADD",
	"SUB",
	"DIV",
	"NEG",
	"EQ",
	"DROP",
	"PUT",
	"GET",
	"IFELSE",
	"RANDOM",
	"MUL",
	"SQRT",
	"DUP",
	"EXCH",
	"INDEX",
	"ROLL",
	"HFLEX",
	"FLEX",
	"HFLEX1",
	"FLEX1",
}
