package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const tableName = "application-data"

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	svc := dynamodb.NewFromConfig(cfg)

	// Sample IDs
	userID := "user-001"
	orgID := "org-001"
	projectID := "proj-001"
	diagramID1 := "diag-001"
	diagramID2 := "diag-002"
	now := time.Now().UTC().Format(time.RFC3339)

	items := []map[string]interface{}{
		// User
		{
			"PK":   fmt.Sprintf("USER#%s", userID),
			"SK":   "PROFILE",
			"Type": "USER",
			"Data": map[string]interface{}{
				"id":               userID,
				"name":             "Test User",
				"email":            "testuser@example.com",
				"accountCreatedOn": now,
			},
		},
		// Organization
		{
			"PK":   fmt.Sprintf("USER#%s", userID),
			"SK":   fmt.Sprintf("ORG#%s", orgID),
			"Type": "ORGANIZATION",
			"Data": map[string]interface{}{
				"id":          orgID,
				"name":        "Test Organization",
				"createdBy":   userID,
				"createdAt":   now,
				"description": "A sample organization for testing",
				"members":     []string{userID},
			},
		},
		// Project
		{
			"PK":   fmt.Sprintf("ORG#%s", orgID),
			"SK":   fmt.Sprintf("PROJECT#%s", projectID),
			"Type": "PROJECT",
			"Data": map[string]interface{}{
				"id":          projectID,
				"name":        "Sample Project",
				"description": "A sample project for testing diagrams",
				"createdBy":   userID,
				"createdAt":   now,
			},
		},
		// Diagram 1
		{
			"PK":   fmt.Sprintf("PROJECT#%s", projectID),
			"SK":   fmt.Sprintf("DIAGRAM#%s", diagramID1),
			"Type": "DIAGRAM",
			"Data": map[string]interface{}{
				"id":        diagramID1,
				"name":      "Main Infrastructure Diagram",
				"createdBy": userID,
				"createdAt": now,
			},
		},
		// Diagram 2
		{
			"PK":   fmt.Sprintf("PROJECT#%s", projectID),
			"SK":   fmt.Sprintf("DIAGRAM#%s", diagramID2),
			"Type": "DIAGRAM",
			"Data": map[string]interface{}{
				"id":        diagramID2,
				"name":      "Backup Architecture Diagram",
				"createdBy": userID,
				"createdAt": now,
			},
		},
	}

	fmt.Println("🌱 Seeding DynamoDB table:", tableName)
	fmt.Println()

	for _, item := range items {
		av, err := attributevalue.MarshalMap(item)
		if err != nil {
			log.Fatalf("Failed to marshal item: %v", err)
		}

		_, err = svc.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      av,
		})
		if err != nil {
			log.Fatalf("Failed to put item %v: %v", item["PK"], err)
		}

		fmt.Printf("✓ Seeded: PK=%s, SK=%s\n", item["PK"], item["SK"])
	}

	fmt.Println()
	fmt.Println("Seeding complete!")
	fmt.Println()
	fmt.Println("Test IDs:")
	fmt.Printf("  userID:     %s\n", userID)
	fmt.Printf("  orgID:      %s\n", orgID)
	fmt.Printf("  projectID:  %s\n", projectID)
	fmt.Printf("  diagramID1: %s\n", diagramID1)
	fmt.Printf("  diagramID2: %s\n", diagramID2)
}
