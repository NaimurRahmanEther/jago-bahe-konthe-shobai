-- Officials directory seeded by tier, within the pilot geography (B0 areas).
-- Officials in the extra unions (the B4 admin-vote pool) arrive with B4.
INSERT INTO officials (id, name, phone, tier, area_id) VALUES
    ('off-1', 'মো. করিম উদ্দিন',   '01710000001', 'ward_member',     'ward-1'),
    ('off-2', 'সালমা বেগম',        '01710000002', 'women_member',    'union-1'),
    ('off-3', 'আব্দুল হক',         '01710000003', 'union_chairman',  'union-1'),
    ('off-4', 'ফরিদা ইয়াসমিন',     '01710000004', 'upazila_chairman','upazila-1'),
    ('off-5', 'রফিকুল ইসলাম',      '01710000005', 'mp',              'seat-1')
ON CONFLICT (id) DO NOTHING;

-- Login accounts, one per role, matching the frontend's mock login hint so the
-- same credentials work once USE_MOCK is flipped off:
--   resident 01810000001 / resident123
--   official 01710000001 / official123
--   admin    01710000003 / admin123
-- Password hashes are bcrypt (cost 10).
INSERT INTO accounts (id, name, phone, password_hash, role, nid, union_id, verified, official_id) VALUES
    ('acct-seed-res-1',   'রহিমা খাতুন',    '01810000001', '$2a$10$7Olw9BI7RPrbTZ7q0HDrheVdzrdMw0HTHXu0ob.GcMYtxLmN/6Csy', 'resident', '1234567890111', 'union-1', true, NULL),
    ('acct-seed-off-1',   'মো. করিম উদ্দিন', '01710000001', '$2a$10$ZMBuPtJyAB0QBvb4MulntucThVDe.HUrGazbc50sAIgMBZygUG8N.', 'official', NULL, NULL, true, 'off-1'),
    ('acct-seed-admin-1', 'আব্দুল হক',       '01710000003', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-3')
ON CONFLICT (id) DO NOTHING;
