package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var (
		outDir = flag.String("out", "/opt/dorevia-vault/keys", "Output directory for keys")
		kid    = flag.String("kid", "key-2025-Q1", "Key ID (kid) for JWKS")
		bits   = flag.Int("bits", 2048, "RSA key size in bits")
	)
	flag.Parse()

	// Créer le répertoire de sortie
	if err := os.MkdirAll(*outDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Générer la paire de clés RSA
	fmt.Printf("Generating RSA-%d key pair...\n", *bits)
	privateKey, err := rsa.GenerateKey(rand.Reader, *bits)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
		os.Exit(1)
	}

	// Encoder la clé privée en PEM
	privateKeyPath := filepath.Join(*outDir, "private.pem")
	privateKeyFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating private key file: %v\n", err)
		os.Exit(1)
	}
	defer privateKeyFile.Close()

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	if _, err := privateKeyFile.Write(privateKeyPEM); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing private key: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Private key written to: %s\n", privateKeyPath)

	// Encoder la clé publique en PEM
	publicKeyPath := filepath.Join(*outDir, "public.pem")
	publicKeyFile, err := os.OpenFile(publicKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating public key file: %v\n", err)
		os.Exit(1)
	}
	defer publicKeyFile.Close()

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling public key: %v\n", err)
		os.Exit(1)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	if _, err := publicKeyFile.Write(publicKeyPEM); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing public key: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Public key written to: %s\n", publicKeyPath)

	// Générer le JWKS
	jwksPath := filepath.Join(*outDir, "jwks.json")
	jwks, err := generateJWKS(&privateKey.PublicKey, *kid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JWKS: %v\n", err)
		os.Exit(1)
	}

	jwksFile, err := os.OpenFile(jwksPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating JWKS file: %v\n", err)
		os.Exit(1)
	}
	defer jwksFile.Close()

	if _, err := jwksFile.Write(jwks); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing JWKS: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ JWKS written to: %s\n", jwksPath)
	fmt.Printf("\n📝 Next steps:\n")
	fmt.Printf("  1. Set permissions: chmod 600 %s\n", privateKeyPath)
	fmt.Printf("  2. Configure environment:\n")
	fmt.Printf("     JWS_PRIVATE_KEY_PATH=%s\n", privateKeyPath)
	fmt.Printf("     JWS_PUBLIC_KEY_PATH=%s\n", publicKeyPath)
	fmt.Printf("     JWS_KID=%s\n", *kid)
}

// generateJWKS génère le JWKS pour une clé publique RSA
func generateJWKS(publicKey *rsa.PublicKey, kid string) ([]byte, error) {
	// Encoder n (modulus)
	nBytes := publicKey.N.Bytes()
	keySize := (publicKey.N.BitLen() + 7) / 8
	paddedN := make([]byte, keySize)
	copy(paddedN[keySize-len(nBytes):], nBytes)
	nBase64 := base64.RawURLEncoding.EncodeToString(paddedN)

	// Encoder e (exponent)
	eBytes := make([]byte, 4)
	e := publicKey.E
	for i := 3; i >= 0; i-- {
		eBytes[i] = byte(e)
		e >>= 8
	}
	// Supprimer les zéros de tête
	start := 0
	for start < len(eBytes) && eBytes[start] == 0 {
		start++
	}
	eBase64 := base64.RawURLEncoding.EncodeToString(eBytes[start:])

	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"kid": kid,
				"use": "sig",
				"alg": "RS256",
				"n":   nBase64,
				"e":   eBase64,
			},
		},
	}

	return json.MarshalIndent(jwks, "", "  ")
}

