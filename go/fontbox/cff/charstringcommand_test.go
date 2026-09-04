package cff

import "testing"

// Port of org.apache.fontbox.cff.CharStringCommandTest.

func TestValue(t *testing.T) {
	cases := []struct {
		command *CharStringCommand
		want    int
		name    string
	}{
		{CmdHSTEM, 1, "HSTEM"},
		{CmdESCAPE, 12, "ESCAPE"},
		{CmdDOTSECTION, (12 << 4) + 0, "DOTSECTION"},
		{CmdAND, (12 << 4) + 3, "AND"},
		{CmdHSBW, 13, "HSBW"},
	}
	for _, c := range cases {
		if got := c.command.Value(); got != c.want {
			t.Errorf("%s.Value() = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestCharStringCommand(t *testing.T) {
	charStringCommand1 := GetInstance(1)
	if got := charStringCommand1.Type1KeyWord(); got != Type1HSTEM {
		t.Errorf("GetInstance(1).Type1KeyWord() = %v, want %v", got, Type1HSTEM)
	}
	if got := charStringCommand1.Type2KeyWord(); got != Type2HSTEM {
		t.Errorf("GetInstance(1).Type2KeyWord() = %v, want %v", got, Type2HSTEM)
	}
	if got := charStringCommand1.String(); got != "HSTEM|" {
		t.Errorf("GetInstance(1).String() = %q, want %q", got, "HSTEM|")
	}

	charStringCommand120 := GetInstance2(12, 0)
	if got := charStringCommand120.Type1KeyWord(); got != Type1DOTSECTION {
		t.Errorf("GetInstance(12, 0).Type1KeyWord() = %v, want %v", got, Type1DOTSECTION)
	}
	if got := charStringCommand120.Type2KeyWord(); got != Type2KeyWordNone {
		t.Errorf("GetInstance(12, 0).Type2KeyWord() = %v, want none", got)
	}
	if got := charStringCommand120.String(); got != "DOTSECTION|" {
		t.Errorf("GetInstance(12, 0).String() = %q, want %q", got, "DOTSECTION|")
	}

	values123 := []int{12, 3}
	charStringCommand123 := GetInstanceValues(values123)
	if got := charStringCommand123.Type1KeyWord(); got != Type1KeyWordNone {
		t.Errorf("GetInstance([12 3]).Type1KeyWord() = %v, want none", got)
	}
	if got := charStringCommand123.Type2KeyWord(); got != Type2AND {
		t.Errorf("GetInstance([12 3]).Type2KeyWord() = %v, want %v", got, Type2AND)
	}
	if got := charStringCommand123.String(); got != "AND|" {
		t.Errorf("GetInstance([12 3]).String() = %q, want %q", got, "AND|")
	}
}

func TestUnknownCharStringCommand(t *testing.T) {
	charStringCommandUnknown := GetInstance(99)
	if got := charStringCommandUnknown.String(); got != "unknown command|" {
		t.Errorf("GetInstance(99).String() = %q, want %q", got, "unknown command|")
	}
}
