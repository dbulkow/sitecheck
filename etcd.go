package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
func (e *Etcd) Check(srv Service) (bool, error) {
	if srv.Timeout == 0 {
		srv.Timeout = 30
	}

	timeout := time.Duration(time.Duration(srv.Timeout) * time.Second)
	client := &http.Client{Timeout: timeout}

	health := false

	err, members := etcdMembers(srv.URL, client)
	if err != nil {
		return health, err
	}

	for _, m := range members.Members {
		for _, url := range m.ClientURLs {
			req, err := http.NewRequest(http.MethodGet, url+"/health", nil)
			if err != nil {
				return false, fmt.Errorf("newrequest: %v", err)
			}

			req.Close = true

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("failed to check health of member %s on %s: %v\n", m.ID, url, err)
				continue
			}
			if resp == nil {
				log.Printf("response body empty, member %s on %s\n", m.ID, url)
				continue
			}

			if resp.StatusCode != 200 {
				log.Printf("/health not found, member %s on %s\n", m.ID, url)
				continue
			}

			result := struct{ Health string }{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			io.Copy(ioutil.Discard, resp.Body)
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
