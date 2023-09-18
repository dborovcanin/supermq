// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"encoding/json"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

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

	fmt.Println("Issuing:", key)
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
	if err := jwt.Validate(tkn); err != nil {
		fmt.Println("INALID", err)
		return auth.Key{}, err
	}
	fmt.Println("parsed", tkn)
	//	tt, ok := tkn.Get(tokenType)
	//	if !ok {
	//		return auth.Key{}, errors.Wrap(errors.ErrAuthentication, err)
	//	}
	//	var tType auth.KeyType
	//	for k, v := range tkn.PrivateClaims() {
	//		fmt.Println(k, v)
	//	}
	jsn, err := json.Marshal(tkn.PrivateClaims())
	if err != nil {
		return auth.Key{}, err
	}
	fmt.Println("TYPE")
	fmt.Println(tkn.Get("type"))
	var key auth.Key
	if err := json.Unmarshal(jsn, &key); err != nil {
		return auth.Key{}, err
	}

	// 	switch t := tt.(type) {
	// 	case float64:
	// 		tType = auth.KeyType(t)
	// 	case uint32:
	// 		tType = auth.KeyType(t)
	// 	case int:
	// 		tType = auth.KeyType(t)
	// 	}
	// tt, ok1 := tType.(auth.KeyType)
	// fmt.Println("ttype", tType, reflect.TypeOf(tType))
	// if !(ok && ok1) {
	// 	return auth.Key{}, errors.Wrap(errors.ErrAuthentication, err)
	// }
	//identity, ok := tkn.Get(identity)
	//fmt.Println("type parsed")
	//id, ok1 := identity.(string)
	//if !(ok && ok1) {
	//	return auth.Key{}, errors.Wrap(errors.ErrAuthentication, err)
	//}
	//fmt.Println("key parsed OK", key)
	//	key = auth.Key{
	//		ID:        tkn.JwtID(),
	//		Type:      tType,
	//		Issuer:    tkn.Issuer(),
	//		SubjectID: id,
	//		Subject:   tkn.Subject(),
	//		IssuedAt:  tkn.IssuedAt(),
	//		ExpiresAt: tkn.Expiration(),
	//	}
	key.ID = tkn.JwtID()
	key.Issuer = tkn.Issuer()
	key.Subject = tkn.Subject()
	key.IssuedAt = tkn.IssuedAt()
	key.ExpiresAt = tkn.Expiration()
	fmt.Println("KEY", key)
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
