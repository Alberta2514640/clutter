DROP TABLE IF EXISTS public.diagram_history;
DROP TABLE IF EXISTS public.diagrams;
DROP TABLE IF EXISTS public.projects;
DROP TABLE IF EXISTS public.organization_members;
DROP TABLE IF EXISTS public.organizations;
DROP TABLE IF EXISTS public.users;
DROP TABLE IF EXISTS public.resource_connections;
DROP TABLE IF EXISTS public.resource;
DROP TABLE IF EXISTS public.resource_types;

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
-- Resource
-- ============================

CREATE TABLE public.resource_types (
    resource_type VARCHAR(50) PRIMARY KEY,
    description TEXT,
    category VARCHAR(50)
);

-- This table stores individual resource with their configurations
CREATE TABLE public.resource (
    resource_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type VARCHAR(50) NOT NULL REFERENCES public.resource_types(resource_type),
    platform VARCHAR(50) NOT NULL,
    resource_version VARCHAR(20) NOT NULL,
    variables JSONB NOT NULL,
    snippet TEXT NOT NULL
);

-- This table defines possible connections between different resource types
CREATE TABLE public.resource_connections (
    connection_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_version VARCHAR(20) NOT NULL,
    source_resource_type VARCHAR(50) NOT NULL REFERENCES public.resource_types(resource_type) ON DELETE CASCADE,
    target_resource_type VARCHAR(50) NOT NULL REFERENCES public.resource_types(resource_type) ON DELETE CASCADE,
    connection_template TEXT, -- Used to store a template or pattern for the connection
    is_bidirectional BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_resource_type, target_resource_type)
);


-- ============================
-- Resource
-- ============================

CREATE TABLE public.resource_types (
    resource_type VARCHAR(50) PRIMARY KEY,
    description TEXT,
    category VARCHAR(50)
);

-- This table stores individual resource with their configurations
CREATE TABLE public.resource (
    resource_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type VARCHAR(50) NOT NULL REFERENCES public.resource_types(resource_type),
    platform VARCHAR(50) NOT NULL,
    resource_version VARCHAR(20) NOT NULL,
    variables JSONB NOT NULL,
    snippet TEXT NOT NULL
);

-- This table defines possible connections between different resource types
CREATE TABLE public.resource_connections (
    connection_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_version VARCHAR(20) NOT NULL,
    source_resource_type VARCHAR(50) NOT NULL REFERENCES public.resource_types(resource_type) ON DELETE CASCADE,
    target_resource_type VARCHAR(50) NOT NULL REFERENCES public.resource_types(resource_type) ON DELETE CASCADE,
    connection_template TEXT, -- Used to store a template or pattern for the connection
    is_bidirectional BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_resource_type, target_resource_type)
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

ALTER TABLE public.resource_types ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."resource_types"
AS PERMISSIVE TO anon USING (true);

ALTER TABLE public.resource ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."resource"
AS PERMISSIVE TO anon USING (true);

ALTER TABLE public.resource_connections ENABLE ROW LEVEL SECURITY;
CREATE POLICY "anon_all_permissions"
ON "public"."resource_connections"
AS PERMISSIVE TO anon USING (true);
