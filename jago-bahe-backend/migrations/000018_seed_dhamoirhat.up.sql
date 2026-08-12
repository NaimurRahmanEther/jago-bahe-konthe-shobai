-- Re-targets the platform from the invented pilot seat to a real one:
-- Naogaon-2 (seat 47) -> Dhamoirhat Upazila -> 8 union parishads + 1 pourashava.
-- Source: Dhamoirhat_Representatives.md at the repo root.
--
-- THE POURASHAVA IS SEEDED AT level='union', DELIBERATELY. areas.level is
-- constrained to ('seat','upazila','union','ward') by 000001, and a municipality
-- has no level of its own. Adding one would mean changing that CHECK plus
-- UnionOfArea, upazilaOf, MonitorFor and every union-scope guard that walks the
-- tree. The tier vocabulary already treats a mayor as union-level (tier.go's
-- IsUnionLevel), so seeding the pourashava as a union node makes the geography
-- agree with the routing that was already written. The word "union" is doing
-- double duty here; nothing else is.
--
-- Wards are NOT seeded. The source names no ward members, and a resident now
-- picks their union or the pourashava; the ward lives in the free-text address.

-- 1. Wipe the demo data first, so the old areas and officials can be deleted at
--    the end. problems cascades to validation votes, suggestions, assignments,
--    admin votes, cases, plans, updates, evidence and blockers (000006-000010).
--    audit_entries has NO foreign key -- target_id is polymorphic TEXT -- so
--    nothing removes those for us.
DELETE FROM problems;
DELETE FROM audit_entries;

-- 2. The new geography. One seat, one upazila, eight unions, one pourashava.
INSERT INTO areas (id, name, level, parent_id) VALUES
    ('seat-naogaon-2',        'নওগাঁ-২',            'seat',    NULL),
    ('upazila-dhamoirhat',    'ধামইরহাট উপজেলা',     'upazila', 'seat-naogaon-2'),
    ('union-dhamoirhat',      'ধামইরহাট ইউনিয়ন',    'union',   'upazila-dhamoirhat'),
    ('union-agradigun',       'আগ্রাদ্বিগুন ইউনিয়ন',  'union',   'upazila-dhamoirhat'),
    ('union-alampur',         'আলমপুর ইউনিয়ন',      'union',   'upazila-dhamoirhat'),
    ('union-umar',            'উমার ইউনিয়ন',        'union',   'upazila-dhamoirhat'),
    ('union-aranagar',        'আড়ানগর ইউনিয়ন',      'union',   'upazila-dhamoirhat'),
    ('union-jahanpur',        'জাহানপুর ইউনিয়ন',     'union',   'upazila-dhamoirhat'),
    ('union-isabpur',         'ঈসবপুর ইউনিয়ন',      'union',   'upazila-dhamoirhat'),
    ('union-khelna',          'খেলনা ইউনিয়ন',       'union',   'upazila-dhamoirhat'),
    ('pourashava-dhamoirhat', 'ধামইরহাট পৌরসভা',     'union',   'upazila-dhamoirhat')
ON CONFLICT (id) DO NOTHING;

-- 3. The directory. Every office exists so it can be pointed at and claimed.
--
--    NAMES: the MP, the upazila chairman, the vice chairman and the mayor are the
--    real names the source confirms. THE OTHER NINE ARE PLACEHOLDERS -- invented
--    dev names standing in for offices whose holder the source could not confirm
--    (the eight union chairmen and the upazila's woman vice chairman). They are
--    marked below. REPLACE THEM BEFORE ANY REAL DEPLOYMENT: a civic platform that
--    prints a fabricated name against a real public office is publishing a false
--    claim about whoever actually holds it. In the designed flow the real holder
--    attaches to the office by claiming it (B11), which is what fixes the name.
--
--    Phone numbers are dev placeholders. These are directory contact numbers,
--    not login credentials; the seed logins live in accounts below.
--
--    There is no separate tier for the upazila's woman vice chairman, so that
--    post shares 'upazila_vice_chairman'.
INSERT INTO officials (id, name, phone, tier, area_id) VALUES
    -- Confirmed by the source.
    ('off-mp',              'শহীদুজ্জামান সরকার',    '01800000001', 'mp',                    'seat-naogaon-2'),
    ('off-upz-chair',       'আজহার আলী',            '01800000002', 'upazila_chairman',      'upazila-dhamoirhat'),
    ('off-upz-vice',        'সোহেল রানা',           '01800000003', 'upazila_vice_chairman', 'upazila-dhamoirhat'),
    ('off-mayor',           'মো. আমিনুর রহমান',     '01800000005', 'pourashava_mayor',      'pourashava-dhamoirhat'),
    -- PLACEHOLDER NAMES -- invented, not the real officeholders. See the note above.
    ('off-upz-wvice',       'মোছা. রোকেয়া বেগম',    '01800000004', 'upazila_vice_chairman', 'upazila-dhamoirhat'),
    ('off-chair-dhamoirhat','মো. শাহাদত হোসেন',     '01800000011', 'union_chairman',        'union-dhamoirhat'),
    ('off-chair-agradigun', 'মো. আনোয়ার হোসেন',     '01800000012', 'union_chairman',        'union-agradigun'),
    ('off-chair-alampur',   'মো. রেজাউল করিম',      '01800000013', 'union_chairman',        'union-alampur'),
    ('off-chair-umar',      'মো. মিজানুর রহমান',    '01800000014', 'union_chairman',        'union-umar'),
    ('off-chair-aranagar',  'মো. আব্দুল মালেক',      '01800000015', 'union_chairman',        'union-aranagar'),
    ('off-chair-jahanpur',  'মো. নাজমুল হক',        '01800000016', 'union_chairman',        'union-jahanpur'),
    ('off-chair-isabpur',   'মো. সাইফুল ইসলাম',     '01800000017', 'union_chairman',        'union-isabpur'),
    ('off-chair-khelna',    'মো. জাহাঙ্গীর আলম',     '01800000018', 'union_chairman',        'union-khelna')
ON CONFLICT (id) DO NOTHING;

-- 4. Move the two non-admin seed accounts onto the new geography, keeping their
--    credentials so information.md's logins still work.
UPDATE accounts SET union_id = 'union-dhamoirhat' WHERE id = 'acct-seed-res-1';
UPDATE accounts SET official_id = 'off-mayor'     WHERE id = 'acct-seed-off-1';

-- The approved claim 000014 backfilled for that account points at off-1, and
-- official_claims.official_id has NO ON DELETE clause. Without this the delete
-- at step 8 fails on a foreign key.
UPDATE official_claims SET official_id = 'off-mayor' WHERE account_id = 'acct-seed-off-1';

-- 5. Any account created at runtime against the old geography: a resident keeps
--    their account but lands in the pilot's own union, and an official whose
--    claimed office no longer exists drops back to the claim-pending state the
--    UI already renders (OfficialDashboard reads a null official_id as "your
--    claim is still being reviewed", not as "you have no work").
UPDATE accounts SET union_id = 'union-dhamoirhat'
    WHERE union_id IN ('union-1', 'union-2', 'union-3', 'union-4', 'ward-1', 'ward-2', 'upazila-1', 'upazila-2', 'seat-1');

DELETE FROM official_claims WHERE official_id IN
    ('off-1', 'off-2', 'off-3', 'off-4', 'off-5', 'off-6', 'off-7', 'off-8', 'off-9');

UPDATE accounts SET official_id = NULL
    WHERE official_id IN ('off-1', 'off-2', 'off-3', 'off-4', 'off-5', 'off-6', 'off-7', 'off-8', 'off-9');

-- 6. Replace the four pilot admins with one per local unit. Nine admins means an
--    above-union vote has a real quorum (Q = 0.5 -> five of nine must take part).
--    Deleting them cascades their official_claims rows (000014 ON DELETE CASCADE).
DELETE FROM accounts WHERE id IN ('acct-seed-admin-1', 'acct-seed-admin-2', 'acct-seed-admin-3', 'acct-seed-admin-4');

-- An admin's union is derived from their bound official's area_id, not from
-- accounts.union_id (which stays NULL for admins) -- so the binding below is a
-- scoping device, not a claim to office. The pourashava's admin is bound to the
-- mayor, whom acct-seed-off-1 also points at: accounts.official_id has no unique
-- constraint, and uq_official_claims_approved constrains claims rather than
-- bindings, of which there is still exactly one.
--
-- All nine share the dev password `admin123` (same bcrypt hash as the earlier
-- admin seeds). Phones are renumbered into an 019 series because the old
-- convention reused the bound official's phone, and those officials are gone.
INSERT INTO accounts (id, name, phone, password_hash, role, nid, union_id, verified, official_id) VALUES
    ('acct-admin-dhamoirhat', 'প্রশাসক — ধামইরহাট ইউনিয়ন',  '01910000001', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-chair-dhamoirhat'),
    ('acct-admin-agradigun',  'প্রশাসক — আগ্রাদ্বিগুন ইউনিয়ন', '01910000002', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-chair-agradigun'),
    ('acct-admin-alampur',    'প্রশাসক — আলমপুর ইউনিয়ন',    '01910000003', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-chair-alampur'),
    ('acct-admin-umar',       'প্রশাসক — উমার ইউনিয়ন',      '01910000004', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-chair-umar'),
    ('acct-admin-aranagar',   'প্রশাসক — আড়ানগর ইউনিয়ন',    '01910000005', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-chair-aranagar'),
    ('acct-admin-jahanpur',   'প্রশাসক — জাহানপুর ইউনিয়ন',   '01910000006', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-chair-jahanpur'),
    ('acct-admin-isabpur',    'প্রশাসক — ঈসবপুর ইউনিয়ন',    '01910000007', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-chair-isabpur'),
    ('acct-admin-khelna',     'প্রশাসক — খেলনা ইউনিয়ন',     '01910000008', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-chair-khelna'),
    ('acct-admin-pourashava', 'প্রশাসক — ধামইরহাট পৌরসভা',  '01910000009', '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'admin', NULL, NULL, true, 'off-mayor')
ON CONFLICT (id) DO NOTHING;

-- 7. The invented directory goes.
DELETE FROM officials WHERE id IN
    ('off-1', 'off-2', 'off-3', 'off-4', 'off-5', 'off-6', 'off-7', 'off-8', 'off-9');

-- 8. The invented geography goes, children first.
DELETE FROM areas WHERE id IN ('ward-1', 'ward-2');
DELETE FROM areas WHERE id IN ('union-1', 'union-2', 'union-3', 'union-4');
DELETE FROM areas WHERE id IN ('upazila-1', 'upazila-2');
DELETE FROM areas WHERE id = 'seat-1';
