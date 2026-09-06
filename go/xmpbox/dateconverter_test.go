package xmpbox_test

// Port of org.apache.xmpbox.DateConverterTest.

import (
	"os"
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
)

// TestMain fixes the zone every date in this package is read and written in.
//
// The Java suite runs DateConverterTest with the default TimeZone left at UTC
// -- DeserializationTest sets it there and restores it, under @Isolated and
// @ResourceLock(TIME_ZONE) -- and the PDFBOX-6107 case only holds in that zone.
// The port says so rather than depending on where the test is run.
func TestMain(m *testing.M) {
	defaultTZ := time.Local
	time.Local = time.UTC
	code := m.Run()
	time.Local = defaultTZ
	os.Exit(code)
}

// toCalendar reads a date and fails the test where it will not.
func toCalendar(t *testing.T, date string) time.Time {
	t.Helper()
	converted, err := xmpbox.ToCalendar(date)
	if err != nil {
		t.Fatalf("ToCalendar(%q): %v", date, err)
	}
	return converted
}

// toCalendarFails checks that a date is refused, which is Java's
// assertThrows(IOException.class).
func toCalendarFails(t *testing.T, date string) {
	t.Helper()
	if _, err := xmpbox.ToCalendar(date); err == nil {
		t.Errorf("ToCalendar(%q) reported nothing, want a failure", date)
	}
}

// TestDateConversion parses several ISO 8601 date formats, including time zone
// information ISO 8601 does not normally allow.
func TestDateConversion(t *testing.T) {
	// Test partial dates
	equal(t, "toCalendar(2015).Year()", toCalendar(t, "2015").Year(), 2015)
	// Java asks for Calendar.MONTH, which counts from zero; Go's Month counts
	// from one, so the assertions below are one higher than Java's.
	equal(t, "toCalendar(2015-05).Month()", toCalendar(t, "2015-05").Month(), time.May)

	convDate := toCalendar(t, "2015-05-02")
	equal(t, "Year()", convDate.Year(), 2015)
	equal(t, "Month()", convDate.Month(), time.May)
	equal(t, "Day()", convDate.Day(), 2)

	equal(t, "toCalendar(D:2015-02-02).Year()", toCalendar(t, "D:2015-02-02").Year(), 2015)

	for _, date := range []string{
		"D:2015-02-03T10:11:12",
		"D:2015-02-03T10:11:12Z",
		"D:2015-02-03T10:11:12+05:00",
		"D:2015-02-03T10:11:12-05:00",
	} {
		convDate = toCalendar(t, date)
		equal(t, date+" Year()", convDate.Year(), 2015)
		equal(t, date+" Month()", convDate.Month(), time.February)
		equal(t, date+" Day()", convDate.Day(), 3)
		equal(t, date+" Hour()", convDate.Hour(), 10)
		equal(t, date+" Minute()", convDate.Minute(), 11)
		equal(t, date+" Second()", convDate.Second(), 12)
	}

	convDate = toCalendar(t, "D:2015-02-03T10:11:12+05:00")
	_, offset := convDate.Zone()
	equal(t, "zone offset", offset, 5*3600)
	convDate = toCalendar(t, "D:2015-02-03T10:11:12-05:00")
	_, offset = convDate.Zone()
	equal(t, "zone offset", offset, -5*3600)

	// Java asks for Calendar.MILLISECOND; Go keeps nanoseconds.
	convDate = toCalendar(t, "2025-09-03T15:43:47.989082+00:00")
	equal(t, "millisecond", convDate.Nanosecond()/int(time.Millisecond), 989)

	// test some bad strings
	for _, bad := range []string{
		"123",
		"2008-12-31T19:48:30+19:00",
		"2008-12-31T19:48:30-19:00",
		"2008-12-02T21:04:0Z",
		"0-01-01T00:00:00Z",
		"2009-03-16T01:15:19-0-4:00",
		"0-00-00T00:00:00-04:00",
	} {
		toCalendarFails(t, bad)
	}

	// Test missing seconds
	for _, pair := range [][2]string{
		{"2015-12-08T12:07:00-05:00", "2015-12-08T12:07-05:00"},
		{"2011-11-20T10:09:00Z", "2011-11-20T10:09Z"},
	} {
		with, without := toCalendar(t, pair[0]), toCalendar(t, pair[1])
		if !with.Equal(without) {
			t.Errorf("ToCalendar(%q) = %v, ToCalendar(%q) = %v, want the same instant",
				pair[0], with, pair[1], without)
		}
	}

	// Test some time zone offsets, against what Go's own ISO 8601 parser makes
	// of the same string -- which is what Java's DateTimeFormatter is there for.
	for _, date := range []string{
		"2015-12-08T12:07:00-05:00",
		"2015-02-02T16:37:19.192Z",
		"2015-02-02T16:37:19.192+00:00",
		"2015-02-02T16:37:19.192+02:00",
		// PDFBOX-4902: half-hour TZ
		"2015-02-02T16:37:19.192+05:30",
		"2015-02-02T16:37:19.192-05:30",
		"2015-02-02T16:37:19.192+10:30",
	} {
		want, err := time.Parse(time.RFC3339Nano, date)
		noError(t, "time.Parse", err)
		if got := toCalendar(t, date); !got.Equal(want) {
			t.Errorf("ToCalendar(%q) = %v, want the instant %v", date, got, want)
		}
	}
	// A date with no zone is read in the default one, which the Java test runs
	// under as UTC.
	const local = "2024-04-09T14:41:38"
	want, err := time.ParseInLocation("2006-01-02T15:04:05", local, time.UTC)
	noError(t, "time.ParseInLocation", err)
	if got := toCalendar(t, local); !got.Equal(want) {
		t.Errorf("ToCalendar(%q) = %v, want the instant %v", local, got, want)
	}

	// Java asserts null for a null and an empty string; the port answers the
	// zero time, which IsBlankDate is how a caller tells apart.
	if got := toCalendar(t, ""); !got.IsZero() {
		t.Errorf("ToCalendar(\"\") = %v, want the zero time", got)
	}
}

// TestDateFormatting writes ISO 8601 date formats, including time zone
// information ISO 8601 does not normally allow.
func TestDateFormatting(t *testing.T) {
	for _, date := range []string{
		"2015-02-02T16:37:19.192Z",
		"2015-02-02T16:37:19.192+09:09",
		"2015-02-02T16:37:19.192+10:10",
	} {
		cal := toCalendar(t, date)
		again := toCalendar(t, xmpbox.ToISO8601Millis(cal, true))
		if !cal.Equal(again) {
			t.Errorf("ToCalendar(ToISO8601(%q)) = %v, want the instant %v", date, again, cal)
		}
	}

	// PDFBOX-6107
	cal := toCalendar(t, "0000-01-01").In(time.UTC)
	equal(t, "ToISO8601", xmpbox.ToISO8601(cal), "0001-01-01T00:00:00+00:00")
}
