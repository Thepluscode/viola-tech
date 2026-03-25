package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
)

type jwksDoc struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Use string `json:"use"`
	Alg string `json:"alg"`

	// RSA
	N string `json:"n"`
	E string `json:"e"`

	// EC
	X string `json:"x"`
	Y string `json:"y"`
}

func fetchJWKS(ctx context.Context, httpc *http.Client, url string) (map[string]crypto.PublicKey, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("jwks fetch non-2xx")
	}

	var doc jwksDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	out := map[string]crypto.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kid == "" {
			continue
		}
		pk, err := jwkToPublicKey(k)
		if err != nil {
			continue
		}
		out[k.Kid] = pk
	}
	if len(out) == 0 {
		return nil, errors.New("jwks contained no usable keys")
	}
	return out, nil
}

func jwkToPublicKey(k jwk) (crypto.PublicKey, error) {
	switch k.Kty {
	case "RSA":
		n, err := b64ToBigInt(k.N)
		if err != nil {
			return nil, err
		}
		eBytes, err := b64ToBytes(k.E)
		if err != nil {
			return nil, err
		}
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		return &rsa.PublicKey{N: n, E: e}, nil

	case "EC":
		x, err := b64ToBigInt(k.X)
		if err != nil {
			return nil, err
		}
		y, err := b64ToBigInt(k.Y)
		if err != nil {
			return nil, err
		}
		var curve elliptic.Curve
		switch k.Crv {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, errors.New("unsupported EC curve")
		}
		return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
	default:
		return nil, errors.New("unsupported kty")
	}
}

func b64ToBytes(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func b64ToBigInt(s string) (*big.Int, error) {
	b, err := b64ToBytes(s)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(b), nil
}
