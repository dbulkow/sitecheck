package main

import "testing"

func TestCheckStatusWebsite(t *testing.T) {
	testCheckStatus(t, "website", testSimpleResponder, successCheck)
}

func TestCheckStatusWebsiteBad(t *testing.T) {
	testCheckStatus(t, "website", testBadResponder, failCheck)
}

func TestCheckStatusWebsiteMissing(t *testing.T) {
	testCheckStatus(t, "website", nil, failCheck)
}
