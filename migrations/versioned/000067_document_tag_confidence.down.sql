ALTER TABLE document_tags
    DROP CONSTRAINT IF EXISTS document_tags_confidence_check;
ALTER TABLE document_tags
    DROP COLUMN IF EXISTS confidence;
