package gonest

import (
	"net/http"
	"testing"
)

type webdavController struct{}

func newWebdavController() *webdavController { return &webdavController{} }

func (c *webdavController) Register(r Router) {
	r.Prefix("/dav")
	r.Search("/", c.handler)
	r.Propfind("/", c.handler)
	r.Proppatch("/", c.handler)
	r.Mkcol("/col", c.handler)
	r.Copy("/resource", c.handler)
	r.Move("/resource", c.handler)
	r.Lock("/resource", c.handler)
	r.Unlock("/resource", c.handler)
}

func (c *webdavController) handler(ctx Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"method": ctx.Method()})
}

func TestWebDAVMethods_RouteRegistration(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newWebdavController},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	routes := app.GetRoutes()
	expectedMethods := map[string]bool{
		"SEARCH":    false,
		"PROPFIND":  false,
		"PROPPATCH": false,
		"MKCOL":     false,
		"COPY":      false,
		"MOVE":      false,
		"LOCK":      false,
		"UNLOCK":    false,
	}

	for _, route := range routes {
		if _, ok := expectedMethods[route.Method]; ok {
			expectedMethods[route.Method] = true
		}
	}

	for method, found := range expectedMethods {
		if !found {
			t.Errorf("WebDAV method %s not registered", method)
		}
	}
}
