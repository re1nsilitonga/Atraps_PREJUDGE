-- FIXTURE DATA. Not seed data. Production starts empty by design (PRD §4).
-- PURGE THIS before the demo — the opening beat is an empty database (PJ-104, PJ-803).

INSERT INTO fingerprint_clusters (
    id, registrar, hosting_ip, asn, nameserver, tld, domain_count,
    first_registration_date, last_registration_date, registration_window_hours, registration_burst_score
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'fixture-registrar.test', '203.0.113.10', 'AS64500', 'ns1.fixture-host.test', 'xyz', 6,
    '2026-07-01', '2026-07-01', 6, 0.92
);

-- FIXTURE
INSERT INTO domains (domain, cluster_id, status, source, confidence, reason, matched_fields, registered_at, blocked_at) VALUES
    ('fixture-sib1.test', '00000000-0000-0000-0000-000000000001', 'blocked',   'L1', 0.91, 'IP hosting sama dengan 5 situs terkonfirmasi', '["hosting_ip","nameserver","registration_burst"]', '2026-07-01', now()),
    ('fixture-sib2.test', '00000000-0000-0000-0000-000000000001', 'blocked',   'L1', 0.88, 'IP hosting sama dengan 5 situs terkonfirmasi', '["hosting_ip","nameserver"]',                       '2026-07-01', now()),
    ('fixture-sib3.test', '00000000-0000-0000-0000-000000000001', 'blocked',   'L1', 0.85, 'Registrar cocok dengan kluster dikenal',        '["registrar","registration_burst"]',                '2026-07-01', now()),
    ('fixture-sib4.test', '00000000-0000-0000-0000-000000000001', 'blocked',   'L1', 0.79, 'IP hosting sama dengan 5 situs terkonfirmasi', '["hosting_ip"]',                                    '2026-07-01', now()),
    ('fixture-sib5.test', '00000000-0000-0000-0000-000000000001', 'confirmed', 'L2', 0.95, 'slot UI, tombol deposit',                       '["hosting_ip","nameserver","registration_burst"]', '2026-07-01', NULL),
    ('fixture-seed.test',  '00000000-0000-0000-0000-000000000001', 'confirmed', 'L2', 0.97, 'slot UI, tombol deposit',                       '[]',                                                 '2026-07-01', NULL),
    ('fixture-cand01.test', NULL, 'candidate', NULL, NULL, NULL, NULL, NULL, NULL),
    ('fixture-cand02.test', NULL, 'candidate', NULL, NULL, NULL, NULL, NULL, NULL),
    ('fixture-cand03.test', NULL, 'candidate', NULL, NULL, NULL, NULL, NULL, NULL),
    ('fixture-cand04.test', NULL, 'candidate', NULL, NULL, NULL, NULL, NULL, NULL),
    ('fixture-fp01.test', NULL, 'false_pos', 'L2', 0.55, 'ternyata bukan judol, laporan diterima', '[]', NULL, NULL),
    ('fixture-fp02.test', NULL, 'false_pos', 'L2', 0.60, 'ternyata bukan judol, laporan diterima', '[]', NULL, NULL),
    ('fixture-tp01.test', NULL, 'blocked', 'trustpositif', 1.0, 'terverifikasi TrustPositif', '[]', NULL, now()),
    ('fixture-tp02.test', NULL, 'blocked', 'trustpositif', 1.0, 'terverifikasi TrustPositif', '[]', NULL, now()),
    ('fixture-l2-01.test', NULL, 'blocked', 'L2', 0.82, 'iklan slot, tombol deposit', '[]', NULL, now()),
    ('fixture-l2-02.test', NULL, 'blocked', 'L2', 0.90, 'iklan slot, tombol deposit', '[]', NULL, now()),
    ('fixture-l2-03.test', NULL, 'blocked', 'L2', 0.77, 'iklan slot, tombol deposit', '[]', NULL, now()),
    ('fixture-l1-solo01.test', NULL, 'blocked', 'L1', 0.65, 'registrar cocok dengan kluster dikenal', '["registrar"]', NULL, now()),
    ('fixture-l1-solo02.test', NULL, 'blocked', 'L1', 0.68, 'registrar cocok dengan kluster dikenal', '["registrar"]', NULL, now()),
    ('fixture-cand05.test', NULL, 'candidate', NULL, NULL, NULL, NULL, NULL, NULL);

-- === PURGE (run this before the demo, PJ-803) ===
-- Targets only rows this file created. Safe to re-run.
DELETE FROM domains WHERE domain LIKE 'fixture-%';
DELETE FROM fingerprint_clusters WHERE registrar = 'fixture-registrar.test';
