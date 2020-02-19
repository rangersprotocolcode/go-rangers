package common

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru"
)


// MustNewLRUCache creates a new lru cache.
// Caution: if fail, the function will cause panic
func CreateLRUCache(size int) *lru.Cache {
	cache, err := lru.New(size)
	if err != nil {
		panic(fmt.Errorf("new cache fail:%v", err))
	}
	return cache
}
