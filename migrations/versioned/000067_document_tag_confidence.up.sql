ALTER TABLE document_tags
    ADD COLUMN IF NOT EXISTS confidence DECIMAL(5,4) NOT NULL DEFAULT 0;

ALTER TABLE document_tags
    DROP CONSTRAINT IF EXISTS document_tags_confidence_check;
ALTER TABLE document_tags
    ADD CONSTRAINT document_tags_confidence_check CHECK (confidence >= 0 AND confidence <= 1);
