// Copyright 2015 The etcd Authors
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

package embed

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/catyguan/csf/csfserver"
	"github.com/catyguan/csf/discovery"
	"github.com/catyguan/csf/pkg/capnslog"
	"github.com/catyguan/csf/pkg/fileutil"
	"github.com/catyguan/csf/pkg/osutil"
	"github.com/catyguan/csf/pkg/types"
	"github.com/catyguan/csf/version"
)

type dirType string

var plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "csfmain")

var (
	dirMember = dirType("member")
	dirProxy  = dirType("proxy")
	dirEmpty  = dirType("empty")
)

// cfg := newConfig()
// cfg.Parse(os.Args[1:])
func StartApp(appName string, cfg *AppConfig, parse bool) {

	defaultInitialCluster := cfg.InitialCluster

	var err error
	if parse {
		err = cfg.Parse(os.Args[1:])
		if err != nil {
			plog.Errorf("error verifying flags, %v. See '%s --help'.", err, appName)
			switch err {
			case ErrUnsetAdvertiseClientURLsFlag:
				plog.Errorf("When listening on specific address(es), this %s process must advertise accessible url(s) to each connected client.", appName)
			}
			os.Exit(1)
		}
	}
	setupLogging(cfg)

	var stopped <-chan struct{}
	var errc <-chan error

	plog.Infof("CSF Version: %s\n", version.Version)
	// plog.Infof("Git SHA: %s\n", version.GitSHA)
	plog.Infof("Go Version: %s\n", runtime.Version())
	plog.Infof("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	GoMaxProcs := runtime.GOMAXPROCS(0)
	plog.Infof("setting maximum number of CPUs to %d, total number of available CPUs is %d", GoMaxProcs, runtime.NumCPU())

	// TODO: check whether fields are set instead of whether fields have default value
	defaultHost, defaultHostErr := cfg.IsDefaultHost()
	defaultHostOverride := defaultHost == "" || defaultHostErr == nil
	if (defaultHostOverride || cfg.Name != DefaultName) && cfg.InitialCluster == defaultInitialCluster {
		cfg.InitialCluster = cfg.InitialClusterFromName(cfg.Name)
	}

	if cfg.Dir == "" {
		cfg.Dir = fmt.Sprintf("%v.csf", cfg.Name)
		plog.Warningf("no data-dir provided, using default data-dir ./%s", cfg.Dir)
	}

	which := identifyDataDirOrDie(cfg.Dir)
	if which != dirEmpty {
		plog.Noticef("the server is already initialized as %v before, starting as %s %v...", which, appName, which)
		switch which {
		case dirMember:
			stopped, errc, err = startCSF(&cfg.Config)
		case dirProxy:
			// err = startProxy(cfg)
		default:
			plog.Panicf("unhandled dir type %v", which)
		}
	} else {
		// shouldProxy := cfg.isProxy()
		shouldProxy := false
		if !shouldProxy {
			stopped, errc, err = startCSF(&cfg.Config)
			if derr, ok := err.(*csfserver.DiscoveryError); ok && derr.Err == discovery.ErrFullCluster {
				if cfg.shouldFallbackToProxy() {
					plog.Noticef("discovery cluster full, falling back to %s", fallbackFlagProxy)
					shouldProxy = true
				}
			}
		}
		if shouldProxy {
			// err = startProxy(cfg)
		}
	}

	if err != nil {
		if derr, ok := err.(*csfserver.DiscoveryError); ok {
			switch derr.Err {
			case discovery.ErrDuplicateID:
				plog.Errorf("member %q has previously registered with discovery service token (%s).", cfg.Name, cfg.Durl)
				plog.Errorf("But %s could not find valid cluster configuration in the given data dir (%s).", appName, cfg.Dir)
				plog.Infof("Please check the given data dir path if the previous bootstrap succeeded")
				plog.Infof("or use a new discovery token if the previous bootstrap failed.")
			case discovery.ErrDuplicateName:
				plog.Errorf("member with duplicated name has registered with discovery service token(%s).", cfg.Durl)
				plog.Errorf("please check (cURL) the discovery token for more information.")
				plog.Errorf("please do not reuse the discovery token and generate a new one to bootstrap the cluster.")
			default:
				plog.Errorf("%v", err)
				plog.Infof("discovery token %s was used, but failed to bootstrap the cluster.", cfg.Durl)
				plog.Infof("please generate a new discovery token and try to bootstrap again.")
			}
			os.Exit(1)
		}

		if strings.Contains(err.Error(), "include") && strings.Contains(err.Error(), "--initial-cluster") {
			plog.Infof("%v", err)
			if cfg.InitialCluster == cfg.InitialClusterFromName(cfg.Name) {
				plog.Infof("forgot to set --initial-cluster flag?")
			}
			if types.URLs(cfg.APUrls).String() == DefaultInitialAdvertisePeerURLs {
				plog.Infof("forgot to set --initial-advertise-peer-urls flag?")
			}
			if cfg.InitialCluster == cfg.InitialClusterFromName(cfg.Name) && len(cfg.Durl) == 0 {
				plog.Infof("if you want to use discovery service, please set --discovery flag.")
			}
			os.Exit(1)
		}
		plog.Fatalf("%v", err)
	}

	osutil.HandleInterrupts()

	select {
	case lerr := <-errc:
		// fatal out on listener errors
		plog.Fatal(lerr)
	case <-stopped:
	}

	osutil.Exit(0)
}

// startEtcd runs StartEtcd in addition to hooks needed for standalone etcd.
func startCSF(cfg *Config) (<-chan struct{}, <-chan error, error) {
	defaultHost, dhErr := cfg.IsDefaultHost()
	if defaultHost != "" {
		if dhErr == nil {
			plog.Infof("advertising using detected default host %q", defaultHost)
		} else {
			plog.Noticef("failed to detect default host, advertise falling back to %q (%v)", defaultHost, dhErr)
		}
	}

	e, err := StartCluster(cfg)
	if err != nil {
		return nil, nil, err
	}
	osutil.RegisterInterruptHandler(e.Server.Stop)
	<-e.Server.ReadyNotify() // wait for e.Server to join the cluster
	return e.Server.StopNotify(), e.Err(), nil
}

// startProxy launches an HTTP proxy for client communication which proxies to other etcd nodes.
// func startProxy(cfg *config) error {
// 	plog.Notice("proxy: this proxy supports v2 API only!")

// 	pt, err := transport.NewTimeoutTransport(cfg.PeerTLSInfo, time.Duration(cfg.ProxyDialTimeoutMs)*time.Millisecond, time.Duration(cfg.ProxyReadTimeoutMs)*time.Millisecond, time.Duration(cfg.ProxyWriteTimeoutMs)*time.Millisecond)
// 	if err != nil {
// 		return err
// 	}
// 	pt.MaxIdleConnsPerHost = httpproxy.DefaultMaxIdleConnsPerHost

// 	tr, err := transport.NewTimeoutTransport(cfg.PeerTLSInfo, time.Duration(cfg.ProxyDialTimeoutMs)*time.Millisecond, time.Duration(cfg.ProxyReadTimeoutMs)*time.Millisecond, time.Duration(cfg.ProxyWriteTimeoutMs)*time.Millisecond)
// 	if err != nil {
// 		return err
// 	}

// 	cfg.Dir = path.Join(cfg.Dir, "proxy")
// 	err = os.MkdirAll(cfg.Dir, fileutil.PrivateDirMode)
// 	if err != nil {
// 		return err
// 	}

// 	var peerURLs []string
// 	clusterfile := path.Join(cfg.Dir, "cluster")

// 	b, err := ioutil.ReadFile(clusterfile)
// 	switch {
// 	case err == nil:
// 		if cfg.Durl != "" {
// 			plog.Warningf("discovery token ignored since the proxy has already been initialized. Valid cluster file found at %q", clusterfile)
// 		}
// 		if cfg.DNSCluster != "" {
// 			plog.Warningf("DNS SRV discovery ignored since the proxy has already been initialized. Valid cluster file found at %q", clusterfile)
// 		}
// 		urls := struct{ PeerURLs []string }{}
// 		err = json.Unmarshal(b, &urls)
// 		if err != nil {
// 			return err
// 		}
// 		peerURLs = urls.PeerURLs
// 		plog.Infof("proxy: using peer urls %v from cluster file %q", peerURLs, clusterfile)
// 	case os.IsNotExist(err):
// 		var urlsmap types.URLsMap
// 		urlsmap, _, err = cfg.PeerURLsMapAndToken("proxy")
// 		if err != nil {
// 			return fmt.Errorf("error setting up initial cluster: %v", err)
// 		}

// 		if cfg.Durl != "" {
// 			var s string
// 			s, err = discovery.GetCluster(cfg.Durl, cfg.Dproxy)
// 			if err != nil {
// 				return err
// 			}
// 			if urlsmap, err = types.NewURLsMap(s); err != nil {
// 				return err
// 			}
// 		}
// 		peerURLs = urlsmap.URLs()
// 		plog.Infof("proxy: using peer urls %v ", peerURLs)
// 	default:
// 		return err
// 	}

// 	clientURLs := []string{}
// 	uf := func() []string {
// 		gcls, gerr := csfserver.GetClusterFromRemotePeers(peerURLs, tr)

// 		if gerr != nil {
// 			plog.Warningf("proxy: %v", gerr)
// 			return []string{}
// 		}

// 		clientURLs = gcls.ClientURLs()

// 		urls := struct{ PeerURLs []string }{gcls.PeerURLs()}
// 		b, jerr := json.Marshal(urls)
// 		if jerr != nil {
// 			plog.Warningf("proxy: error on marshal peer urls %s", jerr)
// 			return clientURLs
// 		}

// 		err = pkgioutil.WriteAndSyncFile(clusterfile+".bak", b, 0600)
// 		if err != nil {
// 			plog.Warningf("proxy: error on writing urls %s", err)
// 			return clientURLs
// 		}
// 		err = os.Rename(clusterfile+".bak", clusterfile)
// 		if err != nil {
// 			plog.Warningf("proxy: error on updating clusterfile %s", err)
// 			return clientURLs
// 		}
// 		if !reflect.DeepEqual(gcls.PeerURLs(), peerURLs) {
// 			plog.Noticef("proxy: updated peer urls in cluster file from %v to %v", peerURLs, gcls.PeerURLs())
// 		}
// 		peerURLs = gcls.PeerURLs()

// 		return clientURLs
// 	}
// 	ph := httpproxy.NewHandler(pt, uf, time.Duration(cfg.ProxyFailureWaitMs)*time.Millisecond, time.Duration(cfg.ProxyRefreshIntervalMs)*time.Millisecond)
// 	ph = &cors.CORSHandler{
// 		Handler: ph,
// 		Info:    cfg.CorsInfo,
// 	}

// 	if cfg.isReadonlyProxy() {
// 		ph = httpproxy.NewReadonlyHandler(ph)
// 	}
// 	// Start a proxy server goroutine for each listen address
// 	for _, u := range cfg.LCUrls {
// 		var (
// 			l      net.Listener
// 			tlscfg *tls.Config
// 		)
// 		if !cfg.ClientTLSInfo.Empty() {
// 			tlscfg, err = cfg.ClientTLSInfo.ServerConfig()
// 			if err != nil {
// 				return err
// 			}
// 		}

// 		l, err := transport.NewListener(u.Host, u.Scheme, tlscfg)
// 		if err != nil {
// 			return err
// 		}

// 		host := u.String()
// 		go func() {
// 			plog.Info("proxy: listening for client requests on ", host)
// 			mux := http.NewServeMux()
// 			mux.Handle("/metrics", prometheus.Handler())
// 			mux.Handle("/", ph)
// 			plog.Fatal(http.Serve(l, mux))
// 		}()
// 	}
// 	return nil
// }

// identifyDataDirOrDie returns the type of the data dir.
// Dies if the datadir is invalid.
func identifyDataDirOrDie(dir string) dirType {
	names, err := fileutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return dirEmpty
		}
		plog.Fatalf("error listing data dir: %s", dir)
	}

	var m, p bool
	for _, name := range names {
		switch dirType(name) {
		case dirMember:
			m = true
		case dirProxy:
			p = true
		default:
			plog.Warningf("found invalid file/dir %s under data dir %s (Ignore this if you are upgrading CSF)", name, dir)
		}
	}

	if m && p {
		plog.Fatal("invalid datadir. Both member and proxy directories exist.")
	}
	if m {
		return dirMember
	}
	if p {
		return dirProxy
	}
	return dirEmpty
}

func setupLogging(cfg *AppConfig) {
	capnslog.SetGlobalLogLevel(capnslog.INFO)
	if cfg.Debug {
		capnslog.SetGlobalLogLevel(capnslog.DEBUG)
	}
	if cfg.LogPkgLevels != "" {
		repoLog := capnslog.MustRepoLogger("github.com/catyguan/csf")
		settings, err := repoLog.ParseLogLevelConfig(cfg.LogPkgLevels)
		if err != nil {
			plog.Warningf("couldn't parse log level string: %s, continuing with default levels", err.Error())
			return
		}
		repoLog.SetLogLevel(settings)
	}
}
