package main

import (
	"errors"
	"net/http"
)

type Registry struct{}

func (w *Registry) Check(url string) (error, bool) {
	resp, err := http.Get(url + "/v2/")
	if err != nil {
		return err, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Bad status"), false
	}

	return nil, true
}
