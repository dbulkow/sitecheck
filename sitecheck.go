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

func readConfig(conf string) ([]status, error) {
	var config sites

	_, err := toml.DecodeFile(conf, &config)
	if err != nil {
		return nil, err
	}

	return config.Service, nil
}

func checkStatus(status []status) {
	for i, s := range status {
		status[i].Status = "offline"

		ck, ok := check[s.Type]
		if ok == false {
			log.Println(s.Type, s.URL)
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
}

func sendStatus(w http.ResponseWriter, status []status, file string) error {
	t, err := template.ParseFiles(file)
	if err != nil {
		return err
	}

	err = t.Execute(w, status)
	if err != nil {
		return err
	}

	return nil
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Println("request from", host)

	status, err := readConfig("sitecheck.conf")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	checkStatus(status)

	err = sendStatus(w, status, "status.html")
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
