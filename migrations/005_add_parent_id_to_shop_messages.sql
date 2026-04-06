-- Add parent_id column to shop_messages for message reply support
ALTER TABLE shop_messages
    ADD COLUMN IF NOT EXISTS parent_id TEXT REFERENCES shop_messages(id) ON DELETE SET NULL;
