\connect antiplag_analysis;

CREATE TABLE IF NOT EXISTS reports (
                                       id         SERIAL PRIMARY KEY,
                                       work_id    INT             NOT NULL,
                                       status     TEXT            NOT NULL,
                                       similarity DOUBLE PRECISION NOT NULL DEFAULT -1,
                                       details    TEXT            NOT NULL,
                                       created_at TIMESTAMP       NOT NULL DEFAULT NOW(),
    CONSTRAINT reports_similarity_valid
    CHECK (
              similarity = -1
              OR (similarity >= 0 AND similarity <= 100)
    )
    );
