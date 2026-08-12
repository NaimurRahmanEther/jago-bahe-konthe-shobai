-- Reverting B11 removes the super admin role from the code, so any account
-- holding it would fail the reverted CHECK. Remove those accounts first.
DELETE FROM accounts WHERE role = 'super_admin';

DROP TABLE IF EXISTS official_claims;

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_role_check;
ALTER TABLE accounts ADD CONSTRAINT accounts_role_check
    CHECK (role IN ('resident', 'official', 'admin'));

-- Note: accounts.official_id links created by approved claims are intentionally
-- left in place. They record real people holding real offices; dropping them
-- would lock verified officials out of their own cases, which is worse than
-- keeping a link whose claim history is gone.
