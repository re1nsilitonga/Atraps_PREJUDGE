-- FROZEN T+4. Changes require all 4. See PRD.md §9, TASKS.md PJ-103.
-- Idempotent: safe to re-run against the same database.

DO $$ BEGIN
    CREATE TYPE domain_status AS ENUM ('candidate', 'confirmed', 'blocked', 'false_pos');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE detection_source AS ENUM ('L1', 'L2', 'trustpositif');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS fingerprint_clusters (
    id                          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    registrar                   text,
    hosting_ip                  inet,
    asn                         text,
    nameserver                  text,
    tld                         text,
    domain_count                int NOT NULL DEFAULT 0,
    first_registration_date     date,
    last_registration_date      date,
    registration_window_hours   int,
    registration_burst_score    float,
    created_at                  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS domains (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    domain                 text NOT NULL UNIQUE,
    cluster_id             uuid REFERENCES fingerprint_clusters(id),
    status                 domain_status NOT NULL DEFAULT 'candidate',
    source                 detection_source,
    confidence             float,
    reason                 text,
    matched_fields         jsonb,
    first_seen             timestamptz NOT NULL DEFAULT now(),
    registered_at          date,
    blocked_at             timestamptz,
    source_masked_pattern  text
);

CREATE INDEX IF NOT EXISTS idx_domains_status ON domains(status);

CREATE TABLE IF NOT EXISTS detections (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id     uuid NOT NULL REFERENCES domains(id),
    layer         int NOT NULL CHECK (layer IN (1, 2)),
    confidence    float,
    reason        text,
    evidence_url  text,
    raw_response  jsonb,
    detected_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS whois_records (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id      uuid NOT NULL REFERENCES domains(id),
    registrar      text,
    nameservers    text[],
    created_date   date,
    raw            jsonb,
    fetched_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bootstrap_runs (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    run_at                 timestamptz NOT NULL DEFAULT now(),
    l2_confirmations       int NOT NULL DEFAULT 0,
    l1_preemptive_catches  int NOT NULL DEFAULT 0,
    l1_misses              int NOT NULL DEFAULT 0,
    notes                  text
);

-- RLS disabled for MVP: locked RLS silently breaking realtime at demo time
-- is a real failure mode (PRD §14, PJ-103 note). Revisit before any public launch.
ALTER TABLE fingerprint_clusters DISABLE ROW LEVEL SECURITY;
ALTER TABLE domains DISABLE ROW LEVEL SECURITY;
ALTER TABLE detections DISABLE ROW LEVEL SECURITY;
ALTER TABLE whois_records DISABLE ROW LEVEL SECURITY;
ALTER TABLE bootstrap_runs DISABLE ROW LEVEL SECURITY;
