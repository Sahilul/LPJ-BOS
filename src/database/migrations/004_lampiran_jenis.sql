ALTER TABLE lampiran
ADD COLUMN jenis TEXT NOT NULL DEFAULT 'lainnya'
CHECK (jenis IN ('nota', 'bast', 'lainnya'));
