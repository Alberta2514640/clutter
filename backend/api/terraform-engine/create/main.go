package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context,request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Parse the webhook payload
	var payload internal.SupabaseWebhookPayload
	if err := json.Unmarshal([]byte(request.Body), &payload); err != nil {
		return generic.Response(http.StatusBadRequest, generic.Json{
			"error": "Invalid JSON payload",
			"details": err.Error(),
		})
	}

	if !internal.HasDataChanged(payload.OldRecord, payload.Record) {
		// No relevant changes, ignore
		return generic.Response(http.StatusOK, generic.Json{
			"message": "No relevant changes in diagram data, ignoring webhook",
		})
	}

	// Clean the nodes in the record (Remove irrelevant fields)
	payload.Record.Data.Nodes = internal.SanitizeNodes(payload.Record.Data.Nodes)

	// TODO: Use the payload.Record.Data to trigger your Terraform engine logic here

	// Return success response
	return generic.Response(http.StatusOK, generic.Json{
		"message": "New diagram data received for Terraform processing",
		"type":    payload.Type,
		"data":    payload.Record.Data,
	})
}