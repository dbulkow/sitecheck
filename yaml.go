package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Timeout int      `yaml:"timeout"`
	URL     []string `yaml:"url"`
}

func main() {
	data, err := ioutil.ReadFile("sitecheck.yml")
	if err != nil {
		panic(err)
	}

	var cfg []Config

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		panic(err)
	}

	for _, c := range cfg {
		fmt.Println("name:", c.Name)
		fmt.Println("type:", c.Type)
		fmt.Println("url:")
		for _, u := range c.URL {
			fmt.Printf("  %s\n", u)
		}
		fmt.Println()
	}
}
