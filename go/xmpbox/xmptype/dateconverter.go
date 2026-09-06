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

// ToISO8601Millis converts a date to its ISO 8601 string, printMillis saying
// whether the milliseconds are written.
//
// Port of toISO8601(Calendar, boolean).
func ToISO8601Millis(cal time.Time, printMillis bool) string {
	var out strings.Builder
	fmt.Fprintf(&out, "%04d", cal.Year())
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

// isoLayouts are the forms fromISO8601 accepts, in the order it tries them.
//
// Java builds a DateTimeFormatter of ISO_LOCAL_DATE_TIME plus a lenient
// "+HH:MM" or "Z" offset, and falls back to ISO_LOCAL_DATE_TIME alone read as
// UTC. ISO_LOCAL_DATE_TIME makes the seconds and the fraction optional, which
// is why there is a layout per shape rather than one.
var isoLayouts = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04Z07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04-07:00",
}

// localIsoLayouts are the same forms without an offset, which Java reads as
// UTC.
var localIsoLayouts = []string{
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
}

// fromISO8601 parses the ISO 8601 form.
//
// Port of the private fromISO8601, whose DateTimeParseException becomes an
// error.
func fromISO8601(dateString string) (time.Time, error) {
	for _, layout := range isoLayouts {
		if parsed, err := time.Parse(layout, dateString); err == nil {
			return parsed, nil
		}
	}
	for _, layout := range localIsoLayouts {
		if parsed, err := time.ParseInLocation(layout, dateString, time.UTC); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errors.New("Text '" + dateString + "' could not be parsed")
}
