package xmpbox_test

// What DateConverter accepts that Go's own parser does not, and the one place
// the two calendars part company.
//
// Not a case in DateConverterTest: the values are read from Java, by running
// org.apache.xmpbox.DateConverter over them, because they are what the port had
// to be checked against rather than what the Java test asserts. See
// migration/STATUS.md, the track's adversarial review.

import (
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
)

// TestSmartResolverClampsTheDayOfMonth pins the one relaxation Java's
// ResolverStyle.SMART makes: a day of month within 1 to 31 but past the end of
// its month becomes the last day of that month.
func TestSmartResolverClampsTheDayOfMonth(t *testing.T) {
	for _, c := range []struct{ date, want string }{
		{"2015-02-30T00:00:00Z", "2015-02-28T00:00:00.000+00:00"},
		{"2015-02-29T00:00:00Z", "2015-02-28T00:00:00.000+00:00"},
		{"2016-02-29T00:00:00Z", "2016-02-29T00:00:00.000+00:00"},
		{"2015-04-31T00:00:00Z", "2015-04-30T00:00:00.000+00:00"},
		{"2015-06-31T12:00:00+02:00", "2015-06-30T12:00:00.000+02:00"},
		{"2015-02-31T00:00:00.500Z", "2015-02-28T00:00:00.500+00:00"},
	} {
		t.Run(c.date, func(t *testing.T) {
			got := toCalendar(t, c.date)
			equal(t, "ToISO8601", xmpbox.ToISO8601Millis(got, true), c.want)
		})
	}
}

// TestSmartResolverRefusesWhatIsOutOfRange pins the other half: SMART relaxes
// only the day of month, so a month, hour, minute or day outside its own range
// is still refused.
func TestSmartResolverRefusesWhatIsOutOfRange(t *testing.T) {
	for _, date := range []string{
		"2015-01-32T00:00:00Z",
		"2015-00-01T00:00:00Z",
		"2015-13-01T00:00:00Z",
		"2015-02-30T25:00:00Z",
		"2015-02-30T10:61:00Z",
	} {
		t.Run(date, func(t *testing.T) { toCalendarFails(t, date) })
	}
}

// TestDatesBefore1582DifferFromJava pins the one place the port and Java answer
// different instants for the same string.
//
// Java reads a date into a GregorianCalendar, which switches to the Julian
// calendar before the 1582 cutover; Go's time.Time is proleptic Gregorian
// throughout. Both write the same ISO 8601 string back, and the instants differ
// by the days the two calendars had drifted apart. See migration/STATUS.md.
func TestDatesBefore1582DifferFromJava(t *testing.T) {
	for _, c := range []struct {
		date       string
		want       string
		javaMillis int64
	}{
		{"0000-01-01", "0001-01-01T00:00:00.000+00:00", -62167392000000},
		{"123456", "1238-08-01T00:00:00.000+00:00", -23080723200000},
	} {
		t.Run(c.date, func(t *testing.T) {
			got := toCalendar(t, c.date).In(time.UTC)
			equal(t, "ToISO8601", xmpbox.ToISO8601Millis(got, true), c.want)
			if got.UnixMilli() == c.javaMillis {
				t.Errorf("UnixMilli() = %d, which now matches Java's; the "+
					"deviation recorded in STATUS.md is gone and this test "+
					"should say so", got.UnixMilli())
			}
		})
	}
}
