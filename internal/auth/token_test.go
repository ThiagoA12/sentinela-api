package auth_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ThiagoA12/sentinela-api/internal/auth"
)

const (
	testKey       = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"
	testPublicKey = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"
	testHeader    = `{"alg":"EdDSA","typ":"JWT"}`
	testClaims    = `{"iss":"sentinela-api","sub":"01a1cab1-9c80-7a10-8000-000000000001","aud":"sentinela-api","iat":1800000000,"nbf":1800000000,"exp":1800000900}`
)

var (
	testUserID = uuid.MustParse("01a1cab1-9c80-7a10-8000-000000000001")
	issuedAt   = time.Unix(1800000000, 0)
)

func newTestManager(t *testing.T) *auth.TokenManager {
	t.Helper()
	m, err := auth.NewTokenManager(testKey)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	return m
}

func testPrivateKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	seed, err := hex.DecodeString(testKey)
	if err != nil {
		t.Fatalf("semente de teste: %v", err)
	}
	return ed25519.NewKeyFromSeed(seed)
}

func b64(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func sign(signed string, key ed25519.PrivateKey) string {
	return signed + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(signed)))
}

func forge(header, claims string, key ed25519.PrivateKey) string {
	return sign(b64(header)+"."+b64(claims), key)
}

func TestNewTokenManager(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{"64 caracteres hexadecimais", testKey, nil},
		{"maiusculas", strings.ToUpper(testKey), nil},
		{"vazia", "", auth.ErrInvalidKey},
		{"31 bytes", testKey[:62], auth.ErrInvalidKey},
		{"33 bytes", testKey + "00", auth.ErrInvalidKey},
		{"fora do hexadecimal", "zz" + testKey[2:], auth.ErrInvalidKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := auth.NewTokenManager(tt.key); !errors.Is(err, tt.wantErr) {
				t.Errorf("NewTokenManager(%d caracteres) = %v, quero %v", len(tt.key), err, tt.wantErr)
			}
		})
	}
}

func TestIssue(t *testing.T) {
	token := newTestManager(t).Issue(testUserID, issuedAt)

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token = %q, quero tres partes separadas por ponto", token)
	}
	if header, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil || string(header) != testHeader {
		t.Errorf("cabecalho = %s (%v), quero %s", header, err, testHeader)
	}
	if claims, err := base64.RawURLEncoding.DecodeString(parts[1]); err != nil || string(claims) != testClaims {
		t.Errorf("claims = %s (%v), quero %s", claims, err, testClaims)
	}
	publicKey, err := hex.DecodeString(testPublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		t.Fatalf("chave publica de teste invalida: %v", err)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(publicKey, []byte(parts[0]+"."+parts[1]), signature) {
		t.Errorf("a assinatura nao confere com a chave publica do RFC 8032 (%v)", err)
	}
}

func TestVerify(t *testing.T) {
	m := newTestManager(t)
	token := m.Issue(testUserID, issuedAt)

	for _, now := range []time.Time{issuedAt, issuedAt.Add(auth.AccessTokenTTL - time.Second)} {
		got, err := m.Verify(token, now)
		if err != nil || got != testUserID {
			t.Errorf("Verify %v depois da emissao = %v, %v; quero %v, nil", now.Sub(issuedAt), got, err, testUserID)
		}
	}
}

func TestVerifyRecusa(t *testing.T) {
	m := newTestManager(t)
	key := testPrivateKey(t)
	otherKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	valid := m.Issue(testUserID, issuedAt)
	if forged := forge(testHeader, testClaims, key); forged != valid {
		t.Fatalf("forge = %s\nquero %s", forged, valid)
	}
	parts := strings.Split(valid, ".")

	tests := []struct {
		name  string
		token string
		now   time.Time
		want  error
	}{
		{"alg none sem assinatura", b64(`{"alg":"none","typ":"JWT"}`) + "." + b64(testClaims) + ".", issuedAt, auth.ErrInvalidToken},
		{"assinatura de outra chave", forge(testHeader, testClaims, otherKey), issuedAt, auth.ErrInvalidToken},
		{"expirado", valid, issuedAt.Add(auth.AccessTokenTTL), auth.ErrExpiredToken},
		{"outro emissor", forge(testHeader, strings.Replace(testClaims, `"iss":"sentinela-api"`, `"iss":"outro"`, 1), key), issuedAt, auth.ErrInvalidToken},
		{"alg none com assinatura valida", forge(`{"alg":"none","typ":"JWT"}`, testClaims, key), issuedAt, auth.ErrInvalidToken},
		{"sem a assinatura", parts[0] + "." + parts[1], issuedAt, auth.ErrInvalidToken},
		{"sub trocado com a assinatura original", parts[0] + "." + b64(strings.Replace(testClaims, "000000000001", "000000000002", 1)) + "." + parts[2], issuedAt, auth.ErrInvalidToken},
		{"claims fora do base64", sign(parts[0]+".@@@", key), issuedAt, auth.ErrInvalidToken},
		{"claims fora do JSON", forge(testHeader, "nao e json", key), issuedAt, auth.ErrInvalidToken},
		{"outra audiencia", forge(testHeader, strings.Replace(testClaims, `"aud":"sentinela-api"`, `"aud":"outra"`, 1), key), issuedAt, auth.ErrInvalidToken},
		{"antes do nbf", valid, issuedAt.Add(-time.Second), auth.ErrInvalidToken},
		{"sem exp", forge(testHeader, strings.Replace(testClaims, `,"exp":1800000900`, "", 1), key), issuedAt, auth.ErrExpiredToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.Verify(tt.token, tt.now)
			if !errors.Is(err, tt.want) || got != uuid.Nil {
				t.Errorf("Verify = %v, %v; quero %v, %v", got, err, uuid.Nil, tt.want)
			}
		})
	}
}
