// @vitest-environment node
// Node env (not jsdom) so axios uses the Node http adapter and reaches the
// backend directly, without the browser same-origin/CORS restriction.
import { describe, it, expect } from 'vitest'
import { login, register, registerOfficial } from './auth.js'
import {
  listProblems,
  listMyProblems,
  reportProblem,
  getProblem,
  updateProblem,
  withdrawProblem,
  deleteProblem,
  validateProblem,
} from './problems.js'
import { listOfficials } from './officials.js'
import { approveProblem } from './moderation.js'
import {
  listQueue,
  assignWithinUnion,
  listMyForwarding,
  suggestForwarding,
  listForwardingQueue,
  forwardProblem,
} from './assignments.js'
import {
  listNotifications,
  unreadNotificationCount,
  markNotificationRead,
  markAllNotificationsRead,
} from './notifications.js'
import { listSuggestions, proposeSuggestion, upvoteSuggestion } from './suggestions.js'
import { listAreas } from './areas.js'
import { listActivity } from './activity.js'
import { getOfficialScorecard, getSeatOverview } from './scorecard.js'
import {
  getProgress,
  listCases,
  acknowledgeCase,
  submitPlan,
  completeTask,
  reportObstacle,
  postUpdate,
  revisePlan,
} from './resolution.js'
import { listPendingClaims, listPendingResidents, verifyResident, getOversight } from './identity.js'
import { listObservations, noteObservation } from './observation.js'
import { setAuthToken } from './client.js'

// Opt-in live integration: drives the REAL frontend api modules (axios client +
// interceptors) against a running backend. This is the ONLY proof that the two
// halves agree — every other test stubs the api modules, so a shape drift is
// invisible to them (CLAUDE.md A.5.2, A.5.11). Skipped unless VITE_LIVE=1, so
// the normal `npm test` never needs a server. Run it live with:
//   VITE_LIVE=1 VITE_API_URL=http://localhost:8080/api npx vitest run live.integration
const LIVE = process.env.VITE_LIVE === '1'

// Captured by the login test and read by the ones below; tests in a file run in
// order, so the token/id are set before anything needs them.
let residentId = ''
// Reused rather than re-fetched: auth routes are rate limited per IP
// (AUTH_RATE_LIMIT, 20/min), and this file is one client hitting one endpoint, so
// a fresh login per step is what tips the whole suite into 429s.
let residentToken = ''

describe.skipIf(!LIVE)('live API integration (real backend)', () => {
  it('logs in a seeded resident → { token, role, user }', async () => {
    const res = await login({ phone: '01810000001', password: 'resident123' })
    expect(res.token).toBeTruthy()
    expect(res.role).toBe('resident')
    expect(res.user).toEqual(
      expect.objectContaining({ id: expect.any(String), phone: '01810000001', verified: true }),
    )
    setAuthToken(res.token)
    residentId = res.user.id
    residentToken = res.token
  })

  // The bug this pins: the number is stored canonically at registration, and the
  // next login arrives reformatted by browser autofill. Same person, same
  // number, same account — proven against real Postgres, since the backend is
  // the authority on normalization and lib/phone.js only mirrors it.
  it('logs in with the same number spelled the way autofill returns it', async () => {
    const res = await login({ phone: '+880 1810-000001', password: 'resident123' })
    expect(res.token).toBeTruthy()
    expect(res.user.id).toBe(residentId)
    expect(res.user.phone).toBe('01810000001')
  })

  it('rejects a malformed phone as invalid_phone, never as bad credentials', async () => {
    // 400, not 401: a malformed string cannot be anyone's account, so saying so
    // leaks nothing — and calling it a credential failure told the user to doubt
    // a password that was correct.
    await expect(login({ phone: 'not-a-phone', password: 'resident123' })).rejects.toMatchObject({
      status: 400,
    })
  })

  it('lists officials with the Official shape', async () => {
    const officials = await listOfficials()
    expect(Array.isArray(officials)).toBe(true)
    expect(officials[0]).toEqual(
      expect.objectContaining({
        id: expect.any(String),
        name: expect.any(String),
        tier: expect.any(String),
        areaId: expect.any(String),
      }),
    )
  })

  it('lists problems as an array', async () => {
    const problems = await listProblems()
    expect(Array.isArray(problems)).toBe(true)
  })

  it('returns an official scorecard with the OfficialStats shape', async () => {
    const stats = await getOfficialScorecard('off-chair-dhamoirhat')
    expect(stats).toEqual(
      expect.objectContaining({
        officialId: 'off-chair-dhamoirhat',
        resolved: expect.any(Number),
        pending: expect.any(Number),
        blocked: expect.any(Number),
        avgResponseDays: expect.any(Number),
      }),
    )
  })

  // --- The pre-publication gate ---

  // The whole point of the design, proven end to end against the real backend: a
  // resident files a report and it is HIDDEN — not in the anonymous feed, 404 to a
  // stranger — until its union admin approves it into public view. The api layer is
  // what a reviewer would trust here, so it is what drives it; the backend's own
  // integration test proves the same rule against Postgres, and these must not be
  // allowed to disagree.
  it('holds a new report hidden until its union admin approves it', async () => {
    const filed = await reportProblem({
      title: `live check: gate ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'union-dhamoirhat', address: 'live test' },
      pointedOfficialId: 'off-chair-dhamoirhat',
    })
    expect(filed.status).toBe('PendingApproval')

    // Anonymously — no token at all. A pending report 404s for a stranger (not a
    // 403), so no one can confirm a hidden report exists at this id, and it is
    // absent from the feed.
    setAuthToken(null)
    await expect(getProblem(filed.id)).rejects.toMatchObject({ status: 404 })
    const hidden = await listProblems()
    expect(hidden.map((p) => p.id)).not.toContain(filed.id)

    // Dhamoirhat union's admin approves it → Reported, and only now is it public.
    const admin = await login({ phone: '01910000001', password: 'admin123' })
    setAuthToken(admin.token)
    const approved = await approveProblem(filed.id)
    expect(approved.status).toBe('Reported')

    setAuthToken(null)
    const anon = await getProblem(filed.id)
    expect(anon.id).toBe(filed.id)
    expect(anon.status).toBe('Reported')
    const feed = await listProblems()
    expect(feed.map((p) => p.id)).toContain(filed.id)

    const res = await login({ phone: '01810000001', password: 'resident123' })
    setAuthToken(res.token)
  })

  // --- B17: the admin's queue, and forwarding a report before it reaches V ---
  //
  // The queue endpoint had never been driven from the frontend at all, and its DTO
  // and its own UI had silently disagreed about four fields the whole time — the
  // A.5.2 failure mode exactly. This is the test that would have caught it.
  it('lists the admin’s own union queue in the shape the UI reads', async () => {
    const filed = await reportProblem({
      title: `live check: queue ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'union-dhamoirhat', address: 'live queue test' },
      pointedOfficialId: 'off-chair-dhamoirhat',
    })

    const admin = await login({ phone: '01910000001', password: 'admin123' })
    setAuthToken(admin.token)
    await approveProblem(filed.id)

    // Approved and unvalidated, it is in the assignment queue at once. Before B17
    // the queue was Validated-only, so this row simply would not be here.
    const queue = await listQueue()
    const row = queue.find((q) => q.problemId === filed.id)
    expect(row).toBeDefined()
    expect(row.status).toBe('Reported')

    // The four fields the UI reads and the API did not send. `problemId` rather
    // than `id` is the shape itself — asserting `row.id` here would pass on
    // undefined, which is precisely how the drift survived.
    expect(row.address).toBeTruthy()
    expect(row.routing).toBe('union')
    expect(row.validCount).toBe(0)
    expect(row.validationThreshold).toBeGreaterThan(0)

    // Scoped to the caller's own union. Agradigun's admin must not see a Dhamoirhat union
    // problem: the route used to ignore the caller entirely and hand every admin
    // every union's rows.
    const other = await login({ phone: '01910000002', password: 'admin123' })
    setAuthToken(other.token)
    const otherQueue = await listQueue()
    expect(otherQueue.map((q) => q.problemId)).not.toContain(filed.id)

    // And the load-bearing rule: Dhamoirhat union's admin forwards it to an official with
    // ZERO validations. This was a 409 not_validated before B17 — the community's
    // votes are now evidence, not a condition (A.3.1).
    setAuthToken(admin.token)
    const assignment = await assignWithinUnion(filed.id, { officialId: 'off-chair-dhamoirhat', priority: 'normal' })
    expect(assignment.problemId).toBe(filed.id)

    setAuthToken(null)
    const after = await getProblem(filed.id)
    expect(after.status).toBe('Assigned')

    setAuthToken(residentToken)
  })

  // --- B20: above-union forwarding — the admins advise, the super admin decides ---
  //
  // The whole chain in one test, because the parts only mean anything together:
  // an above-union report leaves the union admin's ASSIGN queue, appears on their
  // ADVISORY list, collects advice from two admins, and is finally forwarded by the
  // super admin. Six endpoints the UI had no proof of.
  it('routes an above-union report through advice to the super admin’s decision', async () => {
    setAuthToken(residentToken)
    const filed = await reportProblem({
      title: `live check: forwarding ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'union-dhamoirhat', address: 'live forwarding test' },
      // An UPAZILA-tier office: above the union, so the super admin forwards it and
      // the admins of that upazila advise. A union chairman here would take the
      // ordinary assign path and prove nothing.
      pointedOfficialId: 'off-upz-chair',
    })

    const admin = await login({ phone: '01910000001', password: 'admin123' })
    setAuthToken(admin.token)
    await approveProblem(filed.id)

    // It is NOT on the assignment queue — assigning it would be refused, so it must
    // not be offered there (the leak B17 fixed at the union boundary).
    const queue = await listQueue()
    expect(queue.map((q) => q.problemId)).not.toContain(filed.id)

    // It IS on the advisory list, in the shape the UI reads.
    const advisory = await listMyForwarding()
    const row = advisory.find((r) => r.problemId === filed.id)
    expect(row).toBeDefined()
    expect(row.scope).toBe('upazila')
    expect(row.pointedOfficialId).toBe('off-upz-chair')
    expect(row.address).toBeTruthy()
    expect(row.suggestions).toEqual([])
    expect(row.topOfficialId).toBe('')
    // Present-and-null, never absent — the myVote lesson at the field level. Both
    // are falsy in JS, so a regression to "sometimes missing" would show up only as
    // a wrongly pre-filled select (A.3.2.1 rule 2).
    expect(row.mySuggestion).toBeNull()
    expect('mySuggestion' in row).toBe(true)

    await suggestForwarding(filed.id, { officialId: 'off-upz-chair', reason: 'upazila budget' })

    // Advice is scoped by UPAZILA, not by the admin's own union, so a neighbouring
    // union's admin both sees this report and may advise on it — though they saw
    // nothing of it in the assignment queue above. That difference is the point.
    const other = await login({ phone: '01910000002', password: 'admin123' })
    setAuthToken(other.token)
    expect((await listMyForwarding()).map((r) => r.problemId)).toContain(filed.id)
    await suggestForwarding(filed.id, { officialId: 'off-upz-chair' })

    const agreed = (await listMyForwarding()).find((r) => r.problemId === filed.id)
    expect(agreed.topOfficialId).toBe('off-upz-chair')
    expect(agreed.topCount).toBe(2)
    expect(agreed.mySuggestion).toMatchObject({ officialId: 'off-upz-chair' })

    // Advice is REVISABLE, unlike the ballot it replaced. Moving one of the two
    // makes it a TIE, and a tie has NO top: breaking it would invent a consensus
    // and make the reason requirement turn on a coin flip.
    await suggestForwarding(filed.id, { officialId: 'off-upz-vice' })
    const tied = (await listMyForwarding()).find((r) => r.problemId === filed.id)
    expect(tied.topOfficialId).toBe('')
    expect(tied.topCount).toBe(0)
    // Replaced, not appended — still two advisers, not three.
    expect(tied.suggestions).toHaveLength(2)

    // Roles are flat: an admin is refused the super admin's surface outright.
    await expect(listForwardingQueue()).rejects.toMatchObject({ status: 403 })

    const superAdmin = await login({ phone: '01710000010', password: 'admin123' })
    setAuthToken(superAdmin.token)
    const pending = await listForwardingQueue()
    expect(pending.map((r) => r.problemId)).toContain(filed.id)
    // The super admin advises on nothing, so their own row never carries advice.
    expect(pending.find((r) => r.problemId === filed.id).mySuggestion).toBeNull()

    // The advisers are tied, so only the REPORTER's choice still binds a reason.
    // Forwarding elsewhere without one is refused — by the backend, which owns the
    // rule the UI only mirrors (A.5.7).
    await expect(
      forwardProblem(filed.id, { officialId: 'off-upz-vice', priority: 'normal' }),
    ).rejects.toMatchObject({ status: 400, code: 'reason_required' })

    // To the reporter's own choice, no reason is owed.
    const assignment = await forwardProblem(filed.id, {
      officialId: 'off-upz-chair',
      priority: 'normal',
    })
    expect(assignment.problemId).toBe(filed.id)
    expect(assignment.officialId).toBe('off-upz-chair')

    setAuthToken(null)
    expect((await getProblem(filed.id)).status).toBe('Assigned')

    // Decided, it leaves the surface it was waiting on.
    setAuthToken(superAdmin.token)
    expect((await listForwardingQueue()).map((r) => r.problemId)).not.toContain(filed.id)

    setAuthToken(residentToken)
  })

  // PendingApproval is a real status again, but it is not publicly listable: the
  // backend refuses it against PublicStatuses(), so a UI offering it as a feed
  // filter would produce a 400 rather than leaking pending reports.
  it('refuses PendingApproval as a public feed filter', async () => {
    await expect(listProblems({ status: 'PendingApproval' })).rejects.toMatchObject({ status: 400 })
  })

  // --- The reporter edits, then withdraws, their own report ---
  //
  // A report is editable only once approved (Reported, no votes), and withdrawable
  // any time before assignment (including while pending). Driven through the real
  // axios client (PATCH + POST /approve + POST /withdraw) so a shape or verb drift
  // between the two halves is caught here, nowhere else.
  it('lets the reporter edit their own approved, unvalidated report', async () => {
    const filed = await reportProblem({
      title: `live check: edit me ${Date.now()}`,
      description: 'first version',
      location: { areaId: 'union-dhamoirhat', address: 'first address' },
      pointedOfficialId: 'off-chair-dhamoirhat',
    })
    // Approve it first — a pending report is not yet editable.
    const admin = await login({ phone: '01910000001', password: 'admin123' })
    setAuthToken(admin.token)
    await approveProblem(filed.id)
    const res = await login({ phone: '01810000001', password: 'resident123' })
    setAuthToken(res.token)

    const edited = await updateProblem(filed.id, {
      title: `${filed.title} (edited)`,
      description: 'second version',
      location: { areaId: 'union-dhamoirhat', address: 'second address' },
      proposedSolution: 'a fix',
    })
    expect(edited.title).toBe(`${filed.title} (edited)`)
    expect(edited.description).toBe('second version')

    const back = await getProblem(filed.id)
    expect(back.title).toBe(`${filed.title} (edited)`)
  })

  it('lets the reporter withdraw their own report → Withdrawn, still on the public record', async () => {
    const filed = await reportProblem({
      title: `live check: withdraw me ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'union-dhamoirhat', address: 'withdraw test' },
      pointedOfficialId: 'off-chair-dhamoirhat',
    })
    const withdrawn = await withdrawProblem(filed.id, 'duplicate report')
    expect(withdrawn.status).toBe('Withdrawn')

    // It does not vanish — anyone can still read it as Withdrawn.
    setAuthToken(null)
    const anon = await getProblem(filed.id)
    expect(anon.status).toBe('Withdrawn')
    const res = await login({ phone: '01810000001', password: 'resident123' })
    setAuthToken(res.token)
  })

  // myVote is what lets a feed row show a vote already cast instead of re-offering
  // the buttons. The assertion that matters is the negative one: it must be the
  // CALLER's own vote and null for everyone else — a myVote that leaked another
  // account's vote would be a privacy bug wearing the costume of a UI convenience.
  it('reports the caller’s own vote on a problem, and nobody else’s', async () => {
    const filed = await reportProblem({
      title: `live check: myVote ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'union-dhamoirhat', address: 'myVote test' },
      pointedOfficialId: 'off-chair-dhamoirhat',
    })
    const admin = await login({ phone: '01910000001', password: 'admin123' })
    setAuthToken(admin.token)
    await approveProblem(filed.id)

    const res = await login({ phone: '01810000001', password: 'resident123' })
    setAuthToken(res.token)

    // Before voting: present and null, never absent. An absent key and a null are
    // both falsy in JS, so "sometimes missing" would be invisible in the UI.
    const before = await getProblem(filed.id)
    expect(before).toHaveProperty('myVote')
    expect(before.myVote).toBeNull()

    const voted = await validateProblem(filed.id, 'valid')
    // The vote response echoes the vote just cast — without it the UI re-enables
    // the button it just used.
    expect(voted.myVote).toBe('valid')

    // It survives a fresh read, on both the detail and the feed. That is the
    // reload case the whole field exists for.
    expect((await getProblem(filed.id)).myVote).toBe('valid')
    const feed = await listProblems()
    expect(feed.find((p) => p.id === filed.id).myVote).toBe('valid')

    // Anonymously: present, and null.
    setAuthToken(null)
    const anon = await getProblem(filed.id)
    expect(anon).toHaveProperty('myVote')
    expect(anon.myVote).toBeNull()
    const anonRow = (await listProblems()).find((p) => p.id === filed.id)
    expect(anonRow.myVote).toBeNull()

    setAuthToken(res.token)
  })

  // The public record naming who is actually working on a report. Until this, the
  // backend served assignedOfficialId on the detail read alone and NO screen read
  // it — the feed showed what was reported and never who took it up, which kept the
  // accountable half of the record private.
  it('publishes the assigned official on the feed, to an anonymous reader', async () => {
    const filed = await reportProblem({
      title: `live check: assigned ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'union-dhamoirhat', address: 'assignment test' },
      pointedOfficialId: 'off-chair-dhamoirhat',
    })

    const admin = await login({ phone: '01910000001', password: 'admin123' })
    setAuthToken(admin.token)
    await approveProblem(filed.id)

    // Unassigned: the key is ABSENT, not empty-string. omitempty drops the zero
    // view, so "absent" and "not assigned" are one answer for the client.
    setAuthToken(null)
    expect(await getProblem(filed.id)).not.toHaveProperty('assignedOfficialId')

    // Override the public's nominee, which is the case that must carry a reason.
    setAuthToken(admin.token)
    await assignWithinUnion(filed.id, {
      officialId: 'off-mayor',
      priority: 'normal',
      overrideReason: 'এটি পৌরসভার এখতিয়ারভুক্ত',
    })

    // Anonymously, on the FEED — one batch query per page, not a per-row lookup.
    setAuthToken(null)
    const row = (await listProblems()).find((p) => p.id === filed.id)
    expect(row.assignedOfficialId).toBe('off-mayor')

    // And on the detail read, with the justification. Publishing the pointed →
    // assigned gap while withholding the reason would invite the reader to assume
    // the worst of a decision that is usually routine.
    const detail = await getProblem(filed.id)
    expect(detail.assignedOfficialId).toBe('off-mayor')
    expect(detail.pointedOfficialId).toBe('off-chair-dhamoirhat')
    expect(detail.overrideReason).toBe('এটি পৌরসভার এখতিয়ারভুক্ত')

    setAuthToken(residentToken)
  })

  // myUpvote is myVote's twin, one endpoint further along: without it a reload
  // re-offers the button on a suggestion the reader already upvoted, and the toggle
  // then silently WITHDRAWS it. Same privacy rule too — the counts are public, who
  // cast them is not.
  it('reports the caller’s own upvote on a suggestion, and nobody else’s', async () => {
    const filed = await reportProblem({
      title: `live check: myUpvote ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'union-dhamoirhat', address: 'myUpvote test' },
      pointedOfficialId: 'off-chair-dhamoirhat',
    })
    const admin = await login({ phone: '01910000001', password: 'admin123' })
    setAuthToken(admin.token)
    await approveProblem(filed.id)

    setAuthToken(residentToken)
    const proposed = await proposeSuggestion(filed.id, 'live check: একটি প্রস্তাব')

    // Before upvoting: present and null, never absent.
    const [before] = await listSuggestions(filed.id)
    expect(before).toHaveProperty('myUpvote')
    expect(before.myUpvote).toBeFalsy()

    // The toggle response says which way it went, so the caller never infers it
    // from a count other people are also moving.
    const toggled = await upvoteSuggestion(proposed.id)
    expect(toggled.myUpvote).toBe(true)

    // It survives a fresh read — the reload case the field exists for.
    const [after] = await listSuggestions(filed.id)
    expect(after.myUpvote).toBe(true)
    expect(after.upvoteCount).toBe(1)
    // One upvote makes it the top, which is what the official must answer.
    expect(after.isTop).toBe(true)

    // Anonymously: present, and never another account's upvote.
    setAuthToken(null)
    const [anon] = await listSuggestions(filed.id)
    expect(anon).toHaveProperty('myUpvote')
    expect(anon.myUpvote).toBeFalsy()
    expect(anon.upvoteCount).toBe(1)

    setAuthToken(residentToken)
  })

  it('lets the reporter delete their own report outright → gone, 404, off their own list', async () => {
    const filed = await reportProblem({
      title: `live check: delete me ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'union-dhamoirhat', address: 'delete test' },
      pointedOfficialId: 'off-chair-dhamoirhat',
    })
    // 204 No Content: the api function resolves with undefined, not a problem. If
    // this ever comes back as `''` the endpoint is writing an empty body where the
    // contract says none — the A.5.2 failure at the status-code level.
    await expect(deleteProblem(filed.id)).resolves.toBeUndefined()

    // Unlike withdraw above, nothing survives — not even for its own reporter.
    await expect(getProblem(filed.id)).rejects.toMatchObject({ status: 404 })
    const mine = await listMyProblems()
    expect(mine.some((p) => p.id === filed.id)).toBe(false)
  })

  // --- B12: the reporter's own view + public progress ---

  it('returns only the caller’s own reports from /me/problems, in the Problem shape', async () => {
    const mine = await listMyProblems()
    expect(Array.isArray(mine)).toBe(true)
    // Self-derived from the token — the endpoint takes no parameter at all, so a
    // row belonging to anyone else here would mean it is not scoping to the
    // caller. residentId comes from the login above, never from the request.
    for (const p of mine) {
      expect(p.reporterId).toBe(residentId)
    }
    if (mine.length > 0) {
      expect(mine[0]).toEqual(
        expect.objectContaining({
          id: expect.any(String),
          title: expect.any(String),
          location: expect.objectContaining({ areaId: expect.any(String) }),
          reporterId: expect.any(String),
          status: expect.any(String),
          validCount: expect.any(Number),
          validationThreshold: expect.any(Number),
          createdAt: expect.any(String),
        }),
      )
    }
  })

  it('refuses /me/problems without a token — 401, not an empty list', async () => {
    setAuthToken(null)
    await expect(listMyProblems()).rejects.toMatchObject({ status: 401 })
    const res = await login({ phone: '01810000001', password: 'resident123' })
    setAuthToken(res.token)
  })

  // The load-bearing one. The backend answers 200 + null for a problem with no
  // case; axios turns a 404 into a REJECTION, so if the null-shaped rule ever
  // regressed to a 404 this is the assertion that catches it client-side.
  it('resolves null (not a rejection) for a problem with no case', async () => {
    await expect(getProgress('prob-does-not-exist-at-all')).resolves.toBeNull()
  })

  it('returns the Progress shape for a problem whose case exists, without the official’s internals', async () => {
    const problems = await listProblems()
    // 'Assigned' is deliberately NOT in this list. A case is materialized lazily,
    // when the official first acts on it, so an assigned problem legitimately has
    // no case and getProgress correctly answers null — that is the null-shaped rule
    // (A.3.2), not a failure. Only the later statuses imply an official has already
    // acted, and therefore that a case exists to read.
    //
    // Including 'Assigned' made this test assert that the FIRST assigned problem in
    // a created_at DESC feed happens to have been worked on, which nothing
    // guarantees. It needs a problem walked to InProgress in the dev DB; if this
    // throws, that fixture is missing rather than the endpoint being broken.
    const withCase = problems.find((p) =>
      ['InProgress', 'Blocked', 'Done', 'Resolved', 'Reopened'].includes(p.status),
    )
    if (!withCase) {
      throw new Error('live fixture missing: no problem with a materialized case — walk one to InProgress first')
    }

    const progress = await getProgress(withCase.id)
    expect(progress).toEqual(
      expect.objectContaining({
        caseId: expect.any(String),
        problemId: withCase.id,
        status: expect.any(String),
        createdAt: expect.any(String),
        deadline: expect.any(String),
        acknowledged: expect.any(Boolean),
        updates: expect.any(Array),
        evidence: expect.any(Array),
        blockedOnHigherAuthority: expect.any(Boolean),
      }),
    )
    // The B12 disclosure boundary, checked from the client this time: these are
    // the official's working internals and must never reach a problem page.
    expect(progress).not.toHaveProperty('monitorOfficialId')
    expect(progress).not.toHaveProperty('escalationLevel')
    expect(progress).not.toHaveProperty('disputeReason')

    if (progress.plan) {
      expect(progress.plan).toEqual(
        expect.objectContaining({
          id: expect.any(String),
          strategy: expect.any(String),
          timelineWeeks: expect.any(Number),
          suggestionResponse: expect.any(String),
          // The snapshot the plan stored, never today's top.
          answeredSuggestion: expect.any(String),
          // The week-by-week checklist. Always an array (empty for plans that
          // predate the feature), never absent.
          tasks: expect.any(Array),
          createdAt: expect.any(String),
        }),
      )
      expect(progress.plan).not.toHaveProperty('caseId')
    }
  })

  // The plan write path, end to end through the real api modules: an official
  // publishes a week-by-week checklist and checks a week off, and the public
  // progress route reflects it. off-mayor is the only seeded official account;
  // its admin is the pourashava admin, so a pourashava problem is what it is
  // assigned.
  it('submits a weekly-task plan, completes a week, and the public sees the check', async () => {
    const filed = await reportProblem({
      title: `live check: plan tasks ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'pourashava-dhamoirhat', address: 'live plan-task test' },
      pointedOfficialId: 'off-mayor',
    })

    const pouraAdmin = await login({ phone: '01910000009', password: 'admin123' })
    setAuthToken(pouraAdmin.token)
    await approveProblem(filed.id)
    await assignWithinUnion(filed.id, { officialId: 'off-mayor', priority: 'normal' })

    const official = await login({ phone: '01710000001', password: 'official123' })
    setAuthToken(official.token)
    // GET /official/cases materializes the case lazily; find the one we just created.
    const cases = await listCases()
    const kase = cases.find((c) => c.problemId === filed.id)
    expect(kase).toBeDefined()

    await acknowledgeCase(kase.id, { decision: 'accept' })
    const planned = await submitPlan(kase.id, {
      strategy: 'phased repair',
      tasks: ['clear the drain inlet', 'lay new culvert pipe', 'backfill and repave'],
      suggestionResponse: 'adopting the community plan',
    })
    expect(planned.plan.tasks).toHaveLength(3)
    // TimelineWeeks is derived from the checklist — the two can never disagree.
    expect(planned.plan.timelineWeeks).toBe(3)
    const firstTask = planned.plan.tasks[0]
    expect(firstTask).toEqual(
      expect.objectContaining({ id: expect.any(String), weekNumber: 1, task: expect.any(String), completed: false }),
    )

    const updated = await completeTask(kase.id, firstTask.id)
    expect(updated.plan.tasks[0].completed).toBe(true)

    // Public progress reflects the check — one public account of a case, never two.
    setAuthToken(null)
    const progress = await getProgress(filed.id)
    expect(progress.plan.tasks).toHaveLength(3)
    expect(progress.plan.tasks[0].completed).toBe(true)
    expect(progress.plan.tasks[1].completed).toBe(false)

    setAuthToken(residentToken)
  })

  // B18/F21: the official raises an obstacle, resumes, and restarts with a fresh
  // plan — and the public progress record carries the new plan, the restart on the
  // timeline, and the obstacle as the persistent "cause of not solving".
  it('raises an obstacle then restarts the case with a revised plan', async () => {
    const filed = await reportProblem({
      title: `live check: replan ${Date.now()}`,
      description: 'filed by the live integration test',
      location: { areaId: 'pourashava-dhamoirhat', address: 'live replan test' },
      pointedOfficialId: 'off-mayor',
    })

    const pouraAdmin = await login({ phone: '01910000009', password: 'admin123' })
    setAuthToken(pouraAdmin.token)
    await approveProblem(filed.id)
    await assignWithinUnion(filed.id, { officialId: 'off-mayor', priority: 'normal' })

    const official = await login({ phone: '01710000001', password: 'official123' })
    setAuthToken(official.token)
    const kase = (await listCases()).find((c) => c.problemId === filed.id)
    expect(kase).toBeDefined()

    await acknowledgeCase(kase.id, { decision: 'accept' })
    await submitPlan(kase.id, {
      strategy: 'the first plan',
      tasks: ['lay the pipe'],
      suggestionResponse: 'adopting',
    })

    // Raise an obstacle → Blocked; the cause becomes public.
    await reportObstacle(kase.id, {
      category: 'budget',
      whatBlocks: 'no allocation this quarter',
      whoUnblocks: 'upazila engineer',
      proofTried: 'wrote to the UNO twice',
    })
    const blocked = await getProgress(filed.id)
    expect(blocked.obstacles).toHaveLength(1)
    expect(blocked.obstacles[0].whatBlocks).toBe('no allocation this quarter')

    // Resume (Blocked → InProgress) then restart with a fresh plan.
    await postUpdate(kase.id, { kind: 'progress', text: 'budget cleared, resuming' })
    const revised = await revisePlan(kase.id, {
      strategy: 'switch to a box culvert',
      tasks: ['re-survey', 'install box culvert'],
      suggestionResponse: 'adopting the revised idea',
      reason: 'the pipe approach kept flooding',
    })
    expect(revised.status).toBe('InProgress')

    // The public record shows the new plan, the restart on the timeline, and the
    // obstacle still (persistent cause) — all at once.
    setAuthToken(null)
    const after = await getProgress(filed.id)
    expect(after.plan.strategy).toBe('switch to a box culvert')
    expect(after.updates.some((u) => u.text.includes('Plan revised'))).toBe(true)
    expect(after.obstacles).toHaveLength(1)

    setAuthToken(residentToken)
  })

  // --- B13: the seat's geography ---

  it('lists areas without a token, in the Area shape', async () => {
    // Ungated: Register reads this before the resident has an account, so it must
    // answer with no Authorization header at all.
    setAuthToken(null)
    const areas = await listAreas()
    setAuthToken(null)

    expect(Array.isArray(areas)).toBe(true)
    expect(areas.length).toBeGreaterThan(0)
    expect(areas[0]).toEqual(
      expect.objectContaining({
        id: expect.any(String),
        name: expect.any(String),
        level: expect.any(String),
      }),
    )
    // Register's select is a list of unions; without one the pilot's first screen
    // cannot be filled in.
    expect(areas.some((a) => a.level === 'union')).toBe(true)
  })

  it('serializes the seat’s parentId as null, not an empty string', async () => {
    const areas = await listAreas()
    const seat = areas.find((a) => a.level === 'seat')
    expect(seat).toBeDefined()
    // The whole reason this assertion exists: "" and null are both falsy, so the
    // UI could never tell them apart, and the contract says string|null. Checked
    // through the real axios client, which is where a shape drift would land.
    expect(seat.parentId).toBeNull()

    // Re-authenticate for anything that follows.
    const res = await login({ phone: '01810000001', password: 'resident123' })
    setAuthToken(res.token)
  })

  // --- The seat's public decision record ---

  it('lists the seat activity without a token, in the ActivityEntry shape', async () => {
    // Ungated on purpose: this record exists for residents to read, so it must
    // answer with no Authorization header at all.
    setAuthToken(null)
    const entries = await listActivity()
    setAuthToken(null)

    expect(Array.isArray(entries)).toBe(true)
    if (entries.length > 0) {
      expect(entries[0]).toEqual(
        expect.objectContaining({
          id: expect.any(String),
          action: expect.any(String),
          actorId: expect.any(String),
          targetType: expect.any(String),
          targetId: expect.any(String),
          createdAt: expect.any(String),
        }),
      )
    }

    // Re-authenticate for anything that follows.
    const res = await login({ phone: '01810000001', password: 'resident123' })
    setAuthToken(res.token)
  })

  // THE assertion this endpoint exists to keep honest, driven through the real
  // client against real data. `verified` entries certainly exist in the log — the
  // resident-verification test above writes them — so their absence here is the
  // privacy split working, not an empty database.
  it('never publishes the person-actions in the public record', async () => {
    setAuthToken(null)
    const entries = await listActivity()
    setAuthToken(null)

    const leaked = entries.filter((e) =>
      ['verified', 'claim_approved', 'claim_rejected'].includes(e.action),
    )
    expect(leaked).toEqual([])

    const res = await login({ phone: '01810000001', password: 'resident123' })
    setAuthToken(res.token)
  })

  it('returns the seat overview with the SeatOverview shape', async () => {
    const seat = await getSeatOverview()
    expect(seat).toEqual(
      expect.objectContaining({
        problems: expect.any(Number),
        cases: expect.any(Number),
        resolved: expect.any(Number),
        pending: expect.any(Number),
        blocked: expect.any(Number),
        avgResponseDays: expect.any(Number),
      }),
    )
  })

  // --- B11: identity — resident verification, official claims, oversight ---
  //
  // Paying a debt: B11 shipped seven endpoints and never extended this file, which
  // is exactly how "no UI reaches any of them" went unnoticed until F15 removed the
  // mock that hid it. These drive the real api modules the F13 screens use.
  //
  // Fresh accounts per run (phone/nid from Date.now): verification is irreversible,
  // so mutating the standing fixture would make a second run assert against an
  // already-verified account.
  async function loginAs(phone, password) {
    const res = await login({ phone, password })
    setAuthToken(res.token)
    return res
  }

  it('opens the V gate: a fresh resident is unverified until their union admin verifies them', async () => {
    const stamp = String(Date.now())
    const phone = `018${stamp.slice(-8)}`
    await register({ name: 'পরীক্ষা বাসিন্দা', phone, password: 'resident123', nid: stamp, unionId: 'union-dhamoirhat' })

    // Freshly registered → not yet verified, so cannot count toward V.
    const before = await login({ phone, password: 'resident123' })
    expect(before.user.verified).toBe(false)
    const residentId = before.user.id

    // Dhamoirhat union's admin sees them in the pending queue and verifies them.
    await loginAs('01910000001', 'admin123')
    const pending = await listPendingResidents()
    expect(pending.map((u) => u.id)).toContain(residentId)
    await expect(verifyResident(residentId)).resolves.toMatchObject({ verified: true })

    // The gate is open: the same resident now reads back verified.
    const after = await login({ phone, password: 'resident123' })
    expect(after.user.verified).toBe(true)
  })

  it('refuses an admin verifying a resident of another union — 403', async () => {
    const stamp = String(Date.now())
    const phone = `019${stamp.slice(-8)}`
    await register({ name: 'অন্য ইউনিয়নের বাসিন্দা', phone, password: 'resident123', nid: stamp, unionId: 'union-agradigun' })
    const other = await login({ phone, password: 'resident123' })

    // Dhamoirhat union's admin has no standing over an Agradigun resident.
    await loginAs('01910000001', 'admin123')
    await expect(verifyResident(other.user.id)).rejects.toMatchObject({ status: 403 })
  })

  it('registers an official against a directory office → token + user, and the union admin sees the pending claim', async () => {
    const stamp = String(Date.now())
    const phone = `017${stamp.slice(-8)}`
    // The Dhamoirhat union chairman is a union-level office, so that union's own
    // admin reviews the claim. (The admin is *bound* to this office for scoping,
    // which is not a claim on it — 000014's backfill deliberately skips admins.)
    const reg = await registerOfficial({
      name: 'দাবিদার কর্মকর্তা',
      phone,
      password: 'official123',
      nid: stamp,
      officialId: 'off-chair-dhamoirhat',
    })
    expect(reg.token).toBeTruthy()
    expect(reg.user).toEqual(expect.objectContaining({ role: 'official', phone }))

    await loginAs('01910000001', 'admin123')
    const claims = await listPendingClaims()
    const mine = claims.find((c) => c.accountId === reg.user.id)
    // Not approved here on purpose: approval binds the office permanently (a partial
    // unique index on approved claims), which would break the next run.
    expect(mine).toEqual(
      expect.objectContaining({
        id: expect.any(String),
        officialId: 'off-chair-dhamoirhat',
        status: 'Pending',
        officialName: expect.any(String),
        officialTier: 'union_chairman',
        areaId: 'union-dhamoirhat',
        claimantName: 'দাবিদার কর্মকর্তা',
      }),
    )
  })

  // The load-bearing authorization assertion: RequireAdmin does not admit the super
  // admin, and RequireSuperAdmin does not admit an admin — the roles are flat and
  // disjoint, not nested. This is the single rule every client-side guard mirrors.
  it('refuses oversight to a union admin — 403', async () => {
    await loginAs('01910000001', 'admin123')
    await expect(getOversight()).rejects.toMatchObject({ status: 403 })
  })

  it('reads oversight as the super admin, in the AuditEntry shape', async () => {
    await loginAs('01710000010', 'admin123')
    const entries = await getOversight()
    expect(Array.isArray(entries)).toBe(true)
    if (entries.length > 0) {
      expect(entries[0]).toEqual(
        expect.objectContaining({
          id: expect.any(String),
          actor: expect.any(String),
          action: expect.any(String),
          createdAt: expect.any(String),
        }),
      )
    }
  })

  // --- B19: the observation ladder ---
  //
  // The rows themselves are opened by cmd/worker, not by any endpoint, so this
  // asserts the SHAPE the page reads and the two rules a client could get wrong.
  // Whether a row is present depends on whether a silent case exists in the DB
  // when the suite runs, which is why the assertions are conditional on it.

  it('lists the observations of the official who monitors them, in the ObservedCase shape', async () => {
    // The seeded official is the mayor; the upazila chairman monitors them.
    await loginAs('01710000001', 'official123')
    const rows = await listObservations()
    expect(Array.isArray(rows)).toBe(true)
    for (const row of rows) {
      expect(row).toEqual(
        expect.objectContaining({
          caseId: expect.any(String),
          problemId: expect.any(String),
          officialId: expect.any(String),
          status: expect.any(String),
          deadline: expect.any(String),
          silentDays: expect.any(Number),
          direct: expect.any(Boolean),
          observations: expect.any(Array),
        }),
      )
      // Present-and-null, never absent: both are falsy in JS, so a field that
      // came and went would surface only as a wrongly offered note box.
      expect(row).toHaveProperty('lastActivityAt')
      for (const obs of row.observations) {
        expect(obs).toHaveProperty('resolvedAt')
        expect(obs).toEqual(
          expect.objectContaining({ id: expect.any(String), level: expect.any(Number), openedAt: expect.any(String) }),
        )
      }
      // A monitor watches; they never carry the assignee's working internals.
      expect(row).not.toHaveProperty('disputeReason')
      expect(row).not.toHaveProperty('monitorOfficialId')
    }
  })

  it('refuses /official/observations without a token — 401, not an empty list', async () => {
    setAuthToken(null)
    await expect(listObservations()).rejects.toMatchObject({ status: 401 })
    setAuthToken(residentToken)
  })

  // Two guards, asserted where they do not depend on what is in the DB today.
  // The note route is officials-only, and within it an unknown observation is a
  // 404 — proving the route is mounted and the handler actually runs, rather
  // than the request dying at the gate.
  it('gates the note route to officials, and 404s an unknown observation', async () => {
    await loginAs('01810000001', 'resident123')
    await expect(noteObservation('obs-nope', { text: 'x' })).rejects.toMatchObject({ status: 403 })

    await loginAs('01710000001', 'official123')
    await expect(noteObservation('obs-nope', { text: 'x' })).rejects.toMatchObject({
      status: 404,
      code: 'observation_not_found',
    })
  })

  // --- B21: in-app notifications -------------------------------------------
  //
  // The rows are written by the seven state-changing use cases, not by any route
  // here, so what this proves is the SHAPE the bell and the page read plus the
  // three rules a client could get wrong: the list is self-derived (401 for
  // anonymous, never an empty list), `readAt` is present-and-null, and someone
  // else's id is a 404 rather than a 403. Whether a row is present depends on
  // what the DB holds when the suite runs, which is why the shape assertions are
  // conditional on it — the same shape the Go integration test walks in full.

  it('lists the caller own notifications in the Notification shape', async () => {
    await loginAs('01810000001', 'resident123')
    const rows = await listNotifications()
    expect(Array.isArray(rows)).toBe(true)

    const TYPES = [
      'problem_approved',
      'problem_rejected',
      'problem_assigned',
      'confirmation_requested',
      'case_assigned',
      'case_reopened',
      'obstacle_declared',
      'obstacle_adjudicated',
    ]
    for (const row of rows) {
      expect(row).toEqual(
        expect.objectContaining({
          id: expect.any(String),
          type: expect.any(String),
          problemId: expect.any(String),
          problemTitle: expect.any(String),
          detail: expect.any(String),
          createdAt: expect.any(String),
        }),
      )
      // models.js's NotificationType is the frontend's copy of the backend's
      // whitelist; a ninth type reaching here means the two have drifted, and the
      // row's sentence switch would silently render nothing.
      expect(TYPES).toContain(row.type)
      // Present-and-null, never absent: both are falsy in JS, so a field that came
      // and went would surface only as a wrongly-styled row.
      expect(row).toHaveProperty('readAt')
      // The caller's own identities are implied, never echoed — a recipientId on
      // the wire is an id space for someone to start trusting.
      expect(row).not.toHaveProperty('recipientId')
      expect(row).not.toHaveProperty('recipientKind')
    }
  })

  it('returns an unread count for the bell', async () => {
    await loginAs('01810000001', 'resident123')
    const body = await unreadNotificationCount()
    expect(body).toEqual(expect.objectContaining({ count: expect.any(Number) }))
    expect(body.count).toBeGreaterThanOrEqual(0)
  })

  // The gate. RequireAnyRole admits all four roles and refuses anonymous — and an
  // anonymous caller must get a 401, never a silently empty list that reads as
  // "you have no messages".
  it('is reachable by every signed-in role and refuses anonymous', async () => {
    for (const [phone, password] of [
      ['01810000001', 'resident123'],
      ['01710000001', 'official123'],
      ['01910000001', 'admin123'],
      ['01710000010', 'admin123'],
    ]) {
      await loginAs(phone, password)
      await expect(listNotifications()).resolves.toEqual(expect.any(Array))
      await expect(unreadNotificationCount()).resolves.toEqual(
        expect.objectContaining({ count: expect.any(Number) }),
      )
    }

    setAuthToken(null)
    await expect(listNotifications()).rejects.toMatchObject({ status: 401 })
    await expect(unreadNotificationCount()).rejects.toMatchObject({ status: 401 })
    setAuthToken(residentToken)
  })

  // The no-oracle rule: an unknown id and someone else's must be indistinguishable,
  // so both are 404 and neither is 403.
  it('404s an unknown notification id rather than 403ing it', async () => {
    await loginAs('01810000001', 'resident123')
    await expect(markNotificationRead('notif-does-not-exist')).rejects.toMatchObject({
      status: 404,
      code: 'not_found',
    })
  })

  // Mark-all is idempotent and safe to run against whatever the DB holds: it only
  // ever clears the CALLER'S OWN rows, which is the same property that makes the
  // count leak nothing.
  it('clears the caller own badge, and only theirs', async () => {
    await loginAs('01810000001', 'resident123')
    const cleared = await markAllNotificationsRead()
    expect(cleared).toEqual(expect.objectContaining({ marked: expect.any(Number) }))
    expect(await unreadNotificationCount()).toEqual({ count: 0 })

    // Read is not deleted — the rows survive, with a timestamp.
    for (const row of await listNotifications()) {
      expect(row.readAt).not.toBeNull()
    }
  })
})
