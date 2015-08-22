package main

import (
	"log"
	"net/http"
	"text/template"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("status.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	status := []struct {
		Name    string
		Service string
		Status  string
	}{
		{
			Name:    "name1",
			Service: "service1",
			Status:  "status1",
		},
		{
			Name:    "name2",
			Service: "service2",
			Status:  "status2",
		},
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
