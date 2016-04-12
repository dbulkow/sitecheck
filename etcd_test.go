package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckStatusEtcd(t *testing.T) {
	var f func(http.ResponseWriter, *http.Request)
	f = func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/v2/members":
			m1 := []memb{{
				ClientURLs: []string{"http://" + r.Host},
				ID:         "fumble",
				Name:       "newstuff",
				PeerURLs:   []string{"http://" + r.Host},
			}}
			m := &members{Members: m1}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal("Marshal failed")
			}
			fmt.Fprintf(w, string(b))
			return
		case "/health":
			b, err := json.Marshal(struct {
				Health string `json:"health"`
			}{Health: "true"})
			if err != nil {
				t.Fatal("Marshal failed")
			}
			fmt.Fprint(w, string(b))
			return
		}
		fmt.Fprintln(w, "hello there etcd user")
	}

	testCheckStatus(t, "etcd", f, successCheck)
}

func TestCheckStatusEtcdNoHealth(t *testing.T) {
	var f func(http.ResponseWriter, *http.Request)
	f = func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/v2/members":
			m1 := []memb{{
				ClientURLs: []string{"http://" + r.Host},
				ID:         "fumble",
				Name:       "newstuff",
				PeerURLs:   []string{"http://" + r.Host},
			}}
			m := &members{Members: m1}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal("Marshal failed")
			}
			fmt.Fprintf(w, string(b))
			return
		case "/health":
			http.NotFound(w, r)
			return
		}
		fmt.Fprintln(w, "hello there etcd user")
	}

	testCheckStatus(t, "etcd", f, failCheck)
}

func TestCheckStatusEtcdPoorHealth(t *testing.T) {
	var f func(http.ResponseWriter, *http.Request)
	f = func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/v2/members":
			m1 := []memb{{
				ClientURLs: []string{"http://" + r.Host},
				ID:         "fumble",
				Name:       "newstuff",
				PeerURLs:   []string{"http://" + r.Host},
			}}
			m := &members{Members: m1}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal("Marshal failed")
			}
			fmt.Fprintf(w, string(b))
			return
		case "/health":
			b, err := json.Marshal(struct {
				Health string `json:"health"`
			}{Health: "false"})
			if err != nil {
				t.Fatal("Marshal failed")
			}
			fmt.Fprint(w, string(b))
			return
		}
		fmt.Fprintln(w, "hello there etcd user")
	}

	testCheckStatus(t, "etcd", f, failCheck)
}

func TestCheckStatusEtcdNoHealthNoConnect(t *testing.T) {
	t.Skip("this test isn't working - hangs")

	var ts *httptest.Server

	var f func(http.ResponseWriter, *http.Request)
	f = func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/v2/members":
			m1 := []memb{{
				ClientURLs: []string{"http://" + r.Host},
				ID:         "fumble",
				Name:       "newstuff",
				PeerURLs:   []string{"http://" + r.Host},
			}}
			m := &members{Members: m1}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal("Marshal failed")
			}
			fmt.Fprintf(w, string(b))
			return
		case "/health":
			ts.Close()
			return
		}
		fmt.Fprintln(w, "hello there etcd user")
	}

	ts = httptest.NewServer(http.HandlerFunc(f))
	defer ts.Close()

	status := []status{{
		Name:    "SiteCheckTest",
		Type:    "etcd",
		URL:     ts.URL,
		Timeout: 20,
	}}

	s := &server{site_status: status}

	s.refresh(Wait)

	failCheck(t, status)
}

func TestCheckStatusEtcdBad(t *testing.T) {
	testCheckStatus(t, "etcd", testSimpleResponder, failCheck)
}

func TestCheckStatusEtcdBad2(t *testing.T) {
	testCheckStatus(t, "etcd", testBadResponder, failCheck)
}

func TestCheckStatusEtcdMissing(t *testing.T) {
	testCheckStatus(t, "etcd", nil, failCheck)
}

func TestCheckStatusEtcdBadHealth(t *testing.T) {
	var f func(http.ResponseWriter, *http.Request)
	f = func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/v2/members":
			m1 := []memb{{
				ClientURLs: []string{"http://" + r.Host},
				ID:         "fumble",
				Name:       "newstuff",
				PeerURLs:   []string{"http://" + r.Host},
			}}
			m := &members{Members: m1}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal("Marshal failed")
			}
			fmt.Fprintf(w, string(b))
			return
		}
		fmt.Fprintln(w, "hello there etcd user")
	}

	testCheckStatus(t, "etcd", f, failCheck)
}
