ALTER TABLE federated_servers
  ADD COLUMN endpoint TEXT;

UPDATE federated_servers
SET endpoint = 'https://' || domain
WHERE endpoint IS NULL OR endpoint = '';
