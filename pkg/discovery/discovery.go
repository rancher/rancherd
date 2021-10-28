package discovery

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-discover"
	"github.com/rancher/rancherd/pkg/config"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/rancher/wrangler/pkg/randomtoken"
	"github.com/rancher/wrangler/pkg/slice"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/cert"

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
	if cfg.Discovery == nil {
		if cfg.Server == "" && cfg.Role == "server" {
			cfg.Role = "cluster-init"
		}
		return nil
	}

	if cfg.Token == "" {
		return fmt.Errorf("token is required to be set when discovery is set")
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
	logrus.Infof("Using role=%s and server=%s", cfg.Role, cfg.Server)
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

	server, err := newJoinServer(ctx, cfg.Discovery.ServerCacheDuration, port)
	if err != nil {
		return "", false, err
	}

	count := cfg.Discovery.ExpectedServers
	if count == 0 {
		count = 3
	}

	for {
		server, clusterInit := server.loop(ctx, count, cfg.Discovery.Params, port, discovery)
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

func (j *joinServer) addresses(params map[string]string, discovery *discover.Discover) ([]string, error) {
	if params["provider"] == "mdns" {
		params["v6"] = "false"
	}
	addrs, err := discovery.Addrs(discover.Config(params).String(), log.Default())
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, addr := range addrs {
		host, _, err := net.SplitHostPort(addr)
		if err == nil {
			ips = append(ips, host)
		} else {
			ips = append(ips, addr)
		}
	}

	return ips, nil
}

func (j *joinServer) loop(ctx context.Context, count int, params map[string]string, port int64, discovery *discover.Discover) (string, bool) {
	addrs, err := j.addresses(params, discovery)
	if err != nil {
		logrus.Errorf("failed to discover peers to: %v", err)
		return "", false
	}

	addrs = j.setPeers(addrs)

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
			logrus.Infof("failed to connect to %s: %v", url, err)
			allAgree = false
			continue
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			logrus.Infof("failed to read response from %s: code %d: %v", url, resp.StatusCode, err)
			allAgree = false
			continue
		}

		rancherID := resp.Header.Get("X-Cattle-Rancherd-Id")
		if rancherID == "" {
			return fmt.Sprintf("https://%s", net.JoinHostPort(addr, strconv.FormatInt(port, 10))), false
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

	if len(addrs) == 0 {
		logrus.Infof("No available peers")
		return "", false
	}

	if firstID != j.id {
		logrus.Infof("Waiting for peer %s from %v to initialize", addrs[0], addrs)
		return "", false
	}

	if len(addrs) != count {
		logrus.Infof("Expecting %d servers currently have %v", count, addrs)
		return "", false
	}

	if !allAgree {
		logrus.Infof("All peers %v do not agree on the peer list", addrs)
		return "", false
	}

	logrus.Infof("Currently the elected leader %s from peers %v", firstID, addrs)
	return "", true
}

type joinServer struct {
	lock          sync.Mutex
	id            string
	peers         []string
	peerSeen      map[string]time.Time
	cacheDuration time.Duration
}

type pingResponse struct {
	Peers []string `json:"peers,omitempty"`
}

func newJoinServer(ctx context.Context, cacheDuration string, port int64) (*joinServer, error) {
	id, err := randomtoken.Generate()
	if err != nil {
		return nil, err
	}

	if cacheDuration == "" {
		cacheDuration = "1m"
	}

	duration, err := time.ParseDuration(cacheDuration)
	if err != nil {
		return nil, err
	}

	j := &joinServer{
		id:            id,
		cacheDuration: duration,
		peerSeen:      map[string]time.Time{},
	}

	cert, key, err := cert.GenerateSelfSignedCertKey("rancherd-bootstrap", nil, nil)
	if err != nil {
		return nil, err
	}
	certs, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	l, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), &tls.Config{
		Certificates: []tls.Certificate{
			certs,
		},
	})
	if err != nil {
		return nil, err
	}
	server := &http.Server{
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Handler: j,
	}
	go func() {
		err := server.Serve(l)
		if err != nil {
			logrus.Errorf("failed to server bootstrap http server: %v", err)
		}
	}()
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
		l.Close()
	}()

	return j, nil
}

func (j *joinServer) setPeers(peers []string) []string {
	j.lock.Lock()
	defer j.lock.Unlock()

	// purge
	now := time.Now()
	for k, v := range j.peerSeen {
		if v.Add(j.cacheDuration).Before(now) {
			logrus.Infof("Forgetting peer %s", k)
			delete(j.peerSeen, k)
		}
	}

	// add
	for _, peer := range peers {
		if _, ok := j.peerSeen[peer]; !ok {
			logrus.Infof("New peer discovered %s", peer)
		}
		j.peerSeen[peer] = now
	}

	// sort
	newPeers := make([]string, 0, len(j.peerSeen))
	for k := range j.peerSeen {
		newPeers = append(newPeers, k)
	}
	sort.Strings(newPeers)

	j.peers = newPeers
	logrus.Infof("current set of peers: %v", j.peers)
	return j.peers
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
