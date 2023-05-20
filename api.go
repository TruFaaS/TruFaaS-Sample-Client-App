package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func main() {
	http.HandleFunc("/upload", uploadFileHandler)
	http.HandleFunc("/file", getFileHandler)

	fmt.Println("Server listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
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
	result := fnCreate(fnName, fileName)

	var data map[string]interface{}
	if result {
		data = map[string]interface{}{
			"fn_name": fnName,
			"result":  "Trust Created Successfully",
		}
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

func getFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the file from storage or generate it dynamically
	fileData := []byte("This is the file content")

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=file.txt")

	_, err := w.Write(fileData)
	if err != nil {
		http.Error(w, "Failed to send file", http.StatusInternalServerError)
		return
	}
}

func fnCreate(fnName string, fileName string) bool {

	// Command 1: fission fn create --name test --env nodejs --code sample_fn.js
	envCmdPy := exec.Command("fission env create --name python --image fission/python-env:latest --builder fission/python-builder:latest")
	envCmdPy.Dir = "."
	envCmdPy.Stdout = os.Stdout
	envCmdPy.Stderr = os.Stderr
	_ = envCmdPy.Run()

	envCmdJs := exec.Command("fission env create --name node --image fission/node-env")
	envCmdJs.Dir = "."
	envCmdJs.Stdout = os.Stdout
	envCmdJs.Stderr = os.Stderr
	_ = envCmdJs.Run()

	// Command 1: fission fn create --name test --env nodejs --code sample_fn.js
	cmd1 := exec.Command("fission", "fn", "create", "--name", fnName, "--env", "nodejs", "--code", fileName, "--idletimeout=1")
	cmd1.Dir = "."
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	err1 := cmd1.Run()

	if err1 != nil {
		print("Error creating function")
		return false
	}
	//Command 2: fission route create --name test --function test --url test
	cmd2 := exec.Command("fission", "route", "create", "--name", fnName, "--function", fnName, "--url", fnName)
	cmd2.Dir = "."
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		print("Error in calling command")
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

	return handler.Filename
}
