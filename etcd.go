package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Etcd struct{}

type memb struct {
	ClientURLs []string `json:"clientURLs"`
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	PeerURLs   []string `json:"peerURLs"`
}

type members struct {
	Members []memb `json:"members"`
}

// Read all members of an etcd cluster
func etcdMembers(url string, client *http.Client) (error, *members) {
	resp, err := client.Get(url + "/v2/members")
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("unsuccesful http status code"), nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, nil
	}

	members := &members{}

	err = json.Unmarshal(body, &members)
	if err != nil {
		return err, nil
	}

	return nil, members
}

// Iterate over all members looking for health
func (e *Etcd) Check(site status) (bool, error) {
	if site.Timeout == 0 {
		site.Timeout = 30
	}

	timeout := time.Duration(time.Duration(site.Timeout) * time.Second)
	client := &http.Client{Timeout: timeout}

	health := false

	err, members := etcdMembers(site.URL, client)
	if err != nil {
		return health, err
	}

	for _, m := range members.Members {
		for _, url := range m.ClientURLs {
			resp, err := client.Get(url + "/health")
			if err != nil {
				log.Printf("failed to check health of member %s on %s: %v\n", m.ID, url, err)
				continue
			}

			if resp.StatusCode != 200 {
				log.Printf("/health not found, member %s on %s\n", m.ID, url)
				continue
			}

			result := struct{ Health string }{}
			d := json.NewDecoder(resp.Body)
			err = d.Decode(&result)
			resp.Body.Close()
			if err != nil {
				fmt.Printf("failed to check the health of member %s on %s: %v\n", m.ID, url, err)
				continue
			}

			if result.Health == "true" {
				health = true
				// fmt.Printf("member %s is healthy: got healthy result from %s\n", m.ID, url)
			} else {
				fmt.Printf("member %s is unhealthy: got unhealthy result from %s\n", m.ID, url)
			}

			break
		}
	}

	return health, nil
}
