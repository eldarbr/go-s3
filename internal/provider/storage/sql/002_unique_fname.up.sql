BEGIN;

ALTER TABLE "files"
  ADD COLUMN "filename_suffix" INT NOT NULL DEFAULT 0;

-- assign proper suffixes to the existing files.
WITH existing_suffixes AS (
  SELECT
    "id",
    (ROW_NUMBER() OVER(PARTITION BY "filename") - 1) AS "suffix"
  FROM "files"
  ORDER BY "created_ts" ASC
)
UPDATE "files"
SET "filename_suffix" = "existing_suffixes"."suffix"
FROM "existing_suffixes"
WHERE "files"."id" = "existing_suffixes"."id";

CREATE UNIQUE INDEX "uk_files_filename_filename_suffix"
  ON "files"("filename", "filename_suffix");

COMMIT;
