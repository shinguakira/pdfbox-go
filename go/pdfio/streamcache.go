package pdfio

// StreamCache hands out the scratch buffers used while creating or writing the
// streams of a PDF.
//
// Port of org.apache.pdfbox.io.RandomAccessStreamCache. Buffers returned by
// CreateBuffer should be closed by the caller; an implementation may also close
// any it still owns when the cache itself is closed, though the memory-backed
// one has nothing to release and does not.
type StreamCache interface {
	// CreateBuffer returns a fresh read-write buffer.
	CreateBuffer() (RandomAccess, error)

	// Close releases every buffer the cache still owns.
	Close() error
}

// StreamCacheFunc creates a StreamCache on demand.
//
// Port of the RandomAccessStreamCache.StreamCacheCreateFunction functional
// interface, which Go expresses as a plain function type.
type StreamCacheFunc func() (StreamCache, error)

// MemoryStreamCache is the default StreamCache: every buffer it hands out is
// held in memory.
//
// Port of org.apache.pdfbox.io.RandomAccessStreamCacheImpl.
type MemoryStreamCache struct{}

var _ StreamCache = (*MemoryStreamCache)(nil)

// NewMemoryStreamCache returns a cache backed by in-memory buffers.
func NewMemoryStreamCache() *MemoryStreamCache { return &MemoryStreamCache{} }

// CreateBuffer returns a new in-memory read-write buffer.
func (c *MemoryStreamCache) CreateBuffer() (RandomAccess, error) {
	return NewReadWriteBuffer(), nil
}

// Close is a no-op: the buffers are garbage collected with their callers.
func (c *MemoryStreamCache) Close() error { return nil }

// MemoryOnlyStreamCache returns a StreamCacheFunc producing memory-backed
// caches, the equivalent of IOUtils.createMemoryOnlyStreamCache in Java.
func MemoryOnlyStreamCache() StreamCacheFunc {
	return func() (StreamCache, error) { return NewMemoryStreamCache(), nil }
}
