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
	"os"
)

func main() {
	url := os.Args[1]
	fmt.Println("Invoking function at URL ", url)

	// Generate ECDSA private key
	clientPrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Get the public key from the private key
	clientPubKey := clientPrivKey.PublicKey

	// Convert the client public key to hex
	clientPubKeyBytes := append(clientPubKey.X.Bytes(), clientPubKey.Y.Bytes()...)
	clientPubKeyHex := hex.EncodeToString(clientPubKeyBytes)

	// Invoking the function at given URL
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set(constants.ClientPublicKeyHeader, clientPubKeyHex)

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

	// Accessing the headers
	// Get the server's public key
	serverPublicKeyHex := resp.Header.Get(constants.ServerPublicKeyHeader)
	// Get the MAC tag
	macTag := resp.Header.Get(constants.MacHeader)
	// Get the trust verification result
	trustVerificationTag := resp.Header.Get(constants.TrustVerificationHeader)

	if resp.StatusCode == http.StatusNotFound {
		fmt.Println("The function you are trying to invoke does not exist.")
		return
	}

	if serverPublicKeyHex == "" {
		fmt.Println(resp.Status)
		fmt.Println("Did not receive TruFaaS headers.")
		return
	}

	if !verifyMacTag1(serverPublicKeyHex, clientPrivKey, trustVerificationTag, macTag) {
		fmt.Println("MAC tag verification failed")
		return
	}

	fmt.Println("MAC tag verification succeeded")
	fmt.Println("[TruFaaS] Trust verification value received: ", trustVerificationTag)
	fmt.Println("Function invocation result: ", string(body))
}

func verifyMacTag1(serverPubKeyHex string, clientPrivateKey *ecdsa.PrivateKey, trustVerificationHeader string, macTag string) bool {
	// Compute the shared secret using the client's private key and the server's public key

	serverPubKeyBytes, _ := hex.DecodeString(serverPubKeyHex)
	serverPubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(serverPubKeyBytes[:32]),
		Y:     new(big.Int).SetBytes(serverPubKeyBytes[32:]),
	}
	sharedSecret, _ := serverPubKey.Curve.ScalarMult(serverPubKey.X, serverPubKey.Y, clientPrivateKey.D.Bytes())

	// Compute the MAC tag using the secret key and the header
	hMac := hmac.New(sha256.New, sharedSecret.Bytes())
	hMac.Write([]byte(trustVerificationHeader))
	expectedMacTag := hex.EncodeToString(hMac.Sum(nil))

	// Compare the computed MAC tag with the received MAC tag
	return macTag == expectedMacTag
}
