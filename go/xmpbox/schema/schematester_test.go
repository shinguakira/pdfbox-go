package schema_test

// The harness the schema tests are written against.
//
// Port of org.apache.xmpbox.schema.SchemaTester and XMPSchemaTester, which
// between them make three assertions about every field a schema declares: that
// it reads back as nothing before anything is set, that a value set through the
// accessors comes back through them, and that setting one field leaves every
// other field of the schema alone.
//
// Java reaches the accessors by reflection, naming them from the field with
// calculateSimpleGetter and its neighbours, and finds the fields by walking the
// class for @PropertyType annotations. There is no reflection in the port, so
// each test names its schema's fields and its accessors in a table, and this is
// what runs one.
//
// SchemaTester's random loops -- fifty rounds of getJavaValue for each field --
// become the several values a row may carry: the runner exercises the field
// once for each.

import (
	"strconv"
	"testing"
)

// fieldCase is one field of a schema, and what the test does with it.
type fieldCase[S any] struct {
	// name is the field, as the Java parameter list writes it.
	name string

	// absent reports whether every accessor of the field answers nothing,
	// which is Java's assertNull on the getter and on the property getter.
	absent func(*S) bool

	// exercise sets the field through the accessors and checks what they
	// answer. One call per value the Java parameter list gives.
	exercise []func(*testing.T, *S)
}

// runFieldCases runs the three assertions over a schema's fields.
func runFieldCases[S any](t *testing.T, fresh func(*testing.T) *S, cases []fieldCase[S]) {
	t.Helper()

	t.Run("InitializedToNull", func(t *testing.T) {
		s := fresh(t)
		for _, c := range cases {
			if !c.absent(s) {
				t.Errorf("%s reads back as set on a schema nothing was set on", c.name)
			}
		}
	})

	for _, c := range cases {
		for i, exercise := range c.exercise {
			t.Run(caseName(c.name, i, len(c.exercise)), func(t *testing.T) {
				s := fresh(t)
				exercise(t, s)
				if c.absent(s) {
					t.Errorf("%s reads back as nothing after it was set", c.name)
				}
				// check other properties not modified
				for _, other := range cases {
					if other.name == c.name {
						continue
					}
					if !other.absent(s) {
						t.Errorf("setting %s also set %s", c.name, other.name)
					}
				}
			})
		}
	}
}

// caseName names one round of a field's exercise, numbering the rounds only
// where there is more than one.
func caseName(name string, i, count int) string {
	if count == 1 {
		return name
	}
	return name + "/" + strconv.Itoa(i+1)
}

// equal reports whether two values of a comparable type are the same, and
// reports the difference where they are not.
func equal[T comparable](t *testing.T, what string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// noError fails the test where an accessor reported a problem.
func noError(t *testing.T, what string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

// holdsAll reports whether the list holds every one of the values, which is
// Java's binarySearch over the sorted parameter array.
func holdsAll(t *testing.T, what string, got, want []string) {
	t.Helper()
	for _, value := range want {
		found := false
		for _, held := range got {
			if held == value {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s = %v, which does not hold %q", what, got, value)
		}
	}
}
