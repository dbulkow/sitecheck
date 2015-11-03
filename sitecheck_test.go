package main

import (
	"bytes"
	"fmt"
	"html/template"
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

func testCheckStatus(t *testing.T, sitetype string, handler func(http.ResponseWriter, *http.Request), checker func(*testing.T, []status)) {
	var ts *httptest.Server

	if handler != nil {
		ts = httptest.NewServer(http.HandlerFunc(handler))
		defer ts.Close()
	} else {
		ts = &httptest.Server{URL: "http://127.0.0.1:55555"}
	}

	status := []status{{
		Name: "SiteCheckTest",
		Type: sitetype,
		URL:  ts.URL,
	}}

	s := &server{site_status: status}

	s.checkStatus()

	checker(t, status)
}

func successCheck(t *testing.T, status []status) {
	if status[0].Status != "online" {
		t.Fatal("Status != online")
	}
}

func failCheck(t *testing.T, status []status) {
	if status[0].Status == "online" {
		t.Fatal("Status == online, expected another state")
	}
}

func TestSendStatus(t *testing.T) {
	status := []status{{
		Name:   "SiteCheckTest",
		Type:   "website",
		Status: "online",
		URL:    "http://sitecheck.com",
	}}

	w := httptest.NewRecorder()
	s := &server{
		htmlfile:    "status.html",
		site_status: status,
	}
	s.initialize()
	s.checkStatus()
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

func TestSendStatusParseFiles(t *testing.T) {
	s := &server{
		htmlfile: "mumble",
	}
	err := s.initialize()
	if err == nil {
		t.Error("expected sendStatus to return an error")
	}
}

func TestCheckMany(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testSimpleResponder))
	defer ts.Close()

	status := []status{
		{
			Name: "SiteCheckTest",
			Type: "website",
			URL:  ts.URL,
		},
		{
			Name: "SiteCheckTest2",
			Type: "registry",
			URL:  ts.URL,
		},
		{
			Name: "SiteCheckTest3",
			Type: "registry",
			URL:  ts.URL,
		},
		{
			Name: "SiteCheckTest4",
			Type: "registry",
			URL:  ts.URL,
		},
	}

	s := &server{site_status: status}

	s.checkStatus()

	for i := range status {
		if status[i].Status != "online" {
			t.Errorf("Status[%d] \"%s\" incorrect status: \"%s\"\n", i, status[i].Name, status[i].Status)
		}
	}
}

func TestCheckUnknownType(t *testing.T) {
	testCheckStatus(t, "fumble", nil, failCheck)
}

func TestReadConfig(t *testing.T) {
	s := &server{configfile: "mumble"}
	err := s.readConfig()
	if err == nil {
		t.Error("expected readConfig to fail")
	}
}

func TestUpdateStatusBadConfig(t *testing.T) {
	s := &server{configfile: "mumble", htmlfile: "status.html"}
	err := s.initialize()
	if err != nil {
		t.Fatal(err)
	}
	err = s.updateStatus()
	if err == nil {
		t.Error("expected updateStatus to fail")
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
		configfile: "sitecheck.conf",
		htmlfile:   "status.html",
	}
	err := s.initialize()
	if err != nil {
		t.Fatal(err)
	}
	s.statusHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Home page didn't return %v", http.StatusOK)
	}
}

func TestUpdateStatusBadExecute(t *testing.T) {
	var err error

	s := &server{configfile: "sitecheck.conf"}
	s.templ, err = template.New("test").Parse("{{.Hello}}")
	if err != nil {
		t.Fatal(err)
	}
	err = s.updateStatus()
	if err == nil {
		t.Error("expected a failure to execute the template")
	}
}
