package traefik_dynamic_public_whitelist

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/tls"
)

// Config the plugin configuration.
type Config struct {
	PollInterval  string `json:"pollInterval,omitempty"`
	IPv4Resolver  string `json:"ipv4Resolver,omitempty"`
	IPv6Resolver  string `json:"ipv6Resolver,omitempty"`
	WhitelistIPv6 bool   `json:"whitelistIPv6,omitempty"`
	IPStrategy    dynamic.IPStrategy
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		PollInterval:  "300s",
		IPv4Resolver:  "https://api.ipify.org?format=text",
		IPv6Resolver:  "https://api64.ipify.org?format=text",
		WhitelistIPv6: true,
		IPStrategy: dynamic.IPStrategy{
			Depth:       0,
			ExcludedIPs: nil,
		},
	}
}

// Provider a simple provider plugin.
type Provider struct {
	name          string
	pollInterval  time.Duration
	ipv4Resolver  string
	ipv6Resolver  string
	whitelistIPv6 bool
	ipStrategy    dynamic.IPStrategy

	cancel func()
}

// New creates a new Provider plugin.
func New(ctx context.Context, config *Config, name string) (*Provider, error) {
	pi, err := time.ParseDuration(config.PollInterval)
	if err != nil {
		return nil, err
	}

	return &Provider{
		name:          name,
		pollInterval:  pi,
		ipv4Resolver:  config.IPv4Resolver,
		ipv6Resolver:  config.IPv6Resolver,
		whitelistIPv6: config.WhitelistIPv6,
		ipStrategy:    config.IPStrategy,
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

	configuration := generateConfiguration(p.ipv4Resolver, p.ipv6Resolver, p.whitelistIPv6, p.ipStrategy)
	cfgChan <- &dynamic.JSONPayload{Configuration: configuration}

	for {
		select {
		case <-ticker.C:
			configuration := generateConfiguration(p.ipv4Resolver, p.ipv6Resolver, p.whitelistIPv6, p.ipStrategy)
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

type IPAddresses struct {
	v4     string
	v6CIDR string
}

func ipv6ToCIDR(ipv6 string) (string, error) {
	const MaskSize = 64 // most providers supply 64 bit ipv6 addresses

	ip := net.ParseIP(ipv6)
	ip = ip.To16()

	if ip == nil {
		return "", fmt.Errorf("input is not an IPv6 address: %s", ipv6)
	}

	cidr := ip.Mask(net.CIDRMask(MaskSize, 128)).String() + "/" + strconv.Itoa(MaskSize)

	return cidr, nil
}

func getPublicIp(ipv4Resolver string, ipv6Resolver string, whitelistIpv6 bool) (IPAddresses, error) {
	ipv4, err := getBody(ipv4Resolver)

	if err != nil {
		return IPAddresses{}, err
	}

	if net.ParseIP(ipv4) == nil {
		return IPAddresses{}, fmt.Errorf("could not parse resolver response")
	}

	if !whitelistIpv6 {
		return IPAddresses{
			v4:     ipv4,
			v6CIDR: "",
		}, nil
	}

	ipv6, err := getBody(ipv6Resolver)

	if err != nil {
		return IPAddresses{}, err
	}

	if net.ParseIP(ipv6) == nil {
		return IPAddresses{}, fmt.Errorf("could not parse resolver response")
	}

	ipv6CIDR, err := ipv6ToCIDR(ipv6)

	if err != nil {
		return IPAddresses{}, err
	}

	return IPAddresses{
		v4:     ipv4,
		v6CIDR: ipv6CIDR,
	}, nil
}

func getBody(address string) (string, error) {
	resp, err := http.Get(address)
	if err != nil {
		log.Print(err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err)
		return "", err
	}

	return string(body), nil
}

func generateConfiguration(ipv4Resolver string, ipv6Resolver string, whitelistIPv6 bool, ipStrategy dynamic.IPStrategy) *dynamic.Configuration {
	configuration := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:           make(map[string]*dynamic.Router),
			Middlewares:       make(map[string]*dynamic.Middleware),
			Services:          make(map[string]*dynamic.Service),
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

	ipAddresses, err := getPublicIp(ipv4Resolver, ipv6Resolver, whitelistIPv6)

	sourceRange := make([]string, 0, 2)
	sourceRange = append(sourceRange, ipAddresses.v4)

	if whitelistIPv6 {
		sourceRange = append(sourceRange, ipAddresses.v6CIDR)
	}

	if err != nil {
		log.Fatalln(err)
	}

	configuration.HTTP.Middlewares["public_ipwhitelist"] = &dynamic.Middleware{
		IPWhiteList: &dynamic.IPWhiteList{
			SourceRange: sourceRange,
			IPStrategy: &dynamic.IPStrategy{
				Depth:       ipStrategy.Depth,
				ExcludedIPs: ipStrategy.ExcludedIPs,
			},
		},
	}

	return configuration
}
