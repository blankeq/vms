CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user'
);

CREATE TABLE IF NOT EXISTS cameras (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    rtsp_link TEXT NOT NULL
);

INSERT INTO users (login, password_hash, role)
VALUES (
    'admin',
    '$2b$10$V22uRb8lhizhJyhekQctrOUxs8Z7LkdOvvgQjWPw9kpe5T/SeW7Dm',
    'admin'
);
