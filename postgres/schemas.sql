CREATE TYPE RunStatus AS ENUM ('started', 'finished');

CREATE TABLE run (
    id SERIAL PRIMARY KEY,
    run_at TIMESTAMP NOT NULL,
    run_status RunStatus NOT NULL,
);

CREATE TYPE TaskStatus AS ENUM ('cancelled', 'ok', 'error');

CREATE TABLE run_status_codes (
    run_id  INTEGER NOT NULL,
    url_address TEXT NOT NULL,
    task_status TaskStatus NOT NULL,
    status_code INT,
);