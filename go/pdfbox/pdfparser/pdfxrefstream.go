package pdfparser

import (
	"io"
	"slices"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser/xref"
)

// XRefStream builds the cross-reference stream of a written document.
//
// Port of org.apache.pdfbox.pdfparser.PDFXRefStream.
type XRefStream struct {
	streamData []xref.Entry

	// objectNumbers is Java's TreeSet<Long>: the numbers already added, which
	// getIndexEntry walks in order.
	objectNumbers map[int64]bool

	stream *cos.Stream

	size int64
}

// NewXRefStream creates a fresh XRef stream like for a fresh file or an
// incremental update, using the given Document to create a new Stream.
func NewXRefStream(cosDocument *cos.Document) *XRefStream {
	return &XRefStream{
		objectNumbers: map[int64]bool{},
		stream:        cosDocument.CreateStream(),
		size:          -1,
	}
}

// Stream returns the stream of the XRef.
func (x *XRefStream) Stream() (*cos.Stream, error) {
	x.stream.SetItem(cos.Type, cos.XRef)
	if x.size == -1 {
		// Java throws IllegalArgumentException, which is unchecked.
		panic("size is not set in xrefstream")
	}
	x.stream.SetLong(cos.Size, x.size)

	indexEntry := x.indexEntry()
	indexAsArray := cos.NewArray()
	for _, i := range indexEntry {
		indexAsArray.Add(cos.GetInteger(i))
	}
	x.stream.SetItem(cos.Index, indexAsArray)

	wEntry := x.wEntry()
	wAsArray := cos.NewArray()
	for _, j := range wEntry {
		wAsArray.Add(cos.GetInteger(int64(j)))
	}
	x.stream.SetItem(cos.W, wAsArray)

	outputStream, err := x.stream.CreateWriterWithFilters(cos.FlateDecode)
	if err != nil {
		return nil, err
	}
	if err := x.writeStreamData(outputStream, wEntry); err != nil {
		outputStream.Close()
		return nil, err
	}
	if err := outputStream.Close(); err != nil {
		return nil, err
	}

	for _, cosName := range x.stream.KeySet() {
		// "Other cross-reference stream entries not listed in Table 17 may be indirect; in fact,
		// some (such as Root in Table 15) shall be indirect."
		if cosName == cos.Root || cosName == cos.Info || cosName == cos.Prev {
			continue
		}
		// this one too, because it has already been written in COSWriter.doWriteBody()
		if cosName == cos.Encrypt {
			continue
		}
		dictionaryObject := x.stream.GetDictionaryObject(cosName)
		dictionaryObject.SetDirect(true)
	}
	return x.stream, nil
}

// AddTrailerInfo copies all trailer information to this file.
func (x *XRefStream) AddTrailerInfo(trailerDict *cos.Dictionary) {
	for _, key := range trailerDict.KeySet() {
		if key == cos.Info || key == cos.Root || key == cos.Encrypt ||
			key == cos.ID || key == cos.Prev {
			x.stream.SetItem(key, trailerDict.GetItem(key))
		}
	}
}

// AddEntry adds a new entry to the XRef stream.
func (x *XRefStream) AddEntry(entry xref.Entry) {
	number := entry.ReferencedKey().Number()
	if x.objectNumbers[number] {
		return
	}
	x.objectNumbers[number] = true
	x.streamData = append(x.streamData, entry)
}

// wEntry determines the minimal length required for all the lengths.
func (x *XRefStream) wEntry() [3]int {
	var wMax [3]int64
	for _, entry := range x.streamData {
		wMax[0] = max(wMax[0], entry.FirstColumnValue())
		wMax[1] = max(wMax[1], entry.SecondColumnValue())
		wMax[2] = max(wMax[2], entry.ThirdColumnValue())
	}
	// find the max bytes needed to display that column
	var w [3]int
	for i := range w {
		for wMax[i] > 0 {
			w[i]++
			wMax[i] >>= 8
		}
	}
	return w
}

// SetSize sets the size of the XRef stream.
func (x *XRefStream) SetSize(streamSize int64) {
	x.size = streamSize
}

// indexEntry returns the /Index array: pairs of a first object number and a
// count.
//
// Java walks a TreeSet, so the numbers come out sorted, and object number 0 is
// always in it.
func (x *XRefStream) indexEntry() []int64 {
	var linkedList []int64
	var first, length int64
	haveFirst := false

	objNumbers := make([]int64, 0, len(x.objectNumbers)+1)
	// add object number 0 to the set
	objNumbers = append(objNumbers, 0)
	for number := range x.objectNumbers {
		if number != 0 {
			objNumbers = append(objNumbers, number)
		}
	}
	slices.Sort(objNumbers)

	for _, objNumber := range objNumbers {
		if !haveFirst {
			first = objNumber
			length = 1
			haveFirst = true
		}
		if first+length == objNumber {
			length++
		}
		if first+length < objNumber {
			linkedList = append(linkedList, first, length)
			first = objNumber
			length = 1
		}
	}
	linkedList = append(linkedList, first, length)

	return linkedList
}

// writeNumber writes number as the given count of big-endian bytes.
func writeNumber(os io.Writer, number int64, bytes int) error {
	buffer := make([]byte, bytes)
	for i := 0; i < bytes; i++ {
		buffer[i] = byte(number & 0xff)
		number >>= 8
	}
	for i := 0; i < bytes; i++ {
		if _, err := os.Write(buffer[bytes-i-1 : bytes-i]); err != nil {
			return err
		}
	}
	return nil
}

func (x *XRefStream) writeStreamData(os io.Writer, w [3]int) error {
	slices.SortStableFunc(x.streamData, xref.Compare)
	nullEntry := xref.NullEntry
	if err := writeNumber(os, nullEntry.FirstColumnValue(), w[0]); err != nil {
		return err
	}
	if err := writeNumber(os, nullEntry.SecondColumnValue(), w[1]); err != nil {
		return err
	}
	if err := writeNumber(os, nullEntry.ThirdColumnValue(), w[2]); err != nil {
		return err
	}
	// iterate over all streamData and write it in the required format
	for _, entry := range x.streamData {
		if err := writeNumber(os, entry.FirstColumnValue(), w[0]); err != nil {
			return err
		}
		if err := writeNumber(os, entry.SecondColumnValue(), w[1]); err != nil {
			return err
		}
		if err := writeNumber(os, entry.ThirdColumnValue(), w[2]); err != nil {
			return err
		}
	}
	return nil
}
