package generic

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

type Json map[string]any

func Response(status int, body Json, cookie ...string) (events.APIGatewayProxyResponse, error) {

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"message": "Internal server error [Response()]"}`,
		}, nil
	}

	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Headers": "Content-Type,Authorization",
		"Access-Control-Allow-Methods": "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}

	if len(cookie) > 0 {
		headers["Set-Cookie"] = cookie[0]
	}

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers:    headers,
		Body:       string(jsonBody),
	}, nil

}
