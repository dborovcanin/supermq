// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	issuerName = "mainflux.auth"
	identity   = "identity"
	tokenType  = "type"
)

type tokenizer struct {
	secret []byte
}

// NewRepository instantiates an implementation of Token repository.
func New(secret []byte) auth.Tokenizer {
	return &tokenizer{
		secret: secret,
	}
}

func (t tokenizer) Issue(key auth.Key) (string, error) {
	tkn, err := jwt.NewBuilder().
		Issuer(issuerName).
		IssuedAt(time.Now()).
		Subject(key.ID).
		Claim(issuerName, key.Subject).
		Claim(tokenType, key.Type).
		Expiration(key.ExpiresAt).Build()
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthentication, err)
	}
	signedTkn, err := jwt.Sign(tkn, jwt.WithKey(jwa.HS512, t.secret))
	if err != nil {
		return "", err
	}
	return string(signedTkn), nil
}

func (t tokenizer) Parse(token string) (auth.Key, error) {
	tkn, err := jwt.Parse(
		[]byte(token),
		jwt.WithValidate(true),
		jwt.WithKey(jwa.HS512, t.secret),
	)
	if err != nil {
		return auth.Key{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	tType, ok := tkn.Get(tokenType)
	tt, ok1 := tType.(uint32)
	if !(ok && ok1) {
		return auth.Key{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	identity, ok := tkn.Get(identity)
	id, ok1 := identity.(string)
	if !(ok && ok1) {
		return auth.Key{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	key := auth.Key{
		ID:      tkn.Subject(),
		Type:    tt,
		Subject: id,
	}

	return key, nil
}

// func (svc tokenizer) Issue(key auth.Key) (string, error) {
// 	claims := claims{
// 		StandardClaims: jwt.StandardClaims{
// 			Issuer:   issuerName,
// 			Subject:  key.Subject,
// 			IssuedAt: key.IssuedAt.UTC().Unix(),
// 		},
// 		IssuerID: key.IssuerID,
// 		Type:     &key.Type,
// 	}

// 	if !key.ExpiresAt.IsZero() {
// 		claims.ExpiresAt = key.ExpiresAt.UTC().Unix()
// 	}
// 	if key.ID != "" {
// 		claims.Id = key.ID
// 	}

// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	return token.SignedString([]byte(svc.secret))
// }

// func (svc tokenizer) Parse(token string) (auth.Key, error) {
// 	c := claims{}
// 	_, err := jwt.ParseWithClaims(token, &c, func(token *jwt.Token) (interface{}, error) {
// 		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
// 			return nil, errors.ErrAuthentication
// 		}
// 		return []byte(svc.secret), nil
// 	})

// 	if err != nil {
// 		if e, ok := err.(*jwt.ValidationError); ok && e.Errors == jwt.ValidationErrorExpired {
// 			// Expired User key needs to be revoked.
// 			if c.Type != nil && *c.Type == auth.APIKey {
// 				return c.toKey(), auth.ErrAPIKeyExpired
// 			}
// 			return auth.Key{}, errors.Wrap(auth.ErrKeyExpired, err)
// 		}
// 		return auth.Key{}, errors.Wrap(errors.ErrAuthentication, err)
// 	}

// 	return c.toKey(), nil
// }

// func (c claims) toKey() auth.Key {
// 	key := auth.Key{
// 		ID:       c.Id,
// 		IssuerID: c.IssuerID,
// 		Subject:  c.Subject,
// 		IssuedAt: time.Unix(c.IssuedAt, 0).UTC(),
// 	}
// 	if c.ExpiresAt != 0 {
// 		key.ExpiresAt = time.Unix(c.ExpiresAt, 0).UTC()
// 	}

// 	// Default type is 0.
// 	if c.Type != nil {
// 		key.Type = *c.Type
// 	}

// 	return key
// }
