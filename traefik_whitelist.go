package traefik_whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/tls"
)

// Config the plugin configuration.
type Config struct {
	PollInterval string `json:"pollInterval,omitempty"`
	Lists        map[string]string
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		PollInterval: "120s", // 120 * time.Second
		Lists:        make(map[string]string),
	}
}

// Provider a simple provider plugin.
type Provider struct {
	name         string
	pollInterval time.Duration
	lists        map[string]string
	lists_data   map[string]string

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
		lists:        config.Lists,
		lists_data:   make(map[string]string),
	}, nil
}

// Init the provider.
func (p *Provider) Init() error {
	if p.pollInterval <= 0 {
		return fmt.Errorf("poll interval must be greater than 0")
	}

	if len(p.lists) == 0 {
		return fmt.Errorf("at least one portbrella list must be configured")
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

	t := time.Now()
	configuration := generateConfiguration(t, p.lists, p.lists_data)
	cfgChan <- &dynamic.JSONPayload{Configuration: configuration}

	for {
		select {
		case t := <-ticker.C:
			configuration := generateConfiguration(t, p.lists, p.lists_data)
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

func delete_empty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func generateConfiguration(date time.Time, lists map[string]string, lists_data map[string]string) *dynamic.Configuration {
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

	// fetch list with https GET request
	for key, element := range lists {
		// fmt.Println("Key:", key, "=>", "Element:", element)

		resp, err := http.Get("https://wl.portbrella.com/" + element)

		if err != nil {
			log.Fatalln(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}

		ips := lists_data[element]

		if resp.StatusCode == 200 {
			ips = string(body)
			lists_data[element] = ips
		}

		splitted := delete_empty(strings.Split(ips, "\n"))

		configuration.HTTP.Middlewares[key] = &dynamic.Middleware{
			IPWhiteList: &dynamic.IPWhiteList{
				SourceRange: splitted,
			},
		}

		configuration.TCP.Middlewares[key] = &dynamic.TCPMiddleware{
			IPWhiteList: &dynamic.TCPIPWhiteList{
				SourceRange: splitted,
			},
		}
	}

	return configuration
}

func boolPtr(v bool) *bool {
	return &v
}
