package cert

import (
	"crypto/x509"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nickshiro/veil/common"
	"github.com/nickshiro/veil/common/errors"
)

func TestGenerate(t *testing.T) {
	err := generate(nil, true, true)
	if err != nil {
		t.Fatal(err)
	}
}

func generate(domainNames []string, isCA bool, jsonOutput bool) error {
	commonName := "Xray Root CA"
	organization := "Xray Inc"

	expire := time.Hour * 3

	var opts []Option
	if isCA {
		opts = append(opts, Authority(isCA))
		opts = append(opts, KeyUsage(x509.KeyUsageCertSign|x509.KeyUsageKeyEncipherment|x509.KeyUsageDigitalSignature))
	}

	opts = append(opts, NotAfter(time.Now().Add(expire)))
	opts = append(opts, CommonName(commonName))
	if len(domainNames) > 0 {
		opts = append(opts, DNSNames(domainNames...))
	}
	opts = append(opts, Organization(organization))

	cert, err := Generate(nil, opts...)
	if err != nil {
		return errors.New("failed to generate TLS certificate").Base(err)
	}

	if jsonOutput {
		printJSON(cert)
	}

	return nil
}

type jsonCert struct {
	Certificate []string `json:"certificate"`
	Key         []string `json:"key"`
}

func printJSON(certificate *Certificate) {
	certPEM, keyPEM := certificate.ToPEM()
	jCert := &jsonCert{
		Certificate: strings.Split(strings.TrimSpace(string(certPEM)), "\n"),
		Key:         strings.Split(strings.TrimSpace(string(keyPEM)), "\n"),
	}
	content, err := json.MarshalIndent(jCert, "", "  ")
	common.Must(err)
	os.Stdout.Write(content)
	os.Stdout.WriteString("\n")
}
