-- =============================================================================
-- Deployment Runs Table
-- Stores Terraform deployment execution history and logs
-- =============================================================================

-- Drop existing table if it exists (for development only)
-- DROP TABLE IF EXISTS public.deployment_runs;

CREATE TABLE IF NOT EXISTS public.deployment_runs (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES public.projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    log TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    
    -- Ensure status is one of the allowed values
    CONSTRAINT valid_status CHECK (status IN ('PENDING', 'RUNNING', 'SUCCESS', 'FAILED', 'CANCELLED'))
);

-- Index for querying deployments by project
CREATE INDEX IF NOT EXISTS idx_deployment_runs_project 
ON public.deployment_runs(project_id);

-- Index for querying deployments by status
CREATE INDEX IF NOT EXISTS idx_deployment_runs_status 
ON public.deployment_runs(status);

-- Index for querying deployments by user
CREATE INDEX IF NOT EXISTS idx_deployment_runs_user 
ON public.deployment_runs(user_id);

-- Index for querying recent deployments
CREATE INDEX IF NOT EXISTS idx_deployment_runs_created_at 
ON public.deployment_runs(created_at DESC);

-- Row Level Security
ALTER TABLE public.deployment_runs ENABLE ROW LEVEL SECURITY;

-- Policy for authenticated users to access deployment runs for projects they are members of
CREATE POLICY "project_members_access"
ON "public"."deployment_runs"
AS PERMISSIVE
FOR ALL
TO authenticated
USING (
  EXISTS (
    SELECT 1 FROM public.project_members pm
    WHERE pm.project_id = deployment_runs.project_id
      AND pm.user_id = auth.uid()
  )
)
WITH CHECK (
  user_id = auth.uid()
  AND EXISTS (
    SELECT 1 FROM public.project_members pm
    WHERE pm.project_id = deployment_runs.project_id
      AND pm.user_id = auth.uid()
  )
);

-- Policy for users to see their own triggered deployments
CREATE POLICY "users_own_deployments"
ON "public"."deployment_runs"
AS PERMISSIVE
FOR SELECT
TO authenticated
USING (user_id = auth.uid());

-- Comments for documentation
COMMENT ON TABLE public.deployment_runs IS 'Stores Terraform deployment execution history and logs';
COMMENT ON COLUMN public.deployment_runs.id IS 'Unique identifier for the deployment run (UUIDv4)';
COMMENT ON COLUMN public.deployment_runs.project_id IS 'Reference to the project being deployed';
COMMENT ON COLUMN public.deployment_runs.user_id IS 'User who triggered the deployment';
COMMENT ON COLUMN public.deployment_runs.status IS 'Current status: PENDING, RUNNING, SUCCESS, FAILED, CANCELLED';
COMMENT ON COLUMN public.deployment_runs.log IS 'Full Terraform execution log output';
COMMENT ON COLUMN public.deployment_runs.created_at IS 'Timestamp when deployment was triggered';
COMMENT ON COLUMN public.deployment_runs.completed_at IS 'Timestamp when deployment finished (success or failure)';
