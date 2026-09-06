package pdfio

// A read-write buffer whose contents live in the pages a ScratchFile hands out.
//
// Port of org.apache.pdfbox.io.ScratchFileBuffer, which is package-private in
// Java and unexported here for the same reason: a caller reaches it only
// through ScratchFile.CreateBuffer.

import (
	"errors"
	"fmt"
	"io"
)

// ErrBufferClosed is returned by every operation on a scratch file buffer that
// has already been closed.
//
// Java raises IOException("Buffer already closed").
var ErrBufferClosed = errors.New("pdfio: buffer already closed")

// ScratchFileBuffer holds its content in pages of a ScratchFile.
type ScratchFileBuffer struct {
	pageSize int

	// pageHandler is the ScratchFile the pages come from, and nil once this
	// buffer is closed.
	pageHandler *ScratchFile

	// size is the number of bytes of content in this buffer.
	size int64

	// currentPagePositionInPageIndexes is the index within pageIndexes of the
	// page the cursor is on, so the nth page of this buffer.
	currentPagePositionInPageIndexes int

	// currentPageOffset is the offset of the current page within this buffer.
	currentPageOffset int64

	// currentPage is the data of the current page.
	currentPage []byte

	// positionInPage is where the next read or write goes, as an offset within
	// the current page.
	positionInPage int

	// currentPageContentChanged says whether the current page was written to.
	currentPageContentChanged bool

	pageIndexes []int
	pageCount   int
}

var _ RandomAccess = (*ScratchFileBuffer)(nil)

// newScratchFileBuffer returns a buffer using the pages of the given handler.
//
// Port of the ScratchFileBuffer(ScratchFile) constructor.
func newScratchFileBuffer(pageHandler *ScratchFile) (*ScratchFileBuffer, error) {
	if err := pageHandler.checkClosed(); err != nil {
		return nil, err
	}
	b := &ScratchFileBuffer{
		pageHandler: pageHandler,
		pageSize:    pageHandler.getPageSize(),
		pageIndexes: make([]int, 16),
	}
	if err := b.addPage(); err != nil {
		return nil, err
	}
	return b, nil
}

// checkClosed reports whether this buffer, or the scratch file behind it, has
// been closed.
func (b *ScratchFileBuffer) checkClosed() error {
	if b.pageHandler == nil {
		return ErrBufferClosed
	}
	return b.pageHandler.checkClosed()
}

// addPage adds a page and points everything at the start of it.
func (b *ScratchFileBuffer) addPage() error {
	if b.pageCount+1 >= len(b.pageIndexes) {
		newSize := len(b.pageIndexes) * 2
		// check overflow
		if newSize < len(b.pageIndexes) {
			if len(b.pageIndexes) == maxInt32 {
				return errors.New("Maximum buffer size reached.")
			}
			newSize = maxInt32
		}
		grown := make([]int, newSize)
		copy(grown, b.pageIndexes[:b.pageCount])
		b.pageIndexes = grown
	}

	newPageIdx, err := b.pageHandler.getNewPage()
	if err != nil {
		return err
	}

	b.pageIndexes[b.pageCount] = newPageIdx
	b.currentPagePositionInPageIndexes = b.pageCount
	b.currentPageOffset = int64(b.pageCount) * int64(b.pageSize)
	b.pageCount++
	b.currentPage = make([]byte, b.pageSize)
	b.positionInPage = 0
	return nil
}

// maxInt32 is Integer.MAX_VALUE, which bounds the page index array.
const maxInt32 = 1<<31 - 1

// Length returns the number of bytes of content in this buffer.
func (b *ScratchFileBuffer) Length() (int64, error) { return b.size, nil }

// ensureAvailableBytesInPage makes sure the current page has at least one byte
// left, moving to the next page and writing the current one where it does not.
//
// It reports false where the cursor is at the end of the last page and adding
// one is not allowed.
func (b *ScratchFileBuffer) ensureAvailableBytesInPage(addNewPageIfNeeded bool) (bool, error) {
	if b.positionInPage < b.pageSize {
		return true, nil
	}
	// page full
	if b.currentPageContentChanged {
		// write page
		if err := b.pageHandler.writePage(
			b.pageIndexes[b.currentPagePositionInPageIndexes], b.currentPage); err != nil {
			return false, err
		}
		b.currentPageContentChanged = false
	}
	// get new page
	switch {
	case b.currentPagePositionInPageIndexes+1 < b.pageCount:
		// we already have more pages assigned (there was a backward seek
		// before)
		b.currentPagePositionInPageIndexes++
		page, err := b.pageHandler.readPage(
			b.pageIndexes[b.currentPagePositionInPageIndexes])
		if err != nil {
			return false, err
		}
		b.currentPage = page
		b.currentPageOffset = int64(b.currentPagePositionInPageIndexes) * int64(b.pageSize)
		b.positionInPage = 0
	case addNewPageIfNeeded:
		// need new page
		if err := b.addPage(); err != nil {
			return false, err
		}
	default:
		// we are at last page and are not allowed to add new page
		return false, nil
	}
	return true, nil
}

// WriteByte writes one byte.
//
// Port of write(int).
func (b *ScratchFileBuffer) WriteByte(c byte) error {
	if err := b.checkClosed(); err != nil {
		return err
	}
	if _, err := b.ensureAvailableBytesInPage(true); err != nil {
		return err
	}

	b.currentPage[b.positionInPage] = c
	b.positionInPage++
	b.currentPageContentChanged = true

	if b.currentPageOffset+int64(b.positionInPage) > b.size {
		b.size = b.currentPageOffset + int64(b.positionInPage)
	}
	return nil
}

// Write writes the whole slice.
//
// Port of write(byte[], int, int); Go's io.Writer carries the offset and length
// in the slice it is handed.
func (b *ScratchFileBuffer) Write(p []byte) (int, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	remain := len(p)
	bOff := 0

	for remain > 0 {
		if _, err := b.ensureAvailableBytesInPage(true); err != nil {
			return bOff, err
		}
		bytesToWrite := remain
		if room := b.pageSize - b.positionInPage; bytesToWrite > room {
			bytesToWrite = room
		}

		copy(b.currentPage[b.positionInPage:], p[bOff:bOff+bytesToWrite])

		b.positionInPage += bytesToWrite
		b.currentPageContentChanged = true

		bOff += bytesToWrite
		remain -= bytesToWrite
	}

	if b.currentPageOffset+int64(b.positionInPage) > b.size {
		b.size = b.currentPageOffset + int64(b.positionInPage)
	}
	return len(p), nil
}

// Clear discards everything the buffer holds, keeping its first page.
func (b *ScratchFileBuffer) Clear() error {
	if err := b.checkClosed(); err != nil {
		return err
	}

	// keep only the first page, discard all other pages
	b.pageHandler.markPagesAsFree(b.pageIndexes, 1, b.pageCount-1)
	b.pageCount = 1

	// change to first page if we are not already there
	if b.currentPagePositionInPageIndexes > 0 {
		page, err := b.pageHandler.readPage(b.pageIndexes[0])
		if err != nil {
			return err
		}
		b.currentPage = page
		b.currentPagePositionInPageIndexes = 0
		b.currentPageOffset = 0
	}
	b.positionInPage = 0
	b.size = 0
	b.currentPageContentChanged = false
	return nil
}

// Position returns the offset of the next byte to be read or written.
func (b *ScratchFileBuffer) Position() (int64, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	return b.currentPageOffset + int64(b.positionInPage), nil
}

// Seek moves the cursor. Only io.SeekStart is used by the port, which is what
// Java's seek(long) takes.
func (b *ScratchFileBuffer) Seek(offset int64, whence int) (int64, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	// Java's seek(long) takes no whence; the other two are what io.Seeker
	// obliges the port to answer, and the rest of this package spells them out
	// the same way.
	seekToPosition := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		seekToPosition += b.currentPageOffset + int64(b.positionInPage)
	case io.SeekEnd:
		seekToPosition += b.size
	default:
		return 0, ErrInvalidPosition
	}

	// for now we won't allow to seek past end of buffer; this can be changed
	// by adding new pages as needed
	if seekToPosition > b.size {
		return 0, io.EOF
	}

	if seekToPosition < 0 {
		return 0, fmt.Errorf("%w: negative seek offset: %d", ErrInvalidPosition,
			seekToPosition)
	}

	if seekToPosition >= b.currentPageOffset &&
		seekToPosition <= b.currentPageOffset+int64(b.pageSize) {
		// within same page
		b.positionInPage = int(seekToPosition - b.currentPageOffset)
		return seekToPosition, nil
	}

	// have to go to another page

	// check if current page needs to be written to file
	if b.currentPageContentChanged {
		if err := b.pageHandler.writePage(
			b.pageIndexes[b.currentPagePositionInPageIndexes], b.currentPage); err != nil {
			return 0, err
		}
		b.currentPageContentChanged = false
	}

	newPagePosition := int(seekToPosition / int64(b.pageSize))
	if seekToPosition%int64(b.pageSize) == 0 && seekToPosition == b.size {
		// PDFBOX-4756: Prevent seeking a non-yet-existent page
		newPagePosition--
	}

	page, err := b.pageHandler.readPage(b.pageIndexes[newPagePosition])
	if err != nil {
		return 0, err
	}
	b.currentPage = page
	b.currentPagePositionInPageIndexes = newPagePosition
	b.currentPageOffset = int64(b.currentPagePositionInPageIndexes) * int64(b.pageSize)
	b.positionInPage = int(seekToPosition - b.currentPageOffset)
	return seekToPosition, nil
}

// IsClosed reports whether this buffer has been closed.
func (b *ScratchFileBuffer) IsClosed() bool { return b.pageHandler == nil }

// IsEOF reports whether the cursor sits at or past the end of the content.
func (b *ScratchFileBuffer) IsEOF() (bool, error) {
	if err := b.checkClosed(); err != nil {
		return false, err
	}
	return b.currentPageOffset+int64(b.positionInPage) >= b.size, nil
}

// ReadByte reads one byte, reporting io.EOF at the end of the content where
// Java's read() answers -1.
func (b *ScratchFileBuffer) ReadByte() (byte, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	if b.currentPageOffset+int64(b.positionInPage) >= b.size {
		return 0, io.EOF
	}
	available, err := b.ensureAvailableBytesInPage(false)
	if err != nil {
		return 0, err
	}
	if !available {
		// should not happen, we checked it before
		return 0, errors.New("Unexpectedly no bytes available for read in buffer.")
	}

	c := b.currentPage[b.positionInPage]
	b.positionInPage++
	return c, nil
}

// Read fills p, reporting io.EOF at the end of the content where Java's
// read(byte[], int, int) answers -1.
func (b *ScratchFileBuffer) Read(p []byte) (int, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	if b.currentPageOffset+int64(b.positionInPage) >= b.size {
		return 0, io.EOF
	}
	remain := len(p)
	if left := b.size - (b.currentPageOffset + int64(b.positionInPage)); int64(remain) > left {
		remain = int(left)
	}
	totalBytesRead := 0
	bOff := 0

	for remain > 0 {
		available, err := b.ensureAvailableBytesInPage(false)
		if err != nil {
			return totalBytesRead, err
		}
		if !available {
			// should not happen, we checked it before
			return totalBytesRead,
				errors.New("Unexpectedly no bytes available for read in buffer.")
		}

		readBytes := remain
		if room := b.pageSize - b.positionInPage; readBytes > room {
			readBytes = room
		}
		copy(p[bOff:bOff+readBytes], b.currentPage[b.positionInPage:])
		b.positionInPage += readBytes
		totalBytesRead += readBytes
		bOff += readBytes
		remain -= readBytes
	}
	return totalBytesRead, nil
}

// Close releases the pages this buffer holds and takes it off its scratch file.
func (b *ScratchFileBuffer) Close() error {
	b.closeBuffer(true)
	return nil
}

// closeBuffer releases the pages, taking the buffer off its scratch file only
// where removeBuffer says to -- the scratch file's own Close walks the list it
// would otherwise be removing from.
//
// Port of close(boolean).
func (b *ScratchFileBuffer) closeBuffer(removeBuffer bool) {
	if b.pageHandler == nil {
		return
	}
	b.pageHandler.markPagesAsFree(b.pageIndexes, 0, b.pageCount)
	if removeBuffer {
		b.pageHandler.removeBuffer(b)
	}
	b.pageHandler = nil
	b.pageIndexes = nil
	b.currentPage = nil
	b.currentPageOffset = 0
	b.currentPagePositionInPageIndexes = -1
	b.positionInPage = 0
	b.size = 0
}

// CreateView is not supported: a scratch file buffer cannot make an
// independent cursor over itself.
//
// Java raises UnsupportedOperationException.
func (b *ScratchFileBuffer) CreateView(start, length int64) (RandomAccessRead, error) {
	return nil, fmt.Errorf("%w: ScratchFileBuffer.CreateView isn't supported",
		ErrViewNotSupported)
}
