package util

import (
	"strings"
	"time"
)

// parseWithFormat reads a date in one of DateConverter's fifteen patterns,
// starting at where and moving it over what it read.
//
// Port of what java.text.SimpleDateFormat.parse(String, ParsePosition) does for
// those patterns, which use the letters E, M, d, y, h, H, m, s, a and z and the
// quoted literal 'at'. A general SimpleDateFormat is far larger; this is the
// part of it DateConverter reaches, and it keeps the two rules that decide what
// the patterns mean: a numeric field directly followed by another numeric field
// reads exactly as many digits as the pattern has letters, and a two digit year
// is read against the eighty year window ending at the moment of the parse.
func parseWithFormat(text []rune, pattern string, where *parsePosition) (time.Time, bool) {
	tokens := patternTokens(pattern)
	fields := parsedFields{year: 1970, month: 1, day: 1}
	zone := time.UTC

	for i, token := range tokens {
		if where.index >= len(text) && token.numeric() {
			return time.Time{}, false
		}
		nextIsNumeric := i+1 < len(tokens) && tokens[i+1].numeric()
		if !token.parse(text, where, &fields, &zone, nextIsNumeric) {
			return time.Time{}, false
		}
	}

	if fields.hasAMPM {
		// hh is one to twelve, which the marker turns into hours of the day
		hour := fields.hour % 12
		if fields.pm {
			hour += 12
		}
		fields.hour = hour
	}
	date, ok := strictDate(fields.year, fields.month, fields.day,
		fields.hour, fields.minute, fields.second)
	if !ok {
		return time.Time{}, false
	}
	return adjustTimeZoneNicely(date, zone), true
}

// parsedFields are the calendar fields a pattern fills in, starting from the
// epoch defaults SimpleDateFormat.parse clears the calendar to.
type parsedFields struct {
	year   int
	month  int
	day    int
	hour   int
	minute int
	second int

	hasAMPM bool
	pm      bool
}

// patternToken is one letter run or literal run of a pattern.
type patternToken struct {
	letter rune
	count  int
	// literal holds the text of a literal token, whose letter is zero.
	literal string
}

// numeric reports whether the token reads digits.
func (t patternToken) numeric() bool {
	switch t.letter {
	case 'M':
		// three or more letters is a month name
		return t.count < 3
	case 'd', 'y', 'h', 'H', 'm', 's':
		return true
	}
	return false
}

// patternTokens splits a pattern into its letter runs and literals.
func patternTokens(pattern string) []patternToken {
	tokens := []patternToken{}
	runes := []rune(pattern)
	for i := 0; i < len(runes); {
		r := runes[i]
		switch {
		case r == '\'':
			// a quoted literal, in which two apostrophes stand for one
			literal := []rune{}
			i++
			for i < len(runes) {
				if runes[i] == '\'' {
					if i+1 < len(runes) && runes[i+1] == '\'' {
						literal = append(literal, '\'')
						i += 2
						continue
					}
					i++
					break
				}
				literal = append(literal, runes[i])
				i++
			}
			tokens = append(tokens, patternToken{literal: string(literal)})
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			count := 0
			for i+count < len(runes) && runes[i+count] == r {
				count++
			}
			tokens = append(tokens, patternToken{letter: r, count: count})
			i += count
		default:
			literal := []rune{}
			for i < len(runes) {
				c := runes[i]
				if c == '\'' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
					break
				}
				literal = append(literal, c)
				i++
			}
			tokens = append(tokens, patternToken{literal: string(literal)})
		}
	}
	return tokens
}

// monthNames and dayNames are the English names SimpleDateFormat reads with
// Locale.ENGLISH.
var (
	monthNames = []string{"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	monthAbbreviations = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	dayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday",
		"Friday", "Saturday"}
	dayAbbreviations = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
)

// parse reads what this token asks for, reporting whether it was there.
func (t patternToken) parse(text []rune, where *parsePosition, fields *parsedFields,
	zone **time.Location, nextIsNumeric bool) bool {
	if t.letter == 0 {
		// a literal, which SimpleDateFormat matches exactly
		return skipString(text, t.literal, where)
	}

	switch t.letter {
	case 'E':
		return matchName(text, where, dayNames, dayAbbreviations) >= 0
	case 'M':
		if t.count >= 3 {
			month := matchName(text, where, monthNames, monthAbbreviations)
			if month < 0 {
				return false
			}
			fields.month = month + 1
			return true
		}
		value, ok := parseNumber(text, where, t.count, nextIsNumeric)
		fields.month = value
		return ok
	case 'd':
		value, ok := parseNumber(text, where, t.count, nextIsNumeric)
		fields.day = value
		return ok
	case 'y':
		start := where.index
		value, ok := parseNumber(text, where, t.count, nextIsNumeric)
		if !ok {
			return false
		}
		if where.index-start == 2 {
			// SimpleDateFormat reads exactly two digits against the eighty year
			// window that ends at the moment the format was made.
			value = twoDigitYear(value)
		}
		fields.year = value
		return true
	case 'h', 'H':
		value, ok := parseNumber(text, where, t.count, nextIsNumeric)
		fields.hour = value
		return ok
	case 'm':
		value, ok := parseNumber(text, where, t.count, nextIsNumeric)
		fields.minute = value
		return ok
	case 's':
		value, ok := parseNumber(text, where, t.count, nextIsNumeric)
		fields.second = value
		return ok
	case 'a':
		if skipStringFold(text, "AM", where) {
			fields.hasAMPM = true
			fields.pm = false
			return true
		}
		if skipStringFold(text, "PM", where) {
			fields.hasAMPM = true
			fields.pm = true
			return true
		}
		return false
	case 'z':
		return parseZoneName(text, where, zone)
	}
	return false
}

// parseNumber reads a run of digits, limited to count of them where the next
// field is numeric too.
func parseNumber(text []rune, where *parsePosition, count int, nextIsNumeric bool) (int, bool) {
	limit := len(text) - where.index
	if nextIsNumeric {
		limit = count
	}
	start := where.index
	value := parseTimeField(text, where, limit, -1)
	if where.index == start {
		return 0, false
	}
	return value, true
}

// twoDigitYear places a two digit year in the eighty year window that ends now,
// which is the default century of a SimpleDateFormat.
func twoDigitYear(value int) int {
	centuryStart := time.Now().Year() - 80
	year := centuryStart/100*100 + value
	if year < centuryStart {
		year += 100
	}
	return year
}

// matchName reads one of the given names, long form first, and answers its
// index, or -1.
func matchName(text []rune, where *parsePosition, names, abbreviations []string) int {
	for i, name := range names {
		if skipStringFold(text, name, where) {
			return i
		}
	}
	for i, name := range abbreviations {
		if skipStringFold(text, name, where) {
			return i
		}
	}
	return -1
}

// skipStringFold steps over the given text where it is next, ignoring case.
func skipStringFold(text []rune, victim string, where *parsePosition) bool {
	victimRunes := []rune(victim)
	if where.index+len(victimRunes) > len(text) {
		return false
	}
	if !strings.EqualFold(string(text[where.index:where.index+len(victimRunes)]), victim) {
		return false
	}
	where.index += len(victimRunes)
	return true
}

// parseZoneName reads a time zone name, which is what the pattern letter z asks
// for.
//
// Java looks the name up in the JRE's zone table, which knows the three letter
// ids as well as the region ones; Go's LoadLocation knows the region ones and
// UTC, so a name it does not know fails the parse here where Java would have
// taken it.
func parseZoneName(text []rune, where *parsePosition, zone **time.Location) bool {
	start := where.index
	end := start
	for end < len(text) {
		r := text[end]
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '/' || r == '_' ||
			r == '+' || r == '-' || r == ':' || (r >= '0' && r <= '9') {
			end++
			continue
		}
		break
	}
	if end == start {
		return false
	}
	name := string(text[start:end])
	if name == "GMT" || name == "UTC" || name == "Z" {
		*zone = time.UTC
		where.index = end
		return true
	}
	loaded, known := loadZone(name)
	if !known {
		return false
	}
	*zone = loaded
	where.index = end
	return true
}
