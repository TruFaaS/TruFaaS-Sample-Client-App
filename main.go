package main

import (
	"TruFaaSClientApp/constants"
	"bytes"
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

	var jsonData = []byte(`{
    "function_information": {
        "function_name": "gs",
        "function_namespace": "default",
        "function_spec": {
            "environment": {
                "namespace": "default",
                "name": "nodejs"
            },
            "package_ref": {
                "namespace": "default",
                "name": "hello-js-73844e1f-92fc-4132-9d30-bbfdf57c17cb",
                "resource_version": "110732"
            },
            "invoke_strategy": {
                "execution_strategy": {
                    "executor-type": "poolmgr",
                    "min_scale": 0,
                    "max_scale": 10,
                    "target_cpu_percent": 0,
                    "specialization_timeout": 120
                },
                "strategy_type": "execution"
            },
            "function_timeout": 70,
            "idle_timeout": 120,
            "concurrency": 500,
            "requests_per_pod": 1
        }
    },
    "package_information": {
        "package_name": "hello-js-73844e1f-92fc-4132-9d30-bbfdf57c17cb",
        "package_namespace": "default",
        "package_spec": {
            "environment": {
                "namespace": "default",
                "name": "nodejs"
            },
            "source": {
                "checksum": {
                    "type": "",
                    "sum": ""
                }
            },
            "deployment": {
                "type": "literal",
                "literal": "Cm1vZHVsZS5leHBvcnRzID0gYXN5bmMgZnVuY3Rpb24oY29udGV4dCkgewogICAgcmV0dXJuIHsKICAgICAgICBzdGF0dXM6IDIwMCwKICAgICAgICBib2R5OiAiaGVsbG8sIHdvcmxkIVxuIgogICAgfTsKfQo=",
                "checksum": {
                    "type": "",
                    "sum": ""
                }
            }
        }
    }
}`)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
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

	// Accessing the headers
	// Get the server's public key
	serverPublicKeyHex := resp.Header.Get(constants.ServerPublicKeyHeader)
	// Get the MAC tag
	macTag := resp.Header.Get(constants.MacHeader)
	// Get the trust verification result
	trustVerificationTag := resp.Header.Get(constants.TrustVerificationHeader)

	//for key, values := range resp.Header {
	//	for _, value := range values {
	//		fmt.Printf("%s: %s\n", key, value)
	//	}
	//}

	if !verifyMacTag(serverPublicKeyHex, clientPrivKey, trustVerificationTag, macTag) {
		fmt.Println("MAC tag verification failed")
		return
	}

	fmt.Println("MAC tag verification succeeded")
	fmt.Println("Function invocation results are: ", body)
}

func verifyMacTag(serverPubKeyHex string, clientPrivateKey *ecdsa.PrivateKey, trustVerificationHeader string, macTag string) bool {
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
	hMac := hmac.New(sha256.New, sharedSecret.Bytes())
	hMac.Write([]byte(trustVerificationHeader))
	expectedMacTag := hex.EncodeToString(hMac.Sum(nil))

	// Compare the computed MAC tag with the received MAC tag
	return macTag == expectedMacTag
}
