package pdfwriter

import (
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The tokens a content stream is built out of.
//
// Port of the public byte array constants of ContentStreamWriter. They shadow
// nothing: COSWriter's SPACE is the same byte, but Java declares them
// separately and so does the port.
var (
	// ContentSpace is a space.
	ContentSpace = []byte{32}
	// ContentEOL is the end of line token.
	ContentEOL = []byte{0x0A}
)

// ContentStreamWriter writes a content stream out of the tokens a parser read
// from one.
//
// Port of org.apache.pdfbox.pdfwriter.ContentStreamWriter.
type ContentStreamWriter struct {
	output io.Writer
}

// NewContentStreamWriter returns a writer over the given stream.
func NewContentStreamWriter(out io.Writer) *ContentStreamWriter {
	return &ContentStreamWriter{output: out}
}

// WriteToken writes a single COS token.
//
// Port of writeToken(COSBase).
func (w *ContentStreamWriter) WriteToken(base cos.Base) error {
	return w.writeCOSObject(base)
}

// WriteOperatorToken writes a single operator.
//
// Port of writeToken(Operator), which Go cannot overload.
func (w *ContentStreamWriter) WriteOperatorToken(op *operator.Operator) error {
	return w.writeOperator(op)
}

// WriteTokensVarargs writes the given tokens and a newline after them.
//
// Port of writeTokens(Object...). Each token must be a cos.Base or an
// *operator.Operator; anything else is an error, as it is in Java.
func (w *ContentStreamWriter) WriteTokensVarargs(tokens ...any) error {
	for _, token := range tokens {
		if err := w.writeObject(token); err != nil {
			return err
		}
	}
	_, err := w.output.Write([]byte{'\n'})
	return err
}

// WriteTokens writes the given tokens, without a newline after them.
//
// Port of writeTokens(List<?>).
func (w *ContentStreamWriter) WriteTokens(tokens []any) error {
	for _, token := range tokens {
		if err := w.writeObject(token); err != nil {
			return err
		}
	}
	return nil
}

func (w *ContentStreamWriter) writeObject(o any) error {
	switch value := o.(type) {
	case cos.Base:
		return w.writeCOSObject(value)
	case *operator.Operator:
		return w.writeOperator(value)
	}
	return fmt.Errorf("Error:Unknown type in content stream:%v", o)
}

func (w *ContentStreamWriter) writeOperator(op *operator.Operator) error {
	if op.Name() != operator.BeginInlineImage {
		if _, err := w.output.Write(operator.GetNameAsBytes(op.Name())); err != nil {
			return err
		}
		_, err := w.output.Write(ContentEOL)
		return err
	}

	if _, err := w.output.Write(
		operator.GetNameAsBytes(operator.BeginInlineImage)); err != nil {
		return err
	}
	if _, err := w.output.Write(ContentEOL); err != nil {
		return err
	}
	dic := op.ImageParameters()
	for _, key := range dic.KeySet() {
		value := dic.GetDictionaryObject(key)
		if err := key.WritePDF(w.output); err != nil {
			return err
		}
		if _, err := w.output.Write(ContentSpace); err != nil {
			return err
		}
		if err := w.writeObject(value); err != nil {
			return err
		}
		if _, err := w.output.Write(ContentEOL); err != nil {
			return err
		}
	}
	if _, err := w.output.Write(
		operator.GetNameAsBytes(operator.BeginInlineImageData)); err != nil {
		return err
	}
	if _, err := w.output.Write(ContentEOL); err != nil {
		return err
	}
	if _, err := w.output.Write(op.ImageData()); err != nil {
		return err
	}
	if _, err := w.output.Write(ContentEOL); err != nil {
		return err
	}
	if _, err := w.output.Write(
		operator.GetNameAsBytes(operator.EndInlineImage)); err != nil {
		return err
	}
	_, err := w.output.Write(ContentEOL)
	return err
}

func (w *ContentStreamWriter) writeCOSObject(o cos.Base) error {
	switch value := o.(type) {
	case *cos.StringObj:
		if err := WriteString(value, w.output); err != nil {
			return err
		}
	case *cos.Float:
		if err := value.WritePDF(w.output); err != nil {
			return err
		}
	case *cos.Integer:
		if err := value.WritePDF(w.output); err != nil {
			return err
		}
	case *cos.Boolean:
		if err := value.WritePDF(w.output); err != nil {
			return err
		}
	case *cos.Name:
		if err := value.WritePDF(w.output); err != nil {
			return err
		}
	case *cos.Array:
		if _, err := w.output.Write(ArrayOpen); err != nil {
			return err
		}
		for i := 0; i < value.Size(); i++ {
			if err := w.writeCOSObject(value.Get(i)); err != nil {
				return err
			}
		}
		if _, err := w.output.Write(ArrayClose); err != nil {
			return err
		}
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so it takes the same branch.
		if err := w.writeCOSDictionary(&value.Dictionary); err != nil {
			return err
		}
	case *cos.Dictionary:
		if err := w.writeCOSDictionary(value); err != nil {
			return err
		}
	case *cos.Null:
		if _, err := w.output.Write(cos.NullBytes); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Error:Unknown type in content stream:%v", o)
	}
	_, err := w.output.Write(ContentSpace)
	return err
}

func (w *ContentStreamWriter) writeCOSDictionary(obj *cos.Dictionary) error {
	if _, err := w.output.Write(DictOpen); err != nil {
		return err
	}
	for _, key := range obj.KeySet() {
		value := obj.GetItem(key)
		if value == nil {
			continue
		}
		if err := w.writeCOSObject(key); err != nil {
			return err
		}
		if err := w.writeCOSObject(value); err != nil {
			return err
		}
	}
	_, err := w.output.Write(DictClose)
	return err
}
