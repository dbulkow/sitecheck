package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func TestSendStatus(t *testing.T) {
	status := []status{{
		Name:   "SiteCheck",
		Type:   "website",
		Status: "online",
		URL:    "http://sitecheck.com",
	}}

	w := httptest.NewRecorder()
	sendStatus(w, status, "status.html")

	const (
		FindName = iota
		FindURL
		FindStatus
	)

	state := FindName
	checkCell := false

	z := html.NewTokenizer(w.Body)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			return
		case tt == html.StartTagToken:
			tok := z.Token()

			if tok.Data == "td" {
				checkCell = true
				continue
			}

			if state == FindURL && tok.Data == "a" {
				for _, a := range tok.Attr {
					if a.Key == "href" {
						if a.Val != "http://sitecheck.com" {
							t.Fatal("Incorrect URL in response", tok.Data)
						}
						state = FindStatus
					}
				}
				checkCell = false
				continue
			}
		case tt == html.EndTagToken:
			checkCell = false
		case tt == html.TextToken && checkCell:
			tok := z.Token()

			switch state {
			case FindName:
				if tok.Data != "SiteCheck" {
					t.Fatal("Incorrect Name in response", tok.Data)
				}
				state = FindURL
			case FindStatus:
				if tok.Data != "online" {
					t.Fatal("Incorrect Status", tok.Data)
				}
				return
			}
		}
	}
}

func testSimpleResponder(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello, client", r.URL)
}

func testCheckStatus(t *testing.T, sitetype string, handler func(http.ResponseWriter, *http.Request)) {
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	status := []status{{
		Name: "SiteCheck",
		Type: sitetype,
		URL:  ts.URL,
	}}

	checkStatus(status)

	if status[0].Status != "online" {
		t.Error("Status != online")
	}
}

func TestCheckStatusWebsite(t *testing.T) {
	testCheckStatus(t, "website", testSimpleResponder)
}

func TestCheckStatusRegistry(t *testing.T) {
	testCheckStatus(t, "registry", testSimpleResponder)
}

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

	testCheckStatus(t, "etcd", f)
}

func TestCheckStatusDocker(t *testing.T) {
	t.Skip("Need to fix test to share certs/keys")

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello docker guy", r.URL)
	}))
	defer ts.Close()

	status := []status{{
		Name: "SiteCheck",
		Type: "docker",
		URL:  ts.URL,
	}}

	checkStatus(status)

	if status[0].Status != "online" {
		t.Error("Status != online")
	}
}

func TestSingle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	req, _ := http.NewRequest("GET", "", nil)
	req.RemoteAddr = "sitecheck_test_single:"
	w := httptest.NewRecorder()
	statusHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Home page didn't return %v", http.StatusOK)
	}
}

func TestForLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	req, _ := http.NewRequest("GET", "", nil)
	req.RemoteAddr = "sitecheck_test:"
	for i := 0; i < 1000000; i++ {
		w := httptest.NewRecorder()
		statusHandler(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Home page didn't return %v", http.StatusOK)
		}
		time.Sleep(time.Second)
	}
}
