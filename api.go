package main

import (
	"TruFaaSClientApp/constants"
	"TruFaaSClientApp/utils"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/cors"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type functionInvocationResults struct {
	error        string
	fnInvResults string
	fnName       string
}

func main() {
	corsHandler := cors.AllowAll()

	// Create a new HTTP server and wrap the default handler with the CORS middleware
	server := http.Server{
		Addr:    ":8000",
		Handler: corsHandler.Handler(http.DefaultServeMux),
	}
	http.HandleFunc("/create", clientDeployFunction)
	http.HandleFunc("/invoke/", clientVerifyFunction)
	http.HandleFunc("/generate", clientGenerateKeys)
	fmt.Println("Server listening on port 8000...")
	log.Fatal(server.ListenAndServe())
}

func clientDeployFunction(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20) // Max memory to allocate for form data
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusInternalServerError)
		return
	}

	fileName := getFileAndSaveLocally(r, w)

	fnName := r.FormValue("fn_name")
	env := r.FormValue("env")
	err = fnCreate(fnName, fileName, env)

	var data map[string]interface{}
	if err == nil {
		data = map[string]interface{}{
			"fn_name": fnName,
			"result":  "TruFaaS Trust Value Created Successfully!!\n Function Created Successfully!!",
		}
	} else {
		data = map[string]interface{}{
			"fn_name": fnName,
			"result":  "OOPS! " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}

	// Set the appropriate headers
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response
	w.Write(jsonData)
}

func clientVerifyFunction(w http.ResponseWriter, r *http.Request) {
	errResponse := utils.ErrorResponse{}
	var data map[string]interface{}
	if r.Method != http.MethodGet {
		errResponse.StatusCode = http.StatusBadRequest
		errResponse.ErrorMsg = "Incorrect HTTP Method"
		sendErrorResponse(w, errResponse)
		return
	}

	//fnName := r.URL.Path[len("/invoke/"):]
	fnName := strings.TrimPrefix(r.URL.Path, "/invoke/")

	url := "http://localhost:31314/" + fnName

	clientPrivKeyHex := r.Header.Get("private_key")
	clientPubKeyHex := r.Header.Get("public_key")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		errResponse.StatusCode = http.StatusBadRequest
		errResponse.ErrorMsg = err.Error()
		sendErrorResponse(w, errResponse)
		return
	}

	if clientPrivKeyHex != "" && clientPubKeyHex != "" {
		req.Header.Set(constants.ClientPublicKeyHeader, clientPubKeyHex)
	}

	// Sending the request to Fission
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		errResponse.StatusCode = http.StatusInternalServerError
		errResponse.ErrorMsg = err.Error()
		sendErrorResponse(w, errResponse)
		return
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			errResponse.StatusCode = http.StatusInternalServerError
			errResponse.ErrorMsg = err.Error()
			sendErrorResponse(w, errResponse)
			return
		}
	}(resp.Body)

	// Reading the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read the response body, error: ", err)
		errResponse.StatusCode = http.StatusInternalServerError
		errResponse.ErrorMsg = err.Error()
		sendErrorResponse(w, errResponse)
		return
	}

	fmt.Println(resp.Header)

	// if trust headers are sent
	if clientPrivKeyHex != "" && clientPubKeyHex != "" {
		// Accessing the headers
		// Get the server's public key
		serverPublicKeyHex := resp.Header.Get(constants.ServerPublicKeyHeader)
		// Get the MAC tag
		macTag := resp.Header.Get(constants.MacHeader)
		// Get the trust verification result
		trustVerificationTag := resp.Header.Get(constants.TrustVerificationHeader)

		if resp.StatusCode == http.StatusNotFound {
			fmt.Println("The function you are trying to invoke does not exist.")
			errResponse.StatusCode = http.StatusNotFound
			errResponse.ErrorMsg = "The function you are trying to invoke does not exist."
			sendErrorResponse(w, errResponse)
			return
		}

		if serverPublicKeyHex == "" {
			fmt.Println("Did not receive TruFaaS headers.")
			errResponse.StatusCode = http.StatusInternalServerError
			errResponse.ErrorMsg = "Did not receive TruFaaS headers."
			sendErrorResponse(w, errResponse)
			return
		}

		data = map[string]interface{}{
			"fn_name":            fnName,
			"result":             string(body),
			"mac_verification":   verifyMacTag(serverPublicKeyHex, clientPrivKeyHex, trustVerificationTag, macTag),
			"trust_verification": trustVerificationTag,
			"mac_tag":            macTag,
			"ex_comp_public_key": serverPublicKeyHex,
		}

		fmt.Println("MAC tag verification succeeded")
		fmt.Println("[TruFaaS] Trust verification value received: ", trustVerificationTag)
		fmt.Println("Function invocation result: ", string(body))
	} else {
		data = map[string]interface{}{
			"fn_name": fnName,
			"result":  string(body),
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		errResponse.StatusCode = http.StatusBadRequest
		errResponse.ErrorMsg = err.Error()
		sendErrorResponse(w, errResponse)
		return
	}

	// Set the appropriate headers
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response
	w.Write(jsonData)
}

func clientGenerateKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clientPrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	clientPrivKeyBytes := clientPrivKey.D.Bytes()
	clientPrivKeyHex := hex.EncodeToString(clientPrivKeyBytes)
	// Get the public key from the private key
	clientPubKey := clientPrivKey.PublicKey
	fmt.Println("Generated Key Pair.")

	// Convert the client public key to hex
	clientPubKeyBytes := append(clientPubKey.X.Bytes(), clientPubKey.Y.Bytes()...)
	clientPubKeyHex := hex.EncodeToString(clientPubKeyBytes)

	data := map[string]interface{}{
		"private_key": clientPrivKeyHex,
		"public_key":  clientPubKeyHex,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}
	// Set the appropriate headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Write the JSON response
	w.Write(jsonData)
}

func fnCreate(fnName string, fileName string, env string) error {

	// Command 1: fission fn create --name test --env nodejs --code sample_fn.js
	cmd1 := exec.Command("fission", "fn", "create", "--name", fnName, "--env", env, "--code", fileName, "--idletimeout=1")
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err1 := cmd1.Run()

	if err1 != nil {
		return errors.New("Failed to create function. " + err1.Error())
		//return errors.New(err1.Error())
	}

	//Command 2: fission route create --name test --function test --url test
	cmd2 := exec.Command("fission", "route", "create", "--name", fnName, "--function", fnName, "--url", fnName)
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		return errors.New("Failed to create route.\n " + err2.Error())
	}
	return nil
}

func verifyMacTag(serverPubKeyHex string, clientPrivateKeyHex string, trustVerificationHeader string, macTag string) bool {
	// Compute the shared secret using the client's private key and the server's public key

	// TODO: need to check
	clientPrivKeyBytes, _ := hex.DecodeString(clientPrivateKeyHex)
	clientPrivateKey := new(ecdsa.PrivateKey)
	clientPrivateKey.Curve = elliptic.P256() // Replace with the appropriate elliptic curve if needed
	clientPrivateKey.D = new(big.Int).SetBytes(clientPrivKeyBytes)
	fmt.Println(clientPrivKeyBytes)

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

func cleanUp(fnName string) bool {

	// Command 1: fission fn create --name test --env nodejs --code sample_fn.js
	cmd1 := exec.Command("fission", "fn", "delete", "--name", fnName)
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err1 := cmd1.Run()

	if err1 != nil {
		print("Error deleting function")
		return false
	}
	//Command 2: fission route create --name test --function test --url test
	cmd2 := exec.Command("fission", "httptrigger", "delete", "--name", fnName)
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		print("Error in deleting trigger")
		return false
	}
	return true
}

func getFileAndSaveLocally(r *http.Request, w http.ResponseWriter) string {

	file, handler, err := r.FormFile("code")
	if err != nil {
		print("error reading file from http request")
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return ""
	}
	defer file.Close()

	localFile, err := os.Create(handler.Filename)
	if err != nil {
		print("error creating file")
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return ""
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, file)
	if err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return ""
	}

	return handler.Filename
}

func createEnvs() {

	// Command 1: fission fn create --name test --env nodejs --code sample_fn.js
	envCmdPy := exec.Command("fission env create --name python --image fission/python-env:latest --builder fission/python-builder:latest")
	envCmdPy.Dir = "."
	envCmdPy.Stdout = os.Stdout
	envCmdPy.Stderr = os.Stderr
	err := envCmdPy.Run()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("===========================")
	envCmdJs := exec.Command("fission env create --name node --image fission/node-env")
	envCmdJs.Dir = "."
	envCmdJs.Stdout = os.Stdout
	envCmdJs.Stderr = os.Stderr
	_ = envCmdJs.Run()
}

func sendErrorResponse(respWriter http.ResponseWriter, body utils.ErrorResponse) {
	fmt.Println(body)
	jsonResponse, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("failed to marshal body, Error:%s", err)
		return
	}
	respWriter.Header().Set("Content-Type", constants.ContentTypeJSON)
	respWriter.WriteHeader(body.StatusCode)
	_, err = respWriter.Write(jsonResponse)
	if err != nil {
		fmt.Printf("failed to marshal body, Error:%s", err)
		return
	}
}
