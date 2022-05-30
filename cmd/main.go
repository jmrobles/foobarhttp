package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jmrobles/foobarhttp/pkg/proxy"
	log "github.com/sirupsen/logrus"
)

const listen_port = 8000

func main() {
	var err error
	epMode := flag.Bool("endpoint", false, "endpoint mode")
	flag.Parse()

	log.Info("foobarhttp proxy")
	log.SetLevel(log.DebugLevel)
	if *epMode {
		log.Printf("endpoint mode!")
		if len(flag.Args()) != 2 {
			log.Printf("You need to specify port and content")
			return
		}
		port, err := strconv.Atoi(flag.Arg(0))
		if err != nil {
			log.Fatalf("Port must be a number")
		}
		content := flag.Arg(1)

		runEndpointMode(port, content)
		return
	}
	httpProxy := proxy.NewHttpProxy()
	url1, _ := url.Parse("http://localhost:5001")
	rp := proxy.ReplacePath{Prefix: "/api/v1", Value: ""}
	rs1 := proxy.RouteSpec{Path: "/api/v1/core", Target: *url1, ReplacePath: &rp}
	httpProxy.AddRoute(&rs1)
	url2, _ := url.Parse("http://localhost:5002")
	rs2 := proxy.RouteSpec{Path: "/workshop", Target: *url2}
	httpProxy.AddRoute(&rs2)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Path: %s", r.URL.Path)
		err := httpProxy.Handle(w, r)
		if err != nil {
			log.Warningf("Error processing request: %s", err)
		}
	})
	listenAddr := fmt.Sprintf(":%d", listen_port)
	log.Infof("Listening on: %s", listenAddr)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatalf("Can't listen in port %d", listen_port)
	}

}

func runEndpointMode(port int, content string) {

	listenAddr := fmt.Sprintf(":%d", port)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		contentType := "text/html"
		if IsJSON(content) {
			contentType = "application/json"
		}
		log.Infof("Request! Path: %s - Method: %s", r.URL, r.Method)
		log.Info("Headers:")
		for k, v := range r.Header {
			values := strings.Join(v, ",")
			log.Infof("\t%s: %v", k, values)
		}
		data := make([]byte, 1024)
		nLen, err := r.Body.Read(data)
		if err != nil && err != io.EOF {
			log.Warningf("Can't read body: %s", err)
		}
		defer r.Body.Close()
		if nLen > 0 {
			log.Infof("Data (%d): %s", nLen, data)
		}
		w.Header().Add("Content-type", contentType)
		w.Header().Add("X-Served-By", "foobarhttp")
		w.WriteHeader(200)
		w.Write([]byte(content))
	})
	log.Infof("EP mode listening at: %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("Can't bind and listen: %s", err)
	}
}

func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}
