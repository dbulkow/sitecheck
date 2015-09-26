package main

import "net/http"

type Website struct{}

func (w *Website) Check(url string) (bool, error) {
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, nil
	}

	return true, nil
}
