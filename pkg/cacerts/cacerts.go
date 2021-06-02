package cacerts

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	url2 "net/url"
	"time"

	"github.com/rancher/wrangler/pkg/randomtoken"
)

var insecureClient = &http.Client{
	Timeout: time.Second * 5,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

func Get(server, token, path string) ([]byte, string, error) {
	u, err := url2.Parse(server)
	if err != nil {
		return nil, "", err
	}
	u.Path = path

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}

	cacert, caChecksum, err := CACerts(server, token)
	if err != nil {
		return nil, "", err
	}

	var resp *http.Response
	if len(cacert) == 0 {
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, "", err
		}
	} else {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(cacert)
		client := http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
				},
			},
		}
		defer client.CloseIdleConnections()

		resp, err = client.Do(req)
		if err != nil {
			return nil, "", err
		}
	}

	data, err := ioutil.ReadAll(resp.Body)
	return data, caChecksum, err
}

func CACerts(server, token string) ([]byte, string, error) {
	nonce, err := randomtoken.Generate()
	if err != nil {
		return nil, "", err
	}

	url, err := url2.Parse(server)
	if err != nil {
		return nil, "", err
	}

	requestURL := fmt.Sprintf("https://%s/cacerts", url.Host)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("X-Cattle-Nonce", nonce)
	req.Header.Set("Authorization", "Bearer "+hashBase64([]byte(token)))

	resp, err := insecureClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("insecure cacerts download from %s: %w", requestURL, err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	if resp.Header.Get("X-Cattle-Hash") != hash(token, nonce, data) {
		return nil, "", fmt.Errorf("response hash (%s) does not match (%s)",
			resp.Header.Get("X-Cattle-Hash"),
			hash(token, nonce, data))
	}

	if len(data) == 0 {
		return nil, "", nil
	}

	return data, hashHex(data), nil
}

func hashHex(token []byte) string {
	hash := sha256.Sum256(token)
	return hex.EncodeToString(hash[:])
}

func hashBase64(token []byte) string {
	hash := sha256.Sum256(token)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func hash(token, nonce string, bytes []byte) string {
	digest := hmac.New(sha512.New, []byte(token))
	digest.Write([]byte(nonce))
	digest.Write([]byte{0})
	digest.Write(bytes)
	digest.Write([]byte{0})
	hash := digest.Sum(nil)
	return base64.StdEncoding.EncodeToString(hash)
}
