\connect antiplag_analysis




CREATE TABLE reports (
    id SERIAL PRIMARY KEY,
    work_id INT NOT NULL,
    status TEXT NOT NULL,
    similarity FLOAT,
    details TEXT NOT NULL ,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);