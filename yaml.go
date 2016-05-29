package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Timeout int      `yaml:"timeout"`
	URL     []string `yaml:"url"`
}

type Sites struct {
	cfg      []*Config
	TopLevel string  `json:"name"`
	Sites    []*Site `json:"children"`
}

type URL struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type Site struct {
	Name string `json:"name"`
	URLs []*URL `json:"children"`
}

func (s *Sites) statusHandler(w http.ResponseWriter, r *http.Request) {
	b, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-type", "application/json")
	w.Write(b)
}

func main() {
	data, err := ioutil.ReadFile("sitecheck.yml")
	if err != nil {
		panic(err)
	}

	var sites Sites

	err = yaml.Unmarshal(data, &sites.cfg)
	if err != nil {
		panic(err)
	}

	sites.TopLevel = "sites"
	sites.Sites = make([]*Site, 0)
	for _, c := range sites.cfg {
		urls := make([]*URL, 0)
		for _, u := range c.URL {
			urls = append(urls, &URL{Name: u, Size: 10})
		}

		s := &Site{
			Name: c.Name,
			URLs: urls,
		}

		sites.Sites = append(sites.Sites, s)
	}

	http.HandleFunc("/sitecheck.json", sites.statusHandler)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(""))))

	log.Fatal(http.ListenAndServe(":9999", nil))
}
