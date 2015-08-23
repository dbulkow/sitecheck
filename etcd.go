package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Members struct {
	Members []struct {
		ClientURLs []string `json:"clientURLs"`
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		PeerURLs   []string `json:"peerURLs"`
	} `json:"members"`
}

type member struct {
	ClientURLs []string
	ID         string
	Name       string
	PeerURLs   []string
}

type Stats struct {
	ID         string `json:"id"`
	LeaderInfo struct {
		Leader    string `json:"leader"`
		StartTime string `json:"startTime"`
		Uptime    string `json:"uptime"`
	} `json:"leaderInfo"`
	Name                 string  `json:"name"`
	RecvAppendRequestCnt int     `json:"recvAppendRequestCnt"`
	RecvBandwidthRate    float64 `json:"recvBandwidthRate"`
	RecvPkgRate          float64 `json:"recvPkgRate"`
	SendAppendRequestCnt int     `json:"sendAppendRequestCnt"`
	StartTime            string  `json:"startTime"`
	State                string  `json:"state"`
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

func getStats(m *member) *Stats {
	for _, url := range m.ClientURLs {
		resp, err := http.Get(url + "/v2/stats/self")
		if err != nil {
			log.Printf("failed to check health of member %s on %s: %v\n", m.ID, url, err)
			continue
		}

		stats := &Stats{}
		d := json.NewDecoder(resp.Body)
		err = d.Decode(&stats)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("failed to check the health of member %s on %s: %v\n", m.ID, url, err)
			continue
		}

		return stats
	}

	return nil
}

// Iterate over all members looking for health
func etcdStatus(url string) (error, bool) {
	err, members := etcdMembers(url)
	if err != nil {
		return err, false
	}

	// collect stats from all members
	s1 := make([]*Stats, 0)
	for _, m := range members.Members {
		member := &member{
			ClientURLs: m.ClientURLs,
			ID:         m.ID,
			Name:       m.Name,
			PeerURLs:   m.PeerURLs,
		}
		s := getStats(member)
		if s != nil {
			s1 = append(s1, s)
		}
	}

	time.Sleep(time.Millisecond * 500)

	// do it again
	s2 := make([]*Stats, 0)
	for _, m := range members.Members {
		member := &member{
			ClientURLs: m.ClientURLs,
			ID:         m.ID,
			Name:       m.Name,
			PeerURLs:   m.PeerURLs,
		}
		s := getStats(member)
		if s != nil {
			s2 = append(s2, s)
		}
	}

	for _, p1 := range s1 {
		found := false
		for _, p2 := range s2 {
			if strings.EqualFold(p2.ID, p1.ID) {
				found = true
				// making progress?
				if p2.RecvAppendRequestCnt == p1.RecvAppendRequestCnt && p2.SendAppendRequestCnt == p1.SendAppendRequestCnt {
					return nil, false
				}
			}
		}
		if !found {
			fmt.Println(p1.ID, "not found")
			return nil, false
		}
	}

	// if all still alive and making progress

	return nil, true
}
