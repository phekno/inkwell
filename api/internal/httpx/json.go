package httpx

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

func JSON(status int, body any) (events.APIGatewayV2HTTPResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Body:       `{"error":"marshal failed"}`,
			Headers:    map[string]string{"content-type": "application/json"},
		}, nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Body:       string(b),
		Headers:    map[string]string{"content-type": "application/json"},
	}, nil
}
