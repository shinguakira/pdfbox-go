package cmap

import (
	"reflect"
	"testing"
)

// Port of org.apache.fontbox.cmap.CMapStringsTest.
//
// Java's assertSame checks on the returned Strings and Integers are about
// interning: a Go string and a Go int are values with no observable identity,
// so those assertions collapse into the equality ones beside them. The
// assertSame checks on the byte arrays do carry over, and sameSlice makes them.

// sameSlice reports whether two slices share a backing array, which is what
// Java's assertSame asks of the cached byte arrays.
func sameSlice(a, b []byte) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

func TestGetNonCachedMappings(t *testing.T) {
	// arrays consisting of more than 2 bytes aren't cached.
	if _, ok := GetMapping([]byte{0, 0, 0}); ok {
		t.Error("a three-byte code was cached")
	}
	if _, ok := GetMapping([]byte{0, 0, 0, 0}); ok {
		t.Error("a four-byte code was cached")
	}
}

func TestGetMappingOneByte(t *testing.T) {
	cases := []struct {
		code []byte
		want string
	}{
		{[]byte{0}, "\x00"},
		{[]byte{0xff}, "ÿ"}, // ISO-8859-1 0xFF is U+00FF
		{[]byte{98}, "b"},
	}
	for _, c := range cases {
		got, ok := GetMapping(c.code)
		if !ok {
			t.Errorf("GetMapping(%v) was not cached", c.code)
			continue
		}
		if got != c.want {
			t.Errorf("GetMapping(%v) = %q, want %q", c.code, got, c.want)
		}
		// the same value every time
		again, _ := GetMapping(c.code)
		if again != got {
			t.Errorf("GetMapping(%v) gave %q then %q", c.code, got, again)
		}
	}
}

func TestGetMappingTwoByte(t *testing.T) {
	cases := []struct {
		code []byte
		want string
	}{
		{[]byte{0, 0}, "\x00"},    // UTF-16BE 0000 is U+0000
		{[]byte{0xff, 0xff}, "￿"}, // UTF-16BE FFFF is U+FFFF
		{[]byte{0x62, 0x43}, "扃"},
		{[]byte{0xff, 0x43}, "ｃ"},
		{[]byte{0x38, 0xff}, "㣿"},
	}
	for _, c := range cases {
		got, ok := GetMapping(c.code)
		if !ok {
			t.Errorf("GetMapping(%v) was not cached", c.code)
			continue
		}
		if got != c.want {
			t.Errorf("GetMapping(%v) = %q, want %q", c.code, got, c.want)
		}
		again, _ := GetMapping(c.code)
		if again != got {
			t.Errorf("GetMapping(%v) gave %q then %q", c.code, got, again)
		}
	}
}

func TestGetByteValuesOneByte(t *testing.T) {
	for _, code := range [][]byte{{0}, {0xff}, {98}} {
		first := GetByteValue(code)
		second := GetByteValue(code)
		if first == nil {
			t.Errorf("GetByteValue(%v) was not cached", code)
			continue
		}
		// the given values are the same objects
		if !sameSlice(first, second) {
			t.Errorf("GetByteValue(%v) gave two different slices", code)
		}
		// the cached value isn't the same object than the given one
		if sameSlice(code, first) {
			t.Errorf("GetByteValue(%v) handed back the caller's own slice", code)
		}
		if len(first) != len(code) || first[0] != code[0] {
			t.Errorf("GetByteValue(%v) = %v, want the same bytes", code, first)
		}
	}
}

func TestGetByteValuesTwoByte(t *testing.T) {
	for _, code := range [][]byte{{0, 0}, {0xff, 0xff}, {0x62, 0x43}, {0xff, 0x43}, {0x38, 0xff}} {
		first := GetByteValue(code)
		second := GetByteValue(code)
		if first == nil {
			t.Errorf("GetByteValue(%v) was not cached", code)
			continue
		}
		if !sameSlice(first, second) {
			t.Errorf("GetByteValue(%v) gave two different slices", code)
		}
		if sameSlice(code, first) {
			t.Errorf("GetByteValue(%v) handed back the caller's own slice", code)
		}
		if len(first) != len(code) || first[0] != code[0] || first[1] != code[1] {
			t.Errorf("GetByteValue(%v) = %v, want the same bytes", code, first)
		}
	}
}

func TestGetNonCachedByteValues(t *testing.T) {
	// arrays consisting of more than 2 bytes aren't cached.
	if GetByteValue([]byte{0, 0, 0}) != nil {
		t.Error("a three-byte code was cached")
	}
	if GetByteValue([]byte{0, 0, 0, 0}) != nil {
		t.Error("a four-byte code was cached")
	}
}

func TestGetIndexValues(t *testing.T) {
	cases := []struct {
		code []byte
		want int
	}{
		{[]byte{0}, 0},
		{[]byte{0xff}, 0xff},
		{[]byte{0, 0}, 0},
		{[]byte{0xff, 0xff}, 0xffff},
		{[]byte{0x62, 0x43}, 0x6243},
	}
	for _, c := range cases {
		got, ok := GetIndexValue(c.code)
		if !ok {
			t.Errorf("GetIndexValue(%v) was not cached", c.code)
			continue
		}
		if got != c.want {
			t.Errorf("GetIndexValue(%v) = %d, want %d", c.code, got, c.want)
		}
	}
	if _, ok := GetIndexValue([]byte{0, 0, 0}); ok {
		t.Error("a three-byte code was cached")
	}
}
