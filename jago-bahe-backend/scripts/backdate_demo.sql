-- Step 3 of the demo pipeline: shift the freshly seeded board into the past.
--
-- seed_demo.mjs drives the real HTTP API, so every row it creates is stamped with
-- the instant the seeder ran. A feed where two dozen reports share one timestamp
-- reads as obviously fabricated, and the date filters (A.5.12) have nothing to
-- filter. This pass gives each problem one deterministic delta and moves the
-- problem AND EVERYTHING HANGING OFF IT by that same amount, so relative ordering
-- inside a case -- report, votes, assign, plan, updates, evidence, confirm -- is
-- preserved exactly. Nothing is reordered; the whole story just happened earlier.
--
-- IT DELETES NOTHING, despite running through cmd/reset -- that command is only
-- the multi-statement SQL runner (the extended protocol will not take a script).
-- Its dry-run counts PROBLEMS, so it reports a number that is irrelevant here.
--
--     go run ./cmd/reset -file scripts/backdate_demo.sql confirm
--   or
--     make backdate-demo
--
-- Re-running it shifts the board FURTHER back, by the same deltas again. That is
-- harmless but cumulative -- run it once per seed.

BEGIN;

-- One delta per problem, derived from the id so the result is stable across runs
-- and independent of insertion order.
--
-- The size of the delta depends on what state the report is in, because
-- cases.deadline is NOT shifted (see step 3) and an arbitrary delta would put the
-- two out of step:
--
--   * A report with OPEN work carries a deadline of roughly now + D. Aging it two
--     months would read as "reported in June, first response due next week".
--     Those stay recent, which is also how a real board looks: what is still being
--     worked on came in lately.
--   * A report whose case was assigned with a DELIBERATELY PAST deadline (the
--     silent-official scenario) must be older than that deadline, or the case
--     would be due before it was filed.
--   * Everything else -- resolved, done, rejected, withdrawn, never assigned --
--     is free to be old, and is what gives the feed its two-month spread.
CREATE TEMP TABLE shift ON COMMIT DROP AS
SELECT p.id AS problem_id,
       CASE
         WHEN c.deadline < now() THEN
           make_interval(days => 14 + (abs(hashtext(p.id)) % 7),  hours => (abs(hashtext(p.id)) % 11))
         WHEN c.status IN ('Assigned', 'Acknowledged', 'Planned', 'InProgress', 'Reopened', 'Blocked') THEN
           make_interval(days => (abs(hashtext(p.id)) % 9),       hours => (abs(hashtext(p.id)) % 11))
         ELSE
           make_interval(days => 14 + (abs(hashtext(p.id)) % 42), hours => (abs(hashtext(p.id)) % 11))
       END AS delta
  FROM problems p
  LEFT JOIN cases c ON c.problem_id = p.id;

CREATE INDEX ON shift (problem_id);

-- Resolve the child tables that hang off a case or a suggestion back to their
-- problem once, rather than repeating the join in every UPDATE below.
-- `resp` is a SECOND, forward shift applied to the acknowledgement and to
-- everything that follows it. The seeder acknowledges a case within seconds of
-- assigning it, and a uniform backward shift preserves that gap exactly -- so
-- every scorecard's "গড় সাড়াদানের সময়" tile reads ০ দিন, which looks like a broken
-- query rather than a fast official. Pushing the acknowledgement one to five days
-- forward of the assignment gives that tile a real number.
--
-- It must be applied to the whole post-acknowledgement tail, not to
-- acknowledged_at alone, or a plan ends up dated BEFORE the acknowledgement that
-- legally precedes it (Transition: Assigned -> Acknowledged -> Planned).
CREATE TEMP TABLE case_shift ON COMMIT DROP AS
SELECT c.id AS case_id, s.delta,
       make_interval(days  => 1 + (abs(hashtext(c.id)) % 5),
                     hours => (abs(hashtext(c.id)) % 7)) AS resp
  FROM cases c JOIN shift s ON s.problem_id = c.problem_id;

CREATE TEMP TABLE plan_shift ON COMMIT DROP AS
SELECT p.id AS plan_id, cs.delta, cs.resp FROM plans p JOIN case_shift cs ON cs.case_id = p.case_id;

CREATE TEMP TABLE obstacle_shift ON COMMIT DROP AS
SELECT o.id AS obstacle_id, cs.delta, cs.resp FROM obstacles o JOIN case_shift cs ON cs.case_id = o.case_id;

CREATE TEMP TABLE suggestion_shift ON COMMIT DROP AS
SELECT sg.id AS suggestion_id, s.delta FROM suggestions sg JOIN shift s ON s.problem_id = sg.problem_id;

CREATE TEMP TABLE unblocking_shift ON COMMIT DROP AS
SELECT up.id AS plan_id, os.delta, os.resp FROM unblocking_plans up JOIN obstacle_shift os ON os.obstacle_id = up.obstacle_id;

-- 1. The problem row itself, and the screening decision on it.
UPDATE problems p SET created_at = p.created_at - s.delta,
                      reviewed_at = p.reviewed_at - s.delta
  FROM shift s WHERE s.problem_id = p.id;

-- 2. Everything keyed directly on the problem.
UPDATE validation_votes v      SET created_at = v.created_at - s.delta FROM shift s WHERE s.problem_id = v.problem_id;
UPDATE suggestions sg          SET created_at = sg.created_at - s.delta FROM shift s WHERE s.problem_id = sg.problem_id;
UPDATE assignments a           SET created_at = a.created_at - s.delta FROM shift s WHERE s.problem_id = a.problem_id;
UPDATE forwarding_suggestions f SET created_at = f.created_at - s.delta FROM shift s WHERE s.problem_id = f.problem_id;
UPDATE notifications n         SET created_at = n.created_at - s.delta FROM shift s WHERE s.problem_id = n.problem_id;

-- The resident's confirmation is the LAST thing that happens to a case, so it
-- carries the response offset like the rest of the post-acknowledgement tail.
UPDATE confirmations cf SET created_at = cf.created_at - cs.delta + cs.resp
  FROM cases c JOIN case_shift cs ON cs.case_id = c.id WHERE c.problem_id = cf.problem_id;

-- 3. The case and its children.
--
--    An OPEN case's deadline is DELIBERATELY NOT SHIFTED, and this is the line a
--    later reader will "tidy" into consistency. The deadline is the escalation
--    anchor (A.3.7): the lateness clock measures from it and the silence clock
--    re-anchors on it. Dragging every open case's deadline into the past would
--    make the next worker tick escalate the whole board, and the পর্যবেক্ষণ page
--    would show a dozen flagged cases instead of the one official who is actually
--    silent -- which is the entire point of that screen. The `shift` table above
--    keeps open reports RECENT instead, so their untouched deadlines stay
--    coherent with the date the report shows.
UPDATE cases c SET created_at = c.created_at - cs.delta,
                   acknowledged_at = c.acknowledged_at - cs.delta + cs.resp
  FROM case_shift cs WHERE cs.case_id = c.id;

-- A FINISHED case is different, and safe: IsEscalatable excludes Done, Resolved
-- and Disputed, so nothing reads their deadlines and they can move. Without this,
-- a case resolved six weeks ago still advertises a deadline in the future.
-- Reopened and Blocked are deliberately NOT in this list -- both are still open,
-- still escalatable, and still anchored on the deadline.
UPDATE cases c SET deadline = c.deadline - cs.delta
  FROM case_shift cs
 WHERE cs.case_id = c.id AND c.status IN ('Done', 'Resolved', 'Disputed');

UPDATE assignments a SET deadline = a.deadline - cs.delta
  FROM cases c JOIN case_shift cs ON cs.case_id = c.id
 WHERE c.problem_id = a.problem_id AND c.status IN ('Done', 'Resolved', 'Disputed');

UPDATE plans pl            SET created_at = pl.created_at - cs.delta + cs.resp FROM case_shift cs WHERE cs.case_id = pl.case_id;
UPDATE progress_updates pu SET created_at = pu.created_at - cs.delta + cs.resp FROM case_shift cs WHERE cs.case_id = pu.case_id;
UPDATE evidence e          SET created_at = e.created_at - cs.delta + cs.resp FROM case_shift cs WHERE cs.case_id = e.case_id;

UPDATE obstacles o SET created_at    = o.created_at - cs.delta + cs.resp,
                       adjudicated_at = o.adjudicated_at - cs.delta + cs.resp,
                       resolved_at    = o.resolved_at - cs.delta + cs.resp
  FROM case_shift cs WHERE cs.case_id = o.case_id;

-- plan_tasks hangs off plans, not cases -- both timestamps move, or a week ticked
-- off would appear to have been completed before the plan was written.
UPDATE plan_tasks t SET created_at   = t.created_at - ps.delta + ps.resp,
                        completed_at = t.completed_at - ps.delta + ps.resp
  FROM plan_shift ps WHERE ps.plan_id = t.plan_id;

-- 4. The public's judgment of an obstacle -- post-acknowledgement, so it carries
--    the response offset too.
UPDATE obstacle_votes ov       SET created_at = ov.created_at - os.delta + os.resp FROM obstacle_shift os WHERE os.obstacle_id = ov.obstacle_id;
UPDATE unblocking_plans up     SET created_at = up.created_at - os.delta + os.resp FROM obstacle_shift os WHERE os.obstacle_id = up.obstacle_id;
UPDATE unblocking_plan_votes v SET created_at = v.created_at - us.delta + us.resp FROM unblocking_shift us WHERE us.plan_id = v.plan_id;

-- 5. Suggestion upvotes.
UPDATE suggestion_votes sv SET created_at = sv.created_at - ss.delta FROM suggestion_shift ss WHERE ss.suggestion_id = sv.suggestion_id;

-- 6. The audit trail moves WITH its problem, or the public timeline on
--    /problems/{id} reads backwards: entries stamped today under a report filed
--    six weeks ago. audit_entries has no foreign key -- target_id is polymorphic
--    TEXT -- so the join is on the value, filtered by target_type.
UPDATE audit_entries ae SET created_at = ae.created_at - s.delta
  FROM shift s WHERE ae.target_type = 'problem' AND ae.target_id = s.problem_id;

-- ...and the entries that record post-acknowledgement events take the response
-- offset on top, so the public trail agrees with the case's own timestamps. The
-- action list is ENUMERATED rather than derived: a new lifecycle action added
-- later will simply keep the earlier timestamp here, which is a visible date on a
-- demo board -- far better than a WHERE NOT IN that silently sweeps the screening
-- actions (`approved`, `rejected`) forward past the acknowledgement they precede.
UPDATE audit_entries ae SET created_at = ae.created_at + cs.resp
  FROM cases c JOIN case_shift cs ON cs.case_id = c.id
 WHERE ae.target_type = 'problem' AND ae.target_id = c.problem_id
   AND ae.action IN ('acknowledged', 'plan_submitted', 'plan_revised', 'progress_update',
                     'task_completed', 'evidence_uploaded', 'blocked', 'obstacle_vote',
                     'unblocking_plan_proposed', 'blocker_confirmed', 'blocker_denied',
                     'marked_done', 'confirmed_resolved', 'confirmed_reopened');

-- case_observations and observation_notes are deliberately absent: no observation
-- exists yet at this point in the pipeline. The worker opens them in step 4,
-- AFTER this runs, and stamps them with the real now -- which is correct, because
-- an observation is a fact about the present silence, not about the report's past.

COMMIT;
