package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/html"
)

func testSimpleResponder(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello, client", r.URL)
}

func testBadResponder(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func testCheckStatus(t *testing.T, sitetype string, handler func(http.ResponseWriter, *http.Request), checker func(*testing.T, []*Config)) {
	var ts *httptest.Server

	if handler != nil {
		ts = httptest.NewServer(http.HandlerFunc(handler))
		defer ts.Close()
	} else {
		ts = &httptest.Server{URL: "http://127.0.0.1:55555"}
	}

	cfg := []*Config{{
		Name:    "SiteCheckTest",
		Type:    sitetype,
		URL:     []string{ts.URL},
		state:   []string{"unknown"},
		Timeout: 20,
	}}

	s := &server{cfg: cfg}

	s.refresh(Wait)

	checker(t, cfg)
}

func successCheck(t *testing.T, cfg []*Config) {
	if cfg[0].state[0] != "online" {
		t.Fatal("Status != online")
	}
}

func failCheck(t *testing.T, cfg []*Config) {
	if cfg[0].state[0] == "online" {
		t.Fatal("Status == online, expected another state")
	}
}

func TestSendStatus(t *testing.T) {
	cfg := []*Config{{
		Name:    "SiteCheckTest",
		Type:    "website",
		state:   []string{"online"},
		URL:     []string{"http://sitecheck.com"},
		Timeout: 20,
	}}

	w := httptest.NewRecorder()
	s := &server{
		htmlfile: "sitecheck.html",
		cfg:      cfg,
	}
	s.refresh(Wait)
	b := bytes.NewBuffer(s.html)
	io.Copy(w, b)

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
				if tok.Data != "SiteCheckTest" {
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

func TestCheckMany(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testSimpleResponder))
	defer ts.Close()

	cfg := []*Config{
		{
			Name:    "SiteCheckTest",
			Type:    "website",
			URL:     []string{ts.URL},
			state:   []string{"unknown"},
			Timeout: 20,
		},
		{
			Name:    "SiteCheckTest2",
			Type:    "registry",
			URL:     []string{ts.URL},
			state:   []string{"unknown"},
			Timeout: 20,
		},
		{
			Name:    "SiteCheckTest3",
			Type:    "registry",
			URL:     []string{ts.URL},
			state:   []string{"unknown"},
			Timeout: 20,
		},
		{
			Name:    "SiteCheckTest4",
			Type:    "registry",
			URL:     []string{ts.URL},
			state:   []string{"unknown"},
			Timeout: 20,
		},
	}

	s := &server{cfg: cfg}

	s.refresh(Wait)

	for i := range cfg {
		if cfg[i].state[0] != "online" {
			t.Errorf("Status[%d] \"%s\" incorrect status: \"%s\"\n", i, cfg[i].Name, cfg[i].state[0])
		}
	}
}

func TestCheckUnknownType(t *testing.T) {
	testCheckStatus(t, "fumble", nil, failCheck)
}

func TestReadConfig(t *testing.T) {
	s := &server{configfile: "mumble"}
	err := s.parseConfig()
	if err == nil {
		t.Error("expected readConfig to fail")
	}
}

func TestUpdateStatusBadConfig(t *testing.T) {
	s := &server{configfile: "mumble"}
	if err := s.parseConfig(); err == nil {
		t.Error("expected parseConfig to fail")
	}
}

func TestSingleRealWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	req, _ := http.NewRequest("GET", "", nil)
	req.RemoteAddr = "sitecheck_test_single:"
	w := httptest.NewRecorder()
	s := &server{
		configfile: "sitecheck.yml",
		htmlfile:   "sitecheck.html",
		timeout:    20,
	}
	s.statusHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Home page didn't return %v", http.StatusOK)
	}
}
