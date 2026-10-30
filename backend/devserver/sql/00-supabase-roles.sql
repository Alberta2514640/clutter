-- Supabase creates these roles; plain Postgres does not. The schema in
-- backend/storage-sql is written for Supabase (seed-resources.sql grants a
-- policy to anon), so they must exist before it loads.
CREATE ROLE anon NOLOGIN;
CREATE ROLE authenticated NOLOGIN;
CREATE ROLE service_role NOLOGIN BYPASSRLS;
