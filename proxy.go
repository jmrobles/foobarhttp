package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func parseProxyMap(arg string) (map[string]string, error) {
	ret := make(map[string]string)
	parts := strings.Split(arg, ",")
	for _, part := range parts {
		item := strings.Split(part, "|")
		if len(item) != 2 {
			return nil, fmt.Errorf("Invalid inverse proxy format")
		}
		if _, err := url.Parse(item[0]); err != nil {
			return nil, fmt.Errorf("Invalid inverse proxy format: %s", err)
		}
		if _, err := url.Parse(item[1]); err != nil {
			return nil, fmt.Errorf("Invalid inverse proxy format: %s", err)
		}
		// TODO: improve more validation tests
		ret[item[0]] = item[1]
	}
	return ret, nil
}

func serveProxyRequest(w http.ResponseWriter, r *http.Request, proxyPath string) error {
	var err error

	client := &http.Client{
		Timeout: 180 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	urlProxy, _ := url.Parse(proxyPath)
	targetURL := fmt.Sprintf("%s://%s%s%s", urlProxy.Scheme, urlProxy.Host, urlProxy.Path, r.URL.Path)
	log.Printf("DEBUG: Target URL: \"%s\"", targetURL)
	r.RequestURI = ""
	r.URL, err = url.Parse(targetURL)
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if prior, ok := r.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		r.Header.Set("X-Forwarded-For", clientIP)
	}
	resp, err := client.Do(r)
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return err
	}
	defer resp.Body.Close()
	delHopHeaders(resp.Header)

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	return nil
}
