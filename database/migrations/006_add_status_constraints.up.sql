-- Добавляем проверку валидных статусов для кастингов
ALTER TABLE castings 
ADD CONSTRAINT check_casting_status 
CHECK (status IN ('draft', 'active', 'closed', 'cancelled'));

-- Добавляем проверку валидных статусов для откликов
ALTER TABLE responses 
ADD CONSTRAINT check_response_status 
CHECK (status IN ('pending', 'accepted', 'rejected'));

-- Добавляем проверку валидных статусов для пользователей
ALTER TABLE users 
ADD CONSTRAINT check_user_status 
CHECK (status IN ('pending', 'active', 'suspended', 'banned'));

-- Добавляем индексы для оптимизации запросов по статусам
CREATE INDEX idx_castings_status ON castings(status);
CREATE INDEX idx_castings_event_date ON castings(event_date) WHERE status = 'active';
CREATE INDEX idx_responses_status ON responses(status);
CREATE INDEX idx_users_status ON users(status);

-- Добавляем проверку диапазонов зарплаты
ALTER TABLE castings 
ADD CONSTRAINT check_salary_range 
CHECK (min_salary IS NULL OR max_salary IS NULL OR min_salary <= max_salary);

-- Добавляем проверку диапазонов возраста
ALTER TABLE castings 
ADD CONSTRAINT check_age_range 
CHECK (min_age IS NULL OR max_age IS NULL OR min_age <= max_age);

-- Добавляем проверку диапазонов роста
ALTER TABLE castings 
ADD CONSTRAINT check_height_range 
CHECK (min_height IS NULL OR max_height IS NULL OR min_height <= max_height);
