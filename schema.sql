-- Create the subscriptions table with a many-to-one relation to recipients
CREATE TABLE webpush_subscriptions (
  endpoint_hash BYTEA PRIMARY KEY,
  endpoint BYTEA NOT NULL,
  expiration_time TIMESTAMPTZ,
  client_id VARCHAR(255) NOT NULL,
  recipient_id VARCHAR(255) NOT NULL
);

-- Create indexes for efficient querying
CREATE INDEX idx_subscription_recipient_id ON webpush_subscriptions(recipient_id);
CREATE INDEX idx_subscription_client_id ON webpush_subscriptions(client_id);
CREATE INDEX idx_subscription_expiration_time ON webpush_subscriptions(expiration_time)
  WHERE expiration_time IS NOT NULL;

-- Create the keys table with a one-to-one relation to subscriptions
CREATE TABLE webpush_keys (
  p256dh_hash BYTEA PRIMARY KEY,
  p256dh BYTEA NOT NULL,
  auth_secret_hash BYTEA NOT NULL UNIQUE,
  auth_secret BYTEA NOT NULL,
  subscription_hash BYTEA NOT NULL,
  FOREIGN KEY (subscription_hash) REFERENCES webpush_subscriptions(endpoint_hash) ON
  DELETE
    CASCADE
);

-- Create indexes for efficient querying
CREATE UNIQUE INDEX idx_keys_subscription_hash ON webpush_keys(subscription_hash);