package xmptype

// Dates, to and from the strings XMP writes them as.
//
// Port of org.apache.xmpbox.DateConverter, which Java declares final with a
// private constructor. It is declared here rather than in the root package for
// the reason the package comment gives; xmpbox aliases it back.
//
// Java's Calendar becomes a time.Time. A Calendar carries a TimeZone and a
// Calendar built by toCalendar carries a SimpleTimeZone of the parsed offset;
// a time.Time carries a *time.Location, and a fixed-offset zone stands in for
// the SimpleTimeZone. updateZoneId, which names that zone "GMT+HH:MM", is the
// name the Location is given.

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// isoDateTime matches the ISO 8601 form toCalendar hands to fromISO8601.
var isoDateTime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T.*`)

// IsBlankDate reports whether a date string is the one toCalendar answers null
// for, which a caller that has to tell that apart from a date asks.
func IsBlankDate(date string) bool { return date == "" || strings.TrimSpace(date) == "" }

// ToCalendar converts a string to a date, and answers the zero time where the
// string is empty or blank, which is Java's null.
//
// Port of toCalendar(String), whose IOException becomes an error. A caller that
// must tell Java's null apart from a date at the epoch tests the string with
// IsBlankDate first.
func ToCalendar(date string) (time.Time, error) {
	if IsBlankDate(date) {
		return time.Time{}, nil
	}
	date = strings.TrimSpace(date)

	// these are the default values
	month := 1
	day := 1
	hour := 0
	minute := 0
	second := 0

	// first string off the prefix if it exists
	var zone *time.Location
	if isoDateTime.MatchString(date) {
		parsed, err := fromISO8601(date)
		if err != nil {
			return time.Time{}, fmt.Errorf("xmptype: %w", err)
		}
		return parsed, nil
	}
	date = strings.TrimPrefix(date, "D:")

	posOfT := strings.IndexByte(date, 'T')
	if posOfT != 10 && posOfT != -1 {
		return time.Time{}, fmt.Errorf("Error converting date:%s", date)
	}
	date = strings.NewReplacer("-", "", ":", "", "T", "").Replace(date)
	if len(date) < 4 {
		return time.Time{}, fmt.Errorf("Error: Invalid date format '%s'", date)
	}

	year, err := atoi(date[0:4])
	if err != nil {
		return time.Time{}, numberError(date, err)
	}
	if len(date) >= 6 {
		if month, err = atoi(date[4:6]); err != nil {
			return time.Time{}, numberError(date, err)
		}
	}
	if len(date) >= 8 {
		if day, err = atoi(date[6:8]); err != nil {
			return time.Time{}, numberError(date, err)
		}
	}
	if len(date) >= 10 {
		if hour, err = atoi(date[8:10]); err != nil {
			return time.Time{}, numberError(date, err)
		}
	}
	if len(date) >= 12 {
		if minute, err = atoi(date[10:12]); err != nil {
			return time.Time{}, numberError(date, err)
		}
	}
	timeZonePos := 12
	if len(date) == 14 || len(date)-12 > 5 || (len(date)-12 == 3 && strings.HasSuffix(date, "Z")) {
		if second, err = atoi(date[12:14]); err != nil {
			return time.Time{}, numberError(date, err)
		}
		timeZonePos = 14
	}
	if len(date) >= timeZonePos+1 {
		sign := date[timeZonePos]
		if sign == 'Z' {
			zone = fixedZone(0)
		} else {
			hours := 0
			minutes := 0
			if len(date) >= timeZonePos+3 {
				if sign == '+' {
					// parseInt cannot handle the + sign
					if hours, err = atoi(date[timeZonePos+1 : timeZonePos+3]); err != nil {
						return time.Time{}, numberError(date, err)
					}
				} else {
					if hours, err = atoi(date[timeZonePos : timeZonePos+2]); err != nil {
						return time.Time{}, numberError(date, err)
					}
					hours = -hours
				}
			}
			if sign == '+' {
				if len(date) >= timeZonePos+5 {
					if minutes, err = atoi(date[timeZonePos+3 : timeZonePos+5]); err != nil {
						return time.Time{}, numberError(date, err)
					}
				}
			} else {
				if len(date) >= timeZonePos+4 {
					if minutes, err = atoi(date[timeZonePos+2 : timeZonePos+4]); err != nil {
						return time.Time{}, numberError(date, err)
					}
				}
			}
			zone = fixedZone(hours*60*60 + minutes*60)
		}
	}

	// Java builds a GregorianCalendar in the default time zone where the string
	// named none, clears it and sets the six fields; time.Date in time.Local is
	// the same thing.
	if zone == nil {
		zone = time.Local
	}
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, zone), nil
}

// atoi is Integer.parseInt, whose NumberFormatException toCalendar catches.
func atoi(s string) (int, error) { return strconv.Atoi(s) }

// numberError is the IOException toCalendar wraps a NumberFormatException in.
func numberError(date string, err error) error {
	return fmt.Errorf("Error converting date:%s: %w", date, err)
}

// fixedZone returns the zone of the given offset in seconds, named the way
// updateZoneId names a SimpleTimeZone.
//
// Port of the private updateZoneId, which Java calls for its side effect on the
// TimeZone; a time.Location is immutable, so the name is given when it is made.
func fixedZone(offsetSeconds int) *time.Location {
	offset := offsetSeconds * 1000
	pm := byte('+')
	if offset < 0 {
		pm = '-'
		offset = -offset
	}
	hh := offset / 3600000
	mm := offset % 3600000 / 60000

	var name string
	switch {
	case offset == 0:
		name = "GMT"
	case pm == '+' && hh <= 12:
		name = fmt.Sprintf("GMT+%02d:%02d", hh, mm)
	case pm == '-' && hh <= 14:
		name = fmt.Sprintf("GMT-%02d:%02d", hh, mm)
	default:
		name = "unknown"
	}
	return time.FixedZone(name, offsetSeconds)
}

// ToISO8601 converts a date to its ISO 8601 string, without milliseconds.
//
// Port of toISO8601(Calendar).
func ToISO8601(cal time.Time) string { return ToISO8601Millis(cal, false) }

// eraYear returns the year Calendar.get(Calendar.YEAR) answers, which is what
// toISO8601 writes.
//
// A Calendar counts within an era, so the year before 1 AD is 1 BC and the one
// before that is 2 BC; Go counts them 0 and -1. Java writes the era's number
// without saying which era it is. See PDFBOX-6107, whose case reads "0000-01-01"
// and expects "0001-01-01" back.
func eraYear(year int) int {
	if year <= 0 {
		return 1 - year
	}
	return year
}

// ToISO8601Millis converts a date to its ISO 8601 string, printMillis saying
// whether the milliseconds are written.
//
// Port of toISO8601(Calendar, boolean).
func ToISO8601Millis(cal time.Time, printMillis bool) string {
	var out strings.Builder
	fmt.Fprintf(&out, "%04d", eraYear(cal.Year()))
	out.WriteByte('-')
	fmt.Fprintf(&out, "%02d", int(cal.Month()))
	out.WriteByte('-')
	fmt.Fprintf(&out, "%02d", cal.Day())
	out.WriteByte('T')
	fmt.Fprintf(&out, "%02d", cal.Hour())
	out.WriteByte(':')
	fmt.Fprintf(&out, "%02d", cal.Minute())
	out.WriteByte(':')
	fmt.Fprintf(&out, "%02d", cal.Second())
	if printMillis {
		out.WriteByte('.')
		fmt.Fprintf(&out, "%03d", cal.Nanosecond()/1000000)
	}

	// Java adds the raw zone offset and the daylight saving offset, which
	// together are what a time.Time's zone offset already is.
	_, offsetSeconds := cal.Zone()
	timeZone := offsetSeconds * 1000
	if timeZone < 0 {
		out.WriteByte('-')
	} else {
		out.WriteByte('+')
	}
	if timeZone < 0 {
		timeZone = -timeZone
	}
	// milliseconds/1000 = seconds; seconds / 60 = minutes; minutes/60 = hours
	hours := timeZone / 1000 / 60 / 60
	minutes := (timeZone - (hours * 1000 * 60 * 60)) / 1000 / 60
	if hours < 10 {
		out.WriteByte('0')
	}
	fmt.Fprintf(&out, "%d", hours)
	out.WriteByte(':')
	if minutes < 10 {
		out.WriteByte('0')
	}
	fmt.Fprintf(&out, "%d", minutes)
	return out.String()
}

// localIsoLayouts are the forms ISO_LOCAL_DATE_TIME reads. It makes the seconds
// and the fraction optional, which is why there is a layout per shape rather
// than one.
var localIsoLayouts = []string{
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
}

// maxZoneOffsetSeconds is the largest offset from UTC a zone may have.
//
// Port of the range java.time.ZoneOffset accepts, which is what refuses
// "2008-12-31T19:48:30+19:00".
const maxZoneOffsetSeconds = 18 * 3600

// isZoneOffset reports whether an offset from UTC is one a zone may have.
func isZoneOffset(offset int) bool {
	return offset >= -maxZoneOffsetSeconds && offset <= maxZoneOffsetSeconds
}

// clampDayOfMonth returns the date string with a day of month beyond the length
// of its month brought back to the last day of that month.
//
// Java's DateTimeFormatter resolves with ResolverStyle.SMART, which is its
// default: a day of month within 1 to 31 but past the end of the month becomes
// the last day of the month, so "2015-02-30" reads as the 28th and
// "2015-04-31" as the 30th. A month outside 1 to 12, or a day outside 1 to 31,
// is still refused, and so is an hour, minute or second out of range. Go's
// parser refuses all of them, so the one relaxation is done here.
func clampDayOfMonth(dateString string) (string, bool) {
	if len(dateString) < 10 || dateString[4] != '-' || dateString[7] != '-' {
		return dateString, false
	}
	year, errYear := strconv.Atoi(dateString[0:4])
	month, errMonth := strconv.Atoi(dateString[5:7])
	day, errDay := strconv.Atoi(dateString[8:10])
	if errYear != nil || errMonth != nil || errDay != nil {
		return dateString, false
	}
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return dateString, false
	}
	// The zeroth day of the next month is the last day of this one.
	length := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day <= length {
		return dateString, false
	}
	return fmt.Sprintf("%s%02d%s", dateString[:8], length, dateString[10:]), true
}

// fromISO8601 parses the ISO 8601 form.
//
// Port of the private fromISO8601, whose DateTimeParseException becomes an
// error. Java parses with DATE_TIME_FORMATTER, which is ISO_LOCAL_DATE_TIME and
// then an offset, and falls back to ISO_LOCAL_DATE_TIME alone read as UTC; the
// fallback reads only a string with no offset, because the formatter wants the
// whole string read. The offset is read by lenientOffset, the way java.time
// reads it, rather than by Go's parser, which takes neither "+03" nor
// "+03:00:00" and takes "+03:60".
func fromISO8601(dateString string) (time.Time, error) {
	if clamped, wasClamped := clampDayOfMonth(dateString); wasClamped {
		dateString = clamped
	}
	local, offset := dateString, ""
	// Nothing ISO_LOCAL_TIME reads is a sign or a Z, so the first after the T
	// is where the offset starts.
	if len(dateString) > 11 {
		if at := strings.IndexAny(dateString[11:], "+-Zz"); at >= 0 {
			local, offset = dateString[:11+at], dateString[11+at:]
		}
	}
	zone := time.UTC
	offsetSeconds := 0
	if offset != "" {
		var ok bool
		if offsetSeconds, ok = lenientOffset(offset); !ok {
			return time.Time{}, errors.New("Text '" + dateString + "' could not be parsed")
		}
		zone = calendarZone(offsetSeconds)
	}
	for _, layout := range localIsoLayouts {
		if parsed, err := time.ParseInLocation(layout, local, time.UTC); err == nil {
			return parsed.Add(-time.Duration(offsetSeconds) * time.Second).In(zone), nil
		}
	}
	return time.Time{}, errors.New("Text '" + dateString + "' could not be parsed")
}

// lenientOffset reads an offset the way DATE_TIME_FORMATTER does, and answers
// it in seconds.
//
// Port of java.time's OffsetIdPrinterParser.parse for the pattern "+HH:MM" with
// "Z" for no offset, in lenient mode, which reads "+HH:MM" as "+HH:mm:ss": the
// hour is two digits and not past 23, the minutes and then the seconds may each
// be left off, each is two digits after a colon and not past 59, and "Z" is
// matched without regard to case because the formatter parses case
// insensitively. Whatever is left unread refuses the string, and so does an
// offset ZoneOffset will not take.
func lenientOffset(text string) (int, bool) {
	if strings.EqualFold(text, "Z") {
		return 0, true
	}
	if text[0] != '+' && text[0] != '-' {
		return 0, false
	}
	twoDigits := func(at int) (int, bool) {
		if at+2 > len(text) || text[at] < '0' || text[at] > '9' || text[at+1] < '0' || text[at+1] > '9' {
			return 0, false
		}
		value := int(text[at]-'0')*10 + int(text[at+1]-'0')
		return value, value <= 59
	}
	hours, ok := twoDigits(1)
	if !ok || hours > 23 {
		return 0, false
	}
	minutes, seconds, at := 0, 0, 3
	if at < len(text) && text[at] == ':' {
		if minutes, ok = twoDigits(at + 1); ok {
			at += 3
			if at < len(text) && text[at] == ':' {
				if seconds, ok = twoDigits(at + 1); ok {
					at += 3
				} else {
					seconds = 0
				}
			}
		} else {
			minutes = 0
		}
	}
	if at != len(text) {
		return 0, false
	}
	total := hours*3600 + minutes*60 + seconds
	if text[0] == '-' {
		total = -total
	}
	return total, isZoneOffset(total)
}

// calendarZone is the zone of the calendar Java builds from a parsed offset.
//
// GregorianCalendar.from asks TimeZone.getTimeZone for the zone the offset
// names: "UTC" for none, "GMT+03:00" for three hours. JDK 17's TimeZone reads
// no seconds in a custom zone, so for an offset with seconds, "GMT+03:00:30",
// it answers GMT; the instant is still the one the offset gave.
func calendarZone(offsetSeconds int) *time.Location {
	switch {
	case offsetSeconds == 0:
		return time.UTC
	case offsetSeconds%60 != 0:
		return time.FixedZone("GMT", 0)
	}
	sign, magnitude := '+', offsetSeconds
	if offsetSeconds < 0 {
		sign, magnitude = '-', -offsetSeconds
	}
	return time.FixedZone(fmt.Sprintf("GMT%c%02d:%02d", sign, magnitude/3600, magnitude%3600/60), offsetSeconds)
}
