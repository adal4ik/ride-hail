CREATE TABLE IF NOT EXISTS drivers (
    driver_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    license_number TEXT UNIQUE NOT NULL,
    vehicle_type vehicle_type NOT NULL,
    vehicle_attrs JSONB DEFAULT '{}'::JSONB,
    rating DECIMAL(3,2) DEFAULT 5.0 CHECK (rating BETWEEN 1.0 AND 5.0),
    total_rides INTEGER DEFAULT 0 CHECK (total_rides >= 0),
    total_earnings DECIMAL(10,2) DEFAULT 0 CHECK (total_earnings >= 0),
    status driver_status DEFAULT 'OFFLINE',
    is_verified BOOLEAN DEFAULT false,
    user_attrs JSONB DEFAULT '{}'::JSONB
);
