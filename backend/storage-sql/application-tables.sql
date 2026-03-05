DROP TABLE IF EXISTS public.diagram_history;
DROP TABLE IF EXISTS public.diagrams;
DROP TABLE IF EXISTS public.projects;
DROP TABLE IF EXISTS public.aws_account_access_roles;
DROP TABLE IF EXISTS public.organization_members;
DROP TABLE IF EXISTS public.organizations;
DROP TABLE IF EXISTS public.users;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================
-- USERS
-- ============================
CREATE TABLE public.users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    full_name VARCHAR(255) NOT NULL,
    picture_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================
-- ORGANIZATIONS
-- ============================
CREATE TABLE public.organizations (
    id UUID PRIMARY KEY,
    name VARCHAR(32) NOT NULL,
    description VARCHAR(300),
    created_by UUID NOT NULL REFERENCES public.users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_org_name_per_user UNIQUE (created_by, name)
);

CREATE TABLE public.organization_members (
    organization_id UUID NOT NULL REFERENCES public.organizations(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, member_id)
);

-- ============================
-- AWS ACCOUNT ACCESS ROLES
-- ============================
CREATE TABLE public.aws_account_access_roles (

    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES public.organizations(id) ON DELETE CASCADE,
    unique_id VARCHAR(8) NOT NULL,
    account_name VARCHAR(32) NOT NULL,
    role_arn TEXT,
    external_id VARCHAR(36) NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'incomplete',
    created_by UUID NOT NULL REFERENCES public.users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_role_name_per_org UNIQUE (organization_id, account_name),
    CONSTRAINT unique_unique_id_per_org UNIQUE (organization_id, unique_id),
    CONSTRAINT valid_status CHECK (status IN ('incomplete', 'complete', 'revoked'))
);

-- ============================
-- PROJECTS
-- ============================
CREATE TABLE public.projects (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES public.organizations(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES public.users(id),
    name VARCHAR(32) NOT NULL,
    description VARCHAR(300),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_project_name_per_org UNIQUE (organization_id, name)
);

-- ============================
-- DIAGRAMS
-- ============================
CREATE TABLE public.diagrams (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES public.projects(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES public.users(id),
    name VARCHAR(32) NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    latest_update_by UUID REFERENCES public.users(id),
    latest_update_at TIMESTAMPTZ,

    CONSTRAINT unique_diagram_name_per_project UNIQUE (project_id, name)
);

CREATE TABLE public.diagram_history (
    diagram_id UUID NOT NULL REFERENCES public.diagrams(id) ON DELETE CASCADE,
    version INT NOT NULL,
    updated_by UUID NOT NULL REFERENCES public.users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name VARCHAR(32) NOT NULL,
    data JSONB NOT NULL,
    comment VARCHAR(300),

    PRIMARY KEY (diagram_id, version)
);

-- ============================
-- ROW LEVEL SECURITY
-- ============================
ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."users"
AS PERMISSIVE
TO anon
USING (true);

ALTER TABLE public.organizations ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."organizations"
AS PERMISSIVE
TO anon
USING (true);

ALTER TABLE public.aws_account_access_roles ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."aws_account_access_roles"
AS PERMISSIVE
TO anon
USING (true);

ALTER TABLE public.organization_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."organization_members"
AS PERMISSIVE
TO anon
USING (true);

ALTER TABLE public.projects ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."projects"
AS PERMISSIVE TO anon USING (true);

ALTER TABLE public.diagrams ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."diagrams"
AS PERMISSIVE TO anon USING (true);

ALTER TABLE public.diagram_history ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."diagram_history"
AS PERMISSIVE TO anon USING (true);

-- ============================
-- JOBS
-- ============================
CREATE TABLE public.jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type VARCHAR(20) NOT NULL DEFAULT 'ansible' CHECK (job_type IN ('ansible', 'terraform')),
    status VARCHAR(50) NOT NULL DEFAULT 'QUEUED' CHECK (status IN ('QUEUED', 'STARTING', 'RUNNING', 'COMPLETED', 'FAILED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL REFERENCES public.users(id),

    -- Ansible-specific fields
    target_instance_ids JSONB,
    playbook_s3_key VARCHAR(255),

    -- Terraform-specific fields
    terraform_directory VARCHAR(255),
    role_arn VARCHAR(255),
    assume_role_external_id VARCHAR(36),

    -- Shared fields
    extra_vars JSONB DEFAULT '{}'::jsonb,
    task_arn VARCHAR(255),
    error_message TEXT,
    config_id UUID
);

CREATE INDEX idx_jobs_status ON public.jobs(status);
CREATE INDEX idx_jobs_created_at ON public.jobs(created_at DESC);
CREATE INDEX idx_jobs_created_by ON public.jobs(created_by);
CREATE INDEX idx_jobs_job_type ON public.jobs(job_type);

ALTER TABLE public.jobs ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."jobs"
AS PERMISSIVE TO anon USING (true);
