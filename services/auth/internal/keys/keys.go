// Package keys manages the RSA signing key pair used to issue and verify JWTs.
// In production, the private key is loaded from an env var or secrets manager.
// In dev, a key is generated at startup and the public JWKS endpoint is served
// so the gateway-api can verify tokens without a separate IdP.
package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
)

// KeyPair holds the RSA private key used for signing and the JWKS JSON
// representation of the corresponding public key.
type KeyPair struct {
	Private *rsa.PrivateKey
	KID     string
	JWKS    []byte // pre-serialised JSON for the /jwks endpoint
}

// Load returns a KeyPair. If SIGNING_KEY_PEM is set it reads that PEM;
// otherwise it generates a fresh 2048-bit key (dev mode only).
func Load() (*KeyPair, error) {
	kid := os.Getenv("SIGNING_KID")
	if kid == "" {
		kid = "viola-dev-key-1"
	}

	var priv *rsa.PrivateKey

	if pemStr := os.Getenv("SIGNING_KEY_PEM"); pemStr != "" {
		block, _ := pem.Decode([]byte(pemStr))
		if block == nil {
			return nil, errors.New("keys: SIGNING_KEY_PEM is not valid PEM")
		}
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			// Try PKCS1
			key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, errors.New("keys: cannot parse SIGNING_KEY_PEM")
			}
		}
		var ok bool
		priv, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("keys: SIGNING_KEY_PEM is not an RSA key")
		}
	} else {
		var err error
		priv, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
	}

	jwks, err := buildJWKS(kid, &priv.PublicKey)
	if err != nil {
		return nil, err
	}

	return &KeyPair{Private: priv, KID: kid, JWKS: jwks}, nil
}

// buildJWKS serialises the public key as a JSON Web Key Set.
func buildJWKS(kid string, pub *rsa.PublicKey) ([]byte, error) {
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())

	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kid,
				"n":   n,
				"e":   e,
			},
		},
	}
	return json.Marshal(jwks)
}
