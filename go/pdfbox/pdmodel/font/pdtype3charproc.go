package font

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// PDType3CharProc is the content stream that draws one glyph of a Type 3 font.
//
// Port of org.apache.pdfbox.pdmodel.font.PDType3CharProc.
//
// Java has it implement PDContentStream, whose getResources returns a
// PDResources; that type lives in pdmodel, which imports this package. The port
// hands out the resource dictionary instead and lets pdmodel wrap it, which is
// how the cycle Java does not have is broken. See migration/STATUS.md.
type PDType3CharProc struct {
	font       *PDType3Font
	charStream *cos.Stream
}

var _ common.COSObjectable = (*PDType3CharProc)(nil)

// NewPDType3CharProc returns the char proc the given stream holds.
func NewPDType3CharProc(font *PDType3Font, charStream *cos.Stream) *PDType3CharProc {
	return &PDType3CharProc{font: font, charStream: charStream}
}

// COSObject returns the stream behind the char proc.
//
// Java declares it COSStream, narrowing COSObjectable.getCOSObject; Go cannot
// narrow a method in an interface, so the typed form is Stream and COSObject
// satisfies common.COSObjectable.
func (c *PDType3CharProc) COSObject() cos.Base { return c.charStream }

// Stream returns the stream behind the char proc, typed.
func (c *PDType3CharProc) Stream() *cos.Stream { return c.charStream }

// Font returns the font the glyph belongs to.
func (c *PDType3CharProc) Font() *PDType3Font { return c.font }

// ContentStream returns the stream behind the char proc, wrapped.
func (c *PDType3CharProc) ContentStream() *common.PDStream {
	return common.NewPDStream(c.charStream)
}

// ContentsForRandomAccess returns the contents of the char proc.
func (c *PDType3CharProc) ContentsForRandomAccess() (pdfio.RandomAccessRead, error) {
	return c.charStream.CreateView()
}

// ResourcesDictionary returns the resource dictionary the glyph is drawn
// against, which is the font's unless the char proc carries one of its own.
func (c *PDType3CharProc) ResourcesDictionary() *cos.Dictionary {
	if c.charStream.ContainsKey(cos.Resources) {
		// PDFBOX-5294
		// Using resources dictionary found in charproc entry.
		// This should have been in the font or in the page dictionary.
		return c.charStream.GetCOSDictionary(cos.Resources)
	}
	return c.font.ResourcesDictionary()
}

// BBox returns the box the glyph is drawn in, which is the font's.
func (c *PDType3CharProc) BBox() *common.PDRectangle { return c.font.FontBBox() }

// GlyphBBox returns the box the glyph declares for itself with its d1
// operator, or nil where it declares none.
func (c *PDType3CharProc) GlyphBBox() (*common.PDRectangle, error) {
	var arguments []cos.Base
	parser, err := c.parser()
	if err != nil {
		return nil, err
	}
	token, err := parser.ParseNextToken()
	if err != nil {
		return nil, err
	}
	for token != nil {
		if op, ok := token.(*operator.Operator); ok {
			if op.Name() == "d1" && len(arguments) == 6 {
				for i := 0; i < 6; i++ {
					if _, ok := arguments[i].(cos.Number); !ok {
						return nil, nil
					}
				}
				x := arguments[2].(cos.Number).FloatValue()
				y := arguments[3].(cos.Number).FloatValue()
				return common.NewPDRectangleOf(x, y,
					arguments[4].(cos.Number).FloatValue()-x,
					arguments[5].(cos.Number).FloatValue()-y), nil
			}
			return nil, nil
		}
		arguments = append(arguments, token.(cos.Base))
		token, err = parser.ParseNextToken()
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// Matrix returns the transform from glyph space to text space, which is the
// font's.
func (c *PDType3CharProc) Matrix() *util.Matrix { return c.font.FontMatrix() }

// Width returns the width the glyph declares for itself with its d0 or d1
// operator.
func (c *PDType3CharProc) Width() (float32, error) {
	var arguments []cos.Base
	parser, err := c.parser()
	if err != nil {
		return 0, err
	}
	token, err := parser.ParseNextToken()
	if err != nil {
		return 0, err
	}
	for token != nil {
		if op, ok := token.(*operator.Operator); ok {
			return parseWidth(op, arguments)
		}
		arguments = append(arguments, token.(cos.Base))
		token, err = parser.ParseNextToken()
		if err != nil {
			return 0, err
		}
	}
	return 0, fmt.Errorf("font: Unexpected end of stream")
}

// parseWidth reads the width out of the operator that opens the glyph.
func parseWidth(op *operator.Operator, arguments []cos.Base) (float32, error) {
	if op.Name() == "d0" || op.Name() == "d1" {
		obj := arguments[0]
		if number, ok := obj.(cos.Number); ok {
			return number.FloatValue(), nil
		}
		return 0, fmt.Errorf("font: Unexpected argument type: %T", obj)
	}
	return 0, fmt.Errorf("font: First operator must be d0 or d1")
}

// parser returns a token parser over the contents of the char proc.
func (c *PDType3CharProc) parser() (*pdfparser.StreamTokenParser, error) {
	contents, err := c.ContentsForRandomAccess()
	if err != nil {
		return nil, err
	}
	return pdfparser.NewStreamTokenParserSource(contents)
}
