package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

type status struct {
	Name    string `json:"name" toml:"name"`
	Type    string `json:"type" toml:"type"`
	Status  string `json:"status"`
	URL     string `json:"url" toml:"url"`
	Timeout int    `json:"timeout" toml:"timeout"`
	Hash    string `json:"hash"`
	last    time.Time
}

type sites struct {
	Service []status
}

type Status interface {
	Check(stat status) (bool, error)
}

var check map[string]Status

type server struct {
	configfile  string
	lastconfig  time.Time
	htmlfile    string
	templ       *template.Template
	site_status []status
	next_status time.Time
	last_status time.Time
	timer_epoch int
	html        []byte
	timeout     int
	epoch       int
	sync.Mutex
}

const (
	Wait   = true
	NoWait = false
)

func (s *server) initialize() error {
	var err error

	s.templ, err = template.ParseFiles(s.htmlfile)
	if err != nil {
		return err
	}

	return nil
}

func genhash(name, url string) string {
	h := md5.New()
	io.WriteString(h, name)
	io.WriteString(h, url)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *server) genHtml() error {
	s.Lock()
	defer s.Unlock()

	b := &bytes.Buffer{}

	err := s.templ.Execute(b, s.site_status)
	if err != nil {
		return err
	}

	s.html, err = ioutil.ReadAll(b)
	if err != nil {
		return err
	}

	return nil
}

func (s *server) parseConfig() error {
	s.Lock()
	defer s.Unlock()

	fi, err := os.Stat(s.configfile)
	if err != nil {
		return err
	}

	if s.lastconfig.After(fi.ModTime()) {
		return nil
	}

	s.epoch += 1

	var config sites

	_, err = toml.DecodeFile(s.configfile, &config)
	if err != nil {
		return err
	}

	s.lastconfig = time.Now()
	s.site_status = config.Service

	for i, _ := range s.site_status {
		s.site_status[i].Hash = genhash(s.site_status[i].Name, s.site_status[i].URL)
		if s.site_status[i].Timeout == 0 {
			s.site_status[i].Timeout = s.timeout
		}
	}

	return nil
}

func (s *server) checkStatus(idx, epoch int, wg *sync.WaitGroup) {
	defer wg.Done()

	ck, ok := check[s.site_status[idx].Type]
	if ok == false {
		log.Println(s.site_status[idx].Type, s.site_status[idx].URL, "unknown type")
		return
	}

	healthy, err := ck.Check(s.site_status[idx])

	if epoch != s.epoch {
		fmt.Println("took too long - epoch has passed")
		return
	}

	if err == nil && healthy {
		s.Lock()
		s.site_status[idx].Status = "online"
		s.Unlock()
		return
	}

	s.Lock()
	s.site_status[idx].Status = "offline"
	s.Unlock()
	log.Println(s.site_status[idx].Type, s.site_status[idx].URL, err)
}

func (s *server) refresh(wait bool) {
	s.Lock()
	defer s.Unlock()

	if s.timer_epoch == s.epoch && s.next_status.After(time.Now()) {
		return
	}

	var wg sync.WaitGroup

	for i, _ := range s.site_status {
		s.site_status[i].Status = "unknown"
		wg.Add(1)
		go s.checkStatus(i, s.epoch, &wg)
	}

	if wait {
		// unlock around waitgroup to allow goroutines to complete
		s.Unlock()
		wg.Wait()
		s.Lock()
	}

	s.last_status = time.Now()
	s.next_status = s.last_status.Add(time.Minute)
	s.timer_epoch = s.epoch
}

func (s *server) statusHandler(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Println("request from", host)

	s.parseConfig()
	s.refresh(NoWait)
	s.genHtml()

	b := bytes.NewBuffer(s.html)
	io.Copy(w, b)
}

func (s *server) statusAPI(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Println("api req from", host)

	s.parseConfig()

	wait := NoWait
	if r.URL.Query().Get("wait") == "true" {
		wait = Wait
	}
	s.refresh(wait)

	s.Lock()
	defer s.Unlock()

	w.Header().Set("Content-Type", "application/json")
	b, err := json.MarshalIndent(s.site_status, "", "\t")
	if err != nil {
		h := &struct {
			Code    int    `json:"code"`
			Id      string `json:"id"`
			Message string `json:"message"`
			Detail  string `json:"detail"`
		}{
			Code:    http.StatusInternalServerError,
			Id:      "internalerror",
			Message: "internal error",
			Detail:  "Unable to Marshal jobs list",
		}
		b, _ := json.MarshalIndent(h, "", "\t")
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, string(b))
}

func init() {
	check = map[string]Status{
		"website":    new(Website),
		"etcd":       new(Etcd),
		"docker":     new(Docker),
		"swarm":      new(Swarm),
		"registry":   new(Registry),
		"subversion": new(Subversion),
	}
}

func main() {
	var (
		port     = flag.String("port", "", "HTTP service address (.e.g. 8080)")
		conffile = flag.String("conf", "sitecheck.conf", "Configuration file")
		timeout  = flag.Int("timeout", 20, "default timeout")
	)

	flag.Parse()

	if *port == "" {
		flag.Usage()
		return
	}

	s := &server{
		configfile: *conffile,
		htmlfile:   "status.html",
		timeout:    *timeout,
	}
	err := s.initialize()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.statusHandler)
	mux.HandleFunc("/status", s.statusAPI)

	srv := &http.Server{
		Addr:           ":" + *port,
		Handler:        mux,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSNextProto:   nil,
	}

	// go log.Fatal(srv.ListenAndServeTLS("cert.pem", "key.pem"))
	log.Fatal(srv.ListenAndServe())
}
