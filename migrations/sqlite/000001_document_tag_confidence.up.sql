ALTER TABLE document_tags
    ADD COLUMN confidence REAL NOT NULL DEFAULT 0 CHECK (confidence >= 0 AND confidence <= 1);
