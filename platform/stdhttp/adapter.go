package stdhttp

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gonest/platform"
)

// trieNode is a node in the URL path trie used for route matching.
type trieNode struct {
	children   map[string]*trieNode
	paramChild *trieNode
	paramName  string
	wildChild  *trieNode
	handlers   map[string]platform.HandlerFunc // method -> handler
}

func newTrieNode() *trieNode {
	return &trieNode{
		children: make(map[string]*trieNode),
		handlers: make(map[string]platform.HandlerFunc),
	}
}

// Adapter is the default HTTP adapter built on net/http with a trie-based router.
type Adapter struct {
	root                    *trieNode
	middleware              []func(http.Handler) http.Handler
	notFoundHandler         http.HandlerFunc
	methodNotAllowedHandler http.HandlerFunc
	server                  *http.Server
	mu                      sync.RWMutex
}

// New creates a new stdlib HTTP adapter.
func New() *Adapter {
	return &Adapter{
		root: newTrieNode(),
		notFoundHandler: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"statusCode":404,"message":"Not Found"}`, http.StatusNotFound)
		},
		methodNotAllowedHandler: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"statusCode":405,"message":"Method Not Allowed"}`, http.StatusMethodNotAllowed)
		},
	}
}

// Handle registers a handler for a method and path pattern.
// Path parameters use :param syntax. Wildcards use *.
func (a *Adapter) Handle(method, path string, handler platform.HandlerFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()

	segments := splitPath(path)
	node := a.root

	for _, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			if node.paramChild == nil {
				node.paramChild = newTrieNode()
				node.paramChild.paramName = seg[1:]
			}
			node = node.paramChild
		} else if seg == "*" {
			if node.wildChild == nil {
				node.wildChild = newTrieNode()
			}
			node = node.wildChild
			break
		} else {
			child, ok := node.children[seg]
			if !ok {
				child = newTrieNode()
				node.children[seg] = child
			}
			node = child
		}
	}

	if method == "*" {
		for _, m := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"} {
			node.handlers[m] = handler
		}
	} else {
		node.handlers[strings.ToUpper(method)] = handler
	}
}

// Use registers global middleware.
func (a *Adapter) Use(middleware func(http.Handler) http.Handler) {
	a.middleware = append(a.middleware, middleware)
}

// Listen starts the server.
func (a *Adapter) Listen(addr string) error {
	a.server = &http.Server{
		Addr:    addr,
		Handler: a.Handler(),
	}
	return a.server.ListenAndServe()
}

// Handler returns the composed http.Handler.
func (a *Adapter) Handler() http.Handler {
	var h http.Handler = a
	for i := len(a.middleware) - 1; i >= 0; i-- {
		h = a.middleware[i](h)
	}
	return h
}

// ServeHTTP implements http.Handler.
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	handler, params, methodsAllowed := a.lookup(r.Method, r.URL.Path)
	if handler != nil {
		handler(w, r, params)
		return
	}

	if len(methodsAllowed) > 0 {
		w.Header().Set("Allow", strings.Join(methodsAllowed, ", "))
		a.methodNotAllowedHandler(w, r)
		return
	}

	a.notFoundHandler(w, r)
}

// SetNotFoundHandler sets the 404 handler.
func (a *Adapter) SetNotFoundHandler(handler http.HandlerFunc) {
	a.notFoundHandler = handler
}

// SetMethodNotAllowedHandler sets the 405 handler.
func (a *Adapter) SetMethodNotAllowedHandler(handler http.HandlerFunc) {
	a.methodNotAllowedHandler = handler
}

// Shutdown gracefully shuts down the server.
func (a *Adapter) Shutdown() error {
	if a.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return a.server.Shutdown(ctx)
}

// lookup finds the handler and extracts path params for a request.
func (a *Adapter) lookup(method, path string) (platform.HandlerFunc, map[string]string, []string) {
	segments := splitPath(path)
	params := make(map[string]string)
	node := a.root

	for _, seg := range segments {
		// Exact match first
		if child, ok := node.children[seg]; ok {
			node = child
			continue
		}
		// Parameter match
		if node.paramChild != nil {
			params[node.paramChild.paramName] = seg
			node = node.paramChild
			continue
		}
		// Wildcard match
		if node.wildChild != nil {
			node = node.wildChild
			break
		}
		// No match
		return nil, nil, nil
	}

	handler, ok := node.handlers[method]
	if ok {
		return handler, params, nil
	}

	// Collect allowed methods for 405
	var allowed []string
	for m := range node.handlers {
		allowed = append(allowed, m)
	}
	if len(allowed) > 0 {
		return nil, nil, allowed
	}

	return nil, nil, nil
}

func splitPath(path string) []string {
	if path == "" || path == "/" {
		return nil
	}
	// Remove leading slash
	if path[0] == '/' {
		path = path[1:]
	}
	// Remove trailing slash
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
