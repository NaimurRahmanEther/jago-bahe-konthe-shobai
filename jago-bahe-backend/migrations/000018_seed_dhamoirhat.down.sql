-- Restores the invented pilot geography (000003 + 000005 + 000009) and removes
-- the Dhamoirhat one.
--
-- THIS DOWN IS DELIBERATELY INCOMPLETE, AND CANNOT BE OTHERWISE. The up migration
-- deleted every problem and every audit entry so the old areas could be dropped,
-- and nothing records what they were. Rolling back restores the seed rows; it
-- does not bring the reports back. Run the up migration knowing that.

-- 1. The old geography and directory, exactly as 000003/000005/000009 seeded them.
INSERT INTO areas (id, name, level, parent_id) VALUES
    ('seat-1',    'আসন-১',           'seat',    NULL),
    ('upazila-1', 'সদর উপজেলা',       'upazila', 'seat-1'),
    ('upazila-2', 'পূর্ব উপজেলা',      'upazila', 'seat-1'),
    ('union-1',   'রায়পুর ইউনিয়ন',   'union',   'upazila-1'),
    ('union-2',   'বাগমারা ইউনিয়ন',  'union',   'upazila-1'),
    ('union-3',   'চরপাড়া ইউনিয়ন',   'union',   'upazila-1'),
    ('union-4',   'দক্ষিণ ইউনিয়ন',    'union',   'upazila-2'),
    ('ward-1',    'ওয়ার্ড ১',         'ward',    'union-1'),
    ('ward-2',    'ওয়ার্ড ২',         'ward',    'union-1')
ON CONFLICT (id) DO NOTHING;

INSERT INTO officials (id, name, phone, tier, area_id) VALUES
    ('off-1', 'মো. করিম উদ্দিন',   '01710000001', 'ward_member',      'ward-1'),
    ('off-2', 'সালমা বেগম',        '01710000002', 'women_member',     'union-1'),
    ('off-3', 'আব্দুল হক',         '01710000003', 'union_chairman',   'union-1'),
    ('off-4', 'ফরিদা ইয়াসমিন',     '01710000004', 'upazila_chairman', 'upazila-1'),
    ('off-5', 'রফিকুল ইসলাম',      '01710000005', 'mp',               'seat-1'),
    ('off-6', 'নূরুল আমিন',        '01710000006', 'union_chairman',   'union-2'),
    ('off-7', 'শাহানা পারভীন',     '01710000007', 'union_chairman',   'union-3'),
    ('off-8', 'জামাল হোসেন',       '01710000008', 'union_chairman',   'union-4'),
    ('off-9', 'তৌহিদুল ইসলাম',     '01710000009', 'upazila_chairman', 'upazila-2')
ON CONFLICT (id) DO NOTHING;

-- 2. Put the surviving seed accounts back where they were.
UPDATE accounts SET union_id = 'union-1' WHERE id = 'acct-seed-res-1';
UPDATE accounts SET official_id = 'off-1' WHERE id = 'acct-seed-off-1';
UPDATE official_claims SET official_id = 'off-1' WHERE account_id = 'acct-seed-off-1';

-- Residents who registered against a Dhamoirhat union land back in union-1.
UPDATE accounts SET union_id = 'union-1'
    WHERE union_id IN ('union-dhamoirhat', 'union-agradigun', 'union-alampur', 'union-umar',
                       'union-aranagar', 'union-jahanpur', 'union-isabpur', 'union-khelna',
                       'pourashava-dhamoirhat', 'upazila-dhamoirhat', 'seat-naogaon-2');

DELETE FROM official_claims WHERE official_id IN
    ('off-mp', 'off-upz-chair', 'off-upz-vice', 'off-upz-wvice', 'off-mayor',
     'off-chair-dhamoirhat', 'off-chair-agradigun', 'off-chair-alampur', 'off-chair-umar',
     'off-chair-aranagar', 'off-chair-jahanpur', 'off-chair-isabpur', 'off-chair-khelna')
    AND account_id <> 'acct-seed-off-1';

UPDATE accounts SET official_id = NULL
    WHERE official_id IN
    ('off-mp', 'off-upz-chair', 'off-upz-vice', 'off-upz-wvice', 'off-mayor',
     'off-chair-dhamoirhat', 'off-chair-agradigun', 'off-chair-alampur', 'off-chair-umar',
     'off-chair-aranagar', 'off-chair-jahanpur', 'off-chair-isabpur', 'off-chair-khelna')
    AND id <> 'acct-seed-off-1';

-- 3. The nine Dhamoirhat admins go; the four pilot admins come back.
DELETE FROM accounts WHERE id IN
    ('acct-admin-dhamoirhat', 'acct-admin-agradigun', 'acct-admin-alampur', 'acct-admin-umar',
     'acct-admin-aranagar', 'acct-admin-jahanpur', 'acct-admin-isabpur', 'acct-admin-khelna',
     'acct-admin-pourashava');

INSERT INTO accounts (id, name, phone, password_hash, role, nid, union_id, verified, official_id) VALUES
    ('acct-seed-admin-1', 'আব্দুল হক',      '01710000003', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-3'),
    ('acct-seed-admin-2', 'নূরুল আমিন',    '01710000006', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-6'),
    ('acct-seed-admin-3', 'শাহানা পারভীন', '01710000007', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-7'),
    ('acct-seed-admin-4', 'জামাল হোসেন',   '01710000008', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-8')
ON CONFLICT (id) DO NOTHING;

-- 4. The Dhamoirhat directory and geography go, children first.
DELETE FROM officials WHERE id IN
    ('off-mp', 'off-upz-chair', 'off-upz-vice', 'off-upz-wvice', 'off-mayor',
     'off-chair-dhamoirhat', 'off-chair-agradigun', 'off-chair-alampur', 'off-chair-umar',
     'off-chair-aranagar', 'off-chair-jahanpur', 'off-chair-isabpur', 'off-chair-khelna');

DELETE FROM areas WHERE id IN
    ('union-dhamoirhat', 'union-agradigun', 'union-alampur', 'union-umar',
     'union-aranagar', 'union-jahanpur', 'union-isabpur', 'union-khelna',
     'pourashava-dhamoirhat');
DELETE FROM areas WHERE id = 'upazila-dhamoirhat';
DELETE FROM areas WHERE id = 'seat-naogaon-2';
