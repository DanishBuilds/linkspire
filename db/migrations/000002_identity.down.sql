-- db/migrations/000002_identity.down.sql
DROP TABLE IF EXISTS webauthn_credentials;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS workspaces;
