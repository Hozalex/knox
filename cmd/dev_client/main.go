package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/user"
	"time"

	"github.com/pinterest/knox"
	"github.com/pinterest/knox/client"
)

// certPEMBlock is the certificate signed by the CA to identify the machine using the client
// (Should be pulled from a file or via another process)
const certPEMBlock = `-----BEGIN CERTIFICATE-----
MIICkjCCAjmgAwIBAgIUTCFC8AeJzYjPVDb4wvQbqVcbl/8wCgYIKoZIzj0EAwIw
fjELMAkGA1UEBhMCVVMxFjAUBgNVBAgTDVNhbiBGcmFuY2lzY28xCzAJBgNVBAcT
AkNBMRgwFgYDVQQKEw9NeSBDb21wYW55IE5hbWUxEzARBgNVBAsTCk9yZyBVbml0
IDIxGzAZBgNVBAMTEnVzZU9ubHlJbkRldk9yVGVzdDAeFw0xNjA0MjgxODQ4MDBa
Fw0xNzA0MjgxODQ4MDBaMGkxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9y
bmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMR8wHQYDVQQKExZJbnRlcm5ldCBX
aWRnZXRzLCBJbmMuMQwwCgYDVQQLEwNXV1cwWTATBgcqhkjOPQIBBggqhkjOPQMB
BwNCAATxZJgi7YWQtewgoC3dKrooyq4Be7u1yghoT4OiFiOqUmgUxfQiVenSJIUM
A2pgcOix66a9j/4KqGqAi3WmFdmSo4GpMIGmMA4GA1UdDwEB/wQEAwIFoDAdBgNV
HSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAdBgNVHQ4E
FgQUMiS1WUuVsQLL5cxbzNRDUojaHFkwHwYDVR0jBBgwFoAUTS1iWIo7D/Erlcqp
YD12QGouqlYwJwYDVR0RBCAwHoILZXhhbXBsZS5jb22CD3d3dy5leGFtcGxlLmNv
bTAKBggqhkjOPQQDAgNHADBEAiAvguEAh8iAyJsG8bb/5z6z5LQQZtVRqeNSes2i
YEgUtwIgTQ0dbp7Gtm5PcTTYQb83Hbo9MDGIi4FEfL0Rw4P4Tyw=
-----END CERTIFICATE-----`

// keyPEMBlock is the private key that should only be available on the machine running this client
// (Should be pulled from a file or via another process)
const keyPEMBlock = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEILwfFi1LNc4OG8GAQTHTFC4WbtgwUUfoYNnrrrtaAIH+oAoGCCqGSM49
AwEHoUQDQgAE8WSYIu2FkLXsIKAt3Sq6KMquAXu7tcoIaE+DohYjqlJoFMX0IlXp
0iSFDANqYHDoseumvY/+CqhqgIt1phXZkg==
-----END EC PRIVATE KEY-----`

// hostname is the host running the knox server
const hostname = "localhost:9000"

// tokenEndpoint and clientID are used by "knox login" if your oauth client supports password flows.
const tokenEndpoint = "https://oauth.token.endpoint.used.for/knox/login"
const clientID = ""

// keyFolder is the directory where keys are cached
const keyFolder = "/var/lib/knox/v0/keys/"

// authTokenResp is the format of the OAuth response generated by "knox login"
type authTokenResp struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

// getCert returns the cert in the tls.Certificate format. This should be a config option in prod.
func getCert() (tls.Certificate, error) {
	return tls.X509KeyPair([]byte(certPEMBlock), []byte(keyPEMBlock))
}

// authHandler is used to generate an authentication header.
// The server expects VersionByte + TypeByte + IDToPassToAuthHandler.
func authHandler() string {
	if s := os.Getenv("KNOX_USER_AUTH"); s != "" {
		return "0u" + s
	}
	if s := os.Getenv("KNOX_MACHINE_AUTH"); s != "" {
		c, _ := getCert()
		x509Cert, err := x509.ParseCertificate(c.Certificate[0])
		if err != nil {
			return "0t" + s
		}
		if len(x509Cert.Subject.CommonName) > 0 {
			return "0t" + x509Cert.Subject.CommonName
		} else if len(x509Cert.DNSNames) > 0 {
			return "0t" + x509Cert.DNSNames[0]
		} else {
			return "0t" + s
		}
	}
	u, err := user.Current()
	if err != nil {
		return ""
	}

	d, err := ioutil.ReadFile(u.HomeDir + "/.knox_user_auth")
	if err != nil {
		return ""
	}
	var authResp authTokenResp
	err = json.Unmarshal(d, &authResp)
	if err != nil {
		return ""
	}

	return "0u" + authResp.AccessToken
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	tlsConfig := &tls.Config{
		ServerName:         "knox",
		InsecureSkipVerify: true,
	}

	cert, err := getCert()
	if err == nil {
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	cli := &knox.HTTPClient{
		Host:        hostname,
		AuthHandler: authHandler,
		KeyFolder:   keyFolder,
		Client:      &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}},
	}

	client.Run(cli, &client.VisibilityParams{log.Printf, log.Printf, func(map[string]uint64) {}}, tokenEndpoint, clientID)
}
