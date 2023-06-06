package main

import (
	"TruFaaSClientApp/constants"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptoRand "crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func main() {

	// Port forwarding must be manually run --> kubectl port-forward svc/router 31314:80 -n fission

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Access the variables from the .env file
	noOfRuns := os.Getenv("NO_OF_RUNS")
	noOfFunctions := []int{10, 25, 50, 100, 250, 500, 750, 1000, 1500}
	treeResetURL := os.Getenv("TREE_RESET_URL")
	testedFnName := os.Getenv("FN_NAME")

	var createdFnNames []string
	n, _ := strconv.Atoi(noOfRuns)
	for j := 0; j < len(noOfFunctions); j++ {
		//create 'f-1' number of functions first

		if j > 0 {
			createdFnNames = append(createdFnNames, deployInitFunctions(noOfFunctions[j]-noOfFunctions[j-1])...)
		} else {
			createdFnNames = append(createdFnNames, deployInitFunctions(noOfFunctions[j])...)
		}
		for i := 0; i < n; i++ {

			fmt.Println("<<<<<<<<<<<<<<<< Run ", i+1, " >>>>>>>>>>>>>>>>>>>")

			//deploy 'f'th function
			deploymentTime := deployFunction(testedFnName)

			//deploy 'f'th function
			invocationTime := invokeFunction(testedFnName)
			deleteFunction(testedFnName)

			writeToCSV(noOfFunctions[j], i+1, deploymentTime, invocationTime)
		}
	}

	cleanUp(createdFnNames, treeResetURL)

}

func deployInitFunctions(f int) []string {
	var fnNames []string

	for j := 0; j < f-1; j++ {
		fnName := generateRandomString(5)
		cmd := exec.Command("fission", "fn", "create", "--name", fnName, "--env", "nodejs", "--code", "./functions/hello.js")
		cmd.Dir = "."
		err := cmd.Run()
		if err != nil {
			panic(err)
		}
		fnNames = append(fnNames, fnName)
	}
	fmt.Println(f-1, "dummy functions created")
	return fnNames
}

func deployFunction(fnName string) int64 {
	// Command 1: fission fn create --name test --env nodejs --code ./functions/hello.js
	cmd1 := exec.Command("fission", "fn", "create", "--name", fnName, "--env", "nodejs", "--code", "./functions/hello.js")
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	start := time.Now() // Record the start time
	err1 := cmd1.Run()
	elapsed := time.Since(start)
	if err1 != nil {
		panic(err1)
	}
	//Command 2: fission route create --name test --function test --url test
	cmd2 := exec.Command("fission", "route", "create", "--name", fnName, "--function", fnName, "--url", fnName)
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		panic(err2)
	}

	fmt.Println("function to test created.")
	return elapsed.Milliseconds()
}

func invokeFunction(fnName string) int64 {
	clientPrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), cryptoRand.Reader)
	clientPubKey := clientPrivKey.PublicKey
	clientPubKeyBytes := append(clientPubKey.X.Bytes(), clientPubKey.Y.Bytes()...)
	clientPubKeyHex := hex.EncodeToString(clientPubKeyBytes)

	// Command: curl http://localhost:31314/test
	start := time.Now() // Record the start time
	cmd := exec.Command("curl", "-H", constants.ClientPublicKeyHeader+":"+clientPubKeyHex, "http://localhost:31314/"+fnName)
	output, err := cmd.Output()
	elapsed := time.Since(start)
	if err != nil {
		fmt.Println("error invoking:", err)
	}
	fmt.Println("function invoked. result:", string(output))
	return elapsed.Milliseconds()
}

func cleanUp(createdFnNames []string, treeResetURL string) {

	//Delete all functions,routes and pkgs

	for _, value := range createdFnNames {
		// Command: fission fn delete --name value
		cmd := exec.Command("fission", "fn", "delete", "--name", value)
		cmd.Dir = "."
		err := cmd.Run()
		if err != nil {
			panic(err)
		}
	}

	// Command 3: fission pkg delete --orphan=true
	cmd3 := exec.Command("fission", "pkg", "delete", "--orphan=true")
	cmd3.Dir = "."
	err3 := cmd3.Run()
	if err3 != nil {
		panic(err3)
	}

	fmt.Println("Deleted all functions,routes and pkgs")

	// Clean the External Comp
	req, err := http.NewRequest("GET", treeResetURL, nil)
	if err != nil {
		panic(err)
	}
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	fmt.Println("API cleaned")
}

func deleteFunction(testedFnName string) {

	// Command 1: fission fn delete --name test
	cmd1 := exec.Command("fission", "fn", "delete", "--name", testedFnName)
	cmd1.Dir = "."
	err1 := cmd1.Run()
	if err1 != nil {
		panic(err1)
	}

	//Command 2: fission route delete --name test
	cmd2 := exec.Command("fission", "route", "delete", "--name", testedFnName)
	cmd2.Dir = "."
	err2 := cmd2.Run()
	if err2 != nil {
		panic(err2)
	}
}

func generateRandomString(length int) string {
	// Define the character set to use for the random string
	const charset = "abcdefghijklmnopqrstuvwxyz"

	// Seed the random number generator with the current time
	rand.Seed(time.Now().UnixNano())

	// Create a byte slice of the specified length
	randomBytes := make([]byte, length)

	// Fill the byte slice with random characters from the charset
	for i := 0; i < length; i++ {
		randomBytes[i] = charset[rand.Intn(len(charset))]
	}

	// Return the random string
	return string(randomBytes)
}

func writeToCSV(noOfFunctions int, runNumber int, depTime int64, invocTime int64) {
	// Check if the file "test-results.csv" exists.
	_, err := os.Stat("test-results.csv")
	var file *os.File
	var writer *csv.Writer
	if os.IsNotExist(err) {
		// The file does not exist, so create a new file.
		file, err = os.Create("test-results.csv")
		if err != nil {
			panic(err)
		}

		// Create a CSV writer.
		writer = csv.NewWriter(file)

		// Write the headers to the file.
		headers := []string{"functionCount", "Run Number", "deploymentTime", "invocationTime"}
		err = writer.Write(headers)
		if err != nil {
			panic(err)
		}
	} else {
		// The file already exists, so open the file in append mode.
		file, err = os.OpenFile("test-results.csv", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			panic(err)
		}

		// Create a CSV writer.
		writer = csv.NewWriter(file)
	}

	defer file.Close()

	// Write the data to the file.
	data := []string{strconv.Itoa(noOfFunctions), strconv.Itoa(runNumber), strconv.FormatInt(depTime, 10), strconv.FormatInt(invocTime, 10)}
	err = writer.Write(data)
	if err != nil {
		panic(err)
	}
	fmt.Println("results written to csv file")
	// Flush the data to the file.
	writer.Flush()
}
