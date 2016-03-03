package rules

import (
	"errors"
	"time"

	"github.com/v2ray/v2ray-core/app/router"
	"github.com/v2ray/v2ray-core/common/collect"
	v2net "github.com/v2ray/v2ray-core/common/net"
)

var (
	ErrorInvalidRule      = errors.New("Invalid Rule")
	ErrorNoRuleApplicable = errors.New("No rule applicable")
)

type cacheEntry struct {
	tag        string
	err        error
	validUntil time.Time
}

func newCacheEntry(tag string, err error) *cacheEntry {
	this := &cacheEntry{
		tag: tag,
		err: err,
	}
	this.Extend()
	return this
}

func (this *cacheEntry) IsValid() bool {
	return this.validUntil.Before(time.Now())
}

func (this *cacheEntry) Extend() {
	this.validUntil = time.Now().Add(time.Hour)
}

type Router struct {
	config *RouterRuleConfig
	cache  *collect.ValidityMap
}

func NewRouter(config *RouterRuleConfig) *Router {
	return &Router{
		config: config,
		cache:  collect.NewValidityMap(3600),
	}
}

func (this *Router) takeDetourWithoutCache(dest v2net.Destination) (string, error) {
	for _, rule := range this.config.Rules {
		if rule.Apply(dest) {
			return rule.Tag, nil
		}
	}
	return "", ErrorNoRuleApplicable
}

func (this *Router) TakeDetour(dest v2net.Destination) (string, error) {
	rawEntry := this.cache.Get(dest)
	if rawEntry == nil {
		tag, err := this.takeDetourWithoutCache(dest)
		this.cache.Set(dest, newCacheEntry(tag, err))
		return tag, err
	}
	entry := rawEntry.(*cacheEntry)
	return entry.tag, entry.err
}

type RouterFactory struct {
}

func (this *RouterFactory) Create(rawConfig interface{}) (router.Router, error) {
	return NewRouter(rawConfig.(*RouterRuleConfig)), nil
}

func init() {
	router.RegisterRouter("rules", &RouterFactory{})
}
