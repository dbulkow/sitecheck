package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"text/template"

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

func readConfig() ([]status, error) {
	var config sites

	_, err := toml.DecodeFile("sitecheck.conf", &config)
	if err != nil {
		return nil, err
	}

	return config.Service, nil
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	hosts, err := net.LookupAddr(host)
	if err != nil {
		log.Println("request from", r.RemoteAddr)
	} else {
		log.Println("request from", hosts)
	}

	status, err := readConfig()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	for i, s := range status {
		status[i].Status = "offline"

		ck, ok := check[s.Type]
		if ok == false {
			log.Println(s.Type, s.URL, err)
			continue
		}

		healthy, err := ck.Check(s.URL)
		if err != nil {
			log.Println(s.Type, s.URL, err)
			continue
		}

		if healthy {
			status[i].Status = "online"
		}
	}

	t, err := template.ParseFiles("status.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	err = t.Execute(w, status)
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
	var port = flag.String("http", "", "HTTP service address (.e.g. :8080)")

	flag.Parse()

	if *port == "" {
		flag.Usage()
		return
	}

	http.HandleFunc("/", statusHandler)

	log.Fatal(http.ListenAndServe(*port, nil))
}
