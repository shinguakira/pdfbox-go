package ttf

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"testing"
)

// Port of org.apache.fontbox.ttf.TrueTypeFontCollectionTest.

func TestMissingTtcHeader(t *testing.T) {
	_, err := NewTrueTypeCollection(bytes.NewReader(make([]byte, 4)))
	if err == nil {
		t.Fatal("Missing ttc header not detected!")
	}
	if got, want := err.Error(), "Missing TTC header"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

func TestNumberOfFonts(t *testing.T) {
	payload := []byte{0x74, 0x74, 0x63, 0x66, 0x00, 0x00, 0x00, 0x00, 0x7F, 0xFF, 0xFF, 0xFF}
	_, err := NewTrueTypeCollection(bytes.NewReader(payload))
	if err == nil {
		t.Fatal("Invalid number of fonts not detected!")
	}
	if got, want := err.Error(), "Invalid number of fonts 2147483647"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// TestOnWindows is the Java parameterised test, whose cases are the three
// collections Windows ships.
func TestOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the Java test is @EnabledOnOs(OS.WINDOWS)")
	}
	cases := []struct {
		filename string
		listText string
	}{
		{"c:/windows/fonts/mingliu.ttc", "[MingLiU, PMingLiU, Ming-Lt-HKSCS-UNI-H]"},
		{"c:/windows/fonts/msmincho.ttc", "[MS-Mincho, MS-PMincho]"},
		{"c:/windows/fonts/simsun.ttc", "[SimSun, NSimSun]"},
	}
	for _, c := range cases {
		t.Run(c.filename, func(t *testing.T) {
			checkTrueTypeCollection(t, c.filename, c.listText)
		})
	}
}

// TestOnMac is the Java parameterised test for the two collections macOS
// ships.
func TestOnMac(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the Java test is @EnabledOnOs(OS.MAC)")
	}
	cases := []struct {
		filename string
		listText string
	}{
		{"/System/Library/Fonts/LucidaGrande.ttc",
			"[LucidaGrande, LucidaGrande-Bold, .LucidaGrandeUI, .LucidaGrandeUI-Bold]"},
		{"/System/Library/Fonts/Avenir.ttc",
			"[Avenir-Book, Avenir-BookOblique, Avenir-Black, Avenir-BlackOblique, " +
				"Avenir-Heavy, Avenir-HeavyOblique, Avenir-Light, Avenir-LightOblique, " +
				"Avenir-Medium, Avenir-MediumOblique, Avenir-Oblique, Avenir-Roman]"},
	}
	for _, c := range cases {
		t.Run(c.filename, func(t *testing.T) {
			checkTrueTypeCollection(t, c.filename, c.listText)
		})
	}
}

// test with https://raw.githubusercontent.com/notofonts/noto-cjk/main/Sans/OTC/NotoSansCJK-Regular.ttc
// could be possible, but that one is 19MB

func checkTrueTypeCollection(t *testing.T, filename, expected string) {
	t.Helper()
	if _, err := os.Stat(filename); err != nil {
		t.Skipf("the system font collection is not present: %v", err)
	}
	ttc, err := NewTrueTypeCollectionFile(filename)
	if err != nil {
		t.Fatalf("opening %s: %v", filename, err)
	}
	var list []string
	err = ttc.ProcessAllFonts(func(font *TrueTypeFont) error {
		name, err := font.Name()
		if err != nil {
			return err
		}
		list = append(list, name)
		byName, err := ttc.FontByName(name)
		if err != nil {
			return err
		}
		defer byName.Close()
		byNameName, err := byName.Name()
		if err != nil {
			return err
		}
		if byNameName != name {
			t.Errorf("FontByName(%s) is named %q", name, byNameName)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ProcessAllFonts: %v", err)
	}
	if got := listString(list); got != expected {
		t.Errorf("fonts = %s, want %s", got, expected)
	}
	ttc.Close()

	var headerList []string
	if err := ProcessAllFontHeaders(filename, func(fontHeaders *FontHeaders) {
		headerList = append(headerList, fontHeaders.Name())
	}); err != nil {
		t.Fatalf("ProcessAllFontHeaders: %v", err)
	}
	if got := listString(headerList); got != expected {
		t.Errorf("header fonts = %s, want %s", got, expected)
	}
}

// listString renders a list the way Java's List.toString does.
func listString(list []string) string {
	return "[" + strings.Join(list, ", ") + "]"
}
