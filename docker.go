package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
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

func (d *Docker) Check(url string) (bool, error) {
	var err error
	var resp *http.Response

	d.once.Do(d.setupTLS)
	if d.transport != nil {
		timeout := time.Duration(30 * time.Second)
		client := &http.Client{Transport: d.transport, Timeout: timeout}

		resp, err = client.Get(url + "/info")
	} else {
		resp, err = http.Get(url + "/info")
	}
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, nil
	}

	ioutil.ReadAll(resp.Body)

	return true, nil
}
