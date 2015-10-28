package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"sync"
	"text/template"
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

func readConfig(conf string) ([]status, error) {
	var config sites

	_, err := toml.DecodeFile(conf, &config)
	if err != nil {
		return nil, err
	}

	return config.Service, nil
}

func checkStatus(site_status []status) {
	var wg sync.WaitGroup

	for i, s := range site_status {
		site_status[i].Status = "offline"

		ck, ok := check[s.Type]
		if ok == false {
			log.Println(s.Type, s.URL, "unknown type")
			continue
		}

		wg.Add(1)

		go func(site status, i int) {
			defer wg.Done()

			healthy, err := ck.Check(site.URL)
			if err == nil && healthy {
				site_status[i].Status = "online"
			}

			if err != nil {
				log.Println(site.Type, site.URL, err)
			}
		}(s, i)
	}

	wg.Wait()
}

func sendStatus(w http.ResponseWriter, site_status []status, file string) error {
	t, err := template.ParseFiles(file)
	if err != nil {
		return err
	}

	err = t.Execute(w, site_status)
	if err != nil {
		return err
	}

	return nil
}

var site_status []status
var next_status time.Time
var lock sync.Mutex

func statusHandler(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()

	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Println("request from", host)

	var err error

	if next_status.Before(time.Now()) {
		site_status, err = readConfig("sitecheck.conf")
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		checkStatus(site_status)

		next_status = time.Now().Add(time.Second * 30)
	}

	err = sendStatus(w, site_status, "status.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Println(err)
		return
	}
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
	var port = flag.String("port", "", "HTTP service address (.e.g. 8080)")

	flag.Parse()

	if *port == "" {
		flag.Usage()
		return
	}

	http.HandleFunc("/", statusHandler)

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
