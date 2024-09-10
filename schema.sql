CREATE TABLE request_logs (
  id SERIAL PRIMARY KEY,
  ip INET NOT NULL,
  user_agent TEXT,
  path TEXT NOT NULL,
  request_time TIMESTAMP WITH TIME ZONE NOT NULL,
  method VARCHAR(10) NOT NULL,
  protocol VARCHAR(10) NOT NULL,
  host TEXT NOT NULL,
  accept_language TEXT,
  referer TEXT,
  content_type TEXT,
  content_length BIGINT,
  response_time INTERVAL NOT NULL
);

CREATE INDEX idx_request_logs_request_time ON request_logs (request_time);
CREATE INDEX idx_request_logs_path ON request_logs (path);
CREATE INDEX idx_request_logs_ip ON request_logs (ip);
