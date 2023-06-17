package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/distribution/registry/auth/token"
	"github.com/docker/libtrust"
	"github.com/joho/godotenv"
	"github.com/lavalleeale/ContinuousIntegration/lib/auth"
	"github.com/lavalleeale/ContinuousIntegration/lib/db"
	sessionseal "github.com/lavalleeale/SessionSeal"
)

type authData struct {
	OrganizationID string `json:"organizationId"`
}

type tokenServer struct {
	privateKey libtrust.PrivateKey
	pubKey     libtrust.PublicKey
	crt, key   string
}

// newTokenServer creates a new tokenServer
func newTokenServer(crt, key string) (*tokenServer, error) {
	pk, prk, err := loadCertAndKey(crt, key)
	if err != nil {
		return nil, err
	}
	t := &tokenServer{privateKey: prk, pubKey: pk, crt: crt, key: key}
	return t, nil
}

// loadCertAndKey from filesystem
func loadCertAndKey(certFile, keyFile string) (libtrust.PublicKey, libtrust.PrivateKey, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, nil, err
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, nil, err
	}
	pk, err := libtrust.FromCryptoPublicKey(x509Cert.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	prk, err := libtrust.FromCryptoPrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, nil, err
	}
	return pk, prk, nil
}

type Option struct {
	issuer, typ, name, account, service string
	actions                             []string // requested actions
}

type Token struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
}

func (srv *tokenServer) createToken(opt *Option, actions []string) (*Token, error) {
	// sign any string to get the used signing Algorithm for the private key
	_, algo, err := srv.privateKey.Sign(strings.NewReader("AUTH"), 0)
	if err != nil {
		return nil, err
	}
	header := token.Header{
		Type:       "JWT",
		SigningAlg: algo,
		KeyID:      srv.pubKey.KeyID(),
	}
	headerJson, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	exp := now.Add(60 * time.Second)
	claim := token.ClaimSet{
		Issuer:     opt.issuer,
		Subject:    opt.account,
		Audience:   opt.service,
		Expiration: exp.Unix(),
		NotBefore:  now.Add(time.Second * -10).Unix(),
		IssuedAt:   now.Unix(),
		JWTID:      fmt.Sprintf("%d", rand.Int63()),
		Access:     []*token.ResourceActions{},
	}
	claim.Access = append(claim.Access, &token.ResourceActions{
		Type:    opt.typ,
		Name:    opt.name,
		Actions: actions,
	})
	claimJson, err := json.Marshal(claim)
	if err != nil {
		return nil, err
	}
	payload := fmt.Sprintf("%s%s%s", base64.RawURLEncoding.EncodeToString(headerJson),
		token.TokenSeparator, base64.RawURLEncoding.EncodeToString(claimJson))
	sig, sigAlgo, err := srv.privateKey.Sign(strings.NewReader(payload), 0)
	if err != nil && sigAlgo != algo {
		return nil, err
	}
	tk := fmt.Sprintf("%s%s%s", payload, token.TokenSeparator,
		base64.RawURLEncoding.EncodeToString(sig))
	return &Token{Token: tk, AccessToken: tk}, nil
}

func (srv *tokenServer) createTokenOption(r *http.Request) *Option {
	opt := &Option{}
	q := r.URL.Query()
	user, _, _ := r.BasicAuth()

	opt.service = q.Get("service")
	opt.account = user
	opt.issuer = "Sample Issuer" // issuer value must match the value configured via docker-compose

	parts := strings.Split(q.Get("scope"), ":")
	if len(parts) > 0 {
		opt.typ = parts[0] // repository
	}
	if len(parts) > 1 {
		opt.name = parts[1] // foo/repoName
	}
	if len(parts) > 2 {
		opt.actions = strings.Split(parts[2], ",") // requested actions
	}
	return opt
}

func (srv *tokenServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	opt := srv.createTokenOption(r)

	data, err := srv.authenticate(r, opt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	actions := srv.authorize(opt, data)
	tk, err := srv.createToken(opt, actions)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	srv.ok(w, tk)
}

func (srv *tokenServer) authenticate(r *http.Request, opt *Option) (authData, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return authData{}, fmt.Errorf("auth credentials not found")
	}
	if username != opt.account {
		return authData{}, fmt.Errorf("invalid username")
	}

	if username == "token" {
		marshaledData, err := sessionseal.Unseal(os.Getenv("JWT_SECRET"), password)
		if err != nil {
			return authData{}, fmt.Errorf("invalid password")
		}

		var data authData

		err = json.Unmarshal(marshaledData, &data)

		if err != nil {
			return authData{}, fmt.Errorf("invalid json")
		}

		return data, nil
	} else {
		user, err := auth.Login(username, password, false)
		if err != nil {
			return authData{}, err
		}
		return authData{OrganizationID: user.OrganizationID}, nil
	}
}

func (srv *tokenServer) authorize(opt *Option, data authData) []string {
	if data.OrganizationID == "root" {
		return []string{"pull", "push"}
	}
	if strings.Split(opt.name, "/")[0] == data.OrganizationID {
		return []string{"pull", "push"}
	}
	// unauthorized, no permission is granted
	return []string{}
}

func (srv *tokenServer) run() error {
	http.Handle("/auth", srv)
	return http.ListenAndServeTLS(":8081", srv.crt, srv.key, nil)
}

func (srv *tokenServer) ok(w http.ResponseWriter, tk *Token) {
	data, _ := json.Marshal(tk)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}

	db.Open()

	srv, err := newTokenServer("certs/auth.crt", "certs/auth.key")
	if err != nil {
		panic(err)
	}
	if err := srv.run(); err != nil {
		panic(err)
	}
}
