package traefik_dynamic_public_whitelist_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	//"time"

	"github.com/Shoggomo/traefik_dynamic_public_whitelist"
	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/tls"
)

func TestNew(t *testing.T) {
	// Create a test server to mock the HTTP endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		println("Request!")
		w.Write([]byte("192.0.2.123")) // Mock response with a sample IP address
	}))
	defer mockServer.Close()

	config := traefik_dynamic_public_whitelist.CreateConfig()
	config.PollInterval = "1s"
	config.IPResolver = mockServer.URL

	provider, err := traefik_dynamic_public_whitelist.New(context.Background(), config, "test")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = provider.Stop()
		if err != nil {
			t.Fatal(err)
		}
	})

	err = provider.Init()
	if err != nil {
		t.Fatal(err)
	}

	cfgChan := make(chan json.Marshaler)

	err = provider.Provide(cfgChan)
	if err != nil {
		t.Fatal(err)
	}

	data := <-cfgChan

	expected := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:  make(map[string]*dynamic.Router),
			Services: make(map[string]*dynamic.Service),
			Middlewares: map[string]*dynamic.Middleware{
				"dpw_middleware": {
					IPWhiteList: &dynamic.IPWhiteList{
						SourceRange: []string{"192.0.2.123"},
					},
				},
			},
			ServersTransports: make(map[string]*dynamic.ServersTransport),
		},
		TCP: &dynamic.TCPConfiguration{
			Routers:  make(map[string]*dynamic.TCPRouter),
			Services: make(map[string]*dynamic.TCPService),
			Middlewares: map[string]*dynamic.TCPMiddleware{
				"dpw_middleware": {
					IPWhiteList: &dynamic.TCPIPWhiteList{
						SourceRange: []string{"192.0.2.123"},
					},
				},
			},
		},
		TLS: &dynamic.TLSConfiguration{
			Stores:  make(map[string]tls.Store),
			Options: make(map[string]tls.Options),
		},
		UDP: &dynamic.UDPConfiguration{
			Routers:  make(map[string]*dynamic.UDPRouter),
			Services: make(map[string]*dynamic.UDPService),
		},
	}

	expectedJSON, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expectedJSON, dataJSON) {
		t.Fatalf("got %s, want: %s", string(dataJSON), string(expectedJSON))
	}
}

func boolPtr(v bool) *bool {
	return &v
}
