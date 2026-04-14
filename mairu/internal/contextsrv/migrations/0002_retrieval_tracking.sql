-- Add retrieval tracking columns to memories and bash_history

ALTER TABLE memories ADD COLUMN retrieval_count INT NOT NULL DEFAULT 0;
ALTER TABLE memories ADD COLUMN feedback_count INT NOT NULL DEFAULT 0;
ALTER TABLE memories ADD COLUMN last_retrieved_at DATETIME NULL;

ALTER TABLE bash_history ADD COLUMN importance INT NOT NULL DEFAULT 5;
ALTER TABLE bash_history ADD COLUMN feedback_count INT NOT NULL DEFAULT 0;
