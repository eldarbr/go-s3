BEGIN;

ALTER TABLE "files"
  ADD COLUMN "is_deleted" BOOL NOT NULL DEFAULT FALSE;

COMMIT;
