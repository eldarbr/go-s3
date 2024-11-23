BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE "bucket_availability_enum" AS ENUM (
  'closed',
  'accessible'
);

CREATE TABLE "buckets" (
  "id"           BIGSERIAL PRIMARY KEY,
  "name"         VARCHAR(30) NOT NULL UNIQUE,
  "owner_id"     UUID NOT NULL,
  "availability" "bucket_availability_enum" NOT NULL DEFAULT 'closed',
  "size_quota"   DECIMAL NOT NULL
);

-- a file can only exist in a bucket, and only in one bucket.
CREATE TABLE "files" (
  "id"           BIGSERIAL PRIMARY KEY,
  "filename"     TEXT,
  "mime"         TEXT,
  "created_ts"   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  "bucket_id"    BIGINT NOT NULL REFERENCES "buckets"("id") ON DELETE RESTRICT
);

COMMIT;
