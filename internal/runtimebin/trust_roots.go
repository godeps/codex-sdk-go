package runtimebin

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

type TrustRoot struct {
	KeyID     string
	PublicKey ed25519.PublicKey
}

var defaultTrustRoots = []TrustRoot{
	{
		KeyID:     "runtime-manifest-v3",
		PublicKey: decodePublicKey("tPmRDQcML5QPrkLA8g6nWiIz5PQ3ekzs/cW9qq9D8XQ="),
	},
}

func DefaultTrustRoots() []TrustRoot {
	out := make([]TrustRoot, len(defaultTrustRoots))
	copy(out, defaultTrustRoots)
	return out
}

func decodePublicKey(value string) ed25519.PublicKey {
	bytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return ed25519.PublicKey(bytes)
}

func VerifyManifestSignature(manifest []byte, envelope SignatureEnvelope, roots []TrustRoot) error {
	if envelope.Algorithm != SignatureAlgorithm {
		return fmt.Errorf("runtimebin: unsupported signature algorithm %q", envelope.Algorithm)
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil {
		return fmt.Errorf("runtimebin: invalid signature encoding: %w", err)
	}
	for _, root := range roots {
		if root.KeyID != envelope.KeyID {
			continue
		}
		if ed25519.Verify(root.PublicKey, manifest, signature) {
			return nil
		}
		return fmt.Errorf("runtimebin: manifest signature verification failed for key %q", envelope.KeyID)
	}
	return fmt.Errorf("runtimebin: untrusted manifest signing key %q", envelope.KeyID)
}

func SignManifest(manifest []byte, keyID string, privateKey ed25519.PrivateKey) (SignatureEnvelope, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return SignatureEnvelope{}, fmt.Errorf("runtimebin: invalid ed25519 private key length %d", len(privateKey))
	}
	return SignatureEnvelope{
		KeyID:     keyID,
		Algorithm: SignatureAlgorithm,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, manifest)),
	}, nil
}
