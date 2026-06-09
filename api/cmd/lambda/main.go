package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/phekno/inkwell/api/internal/httpx"
)

type handler struct {
	log *slog.Logger
}

func (h *handler) handle(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	route := req.RequestContext.HTTP.Method + " " + req.RequestContext.HTTP.Path
	h.log.InfoContext(ctx, "request", "route", route, "request_id", req.RequestContext.RequestID)

	switch route {
	case "GET /health":
		return httpx.JSON(200, map[string]string{"status": "ok"})
	case "GET /entries", "POST /entries":
		return httpx.JSON(501, map[string]string{"error": "not implemented"})
	default:
		return httpx.JSON(404, map[string]string{"error": "not found"})
	}
}

func main() {
	h := &handler{log: slog.New(slog.NewJSONHandler(os.Stdout, nil))}
	lambda.Start(h.handle)
}
