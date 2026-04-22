-- Cannot reverse this migration as it deletes data
-- Topics will need to be regenerated from articles
COMMENT ON TABLE topics IS 'Auto-detected topics are automatically cleaned up if not referenced by articles';
