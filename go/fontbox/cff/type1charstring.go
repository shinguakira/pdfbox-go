package cff

import (
	"log/slog"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/fontbox/encoding"
)

// Type1CharStringReader is something which can read Type 1 CharStrings, namely
// Type 1 and CFF fonts.
//
// Port of the org.apache.fontbox.type1.Type1CharStringReader interface. It is
// declared here rather than in type1 because it names Type1CharString, and Go
// forbids the import cycle Java has; type1 aliases the name back into place.
type Type1CharStringReader interface {
	// Type1CharString returns the Type 1 CharString for the character with the
	// given name.
	Type1CharString(name string) (*Type1CharString, error)
}

// Type1CharString represents and renders a Type 1 CharString.
//
// Port of org.apache.fontbox.cff.Type1CharString.
type Type1CharString struct {
	font            Type1CharStringReader
	fontName        string
	glyphName       string
	path            *geom.Path2D
	width           int
	leftSideBearing *geom.PointFloat
	current         *geom.PointFloat
	isFlex          bool
	flexPoints      []*geom.PointFloat
	type1Sequence   []any
	commandCount    int
}

// NewType1CharString constructs a new Type1CharString object, font being the
// parent Type 1 CharString font and sequence the Type 1 char string sequence.
func NewType1CharString(font Type1CharStringReader, fontName, glyphName string,
	sequence []any) *Type1CharString {
	c := newType1CharString(font, fontName, glyphName)
	c.type1Sequence = append(c.type1Sequence, sequence...)
	return c
}

// newType1CharString is the constructor for use in subclasses.
func newType1CharString(font Type1CharStringReader, fontName, glyphName string) *Type1CharString {
	return &Type1CharString{
		font:      font,
		fontName:  fontName,
		glyphName: glyphName,
		current:   geom.NewPointFloat(0, 0),
	}
}

// Name returns the glyph name.
//
// todo: NEW name (or CID as hex)
func (c *Type1CharString) Name() string { return c.glyphName }

// Bounds returns the bounds of the renderer path.
func (c *Type1CharString) Bounds() *geom.Rectangle2D {
	c.renderOnce()
	return c.path.Bounds2D()
}

// Width returns the advance width of the glyph.
func (c *Type1CharString) Width() int {
	c.renderOnce()
	return c.width
}

// Path returns the path of the character.
func (c *Type1CharString) Path() *geom.Path2D {
	c.renderOnce()
	return c.path
}

// renderOnce renders the charstring the first time one of the three getters
// above asks for it.
//
// Java wraps the check in synchronized(LOG), a lock shared by every charstring
// of every font. It cannot be a Go mutex: seac renders another charstring from
// inside render, and where the accent resolves back to this same charstring a
// Go mutex would deadlock where Java's re-entrant one does not. The port leaves
// the check unguarded, as the TrueTypeFont table read does for the same reason.
func (c *Type1CharString) renderOnce() {
	if c.path == nil {
		c.render()
	}
}

// render renders the Type 1 char string sequence to a path.
func (c *Type1CharString) render() {
	c.path = geom.NewPathFloat()
	c.leftSideBearing = geom.NewPointFloat(0, 0)
	c.width = 0
	numbers := []any{}
	for _, obj := range c.type1Sequence {
		if command, ok := obj.(*CharStringCommand); ok {
			numbers = c.handleType1Command(numbers, command)
		} else {
			numbers = append(numbers, obj)
		}
	}
}

// handleType1Command runs one command and gives back the number list it leaves
// behind. Java mutates the caller's list in place, which Go cannot do through a
// slice, so the list travels back as a result.
func (c *Type1CharString) handleType1Command(numbers []any, command *CharStringCommand) []any {
	c.commandCount++
	type1KeyWord := command.Type1KeyWord()
	if type1KeyWord == Type1KeyWordNone {
		// indicates an invalid charstring
		slog.Warn("Unknown charstring command", "glyph", c.glyphName, "font", c.fontName)
		return numbers[:0]
	}
	switch type1KeyWord {
	case Type1RMOVETO:
		if len(numbers) >= 2 {
			if c.isFlex {
				c.flexPoints = append(c.flexPoints,
					geom.NewPointFloat(numberFloat(numbers[0]), numberFloat(numbers[1])))
			} else {
				c.rmoveTo(numberFloat(numbers[0]), numberFloat(numbers[1]))
			}
		}
	case Type1VMOVETO:
		if len(numbers) != 0 {
			if c.isFlex {
				// not in the Type 1 spec, but exists in some fonts
				c.flexPoints = append(c.flexPoints, geom.NewPointFloat(0, numberFloat(numbers[0])))
			} else {
				c.rmoveTo(0, numberFloat(numbers[0]))
			}
		}
	case Type1HMOVETO:
		if len(numbers) != 0 {
			if c.isFlex {
				// not in the Type 1 spec, but exists in some fonts
				c.flexPoints = append(c.flexPoints, geom.NewPointFloat(numberFloat(numbers[0]), 0))
			} else {
				c.rmoveTo(numberFloat(numbers[0]), 0)
			}
		}
	case Type1RLINETO:
		if len(numbers) >= 2 {
			c.rlineTo(numberFloat(numbers[0]), numberFloat(numbers[1]))
		}
	case Type1HLINETO:
		if len(numbers) != 0 {
			c.rlineTo(numberFloat(numbers[0]), 0)
		}
	case Type1VLINETO:
		if len(numbers) != 0 {
			c.rlineTo(0, numberFloat(numbers[0]))
		}
	case Type1RRCURVETO:
		if len(numbers) >= 6 {
			c.rrcurveTo(numberFloat(numbers[0]), numberFloat(numbers[1]), numberFloat(numbers[2]),
				numberFloat(numbers[3]), numberFloat(numbers[4]), numberFloat(numbers[5]))
		}
	case Type1CLOSEPATH:
		c.closeCharString1Path()
	case Type1SBW:
		if len(numbers) >= 3 {
			c.leftSideBearing = geom.NewPointFloat(numberFloat(numbers[0]), numberFloat(numbers[1]))
			c.width = numberInt(numbers[2])
			c.current.SetLocation(c.leftSideBearing.X(), c.leftSideBearing.Y())
		}
	case Type1HSBW:
		if len(numbers) >= 2 {
			c.leftSideBearing = geom.NewPointFloat(numberFloat(numbers[0]), 0)
			c.width = numberInt(numbers[1])
			c.current.SetLocation(c.leftSideBearing.X(), c.leftSideBearing.Y())
		}
	case Type1VHCURVETO:
		if len(numbers) >= 4 {
			c.rrcurveTo(0, numberFloat(numbers[0]), numberFloat(numbers[1]),
				numberFloat(numbers[2]), numberFloat(numbers[3]), 0)
		}
	case Type1HVCURVETO:
		if len(numbers) >= 4 {
			c.rrcurveTo(numberFloat(numbers[0]), 0, numberFloat(numbers[1]),
				numberFloat(numbers[2]), 0, numberFloat(numbers[3]))
		}
	case Type1SEAC:
		if len(numbers) >= 5 {
			c.seac(numberFloat(numbers[0]), numberFloat(numbers[1]), numberFloat(numbers[2]),
				numberInt(numbers[3]), numberInt(numbers[4]))
		}
	case Type1SETCURRENTPOINT:
		if len(numbers) >= 2 {
			c.setCurrentPoint(numberFloat(numbers[0]), numberFloat(numbers[1]))
		}
	case Type1CALLOTHERSUBR:
		if len(numbers) != 0 {
			c.callOtherSubr(numberInt(numbers[0]))
		}
	case Type1DIV:
		if len(numbers) >= 2 {
			b := numberFloat(numbers[len(numbers)-1])
			a := numberFloat(numbers[len(numbers)-2])

			result := a / b

			numbers = numbers[:len(numbers)-2]
			numbers = append(numbers, result)
			return numbers
		}
	case Type1HSTEM, Type1VSTEM, Type1HSTEM3, Type1VSTEM3, Type1DOTSECTION:
		// ignore hints
	case Type1ENDCHAR:
		// end
	case Type1RET, Type1CALLSUBR:
		// indicates an invalid charstring
		slog.Warn("Unexpected charstring command", "command", type1KeyWord,
			"glyph", c.glyphName, "font", c.fontName)
	default:
		// indicates a PDFBox bug
		panic("Unhandled command: " + type1KeyWord.String())
	}
	return numbers[:0]
}

// setCurrentPoint sets the current absolute point without performing a moveto.
// Used only with results from callothersubr.
func (c *Type1CharString) setCurrentPoint(x, y float32) {
	c.current.SetLocationFloat(x, y)
}

// callOtherSubr is flex (via OtherSubrs), num being the OtherSubrs entry number.
func (c *Type1CharString) callOtherSubr(num int) {
	switch {
	case num == 0:
		// end flex
		c.isFlex = false

		if len(c.flexPoints) < 7 {
			slog.Warn("flex without moveTo", "font", c.fontName, "glyph", c.glyphName,
				"command", c.commandCount)
			return
		}

		// reference point is relative to start point
		reference := c.flexPoints[0]
		reference.SetLocation(c.current.X()+reference.X(), c.current.Y()+reference.Y())

		// first point is relative to reference point
		first := c.flexPoints[1]
		first.SetLocation(reference.X()+first.X(), reference.Y()+first.Y())

		// make the first point relative to the start point
		first.SetLocation(first.X()-c.current.X(), first.Y()-c.current.Y())

		p1 := c.flexPoints[1]
		p2 := c.flexPoints[2]
		p3 := c.flexPoints[3]
		c.rrcurveTo(p1.XFloat(), p1.YFloat(), p2.XFloat(), p2.YFloat(), p3.XFloat(), p3.YFloat())

		p4 := c.flexPoints[4]
		p5 := c.flexPoints[5]
		p6 := c.flexPoints[6]
		c.rrcurveTo(p4.XFloat(), p4.YFloat(), p5.XFloat(), p5.YFloat(), p6.XFloat(), p6.YFloat())

		c.flexPoints = c.flexPoints[:0]
	case num == 1:
		// begin flex
		c.isFlex = true
	default:
		slog.Warn("Invalid callothersubr parameter", "num", num)
	}
}

// rmoveTo is a relative moveto.
func (c *Type1CharString) rmoveTo(dx, dy float32) {
	x := float32(c.current.X()) + dx
	y := float32(c.current.Y()) + dy
	c.path.MoveTo(float64(x), float64(y))
	c.current.SetLocationFloat(x, y)
}

// rlineTo is a relative lineto.
func (c *Type1CharString) rlineTo(dx, dy float32) {
	x := float32(c.current.X()) + dx
	y := float32(c.current.Y()) + dy
	if c.path.CurrentPoint() == nil {
		slog.Warn("rlineTo without initial moveTo", "font", c.fontName, "glyph", c.glyphName)
		c.path.MoveTo(float64(x), float64(y))
	} else {
		c.path.LineTo(float64(x), float64(y))
	}
	c.current.SetLocationFloat(x, y)
}

// rrcurveTo is a relative curveto.
func (c *Type1CharString) rrcurveTo(dx1, dy1, dx2, dy2, dx3, dy3 float32) {
	x1 := float32(c.current.X()) + dx1
	y1 := float32(c.current.Y()) + dy1
	x2 := x1 + dx2
	y2 := y1 + dy2
	x3 := x2 + dx3
	y3 := y2 + dy3
	if c.path.CurrentPoint() == nil {
		slog.Warn("rrcurveTo without initial moveTo", "font", c.fontName, "glyph", c.glyphName)
		c.path.MoveTo(float64(x3), float64(y3))
	} else {
		c.path.CurveTo(float64(x1), float64(y1), float64(x2), float64(y2),
			float64(x3), float64(y3))
	}
	c.current.SetLocationFloat(x3, y3)
}

// closeCharString1Path closes the path.
func (c *Type1CharString) closeCharString1Path() {
	if c.path.CurrentPoint() == nil {
		slog.Warn("closepath without initial moveTo", "font", c.fontName, "glyph", c.glyphName)
	} else {
		c.path.ClosePath()
	}
	c.path.MoveTo(c.current.X(), c.current.Y())
}

// seac is the Standard Encoding Accented Character, which makes an accented
// character from two other characters.
func (c *Type1CharString) seac(asb, adx, ady float32, bchar, achar int) {
	// base character
	baseName := encoding.StandardEncoding.Name(bchar)
	base, err := c.font.Type1CharString(baseName)
	if err != nil {
		slog.Warn("invalid seac character", "glyph", c.glyphName, "font", c.fontName, "err", err)
	} else {
		c.path.AppendIterator(base.Path().PathIterator(nil), false)
	}
	// accent character
	accentName := encoding.StandardEncoding.Name(achar)
	accent, err := c.font.Type1CharString(accentName)
	if err != nil {
		slog.Warn("invalid seac character", "glyph", c.glyphName, "font", c.fontName, "err", err)
		return
	}
	accentPath := accent.Path()
	if c.path == accentPath {
		// PDFBOX-5339: avoid ArrayIndexOutOfBoundsException
		// reproducable with poc file crash-4698e0dc7833a3f959d06707e01d03cda52a83f4
		slog.Warn("Path for base and for accent are same, ignored",
			"base", baseName, "accent", accentName)
		return
	}
	at := geom.TranslateInstance(
		c.leftSideBearing.X()+float64(adx)-float64(asb),
		c.leftSideBearing.Y()+float64(ady))
	c.path.AppendIterator(accentPath.PathIterator(at), false)
}

// addCommand adds a command to the type1 sequence, numbers being the parameters
// of the command to be added.
func (c *Type1CharString) addCommand(numbers []any, command *CharStringCommand) {
	c.type1Sequence = append(c.type1Sequence, numbers...)
	c.type1Sequence = append(c.type1Sequence, command)
}

// isSequenceEmpty indicates if the underlying type1 sequence is empty.
func (c *Type1CharString) isSequenceEmpty() bool { return len(c.type1Sequence) == 0 }

// lastSequenceEntry returns the last entry of the underlying type1 sequence, or
// nil if empty.
func (c *Type1CharString) lastSequenceEntry() any {
	if len(c.type1Sequence) != 0 {
		return c.type1Sequence[len(c.type1Sequence)-1]
	}
	return nil
}

// String renders the sequence, as Java's toString does.
func (c *Type1CharString) String() string {
	s := sequenceString(c.type1Sequence)
	s = strings.ReplaceAll(s, "|", "\n")
	return strings.ReplaceAll(s, ",", " ")
}
