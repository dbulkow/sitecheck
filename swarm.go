package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Swarm struct {
	transport *http.Transport
	once      sync.Once
}

type swarminfo struct {
	BridgeNfIp6tables  bool        `json:"BridgeNfIp6tables"`
	BridgeNfIptables   bool        `json:"BridgeNfIptables"`
	Containers         float64     `json:"Containers"`
	Debug              bool        `json:"Debug"`
	DockerRootDir      string      `json:"DockerRootDir"`
	Driver             string      `json:"Driver"`
	DriverStatus       [][]string  `json:"DriverStatus"`
	ExecutionDriver    string      `json:"ExecutionDriver"`
	HttpProxy          string      `json:"HttpProxy"`
	HttpsProxy         string      `json:"HttpsProxy"`
	ID                 string      `json:"ID"`
	IPv4Forwarding     bool        `json:"IPv4Forwarding"`
	Images             float64     `json:"Images"`
	IndexServerAddress string      `json:"IndexServerAddress"`
	InitPath           string      `json:"InitPath"`
	InitSha1           string      `json:"InitSha1"`
	KernelVersion      string      `json:"KernelVersion"`
	Labels             interface{} `json:"Labels"`
	MemTotal           float64     `json:"MemTotal"`
	MemoryLimit        bool        `json:"MemoryLimit"`
	NCPU               float64     `json:"NCPU"`
	NEventsListener    float64     `json:"NEventsListener"`
	NFd                float64     `json:"NFd"`
	NGoroutines        float64     `json:"NGoroutines"`
	Name               string      `json:"Name"`
	NoProxy            string      `json:"NoProxy"`
	OperatingSystem    string      `json:"OperatingSystem"`
	SwapLimit          bool        `json:"SwapLimit"`
	SystemTime         string      `json:"SystemTime"`
}

func (s *Swarm) setupTLS() {
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
	s.transport = &http.Transport{TLSClientConfig: tlsConfig}
}

func (s *Swarm) Check(site status) (bool, error) {
	var err error
	var resp *http.Response

	s.once.Do(s.setupTLS)

	timeout := time.Duration(time.Duration(site.Timeout) * time.Second)
	client := &http.Client{Timeout: timeout}

	if s.transport != nil {
		client.Transport = s.transport
		resp, err = client.Get(site.URL + "/info")
	} else {
		resp, err = http.Get(site.URL + "/info")
	}
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("response status %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	info := &swarminfo{}

	err = json.Unmarshal(b, info)
	if err != nil {
		return false, err
	}
	/*
		for _, x := range info.DriverStatus {
			for _, y := range x {
				fmt.Println(y)
			}
		}
	*/
	return true, nil
}
