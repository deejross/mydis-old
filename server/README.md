Cloud Foundry Deployment
------------------------
For deploying to Cloud Foundry, use the following commands from the `server` directory:
```bash
GOOS=linux GOARCH=amd64 go build server.go
cf push <app-name> -b binary_buildpack -c ./server --no-start
cf set-health-check <app-name> process
```

Using the `cf set-env` command, set the following environment variables:
```bash
GOPACKAGENAME mydis
PORT 8383
```

Start the app:
```bash
cf start <app-name>
```