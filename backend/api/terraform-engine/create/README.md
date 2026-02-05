# Terraform Engine - Create

This Lambda function generates Terraform code from diagram data received via Supabase webhooks.

## Architecture Overview

```
Supabase Webhook → API Gateway → Lambda → Generate Terraform → S3 Output Bucket
                                              ↓
                                    S3 Templates Bucket
```

## How It Works

### 1. Webhook Trigger
When a diagram is created/updated in Supabase, a webhook sends a POST request with:
```json
{
  "type": "UPDATE",
  "record": {
    "id": "diagram-uuid",
    "project_id": "project-uuid",
    "data": {
      "nodes": [...],
      "edges": [...]
    }
  }
}
```

### 2. Node Processing
Each node in the diagram is parsed:
- `data.label` → Determines resource type (Lambda, S3, DynamoDB, API-Gateway)
- `variables.resource_name` → Used for Terraform resource naming and AWS resource names
- `variables.*` → Mapped to template placeholders

### 3. Template Loading
Templates are fetched from S3 and cached in memory:
```
s3://clutter-templates-us-west-2-b35a3c5c/
└── templates/aws/
    ├── lambda/
    │   ├── main.tf.tmpl
    │   └── outputs.tf.tmpl
    └── s3/
        ├── main.tf.tmpl
        └── outputs.tf.tmpl
```

### 4. Template Rendering
Go's `text/template` package renders templates the variables passed into for each Node in the Diagram Layout:
```
{{.ResourceName}} → my_test_function
{{.FunctionName}} → my-test-function
{{.Timeout}}      → 30
```

### 5. IAM Generation
Edges between nodes generate IAM policies automatically:
- Lambda → S3 edge creates S3 read/write permissions

### 6. S3 Output
Generated Terraform is uploaded to:
```
s3://clutter-us-west-2-..../
└── {org-id}/
    └── {project-id}/
        └── {diagram-id}/
            ├── main.tf      (provider + resources + IAM)
            └── outputs.tf   (resource outputs)
```

---

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `TEMPLATE_BUCKET_NAME` | S3 bucket for templates | `clutter-templates-us-west-2-...` |
| `S3_BUCKET_NAME` | S3 bucket for output | `clutter-us-west-2-...` |
| `PSQL_CONNECTION_STRING` | Postgres connection string for org lookup | `postgres://...` |

---

## Local Testing

### Run Tests

**Full end-to-end test (generates + uploads to S3):**
```bash
cd backend/api/terraform-engine/create
go test -v -run TestHandler
```

**Generation only (no S3 upload):**
```bash
cd backend/api/terraform-engine/create
go test -v -run TestGenerateOnly
```

### Verify S3 Output
```bash
# List generated files
aws s3 ls s3://clutter-us-west-2-b35a3c5c/test-org-123/test-project-456/test-diagram-123/

# View main.tf content
aws s3 cp s3://clutter-us-west-2-b35a3c5c/test-org-123/test-project-456/test-diagram-123/main.tf -

# View outputs.tf content
aws s3 cp s3://clutter-us-west-2-b35a3c5c/test-org-123/test-project-456/test-diagram-123/outputs.tf -
```

---

## Template Files

### Lambda Template (`templates/aws/lambda/main.tf.tmpl`)
```hcl
resource "aws_iam_role" "{{.ResourceName}}_role" {
  name = "{{.FunctionName}}-lambda-role"
  ...
}

resource "aws_lambda_function" "{{.ResourceName}}" {
  function_name = "{{.FunctionName}}"
  handler       = "{{.Handler}}"
  timeout       = {{.Timeout}}
  runtime       = "{{.Runtime}}"
  ...
}
```

### S3 Template (`templates/aws/s3/main.tf.tmpl`)
```hcl
resource "aws_s3_bucket" "{{.ResourceName}}" {
  bucket = "{{.BucketName}}"
}
```

### Upload Templates to S3
```bash
cd backend/api/terraform-engine/create
aws s3 cp templates/aws/ s3://clutter-templates-us-west-2-b35a3c5c/templates/aws/ --recursive
```

### List Templates in S3
```bash
aws s3 ls s3://clutter-templates-us-west-2-b35a3c5c/templates/aws/ --recursive
```

---

## Variable Mapping

### Lambda Node
| Diagram Variable | Template Placeholder | Default |
|-----------------|---------------------|---------|
| `resource_name` | `{{.ResourceName}}`, `{{.FunctionName}}` | Required |
| `handler` | `{{.Handler}}` | `main` |
| `timeout` | `{{.Timeout}}` | `3` |
| `runtime` | `{{.Runtime}}` | `provided.al2` |
| `architecture` | `{{.Architecture}}` | `arm64` |
| `memory_size` | `{{.MemorySize}}` | `128` |

### S3 Node
| Diagram Variable | Template Placeholder | Default |
|-----------------|---------------------|---------|
| `resource_name` | `{{.ResourceName}}`, `{{.BucketName}}` | Required |
| `enable_versioning` | `{{.EnableVersioning}}` | `false` |
| `block_public_access` | `{{.BlockPublicAccess}}` | `true` |

---

## Code Structure

```
terraform-engine/create/
├── main.go                          # Lambda handler entry point
├── main_test.go                     # Test file
├── templates/aws/                   # Local template files (upload to S3)
│   ├── lambda/
│   │   ├── main.tf.tmpl
│   │   └── outputs.tf.tmpl
│   └── s3/
│       ├── main.tf.tmpl
│       └── outputs.tf.tmpl
└── internal/
    ├── structs.go                   # Data types
    ├── data_clean.go                # Node sanitization
    └── generator/
        ├── generator.go             # Main generation logic
        ├── template/
        │   ├── loader.go            # S3 template fetching + caching
        │   ├── renderer.go          # Go text/template rendering
        │   └── config.go            # Template path helpers
        ├── resources/
        │   ├── mapper.go            # Resource type → Generator mapping
        │   ├── lambda.go            # Lambda generator
        │   └── s3.go                # S3 generator
        ├── iam/
        │   └── policy.go            # IAM policy generation from edges
        └── writer/
            └── writer.go            # S3 upload with encryption
```

---

## Troubleshooting

### "NoSuchKey" Error
Template file doesn't exist in S3. Upload templates:
```bash
aws s3 cp templates/aws/ s3://clutter-templates-us-west-2-b35a3c5c/templates/aws/ --recursive

```
