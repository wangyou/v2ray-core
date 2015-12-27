package main

import (
	"testing"

	v2testing "github.com/v2ray/v2ray-core/testing"
	"github.com/v2ray/v2ray-core/testing/assert"
)

func TestParseOS(t *testing.T) {
	v2testing.Current(t)

	assert.Pointer(parseOS("windows")).Equals(Windows)
	assert.Pointer(parseOS("macos")).Equals(MacOS)
	assert.Pointer(parseOS("linux")).Equals(Linux)
	assert.Pointer(parseOS("test")).Equals(UnknownOS)
}

func TestParseArch(t *testing.T) {
	v2testing.Current(t)

	assert.Pointer(parseArch("x86")).Equals(X86)
	assert.Pointer(parseArch("x64")).Equals(Amd64)
	assert.Pointer(parseArch("arm")).Equals(Arm)
	assert.Pointer(parseArch("arm64")).Equals(Arm64)
	assert.Pointer(parseArch("test")).Equals(UnknownArch)
}
