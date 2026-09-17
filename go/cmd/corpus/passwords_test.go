package main

// What a line of a passwords table says.
//
// Migration tooling rather than a port, so there is no Java test to carry
// across. What these cases hold is the three shapes a line has, because the
// difference between them is silent: a line read as the wrong shape opens the
// file the wrong way, and a file that then refuses looks like a defect in the
// port rather than a mistake in the table.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLines puts a passwords table in a temporary directory and answers its path.
func writeLines(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "passwords.tsv")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadPasswordsTellsTheThreeShapesApart(t *testing.T) {
	openings = nil
	t.Cleanup(func() { openings = nil })

	table := writeLines(t,
		"# a comment, and a blank line after it",
		"",
		"plain.pdf\tsecret",
		"store.pdf\tkspass\tencryption/test.p12",
		"pair.pdf\t\tcerts/test.crt\tcerts/test.key",
	)
	if err := loadPasswords(table); err != nil {
		t.Fatal(err)
	}
	if len(openings) != 3 {
		t.Fatalf("loadPasswords read %d lines, want 3: the comment and the blank line are not lines", len(openings))
	}

	dir := filepath.Dir(table)
	if o := openings[0]; o.password != "secret" || o.keystore != "" || o.certificate != "" {
		t.Errorf("the password line read as %+v, want the password alone", o)
	}
	if o := openings[1]; o.password != "kspass" || o.keystore != filepath.Join(dir, "encryption", "test.p12") || o.certificate != "" {
		t.Errorf("the keystore line read as %+v, want the passphrase and the keystore beside the table", o)
	}
	if o := openings[2]; o.certificate != filepath.Join(dir, "certs", "test.crt") || o.key != filepath.Join(dir, "certs", "test.key") || o.keystore != "" {
		t.Errorf("the certificate line read as %+v, want the certificate and the key and no keystore", o)
	}
}

func TestLoadPasswordsRefusesALineOfAnotherShape(t *testing.T) {
	openings = nil
	t.Cleanup(func() { openings = nil })

	table := writeLines(t, "five.pdf\tsecret\tone\ttwo\tthree")
	if err := loadPasswords(table); err == nil {
		t.Fatal("loadPasswords took a line of five fields; a table it cannot read is the run's failure, not a file's")
	}
}
