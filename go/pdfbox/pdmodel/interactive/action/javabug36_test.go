package action

// JAVA-BUGS 36: `PDWindowsLaunchParams.setOperation` writes `/D`, which is the
// working directory, where `getOperation` reads `/O`.

import "testing"

// TestSetOperationWritesTheOperationKey is the defect.
//
// The expected behaviour is the pair the two methods are: `getOperation` reads
// `/O`, so `setOperation` writes `/O`. Java writes `/D`, so the operation is
// never stored -- the getter keeps answering its default, "open" -- and the
// directory `setDirectory` wrote is overwritten in the same call.
func TestSetOperationWritesTheOperationKey(t *testing.T) {
	params := NewPDWindowsLaunchParams()
	params.SetDirectory("C:/somewhere")
	params.SetOperation(OperationPrint)

	if got := params.Operation(); got != OperationPrint {
		t.Errorf("Operation() = %q, want %q", got, OperationPrint)
	}
	if got := params.Directory(); got != "C:/somewhere" {
		t.Errorf("Directory() = %q, want the directory that was set", got)
	}
}

// TestOperationStillDefaultsToOpen keeps the default the getter carries, which
// is the only thing the getter answered while the setter wrote elsewhere.
func TestOperationStillDefaultsToOpen(t *testing.T) {
	if got := NewPDWindowsLaunchParams().Operation(); got != OperationOpen {
		t.Errorf("Operation() of fresh parameters = %q, want %q", got, OperationOpen)
	}
}
