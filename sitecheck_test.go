package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"golang.org/x/net/html"
)

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
			t.Errorf("Status[%d] \"%s\"incorrect status\n", i, status[i].Name)
		}
	}
}

func successCheck(t *testing.T, status []status) {
	if status[0].Status != "online" {
		t.Fatal("Status != online")
	}
}

func failCheck(t *testing.T, status []status) {
	if status[0].Status != "offline" {
		t.Fatal("Status != offline")
	}
}

func TestCheckStatusWebsite(t *testing.T) {
	testCheckStatus(t, "website", testSimpleResponder, successCheck)
}

func TestCheckStatusWebsiteBad(t *testing.T) {
	testCheckStatus(t, "website", testBadResponder, failCheck)
}

func TestCheckStatusWebsiteMissing(t *testing.T) {
	testCheckStatus(t, "website", nil, failCheck)
}

func TestCheckStatusRegistry(t *testing.T) {
	testCheckStatus(t, "registry", testSimpleResponder, successCheck)
}

func TestCheckStatusRegistryBad(t *testing.T) {
	testCheckStatus(t, "registry", testBadResponder, failCheck)
}

func TestCheckStatusRegistryMissing(t *testing.T) {
	testCheckStatus(t, "registry", nil, failCheck)
}

func TestCheckUnknownType(t *testing.T) {
	testCheckStatus(t, "fumble", nil, failCheck)
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
		Name: "SiteCheckTest",
		Type: "etcd",
		URL:  ts.URL,
	}}

	s := &server{site_status: status}

	s.checkStatus()

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

func TestReadConfig(t *testing.T) {
	s := &server{configfile: "mumble"}
	err := s.readConfig()
	if err == nil {
		t.Error("expected readConfig to fail")
	}
}

func TestUpdateStatusBadConfig(t *testing.T) {
	s := &server{configfile: "mumble"}
	err := s.updateStatus()
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
