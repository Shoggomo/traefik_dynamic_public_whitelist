package traefik_dynamic_public_whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/tls"
)

// Config the plugin configuration.
type Config struct {
	PollInterval string `json:"pollInterval,omitempty"`
	IPResolver   string `json:"IPResolver,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		PollInterval: "5s", // 5 * time.Second
		IPResolver:   "https://api.ipify.org?format=text",
	}
}

// Provider a simple provider plugin.
type Provider struct {
	name         string
	pollInterval time.Duration
	ipResolver   string

	cancel func()
}

// New creates a new Provider plugin.
func New(ctx context.Context, config *Config, name string) (*Provider, error) {
	pi, err := time.ParseDuration(config.PollInterval)
	if err != nil {
		return nil, err
	}

	return &Provider{
		name:         name,
		pollInterval: pi,
	}, nil
}

// Init the provider.
func (p *Provider) Init() error {
	if p.pollInterval <= 0 {
		return fmt.Errorf("poll interval must be greater than 0")
	}

	return nil
}

// Provide creates and send dynamic configuration.
func (p *Provider) Provide(cfgChan chan<- json.Marshaler) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Print(err)
			}
		}()

		p.loadConfiguration(ctx, cfgChan)
	}()

	return nil
}

func (p *Provider) loadConfiguration(ctx context.Context, cfgChan chan<- json.Marshaler) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	configuration := generateConfiguration(p.ipResolver)
	cfgChan <- &dynamic.JSONPayload{Configuration: configuration}

	for {
		select {
		case <-ticker.C:
			configuration := generateConfiguration(p.ipResolver)
			cfgChan <- &dynamic.JSONPayload{Configuration: configuration}

		case <-ctx.Done():
			return
		}
	}
}

// Stop to stop the provider and the related go routines.
func (p *Provider) Stop() error {
	p.cancel()
	return nil
}

func getPublicIp(ipResolver string) (string, error) {
	resp, err := http.Get(ipResolver)
	if err != nil {
		log.Print(err)
		return "", err
	}
	defer resp.Body.Close()

	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return "", err
	}

	return string(ip), nil
}

func generateConfiguration(ipResolver string) *dynamic.Configuration {
	configuration := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:           make(map[string]*dynamic.Router),
			Middlewares:       make(map[string]*dynamic.Middleware),
			Services:          make(map[string]*dynamic.Service),
			ServersTransports: make(map[string]*dynamic.ServersTransport),
		},
		TCP: &dynamic.TCPConfiguration{
			Routers:     make(map[string]*dynamic.TCPRouter),
			Services:    make(map[string]*dynamic.TCPService),
			Middlewares: make(map[string]*dynamic.TCPMiddleware),
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

	ip, err := getPublicIp(ipResolver)

	if err != nil {
		log.Fatalln(err)
	}

	configuration.HTTP.Middlewares["dpw_middleware"] = &dynamic.Middleware{
		IPWhiteList: &dynamic.IPWhiteList{
			SourceRange: []string{ip},
		},
	}

	configuration.TCP.Middlewares["dpw_middleware"] = &dynamic.TCPMiddleware{
		IPWhiteList: &dynamic.TCPIPWhiteList{
			SourceRange: []string{ip},
		},
	}

	return configuration
}
