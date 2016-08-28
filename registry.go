package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type Registry struct{}

func (w *Registry) Check(srv Service) (bool, error) {
	timeout := time.Duration(time.Duration(srv.Timeout) * time.Second)
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/v2/", nil)
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
		return false, errors.New("Bad status")
	}

	ver := resp.Header.Get("Docker-Distribution-API-Version")

	fmt.Println(ver)

	if ver != "registry/2.0" {
		return false, nil
	}

	return true, nil
}
