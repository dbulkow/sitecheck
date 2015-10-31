package main

import (
	"bytes"
	"flag"
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
	Name   string `toml:"name"`
	Type   string `toml:"type"`
	Status string
	URL    string `toml:"url"`
}

type sites struct {
	Service []status
}

type Status interface {
	Check(url string) (bool, error)
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
	html        []byte
	sync.Mutex
}

func (s *server) initialize() error {
	var err error

	s.templ, err = template.ParseFiles(s.htmlfile)
	if err != nil {
		return err
	}

	return nil
}

func (s *server) readConfig() error {
	var config sites

	fi, err := os.Stat(s.configfile)
	if err != nil {
		return err
	}

	if s.lastconfig.After(fi.ModTime()) {
		return nil
	}

	_, err = toml.DecodeFile(s.configfile, &config)
	if err != nil {
		return err
	}

	s.lastconfig = time.Now()
	s.site_status = config.Service

	for i, _ := range s.site_status {
		s.site_status[i].Status = "unknown"
	}

	return nil
}

func (s *server) checkStatus() {
	var wg sync.WaitGroup

	for i, stat := range s.site_status {
		ck, ok := check[stat.Type]
		if ok == false {
			log.Println(stat.Type, stat.URL, "unknown type")
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			healthy, err := ck.Check(s.site_status[idx].URL)
			if err == nil && healthy {
				s.site_status[idx].Status = "online"
				return
			}

			s.site_status[idx].Status = "offline"
			log.Println(s.site_status[idx].Type, s.site_status[idx].URL, err)
		}(i)
	}

	wg.Wait()
}

func (s *server) updateStatus() error {
	s.Lock()
	defer s.Unlock()

	if s.next_status.Before(time.Now()) {
		err := s.readConfig()
		if err != nil {
			return err
		}

		s.checkStatus()

		s.last_status = time.Now()
		s.next_status = s.last_status.Add(time.Second * 5)
	}

	x := &struct {
		Status   []status
		DateTime string
	}{
		Status:   s.site_status,
		DateTime: s.last_status.String(),
	}

	b := &bytes.Buffer{}

	err := s.templ.Execute(b, x)
	if err != nil {
		return err
	}

	s.html, err = ioutil.ReadAll(b)
	if err != nil {
		return err
	}

	return nil
}

func (s *server) statusHandler(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Println("request from", host)

	s.updateStatus()

	b := bytes.NewBuffer(s.html)

	io.Copy(w, b)
}

func init() {
	check = map[string]Status{
		"website":  new(Website),
		"etcd":     new(Etcd),
		"docker":   new(Docker),
		"registry": new(Registry),
	}
}

func main() {
	var (
		port     = flag.String("port", "", "HTTP service address (.e.g. 8080)")
		conffile = flag.String("conf", "sitecheck.conf", "Configuration file")
	)

	flag.Parse()

	if *port == "" {
		flag.Usage()
		return
	}

	s := &server{
		configfile: *conffile,
		htmlfile:   "status.html",
	}
	err := s.initialize()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", s.statusHandler)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
