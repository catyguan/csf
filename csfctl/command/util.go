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

package command

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/bgentry/speakeasy"
	"github.com/catyguan/csf/client"
	"github.com/catyguan/csf/pkg/transport"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
)

var (
	ErrNoAvailSrc = errors.New("no available argument and stdin")

	// the maximum amount of time a dial will wait for a connection to setup.
	// 30s is long enough for most of the network conditions.
	defaultDialTimeout = 30 * time.Second
)

func argOrStdin(args []string, stdin io.Reader, i int) (string, error) {
	if i < len(args) {
		return args[i], nil
	}
	bytes, err := ioutil.ReadAll(stdin)
	if string(bytes) == "" || err != nil {
		return "", ErrNoAvailSrc
	}
	return string(bytes), nil
}

func getPeersFlagValue(c *cli.Context) []string {
	peerstr := c.GlobalString("endpoints")

	if peerstr == "" {
		peerstr = os.Getenv("ETCDCTL_ENDPOINTS")
	}

	if peerstr == "" {
		peerstr = c.GlobalString("endpoint")
	}

	if peerstr == "" {
		peerstr = os.Getenv("ETCDCTL_ENDPOINT")
	}

	if peerstr == "" {
		peerstr = c.GlobalString("peers")
	}

	if peerstr == "" {
		peerstr = os.Getenv("ETCDCTL_PEERS")
	}

	// If we still don't have peers, use a default
	if peerstr == "" {
		peerstr = "http://127.0.0.1:2379,http://127.0.0.1:4001"
	}

	return strings.Split(peerstr, ",")
}

func getEndpoints(c *cli.Context) ([]string, error) {
	// If domain discovery returns no endpoints, check peer flag
	eps := getPeersFlagValue(c)

	for i, ep := range eps {
		u, err := url.Parse(ep)
		if err != nil {
			return nil, err
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		eps[i] = u.String()
	}

	return eps, nil
}

func getTransport(c *cli.Context) (*http.Transport, error) {
	cafile := c.GlobalString("ca-file")
	certfile := c.GlobalString("cert-file")
	keyfile := c.GlobalString("key-file")

	// Use an environment variable if nothing was supplied on the
	// command line
	if cafile == "" {
		cafile = os.Getenv("CSFCTL_CA_FILE")
	}
	if certfile == "" {
		certfile = os.Getenv("CSFCTL_CERT_FILE")
	}
	if keyfile == "" {
		keyfile = os.Getenv("CSFCTL_KEY_FILE")
	}

	tls := transport.TLSInfo{
		CAFile:     cafile,
		CertFile:   certfile,
		KeyFile:    keyfile,
		ServerName: "",
	}

	dialTimeout := defaultDialTimeout
	totalTimeout := c.GlobalDuration("total-timeout")
	if totalTimeout != 0 && totalTimeout < dialTimeout {
		dialTimeout = totalTimeout
	}
	return transport.NewTransport(tls, dialTimeout)
}

func getUsernamePasswordFromFlag(usernameFlag string) (username string, password string, err error) {
	return getUsernamePassword("Password: ", usernameFlag)
}

func getUsernamePassword(prompt, usernameFlag string) (username string, password string, err error) {
	colon := strings.Index(usernameFlag, ":")
	if colon == -1 {
		username = usernameFlag
		// Prompt for the password.
		password, err = speakeasy.Ask(prompt)
		if err != nil {
			return "", "", err
		}
	} else {
		username = usernameFlag[:colon]
		password = usernameFlag[colon+1:]
	}
	return username, password, nil
}

func mustNewMembersAPI(c *cli.Context) client.MembersAPI {
	return client.NewMembersAPI(MustNewClient(c))
}

func MustNewClient(c *cli.Context) client.Client {
	hc, err := newClient(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	debug := c.GlobalBool("debug")
	if debug {
		client.EnablecURLDebug()
	}

	if !c.GlobalBool("no-sync") {
		if debug {
			fmt.Fprintf(os.Stderr, "start to sync cluster using endpoints(%s)\n", strings.Join(hc.Endpoints(), ","))
		}
		ctx, cancel := ContextWithTotalTimeout(c)
		err := hc.Sync(ctx)
		cancel()
		if err != nil {
			if err == client.ErrNoEndpoints {
				fmt.Fprintf(os.Stderr, "csf cluster has no published client endpoints.\n")
				fmt.Fprintf(os.Stderr, "Try '--no-sync' if you want to access non-published client endpoints(%s).\n", strings.Join(hc.Endpoints(), ","))
				handleError(ExitServerError, err)
			}
			if isConnectionError(err) {
				handleError(ExitBadConnection, err)
			}
		}
		if debug {
			fmt.Fprintf(os.Stderr, "got endpoints(%s) after sync\n", strings.Join(hc.Endpoints(), ","))
		}
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Cluster-Endpoints: %s\n", strings.Join(hc.Endpoints(), ", "))
	}

	return hc
}

func isConnectionError(err error) bool {
	switch t := err.(type) {
	case *client.ClusterError:
		for _, cerr := range t.Errors {
			if !isConnectionError(cerr) {
				return false
			}
		}
		return true
	case *net.OpError:
		if t.Op == "dial" || t.Op == "read" {
			return true
		}
		return isConnectionError(t.Err)
	case net.Error:
		if t.Timeout() {
			return true
		}
	case syscall.Errno:
		if t == syscall.ECONNREFUSED {
			return true
		}
	}
	return false
}

func mustNewClientNoSync(c *cli.Context) client.Client {
	hc, err := newClient(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if c.GlobalBool("debug") {
		fmt.Fprintf(os.Stderr, "Cluster-Endpoints: %s\n", strings.Join(hc.Endpoints(), ", "))
		client.EnablecURLDebug()
	}

	return hc
}

func newClient(c *cli.Context) (client.Client, error) {
	eps, err := getEndpoints(c)
	if err != nil {
		return nil, err
	}

	tr, err := getTransport(c)
	if err != nil {
		return nil, err
	}

	cfg := client.Config{
		Transport:               tr,
		Endpoints:               eps,
		HeaderTimeoutPerRequest: c.GlobalDuration("timeout"),
	}

	uFlag := c.GlobalString("username")

	if uFlag == "" {
		uFlag = os.Getenv("CSFCTL_USERNAME")
	}

	if uFlag != "" {
		username, password, err := getUsernamePasswordFromFlag(uFlag)
		if err != nil {
			return nil, err
		}
		cfg.Username = username
		cfg.Password = password
	}

	return client.New(cfg)
}

func ContextWithTotalTimeout(c *cli.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.GlobalDuration("total-timeout"))
}
