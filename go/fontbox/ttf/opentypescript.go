package ttf

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/shinguakira/pdfbox-go/go/fontbox/resources"
)

// The three script names OpenTypeScript hands out itself.
const (
	// ScriptInherited says the codepoint's script can only be determined by its
	// context.
	ScriptInherited = "Inherited"

	// ScriptUnknown is the script of a codepoint no range covers.
	ScriptUnknown = "Unknown"

	// TagDefault is the OpenType tag of the default script.
	TagDefault = "DFLT"
)

// The Unicode ranges the Scripts.txt file gives, read once when the package
// loads.
//
// Port of the two static arrays of org.apache.fontbox.ttf.OpenTypeScript.
var (
	unicodeRangeStarts  []int
	unicodeRangeScripts []string
)

func init() {
	data, err := resources.Read("unicode/Scripts.txt")
	if err != nil {
		slog.Warn("Could not parse Scripts.txt, mirroring char map will be empty", "err", err)
		return
	}
	if err := parseScriptsFile(data); err != nil {
		slog.Warn("Could not parse Scripts.txt, mirroring char map will be empty", "err", err)
	}
}

// scriptRange is one row of the parsed file, kept so that the ranges can be put
// in order the way Java's TreeMap does.
type scriptRange struct {
	start  int
	end    int
	script string
}

func parseScriptsFile(data []byte) error {
	var unicodeRanges []scriptRange
	// Java keys a TreeMap by the range start, so a repeated start replaces the
	// row that was there; index remembers where each start sits.
	index := map[int]int{}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lastRange := [2]int{-2147483648, -2147483648}
	lastScript := ""
	for scanner.Scan() {
		s := scanner.Text()

		// ignore comments
		if comment := strings.IndexByte(s, '#'); comment != -1 {
			s = s[:comment]
		}

		if len(s) < 2 {
			continue
		}

		fields := splitTokens(s, ";")
		if len(fields) < 2 {
			continue
		}
		characters := strings.TrimSpace(fields[0])
		script := strings.TrimSpace(fields[1])
		var start, end int
		rangeDelim := strings.Index(characters, "..")
		if rangeDelim == -1 {
			value, err := strconv.ParseInt(characters, 16, 64)
			if err != nil {
				return err
			}
			start = int(value)
			end = start
		} else {
			first, err := strconv.ParseInt(characters[:rangeDelim], 16, 64)
			if err != nil {
				return err
			}
			last, err := strconv.ParseInt(characters[rangeDelim+2:], 16, 64)
			if err != nil {
				return err
			}
			start = int(first)
			end = int(last)
		}
		if start == lastRange[1]+1 && script == lastScript {
			// Combine with previous range
			lastRange[1] = end
			unicodeRanges[index[lastRange[0]]].end = end
			continue
		}
		if at, seen := index[start]; seen {
			unicodeRanges[at] = scriptRange{start: start, end: end, script: script}
		} else {
			index[start] = len(unicodeRanges)
			unicodeRanges = append(unicodeRanges, scriptRange{
				start: start, end: end, script: script,
			})
		}
		lastRange = [2]int{start, end}
		lastScript = script
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	sort.Slice(unicodeRanges, func(i, j int) bool {
		return unicodeRanges[i].start < unicodeRanges[j].start
	})
	unicodeRangeStarts = make([]int, len(unicodeRanges))
	unicodeRangeScripts = make([]string, len(unicodeRanges))
	for i, r := range unicodeRanges {
		unicodeRangeStarts[i] = r.start
		unicodeRangeScripts[i] = r.script
	}
	return nil
}

// splitTokens is Java's StringTokenizer over one delimiter, which skips empty
// tokens.
func splitTokens(s, delim string) []string {
	var out []string
	for _, part := range strings.Split(s, delim) {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// unicodeScriptOf obtains the Unicode script associated with the given Unicode
// codepoint, or ScriptUnknown if unknown.
func unicodeScriptOf(codePoint int) string {
	ensureValidCodePoint(codePoint)
	if !isAssignedCodePoint(codePoint) {
		return ScriptUnknown
	}
	// Java's binarySearch gives -(insertion point) - 1 for a miss, and the
	// script wanted is the range that starts before the codepoint.
	scriptIndex := sort.SearchInts(unicodeRangeStarts, codePoint)
	if scriptIndex >= len(unicodeRangeStarts) || unicodeRangeStarts[scriptIndex] != codePoint {
		scriptIndex--
	}
	if scriptIndex < 0 || scriptIndex >= len(unicodeRangeScripts) {
		return ScriptUnknown
	}
	return unicodeRangeScripts[scriptIndex]
}

// isAssignedCodePoint is Java's `Character.getType(codePoint) != UNASSIGNED`.
func isAssignedCodePoint(codePoint int) bool {
	return unicode.IsGraphic(rune(codePoint)) || unicode.IsControl(rune(codePoint)) ||
		unicode.In(rune(codePoint), unicode.Cf, unicode.Co, unicode.Cs)
}

// GetScriptTags obtains the OpenType script tags associated with the given
// Unicode codepoint.
//
// The result may contain the special value ScriptInherited, which indicates
// that the codepoint's script can only be determined by its context. Unknown
// codepoints are mapped to TagDefault.
func GetScriptTags(codePoint int) []string {
	ensureValidCodePoint(codePoint)
	return unicodeScriptToOpenTypeTag[unicodeScriptOf(codePoint)]
}

// ensureValidCodePoint panics where Java throws IllegalArgumentException, which
// is unchecked.
func ensureValidCodePoint(codePoint int) {
	if codePoint < 0 || codePoint > 0x10FFFF {
		panic(fmt.Sprintf("Invalid codepoint: %d", codePoint))
	}
}
