# GitHub Release Proxy

A reverse proxy for downloading GitHub releases.

### Compile

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/github-release-proxy ./cmd/github-release-proxy/main.go
```
