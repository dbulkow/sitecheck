package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
)

type Docker struct{}

func (d *Docker) Check(url string) (bool, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return false, errors.New("HOME environment not configured")
	}

	certFile := home + "/.docker/cert.pem"
	keyFile := home + "/.docker/key.pem"
	caFile := home + "/.docker/ca.pem"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return false, err
	}

	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return false, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	resp, err := client.Get(url + "/info")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, nil
	}
	/*
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			log.Println(url+"/info", string(body))
		}
	*/
	return true, nil
}
