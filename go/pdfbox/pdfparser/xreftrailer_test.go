package pdfparser

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from
// pdfbox/src/main/java/org/apache/pdfbox/pdfparser/XrefTrailerResolver.java.
// The Java suite has no test for this class, so per
// migration/conventions/tdd.md these are written first from the source.

func key(t *testing.T, num int64) *cos.ObjectKey {
	t.Helper()
	k, err := cos.NewObjectKey(num, 0)
	if err != nil {
		t.Fatalf("NewObjectKey(%d): %v", num, err)
	}
	return k
}

func TestResolverSingleSection(t *testing.T) {
	r := NewXrefTrailerResolver()
	r.NextXrefObj(100, XRefTypeTable)
	r.SetXRef(key(t, 1), 10)
	r.SetXRef(key(t, 2), 20)

	trailer := cos.NewDictionary()
	trailer.SetInt(cos.Size, 3)
	r.SetTrailer(trailer)

	if got := r.CurrentTrailer(); got != trailer {
		t.Errorf("CurrentTrailer() = %v, want the trailer that was set", got)
	}
	if got := r.TrailerCount(); got != 1 {
		t.Errorf("TrailerCount() = %d, want 1", got)
	}

	// nothing is resolved until SetStartxref runs
	if got := r.Trailer(); got != nil {
		t.Errorf("Trailer() = %v before SetStartxref, want nil", got)
	}
	if got := r.XrefTable(); got != nil {
		t.Errorf("XrefTable() = %v before SetStartxref, want nil", got)
	}
	if got := r.XrefType(); got != XRefTypeNone {
		t.Errorf("XrefType() = %v before SetStartxref, want None", got)
	}

	r.SetStartxref(100)

	if got := r.XrefType(); got != XRefTypeTable {
		t.Errorf("XrefType() = %v, want Table", got)
	}
	table := r.XrefTable()
	if len(table) != 2 {
		t.Fatalf("XrefTable() has %d entries, want 2", len(table))
	}
	if got := r.Trailer().GetInt(cos.Size); got != 3 {
		t.Errorf("resolved trailer /Size = %d, want 3", got)
	}
}

// TestResolverPrevChain covers the case incremental updates produce: several
// sections chained by /Prev, merged so that the newest wins.
func TestResolverPrevChain(t *testing.T) {
	r := NewXrefTrailerResolver()

	// the older section, at offset 100
	r.NextXrefObj(100, XRefTypeTable)
	r.SetXRef(key(t, 1), 10)
	r.SetXRef(key(t, 2), 20)
	older := cos.NewDictionary()
	older.SetInt(cos.Size, 3)
	older.SetName(cos.Type, "Older")
	r.SetTrailer(older)

	// the newer section, at offset 500, pointing back at it
	r.NextXrefObj(500, XRefTypeTable)
	r.SetXRef(key(t, 2), 999) // supersedes the older offset for object 2
	r.SetXRef(key(t, 3), 30)
	newer := cos.NewDictionary()
	newer.SetLong(cos.Prev, 100)
	newer.SetInt(cos.Size, 4)
	r.SetTrailer(newer)

	r.SetStartxref(500)

	table := r.XrefTable()
	if len(table) != 3 {
		t.Fatalf("XrefTable() has %d entries, want 3", len(table))
	}
	for k, offset := range table {
		switch k.Number() {
		case 1:
			if offset != 10 {
				t.Errorf("object 1 offset = %d, want 10", offset)
			}
		case 2:
			// the newer section wins
			if offset != 999 {
				t.Errorf("object 2 offset = %d, want 999 from the newer section", offset)
			}
		case 3:
			if offset != 30 {
				t.Errorf("object 3 offset = %d, want 30", offset)
			}
		}
	}

	// the merged trailer takes the newer /Size, and keeps entries only the
	// older one had
	if got := r.Trailer().GetInt(cos.Size); got != 4 {
		t.Errorf("merged /Size = %d, want 4 from the newer trailer", got)
	}
	if got := r.Trailer().GetNameAsString(cos.Type, ""); got != "Older" {
		t.Errorf("merged /Type = %q, want Older to survive from the older trailer", got)
	}
}

// TestResolverPDFBOX3506 covers PDFBOX-3506: within one section an entry
// already present is not overwritten, so that in a hybrid file the table
// entries are not replaced by the obsolete ones the /XRefStm repeats.
func TestResolverPDFBOX3506(t *testing.T) {
	r := NewXrefTrailerResolver()
	r.NextXrefObj(100, XRefTypeTable)

	r.SetXRef(key(t, 1), 10)
	r.SetXRef(key(t, 1), 999) // must be ignored

	r.SetTrailer(cos.NewDictionary())
	r.SetStartxref(100)

	for k, offset := range r.XrefTable() {
		if k.Number() == 1 && offset != 10 {
			t.Errorf("object 1 offset = %d, want 10 — the first entry must win", offset)
		}
	}
}

// TestResolverMissingStartxref covers a damaged file whose startxref points
// nowhere: every section is used instead, in offset order.
func TestResolverMissingStartxref(t *testing.T) {
	r := NewXrefTrailerResolver()

	r.NextXrefObj(100, XRefTypeTable)
	r.SetXRef(key(t, 1), 10)
	r.SetTrailer(cos.NewDictionary())

	r.NextXrefObj(500, XRefTypeTable)
	r.SetXRef(key(t, 1), 999)
	r.SetTrailer(cos.NewDictionary())

	r.SetStartxref(99999) // nowhere

	table := r.XrefTable()
	if len(table) != 1 {
		t.Fatalf("XrefTable() has %d entries, want 1", len(table))
	}
	for _, offset := range table {
		// the section at the higher offset is applied last
		if offset != 999 {
			t.Errorf("offset = %d, want 999 from the later section", offset)
		}
	}
}

// TestResolverPrevCycle covers a hostile file whose /Prev chain loops: the walk
// must terminate.
func TestResolverPrevCycle(t *testing.T) {
	r := NewXrefTrailerResolver()

	first := cos.NewDictionary()
	first.SetLong(cos.Prev, 500)
	r.NextXrefObj(100, XRefTypeTable)
	r.SetTrailer(first)

	second := cos.NewDictionary()
	second.SetLong(cos.Prev, 100)
	r.NextXrefObj(500, XRefTypeTable)
	r.SetTrailer(second)

	// must terminate rather than following the cycle forever
	r.SetStartxref(500)

	if r.Trailer() == nil {
		t.Error("Trailer() = nil after resolving a cyclic /Prev chain")
	}
}

// TestResolverContainedObjectNumbers covers how the table records an object
// living inside an object stream: its offset is the negated object number of
// that stream.
func TestResolverContainedObjectNumbers(t *testing.T) {
	r := NewXrefTrailerResolver()
	r.NextXrefObj(100, XRefTypeStream)

	r.SetXRef(key(t, 5), -9) // object 5 lives in object stream 9
	r.SetXRef(key(t, 6), -9) // and so does object 6
	r.SetXRef(key(t, 7), 40) // this one is at a real offset

	r.SetTrailer(cos.NewDictionary())
	r.SetStartxref(100)

	// Java returns a HashSet, so the order is unspecified and the test must not
	// depend on one.
	got := r.ContainedObjectNumbers(9)
	if len(got) != 2 {
		t.Fatalf("ContainedObjectNumbers(9) = %v, want two entries", got)
	}
	seen := map[int64]bool{got[0]: true, got[1]: true}
	if !seen[5] || !seen[6] {
		t.Errorf("ContainedObjectNumbers(9) = %v, want objects 5 and 6", got)
	}
	if got := r.ContainedObjectNumbers(99); len(got) != 0 {
		t.Errorf("ContainedObjectNumbers(99) = %v, want empty", got)
	}
}

func TestResolverFirstAndLastTrailer(t *testing.T) {
	r := NewXrefTrailerResolver()

	firstTrailer := cos.NewDictionary()
	firstTrailer.SetName(cos.Type, "First")
	r.NextXrefObj(100, XRefTypeTable)
	r.SetTrailer(firstTrailer)

	lastTrailer := cos.NewDictionary()
	lastTrailer.SetName(cos.Type, "Last")
	r.NextXrefObj(500, XRefTypeTable)
	r.SetTrailer(lastTrailer)

	if got := r.FirstTrailer(); got != firstTrailer {
		t.Errorf("FirstTrailer() = %v, want the one at the lowest offset", got)
	}
	if got := r.LastTrailer(); got != lastTrailer {
		t.Errorf("LastTrailer() = %v, want the one at the highest offset", got)
	}
}

// TestResolverSetXRefWithoutSection pins that an entry offered before any
// section was signalled is dropped rather than crashing. Java logs and returns.
func TestResolverSetXRefWithoutSection(t *testing.T) {
	r := NewXrefTrailerResolver()
	r.SetXRef(key(t, 1), 10)
	r.SetTrailer(cos.NewDictionary())

	if got := r.TrailerCount(); got != 0 {
		t.Errorf("TrailerCount() = %d, want 0", got)
	}
}

func TestResolverReset(t *testing.T) {
	r := NewXrefTrailerResolver()
	r.NextXrefObj(100, XRefTypeTable)
	r.SetXRef(key(t, 1), 10)
	r.SetTrailer(cos.NewDictionary())
	r.SetStartxref(100)

	r.Reset()

	if got := r.Trailer(); got != nil {
		t.Errorf("Trailer() = %v after Reset, want nil", got)
	}
	if got := r.XrefTable(); got != nil {
		t.Errorf("XrefTable() = %v after Reset, want nil", got)
	}
}

// TestResolverResolvedTypeDefaultsToTable pins the XrefTrailerObj constructor,
// which sets xrefType = XRefType.TABLE. When startxref points nowhere the
// resolved object keeps that default rather than reporting no type at all.
func TestResolverResolvedTypeDefaultsToTable(t *testing.T) {
	r := NewXrefTrailerResolver()
	r.NextXrefObj(100, XRefTypeStream)
	r.SetXRef(key(t, 1), 10)
	r.SetTrailer(cos.NewDictionary())

	r.SetStartxref(99999) // nowhere, so the type is not copied from a section

	if got := r.XrefType(); got != XRefTypeTable {
		t.Errorf("XrefType() = %v, want Table — the resolved object defaults to it", got)
	}
}
