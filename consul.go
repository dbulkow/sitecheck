package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type Consul struct{}

func (c *Consul) Check(srv Service) (bool, error) {
	timeout := time.Duration(time.Duration(srv.Timeout) * time.Second)
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		return false, fmt.Errorf("newrequest: %v", err)
	}

	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("client request: %v", err)
	}
	if resp == nil {
		return false, fmt.Errorf("empty response")
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return false, nil
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

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, fmt.Errorf("unmarshal: %v", err)
	}

	if len(data) < 1 {
		return false, errors.New("too few elements in response")
	}

	if data[0].Status != "passing" {
		return false, nil
	}

	return true, nil
}
