package cloudfoundry

import (
	"net/http"
	"strings"
)

// HandleCatalog handles the catalog HTTP request.
func HandleCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Write(GetCatalog().JSON())
}

// HandleServiceInstance handles the service_instances HTTP request.
func HandleServiceInstance(w http.ResponseWriter, r *http.Request) {
	url := strings.TrimLeft(r.URL.Path, "/v2/service_instances/")
	fields := strings.Split(url, "/")
	id := fields[0]
	if len(fields) == 1 && r.Method == "PUT" {
		// provision
	} else if len(fields) == 1 && r.Method == "PATCH" {
		// update
	} else if len(fields) == 3 && r.Method == "PUT" {
		// bind
	} else if len(fields) == 3 && r.Method == "DELETE" {
		// unbind
	} else if len(fields) == 1 && r.Method == "DELETE" {
		// deprovision
	}

	w.Header().Add("Content-Type", "application/json")
}

// Handler function for HandlerFunc.
func Handler(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/v2/catalog") {
		HandleCatalog(w, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/v2/service_instances") {
		HandleServiceInstance(w, r)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not found"))
}

// HandlerFunc implements the Cloud Foundry Service Broker API.
func HandlerFunc(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			h.ServeHTTP(w, r)
			return
		}
		Handler(w, r)
	}
}
