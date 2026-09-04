package pdfparser

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// XrefEntries is a cross-reference table: where each object of a file is.
//
// Java keys one on COSObjectKey, whose equals and hashCode compare only the
// packed object number and generation — the stream index is left out. A Go map
// on the key struct would compare the stream index too and split entries Java
// merges, and a map on the pointer would not merge them at all, so the key here
// is the packed value and the object key travels alongside it.
type XrefEntries map[int64]XrefEntry

// XrefEntry is one entry of a cross-reference table.
type XrefEntry struct {
	Key *cos.ObjectKey
	// Offset is where the object is, or the negated object number of the
	// object stream holding it.
	Offset int64
}

// NewXrefEntries returns an empty table.
func NewXrefEntries() XrefEntries { return XrefEntries{} }

// Get returns where the given object is. The second result is false where the
// table does not mention it.
func (e XrefEntries) Get(key *cos.ObjectKey) (int64, bool) {
	if key == nil {
		return 0, false
	}
	entry, ok := e[key.InternalHash()]
	return entry.Offset, ok
}

// Put records where an object is.
func (e XrefEntries) Put(key *cos.ObjectKey, offset int64) {
	if key == nil {
		return
	}
	e[key.InternalHash()] = XrefEntry{Key: key, Offset: offset}
}

// PutIfAbsent records where an object is, leaving an entry already there alone.
func (e XrefEntries) PutIfAbsent(key *cos.ObjectKey, offset int64) {
	if key == nil {
		return
	}
	if _, ok := e[key.InternalHash()]; ok {
		return
	}
	e[key.InternalHash()] = XrefEntry{Key: key, Offset: offset}
}

// Delete drops an entry.
func (e XrefEntries) Delete(key *cos.ObjectKey) {
	if key == nil {
		return
	}
	delete(e, key.InternalHash())
}

// PutAll copies every entry of the other table into this one.
func (e XrefEntries) PutAll(other XrefEntries) {
	for hash, entry := range other {
		e[hash] = entry
	}
}

// Clear empties the table.
func (e XrefEntries) Clear() { clear(e) }

// ToKeyed returns the table in the shape cos.Document.AddXRefTable takes.
func (e XrefEntries) ToKeyed() map[*cos.ObjectKey]int64 {
	out := make(map[*cos.ObjectKey]int64, len(e))
	for _, entry := range e {
		out[entry.Key] = entry.Offset
	}
	return out
}

// FromKeyed returns a table holding the entries of the given map.
func FromKeyed(keyed map[*cos.ObjectKey]int64) XrefEntries {
	out := make(XrefEntries, len(keyed))
	for key, offset := range keyed {
		out.Put(key, offset)
	}
	return out
}

// isDigitAt reports whether the byte under the cursor is an ASCII digit,
// leaving the cursor where it was.
//
// Port of the COSParser.isDigit() that the xref and brute force parsers call.
func isDigitAt(source pdfio.RandomAccessRead) (bool, error) {
	position, err := source.Position()
	if err != nil {
		return false, err
	}
	b := make([]byte, 1)
	n, _ := source.Read(b)
	if _, err := source.Seek(position, 0); err != nil {
		return false, err
	}
	if n < 1 {
		return false, nil
	}
	return IsDigit(int(b[0])), nil
}
