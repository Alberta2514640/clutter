-- ============================
-- B's TABLES: connections, deployments, placements, runs, function code
-- ============================
-- B (backend/adapter) applies this file at start-up, and A2 loads it after application-tables.sql.
-- Every statement is idempotent, so the file can run again on a database that already has it.
-- It depends on A's organizations, users and diagrams tables and on A's update_updated_at_column().

DO $$ BEGIN
    CREATE TYPE public.connection_status AS ENUM ('working', 'failed', 'deleting');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE public.deployment_status AS ENUM ('new', 'live', 'failed', 'destroyed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE public.run_action AS ENUM ('plan', 'apply', 'destroy');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE public.run_status AS ENUM ('queued', 'running', 'completed', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ============================
-- CONNECTIONS
-- ============================
-- One organization's linked cloud account. The credential itself lives in a secret store;
-- secret_ref is only where to find it. 'deleting' is a tombstone kept until the secret is gone.
CREATE TABLE IF NOT EXISTS public.connections (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES public.organizations(id) ON DELETE CASCADE,
    connector_type VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    region VARCHAR(64) NOT NULL,
    status public.connection_status NOT NULL,
    status_detail VARCHAR(300),
    secret_ref TEXT NOT NULL,
    role_template_version VARCHAR(32),
    created_by UUID REFERENCES public.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS connections_name_per_org
    ON public.connections (organization_id, name) WHERE status <> 'deleting';
CREATE INDEX IF NOT EXISTS connections_organization_id ON public.connections (organization_id);

-- ============================
-- DEPLOYMENTS
-- ============================
-- One diagram bound to one connection, with its own settings, excluded blocks (node ids),
-- Terraform state and runs.
CREATE TABLE IF NOT EXISTS public.deployments (
    id UUID PRIMARY KEY,
    diagram_id UUID NOT NULL REFERENCES public.diagrams(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES public.connections(id) ON DELETE CASCADE,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    exclusions TEXT[] NOT NULL DEFAULT '{}',
    status public.deployment_status NOT NULL DEFAULT 'new',
    created_by UUID REFERENCES public.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT deployments_one_per_diagram_connection UNIQUE (diagram_id, connection_id),
    CONSTRAINT deployments_settings_object CHECK (jsonb_typeof(settings) = 'object')
);

CREATE INDEX IF NOT EXISTS deployments_connection_id ON public.deployments (connection_id);

-- ============================
-- RUNS
-- ============================
-- One plan, apply or destroy of one deployment. diagram_version is the version of the saved
-- document the run was built from: one past the latest diagram_history row, the row that
-- document becomes on the next save (NULL for a destroy, which reads no diagram).
CREATE TABLE IF NOT EXISTS public.runs (
    id UUID PRIMARY KEY,
    deployment_id UUID NOT NULL REFERENCES public.deployments(id) ON DELETE CASCADE,
    action public.run_action NOT NULL,
    status public.run_status NOT NULL DEFAULT 'queued',
    diagram_version INT,
    failure VARCHAR(300),
    log_key VARCHAR(255) NOT NULL,
    created_by UUID REFERENCES public.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

-- At most one queued or running run per deployment.
CREATE UNIQUE INDEX IF NOT EXISTS runs_one_active_per_deployment
    ON public.runs (deployment_id) WHERE status IN ('queued', 'running');
CREATE INDEX IF NOT EXISTS runs_deployment_created ON public.runs (deployment_id, created_at DESC);

-- ============================
-- LIVE DEPLOYMENTS ARE NOT DELETED
-- ============================
-- A deployment that may hold resources in the user's cloud (live, failed, or with a run in
-- progress) cannot be deleted, directly or by a cascade from its connection, diagram, project or
-- organization. Destroy it first.

-- deployment_is_live is the one statement of that rule: the triggers below and B's store
-- (backend/adapter/internal/store) all call it.
CREATE OR REPLACE FUNCTION public.deployment_is_live(dep_id UUID, dep_status public.deployment_status)
RETURNS BOOLEAN AS $$
    SELECT dep_status IN ('live', 'failed')
        OR EXISTS (SELECT 1 FROM public.runs r
                   WHERE r.deployment_id = dep_id AND r.status IN ('queued', 'running'));
$$ LANGUAGE sql STABLE;

CREATE OR REPLACE FUNCTION public.refuse_live_deployment_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF public.deployment_is_live(OLD.id, OLD.status) THEN
        RAISE EXCEPTION 'deployment % is live; destroy it before deleting', OLD.id
            USING ERRCODE = 'restrict_violation';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER refuse_live_deployment_delete
    BEFORE DELETE ON public.deployments
    FOR EACH ROW
    EXECUTE FUNCTION public.refuse_live_deployment_delete();

CREATE OR REPLACE TRIGGER set_updated_at
    BEFORE UPDATE ON public.connections
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE TRIGGER set_updated_at
    BEFORE UPDATE ON public.deployments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================
-- FUNCTION CODE
-- ============================
-- The zip a user uploaded for one function block of a diagram, one per node. The zip itself is
-- in B's code store at object_key (backend/adapter/internal/codestore); every deployment of the diagram
-- uses it. Deleting the diagram or the organization removes the row, not the object; B's
-- code sweep (codestore.Sweeper) deletes an object no row names, under the same per-node lock
-- uploads take.
CREATE TABLE IF NOT EXISTS public.function_code (
    diagram_id UUID NOT NULL REFERENCES public.diagrams(id) ON DELETE CASCADE,
    node_id VARCHAR(64) NOT NULL,
    organization_id UUID NOT NULL REFERENCES public.organizations(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    size_bytes BIGINT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    uploaded_by UUID REFERENCES public.users(id) ON DELETE SET NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (diagram_id, node_id),
    CONSTRAINT function_code_size CHECK (size_bytes > 0 AND size_bytes <= 52428800),
    CONSTRAINT function_code_sha256 CHECK (sha256 ~ '^[0-9a-f]{64}$')
);

CREATE INDEX IF NOT EXISTS function_code_organization_id ON public.function_code (organization_id);
-- B asks whether any row names an object before it deletes one.
CREATE INDEX IF NOT EXISTS function_code_object_key ON public.function_code (object_key);

-- ============================
-- SPLIT DEPLOYMENTS
-- ============================
-- A deployment can place some of its blocks (node ids) on other connections of the same
-- organization, each with its own settings (ADR-020). Every other block stays on the
-- deployment's own connection. B checks that the connection belongs to the deployment's
-- organization when it writes a row. Each placed connection keeps its own Terraform state,
-- keyed by the deployment and the connection.
CREATE TABLE IF NOT EXISTS public.deployment_placements (
    deployment_id UUID NOT NULL REFERENCES public.deployments(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES public.connections(id) ON DELETE CASCADE,
    node_ids TEXT[] NOT NULL DEFAULT '{}',
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (deployment_id, connection_id),
    CONSTRAINT deployment_placements_settings_object CHECK (jsonb_typeof(settings) = 'object')
);

CREATE INDEX IF NOT EXISTS deployment_placements_connection_id ON public.deployment_placements (connection_id);

-- A placement of a deployment that may hold resources cannot go, so deleting its connection
-- (a cascade) cannot strand that connection's state. Deleting the deployment itself is
-- refused first by refuse_live_deployment_delete.
CREATE OR REPLACE FUNCTION public.refuse_live_placement_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM public.deployments d
               WHERE d.id = OLD.deployment_id AND public.deployment_is_live(d.id, d.status)) THEN
        RAISE EXCEPTION 'deployment % is live; destroy it before removing its placements', OLD.deployment_id
            USING ERRCODE = 'restrict_violation';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER refuse_live_placement_delete
    BEFORE DELETE ON public.deployment_placements
    FOR EACH ROW
    EXECUTE FUNCTION public.refuse_live_placement_delete();

-- Where each cloud of a split run stands: [{connection_id, name, step, status}], or NULL for a
-- run on one cloud.
ALTER TABLE public.runs ADD COLUMN IF NOT EXISTS parts JSONB;

-- ============================
-- ROW-LEVEL SECURITY
-- ============================
-- As for A's tables: B connects as the table owner, so these rules do not apply to it. With RLS
-- on and no policy, Supabase's anon and authenticated roles can read or change no row.
ALTER TABLE public.connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.deployments ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.function_code ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.deployment_placements ENABLE ROW LEVEL SECURITY;
