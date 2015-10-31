package main

import "testing"

func TestCheckStatusRegistry(t *testing.T) {
	testCheckStatus(t, "registry", testSimpleResponder, successCheck)
}

func TestCheckStatusRegistryBad(t *testing.T) {
	testCheckStatus(t, "registry", testBadResponder, failCheck)
}

func TestCheckStatusRegistryMissing(t *testing.T) {
	testCheckStatus(t, "registry", nil, failCheck)
}
