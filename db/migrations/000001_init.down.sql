BEGIN;

DROP TABLE IF EXISTS event_spins         CASCADE;
DROP TABLE IF EXISTS event_prizes        CASCADE;
DROP TABLE IF EXISTS event_registrations CASCADE;
DROP TABLE IF EXISTS events              CASCADE;
DROP TABLE IF EXISTS activity_logs       CASCADE;
DROP TABLE IF EXISTS sales               CASCADE;
DROP TABLE IF EXISTS discounts           CASCADE;
DROP TABLE IF EXISTS inventory           CASCADE;
DROP TABLE IF EXISTS vehicle_catalog     CASCADE;
DROP TABLE IF EXISTS refresh_tokens      CASCADE;
DROP TABLE IF EXISTS customer_rank_history CASCADE;
DROP TABLE IF EXISTS customers           CASCADE;
DROP TABLE IF EXISTS users               CASCADE;
DROP TABLE IF EXISTS settings            CASCADE;

DROP TYPE IF EXISTS event_type;
DROP TYPE IF EXISTS vehicle_status;
DROP TYPE IF EXISTS customer_rank;
DROP TYPE IF EXISTS user_role;

COMMIT;
