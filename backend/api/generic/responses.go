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
		"Content-Type": "application/json",
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
