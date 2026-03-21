package btcvar30

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

type Signer interface {
	Sign(Payload) (string, error)
}

type DeterministicSigner struct {
	secret []byte
}

func NewDeterministicSigner(secret string) *DeterministicSigner {
	return &DeterministicSigner{secret: []byte(secret)}
}

func (s *DeterministicSigner) Sign(payload Payload) (string, error) {
	canonical, err := payload.CanonicalBytes()
	if err != nil {
		return "", err
	}

	// TODO: replace this with the repo's canonical signing scheme once one exists.
	// For v1 we use deterministic HMAC-SHA256 when a key is configured, otherwise a
	// stable content hash placeholder so downstream consumers can integrate now.
	if len(s.secret) == 0 {
		sum := sha256.Sum256(canonical)
		return "sha256:" + hex.EncodeToString(sum[:]), nil
	}

	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write(canonical)
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil)), nil
}
