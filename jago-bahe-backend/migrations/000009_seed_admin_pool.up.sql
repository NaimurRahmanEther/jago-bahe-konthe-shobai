-- B4 admin-vote pool. B0/B1 seeded one union (union-1) with one admin; an
-- above-union vote needs several unions and their admins so quorum + majority is
-- meaningful. This adds two more unions under upazila-1, a second upazila with
-- its own union, the union-chairman officials for each, and one admin account per
-- new union (all sharing the dev password `admin123`, phone = their official's).

-- Extra geography: two more unions in upazila-1, plus a second upazila + union.
INSERT INTO areas (id, name, level, parent_id) VALUES
    ('union-2',   'বাগমারা ইউনিয়ন',  'union',   'upazila-1'),
    ('union-3',   'চরপাড়া ইউনিয়ন',   'union',   'upazila-1'),
    ('upazila-2', 'পূর্ব উপজেলা',      'upazila', 'seat-1'),
    ('union-4',   'দক্ষিণ ইউনিয়ন',    'union',   'upazila-2')
ON CONFLICT (id) DO NOTHING;

-- Union chairmen for the new unions, plus a chairman for the second upazila (so
-- monitor resolution has a tier to point to there).
INSERT INTO officials (id, name, phone, tier, area_id) VALUES
    ('off-6', 'নূরুল আমিন',     '01710000006', 'union_chairman',   'union-2'),
    ('off-7', 'শাহানা পারভীন',  '01710000007', 'union_chairman',   'union-3'),
    ('off-8', 'জামাল হোসেন',    '01710000008', 'union_chairman',   'union-4'),
    ('off-9', 'তৌহিদুল ইসলাম',  '01710000009', 'upazila_chairman', 'upazila-2')
ON CONFLICT (id) DO NOTHING;

-- One admin per new union (bound to that union's chairman). Same bcrypt hash as
-- the B1 admin seed, so `admin123` logs in as each. Phone matches the official.
INSERT INTO accounts (id, name, phone, password_hash, role, nid, union_id, verified, official_id) VALUES
    ('acct-seed-admin-2', 'নূরুল আমিন',    '01710000006', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-6'),
    ('acct-seed-admin-3', 'শাহানা পারভীন', '01710000007', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-7'),
    ('acct-seed-admin-4', 'জামাল হোসেন',   '01710000008', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-8')
ON CONFLICT (id) DO NOTHING;
