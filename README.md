# TruFaaS Client Application

This artifact contains the REST API for the TruFaaS demo application for function creation and function invocation.
This README will guid you through using this application locally.
Prior to using this application, ensure that you have installed the TruFaaS version of Fission 
and the TruFaaS external component.

### Prerequisites:
1. Build and deploy the TruFaaS external component on your local machine on port 8080.
2. Build and deploy the TruFaaS version of Fission.
3. Run the command ```kubectl port-forward svc/router 31314:80 -n fission``` on a terminal. Make sure that it continues to run while you use this application.
4. Create the Fission environment relevant to the programming language of the function.
   - For JS (which is used in the sample application), run ```fission env create --name nodejs --image fission/node-env``` in the terminal.

### Running the API
1. Open a terminal inside the source folder.
2. Run the command ```go run api.go```. The API will begin to run on port 8000. 
3. Set up the frontend of the demo application.
