// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"context"
	"encoding/json"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

var errInvalidIssuer = errors.New("invalid token issuer value")

const (
	issuerName = "mainflux.auth"
	identity   = "subject_id"
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
		JwtID(key.ID).
		Issuer(issuerName).
		IssuedAt(key.IssuedAt).
		Subject(key.Subject).
		Claim(identity, key.SubjectID).
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
	validator := jwt.ValidatorFunc(func(_ context.Context, t jwt.Token) jwt.ValidationError {
		if t.Issuer() != issuerName {
			return jwt.NewValidationError(errInvalidIssuer)
		}
		return nil
	})
	if err := jwt.Validate(tkn, jwt.WithValidator(validator)); err != nil {
		return auth.Key{}, err
	}

	jsn, err := json.Marshal(tkn.PrivateClaims())
	if err != nil {
		return auth.Key{}, err
	}
	var key auth.Key
	if err := json.Unmarshal(jsn, &key); err != nil {
		return auth.Key{}, err
	}

	key.ID = tkn.JwtID()
	key.Issuer = tkn.Issuer()
	key.Subject = tkn.Subject()
	key.IssuedAt = tkn.IssuedAt()
	key.ExpiresAt = tkn.Expiration()
	return key, nil
}
