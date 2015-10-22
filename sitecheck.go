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

func checkStatus(site_status []status) {
	count := 0
	c := make(chan int)

	for i, s := range site_status {
		site_status[i].Status = "offline"

		ck, ok := check[s.Type]
		if ok == false {
			log.Println(s.Type, s.URL, "unknown type")
			continue
		}

		count++

		go func(site status, i int) {
			healthy, err := ck.Check(site.URL)
			if err == nil && healthy {
				site_status[i].Status = "online"
			}

			if err != nil {
				log.Println(site.Type, site.URL, err)
			}

			c <- i
		}(s, i)
	}

	for i := 0; i < count; i++ {
		<-c
	}
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

func statusHandler(w http.ResponseWriter, r *http.Request) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Println("request from", host)

	site_status, err := readConfig("sitecheck.conf")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	checkStatus(site_status)

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
	var port = flag.String("http", "", "HTTP service address (.e.g. :8080)")

	flag.Parse()

	if *port == "" {
		flag.Usage()
		return
	}

	http.HandleFunc("/", statusHandler)

	log.Fatal(http.ListenAndServe(*port, nil))
}
