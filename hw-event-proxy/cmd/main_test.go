//go:build unittests
// +build unittests

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandleHwEventInvalidChars(t *testing.T) {
	b := []byte("€\u263a")
	err := handleHwEvent(b)
	expectedErr := "failed to unmarshal hw event"
	assert.Containsf(t, err.Error(), expectedErr,
		"expected error contains '%v', got %v", expectedErr, err.Error())
}

// verify handleHwEvent can handle large payload without crash
func TestHandleHwEvent64K(t *testing.T) {
	b := make([]byte, 65536)
	err := handleHwEvent(b)
	expectedErr := "failed to unmarshal hw event"
	assert.Containsf(t, err.Error(), expectedErr,
		"expected error contains '%v', got %v", expectedErr, err.Error())
}
