CREATE TABLE IF NOT EXISTS driver_sessions (
  driver_session_id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
  driver_id UUID REFERENCES drivers (driver_id) NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now (),
  ended_at TIMESTAMPTZ,
  total_rides INTEGER DEFAULT 0,
  total_earnings DECIMAL(10, 2) DEFAULT 0
);
