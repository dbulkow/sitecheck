package main

import (
	"log"
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
	Check(url string) (error, bool)
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

		err, healthy := ck.Check(s.URL)
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

func main() {
	check = map[string]Status{
		"website":  new(Website),
		"etcd":     new(Etcd),
		"docker":   new(Docker),
		"registry": new(Registry),
	}

	http.HandleFunc("/", statusHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
