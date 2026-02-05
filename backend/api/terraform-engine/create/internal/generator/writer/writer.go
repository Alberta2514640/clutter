package writer

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"
)

type TerraformWriter struct {
	s3Client   *s3.Client
	bucketName string
}

func NewTerraformWriter(ctx context.Context, bucketName string) (*TerraformWriter, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &TerraformWriter{
		s3Client:   s3.NewFromConfig(cfg),
		bucketName: bucketName,
	}, nil
}

// WriteToS3 uploads generated terraform files to S3
func (w *TerraformWriter) WriteToS3(ctx context.Context, tf *internal.GeneratedTerraform) error {
	files := map[string]string{
		"main.tf":      tf.MainTF,
		"resources.tf": tf.ResourcesTF,
		"iam.tf":       tf.IAMTF,
		"variables.tf": tf.VariablesTF,
		"outputs.tf":   tf.OutputsTF,
	}

	for filename, content := range files {
		if content == "" {
			continue
		}

		key := fmt.Sprintf("%s/%s", tf.DiagramID, filename)
		_, err := w.s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(w.bucketName),
			Key:         aws.String(key),
			Body:        bytes.NewReader([]byte(content)),
			ContentType: aws.String("text/plain"),
		})
		if err != nil {
			return fmt.Errorf("failed to upload %s: %w", filename, err)
		}
	}

	return nil
}

// GenerateMainTF creates the provider configuration
func GenerateMainTF() string {
	return `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  type        = string
  description = "AWS region"
  default     = "us-west-2"
}
`
}
