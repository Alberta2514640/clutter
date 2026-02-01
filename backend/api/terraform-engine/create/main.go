package main

import (
	"context"
	"net/http"
	"encoding/json"
	"log"

	"github.com/Alberta2514640/clutter/backend/api/generic"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type SupabaseWebhookPayload struct {
	Type   string                 `json:"type"`   // INSERT, UPDATE, DELETE
	Table  string                 `json:"table"`  // table name
	Schema string                 `json:"schema"` // usually "public"
	Record map[string]interface{} `json:"record"` // the new/updated record
	OldRecord map[string]interface{} `json:"old_record,omitempty"` // for UPDATE/DELETE
}

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context,request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
// Log the raw body for debugging
	log.Printf("Raw request body: %s", request.Body)
	log.Printf("Headers: %+v", request.Headers)

	// Check if body is empty
	if request.Body == "" {
		log.Println("Warning: Request body is empty")
		return generic.Response(http.StatusBadRequest, generic.Json{
			"error": "Empty request body",
		})
	}

	// Parse the webhook payload
	var payload SupabaseWebhookPayload
	if err := json.Unmarshal([]byte(request.Body), &payload); err != nil {
		log.Printf("Error parsing webhook payload: %v", err)
		return generic.Response(http.StatusBadRequest, generic.Json{
			"error": "Invalid JSON payload",
			"details": err.Error(),
		})
	}

	// Log the parsed payload
	log.Printf("Webhook type: %s, Table: %s", payload.Type, payload.Table)
	log.Printf("Record: %+v", payload.Record)

	// Process the webhook based on type
	switch payload.Type {
	case "INSERT":
		log.Printf("New diagram created: %+v", payload.Record)
	case "UPDATE":
		log.Printf("Diagram updated. Old: %+v, New: %+v", payload.OldRecord, payload.Record)
	case "DELETE":
		log.Printf("Diagram deleted: %+v", payload.OldRecord)
	}

	// Return success response
	return generic.Response(http.StatusOK, generic.Json{
		"message": "Diagram webhook received",
		"type":    payload.Type,
		"table":   payload.Table,
		"recordId": payload.Record["id"], // assuming your table has an id field
	})
}
