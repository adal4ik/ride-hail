CREATE TABLE IF NOT EXISTS coordinates (
  coord_id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
  entity_id UUID NOT NULL, -- driver_id or passenger_id
  entity_type TEXT NOT NULL CHECK (entity_type in ('DRIVER', 'PASSENGER')),
  address TEXT NOT NULL,
  latitude DECIMAL(10, 8) NOT NULL CHECK (latitude BETWEEN -90 AND 90),
  longitude DECIMAL(11, 8) NOT NULL CHECK (longitude BETWEEN -180 AND 180),
  fare_amount DECIMAL(10, 2) CHECK (fare_amount >= 0),
  distance_km DECIMAL(8, 2) CHECK (distance_km >= 0),
  duration_minutes INTEGER CHECK (duration_minutes >= 0),
  is_current BOOLEAN DEFAULT true
);
