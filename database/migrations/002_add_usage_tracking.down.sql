-- Rollback usage tracking
DROP TABLE IF EXISTS usage;
DELETE FROM subscription_plans WHERE id IN ('free-tier', 'mwork-start', 'mwork-pro');
