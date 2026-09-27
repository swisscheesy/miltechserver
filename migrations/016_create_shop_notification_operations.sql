BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE TABLE public.shop_notification_operations (
    user_id text NOT NULL REFERENCES public.users(uid) ON DELETE CASCADE,
    operation_id uuid NOT NULL,
    fingerprint bytea NOT NULL CHECK (octet_length(fingerprint) = 32),
    notification_id uuid NOT NULL,
    committed_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, operation_id)
);
COMMIT;
