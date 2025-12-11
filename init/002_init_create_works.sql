\connect antiplag_storage;

CREATE TABLE IF NOT EXISTS works (
                                     id          SERIAL PRIMARY KEY,
                                     student     TEXT        NOT NULL,
                                     task        TEXT        NOT NULL,
                                     file_path   TEXT        NOT NULL,
                                     uploaded_at TIMESTAMP   NOT NULL DEFAULT NOW()
    );
