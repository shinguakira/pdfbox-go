package pdfparser

import (
	"fmt"
	"sort"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// ObjectStreamParser reads the objects packed into an object stream.
//
// Port of org.apache.pdfbox.pdfparser.PDFObjectStreamParser. Java extends
// COSParser; the port embeds ObjectParser, which is the half it uses.
type ObjectStreamParser struct {
	*ObjectParser

	numberOfObjects int
	firstObject     int
}

// NewObjectStreamParser returns a parser over the given object stream.
func NewObjectStreamParser(stream *cos.Stream, document *cos.Document) (*ObjectStreamParser, error) {
	source, err := stream.CreateView()
	if err != nil {
		return nil, err
	}
	p := &ObjectStreamParser{ObjectParser: NewObjectParser(source, document)}

	// get mandatory number of objects
	p.numberOfObjects = stream.GetIntDefault(cos.N, -1)
	if p.numberOfObjects == -1 {
		return nil, fmt.Errorf("pdfparser: /N entry missing in object stream")
	}
	if p.numberOfObjects < 0 {
		return nil, fmt.Errorf("pdfparser: Illegal /N entry in object stream: %d", p.numberOfObjects)
	}

	// get mandatory stream offset of the first object
	p.firstObject = stream.GetIntDefault(cos.First, -1)
	if p.firstObject == -1 {
		return nil, fmt.Errorf("pdfparser: /First entry missing in object stream")
	}
	if p.firstObject < 0 {
		return nil, fmt.Errorf("pdfparser: Illegal /First entry in object stream: %d", p.firstObject)
	}
	return p, nil
}

// ParseObject reads one object out of the stream, by object number.
func (p *ObjectStreamParser) ParseObject(objectNumber int64) (cos.Base, error) {
	defer p.Source().Close()

	objectNumbers, err := p.privateReadObjectNumbers()
	if err != nil {
		return nil, err
	}
	objectOffset, ok := objectNumbers[objectNumber]
	if !ok {
		return nil, nil
	}

	// jump to the offset of the first object
	currentPosition, err := p.Source().Position()
	if err != nil {
		return nil, err
	}
	if p.firstObject > 0 && currentPosition < int64(p.firstObject) {
		if err := p.skip(int64(p.firstObject) - currentPosition); err != nil {
			return nil, err
		}
	}
	// jump to the offset of the object to be parsed
	if err := p.skip(int64(objectOffset)); err != nil {
		return nil, err
	}
	streamObject, err := p.ParseDirObject()
	if err != nil {
		return nil, err
	}
	if streamObject != nil {
		streamObject.SetDirect(false)
	}
	return streamObject, nil
}

// ParseAllObjects reads every object of the stream, keyed the way a
// cross-reference table keys them.
func (p *ObjectStreamParser) ParseAllObjects() (map[int64]cos.Base, error) {
	defer p.Source().Close()

	allObjects := map[int64]cos.Base{}
	objectOffsets, offsetOrder, err := p.privateReadObjectOffsets()
	if err != nil {
		return nil, err
	}

	// count the number of object numbers eliminating double entries
	distinct := map[int64]bool{}
	for _, objectNumber := range objectOffsets {
		distinct[objectNumber] = true
	}
	// the usage of the index should be restricted to cases where more than one
	// object use the same object number.
	// there are malformed pdfs in the wild which would lead to false results if
	// pdfbox always relies on the index if available. In most cases the object
	// number is sufficient to choose the correct object
	indexNeeded := len(objectOffsets) > len(distinct)

	currentPosition, err := p.Source().Position()
	if err != nil {
		return nil, err
	}
	if p.firstObject > 0 && currentPosition < int64(p.firstObject) {
		if err := p.skip(int64(p.firstObject) - currentPosition); err != nil {
			return nil, err
		}
	}

	index := 0
	for _, offset := range offsetOrder {
		objectNumber := objectOffsets[offset]
		objectKey, err := p.ObjectKey(objectNumber, 0)
		if err != nil {
			return nil, err
		}
		// skip object if the index doesn't match
		if indexNeeded && objectKey.StreamIndex() > -1 && objectKey.StreamIndex() != index {
			index++
			continue
		}
		finalPosition := int64(p.firstObject + offset)
		currentPosition, err := p.Source().Position()
		if err != nil {
			return nil, err
		}
		if finalPosition > 0 && currentPosition < finalPosition {
			// jump to the offset of the object to be parsed
			if err := p.skip(finalPosition - currentPosition); err != nil {
				return nil, err
			}
		}
		streamObject, err := p.ParseDirObject()
		if err != nil {
			return nil, err
		}
		if streamObject != nil {
			streamObject.SetDirect(false)
		}
		allObjects[objectKey.InternalHash()] = streamObject
		index++
	}
	return allObjects, nil
}

// privateReadObjectNumbers reads the object number and offset of each entry,
// keyed by object number.
func (p *ObjectStreamParser) privateReadObjectNumbers() (map[int64]int, error) {
	// don't initialize map using numberOfObjects as there might by less object
	// numbers than expected
	objectNumbers := map[int64]int{}
	position, err := p.Source().Position()
	if err != nil {
		return nil, err
	}
	firstObjectPosition := position + int64(p.firstObject) - 1
	for i := 0; i < p.numberOfObjects; i++ {
		// don't read beyond the part of the stream reserved for the object
		// numbers
		position, err := p.Source().Position()
		if err != nil {
			return nil, err
		}
		if position >= firstObjectPosition {
			break
		}
		objectNumber, err := p.ReadObjectNumber()
		if err != nil {
			return nil, err
		}
		offset, err := p.ReadLong()
		if err != nil {
			return nil, err
		}
		objectNumbers[objectNumber] = int(offset)
	}
	return objectNumbers, nil
}

// privateReadObjectOffsets reads the entries keyed by offset, together with the
// offsets in ascending order.
//
// According to the pdf spec the offsets shall be sorted ascending but we can't
// rely on that, so that we have to sort the offsets as the sequential parsers
// relies on it, see PDFBOX-4927. Java uses a TreeMap; Go has no sorted map, so
// the order travels alongside.
func (p *ObjectStreamParser) privateReadObjectOffsets() (map[int]int64, []int, error) {
	objectOffsets := map[int]int64{}
	position, err := p.Source().Position()
	if err != nil {
		return nil, nil, err
	}
	firstObjectPosition := position + int64(p.firstObject) - 1
	for i := 0; i < p.numberOfObjects; i++ {
		// don't read beyond the part of the stream reserved for the object
		// numbers
		position, err := p.Source().Position()
		if err != nil {
			return nil, nil, err
		}
		if position >= firstObjectPosition {
			break
		}
		objectNumber, err := p.ReadObjectNumber()
		if err != nil {
			return nil, nil, err
		}
		offset, err := p.ReadLong()
		if err != nil {
			return nil, nil, err
		}
		objectOffsets[int(offset)] = objectNumber
	}
	order := make([]int, 0, len(objectOffsets))
	for offset := range objectOffsets {
		order = append(order, offset)
	}
	sort.Ints(order)
	return objectOffsets, order, nil
}

// ReadObjectNumbers reads the object number and offset of each entry.
func (p *ObjectStreamParser) ReadObjectNumbers() (map[int64]int, error) {
	defer p.Source().Close()
	return p.privateReadObjectNumbers()
}

// skip moves the cursor forward by n bytes.
func (p *ObjectStreamParser) skip(n int64) error {
	position, err := p.Source().Position()
	if err != nil {
		return err
	}
	_, err = p.Source().Seek(position+n, 0)
	return err
}
