package main

import (
	"errors"
	"fmt"
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

	ver := resp.Header.Get("Docker-Distribution-API-Version")

	ioutil.ReadAll(resp.Body)

	fmt.Println(ver)

	if ver != "registry/2.0" {
		return false, nil
	}

	return true, nil
}
