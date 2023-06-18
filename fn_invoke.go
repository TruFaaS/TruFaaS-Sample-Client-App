package main

import (
	"TruFaaSClientApp/constants"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
)

func main() {
	functionUrl := "http://localhost:31314/fnName"

	// Generate ECDSA private key
	invokerPrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Get the public key from the private key
	invokerPubKey := invokerPrivKey.PublicKey

	// Convert the client public key to hex
	invokerPubKeyBytes := append(invokerPubKey.X.Bytes(), invokerPubKey.Y.Bytes()...)
	invokerPubKeyHex := hex.EncodeToString(invokerPubKeyBytes)

	// Invoking the function at given URL
	req, err := http.NewRequest(http.MethodGet, functionUrl, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set(constants.InvokerPublicKeyHeader, invokerPubKeyHex)

	// Sending the request to Fission
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Failed to send request, error: ", err)
		return
	}

	defer resp.Body.Close()

	// Reading the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read the response body, error: ", err)
		return
	}

	// Accessing the response headers
	exCompPublicKeyHex := resp.Header.Get(constants.ExCompPublicKeyHeader)
	macTag := resp.Header.Get(constants.MacHeader)
	trustVerificationTag := resp.Header.Get(constants.TrustVerificationHeader)

	// Performing MAC tag verification
	macTagVerification := verifyMacTag(exCompPublicKeyHex, invokerPrivKey, trustVerificationTag, macTag)

	if !macTagVerification {
		fmt.Println("MAC tag verification failed")
		return
	}

	fmt.Println("MAC tag verification succeeded")
	fmt.Println("Function invocation result: ", string(body))
}

func verifyMacTag(exCompPubKeyHex string, invokerPrivateKey *ecdsa.PrivateKey, trustVerificationHeader string, macTag string) bool {
	// Compute the shared secret using the client's private key and the server's public key
	serverPubKeyBytes, _ := hex.DecodeString(exCompPubKeyHex)
	serverPubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(serverPubKeyBytes[:32]),
		Y:     new(big.Int).SetBytes(serverPubKeyBytes[32:]),
	}
	sharedSecret, _ := serverPubKey.Curve.ScalarMult(serverPubKey.X, serverPubKey.Y, invokerPrivateKey.D.Bytes())

	// Compute the MAC tag using the secret key and the header
	hMac := hmac.New(sha256.New, sharedSecret.Bytes())
	hMac.Write([]byte(trustVerificationHeader))
	expectedMacTag := hex.EncodeToString(hMac.Sum(nil))

	// Compare the computed MAC tag with the received MAC tag
	return macTag == expectedMacTag
}
