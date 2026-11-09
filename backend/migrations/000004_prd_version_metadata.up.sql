ALTER TABLE prd_versions ADD COLUMN title text NOT NULL DEFAULT '';
ALTER TABLE prd_versions ADD COLUMN status text NOT NULL DEFAULT 'draft';
UPDATE prd_versions v SET title=d.title,status=d.status FROM prds d WHERE d.id=v.prd_id;
