package writer

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/Alberta2514640/clutter/backend/api/terraform-engine/create/internal"
)

type TerraformWriter struct {
	s3Client   *s3.Client
	bucketName string
}

func NewTerraformWriter(ctx context.Context, bucketName string) (*TerraformWriter, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
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
	// Combine all into main.tf: provider + resources + IAM
	mainContent := tf.MainTF
	if tf.ResourcesTF != "" {
		mainContent += "\n" + tf.ResourcesTF
	}
	if tf.IAMTF != "" {
		mainContent += "\n" + tf.IAMTF
	}

	files := map[string]string{
		"main.tf":    mainContent,
		"outputs.tf": tf.OutputsTF,
	}

	for filename, content := range files {
		if content == "" {
			continue
		}

		if tf.OrgID == "" || tf.ProjectID == "" || tf.DiagramID == "" {
			return fmt.Errorf("missing org/project/diagram id for s3 path (org=%q project=%q diagram=%q)", tf.OrgID, tf.ProjectID, tf.DiagramID)
		}

		key := fmt.Sprintf("%s/%s/%s/%s", tf.OrgID, tf.ProjectID, tf.DiagramID, filename)
		_, err := w.s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:               aws.String(w.bucketName),
			Key:                  aws.String(key),
			Body:                 bytes.NewReader([]byte(content)),
			ContentType:          aws.String("text/plain"),
			ServerSideEncryption: types.ServerSideEncryptionAes256,
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
