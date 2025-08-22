package commands

import "sync"

// pkgMetaCache caches NpmSearchObject per package name for this app session.
var pkgMetaCache = struct {
	mu sync.RWMutex
	m  map[string]NpmSearchObject
}{m: make(map[string]NpmSearchObject)}

func cacheGetPkg(name string) (NpmSearchObject, bool) {
	pkgMetaCache.mu.RLock()
	v, ok := pkgMetaCache.m[name]
	pkgMetaCache.mu.RUnlock()
	return v, ok
}

func cacheSetPkg(name string, obj NpmSearchObject) {
	pkgMetaCache.mu.Lock()
	pkgMetaCache.m[name] = obj
	pkgMetaCache.mu.Unlock()
}

// dlRangeCache caches downloads-over-time values per package and window days.
var dlRangeCache = struct {
	mu sync.RWMutex
	m  map[string][]float64 // key: pkg|days
}{m: make(map[string][]float64)}

// Optional secondary cache for time points could be added if needed later.

func cacheGetDLRange(key string) ([]float64, bool) {
	dlRangeCache.mu.RLock()
	v, ok := dlRangeCache.m[key]
	dlRangeCache.mu.RUnlock()
	return v, ok
}

func cacheSetDLRange(key string, values []float64) {
	dlRangeCache.mu.Lock()
	// store a copy to be safe
	vv := make([]float64, len(values))
	copy(vv, values)
	dlRangeCache.m[key] = vv
	dlRangeCache.mu.Unlock()
}
