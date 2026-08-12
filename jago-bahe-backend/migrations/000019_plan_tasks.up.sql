-- Plan tasks: an official's plan read as a week-by-week checklist, one task per
-- week. week_number is 1-based and matches the task's position in the plan.
-- completed/completed_at are the only mutable columns — the official checks a week
-- off in public as it is finished. plans.timeline_weeks stays as the derived count.
CREATE TABLE plan_tasks (
    id           TEXT        PRIMARY KEY,
    plan_id      TEXT        NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    week_number  INTEGER     NOT NULL,
    task         TEXT        NOT NULL,
    completed    BOOLEAN     NOT NULL DEFAULT false,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_plan_tasks_plan ON plan_tasks (plan_id);
