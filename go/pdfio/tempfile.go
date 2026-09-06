package pdfio

// Temporary files that only their owner can read.
//
// Port of the createProtectedTempFile and applyOwnerOnlyPermissions half of
// org.apache.pdfbox.io.IOUtils, which the package comment of ioutils.go
// records as deferred to the scratch file work.

import (
	"io"
	"os"
	"runtime"
)

// ownerOnlyFilePerm is POSIX_FILE_PERMS: read and write for the owner and
// nothing for anyone else.
const ownerOnlyFilePerm os.FileMode = 0o600

// createProtectedTempFile makes a temporary file readable and writable only by
// its owner, in the given directory or the default temporary one where dir is
// empty.
//
// Port of IOUtils.createProtectedTempFile(Path, String, String). Java sets the
// permissions at creation time where the file system is POSIX, and rewrites
// them straight after otherwise, to keep the window in which the file has
// default permissions as short as it can. os.CreateTemp takes a mode of 0600 on
// every platform Go supports a mode on, so the port has one path rather than
// two; on Windows the mode is not applied and the file inherits the directory's
// access control, which is what Java's own fallback does when there is no ACL
// view.
func createProtectedTempFile(dir, prefix, suffix string) (*os.File, error) {
	file, err := os.CreateTemp(dir, prefix+"*"+suffix)
	if err != nil {
		return nil, err
	}
	if runtime.GOOS != "windows" {
		if err := file.Chmod(ownerOnlyFilePerm); err != nil {
			name := file.Name()
			file.Close()
			os.Remove(name)
			return nil, err
		}
	}
	return file, nil
}

// readFullyAt reads len(p) bytes from f at the given offset, reporting the
// short read Java's RandomAccessFile.readFully raises EOFException for.
func readFullyAt(f *os.File, p []byte, off int64) (int, error) {
	n, err := f.ReadAt(p, off)
	if err == io.EOF && n == len(p) {
		return n, nil
	}
	return n, err
}
