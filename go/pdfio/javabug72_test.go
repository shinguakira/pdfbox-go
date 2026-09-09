package pdfio

// JAVA-BUGS 72: `ScratchFile.close` reads and clears the buffer list while
// holding `ioLock`, where `createBuffer` and `removeBuffer` take the lock the
// list has. Java gets a plain ArrayList appended to while it is iterated; Go
// gets a data race on the slice, which the race detector names.

import (
	"testing"
	"time"
)

// TestCreateBufferDuringCloseIsNotARace is the defect.
//
// The expected behaviour is that the list is touched under one lock: the one
// its other two users take. This is the entry's second answer -- a copy taken
// under that lock and walked outside it -- which also keeps entry 66's fix,
// since `closeBuffer` reaches `removeBuffer` and would deadlock on a lock held
// across the walk.
//
// The race is what there is to see, so this test is only conclusive under
// `go test -race`: without the fix the detector names
// `ScratchFile.Close` reading `buffers` against `CreateBuffer` appending to
// it. Run without it the probe still exercises the window and must not fail.
func TestCreateBufferDuringCloseIsNotARace(t *testing.T) {
	const rounds = 200

	for round := 0; round < rounds; round++ {
		scratchFile, err := NewScratchFile(SetupMainMemoryOnly())
		noError(t, "NewScratchFile", err)

		done := make(chan struct{}, 2)
		go func() {
			for i := 0; i < 50; i++ {
				if _, err := scratchFile.CreateBuffer(); err != nil {
					break // a closed scratch file is the expected end
				}
			}
			done <- struct{}{}
		}()
		go func() {
			_ = scratchFile.Close()
			done <- struct{}{}
		}()

		for finished := 0; finished < 2; finished++ {
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatalf("round %d: creating a buffer and a close did not finish", round)
			}
		}
	}
}
