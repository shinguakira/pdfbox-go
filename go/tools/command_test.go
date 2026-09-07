package tools

// The flag plumbing, and the exit codes and streams every command answers
// through it.
//
// There is no Java test for this layer: picocli is a library, and PDFBox tests
// its own commands through it rather than testing it. What is asserted here is
// picocli's documented contract -- CommandLine.ExitCode.OK is 0, SOFTWARE is 1
// and USAGE is 2 -- and the one of those three the Java tests do pin down,
// which is that a command that ran answers 0 (`assertEquals(0, exitCode)` in
// TestExtractText and TestTextToPdf).
//
// Unlike every other module of this port, `tools` cannot be run here to settle
// an argument: picocli is not in the local Maven repository and there is no
// network to fetch it. The libraries underneath it are runnable, and are what
// the ported Java tests actually exercise.

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

// panicking is a command whose Call panics, which is Java's call() throwing.
type panicking struct {
	streams
	failWith string
}

func (p *panicking) Name() string        { return "panicking" }
func (p *panicking) Header() string      { return "Panics" }
func (p *panicking) Flags(*flag.FlagSet) {}
func (p *panicking) Call() int           { panic(p.failWith) }

// needsInput is a command with a required option, which is picocli's
// `required = true`.
type needsInput struct {
	streams
	infile string
}

func (n *needsInput) Name() string   { return "needsinput" }
func (n *needsInput) Header() string { return "Needs an input file" }

func (n *needsInput) Flags(set *flag.FlagSet) {
	set.StringVar(&n.infile, "i", "", "the PDF file")
	set.StringVar(&n.infile, "input", "", "the PDF file")
}

func (n *needsInput) validate(set *flag.FlagSet) error {
	return requireSet(set, "--input=<infile>", "i", "input")
}

func (n *needsInput) Call() int {
	n.printlnOut("read " + n.infile)
	return ExitOK
}

// run is the shape every case below shares.
func run(command Command, args ...string) (code int, out, errOut string) {
	var stdout, stderr bytes.Buffer
	code = Execute(command, args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// TestVersionCommand is the `version` command: it prints the qualified name and
// the version, and answers 0.
func TestVersionCommand(t *testing.T) {
	code, out, errOut := run(NewVersion())
	if code != ExitOK {
		t.Errorf("version exited %d, want %d", code, ExitOK)
	}
	if want := "version [4.0.0-SNAPSHOT]\n"; out != want {
		t.Errorf("version printed %q, want %q", out, want)
	}
	if errOut != "" {
		t.Errorf("version wrote %q to stderr, want nothing", errOut)
	}
}

// TestStandardHelpOptions is mixinStandardHelpOptions: -h/--help and
// -V/--version, both of which print and exit 0.
func TestStandardHelpOptions(t *testing.T) {
	for _, name := range []string{"-h", "--help"} {
		code, out, errOut := run(&needsInput{}, name)
		if code != ExitOK {
			t.Errorf("%s exited %d, want %d", name, code, ExitOK)
		}
		if !strings.Contains(out, "Needs an input file") {
			t.Errorf("%s printed %q, want the command header", name, out)
		}
		if !strings.Contains(out, "-input") {
			t.Errorf("%s printed %q, want the options", name, out)
		}
		if errOut != "" {
			t.Errorf("%s wrote %q to stderr; help is not an error", name, errOut)
		}
	}
	for _, name := range []string{"-V", "--version"} {
		code, out, errOut := run(&needsInput{}, name)
		if code != ExitOK {
			t.Errorf("%s exited %d, want %d", name, code, ExitOK)
		}
		if want := "needsinput [4.0.0-SNAPSHOT]\n"; out != want {
			t.Errorf("%s printed %q, want %q", name, out, want)
		}
		if errOut != "" {
			t.Errorf("%s wrote %q to stderr", name, errOut)
		}
	}
}

// TestUnrecognisedOptionIsAUsageError is ExitCode.USAGE, and the stream it goes
// to. A tool that prints its error to stdout breaks every pipeline that reads
// its output.
func TestUnrecognisedOptionIsAUsageError(t *testing.T) {
	code, out, errOut := run(&needsInput{}, "-i", "x.pdf", "-nosuchflag")
	if code != ExitUsage {
		t.Errorf("an unrecognised option exited %d, want %d", code, ExitUsage)
	}
	if out != "" {
		t.Errorf("the error went to stdout as %q; it belongs on stderr", out)
	}
	if !strings.Contains(errOut, "nosuchflag") {
		t.Errorf("stderr is %q, want it to name the option", errOut)
	}
}

// TestMissingRequiredOptionIsAUsageError is picocli's `required = true`.
func TestMissingRequiredOptionIsAUsageError(t *testing.T) {
	code, out, errOut := run(&needsInput{})
	if code != ExitUsage {
		t.Errorf("a missing required option exited %d, want %d", code, ExitUsage)
	}
	if out != "" {
		t.Errorf("the error went to stdout as %q; it belongs on stderr", out)
	}
	if want := "Missing required option: '--input=<infile>'"; !strings.Contains(errOut, want) {
		t.Errorf("stderr is %q, want it to contain %q", errOut, want)
	}

	// And it is not an error once either spelling is given.
	for _, name := range []string{"-i", "--input", "-input"} {
		code, out, _ := run(&needsInput{}, name, "x.pdf")
		if code != ExitOK {
			t.Errorf("%s exited %d, want %d", name, code, ExitOK)
		}
		if want := "read x.pdf\n"; out != want {
			t.Errorf("%s printed %q, want %q", name, out, want)
		}
	}
}

// TestAThrowingCommandIsASoftwareError is ExitCode.SOFTWARE. This port renders
// an unchecked Java exception as a panic, so the panic is what picocli's
// execution exception handler catches.
func TestAThrowingCommandIsASoftwareError(t *testing.T) {
	code, out, errOut := run(&panicking{failWith: "something went wrong"})
	if code != ExitSoftware {
		t.Errorf("a command that threw exited %d, want %d", code, ExitSoftware)
	}
	if out != "" {
		t.Errorf("the failure went to stdout as %q; it belongs on stderr", out)
	}
	if !strings.Contains(errOut, "something went wrong") {
		t.Errorf("stderr is %q, want the failure in it", errOut)
	}
}

// TestSingleDashLongOptionsParse is the reason the standard library's flag was
// chosen over a POSIX parser: picocli declares long options with one dash, and
// a POSIX parser would read -addFileName as a cluster of ten single-letter
// flags.
func TestSingleDashLongOptionsParse(t *testing.T) {
	var longOption bool
	command := &flagShapes{onFlags: func(set *flag.FlagSet) {
		set.BoolVar(&longOption, "addFileName", false, "Print PDF file name")
	}}
	for _, args := range [][]string{
		{"-addFileName"},
		{"--addFileName"},
		{"-addFileName=true"},
	} {
		longOption = false
		code, _, errOut := run(command, args...)
		if code != ExitOK {
			t.Errorf("%v exited %d (%s), want %d", args, code, errOut, ExitOK)
		}
		if !longOption {
			t.Errorf("%v did not set the option", args)
		}
	}
}

// flagShapes declares whatever its caller asks for.
type flagShapes struct {
	streams
	onFlags func(*flag.FlagSet)
}

func (f *flagShapes) Name() string            { return "flagshapes" }
func (f *flagShapes) Header() string          { return "Declares what it is told to" }
func (f *flagShapes) Flags(set *flag.FlagSet) { f.onFlags(set) }
func (f *flagShapes) Call() int               { return ExitOK }
