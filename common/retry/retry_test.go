package retry

import (
	"errors"
	"testing"
	"time"

	v2testing "github.com/v2ray/v2ray-core/testing"
	"github.com/v2ray/v2ray-core/testing/assert"
)

var (
	errorTestOnly = errors.New("This is a fake error.")
)

func TestNoRetry(t *testing.T) {
	v2testing.Current(t)

	startTime := time.Now().Unix()
	err := Timed(10, 100000).On(func() error {
		return nil
	})
	endTime := time.Now().Unix()

	assert.Error(err).IsNil()
	assert.Int64(endTime - startTime).AtLeast(0)
}

func TestRetryOnce(t *testing.T) {
	v2testing.Current(t)

	startTime := time.Now()
	called := 0
	err := Timed(10, 1000).On(func() error {
		if called == 0 {
			called++
			return errorTestOnly
		}
		return nil
	})
	duration := time.Since(startTime)

	assert.Error(err).IsNil()
	assert.Int64(int64(duration / time.Millisecond)).AtLeast(900)
}

func TestRetryMultiple(t *testing.T) {
	v2testing.Current(t)

	startTime := time.Now()
	called := 0
	err := Timed(10, 1000).On(func() error {
		if called < 5 {
			called++
			return errorTestOnly
		}
		return nil
	})
	duration := time.Since(startTime)

	assert.Error(err).IsNil()
	assert.Int64(int64(duration / time.Millisecond)).AtLeast(4900)
}

func TestRetryExhausted(t *testing.T) {
	v2testing.Current(t)

	startTime := time.Now()
	called := 0
	err := Timed(2, 1000).On(func() error {
		if called < 5 {
			called++
			return errorTestOnly
		}
		return nil
	})
	duration := time.Since(startTime)

	assert.Error(err).Equals(errorRetryFailed)
	assert.Int64(int64(duration / time.Millisecond)).AtLeast(1900)
}
