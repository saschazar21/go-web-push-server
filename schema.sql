-- Create the recipients table with a composite primary key
CREATE TABLE recipient (
  id VARCHAR(255),
  client_id VARCHAR(255),
  PRIMARY KEY (id, client_id)
);

-- Create the subscriptions table with a many-to-one relation to recipients
CREATE TABLE subscription (
  endpoint VARCHAR(255) PRIMARY KEY,
  expiration_time TIMESTAMPTZ,
  recipient_id VARCHAR(255),
  client_id VARCHAR(255),
  FOREIGN KEY (recipient_id, client_id) REFERENCES recipient(id, client_id) ON
  DELETE
    CASCADE
);

-- Create the keys table with a one-to-one relation to subscriptions
CREATE TABLE keys (
  p256dh VARCHAR(87) PRIMARY KEY,
  auth_secret VARCHAR(22) NOT NULL,
  subscription_endpoint VARCHAR(255),
  FOREIGN KEY (subscription_endpoint) REFERENCES subscription(endpoint) ON
  DELETE
    CASCADE
);