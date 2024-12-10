BEGIN;

ALTER TABLE "files"
  DROP COLUMN "is_deleted";

COMMIT;
