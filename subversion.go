package main

import (
	"bytes"
	"errors"
	"os/exec"
	"time"
)

type Subversion struct{}

func (s *Subversion) Check(srv Service) (bool, error) {
	var buf bytes.Buffer

	cmd := exec.Command("svn", "info", srv.URL)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return false, err
	}

	c := make(chan error)

	go func() { c <- cmd.Wait() }()

	ticker := time.NewTicker(time.Duration(srv.Timeout) * time.Second)

	select {
	case err := <-c:
		if err != nil {
			return false, err
		}
	case <-ticker.C:
		ticker.Stop()
		return false, errors.New("timeout")
	}

	return true, nil
}
