package proxy

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// ReplacePath
type ReplacePath struct {
	Prefix string
	Value  string
}

// RouterSpec
type RouteSpec struct {
	Path        string
	Target      url.URL
	ReplacePath *ReplacePath
}

// type ProcessFunc func(w http.ResponseWriter, r *http.Request) error

// type Processor struct {
// 	ProcessorFunc ProcessFunc
// }

type Processor interface {
	Process(nested Processor, w http.ResponseWriter, r *http.Request) error
}

type WrappedProcessor struct {
	Processor       Processor
	NestedProcessor *WrappedProcessor
}

// HttpProxy
type HttpProxy struct {
	routes     []*RouteSpec
	processors []Processor
}

// NewHttpProxy
func NewHttpProxy() *HttpProxy {
	return &HttpProxy{}
}

// AddRoute
func (h *HttpProxy) AddRoute(route *RouteSpec) error {
	h.routes = append(h.routes, route)
	return nil
}

// Handle Try to dispatch a HTTP request
func (h *HttpProxy) Handle(w http.ResponseWriter, r *http.Request) error {
	rs := h.findRouteSpec(r.URL.Path)
	if rs == nil {
		log.Debugf("Not found: %s", r.URL.Path)
		w.WriteHeader(404)
		return nil
	}
	return h.processRequest(rs, w, r)

}

func (h *HttpProxy) findRouteSpec(path string) *RouteSpec {
	for _, rs := range h.routes {
		if strings.HasPrefix(path, rs.Path) {
			return rs
		}
	}
	return nil
}

func (h *HttpProxy) processRequest(rs *RouteSpec, w http.ResponseWriter, r *http.Request) error {

	// Try to connect with remote target
	client := http.Client{Timeout: 30 * time.Second}
	part := r.URL.Path
	if rs.ReplacePath != nil {
		re := regexp.MustCompile("^" + rs.ReplacePath.Prefix)
		part = re.ReplaceAllString(r.URL.Path, rs.ReplacePath.Value)
		log.Debugf("Replace: %s => %s", r.URL.Path, part)
	}
	targetUrl := rs.Target.String() + part
	log.Debugf("Target URL: %s", targetUrl)
	// for _, proc := range h.processors {
	// 	proc.ProcessorFunc(w, r)
	// }
	req, err := http.NewRequest(r.Method, targetUrl, r.Body)
	if err != nil {
		return err
	}
	req.Header = r.Header
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	for k, v := range resp.Header {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	return nil
}

/*

/api/v1/core => /core
/api/v1/workshop => /workshop





*/

func (h *HttpProxy) AddProcessor(proc Processor) error {
	h.processors = append(h.processors, proc)
	return nil
}

func (h *HttpProxy) wrapProcessor() *WrappedProcessor {
	var first *WrappedProcessor
	var ptr *WrappedProcessor
	var ptr2 *WrappedProcessor
	for _, proc := range h.processors {
		if first == nil {
			first = &WrappedProcessor{Processor: proc}
			ptr = first
		} else {
			ptr2 = &WrappedProcessor{Processor: proc}
			ptr.NestedProcessor = ptr2
			ptr = ptr2
		}
	}
	// Add base proxy processor
	baseProcessor := baseProxyProcessor()
	ptr2 = &WrappedProcessor{
		Processor: baseProcessor,
	}
	return first
}

// proxyProcessor
type proxyProcessor struct {
	Processor
}

func baseProxyProcessor() *proxyProcessor {
	return &proxyProcessor{}
}

func (p *proxyProcessor) Process(nested Processor, w http.ResponseWriter, r *http.Request) error {
	return nil
}

// CORS
type CORSProcessor struct {
}

func CORS() *CORSProcessor {
	return &CORSProcessor{}
}

func (c *CORSProcessor) Process(nested *Processor, w http.ResponseWriter, r *http.Request) error {

	if nested != nil {
		(*nested).Process(nil, w, r)
	}
	return nil
}
