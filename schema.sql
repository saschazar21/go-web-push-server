-- Create the subscriptions table with a many-to-one relation to recipients
CREATE TABLE subscription (
  endpoint TEXT PRIMARY KEY,
  expiration_time TIMESTAMPTZ,
  client_id VARCHAR(255) NOT NULL,
  recipient_id VARCHAR(255) NOT NULL
);

-- Create the keys table with a one-to-one relation to subscriptions
CREATE TABLE keys (
  p256dh VARCHAR(87) PRIMARY KEY,
  auth_secret VARCHAR(22) NOT NULL,
  subscription_endpoint TEXT NOT NULL,
  FOREIGN KEY (subscription_endpoint) REFERENCES subscription(endpoint) ON
  DELETE
    CASCADE
);