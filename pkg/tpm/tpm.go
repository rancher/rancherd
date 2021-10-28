package tpm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-attestation/attest"
)

func ResolveToken(token string) (bool, string, error) {
	if !strings.HasPrefix(token, "tpm://") {
		return false, token, nil
	}

	hash, err := GetPubHash()
	return true, hash, err
}

func GetPubHash() (string, error) {
	ek, err := getEK()
	if err != nil {
		return "", fmt.Errorf("getting EK: %w", err)
	}

	hash, err := getPubHash(ek)
	if err != nil {
		return "", fmt.Errorf("hashing EK: %w", err)
	}

	return hash, nil
}

func getEK() (*attest.EK, error) {
	var err error
	tpm, err := attest.OpenTPM(&attest.OpenConfig{
		TPMVersion: attest.TPMVersion20,
	})
	if err != nil {
		return nil, fmt.Errorf("opening tpm: %w", err)
	}
	defer tpm.Close()

	eks, err := tpm.EKs()
	if err != nil {
		return nil, err
	}

	if len(eks) == 0 {
		return nil, fmt.Errorf("failed to find EK")
	}

	return &eks[0], nil
}

func getToken(data *AttestationData) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshalling attestation data: %w", err)
	}

	return "Bearer TPM" + base64.StdEncoding.EncodeToString(bytes), nil
}

func getAttestationData() (*AttestationData, []byte, error) {
	var err error
	tpm, err := attest.OpenTPM(&attest.OpenConfig{
		TPMVersion: attest.TPMVersion20,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("opening tpm: %w", err)
	}
	defer tpm.Close()

	eks, err := tpm.EKs()
	if err != nil {
		return nil, nil, err
	}
	ak, err := tpm.NewAK(nil)
	if err != nil {
		return nil, nil, err
	}
	defer ak.Close(tpm)
	params := ak.AttestationParameters()

	if len(eks) == 0 {
		return nil, nil, fmt.Errorf("failed to find EK")
	}

	ek := &eks[0]
	ekBytes, err := EncodeEK(ek)
	if err != nil {
		return nil, nil, err
	}

	aikBytes, err := ak.Marshal()
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling AK: %w", err)
	}

	return &AttestationData{
		EK: ekBytes,
		AK: &params,
	}, aikBytes, nil
}
