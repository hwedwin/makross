package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"net/url"

	"github.com/insionng/makross"
	"github.com/stretchr/testify/assert"
)

type (
	closeNotifyRecorder struct {
		*httptest.ResponseRecorder
		closed chan bool
	}
)

func newCloseNotifyRecorder() *closeNotifyRecorder {
	return &closeNotifyRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
}

func (c *closeNotifyRecorder) close() {
	c.closed <- true
}

func (c *closeNotifyRecorder) CloseNotify() <-chan bool {
	return c.closed
}

func TestProxy(t *testing.T) {
	// Setup
	t1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "target 1")
	}))
	defer t1.Close()
	url1, _ := url.Parse(t1.URL)
	t2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "target 2")
	}))
	defer t2.Close()
	url2, _ := url.Parse(t2.URL)

	targets := []*ProxyTarget{
		&ProxyTarget{
			URL: url1,
		},
		&ProxyTarget{
			URL: url2,
		},
	}
	config := ProxyConfig{
		Balancer: &RandomBalancer{
			Targets: targets,
		},
	}

	// Random
	e := makross.New()
	e.Use(Proxy(config))
	req := httptest.NewRequest(makross.GET, "/", nil)
	rec := newCloseNotifyRecorder()
	e.ServeHTTP(rec, req)
	body := rec.Body.String()
	expected := map[string]bool{
		"target 1": true,
		"target 2": true,
	}
	assert.Condition(t, func() bool {
		return expected[body]
	})

	// Round-robin
	config.Balancer = &RoundRobinBalancer{
		Targets: targets,
	}
	e = makross.New()
	e.Use(Proxy(config))
	rec = newCloseNotifyRecorder()
	e.ServeHTTP(rec, req)
	body = rec.Body.String()
	assert.Equal(t, "target 1", body)
	rec = newCloseNotifyRecorder()
	e.ServeHTTP(rec, req)
	body = rec.Body.String()
	assert.Equal(t, "target 2", body)
}