package util

import (
	"fmt"
	"testing"
	"time"
)

// Port of org.apache.pdfbox.util.TestDateUtil.
//
// Java sets the default time zone in testExtract; the port has no default time
// zone to set, since every date it builds names its own.

// bad marks a date the parse is expected to reject.
//
// Port of the private TestDateUtil.BAD.
const bad = -666

// The milliseconds a minute and an hour hold, as the test names them.
const (
	mins = 60 * 1000
	hrs  = 60 * mins
)

// assertCalendarEquals is the private helper of the same name: the instants and
// the zone offsets have to match.
func assertCalendarEquals(t *testing.T, expect, was time.Time) {
	t.Helper()
	if expect.UnixMilli() != was.UnixMilli() {
		t.Errorf("time = %v, want %v", was, expect)
	}
	_, expectOffset := expect.Zone()
	_, wasOffset := was.Zone()
	if expectOffset != wasOffset {
		t.Errorf("zone offset = %d, want %d", wasOffset, expectOffset)
	}
}

func TestExtract(t *testing.T) {
	got, ok := ToCalendar("D:05/12/2005")
	if !ok {
		t.Fatalf("ToCalendar(D:05/12/2005) failed")
	}
	assertCalendarEquals(t, time.Date(2005, time.May, 12, 0, 0, 0, 0, time.UTC), got)

	got, ok = ToCalendar("5/12/2005 15:57:16")
	if !ok {
		t.Fatalf("ToCalendar(5/12/2005 15:57:16) failed")
	}
	assertCalendarEquals(t, time.Date(2005, time.May, 12, 15, 57, 16, 0, time.UTC), got)

	// check that toCalendar gives null for a null arg
	if _, ok := ToCalendar(""); ok {
		t.Errorf("ToCalendar of the empty string answered a date")
	}
}

func TestDateConversion(t *testing.T) {
	c, ok := ToCalendar("D:20050526205258+01'00'")
	if !ok {
		t.Fatalf("ToCalendar failed")
	}
	if c.Year() != 2005 {
		t.Errorf("year = %d, want 2005", c.Year())
	}
	if c.Month() != time.May {
		t.Errorf("month = %v, want May", c.Month())
	}
	if c.Day() != 26 {
		t.Errorf("day = %d, want 26", c.Day())
	}
	if c.Hour() != 20 {
		t.Errorf("hour = %d, want 20", c.Hour())
	}
	if c.Minute() != 52 {
		t.Errorf("minute = %d, want 52", c.Minute())
	}
	if c.Second() != 58 {
		t.Errorf("second = %d, want 58", c.Second())
	}
	if c.Nanosecond() != 0 {
		t.Errorf("nanosecond = %d, want 0", c.Nanosecond())
	}
}

// checkParse is the private helper of the same name.
func checkParse(t *testing.T, yr, mon, day, hr, min, sec, offsetHours, offsetMinutes int,
	orig string) {
	t.Helper()
	pdfDate := fmt.Sprintf("D:%04d%02d%02d%02d%02d%02d%+03d'%02d'",
		yr, mon, day, hr, min, sec, offsetHours, offsetMinutes)
	iso8601Date := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d%+03d:%02d",
		yr, mon, day, hr, min, sec, offsetHours, offsetMinutes)

	cal, ok := ToCalendar(orig)
	if ok {
		if got := ToISO8601(cal); got != iso8601Date {
			t.Errorf("ToISO8601(%q) = %q, want %q", orig, got, iso8601Date)
		}
		if got := ToString(cal); got != pdfDate {
			t.Errorf("ToString(%q) = %q, want %q", orig, got, pdfDate)
		}
	}

	if yr == bad {
		if ok {
			t.Errorf("ToCalendar(%q) = %v, want no date", orig, cal)
		}
		return
	}
	if !ok {
		t.Errorf("ToCalendar(%q) answered no date, want %q", orig, pdfDate)
		return
	}
	if got := ToString(cal); got != pdfDate {
		t.Errorf("ToString(%q) = %q, want %q", orig, got, pdfDate)
	}
}

func TestDateConverter(t *testing.T) {
	year := time.Now().Year()

	checkParse(t, 2010, 4, 23, 0, 0, 0, 0, 0, "D:20100423")
	checkParse(t, 2011, 4, 23, 0, 0, 0, 0, 0, "20110423")
	checkParse(t, 2012, 1, 1, 0, 0, 0, 0, 0, "D:2012")
	checkParse(t, 2013, 1, 1, 0, 0, 0, 0, 0, "2013")
	// PDFBOX-1219
	checkParse(t, 2001, 1, 31, 10, 33, 0, +1, 0, "2001-01-31T10:33+01:00  ")
	// Same with milliseconds
	checkParse(t, 2001, 1, 31, 10, 33, 0, +1, 0, "2001-01-31T10:33.123+01:00")
	// PDFBOX-465
	checkParse(t, 2002, 5, 12, 9, 47, 0, 0, 0, "9:47 5/12/2002")
	// PDFBOX-465
	checkParse(t, 2003, 12, 17, 2, 2, 3, 0, 0, "200312172:2:3")
	// PDFBOX-465
	checkParse(t, 2009, 3, 19, 20, 1, 22, 0, 0, "  20090319 200122")
	checkParse(t, 2014, 4, 1, 0, 0, 0, +2, 0, "20140401+0200")
	// "EEEE, MMM dd, yy",
	checkParse(t, 2115, 1, 11, 0, 0, 0, 0, 0, "Friday, January 11, 2115")
	// "EEEE, MMM dd, yy",
	checkParse(t, 1915, 1, 11, 0, 0, 0, 0, 0, "Monday, Jan 11, 1915")
	// "EEEE, MMM dd, yy",
	checkParse(t, 2215, 1, 11, 0, 0, 0, 0, 0, "Wed, January 11, 2215")
	// "EEEE, MMM dd, yy",
	checkParse(t, 2015, 1, 11, 0, 0, 0, 0, 0, " Sun, January 11, 2015 ")
	checkParse(t, 2016, 4, 1, 0, 0, 0, +4, 0, "20160401+04'00'")
	checkParse(t, 2017, 4, 1, 0, 0, 0, +9, 0, "20170401+09'00'")
	checkParse(t, 2017, 4, 1, 0, 0, 0, +9, 30, "20170401+09'30'")
	checkParse(t, 2018, 4, 1, 0, 0, 0, -2, 0, "20180401-02'00'")
	checkParse(t, 2019, 4, 1, 6, 1, 1, -11, 0, "20190401 6:1:1 -1100")
	checkParse(t, 2020, 5, 26, 11, 25, 10, 0, 0, "26 May 2020 11:25:10")
	checkParse(t, 2021, 5, 26, 11, 23, 0, 0, 0, "26 May 2021 11:23")

	// half hour timezones
	checkParse(t, 2016, 4, 1, 0, 0, 0, +4, 30, "20160401+04'30'")
	checkParse(t, 2017, 4, 1, 0, 0, 0, +9, 30, "20170401+09'30'")
	checkParse(t, 2018, 4, 1, 0, 0, 0, -2, 30, "20180401-02'30'")
	checkParse(t, 2019, 4, 1, 6, 1, 1, -11, 30, "20190401 6:1:1 -1130")
	checkParse(t, 2000, 2, 29, 0, 0, 0, +11, 30, " 2000 Feb 29 GMT + 11:30")

	// try dates invalid due to out of limit values
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "Tuesday, May 32 2000 11:27 UCT")
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "32 May 2000 11:25")
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "Tuesday, May 32 2000 11:25")
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "19921301 11:25")
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "19921232 11:25")
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "19921001 11:60")
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "19920401 24:25")
	// PDFBOX-465
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "20070430193647+713'00' illegal tz hr")
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "nodigits")
	// PDFBOX-465
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "Unknown")
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "333three digit year")
	// valid date
	checkParse(t, 2000, 2, 29, 0, 0, 0, 0, 0, "2000 Feb 29")
	// valid date
	checkParse(t, 2000, 2, 29, 0, 0, 0, +11, 0, " 2000 Feb 29 GMT + 11:00")
	// valid date
	checkParse(t, 2000, 2, 29, 0, 0, 0, +11, 0, " 2000 Feb 29 UTC + 11:00")
	// invalid date
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "2100 Feb 29 GMT+11")
	// valid date
	checkParse(t, 2012, 2, 29, 0, 0, 0, +11, 0, "2012 Feb 29 GMT+11")
	// invalid date
	checkParse(t, bad, 0, 0, 0, 0, 0, 0, 0, "2012 Feb 30 GMT+11")
	// test ambiguous date
	checkParse(t, 1970, 12, 23, 0, 8, 0, 0, 0, "1970 12 23:08")

	// test cases for all entries on old formats list
	//  "E, dd MMM yyyy hh:mm:ss a"
	checkParse(t, 1971, 7, 6, 17, 22, 1, 0, 0, "Tuesday, 6 Jul 1971 5:22:1 PM")
	//  "EE, MMM dd, yyyy hh:mm:ss a"
	checkParse(t, 1972, 7, 6, 17, 22, 1, 0, 0, "Thu, July 6, 1972 5:22:1 pm")
	//  "MM/dd/yyyy hh:mm:ss"
	checkParse(t, 1973, 7, 6, 17, 22, 1, 0, 0, "7/6/1973 17:22:1")
	//  "MM/dd/yyyy"
	checkParse(t, 1974, 7, 6, 0, 0, 0, 0, 0, "7/6/1974")
	//  "yyyy-MM-dd'T'HH:mm:ss'Z'"
	checkParse(t, 1975, 7, 6, 17, 22, 1, -10, 0, "1975-7-6T17:22:1-1000")
	//  "yyyy-MM-dd'T'HH:mm:ssz"
	checkParse(t, 1976, 7, 6, 17, 22, 1, -4, 0, "1976-7-6T17:22:1GMT-4")
	//  "EDT" is not a known tz ID
	checkParse(t, bad, 7, 6, 17, 22, 1, -4, 0, "2076-7-6T17:22:1EDT")
	//  "EST" does not have a DST rule
	checkParse(t, 1960, 7, 6, 17, 22, 1, -5, 0, "1960-7-6T17:22:1EST")
	//  "EEEE, MMM dd, yyyy"
	checkParse(t, 1977, 7, 6, 0, 0, 0, 0, 0, "Wednesday, Jul 6, 1977")
	//  "EEEE MMM dd, yyyy HH:mm:ss"
	checkParse(t, 1978, 7, 6, 17, 22, 1, 0, 0, "Thu Jul 6, 1978 17:22:1")
	//  "EEEE MMM dd HH:mm:ss z yyyy"
	checkParse(t, 1979, 7, 6, 17, 22, 1, +8, 0, "Friday July 6 17:22:1 GMT+08:00 1979")
	//  "EEEE, MMM dd, yyyy 'at' hh:mma"
	checkParse(t, 1980, 7, 6, 16, 23, 0, 0, 0, "Sun, Jul 6, 1980 at 4:23pm")
	//  "EEEEEEEEEE, MMMMMMMMMMMM dd, yyyy"
	checkParse(t, 1981, 7, 6, 0, 0, 0, 0, 0, "Monday, July 6, 1981")
	//  "dd MMM yyyy hh:mm:ss"
	checkParse(t, 1982, 7, 6, 17, 22, 1, 0, 0, "6 Jul 1982 17:22:1")
	//  "M/dd/yyyy hh:mm:ss"
	checkParse(t, 1983, 7, 6, 17, 22, 1, 0, 0, "7/6/1983 17:22:1")
	//  "MM/d/yyyy hh:mm:ss"
	checkParse(t, 1984, 7, 6, 17, 22, 1, 0, 0, "7/6/1984 17:22:01")
	//  "M/dd/yyyy"
	checkParse(t, 1985, 7, 6, 0, 0, 0, 0, 0, "7/6/1985")
	//  "MM/d/yyyy"
	checkParse(t, 1986, 7, 6, 0, 0, 0, 0, 0, "07/06/1986")
	//  "M/d/yyyy hh:mm:ss"
	checkParse(t, 1987, 7, 6, 17, 22, 1, 0, 0, "7/6/1987 17:22:1")
	//  "M/d/yyyy"
	checkParse(t, 1988, 7, 6, 0, 0, 0, 0, 0, "7/6/1988")

	// test ends of range of two digit years
	//  "M/d/yy hh:mm:ss"
	checkParse(t, year-79, 1, 1, 0, 0, 0, 0, 0,
		fmt.Sprintf("1/1/%d 00:00:00", (year-79)%100))
	//  "M/d/yy"
	checkParse(t, year+19, 1, 1, 0, 0, 0, 0, 0, fmt.Sprintf("1/1/%d", (year+19)%100))

	//  "yyyyMMdd hh:mm:ss Z"
	checkParse(t, 1991, 7, 6, 17, 7, 1, +6, 0, "19910706 17:7:1 Z+0600")
	//  "yyyyMMdd hh:mm:ss"
	checkParse(t, 1992, 7, 6, 17, 7, 1, 0, 0, "19920706 17:07:01")
	//  "yyyyMMdd'+00''00'''"
	checkParse(t, 1993, 7, 6, 0, 0, 0, 0, 0, "19930706+00'00'")
	//  "yyyyMMdd'+01''00'''"
	checkParse(t, 1994, 7, 6, 0, 0, 0, 1, 0, "19940706+01'00'")
	//  "yyyyMMdd'+02''00'''"
	checkParse(t, 1995, 7, 6, 0, 0, 0, 2, 0, "19950706+02'00'")
	//  "yyyyMMdd'+03''00'''"
	checkParse(t, 1996, 7, 6, 0, 0, 0, 3, 0, "19960706+03'00'")
	// "yyyyMMdd'-10''00'''"
	checkParse(t, 1997, 7, 6, 0, 0, 0, -10, 0, "19970706-10'00'")
	// "yyyyMMdd'-11''00'''"
	checkParse(t, 1998, 7, 6, 0, 0, 0, -11, 0, "19980706-11'00'")
	//  "yyyyMMdd"
	checkParse(t, 1999, 7, 6, 0, 0, 0, 0, 0, "19990706")
	// ambiguous big-endian date
	checkParse(t, 2073, 12, 25, 0, 8, 0, 0, 0, "2073 12 25:08")
	// PDFBOX-3315 GMT+12
	checkParse(t, 2016, 4, 11, 16, 1, 15, 12, 0, "D:20160411160115+12'00'")
}

// checkToString is the private helper of the same name.
func checkToString(t *testing.T, yr, mon, day, hr, min, sec int, tz *time.Location,
	offsetHours, offsetMinutes int) {
	t.Helper()
	// construct a calendar from args
	cal := time.Date(yr, time.Month(mon), day, hr, min, sec, 0, tz)

	// create expected strings
	pdfDate := fmt.Sprintf("D:%04d%02d%02d%02d%02d%02d%+03d'%02d'",
		yr, mon, day, hr, min, sec, offsetHours, offsetMinutes)
	iso8601Date := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d%+03d:%02d",
		yr, mon, day, hr, min, sec, offsetHours, offsetMinutes)

	// compare outputs from toString and toISO8601 with expected values
	if got := ToString(cal); got != pdfDate {
		t.Errorf("ToString = %q, want %q", got, pdfDate)
	}
	if got := ToISO8601(cal); got != iso8601Date {
		t.Errorf("ToISO8601 = %q, want %q", got, iso8601Date)
	}
}

func TestToString(t *testing.T) {
	//                                                        std DST
	tzPgh := mustLoadZone(t, "America/New_York")        // -5 -4
	tzBerlin := mustLoadZone(t, "Europe/Berlin")        // +1 +2
	tzMaputo := mustLoadZone(t, "Africa/Maputo")        // +2 +2
	tzAruba := mustLoadZone(t, "America/Aruba")         // -4 -4
	tzJamaica := mustLoadZone(t, "America/Jamaica")     // -5 -5
	tzAdelaide := mustLoadZone(t, "Australia/Adelaide") // +9:30 +10:30

	if _, ok := ToCalendar(""); ok {
		t.Errorf("ToCalendar of the empty string answered a date")
	}
	if _, ok := ToCalendar("D:    "); ok {
		t.Errorf("ToCalendar(D:    ) answered a date")
	}
	if _, ok := ToCalendar("D:"); ok {
		t.Errorf("ToCalendar(D:) answered a date")
	}

	checkToString(t, 2013, 8, 28, 3, 14, 15, tzPgh, -4, 0)
	checkToString(t, 2014, 2, 28, 3, 14, 15, tzPgh, -5, 0)
	checkToString(t, 2015, 8, 28, 3, 14, 15, tzBerlin, +2, 0)
	checkToString(t, 2016, 2, 28, 3, 14, 15, tzBerlin, +1, 0)
	checkToString(t, 2017, 8, 28, 3, 14, 15, tzAruba, -4, 0)
	checkToString(t, 2018, 1, 1, 1, 14, 15, tzJamaica, -5, 0)
	checkToString(t, 2019, 12, 31, 12, 59, 59, tzJamaica, -5, 0)
	checkToString(t, 2020, 2, 29, 0, 0, 0, tzMaputo, +2, 0)
	checkToString(t, 2015, 8, 28, 3, 14, 15, tzAdelaide, +9, 30)
	checkToString(t, 2016, 2, 28, 3, 14, 15, tzAdelaide, +10, 30)
}

// mustLoadZone loads a zone or fails the test.
func mustLoadZone(t *testing.T, name string) *time.Location {
	t.Helper()
	zone, ok := loadZone(name)
	if !ok {
		t.Fatalf("no time zone %q", name)
	}
	return zone
}

// checkParseTZ is the private helper of the same name.
func checkParseTZ(t *testing.T, expect int, src string) {
	t.Helper()
	dest := newGreg()
	parsed, _ := parseTZOffset([]rune(src), dest, &parsePosition{})
	_, offsetSeconds := parsed.Zone()
	if offsetSeconds*1000 != expect {
		t.Errorf("parseTZOffset(%q) = %d, want %d", src, offsetSeconds*1000, expect)
	}
}

func TestParseTZ(t *testing.T) {
	// 1st parameter is what to expect
	checkParseTZ(t, 0*hrs+0*mins, "+00:00")
	checkParseTZ(t, 0*hrs+0*mins, "-0000")
	checkParseTZ(t, 1*hrs+0*mins, "+1:00")
	checkParseTZ(t, -(1*hrs + 0*mins), "-1:00")
	checkParseTZ(t, -(1*hrs + 30*mins), "-0130")
	checkParseTZ(t, 11*hrs+59*mins, "1159")
	checkParseTZ(t, 12*hrs+30*mins, "1230")
	checkParseTZ(t, -(12*hrs + 30*mins), "-12:30")
	checkParseTZ(t, 0*hrs+0*mins, "Z")
	checkParseTZ(t, -(8*hrs + 0*mins), "PST")
	checkParseTZ(t, 0*hrs+0*mins, "EDT") // EDT does not parse
	checkParseTZ(t, -(3*hrs + 0*mins), "GMT-0300")
	checkParseTZ(t, +(11*hrs + 0*mins), "GMT+11:00")
	checkParseTZ(t, -(6*hrs + 0*mins), "America/Chicago")
	checkParseTZ(t, +(3*hrs + 0*mins), "Europe/Moscow")
	checkParseTZ(t, +(9*hrs + 30*mins), "Australia/Adelaide")
	checkParseTZ(t, 5*hrs+0*mins, "0500")
	checkParseTZ(t, 5*hrs+0*mins, "+0500")
	checkParseTZ(t, 11*hrs+0*mins, "+11'00'")
	checkParseTZ(t, 0, "Z")
	// PDFBOX-3315, PDFBOX-2420
	checkParseTZ(t, 12*hrs+0*mins, "+12:00")
	checkParseTZ(t, -(12*hrs + 0*mins), "-12:00")
	checkParseTZ(t, 14*hrs+0*mins, "1400")
	checkParseTZ(t, -(14*hrs + 0*mins), "-1400")
}

// checkFormatOffset is the private helper of the same name.
func checkFormatOffset(t *testing.T, off float64, expect string) {
	t.Helper()
	if got := formatTZOffset(int64(off*60*60*1000), ":"); got != expect {
		t.Errorf("formatTZOffset(%v) = %q, want %q", off, got, expect)
	}
}

func TestFormatTZoffset(t *testing.T) {
	// 2nd parameter is what to expect
	checkFormatOffset(t, -12.1, "-12:06")
	checkFormatOffset(t, 12.1, "+12:06")
	checkFormatOffset(t, 0, "+00:00")
	checkFormatOffset(t, -1, "-01:00")
	checkFormatOffset(t, .5, "+00:30")
	checkFormatOffset(t, -0.5, "-00:30")
	checkFormatOffset(t, .1, "+00:06")
	checkFormatOffset(t, -0.1, "-00:06")
	checkFormatOffset(t, -12, "-12:00")
	checkFormatOffset(t, 12, "+12:00")
	checkFormatOffset(t, -11.5, "-11:30")
	checkFormatOffset(t, 11.5, "+11:30")
	checkFormatOffset(t, 11.9, "+11:54")
	checkFormatOffset(t, 11.1, "+11:06")
	checkFormatOffset(t, -11.9, "-11:54")
	checkFormatOffset(t, -11.1, "-11:06")
	// PDFBOX-2420
	checkFormatOffset(t, 14, "+14:00")
	checkFormatOffset(t, -14, "-14:00")
}

// TestSecondsBeyond59AreRefused pins a defect the slice 8 review feedback
// found. It is not a port: PDFBox has no test for it. It asserts what the Java
// does.
//
// parseBigEndianDate builds its result on a GregorianCalendar with leniency
// off, whose maximum for SECOND is 59 -- the leap-second range belongs to
// java.util.Date, not to Calendar -- so getTimeInMillis throws
// IllegalArgumentException for a second of 60 or 61 and the parse answers null.
// The port accepted up to 61 and handed them to time.Date, which normalises
// rather than refusing, so "D:20200101120060Z" came back as 12:01:00: a
// malformed date read as a different valid instant.
func TestSecondsBeyond59AreRefused(t *testing.T) {
	for _, text := range []string{
		"D:20200101120060Z",
		"D:20200101120061Z",
	} {
		if got, ok := ToCalendar(text); ok {
			t.Errorf("ToCalendar(%q) = %v, want it refused", text, got.Format(time.RFC3339))
		}
	}
	// 59 is still accepted, and is not normalised.
	got, ok := ToCalendar("D:20200101120059Z")
	if !ok {
		t.Fatal(`ToCalendar("D:20200101120059Z") was refused`)
	}
	if got.Second() != 59 || got.Minute() != 0 || got.Hour() != 12 {
		t.Errorf("ToCalendar = %v, want 12:00:59", got.Format(time.RFC3339))
	}
}
