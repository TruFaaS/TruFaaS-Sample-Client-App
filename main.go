package TruFaaSClientApp

import (
	"TruFaaSClientApp/constants"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
)

func main() {
	url := os.Args[3]
	fmt.Println("Invoking function at URL ", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	clientPrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Get the public key from the private key
	clientPubKey := clientPrivKey.PublicKey

	// Convert the client public key to hex
	clientPubKeyBytes := append(clientPubKey.X.Bytes(), clientPubKey.Y.Bytes()...)
	clientPubKeyHex := hex.EncodeToString(clientPubKeyBytes)

	req.Header.Set(constants.ClientPublicKeyHeader, clientPubKeyHex)

	// calling the function
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))

	// Get the server's public key
	serverPublicKeyHex := resp.Header.Get(constants.ServerPublicKeyHeader)

	//for key, values := range resp.Header {
	//	for _, value := range values {
	//		fmt.Printf("%s: %s\n", key, value)
	//	}
	//}

	// Get the MAC tag
	macTag := resp.Header.Get(constants.MacHeader)

	// Get the trust verification result
	trustVerificationTag := resp.Header.Get(constants.TrustVerificationHeader)

	functionName := resp.Header.Get(constants.FunctionNameHeader)

	if !verifyMacTag(serverPublicKeyHex, clientPrivKey, trustVerificationTag, functionName, macTag) {
		fmt.Println("MAC tag verification failed")
		return
	}

	fmt.Println("MAC tag verification succeeded")
	fmt.Println("Function invocation results are: ", body)
}

func verifyMacTag(serverPubKeyHex string, clientPrivateKey *ecdsa.PrivateKey, trustVerificationHeader string, functionName string, macTag string) bool {
	// Compute the shared secret using the client's private key and the server's public key
	serverPubKeyBytes, _ := hex.DecodeString(serverPubKeyHex)
	fmt.Println(serverPubKeyBytes)
	serverPubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(serverPubKeyBytes[:32]),
		Y:     new(big.Int).SetBytes(serverPubKeyBytes[32:]),
	}

	sharedSecret, _ := serverPubKey.Curve.ScalarMult(serverPubKey.X, serverPubKey.Y, clientPrivateKey.D.Bytes())
	fmt.Println(sharedSecret.Bytes())

	// Compute the MAC tag using the secret key and the header
	// TODO: add function name if it will be used
	h := hmac.New(sha256.New, sharedSecret.Bytes())
	h.Write([]byte(trustVerificationHeader))
	expectedMacTag := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Compare the computed MAC tag with the received MAC tag
	return macTag == expectedMacTag
}
