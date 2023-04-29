package main

import (
	"TruFaaSClientApp/constants"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/joho/godotenv"
	"log"
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
	fnEnv := os.Getenv("FN_ENV")
	fnName := os.Getenv("FN_NAME")
	fnFile := os.Getenv("FN_FILE")
	fnFileType := os.Getenv("FN_FILE_TYPE")
	noOfRuns := os.Getenv("NO_OF_RUNS")

	n, _ := strconv.Atoi(noOfRuns)
	for i := 0; i < n; i++ {
		// loop body
		deploymentTime := deployFunction(fnName, fnEnv, fnFile, fnFileType)
		invocationTime := invokeFunction(fnName)
		deleteFunction(fnName)

		writeToCSV(fnFile, deploymentTime, invocationTime)
		fmt.Println(deploymentTime, invocationTime)
	}

}

func deployFunction(fnName string, fnEnv string, fnFile string, fnFileType string) int64 {
	// Command 1: fission fn create --name test --env nodejs --code hello.js
	cmd1 := exec.Command("fission", "fn", "create", "--name", fnName, "--env", fnEnv, fnFileType, fnFile)
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	start := time.Now() // Record the start time
	err1 := cmd1.Run()
	elapsed := time.Since(start)
	if err1 != nil {
		panic(err1)
	}

	// Command 2: fission route create --name test --function test --url test
	cmd2 := exec.Command("fission", "route", "create", "--name", fnName, "--function", fnName, "--url", fnName)
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		panic(err2)
	}

	return elapsed.Microseconds()
}

func invokeFunction(fnName string) int64 {

	clientPrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	clientPubKey := clientPrivKey.PublicKey
	clientPubKeyBytes := append(clientPubKey.X.Bytes(), clientPubKey.Y.Bytes()...)
	clientPubKeyHex := hex.EncodeToString(clientPubKeyBytes)

	// Command: curl http://localhost:31314/test
	start := time.Now() // Record the start time
	cmd := exec.Command("curl", "-H", constants.ClientPublicKeyHeader+":"+clientPubKeyHex, "http://localhost:31314/"+fnName)
	elapsed := time.Since(start)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("error invoking:", err)
	}
	fmt.Println("function invoked. result:", string(output))
	return elapsed.Microseconds()
}

func deleteFunction(fnName string) {
	// Command 1: fission fn delete --name test
	cmd1 := exec.Command("fission", "fn", "delete", "--name", fnName)
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err1 := cmd1.Run()
	if err1 != nil {
		panic(err1)
	}

	// Command 2: fission route delete --name test
	cmd2 := exec.Command("fission", "route", "delete", "--name", fnName)
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		panic(err2)
	}

	// Command 2: fission pkg delete --orphan=true
	cmd3 := exec.Command("fission", "pkg", "delete", "--orphan=true")
	cmd3.Dir = "."
	cmd3.Stdout = os.Stdout
	cmd3.Stderr = os.Stderr
	err3 := cmd3.Run()
	if err3 != nil {
		panic(err2)
	}
}

func writeToCSV(fnFile string, depTime int64, invocTime int64) {
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
		headers := []string{"fileName", "deploymentTime", "invocationTime"}
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
	data := []string{fnFile, strconv.FormatInt(depTime, 10), strconv.FormatInt(invocTime, 10)}
	err = writer.Write(data)
	if err != nil {
		panic(err)
	}

	// Flush the data to the file.
	writer.Flush()
}
