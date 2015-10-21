package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSingle(t *testing.T) {
	req, _ := http.NewRequest("GET", "", nil)
	w := httptest.NewRecorder()
	statusHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Home page didn't return %v", http.StatusOK)
	}
}

func TestForLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	req, _ := http.NewRequest("GET", "", nil)
	for i := 0; i < 1000000; i++ {
		w := httptest.NewRecorder()
		statusHandler(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Home page didn't return %v", http.StatusOK)
		}
		time.Sleep(time.Second)
	}
}
