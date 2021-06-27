package discovery

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/hashicorp/go-discover"
	"github.com/rancher/dynamiclistener/server"
	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/rancher/wrangler/pkg/randomtoken"
	"github.com/rancher/wrangler/pkg/slice"
	"github.com/sirupsen/logrus"

	// Include kubernetes provider
	_ "github.com/hashicorp/go-discover/provider/k8s"
)

var (
	insecureHTTPClient = http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
)

func DiscoverServerAndRole(ctx context.Context, cfg *config.Config) error {
	if len(cfg.Discovery) == 0 {
		return nil
	}

	server, clusterInit, err := discoverServerAndRole(ctx, cfg)
	if err != nil {
		return err
	}
	if clusterInit {
		cfg.Role = "cluster-init"
	} else if server != "" {
		cfg.Server = server
	}
	return nil

}
func discoverServerAndRole(ctx context.Context, cfg *config.Config) (string, bool, error) {
	discovery, err := discover.New()
	if err != nil {
		return "", false, err
	}

	port, err := convert.ToNumber(cfg.RancherValues["hostPort"])
	if err != nil || port == 0 {
		port = 8443
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	server, err := newJoinServer(ctx, port)
	if err != nil {
		return "", false, err
	}

	for {
		server, clusterInit := server.loop(ctx, cfg.Discovery, port, discovery)
		if clusterInit {
			return "", true, nil
		}
		if server != "" {
			return server, false, nil
		}
		logrus.Info("Waiting to discover server")
		select {
		case <-ctx.Done():
			return "", false, fmt.Errorf("interrupted waiting to discover server: %w", ctx.Err())
		case <-time.After(5 * time.Second):
		}
	}
}

func (j *joinServer) loop(ctx context.Context, params map[string]string, port int64, discovery *discover.Discover) (string, bool) {
	addrs, err := discovery.Addrs(discover.Config(params).String(), log.Default())
	if err != nil {
		logrus.Errorf("failed to discover peers to: %v", err)
		return "", false
	}

	sort.Strings(addrs)
	j.setPeers(addrs)

	var (
		allAgree = true
		firstID  = ""
	)
	for i, addr := range addrs {
		url := fmt.Sprintf("https://%s:%d/cacerts", addr, port)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			logrus.Errorf("failed to construct request for %s: %v", url, err)
			return "", false
		}
		resp, err := insecureHTTPClient.Do(req)
		if err != nil {
			logrus.Errorf("failed to connect to %s: %v", url, err)
			allAgree = false
			continue
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			logrus.Errorf("failed to read response from %s: code %d: %v", url, resp.StatusCode, err)
			allAgree = false
			continue
		}

		rancherID := resp.Header.Get("X-Cattle-Rancherd-Id")
		if rancherID == "" {
			return fmt.Sprintf("https://%s:%d", addr, port), false
		}
		if i == 0 {
			firstID = rancherID
		}

		var pingResponse pingResponse
		if err := json.Unmarshal(data, &pingResponse); err != nil {
			logrus.Errorf("failed to unmarshal response (%s) from %s: %v", data, url, err)
			allAgree = false
			continue
		}

		if !slice.StringsEqual(addrs, pingResponse.Peers) {
			logrus.Infof("Peer %s does not agree on peer list, %v != %v", addr, addrs, pingResponse.Peers)
			allAgree = false
			continue
		}
	}

	if allAgree && len(addrs) > 2 && firstID == j.id {
		return "", true
	}

	return "", false
}

type joinServer struct {
	lock  sync.Mutex
	id    string
	peers []string
}

type pingResponse struct {
	Peers []string `json:"peers,omitempty"`
}

func newJoinServer(ctx context.Context, port int64) (*joinServer, error) {
	id, err := randomtoken.Generate()
	if err != nil {
		return nil, err
	}

	j := &joinServer{
		id: id,
	}

	return j, server.ListenAndServe(ctx, int(port), 0, j, nil)
}

func (j *joinServer) setPeers(peers []string) {
	j.lock.Lock()
	defer j.lock.Unlock()
	logrus.Infof("current set of peers: %v", peers)
	j.peers = peers
}

func (j *joinServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	j.lock.Lock()
	defer j.lock.Unlock()

	rw.Header().Set("X-Cattle-Rancherd-Id", j.id)
	rw.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(rw).Encode(pingResponse{
		Peers: j.peers,
	})
}
