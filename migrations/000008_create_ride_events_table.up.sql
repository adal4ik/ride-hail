CREATE TABLE IF NOT EXISTS ride_events (
  ride_event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
  ride_id UUID REFERENCES rides (ride_id) NOT NULL,
  event_type ride_event_type,
  event_data jsonb NOT NULL
);
