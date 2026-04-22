-- Migration 014: Add roles to users table
-- First user created will be admin by default

-- Add role column to users table
ALTER TABLE users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'user';

-- Add constraint to ensure role is valid
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'));

-- Create index on role for efficient admin queries
CREATE INDEX idx_users_role ON users(role);

-- Make the first user (by created_at) an admin
UPDATE users
SET role = 'admin'
WHERE id = (
    SELECT id FROM users ORDER BY created_at ASC LIMIT 1
);

-- Log the result
DO $$
DECLARE
    user_count INTEGER;
    admin_count INTEGER;
    first_user_email TEXT;
BEGIN
    SELECT COUNT(*) INTO user_count FROM users;
    SELECT COUNT(*) INTO admin_count FROM users WHERE role = 'admin';
    SELECT email INTO first_user_email FROM users WHERE role = 'admin' ORDER BY created_at ASC LIMIT 1;

    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Migration 014 Complete!';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Total users: %', user_count;
    RAISE NOTICE 'Admins: %', admin_count;
    IF first_user_email IS NOT NULL THEN
        RAISE NOTICE 'First admin: %', first_user_email;
    END IF;
    RAISE NOTICE '=============================================================================';
END $$;
