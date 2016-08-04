package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

type Consul struct{}

func (c *Consul) Check(srv Service) (bool, error) {
	timeout := time.Duration(time.Duration(srv.Timeout) * time.Second)
	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(srv.URL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	type reply struct {
		CheckID     string  `json:"CheckID"`
		CreateIndex float64 `json:"CreateIndex"`
		ModifyIndex float64 `json:"ModifyIndex"`
		Name        string  `json:"Name"`
		Node        string  `json:"Node"`
		Notes       string  `json:"Notes"`
		Output      string  `json:"Output"`
		ServiceID   string  `json:"ServiceID"`
		ServiceName string  `json:"ServiceName"`
		Status      string  `json:"Status"`
	}

	data := make([]reply, 0)

	if err := json.Unmarshal(b, &data); err != nil {
		return false, err
	}

	if len(data) < 1 {
		return false, errors.New("too few elements in response")
	}

	if data[0].Status != "passing" {
		return false, nil
	}

	return true, nil
}
