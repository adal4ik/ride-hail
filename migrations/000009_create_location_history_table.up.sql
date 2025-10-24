CREATE TABLE IF NOT EXISTS location_history (
  location_history_id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
  coord_id UUID REFERENCES coordinates (coord_id),
  driver_id UUID REFERENCES drivers (driver_id),
  latitude DECIMAL(10, 8) NOT NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DECIMAL(13, 8) NOT NULL CHECK (longitude BETWEEN -180 AND 180),
  accuracy_meters DECIMAL(6, 2),
  speed_kmh DECIMAL(5, 2),
  heading_degrees DECIMAL(5, 2) CHECK (heading_degrees BETWEEN 0 AND 360),
  recorded_at timestamptz NOT NULL DEFAULT NOW (),
  ride_id UUID REFERENCES rides (ride_id)
);
