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
	testCORS := flag.Bool("testcors", false, "Dump request headers and check if CORS is requested")
	port := flag.Int("port", 10000, "Port to listen")
	tlsCert := flag.String("tlscert", "", "SSL certificate to use HTTPS")
	tlsKey := flag.String("tlskey", "", "SSL certificate private key")
	relAssetsPath := flag.String("assets", "assets", "Assets path")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <path>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		fmt.Fprint(os.Stderr, "need to specify the path to serve\n")
		os.Exit(-1)
	}

	path := flag.Arg(0)
	assetsPath := filepath.Join(path, *relAssetsPath)
	// Check paths
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatalf("Path directory \"%s\" not exists", path)
	}
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		log.Fatalf("Assets path directory \"%s\" not exists", assetsPath)
	}
	log.Printf("FooBarHttp")
	log.Printf("Listening at :%d", *port)
	log.Printf("Serving path: \"%s\"", flag.Arg(0))
	log.Printf("Assets path: \"%s\"", assetsPath)
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
		if r.URL.Path == "" || r.URL.Path == "/" || !strings.Contains(r.URL.Path, ".") {
			http.ServeFile(w, r, filepath.Join(path, "index.html"))
		} else {
			http.ServeFile(w, r, filepath.Join(path, r.URL.Path))
		}
	})
	if *tlsKey != "" && *tlsCert != "" {
		log.Printf("Enabling HTTPS")
		http.ListenAndServeTLS(fmt.Sprintf(":%d", *port), *tlsCert, *tlsKey, nil)
	} else {
		http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	}

}
