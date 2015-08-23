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
		switch s.Type {
		case "etcd":
			err, healthy := etcdStatus(s.URL)
			if err != nil {
				log.Println(err)
				return
			}

			if healthy {
				status[i].Status = "online"
			}
		case "website":
			err, healthy := websiteStatus(s.URL)
			if err != nil {
				log.Println(err)
				return
			}

			if healthy {
				status[i].Status = "online"
			}
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
	http.HandleFunc("/", statusHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
