package util

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	_ "time/tzdata"
)

// The milliseconds a minute, an hour, half a day and a day hold.
//
// Port of the private constants of DateConverter.
const (
	minutesPerHour  = 60
	secondsPerMinut = 60
	millisPerMinute = secondsPerMinut * 1000
	millisPerHour   = minutesPerHour * millisPerMinute
	halfDay         = 12 * minutesPerHour * millisPerMinute
	fullDay         = 2 * halfDay
)

// alphaStartFormats are the formats tried for a date that starts with a letter.
//
// Port of the private ALPHA_START_FORMATS, with the comment on each.
var alphaStartFormats = []string{
	"EEEE, dd MMM yy hh:mm:ss a",
	"EEEE, MMM dd, yy hh:mm:ss a",
	"EEEE, MMM dd, yy 'at' hh:mma", // Acrobat Net Distiller 1.0 for Windows
	"EEEE, MMM dd, yy",             // Acrobat Distiller 1.0.2 for Macintosh && PDFBOX-465
	"EEEE MMM dd, yy HH:mm:ss",     // ECMP5
	"EEEE MMM dd HH:mm:ss z yy",    // GNU Ghostscript 7.0.7
	"EEEE MMM dd HH:mm:ss yy",      // GNU Ghostscript 7.0.7 variant
}

// digitStartFormats are the formats tried for a date that starts with a digit.
//
// Port of the private DIGIT_START_FORMATS.
var digitStartFormats = []string{
	"dd MMM yy HH:mm:ss", // for 26 May 2000 11:25:00
	"dd MMM yy HH:mm",    // for 26 May 2000 11:25
	"yyyy MMM d",         // ambiguity resolved only by omitting time
	"yyyymmddhh:mm:ss",   // test case "200712172:2:3"
	"H:m M/d/yy",         // test case "9:47 5/12/2008"
	"M/d/yy HH:mm:ss",
	"M/d/yy HH:mm",
	"M/d/yy",
}

// ToString formats the given time as a PDF date string.
//
// Port of the static DateConverter.toString(Calendar).
func ToString(cal time.Time) string {
	_, offsetSeconds := cal.Zone()
	offset := formatTZOffset(int64(offsetSeconds)*1000, "'")
	return fmt.Sprintf("D:%04d%02d%02d%02d%02d%02d%s'",
		cal.Year(), int(cal.Month()), cal.Day(),
		cal.Hour(), cal.Minute(), cal.Second(), offset)
}

// ToISO8601 formats the given time as an ISO 8601 date string.
//
// Port of the static DateConverter.toISO8601(Calendar).
func ToISO8601(cal time.Time) string {
	_, offsetSeconds := cal.Zone()
	offset := formatTZOffset(int64(offsetSeconds)*1000, ":")
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d%s",
		cal.Year(), int(cal.Month()), cal.Day(),
		cal.Hour(), cal.Minute(), cal.Second(), offset)
}

// restrainTZOffset brings a time zone offset into the range a PDF date can
// carry.
//
// Port of the private restrainTZoffset.
func restrainTZOffset(proposedOffset int64) int {
	if proposedOffset <= 14*millisPerHour && proposedOffset >= -14*millisPerHour {
		// https://www.w3.org/TR/xmlschema-2/#dateTime-timezones
		// Timezones between 14:00 and -14:00 are valid
		return int(proposedOffset)
	}
	// Constrain a timezone offset to the range [-11:59 thru +12:00].
	proposedOffset = ((proposedOffset+halfDay)%fullDay + fullDay) % fullDay
	if proposedOffset == 0 {
		return halfDay
	}
	// 0 <= proposedOffset < DAY
	proposedOffset = (proposedOffset - halfDay) % halfDay
	// -HALF_DAY < proposedOffset < HALF_DAY
	return int(proposedOffset)
}

// formatTZOffset writes a time zone offset as a sign, two hour digits, the
// given separator and two minute digits.
//
// Port of the package-private formatTZoffset.
func formatTZOffset(millis int64, sep string) string {
	offset := restrainTZOffset(millis)
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return fmt.Sprintf("%s%02d%s%02d", sign, offset/millisPerHour,
		sep, offset%millisPerHour/millisPerMinute)
}

// parsePosition is where a parse has reached, which Java passes as a
// java.text.ParsePosition.
type parsePosition struct {
	index int
}

// parseTimeField reads up to maxlen digits, and answers remedy where there are
// none.
//
// Port of the private parseTimeField.
func parseTimeField(text []rune, where *parsePosition, maxlen int, remedy int) int {
	if text == nil {
		return remedy
	}
	// it would seem that DecimalFormat.parse() would be simpler;
	// but that class blithely ignores setMaximumIntegerDigits
	retval := 0
	index := where.index
	limit := index + minInt(maxlen, len(text)-index)
	for ; index < limit; index++ {
		// convert digit to integer
		cval := int(text[index]) - '0'
		// test to see if we got a digit
		if cval < 0 || cval > 9 {
			// no digit at index
			break
		}
		// append the digit to the return value
		retval = retval*10 + cval
	}
	if index == where.index {
		return remedy
	}
	where.index = index
	return retval
}

// skipOptionals steps over any of the given characters, answering the last one
// that was not a space.
//
// Port of the private skipOptionals.
func skipOptionals(text []rune, where *parsePosition, optionals string) rune {
	retval := ' '
	for where.index < len(text) && strings.ContainsRune(optionals, text[where.index]) {
		if currch := text[where.index]; currch != ' ' {
			retval = currch
		}
		where.index++
	}
	return retval
}

// skipString steps over the given text where it is next, reporting whether it
// was.
//
// Port of the private skipString.
func skipString(text []rune, victim string, where *parsePosition) bool {
	victimRunes := []rune(victim)
	if where.index+len(victimRunes) <= len(text) &&
		string(text[where.index:where.index+len(victimRunes)]) == victim {
		where.index += len(victimRunes)
		return true
	}
	return false
}

// parseTZOffset reads a time zone and moves the given time into it.
//
// Port of the package-private parseTZoffset. Java hands back a GregorianCalendar
// whose fields are unchanged and whose zone has moved; the port answers the
// same instant in the new zone, which is what adjustTimeZoneNicely leaves.
func parseTZOffset(text []rune, cal time.Time, initialWhere *parsePosition) (time.Time, bool) {
	where := &parsePosition{index: initialWhere.index}
	var zone *time.Location
	sign := skipOptionals(text, where, "Z+- ")
	hadGMT := sign == 'Z' || skipString(text, "GMT", where) || skipString(text, "UTC", where)
	if hadGMT {
		sign = skipOptionals(text, where, "+- ")
	}
	tzHours := parseTimeField(text, where, 2, -999)
	skipOptionals(text, where, "': ")
	tzMin := parseTimeField(text, where, 2, 0)
	skipOptionals(text, where, "' ")

	if tzHours != -999 {
		// we parsed a time zone in default format
		hrSign := 1
		if sign == '-' {
			hrSign = -1
		}
		offset := restrainTZOffset(int64(hrSign) *
			(int64(tzHours)*millisPerHour + int64(tzMin)*millisPerMinute))
		zone = time.FixedZone(zoneID(offset), offset/1000)
	} else if !hadGMT {
		// try to process as a name; "GMT" or "UTC" has already been processed
		tzText := strings.TrimSpace(string(text[initialWhere.index:]))
		loaded, known := loadZone(tzText)
		if !known {
			// getTimeZone returns "GMT" for unknown ids
			// no timezone in text, cal and initialWhere are unchanged
			return cal, false
		}
		// we got a tz by name; use it
		zone = loaded
		where.index = len(text)
	} else {
		zone = time.UTC
	}

	// adjustTimeZoneNicely keeps the fields and moves the instant
	adjusted := adjustTimeZoneNicely(cal, zone)
	initialWhere.index = where.index
	return adjusted, true
}

// zoneID names a fixed offset the way updateZoneId does.
//
// Port of the private updateZoneId.
func zoneID(offset int) string {
	pm := "+"
	if offset < 0 {
		pm = "-"
		offset = -offset
	}
	hh := offset / 3600000
	mm := offset % 3600000 / 60000
	switch {
	case offset == 0:
		return "GMT"
	case pm == "+" && hh <= 12:
		return fmt.Sprintf("GMT+%02d:%02d", hh, mm)
	case pm == "-" && hh <= 14:
		return fmt.Sprintf("GMT-%02d:%02d", hh, mm)
	}
	return "unknown"
}

// adjustTimeZoneNicely reads the fields of cal as if they were in the given
// zone.
//
// Port of the private adjustTimeZoneNicely, which sets the zone and then
// subtracts its offset in minutes, leaving the wall clock reading unchanged.
func adjustTimeZoneNicely(cal time.Time, zone *time.Location) time.Time {
	return time.Date(cal.Year(), cal.Month(), cal.Day(), cal.Hour(), cal.Minute(),
		cal.Second(), cal.Nanosecond(), zone)
}

// newGreg returns the calendar a parse starts from: midnight UTC, with no
// leniency.
//
// Port of the package-private newGreg.
func newGreg() time.Time {
	return time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
}

// parseBigEndianDate reads a date written year first, which is what a PDF date
// is.
//
// Port of the private parseBigEndianDate.
func parseBigEndianDate(text []rune, initialWhere *parsePosition) (time.Time, bool) {
	where := &parsePosition{index: initialWhere.index}
	year := parseTimeField(text, where, 4, 0)
	if where.index != 4+initialWhere.index {
		return time.Time{}, false
	}
	skipOptionals(text, where, "/- ")
	month := parseTimeField(text, where, 2, 1)
	skipOptionals(text, where, "/- ")
	day := parseTimeField(text, where, 2, 1)
	skipOptionals(text, where, " T")
	hour := parseTimeField(text, where, 2, 0)
	skipOptionals(text, where, ": ")
	minute := parseTimeField(text, where, 2, 0)
	skipOptionals(text, where, ": ")
	second := parseTimeField(text, where, 2, 0)
	if nextC := skipOptionals(text, where, "."); nextC == '.' {
		// fractions of a second: skip up to 19 digits
		parseTimeField(text, where, 19, 0)
	}

	dest, ok := strictDate(year, month, day, hour, minute, second)
	if !ok {
		// Java's non-lenient calendar throws here
		slog.Debug("util: could not parse the date", slog.String("text", string(text)))
		return time.Time{}, false
	}
	initialWhere.index = where.index
	skipOptionals(text, initialWhere, " ")
	// dest has at least a year value
	return dest, true
}

// strictDate builds a date and reports whether the fields were in range, which
// is what a GregorianCalendar with leniency off answers.
func strictDate(year, month, day, hour, minute, second int) (time.Time, bool) {
	if month < 1 || month > 12 || day < 1 || day > 31 ||
		hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 61 {
		return time.Time{}, false
	}
	date := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		// the day was past the end of the month, which Java rejects
		return time.Time{}, false
	}
	return date, true
}

// parseSimpleDate tries the given formats in turn.
//
// Port of the private parseSimpleDate.
func parseSimpleDate(text []rune, fmts []string, initialWhere *parsePosition) (time.Time, bool) {
	for _, format := range fmts {
		where := &parsePosition{index: initialWhere.index}
		if retCal, ok := parseWithFormat(text, format, where); ok {
			initialWhere.index = where.index
			skipOptionals(text, initialWhere, " ")
			return retCal, true
		}
	}
	return time.Time{}, false
}

// parseDate reads a date in any of the formats DateConverter knows.
//
// Port of the private parseDate.
func parseDate(text []rune, initialWhere *parsePosition) (time.Time, bool) {
	trimmed := strings.TrimSpace(string(text))
	if len(text) == 0 || trimmed == "D:" {
		return time.Time{}, false
	}

	// remember longest date string
	longestLen := -999999
	var longestDate time.Time
	haveLongest := false

	where := &parsePosition{index: initialWhere.index}
	// check for null (throws exception) and trim off surrounding spaces
	skipOptionals(text, where, " ")
	startPosition := where.index

	// try big-endian parse
	retCal, ok := parseBigEndianDate(text, where)
	if ok {
		// check for success and a timezone
		zoned, hadZone := retCal, where.index == len(text)
		if !hadZone {
			zoned, hadZone = parseTZOffset(text, retCal, where)
		}
		if hadZone {
			// if text is fully consumed, return the date else remember it and its length
			whereLen := where.index
			if whereLen == len(text) {
				initialWhere.index = whereLen
				return zoned, true
			}
			longestLen = whereLen
			longestDate = zoned
			haveLongest = true
		}
	}

	// try one of the sets of standard formats
	where.index = startPosition
	formats := alphaStartFormats
	if startPosition < len(text) && text[startPosition] >= '0' && text[startPosition] <= '9' {
		formats = digitStartFormats
	}
	retCal, ok = parseSimpleDate(text, formats, where)
	if ok {
		// check for success and a timezone
		zoned, hadZone := retCal, where.index == len(text)
		if !hadZone {
			zoned, hadZone = parseTZOffset(text, retCal, where)
		}
		if hadZone {
			// if text is fully consumed, return the date else remember it and its length
			whereLen := where.index
			if whereLen == len(text) {
				initialWhere.index = whereLen
				return zoned, true
			}
			if whereLen > longestLen {
				longestLen = whereLen
				longestDate = zoned
				haveLongest = true
			}
		}
	}

	if haveLongest {
		initialWhere.index = longestLen
		return longestDate, true
	}
	return retCal, ok
}

// ToCalendar reads a date string, and reports false where it holds none, which
// is the null Java answers.
//
// Port of the static DateConverter.toCalendar(String).
func ToCalendar(text string) (time.Time, bool) {
	if strings.TrimSpace(text) == "" {
		return time.Time{}, false
	}
	runes := []rune(text)
	where := &parsePosition{}
	skipOptionals(runes, where, " ")
	skipString(runes, "D:", where)
	calendar, ok := parseDate(runes, where)
	if !ok || where.index != len(runes) {
		// the date string is invalid
		return time.Time{}, false
	}
	return calendar, true
}

// minInt is Math.min for two ints.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// javaLegacyZones are the three and four letter zone ids Java's
// TimeZone.getTimeZone knows and the IANA database does not.
//
// Java keeps a compatibility table of these; Go's LoadLocation reads the IANA
// database, which has only a few of them. The port names the ones Java maps, so
// that a date carrying "PST" reads the same in both.
var javaLegacyZones = map[string]string{
	"ACT":     "Australia/Darwin",
	"AET":     "Australia/Sydney",
	"AGT":     "America/Argentina/Buenos_Aires",
	"ART":     "Africa/Cairo",
	"AST":     "America/Anchorage",
	"BET":     "America/Sao_Paulo",
	"BST":     "Asia/Dhaka",
	"CAT":     "Africa/Harare",
	"CNT":     "America/St_Johns",
	"CST":     "America/Chicago",
	"CTT":     "Asia/Shanghai",
	"EAT":     "Africa/Addis_Ababa",
	"ECT":     "Europe/Paris",
	"IET":     "America/Indiana/Indianapolis",
	"IST":     "Asia/Kolkata",
	"JST":     "Asia/Tokyo",
	"MIT":     "Pacific/Apia",
	"NET":     "Asia/Yerevan",
	"NST":     "Pacific/Auckland",
	"PLT":     "Asia/Karachi",
	"PNT":     "America/Phoenix",
	"PRT":     "America/Puerto_Rico",
	"PST":     "America/Los_Angeles",
	"SST":     "Pacific/Guadalcanal",
	"VST":     "Asia/Ho_Chi_Minh",
	"EST5EDT": "America/New_York",
	"CST6CDT": "America/Chicago",
	"MST7MDT": "America/Denver",
	"PST8PDT": "America/Los_Angeles",
}

// loadZone returns the zone with the given id, the way
// TimeZone.getTimeZone(String) does, and reports false where the id is unknown
// -- where Java answers GMT and its caller treats that as unknown.
func loadZone(name string) (*time.Location, bool) {
	switch name {
	case "GMT", "UTC", "UT", "Z":
		return time.UTC, true
	case "EST":
		// Java maps these three to a fixed offset with no daylight saving
		return time.FixedZone("EST", -5*3600), true
	case "MST":
		return time.FixedZone("MST", -7*3600), true
	case "HST":
		return time.FixedZone("HST", -10*3600), true
	}
	if custom, isCustom := customGMTZone(name); isCustom {
		return custom, true
	}
	if mapped, isLegacy := javaLegacyZones[name]; isLegacy {
		name = mapped
	}
	loaded, err := time.LoadLocation(name)
	if err != nil {
		return nil, false
	}
	return loaded, true
}

// customGMTZone reads a zone written as GMT and an offset, which is the custom
// time zone syntax TimeZone.getTimeZone understands: GMT, a sign, one or two
// hour digits, and optionally a colon and two minute digits.
func customGMTZone(name string) (*time.Location, bool) {
	if !strings.HasPrefix(name, "GMT") || len(name) < 5 {
		return nil, false
	}
	rest := []rune(name[3:])
	sign := 1
	switch rest[0] {
	case '+':
	case '-':
		sign = -1
	default:
		return nil, false
	}
	where := &parsePosition{index: 1}
	hours := parseTimeField(rest, where, 2, -999)
	if hours == -999 {
		return nil, false
	}
	minutes := 0
	if where.index < len(rest) {
		if rest[where.index] != ':' {
			return nil, false
		}
		where.index++
		minutes = parseTimeField(rest, where, 2, -999)
		if minutes == -999 || where.index != len(rest) {
			return nil, false
		}
	}
	if hours > 23 || minutes > 59 {
		return nil, false
	}
	offset := sign * (hours*3600 + minutes*60)
	return time.FixedZone(name, offset), true
}
