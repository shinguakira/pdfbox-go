package pdfio

// The page handler behind a scratch file: fixed-size pages held in memory, and
// spilled to a temporary file once the memory allowance runs out.
//
// Port of org.apache.pdfbox.io.ScratchFile.

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
)

// The three sizes ScratchFile declares.
const (
	// enlargePageCount is how many pages the scratch file grows by.
	enlargePageCount = 16

	// initUnrestrictedMainMemPageCount is the page count inMemoryPages starts
	// at where main memory is not restricted.
	initUnrestrictedMainMemPageCount = 100000

	// scratchFilePageSize is the size of one page.
	scratchFilePageSize = 4096
)

// ErrScratchFileClosed is returned by every operation on a scratch file that
// has already been closed.
//
// Java raises IOException("Scratch file already closed").
var ErrScratchFileClosed = errors.New("pdfio: scratch file already closed")

// ScratchFile hands out fixed-size pages, holding as many in memory as the
// setting allows and writing the rest to a temporary file.
//
// Port of ScratchFile, which implements RandomAccessStreamCache.
//
// Safe for concurrent use. Java takes its two locks in both orders --
// getNewPage holds the page lock and reaches ioLock through enlarge, while
// close holds ioLock and reaches the page lock through a buffer's
// markPagesAsFree -- so a thread writing while another closes can deadlock;
// Close here releases ioLock before it closes the buffers. See
// migration/JAVA-BUGS.md 66. The buffer list is the second hazard, entry 72:
// Close reads it under ioLock while CreateBuffer holds buffersLock.
type ScratchFile struct {
	// ioLock guards the temporary file and the in-memory page array, which is
	// Java's ioLock.
	ioLock sync.Mutex

	scratchFileDirectory string
	file                 *os.File
	fileName             string

	// pagesLock guards pageCount, freePages and the page array's contents,
	// which is Java's synchronization on freePages.
	pagesLock sync.Mutex
	pageCount int
	freePages []bool

	// inMemoryPages holds a page each, or nil for one not written yet. It is
	// sized to inMemoryMaxPageCount where main memory is restricted, and grows
	// from initUnrestrictedMainMemPageCount where it is not.
	inMemoryPages [][]byte

	inMemoryMaxPageCount      int
	maxPageCount              int
	useScratchFile            bool
	maxMainMemoryIsRestricted bool

	// buffersLock guards buffers, which is Java's synchronization on it.
	buffersLock sync.Mutex
	buffers     []*ScratchFileBuffer

	isClosed bool
}

var _ StreamCache = (*ScratchFile)(nil)

// NewScratchFileInDir returns a page handler that stores every page in a
// scratch file made in the given directory, or in the default temporary
// directory where it is empty.
//
// Port of ScratchFile(File).
func NewScratchFileInDir(scratchFileDirectory string) (*ScratchFile, error) {
	return NewScratchFile(SetupTempFileOnly().SetTempDir(scratchFileDirectory))
}

// NewScratchFile returns a page handler set up by the given memory usage
// setting: as many pages as it allows are held in memory and the rest are
// written to a scratch file.
//
// Port of ScratchFile(MemoryUsageSetting).
func NewScratchFile(memUsageSetting *MemoryUsageSetting) (*ScratchFile, error) {
	s := &ScratchFile{}
	s.maxMainMemoryIsRestricted = !memUsageSetting.UseMainMemory() ||
		memUsageSetting.IsMainMemoryRestricted()
	s.useScratchFile = s.maxMainMemoryIsRestricted && memUsageSetting.UseTempFile()
	if s.useScratchFile {
		s.scratchFileDirectory = memUsageSetting.TempDir()
	}
	if s.scratchFileDirectory != "" {
		info, err := os.Stat(s.scratchFileDirectory)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("Scratch file directory does not exist: %s",
				s.scratchFileDirectory)
		}
	}

	s.maxPageCount = math.MaxInt32
	if memUsageSetting.IsStorageRestricted() {
		s.maxPageCount = clampToInt32(memUsageSetting.MaxStorageBytes() / scratchFilePageSize)
	}

	switch {
	case !memUsageSetting.UseMainMemory():
		s.inMemoryMaxPageCount = 0
	case memUsageSetting.IsMainMemoryRestricted():
		s.inMemoryMaxPageCount = clampToInt32(
			memUsageSetting.MaxMainMemoryBytes() / scratchFilePageSize)
	default:
		s.inMemoryMaxPageCount = math.MaxInt32
	}
	return s, nil
}

// clampToInt32 is Java's `(int) Math.min(Integer.MAX_VALUE, value)`.
func clampToInt32(value int64) int {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	return int(value)
}

// initPages makes the in-memory page array on first use, and marks every slot
// of it free.
func (s *ScratchFile) initPages() {
	if s.inMemoryPages != nil {
		return
	}
	size := initUnrestrictedMainMemPageCount
	if s.maxMainMemoryIsRestricted {
		size = s.inMemoryMaxPageCount
	}
	s.inMemoryPages = make([][]byte, size)
	s.setFreePages(0, size)
}

// setFreePages marks the pages in [from, to) free, growing the record as
// needed. Java's freePages is a BitSet, which grows of itself.
func (s *ScratchFile) setFreePages(from, to int) {
	if to > len(s.freePages) {
		grown := make([]bool, to)
		copy(grown, s.freePages)
		s.freePages = grown
	}
	for i := from; i < to; i++ {
		s.freePages[i] = true
	}
}

// nextFreePage is BitSet.nextSetBit(0): the lowest free page, or -1.
func (s *ScratchFile) nextFreePage() int {
	for i, free := range s.freePages {
		if free {
			return i
		}
	}
	return -1
}

// MainMemoryOnlyScratchFile returns a page handler using unrestricted main
// memory, which is NewScratchFile(SetupMainMemoryOnly()).
//
// Port of getMainMemoryOnlyInstance(). Java catches the IOException that cannot
// happen for a main memory setup, logs it and answers null; the port's
// constructor cannot fail for this setting, so there is nothing to catch.
func MainMemoryOnlyScratchFile() *ScratchFile {
	s, _ := NewScratchFile(SetupMainMemoryOnly())
	return s
}

// MainMemoryOnlyScratchFileMax returns a page handler using main memory with
// the given maximum.
//
// Port of getMainMemoryOnlyInstance(long).
func MainMemoryOnlyScratchFileMax(maxMainMemoryBytes int64) *ScratchFile {
	s, _ := NewScratchFile(SetupMainMemoryOnlyMax(maxMainMemoryBytes))
	return s
}

// getNewPage returns a free page, either from the free page pool or by
// enlarging the scratch file, which may create it.
func (s *ScratchFile) getNewPage() (int, error) {
	s.pagesLock.Lock()
	defer s.pagesLock.Unlock()

	s.initPages()
	idx := s.nextFreePage()

	if idx < 0 {
		if err := s.enlarge(); err != nil {
			return 0, err
		}
		idx = s.nextFreePage()
		if idx < 0 {
			return 0, errors.New("Maximum allowed scratch file memory exceeded.")
		}
	}

	s.freePages[idx] = false

	if idx >= s.pageCount {
		s.pageCount = idx + 1
	}
	return idx, nil
}

// enlarge provides new free pages, by growing the scratch file where one is
// allowed or the in-memory array where main memory is not restricted. Where
// neither holds the free page count is unchanged.
//
// Only to be called under pagesLock. Taking ioLock while pagesLock is held is
// the second half of the lock-order inversion of JAVA-BUGS entry 66: Close
// takes them the other way round.
func (s *ScratchFile) enlarge() error {
	s.ioLock.Lock()
	defer s.ioLock.Unlock()

	if err := s.checkClosed(); err != nil {
		return err
	}
	if s.pageCount >= s.maxPageCount {
		return nil
	}

	if s.useScratchFile {
		// create scratch file if needed.
		//
		// Java creates the file and then opens a RandomAccessFile over it,
		// deleting it again where that second step throws
		// FileNotFoundException. os.CreateTemp does both at once and hands
		// back an open read-write file, so there is no window between them
		// and nothing to undo.
		if s.file == nil {
			file, err := createProtectedTempFile(s.scratchFileDirectory, "PDFBox", ".tmp")
			if err != nil {
				return err
			}
			s.file = file
			s.fileName = file.Name()
		}

		info, err := s.file.Stat()
		if err != nil {
			return err
		}
		fileLen := info.Size()
		expectedFileLen := (int64(s.pageCount) - int64(s.inMemoryMaxPageCount)) *
			scratchFilePageSize

		if expectedFileLen != fileLen {
			return fmt.Errorf("Expected scratch file size of %d but found %d",
				expectedFileLen, fileLen)
		}

		// enlarge if we do not overflow. Java's pageCount is an int, so the
		// sum can wrap; Go's int is 64 bits on every platform the port
		// builds for that has more than 2^31 pages of address space, so the
		// test is here for the shape rather than the arithmetic.
		if s.pageCount+enlargePageCount > s.pageCount {
			fileLen += enlargePageCount * scratchFilePageSize
			if err := s.file.Truncate(fileLen); err != nil {
				return err
			}
			s.setFreePages(s.pageCount, s.pageCount+enlargePageCount)
		}
		return nil
	}

	if !s.maxMainMemoryIsRestricted {
		// increase number of in-memory pages
		oldSize := len(s.inMemoryPages)
		// this handles integer overflow
		newSize := clampToInt32(int64(oldSize) * 2)
		if newSize > oldSize {
			grown := make([][]byte, newSize)
			copy(grown, s.inMemoryPages)
			s.inMemoryPages = grown
			s.setFreePages(oldSize, newSize)
		}
	}
	return nil
}

// getPageSize returns the size of one page.
func (s *ScratchFile) getPageSize() int { return scratchFilePageSize }

// readPage returns the page of the given index, which is a page-sized slice.
func (s *ScratchFile) readPage(pageIdx int) ([]byte, error) {
	if pageIdx < 0 || pageIdx >= s.pageCount {
		if err := s.checkClosed(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Page index out of range: %d. Max value: %d",
			pageIdx, s.pageCount-1)
	}

	// check if we have the page in memory
	if pageIdx < s.inMemoryMaxPageCount {
		page := s.inMemoryPages[pageIdx]

		// handle case that we are closed
		if page == nil {
			if err := s.checkClosed(); err != nil {
				return nil, err
			}
			return nil, fmt.Errorf(
				"Requested page with index %d was not written before.", pageIdx)
		}
		return page, nil
	}

	s.ioLock.Lock()
	defer s.ioLock.Unlock()

	if s.file == nil {
		if err := s.checkClosed(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(
			"Missing scratch file to read page with index %d from.", pageIdx)
	}

	page := make([]byte, scratchFilePageSize)
	at := (int64(pageIdx) - int64(s.inMemoryMaxPageCount)) * scratchFilePageSize
	if _, err := readFullyAt(s.file, page, at); err != nil {
		return nil, err
	}
	return page, nil
}

// writePage stores the page of the given index, in memory where its index is
// below the in-memory maximum and in the scratch file otherwise.
//
// The page must not be re-used for another page: an in-memory one is stored as
// it is.
func (s *ScratchFile) writePage(pageIdx int, page []byte) error {
	if pageIdx < 0 || pageIdx >= s.pageCount {
		if err := s.checkClosed(); err != nil {
			return err
		}
		return fmt.Errorf("Page index out of range: %d. Max value: %d",
			pageIdx, s.pageCount-1)
	}

	if len(page) != scratchFilePageSize {
		return fmt.Errorf("Wrong page size to write: %d. Expected: %d",
			len(page), scratchFilePageSize)
	}

	if pageIdx < s.inMemoryMaxPageCount {
		if s.maxMainMemoryIsRestricted {
			s.inMemoryPages[pageIdx] = page
		} else {
			// need synchronization since inMemoryPages may change
			s.ioLock.Lock()
			s.inMemoryPages[pageIdx] = page
			s.ioLock.Unlock()
		}
		// in case we were closed in between report it
		return s.checkClosed()
	}

	s.ioLock.Lock()
	defer s.ioLock.Unlock()
	if err := s.checkClosed(); err != nil {
		return err
	}
	at := (int64(pageIdx) - int64(s.inMemoryMaxPageCount)) * scratchFilePageSize
	_, err := s.file.WriteAt(page, at)
	return err
}

// checkClosed reports whether this page handler has already been closed.
func (s *ScratchFile) checkClosed() error {
	if s.isClosed {
		return ErrScratchFileClosed
	}
	return nil
}

// CreateBuffer returns a new buffer using this page handler.
//
// Port of createBuffer(), which is RandomAccessStreamCache's method.
func (s *ScratchFile) CreateBuffer() (RandomAccess, error) {
	newBuffer, err := newScratchFileBuffer(s)
	if err != nil {
		return nil, err
	}
	s.buffersLock.Lock()
	s.buffers = append(s.buffers, newBuffer)
	s.buffersLock.Unlock()
	return newBuffer, nil
}

// removeBuffer forgets a buffer that has closed itself.
func (s *ScratchFile) removeBuffer(buffer *ScratchFileBuffer) {
	s.buffersLock.Lock()
	defer s.buffersLock.Unlock()
	for i, held := range s.buffers {
		if held == buffer {
			s.buffers = append(s.buffers[:i], s.buffers[i+1:]...)
			return
		}
	}
}

// markPagesAsFree lets a buffer that was cleared or closed release its pages
// for re-use.
func (s *ScratchFile) markPagesAsFree(pageIndexes []int, off, count int) {
	s.pagesLock.Lock()
	defer s.pagesLock.Unlock()

	// Java walks from off to count rather than to off+count, so a release that
	// starts past the first index frees fewer pages than it was given -- and
	// Clear passes an offset of 1, so every clear leaked the buffer's last
	// page. See migration/JAVA-BUGS.md 62.
	for aIdx := off; aIdx < off+count; aIdx++ {
		pageIdx := pageIndexes[aIdx]
		if pageIdx >= 0 && pageIdx < s.pageCount && !s.freePages[pageIdx] {
			s.freePages[pageIdx] = true
			if pageIdx < s.inMemoryMaxPageCount {
				// remark: not under ioLock since behavior won't change even in
				// case of a parallel call to enlarge
				s.inMemoryPages[pageIdx] = nil
			}
		}
	}
}

// Close closes and deletes the temporary file, releases the in-memory pages,
// and closes every buffer that is still open.
//
// No further interaction with the scratch file or its buffers can happen after
// this.
//
// The buffers are closed under ioLock, and closing one takes pagesLock through
// markPagesAsFree -- the opposite order to getNewPage. See JAVA-BUGS entry 66.
func (s *ScratchFile) Close() error {
	var ioexc error

	s.ioLock.Lock()
	if s.isClosed {
		s.ioLock.Unlock()
		return nil
	}
	s.isClosed = true
	// The buffer list is read and cleared here under ioLock, while CreateBuffer
	// and removeBuffer mutate it under buffersLock -- so a buffer created
	// against a close in flight can be missed and left reporting itself open
	// over a scratch file that is gone. That is Java: those two synchronize on
	// the list and close does not. Ported as written; see
	// migration/JAVA-BUGS.md entry 72.
	buffers := s.buffers
	s.buffers = nil
	// Java closes the buffers here, still holding ioLock, and each one reaches
	// markPagesAsFree, which takes the page lock -- while getNewPage holds the
	// page lock and reaches ioLock through enlarge. The two orders deadlock. A
	// close does not need ioLock to close the buffers: isClosed is already set,
	// so enlarge refuses, and the list is already taken. See
	// migration/JAVA-BUGS.md 66.
	s.ioLock.Unlock()
	for _, buffer := range buffers {
		if buffer != nil && !buffer.IsClosed() {
			buffer.closeBuffer(false)
		}
	}
	s.ioLock.Lock()

	if s.file != nil {
		if err := s.file.Close(); err != nil {
			ioexc = err
		}
		if err := os.Remove(s.fileName); err != nil && !os.IsNotExist(err) &&
			ioexc == nil {
			// Java has only File.delete()'s boolean and so no cause to
			// report; os.Remove has one, and dropping it would lose why.
			ioexc = fmt.Errorf("Error deleting scratch file: %s: %w",
				s.fileName, err)
		}
	}
	s.ioLock.Unlock()

	s.pagesLock.Lock()
	s.freePages = nil
	s.pageCount = 0
	s.pagesLock.Unlock()

	return ioexc
}
