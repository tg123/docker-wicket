package handler

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/docker/distribution/registry/auth/token"
	"github.com/docker/libtrust"
)

// thanks to https://github.com/cesanta/docker_auth
// copy and paste some code from docker_auth

type TokenAuth struct {
	Issuer     string
	Service    string
	Expiration int64

	publicKey   libtrust.PublicKey
	privateKey  libtrust.PrivateKey
	rootCerts   *x509.CertPool
	trustedKeys map[string]libtrust.PublicKey
}

type AuthRequest struct {
	Account string
	Type    string
	Name    string
	Service string
	Actions []string
}

type ResourceActions []*token.ResourceActions

func loadCertAndKey(certFile, keyFile string) (x509Cert *x509.Certificate, pk libtrust.PublicKey, prk libtrust.PrivateKey, err error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return
	}
	x509Cert, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return
	}
	pk, err = libtrust.FromCryptoPublicKey(x509Cert.PublicKey)
	if err != nil {
		return
	}
	prk, err = libtrust.FromCryptoPrivateKey(cert.PrivateKey)
	return
}

func (t *TokenAuth) LoadCertAndKey(certFile, keyFile string) error {

	cert, pk, prk, err := loadCertAndKey(certFile, keyFile)

	if err != nil {
		return err
	}

	t.publicKey = pk
	t.privateKey = prk

	t.rootCerts = x509.NewCertPool()
	t.rootCerts.AddCert(cert)

	t.trustedKeys = make(map[string]libtrust.PublicKey, 1)
	t.trustedKeys[pk.KeyID()] = pk

	return nil
}

func (t *TokenAuth) Verify(rawToken string, fn func(access ResourceActions) error) error {

	verifyOpts := token.VerifyOptions{
		TrustedIssuers:    []string{t.Issuer},
		AcceptedAudiences: []string{t.Service},
		Roots:             t.rootCerts,
		TrustedKeys:       t.trustedKeys,
	}

	token, err := token.NewToken(rawToken)

	if err != nil {
		return err
	}

	err = token.Verify(verifyOpts)
	if err != nil {
		return err
	}

	return fn(token.Claims.Access)
}

// https://github.com/docker/distribution/blob/master/docs/spec/auth/token.md#example
func (t *TokenAuth) CreateToken(ar *AuthRequest) (string, error) {
	now := time.Now().Unix()

	// Sign something dummy to find out which algorithm is used.
	_, sigAlg, err := t.privateKey.Sign(strings.NewReader("dummy"), 0)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %s", err)
	}
	header := token.Header{
		Type:       "JWT",
		SigningAlg: sigAlg,
		KeyID:      t.publicKey.KeyID(),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %s", err)
	}

	claims := token.ClaimSet{
		Issuer:     t.Issuer,
		Subject:    ar.Account,
		Audience:   ar.Service,
		NotBefore:  now - 1,
		IssuedAt:   now,
		Expiration: now + t.Expiration,
		JWTID:      fmt.Sprintf("%d", rand.Int63()),
		Access:     []*token.ResourceActions{},
	}
	if len(ar.Actions) > 0 {
		claims.Access = []*token.ResourceActions{
			{Type: ar.Type, Name: ar.Name, Actions: ar.Actions},
		}
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %s", err)
	}

	payload := fmt.Sprintf("%s%s%s", joseBase64UrlEncode(headerJSON), token.TokenSeparator, joseBase64UrlEncode(claimsJSON))

	sig, sigAlg2, err := t.privateKey.Sign(strings.NewReader(payload), 0)
	if err != nil || sigAlg2 != sigAlg {
		return "", fmt.Errorf("failed to sign token: %s", err)
	}

	return fmt.Sprintf("%s%s%s", payload, token.TokenSeparator, joseBase64UrlEncode(sig)), nil
}

// Copy-pasted from libtrust where it is private.
func joseBase64UrlEncode(b []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}
