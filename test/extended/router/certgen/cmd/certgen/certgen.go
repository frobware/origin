package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/openshift/origin/test/extended/router/certgen"
)

func main() {
	notBefore := time.Now()

	cfg := certgen.Config{
		Organization:          []string{"Cert Gen Company"},
		CommonName:            "testcert",
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(100 * time.Hour * 24 * 365), // 100 years
		SubjectAlternateNames: flag.Args(),
	}

	crt, key, err := certgen.GenerateKeyPair(cfg)

	if err != nil {
		log.Fatalf("failed to generate key pair: %v", err)
	}

	s1, err := certgen.MarshalKeyToDERFormat(key)
	if err != nil {
		log.Fatalf("failed to marshal key: %v", err)
	}

	s2, err := certgen.MarshalCertToPEMString(crt)
	if err != nil {
		log.Fatalf("failed to marshal crt: %v", err)
	}

	fmt.Print(s1, s2)
}
