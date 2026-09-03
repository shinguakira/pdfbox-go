package pdfparser

import (
	"log/slog"
	"sort"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// XRefType is the form the cross-reference data takes.
//
// Port of the nested enum XrefTrailerResolver.XRefType.
type XRefType int

const (
	// XRefTypeNone means no cross-reference data has been resolved.
	XRefTypeNone XRefType = iota
	// XRefTypeTable is the classic "xref" table.
	XRefTypeTable
	// XRefTypeStream is a cross-reference stream.
	XRefTypeStream
)

// String returns the Java enum constant name.
func (t XRefType) String() string {
	switch t {
	case XRefTypeTable:
		return "TABLE"
	case XRefTypeStream:
		return "STREAM"
	}
	return "NONE"
}

// xrefTrailerObj is one cross-reference section and the trailer that follows
// it.
//
// Port of the private nested class XrefTrailerResolver.XrefTrailerObj. Its
// table is keyed by the packed object key, for the reason given on
// cos.Document.
type xrefTrailerObj struct {
	trailer   *cos.Dictionary
	xrefType  XRefType
	xrefTable map[int64]*xrefTableEntry
}

type xrefTableEntry struct {
	key    *cos.ObjectKey
	offset int64
}

func newXrefTrailerObj() *xrefTrailerObj {
	return &xrefTrailerObj{xrefTable: make(map[int64]*xrefTableEntry)}
}

// XrefTrailerResolver collects the cross-reference sections of a file and
// resolves them into one table.
//
// Port of org.apache.pdfbox.pdfparser.XrefTrailerResolver. A PDF can carry
// several sections chained by /Prev, from incremental updates; this walks that
// chain and merges them so that later sections win.
type XrefTrailerResolver struct {
	// bytePosToXref holds one section per byte offset.
	bytePosToXref map[int64]*xrefTrailerObj
	// bytePosOrder records the order sections were seen, so that the fallback
	// path is deterministic where Java iterates a HashMap.
	bytePosOrder []int64

	current  *xrefTrailerObj
	resolved *xrefTrailerObj
}

// NewXrefTrailerResolver returns an empty resolver.
func NewXrefTrailerResolver() *XrefTrailerResolver {
	return &XrefTrailerResolver{bytePosToXref: make(map[int64]*xrefTrailerObj)}
}

// NextXrefObj signals the start of a cross-reference section at a byte offset.
func (r *XrefTrailerResolver) NextXrefObj(startBytePos int64, typ XRefType) {
	if _, exists := r.bytePosToXref[startBytePos]; !exists {
		r.bytePosOrder = append(r.bytePosOrder, startBytePos)
	}
	obj := newXrefTrailerObj()
	obj.xrefType = typ
	r.bytePosToXref[startBytePos] = obj
	r.current = obj
}

// XrefType returns the type of the resolved cross-reference data.
func (r *XrefTrailerResolver) XrefType() XRefType {
	if r.resolved == nil {
		return XRefTypeNone
	}
	return r.resolved.xrefType
}

// SetXRef records where one object lives.
//
// PDFBOX-3506: an entry already present is not overwritten, so that in a hybrid
// file the entries of the table are not replaced by the obsolete ones the
// /XRefStm section repeats.
func (r *XrefTrailerResolver) SetXRef(key *cos.ObjectKey, offset int64) {
	if r.current == nil {
		// should not happen
		slog.Warn("pdfparser: cannot add an xref entry, no xref start was signalled",
			"object", key)
		return
	}
	if key == nil {
		return
	}
	hash := key.InternalHash()
	if _, exists := r.current.xrefTable[hash]; exists {
		return
	}
	r.current.xrefTable[hash] = &xrefTableEntry{key: key, offset: offset}
}

// SetTrailer records the trailer of the current section.
func (r *XrefTrailerResolver) SetTrailer(trailer *cos.Dictionary) {
	if r.current == nil {
		// should not happen
		slog.Warn("pdfparser: cannot add a trailer, no xref start was signalled")
		return
	}
	r.current.trailer = trailer
}

// CurrentTrailer returns the trailer of the section being read.
func (r *XrefTrailerResolver) CurrentTrailer() *cos.Dictionary {
	if r.current == nil {
		return nil
	}
	return r.current.trailer
}

// FirstTrailer returns the trailer of the section at the lowest byte offset.
func (r *XrefTrailerResolver) FirstTrailer() *cos.Dictionary {
	return r.trailerAtExtreme(true)
}

// LastTrailer returns the trailer of the section at the highest byte offset.
func (r *XrefTrailerResolver) LastTrailer() *cos.Dictionary {
	return r.trailerAtExtreme(false)
}

func (r *XrefTrailerResolver) trailerAtExtreme(lowest bool) *cos.Dictionary {
	if len(r.bytePosToXref) == 0 {
		return nil
	}
	var chosen int64
	first := true
	for pos := range r.bytePosToXref {
		if first || (lowest && pos < chosen) || (!lowest && pos > chosen) {
			chosen = pos
			first = false
		}
	}
	return r.bytePosToXref[chosen].trailer
}

// TrailerCount returns how many cross-reference sections were seen.
func (r *XrefTrailerResolver) TrailerCount() int { return len(r.bytePosToXref) }

// SetStartxref resolves the sections into one table, following the /Prev chain
// back from the given offset.
//
// Port of setStartxref. Sections are merged oldest first, so that a later
// incremental update overwrites what it supersedes.
func (r *XrefTrailerResolver) SetStartxref(startxrefBytePos int64) {
	if r.resolved != nil {
		slog.Warn("pdfparser: SetStartxref must be called only once, with the last startxref value")
		return
	}

	r.resolved = newXrefTrailerObj()
	r.resolved.trailer = cos.NewDictionary()

	var sequence []int64
	current, ok := r.bytePosToXref[startxrefBytePos]

	if !ok {
		// No section at the stated offset — a damaged file. Fall back to every
		// section in offset order, so that later ones still win.
		slog.Warn("pdfparser: no xref object at the startxref position",
			"position", startxrefBytePos)
		sequence = append(sequence, r.bytePosOrder...)
		sort.Slice(sequence, func(i, j int) bool { return sequence[i] < sequence[j] })
	} else {
		r.resolved.xrefType = current.xrefType
		sequence = append(sequence, startxrefBytePos)

		for current.trailer != nil {
			prev := current.trailer.GetLongDefault(cos.Prev, -1)
			if prev == -1 {
				break
			}
			next, ok := r.bytePosToXref[prev]
			if !ok {
				slog.Warn("pdfparser: no xref object at the position named by /Prev",
					"position", prev)
				break
			}
			current = next
			sequence = append(sequence, prev)
			// A /Prev chain in a hostile file can be a cycle; stop once the
			// chain is as long as the number of sections that exist.
			if len(sequence) >= len(r.bytePosToXref) {
				break
			}
		}

		// oldest first, so later sections overwrite earlier ones
		for i, j := 0, len(sequence)-1; i < j; i, j = i+1, j-1 {
			sequence[i], sequence[j] = sequence[j], sequence[i]
		}
	}

	for _, pos := range sequence {
		obj, ok := r.bytePosToXref[pos]
		if !ok {
			continue
		}
		if obj.trailer != nil {
			r.resolved.trailer.AddAll(obj.trailer)
		}
		for hash, entry := range obj.xrefTable {
			r.resolved.xrefTable[hash] = entry
		}
	}
}

// Trailer returns the merged trailer, or nil before SetStartxref has run.
func (r *XrefTrailerResolver) Trailer() *cos.Dictionary {
	if r.resolved == nil {
		return nil
	}
	return r.resolved.trailer
}

// XrefTable returns the merged cross-reference table, or nil before
// SetStartxref has run.
func (r *XrefTrailerResolver) XrefTable() map[*cos.ObjectKey]int64 {
	if r.resolved == nil {
		return nil
	}
	out := make(map[*cos.ObjectKey]int64, len(r.resolved.xrefTable))
	for _, entry := range r.resolved.xrefTable {
		out[entry.key] = entry.offset
	}
	return out
}

// ContainedObjectNumbers returns the objects stored inside the object stream
// with the given object number.
//
// Port of getContainedObjectNumbers. An object living in an object stream is
// recorded with the negated object number of that stream as its offset, which
// is how the table distinguishes the two cases.
func (r *XrefTrailerResolver) ContainedObjectNumbers(objectStreamNumber int) []int64 {
	if r.resolved == nil {
		return nil
	}
	target := -int64(objectStreamNumber)
	var out []int64
	for _, entry := range r.resolved.xrefTable {
		if entry.offset == target {
			out = append(out, entry.key.Number())
		}
	}
	// Java returns a HashSet, so the order is unspecified. Go map iteration is
	// equally unspecified, which is the faithful equivalent; sorting here would
	// give the port a guarantee the Java does not have.
	return out
}

// Reset discards the resolved data so that the file can be parsed again, as
// the brute force parser does after a failure.
func (r *XrefTrailerResolver) Reset() {
	for _, obj := range r.bytePosToXref {
		obj.trailer = nil
		obj.xrefTable = make(map[int64]*xrefTableEntry)
	}
	r.resolved = nil
	r.current = nil
}
