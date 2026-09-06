package pdfio

import "io"

// Helpers ported from org.apache.pdfbox.io.IOUtils.
//
// Most of that class is already deprecated upstream in favour of the JDK, and
// the Go standard library covers the same ground, so the port does not carry
// those methods over:
//
//	IOUtils.toByteArray(in)         -> io.ReadAll(r)
//	IOUtils.copy(in, out)           -> io.Copy(dst, src)
//	IOUtils.populateBuffer(in, buf) -> io.ReadFull(r, buf)
//	IOUtils.unmap(buffer)           -> not needed, Go has no mapped ByteBuffer
//
// The rest was deferred with the scratch file support and arrives with it:
// createProtectedTempFile is in tempfile.go, and createTempFileOnlyStreamCache
// in streamcache.go alongside createMemoryOnlyStreamCache.
//
// createProtectedTempDir is not ported. Its only caller in the whole tree is
// PDFDebugger, which is not in the migration plan, and it exists to register a
// JVM shutdown hook that deletes the directory -- Go has no shutdown hook, so
// porting it would mean inventing a lifetime rather than reproducing one.

// CloseQuietly closes c and discards any error, for use in deferred cleanup
// where the error cannot be acted on.
//
// Port of IOUtils.closeQuietly. Callers that can report the failure should use
// CloseAndKeepError instead of dropping it here.
func CloseQuietly(c io.Closer) {
	if c == nil {
		return
	}
	_ = c.Close()
}

// CloseAndKeepError closes c and returns the first of the two errors, so that a
// deferred close can surface its failure without masking one already in flight.
//
// Port of IOUtils.closeAndLogException, which logs the close failure and
// returns whichever exception should propagate.
func CloseAndKeepError(c io.Closer, err error) error {
	if c == nil {
		return err
	}
	closeErr := c.Close()
	if err != nil {
		return err
	}
	return closeErr
}
