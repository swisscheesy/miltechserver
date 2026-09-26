-- BLOCKED: no owner-approved, sanitized schema baseline has been supplied.
-- This is an intentional refusal artifact, not a reconstructed schema.
-- Replace only after recording provenance, schema comparison, PostgreSQL version,
-- migration boundary and checksum in docs/testing/shops-database.md.
DO $$ BEGIN
    RAISE EXCEPTION 'Owner-approved Shops schema baseline is unavailable';
END $$;
