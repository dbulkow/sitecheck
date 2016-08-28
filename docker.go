package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type Docker struct {
	transport *http.Transport
	once      sync.Once
}

func (d *Docker) setupTLS() {
	home := os.Getenv("HOME")
	if home == "" {
		log.Println("HOME environment not configured")
		return
	}

	certFile := home + "/.docker/cert.pem"
	keyFile := home + "/.docker/key.pem"
	caFile := home + "/.docker/ca.pem"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Println(err)
		return
	}

	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Println(err)
		return
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	d.transport = &http.Transport{TLSClientConfig: tlsConfig}
}

func (d *Docker) Check(srv Service) (bool, error) {
	var err error
	var resp *http.Response

	d.once.Do(d.setupTLS)

	expire := time.Now().Add(time.Duration(srv.Timeout) * time.Second)

	for time.Now().Before(expire) {
		timeout := time.Duration(5 * time.Second)
		client := &http.Client{Timeout: timeout}

		req, err := http.NewRequest(http.MethodGet, srv.URL+"/info", nil)
		if err != nil {
			return false, fmt.Errorf("newrequest: %v", err)
		}

		req.Close = true

		if d.transport != nil {
			client.Transport = d.transport
		}

		resp, err = client.Do(req)
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("client request: %v", err)
		}
		break
	}
	if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("empty response")
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return false, nil
	}

	return true, nil
}
