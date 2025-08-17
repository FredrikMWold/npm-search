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
