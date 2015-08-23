package main

import "net/http"

type Website struct{}

func (w *Website) Check(url string) (error, bool) {
	resp, err := http.Get(url)
	if err != nil {
		return err, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, false
	}

	return nil, true
}
