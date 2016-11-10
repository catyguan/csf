// Copyright 2016 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/catyguan/csf/interfaces"
	"github.com/catyguan/csf/pkg/cors"
	"github.com/catyguan/csf/pkg/netutil"
	"github.com/catyguan/csf/pkg/transport"
	"github.com/catyguan/csf/pkg/types"
	"github.com/ghodss/yaml"
)

const (
	DefaultName         = "default"
	DefaultMaxSnapshots = 5
	DefaultSnapCount    = 10000
	DefaultMaxWALs      = 5

	// maxElectionMs specifies the maximum value of election timeout.
	// More details are listed in ../Documentation/tuning.md#time-parameters.
	maxElectionMs = 50000
)

var (
	ErrUnsetAdvertiseClientURLsFlag = fmt.Errorf("--advertise-client-urls is required when --listen-client-urls is set explicitly")

	DefaultListenPeerURLs           = "http://localhost:2380"
	DefaultListenClientURLs         = "http://localhost:2379"
	DefaultInitialAdvertisePeerURLs = "http://localhost:2380"
	DefaultAdvertiseClientURLs      = "http://localhost:2379"

	defaultHostname   string = "localhost"
	defaultHostStatus error
)

func init() {
	ip, err := netutil.GetDefaultHost()
	if err != nil {
		defaultHostStatus = err
		return
	}
	// found default host, advertise on it
	DefaultInitialAdvertisePeerURLs = "http://" + ip + ":2380"
	DefaultAdvertiseClientURLs = "http://" + ip + ":2379"
	defaultHostname = ip
}

type PeerConfig struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	PeerURL      types.URLs
	StrPeerURL   string `json:"peer-url"`
	ClientURL    types.URLs
	StrClientURL string `json:"client-url"`
}

type ClientHandlerFactory func(server *CSFNode, mux *http.ServeMux) http.Handler

// Config holds the arguments for configuring an etcd server.
type Config struct {
	// member
	CorsInfo                *cors.CORSInfo
	LPUrls, LCUrls          types.URLs
	Dir                     string `json:"data-dir"`
	WalDir                  string `json:"wal-dir"`
	MaxSnapFiles            uint   `json:"max-snapshots"`
	MaxWalFiles             uint   `json:"max-wals"`
	Name                    string `json:"name"`
	SnapCount               uint64 `json:"snapshot-count"`
	AutoCompactionRetention int    `json:"auto-compaction-retention"`

	// TickMs is the number of milliseconds between heartbeat ticks.
	// TODO: decouple tickMs and heartbeat tick (current heartbeat tick = 1).
	// make ticks a cluster wide configuration.
	TickMs            uint  `json:"heartbeat-interval"`
	ElectionMs        uint  `json:"election-timeout"`
	QuotaBackendBytes int64 `json:"quota-backend-bytes"`

	// clustering
	ClusterName         string        `json:"cluster-name"`
	ClusterToken        string        `json:"cluster-token"`
	StrictReconfigCheck bool          `json:"strict-reconfig-check"`
	ClusterPeers        []*PeerConfig `json:"cluster-peers"`

	// security
	ClientTLSInfo         transport.TLSInfo
	ClientAutoTLS         bool
	PeerTLSInfo           transport.TLSInfo
	PeerAutoTLS           bool
	ClientCertAuthEnabled bool

	// debug
	Debug        bool   `json:"debug"`
	LogPkgLevels string `json:"log-package-levels"`
	EnablePprof  bool

	// UserHandlers is for registering users handlers and only used for
	// embedding etcd into other applications.
	// The map key is the route path for the handler, and
	// you must ensure it can't be conflicted with etcd's.
	UserHandlers map[string]http.Handler `json:"-"`

	shub []interfaces.Service `json:"-"`

	ACL                  interfaces.ACL
	ClientHandlerFactory ClientHandlerFactory
}

// configYAML holds the config suitable for yaml parsing
type configYAML struct {
	Config
	configJSON
}

// configJSON has file options that are translated into Config options
type configJSON struct {
	LPUrlsJSON         string         `json:"listen-peer-urls"`
	LCUrlsJSON         string         `json:"listen-client-urls"`
	CorsJSON           string         `json:"cors"`
	ClientSecurityJSON securityConfig `json:"client-transport-security"`
	PeerSecurityJSON   securityConfig `json:"peer-transport-security"`
}

type securityConfig struct {
	CAFile        string `json:"ca-file"`
	CertFile      string `json:"cert-file"`
	KeyFile       string `json:"key-file"`
	CertAuth      bool   `json:"client-cert-auth"`
	TrustedCAFile string `json:"trusted-ca-file"`
	AutoTLS       bool   `json:"auto-tls"`
}

// NewConfig creates a new Config populated with default values.
func NewConfig() *Config {
	lpurl, _ := url.Parse(DefaultListenPeerURLs)
	lcurl, _ := url.Parse(DefaultListenClientURLs)
	cfg := &Config{
		CorsInfo:            &cors.CORSInfo{},
		MaxSnapFiles:        DefaultMaxSnapshots,
		MaxWalFiles:         DefaultMaxWALs,
		Name:                DefaultName,
		SnapCount:           DefaultSnapCount,
		TickMs:              100,
		ElectionMs:          1000,
		LPUrls:              []url.URL{*lpurl},
		LCUrls:              []url.URL{*lcurl},
		ClusterToken:        "csf-cluster",
		StrictReconfigCheck: true,
	}
	cfg.ClusterName = DefaultName
	peer := &PeerConfig{}
	peer.ID = 1
	peer.Name = cfg.Name
	peer.StrPeerURL = DefaultListenPeerURLs
	peer.StrClientURL = DefaultListenClientURLs
	cfg.ClusterPeers = []*PeerConfig{peer}
	return cfg
}

func ConfigFromFile(path string) (*Config, error) {
	pcfg := NewConfig()
	rcfg, err := BuildConfigFromFile(path, pcfg)
	if err != nil {
		return nil, err
	}
	return rcfg, nil
}

func BuildConfigFromFile(path string, pcfg *Config) (*Config, error) {
	cfg := &configYAML{Config: *pcfg}
	if err := cfg.configFromFile(path); err != nil {
		return nil, err
	}
	return &cfg.Config, nil
}

func (cfg *configYAML) configFromFile(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		return err
	}

	if cfg.LPUrlsJSON != "" {
		u, err := types.NewURLs(strings.Split(cfg.LPUrlsJSON, ","))
		if err != nil {
			plog.Fatalf("unexpected error setting up listen-peer-urls: %v", err)
		}
		cfg.LPUrls = []url.URL(u)
	}

	if cfg.LCUrlsJSON != "" {
		u, err := types.NewURLs(strings.Split(cfg.LCUrlsJSON, ","))
		if err != nil {
			plog.Fatalf("unexpected error setting up listen-client-urls: %v", err)
		}
		cfg.LCUrls = []url.URL(u)
	}

	if cfg.CorsJSON != "" {
		if err := cfg.CorsInfo.Set(cfg.CorsJSON); err != nil {
			plog.Panicf("unexpected error setting up cors: %v", err)
		}
	}

	if cfg.ClusterPeers != nil {
		for _, cpeer := range cfg.ClusterPeers {
			if cpeer.ID == 0 {
				plog.Fatalf("invalid '%s' id", cpeer.Name)
			}
			if cpeer.StrClientURL != "" {
				u, err := types.NewURLs(strings.Split(cpeer.StrClientURL, ","))
				if err != nil {
					plog.Fatalf("unexpected error parse '%s' peer-url: %v", cpeer.Name, err)
				}
				cpeer.ClientURL = u
			}
			if cpeer.StrPeerURL != "" {
				u, err := types.NewURLs(strings.Split(cpeer.StrPeerURL, ","))
				if err != nil {
					plog.Fatalf("unexpected error parse '%s' peer-url: %v", cpeer.Name, err)
				}
				cpeer.PeerURL = u
			}
		}
	}

	copySecurityDetails := func(tls *transport.TLSInfo, ysc *securityConfig) {
		tls.CAFile = ysc.CAFile
		tls.CertFile = ysc.CertFile
		tls.KeyFile = ysc.KeyFile
		tls.ClientCertAuth = ysc.CertAuth
		tls.TrustedCAFile = ysc.TrustedCAFile
	}
	copySecurityDetails(&cfg.ClientTLSInfo, &cfg.ClientSecurityJSON)
	copySecurityDetails(&cfg.PeerTLSInfo, &cfg.PeerSecurityJSON)
	cfg.ClientAutoTLS = cfg.ClientSecurityJSON.AutoTLS
	cfg.PeerAutoTLS = cfg.PeerSecurityJSON.AutoTLS

	return cfg.Validate()
	// return nil
}

func (cfg *Config) Validate() error {
	if err := checkBindURLs(cfg.LPUrls); err != nil {
		return err
	}
	if err := checkBindURLs(cfg.LCUrls); err != nil {
		return err
	}

	if cfg.ClusterPeers == nil || len(cfg.ClusterPeers) == 0 {
		return fmt.Errorf("empty cluster peer")
	}
	for _, cpeer := range cfg.ClusterPeers {
		if cpeer.ID == 0 {
			return fmt.Errorf("invalid '%s' id", cpeer.Name)
		}
		if cpeer.PeerURL == nil || len(cpeer.PeerURL) == 0 {
			return fmt.Errorf("miss node(%s) peer url", cpeer.Name)
		}
	}

	if 5*cfg.TickMs > cfg.ElectionMs {
		return fmt.Errorf("--election-timeout[%vms] should be at least as 5 times as --heartbeat-interval[%vms]", cfg.ElectionMs, cfg.TickMs)
	}
	if cfg.ElectionMs > maxElectionMs {
		return fmt.Errorf("--election-timeout[%vms] is too long, and should be set less than %vms", cfg.ElectionMs, maxElectionMs)
	}

	return nil
}

func (cfg Config) ElectionTicks() int { return int(cfg.ElectionMs / cfg.TickMs) }

// IsDefaultHost returns the default hostname, if used, and the error, if any,
// from getting the machine's default host.
func (cfg Config) IsDefaultHost() (string, error) {
	return "", defaultHostStatus
}

// checkBindURLs returns an error if any URL uses a domain name.
// TODO: return error in 3.2.0
func checkBindURLs(urls []url.URL) error {
	for _, url := range urls {
		if url.Scheme == "unix" || url.Scheme == "unixs" {
			continue
		}
		host, _, err := net.SplitHostPort(url.Host)
		if err != nil {
			return err
		}
		if host == "localhost" {
			// special case for local address
			// TODO: support /etc/hosts ?
			continue
		}
		if net.ParseIP(host) == nil {
			err := fmt.Errorf("expected IP in URL for binding (%s)", url.String())
			plog.Warning(err)
		}
	}
	return nil
}
