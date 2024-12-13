BEGIN;

DROP INDEX "uk_files_filename_filename_suffix";

ALTER TABLE "files"
  DROP COLUMN "filename_suffix";

COMMIT;
