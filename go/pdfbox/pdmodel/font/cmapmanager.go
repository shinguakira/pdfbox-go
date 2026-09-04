package font

import (
	"errors"
	"sync"

	"github.com/shinguakira/pdfbox-go/go/fontbox/cmap"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// cmapCache is the CMap resource cache.
//
// Java uses a ConcurrentHashMap; the mutex here is what stands in for it.
var (
	cmapCacheMu sync.Mutex
	cmapCache   = map[string]*cmap.CMap{}
)

// GetPredefinedCMap fetches the predefined CMap from disk (or cache). It is
// never nil unless the error is.
//
// Port of org.apache.pdfbox.pdmodel.font.CMapManager.getPredefinedCMap.
func GetPredefinedCMap(cMapName string) (*cmap.CMap, error) {
	cmapCacheMu.Lock()
	cached, ok := cmapCache[cMapName]
	cmapCacheMu.Unlock()
	if ok {
		return cached, nil
	}

	targetCmap, err := cmap.NewParser().ParsePredefined(cMapName)
	if err != nil {
		return nil, err
	}

	// limit the cache to predefined CMaps
	cmapCacheMu.Lock()
	cmapCache[targetCmap.Name()] = targetCmap
	cmapCacheMu.Unlock()
	return targetCmap, nil
}

// ParseCMap parses the given CMap.
//
// Port of org.apache.pdfbox.pdmodel.font.CMapManager.parseCMap.
func ParseCMap(randomAccessRead pdfio.RandomAccessRead) (*cmap.CMap, error) {
	if randomAccessRead == nil {
		return nil, nil
	}
	return cmap.NewParser().Parse(randomAccessRead)
}

// errExpectedNameOrStream is what readCMap reports for a /ToUnicode entry that
// is neither.
var errExpectedNameOrStream = errors.New("Expected Name or Stream")
