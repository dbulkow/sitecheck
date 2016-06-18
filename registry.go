package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

type Registry struct{}

func (w *Registry) Check(srv Service) (bool, error) {
	timeout := time.Duration(time.Duration(srv.Timeout) * time.Second)
	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(srv.URL + "/v2/")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, errors.New("Bad status")
	}

	ioutil.ReadAll(resp.Body)

	return true, nil
}
