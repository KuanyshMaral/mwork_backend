-- Удаляем ограничения
ALTER TABLE castings DROP CONSTRAINT IF EXISTS check_casting_status;
ALTER TABLE castings DROP CONSTRAINT IF EXISTS check_salary_range;
ALTER TABLE castings DROP CONSTRAINT IF EXISTS check_age_range;
ALTER TABLE castings DROP CONSTRAINT IF EXISTS check_height_range;

ALTER TABLE responses DROP CONSTRAINT IF EXISTS check_response_status;
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_user_status;

-- Удаляем индексы
DROP INDEX IF EXISTS idx_castings_status;
DROP INDEX IF EXISTS idx_castings_event_date;
DROP INDEX IF EXISTS idx_responses_status;
DROP INDEX IF EXISTS idx_users_status;
