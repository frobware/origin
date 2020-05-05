package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

// readFile reads all data from filename, or fatally fails if an error
// occurs.
func readFile(filename string) []byte {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("failed to read %q: %v", filename, err)
	}
	return data
}

func addPrefix(lines []string, prefix string) []string {
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return lines
}

// split string into chunks limited in length by size.
// Note: assumes 1:1 mapping between bytes/chars (i.e., non-UTF).
func split(s string, size int) []string {
	var chunks []string

	for len(s) > 0 {
		if len(s) < size {
			size = len(s)
		}
		chunks, s = append(chunks, s[:size]), s[size:]
	}

	return chunks
}

func makeTarData(filenames []string) []byte {
	buf := new(bytes.Buffer)

	gz, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		log.Fatalf("Error: gzip.NewWriterLevel(): %v", err)
	}

	tw := tar.NewWriter(gz)

	for _, filename := range filenames {
		fi, err := os.Stat(filename)
		if err != nil {
			log.Fatalf("Error: failed to stat %q: %v", filename, err)
		}

		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			log.Fatalf("Error: failed to create tar header for %q: %v", filename, err)

		}

		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatal(err)
		}

		if _, err := tw.Write(readFile(filename)); err != nil {
			log.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		log.Fatal(err)
	}

	if err := gz.Close(); err != nil {
		log.Fatal(err)
	}

	return buf.Bytes()
}

func keyToString(key *ecdsa.PrivateKey) []string {
	data, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		log.Fatalf("unable to marshal ECDSA private key: %v", err)
	}

	buf := &bytes.Buffer{}

	if err := pem.Encode(buf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: data}); err != nil {
		log.Fatal(err)
	}

	return strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
}

func certToString(derBytes []byte) []string {
	buf := &bytes.Buffer{}

	if err := pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("failed to encode cert data: %v", err)
	}

	return strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
}

func genCertKeyPair(hosts ...string) ([]byte, *ecdsa.PrivateKey) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %v", err)
	}

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate ECDSA key: %v", err)
	}

	validFor := 100 * time.Hour * 24 * 365 // 100 years
	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	rootTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Red Hat"},
			CommonName:   "Root CA",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	if _, err := x509.CreateCertificate(rand.Reader, &rootTemplate, &rootTemplate, &rootKey.PublicKey, rootKey); err != nil {
		log.Fatalf("failed to create root certificate: %v", err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate ECDSA key: %v", err)
	}

	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %v", err)
	}

	leafCertTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Red Hat"},
			CommonName:   "test_cert",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			leafCertTemplate.IPAddresses = append(leafCertTemplate.IPAddresses, ip)
		} else {
			leafCertTemplate.DNSNames = append(leafCertTemplate.DNSNames, h)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &leafCertTemplate, &rootTemplate, &leafKey.PublicKey, rootKey)
	if err != nil {
		log.Fatalf("failed to create leaf certificate: %v", err)
	}

	return derBytes, leafKey
}

func main() {
	flag.Parse()

	data := split(base64.StdEncoding.EncodeToString(makeTarData(flag.Args())), 76)

	tlsSpacer := strings.Repeat(" ", 8)
	dataSpacer := strings.Repeat(" ", 6)

	edgeCert, edgeKey := genCertKeyPair("grpc-interop-edge-*.apps")
	reencryptCert, reencryptKey := genCertKeyPair("grpc-interop-reencrypt-*.apps")

	fmt.Printf(`apiVersion: v1
kind: Template
objects:
- apiVersion: v1
  kind: Service
  metadata:
    name: grpc-interop
    annotations:
      service.beta.openshift.io/serving-cert-secret-name: service-certs
  spec:
    selector:
      app: grpc-interop
    ports:
      - port: 8443
        name: https
        targetPort: 8443
        protocol: TCP
      - port: 1110
        name: h2c
        targetPort: 1110
        protocol: TCP
- apiVersion: v1
  kind: ConfigMap
  labels:
    app: grpc-interop
  metadata:
    name: src-config
  data:
    data.base64: |
%s
- apiVersion: v1
  kind: ConfigMap
  metadata:
    annotations:
      service.beta.openshift.io/inject-cabundle: "true"
    labels:
      app: grpc-interop
    name: service-ca
- apiVersion: v1
  kind: Pod
  metadata:
    name: grpc-interop
    labels:
      app: grpc-interop
  spec:
    containers:
    - image: golang:1.14
      name: server
      command: ["/workdir/grpc-server"]
      env:
      - name: GRPC_GO_LOG_VERBOSITY_LEVEL
        value: "99"
      - name: GRPC_GO_LOG_SEVERITY_LEVEL
        value: "info"
      ports:
      - containerPort: 8443
        protocol: TCP
      - containerPort: 1110
        protocol: TCP
      volumeMounts:
      - name: service-certs
        mountPath: /etc/service-certs
      - name: tmp
        mountPath: /var/run
      - name: workdir
        mountPath: /workdir
      readOnly: true
    - image: golang:1.14
      name: client-shell
      command: ["/bin/bash"]
      args: ["-c", "sleep 100000"]
      volumeMounts:
      - name: service-certs
        secret:
          secretName: service-certs
        mountPath: /etc/service-certs
      - name: tmp
        mountPath: /var/run
      - name: workdir
        mountPath: /workdir
      - name: service-ca
        mountPath: /etc/service-ca
    initContainers:
    - image: golang:1.14
      name: builder
      command: ["/bin/bash", "-c"]
      args:
        - set -e;
          cd /workdir;
          base64 -d /go/src/data.base64 | tar zxf -;
          go build -v -mod=readonly -o /workdir/grpc-client client.go;
          go build -v -mod=readonly -o /workdir/grpc-server server.go;
      env:
      - name: GO111MODULE
        value: "auto"
      - name: GOCACHE
        value: "/tmp"
      - name: GOPROXY
        value: "https://goproxy.golang.org,direct"
      volumeMounts:
      - name: src-volume
        mountPath: /go/src
      - name: tmp
        mountPath: /var/run
      - name: workdir
        mountPath: /workdir
    volumes:
    - name: src-volume
      configMap:
        name: src-config
    - name: service-certs
      secret:
        secretName: service-certs
    - name: tmp
      emptyDir: {}
    - name: workdir
      emptyDir: {}
    - configMap:
        items:
        - key: service-ca.crt
          path: service-ca.crt
        name: service-ca
      name: service-ca
  labels:
    app: grpc-interop
- apiVersion: route.openshift.io/v1
  kind: Route
  metadata:
    annotations:
      haproxy.router.openshift.io/enable-h2c: "true"
    labels:
      app: grpc-interop
    name: grpc-interop-edge
  spec:
    port:
      targetPort: 1110
    tls:
      termination: edge
      insecureEdgeTerminationPolicy: Redirect
      key: |-
%s
      certificate: |-
%s
    to:
      kind: Service
      name: grpc-interop
      weight: 100
    wildcardPolicy: None
- apiVersion: route.openshift.io/v1
  kind: Route
  metadata:
    labels:
      app: grpc-interop
    name: grpc-interop-reencrypt
  spec:
    port:
      targetPort: 8443
    tls:
      termination: reencrypt
      insecureEdgeTerminationPolicy: Redirect
      key: |-
%s
      certificate: |-
%s
    to:
      kind: Service
      name: grpc-interop
      weight: 100
    wildcardPolicy: None
- apiVersion: route.openshift.io/v1
  kind: Route
  metadata:
    labels:
      app: grpc-interop
    name: grpc-interop-passthrough
  spec:
    port:
      targetPort: 8443
    tls:
      termination: passthrough
      insecureEdgeTerminationPolicy: Redirect
    to:
      kind: Service
      name: grpc-interop
      weight: 100
    wildcardPolicy: None
`,
		strings.Join(addPrefix(data, dataSpacer), "\n"),
		strings.Join(addPrefix(keyToString(edgeKey), tlsSpacer), "\n"),
		strings.Join(addPrefix(certToString(edgeCert), tlsSpacer), "\n"),
		strings.Join(addPrefix(keyToString(reencryptKey), tlsSpacer), "\n"),
		strings.Join(addPrefix(certToString(reencryptCert), tlsSpacer), "\n"))
}
