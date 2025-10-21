-- User roles enumeration
CREATE TYPE roles AS ENUM (
  'PASSENGER', -- Passenger/Customer
  'ADMIN' -- Administrator
);

-- User status enumeration
CREATE TYPE user_status AS ENUM (
  'ACTIVE', -- Active user
  'INACTIVE', -- Inactive/Suspended
  'BANNED' -- Banned user
);

-- Ride status enumeration
CREATE TYPE ride_status AS ENUM (
  'REQUESTED', -- Ride has been requested by customer
  'MATCHED', -- Driver has been matched to the ride
  'EN_ROUTE', -- Driver is on the way to pickup location
  'ARRIVED', -- Driver has arrived at pickup location
  'IN_PROGRESS', -- Ride is currently in progress
  'COMPLETED', -- Ride has been successfully completed
  'CANCELLED' -- Ride was cancelled
);

-- Vehicle type enumeration
CREATE TYPE vehicle_type AS ENUM (
  'ECONOMY', -- Standard economy ride
  'PREMIUM', -- Premium comfort ride
  'XL' -- Extra large vehicle for groups
);

-- Event type enumeration for audit trail
CREATE TYPE ride_event_type AS ENUM (
  'RIDE_REQUESTED', -- Initial ride request
  'DRIVER_MATCHED', -- Driver assigned to ride
  'DRIVER_ARRIVED', -- Driver arrived at pickup
  'RIDE_STARTED', -- Ride began
  'RIDE_COMPLETED', -- Ride finished
  'RIDE_CANCELLED', -- Ride was cancelled
  'STATUS_CHANGED', -- General status change
  'LOCATION_UPDATED', -- Location update during ride
  'FARE_ADJUSTED' -- Fare was adjusted
);

-- Driver status enumeration
CREATE TYPE driver_status AS ENUM (
  'OFFLINE', -- Driver is not accepting rides
  'AVAILABLE', -- Driver is available to accept rides
  'BUSY', -- Driver is currently occupied
  'EN_ROUTE' -- Driver is on the way to pickup
);
