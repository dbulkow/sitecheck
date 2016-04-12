package main

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"time"
)

type Subversion struct{}

func (s *Subversion) Check(site status) (bool, error) {
	var buf bytes.Buffer

	cmd := exec.Command("svn", "info", site.URL)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return false, err
	}

	c := make(chan error)

	go func() {
		err := cmd.Wait()
		log.Println("subversion failed:", err)
		c <- err
	}()

	timeout := time.Tick(time.Duration(site.Timeout))

	select {
	case err := <-c:
		if err != nil {
			return false, err
		}
	case <-timeout:
		return false, errors.New("timeout")
	}

	return true, nil
}
