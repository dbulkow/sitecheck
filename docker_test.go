package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCheckStatusDocker(t *testing.T) {
	t.Skip("Need to fix test to share certs/keys with docker module")

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello docker guy", r.URL)
	}))
	defer ts.Close()

	status := []status{{
		Name: "SiteCheckTest",
		Type: "docker",
		URL:  ts.URL,
	}}

	s := &server{site_status: status}

	s.checkStatus()

	successCheck(t, status)
}

func TestDockerMissing(t *testing.T) {
	testCheckStatus(t, "docker", nil, failCheck)
}

func TestDockerBad(t *testing.T) {
	testCheckStatus(t, "docker", testBadResponder, failCheck)
}

func TestDockerHOME(t *testing.T) {
	os.Setenv("HOME", "")
	d := &Docker{}
	d.setupTLS()
}

func TestDockerNoCertFile(t *testing.T) {
	os.Setenv("HOME", "/tmp")
	d := &Docker{}
	d.setupTLS()
}

func TestCheckStatusDockerRealWorld(t *testing.T) {
	t.Skip("move along")
	status := []status{{
		Name: "SiteCheckTest",
		Type: "docker",
		URL:  "https://fumble.foo.com:2376",
	}}

	for i := 0; i < 1000; i++ {
		s := &server{site_status: status}

		s.checkStatus()

		successCheck(t, status)

		//time.Sleep(time.Millisecond * 10)
	}
}
