package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/openshift/origin/test/extended/router/certgen"
)

func main() {
	validFor := 100 * time.Hour * 24 * 365 // 100 years
	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	_, crt, key, err := certgen.GenerateKeyPair(notBefore, notAfter, flag.Args()...)
	if err != nil {
		log.Fatalf("failed to generate key pair: %v", err)
	}

	s1, err := certgen.MarshalPrivateKeyToDERFormat(key)
	if err != nil {
		log.Fatalf("failed to marshal key: %v", err)
	}

	s2, err := certgen.MarshalCertToPEMString(crt)
	if err != nil {
		log.Fatalf("failed to marshal crt: %v", err)
	}

	fmt.Print(s1, "\n", s2)
}
