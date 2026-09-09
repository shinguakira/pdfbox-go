package pdfio

// JAVA-BUGS 66: `ScratchFile` takes `ioLock` and the page lock in both orders
// -- `getNewPage` holds the page lock and reaches `ioLock` through `enlarge`,
// while `Close` holds `ioLock` and reaches the page lock through a buffer's
// `markPagesAsFree` -- so a goroutine writing while another closes can
// deadlock.

import (
	"testing"
	"time"
)

// TestCloseWhileWritingDoesNotDeadlock is the defect.
//
// The expected behaviour is that both finish: a scratch file exists to be
// shared, and closing one while a document is still being written to it is the
// ordinary shutdown race. The Java probe that found this ran the same two
// threads and hung in round 215 of 400, with the JVM's own detector naming
// both monitors.
//
// The cycle needs a writer to be inside `getNewPage`, holding the page lock,
// at the moment the close takes `ioLock`; that is a handful of instructions
// wide, so each round runs several writers, each on a buffer of its own, and
// the file is a temporary one because that is what puts real work inside
// `ioLock`. The round is bounded rather than waited on: a round that does not
// finish is the deadlock, and its goroutines are left where they are, since
// nothing can free them.
func TestCloseWhileWritingDoesNotDeadlock(t *testing.T) {
	const rounds = 150
	const writers = 8
	page := make([]byte, scratchPageSize)

	for round := 0; round < rounds; round++ {
		scratchFile, err := NewScratchFile(SetupTempFileOnly())
		noError(t, "NewScratchFile", err)

		buffers := make([]RandomAccess, writers)
		for i := range buffers {
			buffers[i] = createBuffer(t, scratchFile)
		}

		done := make(chan struct{}, writers+1)
		writing := make(chan struct{})
		for i := range buffers {
			buffer := buffers[i]
			first := i == 0
			go func() {
				// A ScratchFileBuffer is not thread safe -- Java says so on
				// the class -- so a write still in flight when the close
				// clears the buffer's own fields can fail on those. That is
				// not a lock and not what is being tested; Java gets a
				// NullPointerException there. Recover and end the round.
				defer func() {
					_ = recover()
					done <- struct{}{}
				}()
				for w := 0; w < 200; w++ {
					if _, err := buffer.Write(page); err != nil {
						break // a closed scratch file is the expected end
					}
					if first && w == 0 {
						close(writing) // the close can start now
					}
				}
			}()
		}
		go func() {
			<-writing
			_ = scratchFile.Close()
			done <- struct{}{}
		}()

		for finished := 0; finished < writers+1; finished++ {
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatalf("round %d: a writer and a close deadlocked", round)
			}
		}
	}
}
