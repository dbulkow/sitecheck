package main

import (
	"log"
	"net/http"
	"text/template"
)

type Status struct {
	Name    string
	Service string
	Status  string
	URL     string
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	status := make([]Status, 0)

	status = append(status, Status{
		Name:    "etcd cluster",
		Service: "etcd",
		URL:     "http://fumble.foo.com:2379",
		Status:  "offline",
	})
	status = append(status, Status{
		Name:    "Jenkins",
		Service: "website",
		URL:     "http://fumble.foo.com/jenkins",
		Status:  "offline",
	})
	status = append(status, Status{
		Name:    "Jenkins",
		Service: "website",
		URL:     "http://fumble.foo.com/jenkins",
		Status:  "offline",
	})
	status = append(status, Status{
		Name:    "OpenGrok",
		Service: "website",
		URL:     "http://134.111.220.68:8080",
		Status:  "offline",
	})
	status = append(status, Status{
		Name:    "OpenGrok",
		Service: "website",
		URL:     "http://fumble.foo.com/source",
		Status:  "offline",
	})
	status = append(status, Status{
		Name:    "nginx",
		Service: "website",
		URL:     "http://fumble.foo.com",
		Status:  "offline",
	})
	status = append(status, Status{
		Name:    "nginx",
		Service: "website",
		URL:     "http://fumble.foo.com",
		Status:  "offline",
	})

	for i, s := range status {
		switch s.Service {
		case "etcd":
			err, healthy := etcdStatus(s.URL)
			if err != nil {
				log.Println(err)
				return
			}

			if healthy {
				status[i].Status = "online"
			}
		case "website":
			err, healthy := websiteStatus(s.URL)
			if err != nil {
				log.Println(err)
				return
			}

			if healthy {
				status[i].Status = "online"
			}
		}
	}

	t, err := template.ParseFiles("status.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	err = t.Execute(w, status)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Println(err)
		return
	}
}

func main() {
	http.HandleFunc("/", statusHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
