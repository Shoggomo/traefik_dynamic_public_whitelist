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
	mockServerv4 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("192.0.2.123")) // Mock response with a sample IP address
	}))
	defer mockServerv4.Close()

	mockServerv6 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("1234:1234:1234:1234:1234:1234:1234:1234")) // Mock response with a sample IP address
	}))
	defer mockServerv6.Close()

	config := traefik_dynamic_public_whitelist.CreateConfig()
	config.PollInterval = "1s"
	config.IPv4Resolver = mockServerv4.URL
	config.IPv6Resolver = mockServerv6.URL
	config.WhitelistIPv6 = true
	config.IPStrategy = dynamic.IPStrategy{
		Depth:       1,
		ExcludedIPs: []string{"123.0.0.1"},
	}

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
				"public_ipwhitelist": {
					IPWhiteList: &dynamic.IPWhiteList{
						SourceRange: []string{"192.0.2.123", "1234:1234:1234:1234::/64"},
						IPStrategy: &dynamic.IPStrategy{
							Depth:       1,
							ExcludedIPs: []string{"123.0.0.1"},
						},
					},
				},
			},
			ServersTransports: make(map[string]*dynamic.ServersTransport),
		},
		TCP: &dynamic.TCPConfiguration{
			Routers:  make(map[string]*dynamic.TCPRouter),
			Services: make(map[string]*dynamic.TCPService),
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
