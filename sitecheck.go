package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Name        string   `toml:"name"`
	Type        string   `toml:"type"`
	Description string   `yaml:"description"`
	Timeout     int      `toml:"timeout"`
	URL         []string `toml:"url"`
	state       []string
	last        time.Time
}

type Service struct {
	URL     string
	Timeout int
}

type URL struct {
	Name  string `json:"name"`
	State string `json:"state"`
	URL   string `json:"url"`
}

type Site struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URLs        []*URL `json:"children"`
}

type Sites struct {
	TopLevel string  `json:"name"`
	Sites    []*Site `json:"children"`
}

type Status interface {
	Check(Service) (bool, error)
}

var check map[string]Status

type server struct {
	configfile  string
	lastconfig  time.Time
	htmlfile    string
	lasthtml    time.Time
	cfg         []*Config
	sites       Sites
	html        []byte
	next_status time.Time
	last_status time.Time
	timer_epoch int
	timeout     int
	epoch       int
	sync.Mutex
}

const (
	Wait   = true
	NoWait = false
)

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

	log.Println("reading", s.configfile)

	data, err := ioutil.ReadFile(s.configfile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &s.cfg)
	if err != nil {
		return err
	}

	s.lastconfig = time.Now()

	for i, _ := range s.cfg {
		if s.cfg[i].Timeout == 0 {
			s.cfg[i].Timeout = s.timeout
		}
		s.cfg[i].state = make([]string, len(s.cfg[i].URL))
	}

	return nil
}

func (s *server) processHTML() error {
	s.Lock()
	defer s.Unlock()

	fi, err := os.Stat(s.htmlfile)
	if err != nil {
		return err
	}

	if s.lasthtml.After(fi.ModTime()) {
		return nil
	}

	log.Println("reading", s.htmlfile)

	jquery := regexp.MustCompile(`src="(jquery.*js)"`)
	d3 := regexp.MustCompile(`src="(d3.*js)"`)

	var buf bytes.Buffer

	html, err := ioutil.ReadFile(s.htmlfile)
	if err != nil {
		return err
	}

	rdjs := func(buf *bytes.Buffer, filename string) error {
		log.Println("reading", filename)
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		buf.WriteString("<script type=\"text/javascript\">\n")
		buf.Write(b)
		buf.WriteString("</script>\n")
		return nil
	}

	rd := bytes.NewBuffer(html)

	scanner := bufio.NewScanner(rd)

	for scanner.Scan() {
		text := scanner.Text()
		jqfile := jquery.FindStringSubmatch(text)
		d3file := d3.FindStringSubmatch(text)
		switch {
		case jqfile != nil && jqfile[1] != "":
			rdjs(&buf, jqfile[1])
		case d3file != nil && d3file[1] != "":
			rdjs(&buf, d3file[1])
		default:
			buf.WriteString(text)
			buf.WriteString("\n")
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	s.html = buf.Bytes()
	s.lasthtml = time.Now()

	return nil
}

func (s *server) processSites() {
	s.sites.Sites = make([]*Site, 0)
	for _, c := range s.cfg {
		urls := make([]*URL, 0)
		for i, u := range c.URL {
			urls = append(urls, &URL{Name: u, State: c.state[i], URL: u})
		}

		site := &Site{
			Name:        c.Name,
			Description: c.Description,
			URLs:        urls,
		}

		s.sites.Sites = append(s.sites.Sites, site)
	}
}

func (s *server) checkStatus(idx, url, epoch int, wg *sync.WaitGroup) {
	defer wg.Done()

	ck, ok := check[s.cfg[idx].Type]
	if ok == false {
		log.Println(s.cfg[idx].Type, s.cfg[idx].URL[url], "unknown type")
		return
	}

	serv := Service{
		Timeout: s.cfg[idx].Timeout,
		URL:     s.cfg[idx].URL[url],
	}

	healthy, err := ck.Check(serv)

	if epoch != s.epoch {
		log.Println("took too long - epoch has passed")
		return
	}

	if err == nil && healthy {
		s.Lock()
		s.cfg[idx].state[url] = "online"
		s.Unlock()
		return
	}

	s.Lock()
	s.cfg[idx].state[url] = "offline"
	s.Unlock()
	log.Println(s.cfg[idx].Type, s.cfg[idx].URL[url], err)
}

func (s *server) refresh(wait bool) {
	s.Lock()
	defer s.Unlock()

	if s.timer_epoch == s.epoch && s.next_status.After(time.Now()) {
		return
	}

	var wg sync.WaitGroup

	for i, _ := range s.cfg {
		for u, _ := range s.cfg[i].URL {
			s.cfg[i].state[u] = "unknown"
			wg.Add(1)
			go s.checkStatus(i, u, s.epoch, &wg)
		}
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

	s.processSites()
}

func (s *server) statusHandler(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Println("request from", host)

	s.parseConfig()
	s.processHTML()
	//	s.refresh(NoWait)

	w.Header().Set("Content-Type", "text/html")

	b := bytes.NewBuffer(s.html)
	io.Copy(w, b)
}

func (s *server) statusAPI(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Println("api req from", host)

	s.parseConfig()
	s.refresh(r.URL.Query().Get("wait") == "true")

	s.Lock()
	defer s.Unlock()

	w.Header().Set("Content-Type", "application/json")
	b, err := json.MarshalIndent(s.sites, "", "\t")
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
		"telnet":     new(Telnet),
		"consul":     new(Consul),
	}
}

func main() {
	var (
		port     = flag.String("port", "", "HTTP service address (.e.g. 8080)")
		conffile = flag.String("conf", "sitecheck.yml", "Configuration file")
		timeout  = flag.Int("timeout", 20, "default timeout")
	)

	flag.Parse()

	s := &server{
		configfile: *conffile,
		htmlfile:   "sitecheck.html",
		timeout:    *timeout,
	}

	mux := http.NewServeMux()
	mux.Handle("/", makeGzipHandler(s.statusHandler))
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
