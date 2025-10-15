-- Migration: Add test data for admin panel
-- Description: Populates database with realistic test data for all services

BEGIN;

-- Insert Admin Users
INSERT INTO users (user_id, email, role, status, password_hash, attrs) VALUES
-- Admin users
('a0000000-0000-0000-0000-000000000001', 'admin@ridehail.com', 'ADMIN', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "System Administrator", "phone": "+7-701-000-0001"}'),
('a0000000-0000-0000-0000-000000000002', 'manager@ridehail.com', 'ADMIN', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Operations Manager", "phone": "+7-701-000-0002"}');

-- Insert Passenger Users
INSERT INTO users (user_id, email, role, status, password_hash, attrs) VALUES
('b0000000-0000-0000-0000-000000000001', 'saule.karimova@email.com', 'PASSENGER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Saule Karimova", "phone": "+7-701-111-1111", "preferred_payment": "card"}'),
('b0000000-0000-0000-0000-000000000002', 'arman.zhunusov@email.com', 'PASSENGER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Arman Zhunusov", "phone": "+7-701-111-1112", "preferred_payment": "cash"}'),
('b0000000-0000-0000-0000-000000000003', 'ayana.tulegenova@email.com', 'PASSENGER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Ayana Tulegenova", "phone": "+7-701-111-1113", "preferred_payment": "card"}'),
('b0000000-0000-0000-0000-000000000004', 'dmitry.ivanov@email.com', 'PASSENGER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Dmitry Ivanov", "phone": "+7-701-111-1114", "preferred_payment": "card"}'),
('b0000000-0000-0000-0000-000000000005', 'mariya.petrova@email.com', 'PASSENGER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Mariya Petrova", "phone": "+7-701-111-1115", "preferred_payment": "cash"}');

-- Insert Driver Users
INSERT INTO users (user_id, email, role, status, password_hash, attrs) VALUES
('d0000000-0000-0000-0000-000000000001', 'aidar.nurlan@email.com', 'DRIVER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Aidar Nurlan", "phone": "+7-701-222-2221"}'),
('d0000000-0000-0000-0000-000000000002', 'ruslan.omarov@email.com', 'DRIVER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Ruslan Omarov", "phone": "+7-701-222-2222"}'),
('d0000000-0000-0000-0000-000000000003', 'almaz.kairat@email.com', 'DRIVER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Almaz Kairat", "phone": "+7-701-222-2223"}'),
('d0000000-0000-0000-0000-000000000004', 'bekzat.ashimov@email.com', 'DRIVER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Bekzat Ashimov", "phone": "+7-701-222-2224"}'),
('d0000000-0000-0000-0000-000000000005', 'gulnara.temir@email.com', 'DRIVER', 'ACTIVE', '$2a$10$YQvJP2aCzHZ5.5Z5Z5Z5Z.5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z', 
 '{"name": "Gulnara Temir", "phone": "+7-701-222-2225"}');

-- Insert Driver Profiles
INSERT INTO drivers (driver_id, license_number, vehicle_type, vehicle_attrs, rating, total_rides, total_earnings, status, is_verified) VALUES
('d0000000-0000-0000-0000-000000000001', 'DL001ABC', 'ECONOMY', 
 '{"vehicle_make": "Toyota", "vehicle_model": "Camry", "vehicle_color": "White", "vehicle_plate": "KZ 001 ABC", "vehicle_year": 2020}', 
 4.8, 245, 356700.00, 'AVAILABLE', true),
('d0000000-0000-0000-0000-000000000002', 'DL002DEF', 'ECONOMY', 
 '{"vehicle_make": "Hyundai", "vehicle_model": "Sonata", "vehicle_color": "Black", "vehicle_plate": "KZ 002 DEF", "vehicle_year": 2019}', 
 4.6, 189, 278900.00, 'AVAILABLE', true),
('d0000000-0000-0000-0000-000000000003', 'DL003GHI', 'PREMIUM', 
 '{"vehicle_make": "Mercedes", "vehicle_model": "E-Class", "vehicle_color": "Silver", "vehicle_plate": "KZ 003 GHI", "vehicle_year": 2021}', 
 4.9, 156, 412500.00, 'BUSY', true),
('d0000000-0000-0000-0000-000000000004', 'DL004JKL', 'XL', 
 '{"vehicle_make": "Volkswagen", "vehicle_model": "Transporter", "vehicle_color": "Blue", "vehicle_plate": "KZ 004 JKL", "vehicle_year": 2020}', 
 4.5, 98, 189600.00, 'OFFLINE', true),
('d0000000-0000-0000-0000-000000000005', 'DL005MNO', 'ECONOMY', 
 '{"vehicle_make": "Kia", "vehicle_model": "Rio", "vehicle_color": "Red", "vehicle_plate": "KZ 005 MNO", "vehicle_year": 2022}', 
 4.7, 134, 198400.00, 'EN_ROUTE', true);

-- Insert Current Driver Locations
INSERT INTO coordinates (coordinate_id, entity_id, entity_type, address, latitude, longitude, is_current) VALUES
-- Driver 1 - Available near Almaty Central Park
('c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', 'DRIVER', 'Near Almaty Central Park', 43.238900, 76.889700, true),
-- Driver 2 - Available near Republic Square
('c0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000002', 'DRIVER', 'Near Republic Square', 43.235000, 76.945000, true),
-- Driver 3 - Busy in Tsum area
('c0000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000003', 'DRIVER', 'Tsum Shopping Center', 43.250000, 76.950000, true),
-- Driver 4 - Offline (no current location)
-- Driver 5 - En route to pickup
('c0000000-0000-0000-0000-000000000004', 'd0000000-0000-0000-0000-000000000005', 'DRIVER', 'Abay Avenue', 43.242000, 76.920000, true);

-- Insert Ride Coordinates (Pickup and Destination points)
INSERT INTO coordinates (coordinate_id, entity_id, entity_type, address, latitude, longitude, fare_amount, distance_km, duration_minutes, is_current) VALUES
-- Ride 1 - Completed ride
('c0000000-0000-0000-0000-000000000101', 'a0000000-0000-0000-0000-000000000001', 'PASSENGER', 'Almaty Central Park', 43.238949, 76.889709, 1450.00, 5.2, 15, false),
('c0000000-0000-0000-0000-000000000102', 'a0000000-0000-0000-0000-000000000001', 'PASSENGER', 'Kok-Tobe Hill', 43.222015, 76.851511, NULL, NULL, NULL, false),
-- Ride 2 - In progress ride
('c0000000-0000-0000-0000-000000000103', 'a0000000-0000-0000-0000-000000000002', 'PASSENGER', 'Mega Alma-Ata Mall', 43.210000, 76.880000, 890.00, 3.1, 8, false),
('c0000000-0000-0000-0000-000000000104', 'a0000000-0000-0000-0000-000000000002', 'PASSENGER', 'Almaty Airport', 43.352000, 77.040000, NULL, NULL, NULL, false),
-- Ride 3 - Requested ride
('c0000000-0000-0000-0000-000000000105', 'a0000000-0000-0000-0000-000000000003', 'PASSENGER', 'Republic Square', 43.235000, 76.945000, 1250.00, 4.5, 12, false),
('c0000000-0000-0000-0000-000000000106', 'a0000000-0000-0000-0000-000000000003', 'PASSENGER', 'Sairan Resort', 43.180000, 76.890000, NULL, NULL, NULL, false),
-- Ride 4 - Matched ride
('c0000000-0000-0000-0000-000000000107', 'a0000000-0000-0000-0000-000000000004', 'PASSENGER', 'Tsum Shopping Center', 43.250000, 76.950000, 2100.00, 7.8, 20, false),
('c0000000-0000-0000-0000-000000000108', 'a0000000-0000-0000-0000-000000000004', 'PASSENGER', 'Alatau District', 43.300000, 76.970000, NULL, NULL, NULL, false),
-- Ride 5 - Cancelled ride
('c0000000-0000-0000-0000-000000000109', 'a0000000-0000-0000-0000-000000000005', 'PASSENGER', 'Abay Opera House', 43.240000, 76.940000, 680.00, 2.3, 6, false),
('c0000000-0000-0000-0000-000000000110', 'a0000000-0000-0000-0000-000000000005', 'PASSENGER', 'Green Bazaar', 43.255000, 76.950000, NULL, NULL, NULL, false);

-- Insert Rides with Various Statuses
INSERT INTO rides (ride_id, ride_number, passenger_id, driver_id, vehicle_type, status, priority, 
                   requested_at, matched_at, arrived_at, started_at, completed_at, cancelled_at,
                   cancellation_reason, estimated_fare, final_fare, pickup_coord_id, destination_coord_id) VALUES
-- Completed ride (yesterday)
('a0000000-0000-0000-0000-000000000001', 'RIDE_20241215_001', 'b0000000-0000-0000-0000-000000000001', 
 'd0000000-0000-0000-0000-000000000001', 'ECONOMY', 'COMPLETED', 1,
 NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '2 minutes', 
 NOW() - INTERVAL '1 day' + INTERVAL '5 minutes', NOW() - INTERVAL '1 day' + INTERVAL '7 minutes',
 NOW() - INTERVAL '1 day' + INTERVAL '22 minutes', NULL, NULL,
 1450.00, 1520.00, 'c0000000-0000-0000-0000-000000000101', 'c0000000-0000-0000-0000-000000000102'),

-- In progress ride
('a0000000-0000-0000-0000-000000000002', 'RIDE_20241216_001', 'b0000000-0000-0000-0000-000000000002', 
 'd0000000-0000-0000-0000-000000000003', 'PREMIUM', 'IN_PROGRESS', 1,
 NOW() - INTERVAL '15 minutes', NOW() - INTERVAL '12 minutes', 
 NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '4 minutes',
 NULL, NULL, NULL,
 2100.00, NULL, 'c0000000-0000-0000-0000-000000000107', 'c0000000-0000-0000-0000-000000000108'),

-- Requested ride (waiting for driver)
('a0000000-0000-0000-0000-000000000003', 'RIDE_20241216_002', 'b0000000-0000-0000-0000-000000000003', 
 NULL, 'ECONOMY', 'REQUESTED', 1,
 NOW() - INTERVAL '3 minutes', NULL, NULL, NULL, NULL, NULL, NULL,
 1250.00, NULL, 'c0000000-0000-0000-0000-000000000105', 'c0000000-0000-0000-0000-000000000106'),

-- Matched ride (driver on the way)
('a0000000-0000-0000-0000-000000000004', 'RIDE_20241216_003', 'b0000000-0000-0000-0000-000000000004', 
 'd0000000-0000-0000-0000-000000000005', 'ECONOMY', 'EN_ROUTE', 1,
 NOW() - INTERVAL '8 minutes', NOW() - INTERVAL '5 minutes', 
 NULL, NULL, NULL, NULL, NULL,
 890.00, NULL, 'c0000000-0000-0000-0000-000000000103', 'c0000000-0000-0000-0000-000000000104'),

-- Cancelled ride
('a0000000-0000-0000-0000-000000000005', 'RIDE_20241216_004', 'b0000000-0000-0000-0000-000000000005', 
 NULL, 'ECONOMY', 'CANCELLED', 1,
 NOW() - INTERVAL '10 minutes', NULL, NULL, NULL, NULL, 
 NOW() - INTERVAL '8 minutes', 'Changed my mind',
 680.00, NULL, 'c0000000-0000-0000-0000-000000000109', 'c0000000-0000-0000-0000-000000000110'),

-- Arrived at pickup
('a0000000-0000-0000-0000-000000000006', 'RIDE_20241216_005', 'b0000000-0000-0000-0000-000000000001', 
 'd0000000-0000-0000-0000-000000000002', 'ECONOMY', 'ARRIVED', 1,
 NOW() - INTERVAL '12 minutes', NOW() - INTERVAL '8 minutes', 
 NOW() - INTERVAL '1 minute', NULL, NULL, NULL, NULL,
 950.00, NULL, 'c0000000-0000-0000-0000-000000000105', 'c0000000-0000-0000-0000-000000000106');

-- Insert Ride Events for Audit Trail
INSERT INTO ride_events (ride_event_id, ride_id, event_type, event_data) VALUES
-- Completed ride events
('e0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'RIDE_REQUESTED',
 '{"passenger_id": "a0000000-0000-0000-0000-000000000001", "pickup_address": "Almaty Central Park", "destination_address": "Kok-Tobe Hill", "estimated_fare": 1450.00}'::jsonb),

('e0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'DRIVER_MATCHED',
 jsonb_build_object(
   'driver_id', 'd0000000-0000-0000-0000-000000000001',
   'driver_name', 'Aidar Nurlan',
   'estimated_arrival', (NOW() - INTERVAL '1 day' + INTERVAL '5 minutes')::text
 )),

('e0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'DRIVER_ARRIVED',
 jsonb_build_object(
   'driver_id', 'd0000000-0000-0000-0000-000000000001',
   'actual_arrival', (NOW() - INTERVAL '1 day' + INTERVAL '5 minutes')::text
 )),

('e0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'RIDE_STARTED',
 '{"driver_id": "d0000000-0000-0000-0000-000000000001", "start_location": {"lat": 43.238949, "lng": 76.889709}}'::jsonb),

('e0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'RIDE_COMPLETED',
 '{"final_fare": 1520.00, "actual_distance_km": 5.5, "actual_duration_minutes": 15, "end_location": {"lat": 43.222015, "lng": 76.851511}}'::jsonb),

-- In progress ride events
('e0000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000002', 'RIDE_REQUESTED',
 '{"passenger_id": "a0000000-0000-0000-0000-000000000002", "pickup_address": "Mega Alma-Ata Mall", "destination_address": "Almaty Airport", "estimated_fare": 2100.00}'::jsonb),

('e0000000-0000-0000-0000-000000000007', 'a0000000-0000-0000-0000-000000000002', 'DRIVER_MATCHED',
 jsonb_build_object(
   'driver_id', 'd0000000-0000-0000-0000-000000000003',
   'driver_name', 'Almaz Kairat',
   'estimated_arrival', (NOW() - INTERVAL '10 minutes')::text
 )),

-- Cancelled ride events
('e0000000-0000-0000-0000-000000000008', 'a0000000-0000-0000-0000-000000000005', 'RIDE_REQUESTED',
 '{"passenger_id": "a0000000-0000-0000-0000-000000000005", "pickup_address": "Abay Opera House", "destination_address": "Green Bazaar", "estimated_fare": 680.00}'::jsonb),

('e0000000-0000-0000-0000-000000000009', 'a0000000-0000-0000-0000-000000000005', 'RIDE_CANCELLED',
 '{"reason": "Changed my mind", "cancelled_by": "passenger"}'::jsonb);

-- Insert Driver Sessions
INSERT INTO driver_sessions (driver_session_id, driver_id, started_at, ended_at, total_rides, total_earnings) VALUES
-- Active session
('f0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '3 hours', NULL, 4, 5200.00),
('f0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000002', NOW() - INTERVAL '2 hours', NULL, 3, 3800.00),
-- Completed session from yesterday
('f0000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000001', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '8 hours', 12, 18500.00);

-- Insert Location History for Analytics
INSERT INTO location_history (location_history_id, coord_id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees, recorded_at, ride_id) VALUES
-- Driver 1 location history (last 30 minutes)
('f0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', 43.238800, 76.889600, 10.0, 0.0, 0.0, NOW() - INTERVAL '30 minutes', NULL),
('f0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', 43.238850, 76.889650, 8.0, 5.0, 45.0, NOW() - INTERVAL '25 minutes', NULL),
('f0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', 43.238900, 76.889700, 5.0, 0.0, 0.0, NOW() - INTERVAL '20 minutes', NULL),

-- Driver 3 location history during current ride
('f0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000003', 43.250000, 76.950000, 10.0, 0.0, 0.0, NOW() - INTERVAL '15 minutes', 'a0000000-0000-0000-0000-000000000002'),
('f0000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000003', 43.255000, 76.960000, 8.0, 40.0, 90.0, NOW() - INTERVAL '12 minutes', 'a0000000-0000-0000-0000-000000000002'),
('f0000000-0000-0000-0000-000000000006', 'c0000000-0000-0000-0000-000000000003', 'd0000000-0000-0000-0000-000000000003', 43.260000, 76.970000, 8.0, 45.0, 95.0, NOW() - INTERVAL '10 minutes', 'a0000000-0000-0000-0000-000000000002');

COMMIT;