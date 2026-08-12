-- B11: official claims, the super admin, and the link that lets a real person act
-- as a real official.
--
-- Before this, the identity context could only grow residents: the officials
-- directory port was read-only and accounts.official_id was written by no runtime
-- path, so every official and admin came from a seed migration.

-- The super admin is a fourth role, not a rank above admin. They verify
-- above-union claims and read every admin decision; they cannot screen, assign,
-- or vote (RequireAdmin rejects them). See identity/domain/role.go.
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_role_check;
ALTER TABLE accounts ADD CONSTRAINT accounts_role_check
    CHECK (role IN ('resident', 'official', 'admin', 'super_admin'));

-- A claim is a person's assertion that they hold a directory office. The
-- directory records real election results, so a claim never creates an office —
-- it links an account to one that already exists.
CREATE TABLE official_claims (
    id          TEXT PRIMARY KEY,
    account_id  TEXT        NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    official_id TEXT        NOT NULL REFERENCES officials (id),
    status      TEXT        NOT NULL DEFAULT 'Pending' CHECK (status IN ('Pending', 'Approved', 'Rejected')),
    reviewed_by TEXT,
    reviewed_at TIMESTAMPTZ,
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- One claim per account: a person claims one office, and a rejected claimant
    -- does not get to keep guessing at other seats from the same account.
    UNIQUE (account_id)
);

-- The rule that matters: at most one APPROVED claim per office. Two people can
-- never both be the ward member. Rejected and pending claims are unconstrained,
-- so several people may claim a seat and be sorted out by the approver — but the
-- database, not an admin's memory, is what makes the winner unique.
CREATE UNIQUE INDEX uq_official_claims_approved ON official_claims (official_id)
    WHERE status = 'Approved';

CREATE INDEX idx_official_claims_status ON official_claims (status, created_at);

COMMENT ON COLUMN official_claims.reason IS 'required on rejection; shown to the claimant so a refusal is never silent';
COMMENT ON COLUMN official_claims.reviewed_by IS 'the deciding admin or super_admin account id';

-- Seeded OFFICIALS already hold their offices; record that as an approved claim
-- so the directory link has one explanation rather than two.
--
-- Admins are deliberately excluded even though they also carry an official_id.
-- For them that column is a scoping device, not a claim to office: an admin is
-- bound to an official only so their union can be derived (accounts.union_id is
-- NULL for admins). Seeding their link as an approved claim would mark those
-- offices taken and permanently lock the real ward member or chairman out of
-- claiming their own seat.
INSERT INTO official_claims (id, account_id, official_id, status, reviewed_by, reviewed_at)
SELECT 'claim-seed-' || a.id, a.id, a.official_id, 'Approved', 'system', now()
FROM accounts a
WHERE a.official_id IS NOT NULL AND a.role = 'official'
ON CONFLICT DO NOTHING;

-- One seat-level super admin for the pilot. Same dev password as the admin seeds
-- (`admin123`). They carry no official_id: the super admin is not a nominated
-- person and holds no office.
INSERT INTO accounts (id, name, phone, password_hash, role, nid, union_id, verified, official_id) VALUES
    ('acct-seed-super-1', 'সুপার অ্যাডমিন', '01710000010',
     '$2a$10$zNNRl8JPvI6EAbW7ZIGEOOy./bFud7f.4rhikKMyLHXy3NAD3DxGK', 'super_admin', NULL, NULL, true, NULL)
ON CONFLICT (id) DO NOTHING;
