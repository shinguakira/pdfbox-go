package compress

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// ObjectStream represents an object stream, that compresses a number of
// cos.Objects in a stream. It may be added to the top level container of a
// written PDF document in place of the compressed objects. The document's
// cross-reference stream must be adapted accordingly.
//
// Port of org.apache.pdfbox.pdfwriter.compress.COSWriterObjectStream.
type ObjectStream struct {
	compressionPool *CompressionPool
	preparedKeys    []*cos.ObjectKey
	preparedObjects []cos.Base
}

// NewObjectStream creates an object stream for compressible objects from the
// given CompressionPool. The objects must first be prepared for this object
// stream, by adding them via calling PrepareStreamObject, and are written to
// the cos.Stream when WriteObjectsToStream is called.
func NewObjectStream(compressionPool *CompressionPool) *ObjectStream {
	return &ObjectStream{compressionPool: compressionPool}
}

// PrepareStreamObject prepares the given object to be written to this object
// stream, using the given ObjectKey as its ID for indirect references.
func (s *ObjectStream) PrepareStreamObject(key *cos.ObjectKey, object cos.Base) {
	if key != nil && object != nil {
		s.preparedKeys = append(s.preparedKeys, key)
		if reference, ok := object.(*cos.Object); ok {
			s.preparedObjects = append(s.preparedObjects, reference.Object())
		} else {
			s.preparedObjects = append(s.preparedObjects, object)
		}
	}
}

// PreparedKeys returns all ObjectKeys that shall be added to the object stream
// when WriteObjectsToStream is called.
//
// Java wraps the list with Collections.unmodifiableList; the port returns a
// copy, which is the convention for the same in migration/conventions.
func (s *ObjectStream) PreparedKeys() []*cos.ObjectKey {
	return slices.Clone(s.preparedKeys)
}

// WriteObjectsToStream writes all prepared objects to the given stream and
// returns it.
func (s *ObjectStream) WriteObjectsToStream(stream *cos.Stream) (*cos.Stream, error) {
	objectCount := len(s.preparedKeys)
	stream.SetItem(cos.Type, cos.ObjStm)
	stream.SetInt(cos.N, objectCount)
	// Prepare the compressible objects for writing.
	objectNumbers := make([]int64, 0, objectCount)
	objectsBuffer := make([]*bytes.Buffer, 0, objectCount)
	for i := 0; i < objectCount; i++ {
		partialOutput := &bytes.Buffer{}
		objectNumbers = append(objectNumbers, s.preparedKeys[i].Number())
		base := s.preparedObjects[i]
		if err := s.writeObject(partialOutput, base, true); err != nil {
			return nil, err
		}
		objectsBuffer = append(objectsBuffer, partialOutput)
	}

	// Deduce the object stream byte offset map.
	var offsetsMap bytes.Buffer
	nextObjectOffset := 0
	for i := range objectNumbers {
		offsetsMap.WriteString(strconv.FormatInt(objectNumbers[i], 10))
		offsetsMap.Write(Space)
		offsetsMap.WriteString(strconv.Itoa(nextObjectOffset))
		offsetsMap.Write(Space)
		nextObjectOffset += objectsBuffer[i].Len()
	}
	offsetsMapBuffer := offsetsMap.Bytes()

	// Write Flate compressed object stream data.
	output, err := stream.CreateWriterWithFilters(cos.FlateDecode)
	if err != nil {
		return nil, err
	}
	if _, err := output.Write(offsetsMapBuffer); err != nil {
		output.Close()
		return nil, err
	}
	stream.SetInt(cos.First, len(offsetsMapBuffer))
	for _, rawObject := range objectsBuffer {
		if _, err := output.Write(rawObject.Bytes()); err != nil {
			output.Close()
			return nil, err
		}
	}
	if err := output.Close(); err != nil {
		return nil, err
	}
	return stream, nil
}

// writeObject prepares and writes COS data to the object stream by selecting
// appropriate specialized methods for the content.
//
// topLevel is true if the currently written object is a top level entry of this
// object stream.
func (s *ObjectStream) writeObject(output io.Writer, object cos.Base, topLevel bool) error {
	if object == nil {
		return nil
	}
	var base cos.Base
	if reference, isReference := object.(*cos.Object); isReference {
		if !topLevel {
			actualKey := object.Key()
			if actualKey != nil {
				return writeObjectReference(output, actualKey)
			}
		}
		base = reference.Object()
		if base == nil {
			slog.Debug("compress: can't dereference indirect object, writing COSNull instead",
				"object", object)
			return writeCOSNull(output)
		}
		if _, isNested := base.(*cos.Object); isNested {
			slog.Error("compress: COSObject references another COSObject?!", "object", object)
		}
	} else {
		base = object
	}
	if !topLevel && s.compressionPool.Contains(base) {
		key := s.compressionPool.Key(base)
		if key == nil {
			return fmt.Errorf("Error: Adding unknown object reference to object stream:%v", object)
		}
		return writeObjectReference(output, key)
	}
	switch value := base.(type) {
	case *cos.StringObj:
		return writeCOSString(output, value)
	case *cos.Float:
		return writeWithTrailingSpace(output, value.WritePDF)
	case *cos.Integer:
		return writeWithTrailingSpace(output, value.WritePDF)
	case *cos.Boolean:
		return writeWithTrailingSpace(output, value.WritePDF)
	case *cos.Name:
		return writeWithTrailingSpace(output, value.WritePDF)
	case *cos.Array:
		return s.writeCOSArray(output, value)
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so it takes the same branch.
		return s.writeCOSDictionary(output, &value.Dictionary)
	case *cos.Dictionary:
		return s.writeCOSDictionary(output, value)
	case *cos.Null:
		return writeCOSNull(output)
	}
	return fmt.Errorf("Error: Unknown type in object stream:%v", object)
}

// writeCOSString writes the given cos.StringObj to the given stream.
func writeCOSString(output io.Writer, cosString *cos.StringObj) error {
	if err := WriteString(cosString, output); err != nil {
		return err
	}
	_, err := output.Write(Space)
	return err
}

// writeWithTrailingSpace is the shape the COSFloat, COSInteger, COSBoolean and
// COSName cases share: writePDF then a space.
func writeWithTrailingSpace(output io.Writer, writePDF func(io.Writer) error) error {
	if err := writePDF(output); err != nil {
		return err
	}
	_, err := output.Write(Space)
	return err
}

// writeCOSArray writes the given cos.Array to the given stream.
func (s *ObjectStream) writeCOSArray(output io.Writer, cosArray *cos.Array) error {
	if _, err := output.Write(ArrayOpen); err != nil {
		return err
	}
	for i := 0; i < cosArray.Size(); i++ {
		value := cosArray.Get(i)
		if value == nil {
			if err := writeCOSNull(output); err != nil {
				return err
			}
		} else if err := s.writeObject(output, value, false); err != nil {
			return err
		}
	}
	if _, err := output.Write(ArrayClose); err != nil {
		return err
	}
	_, err := output.Write(Space)
	return err
}

// writeCOSDictionary writes the given cos.Dictionary to the given stream.
func (s *ObjectStream) writeCOSDictionary(output io.Writer, cosDictionary *cos.Dictionary) error {
	if _, err := output.Write(DictOpen); err != nil {
		return err
	}
	for _, key := range cosDictionary.KeySet() {
		value := cosDictionary.GetItem(key)
		if value == nil {
			continue
		}
		// PDFBOX-5927: topLevel true to avoid having a dictionary key as an indirect object
		// if it already exists as such
		if err := s.writeObject(output, key, true); err != nil {
			return err
		}
		if err := s.writeObject(output, value, false); err != nil {
			return err
		}
	}
	if _, err := output.Write(DictClose); err != nil {
		return err
	}
	_, err := output.Write(Space)
	return err
}

// writeObjectReference writes the given ObjectKey to the given stream.
func writeObjectReference(output io.Writer, indirectReference *cos.ObjectKey) error {
	parts := [][]byte{
		[]byte(strconv.FormatInt(indirectReference.Number(), 10)), Space,
		[]byte(strconv.Itoa(indirectReference.Generation())), Space,
		Reference, Space,
	}
	for _, part := range parts {
		if _, err := output.Write(part); err != nil {
			return err
		}
	}
	return nil
}

// writeCOSNull writes cos.Null to the given stream.
func writeCOSNull(output io.Writer) error {
	if _, err := output.Write(cos.NullBytes); err != nil {
		return err
	}
	_, err := output.Write(Space)
	return err
}
