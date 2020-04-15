package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var assetsPath string
	var proxyMap map[string]string
	var err error

	testCORS := flag.Bool("testcors", false, "Dump request headers and check if CORS is requested")
	port := flag.Int("port", 10000, "Port to listen")
	tlsCert := flag.String("tlscert", "", "SSL certificate to use HTTPS")
	tlsKey := flag.String("tlskey", "", "SSL certificate private key")
	agstatic := flag.String("agstatic", "", "Angular static serve")
	invproxy := flag.String("invproxy", "", "Inverse proxy access. Ej: /api|http://localhost:8000,/static|http://localhost:9000")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if *invproxy != "" {
		proxyMap, err = parseProxyMap(*invproxy)
		if err != nil {
			log.Fatalf("Can't parse inverse proxy map setup: %s", err)
		}
	}
	log.Printf("FooBarHttp")
	log.Printf("Listening at :%d", *port)
	log.Printf("Serving path: \"%s\"", flag.Arg(0))

	if *agstatic != "" {
		assetsPath = filepath.Join(*agstatic, "assets")
		// Check paths
		if _, err := os.Stat(*agstatic); os.IsNotExist(err) {
			log.Fatalf("Path directory \"%s\" not exists", *agstatic)
		}
		if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
			log.Fatalf("Assets path directory \"%s\" not exists", assetsPath)
		}
		log.Printf("Serving Angular dist static from: \"%s\"", *agstatic)
		log.Printf("with assets path: \"%s\"", assetsPath)
	}

	// Handlers
	//fs := http.FileServer(http.Dir(assetsPath))
	//http.HandleFunc(fmt.Sprintf("/%s", *relAssetsPath), http.StripPrefix(fmt.Sprintf("/%s/", *relAssetsPath), fs))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("URL: %s", r.URL)
		if *testCORS {
			log.Printf("--- Headers ---")
			for k, v := range r.Header {
				vc := strings.Join(v, " ")
				log.Printf("\t%s: %s", k, vc)
			}
			log.Printf("--- End headers ---")
		}
		// 1. Angular dist server?
		if *agstatic != "" {
			if r.URL.Path == "" || r.URL.Path == "/" || !strings.Contains(r.URL.Path, ".") {
				http.ServeFile(w, r, filepath.Join(*agstatic, "index.html"))
			} else {
				http.ServeFile(w, r, filepath.Join(*agstatic, r.URL.Path))
			}
		}
		// 2. Inverse proxy mode?
		if len(proxyMap) > 0 {
			for k, v := range proxyMap {
				if strings.HasPrefix(r.URL.Path, k) {
					log.Printf("DEBUG: proxy entry found: \"%s\"", k)
					err = serveProxyRequest(w, r, v)
					if err != nil {
						log.Printf("WARN: Proxy error: %s", err)
					}
					break
				}
			}
		}
		// 3. 404
		w.WriteHeader(404)
		w.Write([]byte("Not found"))

	})
	if *tlsKey != "" && *tlsCert != "" {
		log.Printf("Enabling HTTPS")
		http.ListenAndServeTLS(fmt.Sprintf(":%d", *port), *tlsCert, *tlsKey, nil)
	} else {
		http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	}

}
