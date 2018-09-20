package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

const (
	adminOrgName      = "d2hub"
	defaultServerPort = "5001"
)

var (
	d2hubURL string
	proxy    *httputil.ReverseProxy
)

func init() {
	d2hubURL = os.Getenv("D2HUB_URL")
	if d2hubURL == "" {
		log.Fatal("doesn't set D2HUB_URL env")
	}

	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		log.Fatal("doesn't set REGISTRY_URL env")
	}

	parsedRegistryURL, err := url.Parse(registryURL)
	if err != nil {
		return
	}

	proxy = httputil.NewSingleHostReverseProxy(parsedRegistryURL)
}

type defaultHandler struct {
}

func (d defaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Request] URL: %s, Method: %s\n", r.URL, r.Method)
	proxy.ServeHTTP(w, r)
}

func pullHandleFunc(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Request] URL: %s, Method: %s\n", r.URL, r.Method)
	orgName, repoName, tagName := parseURLVars(r)

	if res, err := http.Head(fmt.Sprintf("%s/api/orgs/%s/repos/%s", d2hubURL, orgName, repoName)); err == nil {
		defer res.Body.Close()
		if res.StatusCode == http.StatusNotFound {
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf(`{"errors":[{"code":"MANIFEST_UNKNOWN","message":"manifest unknown","detail":{"Tag":"%s"}}]}`, tagName)))
			return
		}
	}

	proxy.ServeHTTP(w, r)

	go func() {
		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/orgs/%s/repos/%s/pull/count", d2hubURL, orgName, repoName), nil)
		if err != nil {
			log.Printf("[ERROR] %s\n", err)
			return
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("[ERROR] %s\n", err)
			return
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			log.Printf("[WARN] status code is %d, except %d\n", res.StatusCode, http.StatusOK)
		}
	}()
}

func putImageHandler(w http.ResponseWriter, r *http.Request) {
	orgName, repoName, tagName := parseURLVars(r)
	proxy.ServeHTTP(w, r)

	go func() {
		postURL := fmt.Sprintf("%s/api/public/orgs/%s/repos/%s/tags/%s/push/event", d2hubURL, orgName, repoName, tagName)
		_, err := http.Post(postURL, "", nil)
		logrus.WithField("URL", postURL).Info("Call d2hub api")
		if err != nil {
			logrus.WithError(err).Error("Requesting a push event has failed")
		}
	}()
}

func parseURLVars(r *http.Request) (orgName, repoName, tagName string) {
	vars := mux.Vars(r)
	var ok bool
	orgName, ok = vars["orgName"]
	if !ok {
		orgName = adminOrgName
	}
	repoName = vars["repoName"]
	tagName = vars["tagName"]
	return
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/v2/{repoName}/manifests/{tagName}", pullHandleFunc).Methods("GET")
	r.HandleFunc("/v2/{orgName}/{repoName}/manifests/{tagName}", pullHandleFunc).Methods("GET")
	r.HandleFunc("/v2/{repoName}/manifests/{tagName}", putImageHandler).Methods("PUT")
	r.HandleFunc("/v2/{orgName}/{repoName}/manifests/{tagName}", putImageHandler).Methods("PUT")
	r.NotFoundHandler = defaultHandler{}
	http.Handle("/", r)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultServerPort
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
