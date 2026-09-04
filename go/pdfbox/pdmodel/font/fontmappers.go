package font

import "sync"

// The singleton FontMapper and the lazy default behind it.
//
// Port of org.apache.pdfbox.pdmodel.font.FontMappers, whose instance() reads
// the field without holding the lock its set() takes. Go's race detector would
// call that read what it is, so the port guards both; the behaviour -- one
// default mapper, built the first time it is asked for, replaceable by set --
// is the same.
var (
	fontMapperMutex    sync.Mutex
	fontMapperInstance FontMapper

	defaultFontMapperOnce sync.Once
	defaultFontMapper     FontMapper
)

// FontMappersInstance returns the singleton FontMapper instance.
//
// Port of FontMappers.instance().
func FontMappersInstance() FontMapper {
	fontMapperMutex.Lock()
	defer fontMapperMutex.Unlock()
	if fontMapperInstance == nil {
		fontMapperInstance = defaultFontMapperInstance()
	}
	return fontMapperInstance
}

// FontMappersSet stores the singleton FontMapper instance.
//
// Port of FontMappers.set().
func FontMappersSet(fontMapper FontMapper) {
	fontMapperMutex.Lock()
	defer fontMapperMutex.Unlock()
	fontMapperInstance = fontMapper
}

// defaultFontMapperInstance is Java's lazy thread safe singleton holder class
// FontMappers.DefaultFontMapper.
func defaultFontMapperInstance() FontMapper {
	defaultFontMapperOnce.Do(func() { defaultFontMapper = newFontMapperImpl() })
	return defaultFontMapper
}
