-- Pilot seed: one union with its wards, plus the upazila and seat above it.
-- Mirrors the frontend mock geography so both sides line up. Idempotent.
INSERT INTO areas (id, name, level, parent_id) VALUES
    ('seat-1',    'আসন-১',           'seat',    NULL),
    ('upazila-1', 'সদর উপজেলা',       'upazila', 'seat-1'),
    ('union-1',   'রায়পুর ইউনিয়ন',   'union',   'upazila-1'),
    ('ward-1',    'ওয়ার্ড ১',         'ward',    'union-1'),
    ('ward-2',    'ওয়ার্ড ২',         'ward',    'union-1')
ON CONFLICT (id) DO NOTHING;
