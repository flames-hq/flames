package memcache_test

import (
	"testing"

	"github.com/flames-hq/flames/provider/cache"
	"github.com/flames-hq/flames/provider/cache/memcache"
	"github.com/flames-hq/flames/providertest/cachetest"
)

func TestConformance(t *testing.T) {
	cachetest.Run(t, func() cache.CacheStore { return memcache.New() })
}

func TestAtomicConformance(t *testing.T) {
	cachetest.RunAtomic(t, func() cache.AtomicCacheStore { return memcache.New() })
}
