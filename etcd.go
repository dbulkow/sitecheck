package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Members struct {
	Members []struct {
		ClientURLs []string `json:"clientURLs"`
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		PeerURLs   []string `json:"peerURLs"`
	} `json:"members"`
}

// Read all members of an etcd cluster
func etcdMembers(url string) (error, *Members) {
	resp, err := http.Get(url + "/v2/members")
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, nil
	}

	members := &Members{}

	err = json.Unmarshal(body, &members)
	if err != nil {
		return err, nil
	}

	return nil, members
}

// Iterate over all members looking for health
func etcdStatus(url string) (error, bool) {
	health := false

	err, members := etcdMembers(url)
	if err != nil {
		return err, health
	}

	for _, m := range members.Members {
		for _, url := range m.ClientURLs {
			resp, err := http.Get(url + "/health")
			if err != nil {
				log.Printf("failed to check health of member %s on %s: %v\n", m.ID, url, err)
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
				fmt.Printf("member %s is healthy: got healthy result from %s\n", m.ID, url)
			} else {
				fmt.Printf("member %s is unhealthy: got unhealthy result from %s\n", m.ID, url)
			}

			break
		}
	}

	return nil, health
}
