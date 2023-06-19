package lib

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/docker/distribution/registry/auth/token"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Token struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
}

func CreateToken(opt *Option, actions []string, cert *tls.Certificate) (*Token, error) {
	now := time.Now()
	exp := now.Add(60 * time.Second)
	tok, err := jwt.NewBuilder().
		Issuer(opt.issuer).
		Subject(opt.account).
		Claim(jwt.AudienceKey, opt.service).
		JwtID(fmt.Sprintf("%d", rand.Int63())).
		Claim("access", []*token.ResourceActions{{
			Type:    opt.typ,
			Name:    opt.name,
			Actions: actions,
		}}).
		Build()
	if err != nil {
		return nil, err
	}
	claims, err := tok.AsMap(context.TODO())
	claims[jwt.ExpirationKey] = exp.Unix()
	claims[jwt.NotBeforeKey] = now.Add(time.Second * -10).Unix()
	claims[jwt.IssuedAtKey] = now.Unix()
	claims[jwt.AudienceKey] = opt.service
	if err != nil {
		return nil, err
	}
	claimsData, err := json.Marshal(claims)
	if err != nil {
		return nil, err
	}
	hdrs := jws.NewHeaders()
	hdrs.Set("typ", "JWT")
	hdrs.Set("x5C", cert.Certificate)
	payloadData, err := jws.Sign(claimsData, jws.WithKey(jwa.RS256, cert.PrivateKey, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		return nil, err
	}
	payloadString := string(payloadData)
	return &Token{Token: payloadString, AccessToken: payloadString}, nil
}
