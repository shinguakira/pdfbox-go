package pdfparser

import (
	"errors"
	"fmt"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// XrefStreamParser reads a cross-reference stream: the compact form of a
// cross-reference table.
//
// Port of org.apache.pdfbox.pdfparser.PDFXrefStreamParser.
type XrefStreamParser struct {
	w             [3]int
	objectNumbers *objectNumbers
	source        pdfio.RandomAccessRead
}

// NewXrefStreamParser returns a parser over the given cross-reference stream.
func NewXrefStreamParser(stream *cos.Stream) (*XrefStreamParser, error) {
	source, err := stream.CreateView()
	if err != nil {
		return nil, err
	}
	p := &XrefStreamParser{source: source}
	if err := p.initParserValues(stream); err != nil {
		p.close()
		return nil, err
	}
	return p, nil
}

// initParserValues reads the /W and /Index arrays that say how the stream is
// laid out.
func (p *XrefStreamParser) initParserValues(stream *cos.Stream) error {
	wArray := stream.GetCOSArray(cos.W)
	if wArray == nil {
		return fmt.Errorf("pdfparser: /W array is missing in Xref stream")
	}
	if wArray.Size() != 3 {
		return fmt.Errorf("pdfparser: Wrong number of values for /W array in XRef: %v", p.w)
	}
	for i := 0; i < 3; i++ {
		p.w[i] = wArray.GetIntDefault(i, 0)
	}
	if p.w[0] < 0 || p.w[1] < 0 || p.w[2] < 0 {
		return fmt.Errorf("pdfparser: Incorrect /W array in XRef: %v", p.w)
	}
	if total := p.w[0] + p.w[1] + p.w[2]; total > 20 || total == 0 {
		// PDFBOX-6037/PDFBOX-6229
		return fmt.Errorf("pdfparser: Incorrect /W array in XRef: %v", p.w)
	}

	indexArray := stream.GetCOSArray(cos.Index)
	if indexArray == nil {
		// If /Index doesn't exist, we will use the default values.
		indexArray = cos.NewArray()
		indexArray.Add(cos.IntegerZero)
		indexArray.Add(cos.GetInteger(int64(stream.GetIntDefault(cos.Size, 0))))
	}
	if indexArray.Size() == 0 || indexArray.Size()%2 == 1 {
		return fmt.Errorf("pdfparser: Wrong number of values for /Index array in XRef: %v", p.w)
	}

	// create an Iterator for all object numbers using the index array
	numbers, err := newObjectNumbers(indexArray)
	if err != nil {
		return err
	}
	p.objectNumbers = numbers
	return nil
}

func (p *XrefStreamParser) close() error {
	p.objectNumbers = nil
	if p.source != nil {
		return p.source.Close()
	}
	return nil
}

// Parse reads every entry of the stream into the resolver.
func (p *XrefStreamParser) Parse(resolver *XrefTrailerResolver) error {
	defer p.close()

	currLine := make([]byte, p.w[0]+p.w[1]+p.w[2])
	for {
		isEOF, err := p.source.IsEOF()
		if err != nil {
			return err
		}
		if isEOF || !p.objectNumbers.hasNext() {
			break
		}
		if err := p.readNextValue(currLine); err != nil {
			return err
		}
		// get the current objID
		objID := p.objectNumbers.next()

		// default value is 1 if w[0] == 0, otherwise parse first field
		typ := 1
		if p.w[0] != 0 {
			typ = int(parseXrefValue(currLine, 0, p.w[0]))
		}
		// Skip free objects (type 0) and invalid types
		if typ == 0 {
			continue
		}
		// second field holds the offset (type 1) or the object stream number
		// (type 2)
		offset := parseXrefValue(currLine, p.w[0], p.w[1])
		// third filed may hold the generation number (type1) or the index
		// within a object stream (type2)
		thirdValue := int(parseXrefValue(currLine, p.w[0]+p.w[1], p.w[2]))

		if typ == 1 {
			// third field holds the generation number for type 1 entries
			key, err := cos.NewObjectKey(objID, thirdValue)
			if err != nil {
				return err
			}
			resolver.SetXRef(key, offset)
		} else {
			// For XRef aware parsers we have to know which objects contain
			// object streams. We will store this information in normal xref
			// mapping table but add object stream number with minus sign in
			// order to distinguish from file offsets
			key, err := cos.NewObjectKeyInStream(objID, 0, thirdValue)
			if err != nil {
				return err
			}
			resolver.SetXRef(key, -offset)
		}
	}
	return nil
}

// readNextValue fills value with the next entry of the stream.
func (p *XrefStreamParser) readNextValue(value []byte) error {
	remainingBytes := len(value)
	for remainingBytes > 0 {
		amountRead, err := p.source.Read(value[len(value)-remainingBytes:])
		// Java's read throws for a failure and returns -1 at the end of the
		// data; only the second ends the loop.
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if amountRead <= 0 {
			// Java's loop ends when read returns 0 or less, leaving the rest of
			// the buffer as it was.
			return nil
		}
		remainingBytes -= amountRead
	}
	return nil
}

// parseXrefValue reads one big-endian field of an entry.
func parseXrefValue(data []byte, start, length int) int64 {
	var value int64
	for i := 0; i < length; i++ {
		value += int64(data[i+start]&0x00ff) << ((length - i - 1) * 8)
	}
	return value
}

// objectNumbers walks the object numbers the /Index array names, as a run of
// ranges.
//
// Port of the ObjectNumbers iterator inside PDFXrefStreamParser.
type objectNumbers struct {
	start         []int64
	end           []int64
	currentRange  int
	currentEnd    int64
	currentNumber int64
}

func newObjectNumbers(indexArray *cos.Array) (*objectNumbers, error) {
	n := &objectNumbers{
		start: make([]int64, indexArray.Size()/2),
	}
	n.end = make([]int64, len(n.start))

	counter := 0
	for i := 0; i < indexArray.Size(); i++ {
		startBase, ok := indexArray.GetObject(i).(*cos.Integer)
		if !ok {
			return nil, fmt.Errorf("pdfparser: Xref stream must have integer in /Index array")
		}
		startValue := startBase.LongValue()
		i++
		if i >= indexArray.Size() {
			break
		}
		sizeBase, ok := indexArray.GetObject(i).(*cos.Integer)
		if !ok {
			return nil, fmt.Errorf("pdfparser: Xref stream must have integer in /Index array")
		}
		sizeValue := sizeBase.LongValue()
		n.start[counter] = startValue
		n.end[counter] = startValue + sizeValue
		counter++
	}
	n.currentNumber = n.start[0]
	n.currentEnd = n.end[0]
	return n, nil
}

func (n *objectNumbers) hasNext() bool {
	if len(n.start) == 1 {
		return n.currentNumber < n.currentEnd
	}
	return n.currentRange < len(n.start)-1 || n.currentNumber < n.currentEnd
}

// next returns the next object number. Java throws NoSuchElementException when
// there is none; every caller here checks hasNext first.
func (n *objectNumbers) next() int64 {
	if n.currentNumber < n.currentEnd {
		value := n.currentNumber
		n.currentNumber++
		return value
	}
	if n.currentRange >= len(n.start)-1 {
		panic("pdfparser: no such element")
	}
	n.currentRange++
	n.currentNumber = n.start[n.currentRange]
	n.currentEnd = n.end[n.currentRange]
	value := n.currentNumber
	n.currentNumber++
	return value
}
