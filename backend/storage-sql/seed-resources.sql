-- ============================
-- SUPPORTED RESOURCES
-- ============================
CREATE TABLE IF NOT EXISTS public.supported_resources (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    label         VARCHAR(50)  NOT NULL UNIQUE, -- matches node data.label in diagram
    display_name  VARCHAR(50)  NOT NULL,
    description   VARCHAR(300),
    variables     JSONB        NOT NULL,        -- variable definitions (name, type, required, default)
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE public.supported_resources ENABLE ROW LEVEL SECURITY;
-- The resource catalog is not secret, so the Data API may read it; nobody may change it through the Data API.
-- The drops make the file safe to re-apply, and remove the old policy that let anon write the catalog.
DROP POLICY IF EXISTS "anon_all_permissions" ON "public"."supported_resources";
DROP POLICY IF EXISTS "catalog_read_only" ON "public"."supported_resources";
CREATE POLICY "catalog_read_only"
ON "public"."supported_resources"
AS PERMISSIVE FOR SELECT TO anon, authenticated USING (true);

-- ============================
-- SEED DATA
-- ============================

INSERT INTO public.supported_resources (label, display_name, description, variables) VALUES

('Lambda', 'AWS Lambda', 'Serverless function', '[
  {"name": "resource_name",        "type": "string",  "required": true,  "default": null,            "description": "Terraform resource ID and function name"},
  {"name": "handler",              "type": "string",  "required": false, "default": "main",          "description": "Lambda handler entrypoint"},
  {"name": "timeout",              "type": "number",  "required": false, "default": 3,               "description": "Timeout in seconds"},
  {"name": "runtime",              "type": "string",  "required": false, "default": "provided.al2",  "description": "Lambda runtime"},
  {"name": "architecture",         "type": "string",  "required": false, "default": "arm64",         "description": "CPU architecture"},
  {"name": "memory_size",          "type": "number",  "required": false, "default": 128,             "description": "Memory in MB"}
]'),

('S3', 'AWS S3', 'Object storage bucket', '[
  {"name": "resource_name",     "type": "string",  "required": true,  "default": null,  "description": "Terraform resource ID and bucket name (underscores converted to hyphens)"},
  {"name": "enable_versioning", "type": "boolean", "required": false, "default": false, "description": "Enable bucket versioning"},
  {"name": "block_public_access","type": "boolean","required": false, "default": true,  "description": "Block all public access"}
]'),

('DynamoDB', 'AWS DynamoDB', 'NoSQL key-value table', '[
  {"name": "resource_name",  "type": "string",  "required": true,  "default": null,              "description": "Terraform resource ID and table name"},
  {"name": "billing_mode",   "type": "string",  "required": false, "default": "PAY_PER_REQUEST", "description": "PAY_PER_REQUEST (on-demand) only"},
  {"name": "hash_key",       "type": "string",  "required": false, "default": "id",              "description": "Partition key attribute name"},
  {"name": "hash_key_type",  "type": "string",  "required": false, "default": "S",               "description": "Partition key type: S, N, or B"},
  {"name": "enable_ttl",     "type": "boolean", "required": false, "default": false,             "description": "Enable TTL"},
  {"name": "ttl_attribute",  "type": "string",  "required": false, "default": "ttl",             "description": "TTL attribute name (only used if enable_ttl is true)"}
]'),

('API Gateway', 'AWS API Gateway', 'REST API (v1) with CORS support', '[
  {"name": "resource_name", "type": "string",  "required": true,  "default": null,   "description": "Terraform resource ID and API name"},
  {"name": "description",   "type": "string",  "required": false, "default": null,   "description": "API description"},
  {"name": "stage_name",    "type": "string",  "required": false, "default": "v1",   "description": "Deployment stage name"},
  {"name": "enable_cors",   "type": "boolean", "required": false, "default": true,   "description": "Generate OPTIONS method and CORS headers"},
  {"name": "http_methods",  "type": "string",  "required": false, "default": "POST", "description": "Comma-separated methods for CORS Allow-Methods header (e.g. GET,POST,DELETE)"}
]'),

('EC2 Container', 'AWS EC2', 'Virtual machine instance', '[
  {"name": "resource_name",  "type": "string",  "required": true,  "default": null,          "description": "Terraform resource ID and instance Name tag"},
  {"name": "instance_type",  "type": "string",  "required": false, "default": "t3.micro",    "description": "EC2 instance type"},
  {"name": "key_name",       "type": "string",  "required": false, "default": null,           "description": "Key pair name for SSH access"}
]')
-- Re-applying the file brings an existing catalog row up to date instead of failing.
ON CONFLICT (label) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description  = EXCLUDED.description,
    variables    = EXCLUDED.variables;
