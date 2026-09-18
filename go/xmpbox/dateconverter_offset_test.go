package xmpbox_test

// The offsets DateConverter's formatter reads leniently.
//
// Not a case in DateConverterTest. DATE_TIME_FORMATTER appends the offset as
// "+HH:MM" between parseLenient and parseStrict, and java.time reads a lenient
// "+HH:MM" as "+HH:mm:ss": the minutes and the seconds may be left off, an hour
// past 23 or a minute past 59 is not read, and "Z" is matched without regard to
// case. Found by the corpus comparison of XMP schemas: two veraPDF PDF/A-4
// files whose xmp:CreateDate is "2022-11-18T11:54:39.000+03", which PDFBox reads
// and the Go version refused, and with it the whole packet.
//
// The expected values are PDFBox's, printed by DateConverter.toCalendar and
// toISO8601(calendar, true) on JDK 17.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
)

func TestLenientOffset(t *testing.T) {
	for _, c := range []struct{ date, want string }{
		{"2022-11-18T11:54:39.000+03", "2022-11-18T11:54:39.000+03:00"},
		{"2022-11-18T11:54:39.853+03", "2022-11-18T11:54:39.853+03:00"},
		{"2022-11-18T11:54:39+03", "2022-11-18T11:54:39.000+03:00"},
		{"2022-11-18T11:54+03", "2022-11-18T11:54:00.000+03:00"},
		{"2022-11-18T11:54:39-03", "2022-11-18T11:54:39.000-03:00"},
		{"2022-11-18T11:54:39+00", "2022-11-18T11:54:39.000+00:00"},
		{"2022-11-18T11:54:39+03:00", "2022-11-18T11:54:39.000+03:00"},
		{"2022-11-18T11:54:39+03:30", "2022-11-18T11:54:39.000+03:30"},
		{"2022-11-18T11:54:39+03:00:00", "2022-11-18T11:54:39.000+03:00"},
		{"2022-11-18T11:54:39+18", "2022-11-18T11:54:39.000+18:00"},
		{"2022-11-18T11:54:39-18:00", "2022-11-18T11:54:39.000-18:00"},
		{"2022-11-18T11:54:39Z", "2022-11-18T11:54:39.000+00:00"},
		{"2022-11-18T11:54:39z", "2022-11-18T11:54:39.000+00:00"},
		{"2022-11-18T11:54:39.5z", "2022-11-18T11:54:39.500+00:00"},
		{"2022-11-18T11:54z", "2022-11-18T11:54:00.000+00:00"},
		// An offset with seconds is read, and the instant is right, but the
		// calendar is in GMT: GregorianCalendar.from asks TimeZone for the zone
		// "GMT+03:00:30", and JDK 17's TimeZone takes no seconds in a custom
		// zone and answers GMT for one it cannot read.
		{"2022-11-18T11:54:39+03:00:30", "2022-11-18T08:54:09.000+00:00"},
		{"2022-11-18T11:54:39+17:59:59", "2022-11-17T17:54:40.000+00:00"},
	} {
		t.Run(c.date, func(t *testing.T) {
			got := toCalendar(t, c.date)
			equal(t, "ToISO8601", xmpbox.ToISO8601Millis(got, true), c.want)
		})
	}

	// "unparsed text found at index 19", every one
	for _, date := range []string{
		"2022-11-18T11:54:39+3",
		"2022-11-18T11:54:39+0300",
		"2022-11-18T11:54:39+03:0",
		"2022-11-18T11:54:39+03:",
		"2022-11-18T11:54:39+03:60",
		"2022-11-18T11:54:39+03:00:3",
		"2022-11-18T11:54:39+03:00:",
		"2022-11-18T11:54:39+19",
		"2022-11-18T11:54:39+23",
		"2022-11-18T11:54:39+24",
		"2022-11-18T11:54:39+18:00:01",
		"2022-11-18T11:54:39+003",
		"2022-11-18T11:54:39 +03",
	} {
		t.Run(date, func(t *testing.T) { toCalendarFails(t, date) })
	}
}
