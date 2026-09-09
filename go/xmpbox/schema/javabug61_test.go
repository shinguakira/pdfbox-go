package schema_test

// JAVA-BUGS 61: `XMPSchema.getUnqualifiedSequenceDateValueList` adds
// `getValue()` for every DateType in the array, and an element holding no date
// answers null, so the `List<Calendar>` it promises comes back with nulls in
// it.

import (
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// TestSequenceDateValueListSkipsAnEmptyDate is the defect.
//
// The expected list is the dates the sequence holds: an element that holds no
// date has no date to add. Java's javadoc says the method answers "the list of
// Calendar", and a null in it raises NullPointerException in whichever caller
// walks it, with nothing to say where it came from. A blank date string is how
// one is built -- DateConverter.toCalendar answers null for it, see
// PDFBOX-6029.
func TestSequenceDateValueListSkipsAnEmptyDate(t *testing.T) {
	parent, schem := bareSchema(t)
	const seqName = "dateSeq"

	when := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	noError(t, "AddUnqualifiedSequenceDateValue",
		schem.AddUnqualifiedSequenceDateValue(seqName, when))

	blank, err := xmptype.NewDateType(parent, "", xmptype.DefaultRDFLocalName,
		xmptype.ListName, "")
	noError(t, "NewDateType", err)
	if _, hasValue := blank.DateValue(); hasValue {
		t.Fatal("the blank date holds a date, so the test proves nothing")
	}
	schem.AddUnqualifiedSequenceField(seqName, blank)

	dates := schem.UnqualifiedSequenceDateValueList(seqName)
	if len(dates) != 1 {
		t.Fatalf("the list holds %d dates, want the one the sequence has: %v",
			len(dates), dates)
	}
	if !dates[0].Equal(when) {
		t.Errorf("the date is %v, want %v", dates[0], when)
	}
}
