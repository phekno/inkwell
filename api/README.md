# api

Go Lambda behind API Gateway HTTP API. Single binary, deployed as `provided.al2023` (custom runtime).

## Build

```sh
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap ./cmd/lambda
zip -q lambda.zip bootstrap
```

## Local invoke

The handler matches `events.APIGatewayV2HTTPRequest`; use `sam local` or invoke directly from a test.

## Routes

- `GET  /health`   — liveness
- `GET  /entries`  — list entries (TODO)
- `POST /entries`  — create entry (TODO)
