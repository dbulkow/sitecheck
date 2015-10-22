package main

import (
	"errors"
	"io/ioutil"
	"net/http"
)

type Registry struct{}

func (w *Registry) Check(url string) (bool, error) {
	resp, err := http.Get(url + "/v2/")
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
