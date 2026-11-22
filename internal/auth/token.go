package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	TokenIssuer    = "sentinela-api"
	TokenAudience  = "sentinela-api"
	AccessTokenTTL = 15 * time.Minute
)

var (
	ErrInvalidKey   = errors.New("chave invalida: use 64 caracteres hexadecimais")
	ErrInvalidToken = errors.New("token invalido")
	ErrExpiredToken = errors.New("token expirado")
)

var tokenPrefix = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","typ":"JWT"}`)) + "."

type tokenClaims struct {
	Issuer    string    `json:"iss"`
	Subject   uuid.UUID `json:"sub"`
	Audience  string    `json:"aud"`
	IssuedAt  int64     `json:"iat"`
	NotBefore int64     `json:"nbf"`
	ExpiresAt int64     `json:"exp"`
}

type TokenManager struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

func NewTokenManager(key string) (*TokenManager, error) {
	seed, err := hex.DecodeString(key)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, ErrInvalidKey
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	return &TokenManager{
		privateKey: privateKey,
		publicKey:  privateKey.Public().(ed25519.PublicKey),
	}, nil
}

func (m *TokenManager) Issue(userID uuid.UUID, now time.Time) string {
	payload, _ := json.Marshal(tokenClaims{
		Issuer:    TokenIssuer,
		Subject:   userID,
		Audience:  TokenAudience,
		IssuedAt:  now.Unix(),
		NotBefore: now.Unix(),
		ExpiresAt: now.Add(AccessTokenTTL).Unix(),
	})
	signed := tokenPrefix + base64.RawURLEncoding.EncodeToString(payload)
	signature := ed25519.Sign(m.privateKey, []byte(signed))
	return signed + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func (m *TokenManager) Verify(token string, now time.Time) (uuid.UUID, error) {
	rest, ok := strings.CutPrefix(token, tokenPrefix)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}
	payloadText, signatureText, ok := strings.Cut(rest, ".")
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(signatureText)
	if err != nil || !ed25519.Verify(m.publicKey, []byte(tokenPrefix+payloadText), signature) {
		return uuid.Nil, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadText)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	if claims.Issuer != TokenIssuer || claims.Audience != TokenAudience || now.Unix() < claims.NotBefore {
		return uuid.Nil, ErrInvalidToken
	}
	if now.Unix() >= claims.ExpiresAt {
		return uuid.Nil, ErrExpiredToken
	}
	return claims.Subject, nil
}
