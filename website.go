package main

import (
	"net/http"
	"time"
)

type Website struct{}

func (w *Website) Check(url string) (bool, error) {
	timeout := time.Duration(20 * time.Second)
	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, nil
	}

	return true, nil
}
