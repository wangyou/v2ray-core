package collect

import (
	"sync"
	"sync/atomic"

	"github.com/v2ray/v2ray-core/common/serial"
)

type Validity interface {
	IsValid() bool
}

type entry struct {
	key   string
	value Validity
}

type ValidityMap struct {
	sync.RWMutex
	cache   map[string]Validity
	opCount int32
}

func NewValidityMap(cleanupIntervalSec int) *ValidityMap {
	instance := &ValidityMap{
		cache: make(map[string]Validity),
	}
	return instance
}

func (this *ValidityMap) cleanup() {
	entry2Remove := make([]entry, 0, 128)
	this.RLock()
	for key, value := range this.cache {
		if !value.IsValid() {
			entry2Remove = append(entry2Remove, entry{
				key:   key,
				value: value,
			})
		}
	}
	this.RUnlock()

	for _, entry := range entry2Remove {
		if !entry.value.IsValid() {
			this.Lock()
			delete(this.cache, entry.key)
			this.Unlock()
		}
	}
}

func (this *ValidityMap) Set(key serial.String, value Validity) {
	this.Lock()
	this.cache[key.String()] = value
	this.Unlock()
	opCount := atomic.AddInt32(&this.opCount, 1)
	if opCount > 1000 {
		atomic.StoreInt32(&this.opCount, 0)
		go this.cleanup()
	}
}

func (this *ValidityMap) Get(key serial.String) Validity {
	this.RLock()
	defer this.RUnlock()
	if value, found := this.cache[key.String()]; found {
		return value
	}
	return nil
}
