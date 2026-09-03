package pdfio

import (
	"container/list"
	"io"
	"os"
	"sync"
)

const (
	pageSizeShift = 12
	pageSize      = 1 << pageSizeShift
	pageOffsetMsk = -1 << pageSizeShift
	maxCachedPage = 1000
)

// fileSource owns the file handle and the page cache shared by a BufferedFile
// and every cursor cloned from it.
//
// Java keeps the LRU cache in a subclassed LinkedHashMap private to each
// RandomAccessReadBufferedFile and gives each thread its own copy of the whole
// object, reopening the file per thread. Sharing one mutex-guarded cache is
// both cheaper and safe for concurrent use.
type fileSource struct {
	file   *os.File
	path   string
	length int64

	mu    sync.Mutex
	pages map[int64]*list.Element
	order *list.List // front is most recently used
}

type cachedPage struct {
	offset int64
	data   []byte
}

// page returns the cached page starting at offset, reading it from the file on
// a miss.
func (s *fileSource) page(offset int64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if el, ok := s.pages[offset]; ok {
		s.order.MoveToFront(el)
		return el.Value.(*cachedPage).data, nil
	}
	data := make([]byte, pageSize)
	n, err := s.file.ReadAt(data, offset)
	if err != nil && err != io.EOF {
		return nil, err
	}
	// Short reads at the end of the file leave the tail zeroed; callers never
	// look past fileLength, so the padding is never observable.
	_ = n
	el := s.order.PushFront(&cachedPage{offset: offset, data: data})
	s.pages[offset] = el
	if s.order.Len() > maxCachedPage {
		// Java reuses the evicted ByteBuffer for the next page read. The port
		// drops it instead, so a cursor still holding an evicted page keeps
		// reading valid bytes rather than someone else's data.
		oldest := s.order.Back()
		s.order.Remove(oldest)
		delete(s.pages, oldest.Value.(*cachedPage).offset)
	}
	return data, nil
}

// BufferedFile provides random access to a file through a cache of fixed size
// pages.
//
// Port of org.apache.pdfbox.io.RandomAccessReadBufferedFile.
type BufferedFile struct {
	src   *fileSource
	owner bool // only the root cursor closes the file

	offset       int64
	pageOffset   int64
	page         []byte
	offsetInPage int
	closed       bool

	mu     sync.Mutex
	clones []*BufferedFile
}

var _ RandomAccessRead = (*BufferedFile)(nil)

// OpenBufferedFile opens the named file for buffered random access reading.
func OpenBufferedFile(path string) (*BufferedFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	b := &BufferedFile{
		src: &fileSource{
			file:   f,
			path:   path,
			length: info.Size(),
			pages:  make(map[int64]*list.Element),
			order:  list.New(),
		},
		owner:      true,
		pageOffset: -1,
	}
	if err := SeekTo(b, 0); err != nil {
		f.Close()
		return nil, err
	}
	return b, nil
}

func (b *BufferedFile) checkClosed() error {
	if b.closed {
		return ErrClosed
	}
	return nil
}

// Position returns the offset of the next byte to be read.
func (b *BufferedFile) Position() (int64, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	return b.offset, nil
}

// Seek moves the cursor, loading the page holding the target offset. Offsets
// past the end of the file are clamped to its length, as they are in Java.
func (b *BufferedFile) Seek(offset int64, whence int) (int64, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	abs := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		abs += b.offset
	case io.SeekEnd:
		abs += b.src.length
	default:
		return 0, ErrInvalidPosition
	}
	if abs < 0 {
		return 0, ErrInvalidPosition
	}
	newPageOffset := abs & pageOffsetMsk
	if newPageOffset != b.pageOffset {
		page, err := b.src.page(newPageOffset)
		if err != nil {
			return 0, err
		}
		b.pageOffset = newPageOffset
		b.page = page
	}
	b.offset = abs
	if b.offset > b.src.length {
		b.offset = b.src.length
	}
	b.offsetInPage = int(b.offset - b.pageOffset)
	return b.offset, nil
}

// ReadByte returns the next byte of the file, or io.EOF at its end.
func (b *BufferedFile) ReadByte() (byte, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	if b.offset >= b.src.length {
		return 0, io.EOF
	}
	if b.offsetInPage == pageSize {
		if _, err := b.Seek(b.offset, io.SeekStart); err != nil {
			return 0, err
		}
	}
	v := b.page[b.offsetInPage]
	b.offsetInPage++
	b.offset++
	return v, nil
}

// Read fills p from the current page. Like the Java version it stops at the
// page boundary, so a caller wanting more must read again.
func (b *BufferedFile) Read(p []byte) (int, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	if b.offset >= b.src.length {
		return 0, io.EOF
	}
	if b.offsetInPage == pageSize {
		if _, err := b.Seek(b.offset, io.SeekStart); err != nil {
			return 0, err
		}
	}
	n := pageSize - b.offsetInPage
	if n > len(p) {
		n = len(p)
	}
	if remaining := b.src.length - b.offset; int64(n) > remaining {
		n = int(remaining)
	}
	copy(p[:n], b.page[b.offsetInPage:b.offsetInPage+n])
	b.offsetInPage += n
	b.offset += int64(n)
	return n, nil
}

// Length returns the size of the file.
func (b *BufferedFile) Length() (int64, error) {
	if err := b.checkClosed(); err != nil {
		return 0, err
	}
	return b.src.length, nil
}

// IsEOF reports whether the cursor has reached the end of the file.
//
// Java implements this as peek() == -1, which reads a byte and rewinds;
// comparing the offset to the length is equivalent and avoids the read.
func (b *BufferedFile) IsEOF() (bool, error) {
	if err := b.checkClosed(); err != nil {
		return false, err
	}
	return b.offset >= b.src.length, nil
}

// Close releases the cursor. The root cursor also closes the file, drops the
// page cache and closes every view handed out by CreateView.
func (b *BufferedFile) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	if !b.owner {
		return nil
	}
	b.mu.Lock()
	clones := b.clones
	b.clones = nil
	b.mu.Unlock()
	for _, c := range clones {
		c.Close()
	}
	b.src.mu.Lock()
	b.src.pages = make(map[int64]*list.Element)
	b.src.order.Init()
	b.src.mu.Unlock()
	return b.src.file.Close()
}

// IsClosed reports whether this cursor has been closed.
func (b *BufferedFile) IsClosed() bool { return b.closed }

// CreateView returns a read-only window over an independent cursor into the
// same file. The file handle and the page cache are shared.
func (b *BufferedFile) CreateView(start, length int64) (RandomAccessRead, error) {
	if err := b.checkClosed(); err != nil {
		return nil, err
	}
	c := &BufferedFile{src: b.src, pageOffset: -1}
	if err := SeekTo(c, 0); err != nil {
		return nil, err
	}
	root := b
	if !root.owner {
		// clones do not track clones of their own
		root = nil
	}
	if root != nil {
		root.mu.Lock()
		root.clones = append(root.clones, c)
		root.mu.Unlock()
	}
	return NewReadView(c, start, length), nil
}

// Path returns the name the file was opened with.
func (b *BufferedFile) Path() string { return b.src.path }
