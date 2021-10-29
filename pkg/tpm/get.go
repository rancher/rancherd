package tpm

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-attestation/attest"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func Get(cacerts []byte, url string, header http.Header) ([]byte, error) {
	dialer := websocket.DefaultDialer
	if len(cacerts) > 0 {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(cacerts)
		dialer = &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		}
	}

	attestationData, aikBytes, err := getAttestationData()
	if err != nil {
		return nil, err
	}

	hash, err := GetPubHash()
	if err != nil {
		return nil, err
	}

	token, err := getToken(attestationData)
	if err != nil {
		return nil, err
	}

	if header == nil {
		header = http.Header{}
	}
	header.Add("Authorization", token)
	wsURL := strings.Replace(url, "http", "ws", 1)
	logrus.Infof("Using TPMHash %s to dial %s", hash, wsURL)
	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			data, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				return nil, errors.New(string(data))
			}
		}
		return nil, err
	}
	defer conn.Close()

	_, msg, err := conn.NextReader()
	if err != nil {
		return nil, fmt.Errorf("reading challenge: %w", err)
	}

	var challenge Challenge
	if err := json.NewDecoder(msg).Decode(&challenge); err != nil {
		return nil, fmt.Errorf("unmarshaling Challenge: %w", err)
	}

	challengeResp, err := getChallengeResponse(challenge.EC, aikBytes)
	if err != nil {
		return nil, err
	}

	writer, err := conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	if err := json.NewEncoder(writer).Encode(challengeResp); err != nil {
		return nil, fmt.Errorf("encoding ChallengeResponse: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing websocket writer: %w", err)
	}

	_, msg, err = conn.NextReader()
	if err != nil {
		return nil, fmt.Errorf("reading payload from tpm get: %w", err)
	}

	return ioutil.ReadAll(msg)
}

func getChallengeResponse(ec *attest.EncryptedCredential, aikBytes []byte) (*ChallengeResponse, error) {
	tpm, err := attest.OpenTPM(&attest.OpenConfig{
		TPMVersion: attest.TPMVersion20,
	})
	if err != nil {
		return nil, fmt.Errorf("opening tpm: %w", err)
	}
	defer tpm.Close()

	aik, err := tpm.LoadAK(aikBytes)
	if err != nil {
		return nil, err
	}
	defer aik.Close(tpm)

	secret, err := aik.ActivateCredential(tpm, *ec)
	if err != nil {
		return nil, fmt.Errorf("failed to activate credential: %w", err)
	}
	return &ChallengeResponse{
		Secret: secret,
	}, nil
}
