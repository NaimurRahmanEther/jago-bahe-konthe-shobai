/**
 * JSDoc @typedef shapes — the JS "contract". Keep in sync with the backend DTOs
 * (interfaces/http/dto.go) and Project Scaffold Spec §2/§5. Reference these
 * typedefs in JSDoc on API functions and hooks; this file is the only guardrail
 * against shape drift in plain JavaScript.
 */

/**
 * `super_admin` is a FOURTH role, not a superset of admin: it reviews above-union
 * official claims and reads every admin decision, but cannot screen, assign, or
 * vote, and can override nothing (Scaffold §2, "Oversight, not override").
 * @typedef {'resident' | 'official' | 'admin' | 'super_admin'} Role
 */

/**
 * Union-level tiers: an official at this tier is decided on by that union's admin alone.
 * Above-union tiers: routed to an admin vote (scope upazila or seat).
 * @typedef {'ward_member' | 'women_member' | 'union_chairman' | 'pourashava_councillor' | 'pourashava_mayor' | 'upazila_chairman' | 'upazila_vice_chairman' | 'mp' | 'minister'} Tier
 */

/**
 * @typedef {'seat' | 'upazila' | 'union' | 'ward'} AreaLevel
 */

/**
 * @typedef {Object} Location
 * @property {string} areaId - the Ward/Union this location resolves to
 * @property {string} [address]
 * @property {number} [lat]
 * @property {number} [lng]
 */

/**
 * The eleven states, mirroring problem/domain/status.go.
 *
 * `PendingApproval` is where every new report starts: held invisible to the public
 * until its union admin approves it — the one status excluded from PublicStatuses()
 * and from the feed's filter (ProblemFeed's STATUSES). On approval it becomes
 * `Reported` (public and votable). `Rejected` is terminal but deliberately public:
 * a report taken down for spam or abuse keeps its reason on the record rather than
 * disappearing. `Withdrawn` is the reporter's own terminal takedown — they retracted
 * the report, and like `Rejected` it stays on the public record rather than vanishing.
 * @typedef {'PendingApproval' | 'Reported' | 'Validated' | 'Assigned' | 'InProgress' | 'Blocked' | 'Done' | 'Resolved' | 'Reopened' | 'Rejected' | 'Withdrawn'} ProblemStatus
 */

/**
 * The four grounds a report may be rejected on, and nothing else. An admin may
 * never reject on merit — whether a genuine report is real is the community's
 * call via V (Concept §4).
 * @typedef {'spam' | 'abusive' | 'duplicate' | 'wrong_area'} RejectionReason
 */

/**
 * @typedef {Object} User
 * @property {string} id
 * @property {string} name
 * @property {string} phone
 * @property {string} nid
 * @property {string} unionId
 * @property {Role} role
 * @property {boolean} verified - only a verified resident may validate, suggest, or upvote
 * @property {string} [officialId] - set once an official's claim is approved; absent for residents
 */

/**
 * @typedef {Object} Official
 * @property {string} id
 * @property {string} name
 * @property {string} phone
 * @property {Tier} tier
 * @property {string} areaId
 */

/**
 * @typedef {'Pending' | 'Approved' | 'Rejected'} ClaimStatus
 */

/**
 * An official's claim to a directory office (GET /api/claims/pending, and the
 * result of approve/reject). Approving one binds the claimant's account to the
 * office, setting accounts.official_id.
 * @typedef {Object} OfficialClaim
 * @property {string} id
 * @property {string} accountId
 * @property {string} officialId - the directory office being claimed
 * @property {ClaimStatus} status
 * @property {string} [reviewedBy] - the deciding reviewer's account id
 * @property {string} [reviewedAt] - ISO timestamp of the decision
 * @property {string} [reason] - set only when Rejected; shown to the claimant
 * @property {string} createdAt
 */

/**
 * A review-queue entry: the claim plus who and which office it concerns, so a
 * reviewer can judge it without three more requests.
 * @typedef {OfficialClaim & {officialName: string, officialTier: Tier, areaId: string, claimantName: string, claimantNid?: string}} PendingClaim
 */

/**
 * The geography is FLAT, not a tree — each node carries its parent's id and the
 * client derives any nesting it needs. Served by GET /api/areas (B13).
 *
 * @typedef {Object} Area
 * @property {string} id
 * @property {string} name
 * @property {AreaLevel} level
 * @property {string|null} parentId - null at the seat (the root), never "" —
 *   the backend serializes it as a JSON null, and both would be falsy here, so
 *   an `if (parentId)` check cannot tell a regression from the contract.
 */

/**
 * A reported problem.
 *
 * A reporter may retract their own report two ways, and they are not degrees of
 * the same thing. `POST /problems/{id}/withdraw` sets `Withdrawn` and leaves it on
 * the public record with its audit trail; `DELETE /problems/{id}` erases the
 * problem, everything cascading from it (votes, suggestions, the case and its
 * plan/updates/evidence) and its audit entries, at any status, with no undo.
 * See CLAUDE.md A.3.3.
 *
 * @typedef {Object} Problem
 * @property {string} id
 * @property {string} title
 * @property {string} description
 * @property {Location} location
 * @property {string} reporterId
 * @property {string} pointedOfficialId - who the PUBLIC asked for, chosen by the reporter at filing
 *   and immutable afterwards. Not necessarily who is working on it — see assignedOfficialId.
 * @property {string} [assignedOfficialId] - who the ADMIN actually handed it to, absent until the
 *   problem is assigned. The gap between this and pointedOfficialId is an override. Present on the
 *   feed as well as the detail read, composed in one batch query per page (handler.assignmentLookup),
 *   never a per-row lookup.
 * @property {string} [overrideReason] - the admin's public justification for assigning someone other
 *   than the public's nominee. Absent in the ordinary case, where the admin confirmed that nominee and
 *   so had nothing to justify. An override the public cannot see is not accountable, which is why the
 *   reason is published with the gap rather than held with the assignment.
 * @property {string|null} [proposedSolution]
 * @property {string|null} [imageUrl] - the reporter's photo of the problem
 * @property {ProblemStatus} status
 * @property {number} validCount - distinct verified area residents who voted valid (display-only)
 * @property {number} validationThreshold - the configured V. ADMIN surfaces only: the public
 *   sees the count alone (ValidationCount), because since B17 reaching V unlocks nothing —
 *   a denominator would advertise a gate that is not there (A.3.1.1).
 * @property {'valid'|'invalid'|null} myVote - the CALLING account's own validation vote, null when
 *   they have not voted, are not a resident, or are anonymous. Always present as JSON `null`,
 *   never absent — an absent key and a null are both falsy here, so a regression to "sometimes
 *   missing" would show up only as a wrongly-enabled vote button. Votes are immutable, so once
 *   non-null this never changes.
 * @property {string} createdAt - ISO timestamp
 * @property {RejectionReason} [rejectionReason] - set only when status is Rejected; shown publicly with the problem
 * @property {string} [reviewedBy] - the deciding admin's account id, or 'system' when the window A expired
 * @property {string} [reviewedAt] - ISO timestamp of the screening decision
 * @property {Suggestion[]} [suggestions] - served by GET /problems/{id}/suggestions; left empty on the problem itself
 * @property {Evidence[]} [evidence] - TODO(contract): problemDTO ships this empty (dto.go's standing B5 TODO), so it is
 *   populated on mocks only. The real per-problem evidence comes from GET /problems/{id}/progress.
 * @property {AuditEntry[]} [audit] - present on GET /problems/{id}
 */

/**
 * @typedef {'valid' | 'invalid'} ValidationChoice
 */

/**
 * @typedef {Object} ValidationVote
 * @property {string} id
 * @property {string} problemId
 * @property {string} voterId
 * @property {ValidationChoice} vote
 * @property {string} createdAt
 */

/**
 * @typedef {Object} Suggestion
 * @property {string} id
 * @property {string} problemId
 * @property {string} authorId
 * @property {string} text
 * @property {number} upvoteCount
 * @property {boolean} isTop - backend-computed; the highest-upvoted suggestion. TIES ALL CARRY IT —
 *   the ranking service marks every suggestion sharing the top count, so more than one row can be
 *   flagged and no consumer may assume exactly one. Zero upvotes is never top, so a problem nobody
 *   has upvoted has no top at all. Never recompute this in the UI.
 * @property {boolean|null} myUpvote - the CALLING account's own upvote, null when they have not
 *   upvoted, are not a resident, or are anonymous. Always present as JSON `null`, never absent —
 *   an absent key and a null are both falsy here, so a regression to "sometimes missing" would show
 *   up only as a wrongly-offered upvote button, which the toggle then turns into a silent WITHDRAWAL
 *   of a vote the reader still held. The upvote counts are public; who cast them is not.
 * @property {string} createdAt
 */

/**
 * One row of GET /api/admin/queue — the problems the acting admin may still
 * forward to an official. NOT a Problem: the queue is a projection, so the field
 * names differ (`problemId`, not `id`; a flat `address`, not `location`).
 *
 * This typedef did not exist until B17, and its absence is exactly how the shape
 * drifted: `QueueItem.jsx` read `item.id`, `item.location.address`,
 * `item.pointedOfficial` and `item.vote` — none of which the API ever sent — so
 * any non-empty queue threw. Nothing caught it because no problem had ever reached
 * the queue against the real API, so only the empty branch had ever rendered.
 *
 * Both `Reported` and `Validated` rows appear here. Since B17 the validation count
 * gates nothing: the admin forwards a report when they judge it trustworthy, and
 * `validCount`/`validationThreshold` are the evidence for that judgment, rendered
 * by ValidationBar. Display only — see CLAUDE.md A.3.1.
 *
 * @typedef {Object} QueueItem
 * @property {string} problemId
 * @property {string} title
 * @property {ProblemStatus} status - 'Reported' or 'Validated'
 * @property {string} address - free text from the problem's location
 * @property {string} pointedOfficialId - resolve the name via useOfficials(); the row carries the id only
 * @property {string} areaId
 * @property {'union'} routing - backend-decided decision path; never re-derive it client-side.
 *   Always 'union' in practice since B20: an above-union report is the super admin's to forward and
 *   the backend leaves it off this queue entirely (see ForwardingItem, CLAUDE.md A.3.8).
 * @property {number} validCount - distinct verified residents who validated (display only)
 * @property {number} validationThreshold - the configured V, for the "X of V" display only
 */

/**
 * The set of union admins entitled to ADVISE on an above-union report: the admins
 * of one upazila, or every admin in the seat. Chosen by the pointed official's
 * tier, backend-side. Was VoteScope until B20 — the scope rule is unchanged, only
 * what the admins' input does.
 *
 * @typedef {'upazila' | 'seat'} AdviceScope
 */

/**
 * @typedef {Object} Assignment
 * @property {string} id
 * @property {string} problemId
 * @property {string} officialId
 * @property {string} monitorOfficialId - the next tier up, notified on obstacles/escalation
 * @property {string} priority
 * @property {string} deadline - ISO timestamp, first-response deadline D
 * @property {string|null} [overrideReason] - required public reason when an admin overrides the public's choice
 * @property {string} createdAt
 */

/**
 * One union admin's advice on where an above-union report should be forwarded.
 *
 * It is ADVICE, not a ballot: no quorum, no window, no outcome it can produce. The
 * super admin decides and may forward at any count including none — the same
 * "signal, not a gate" rule B17 set for V (CLAUDE.md A.3.1.1, A.3.8).
 *
 * THE ADVISER IS NAMED (`adminAccountId`), unlike the secret ballot this replaced.
 * A vote that BINDS earns protection from pressure; advice that binds nobody does
 * not, and the point of recording it is that the seat's reasoning is public.
 *
 * @typedef {Object} ForwardingSuggestion
 * @property {string} id
 * @property {string} adminAccountId - the adviser; the tally is public and attributed
 * @property {string} officialId - whom they advise forwarding to
 * @property {string} [reason] - optional public note; absent when they gave none
 * @property {string} createdAt
 */

/**
 * One above-union report awaiting a forward. The SAME shape on both surfaces —
 * the union admin's advisory list (GET /api/admin/forwarding) and the super
 * admin's decision queue (GET /api/super/queue) — so the two can never disagree
 * about what the seat has been advised.
 *
 * @typedef {Object} ForwardingItem
 * @property {string} problemId
 * @property {string} title
 * @property {ProblemStatus} status - 'Reported' or 'Validated'
 * @property {string} address - free text from the problem's location
 * @property {string} areaId
 * @property {string} pointedOfficialId - whom the REPORTER pointed the report at; one of the two
 *   things a departure requires a public reason from
 * @property {AdviceScope} scope - which admins are entitled to advise on this one
 * @property {number} validCount
 * @property {number} validationThreshold - the configured V, display only
 * @property {ForwardingSuggestion[]} suggestions - every adviser's advice, oldest first
 * @property {string} topOfficialId - whom the advisers most agree on. EMPTY when nobody advised AND
 *   when they TIED: a tie is genuine disagreement, and breaking it arbitrarily would invent a
 *   consensus and make the reason requirement turn on a coin flip. Never treat '' as "nobody advised".
 * @property {number} topCount - how many named `topOfficialId`; 0 in both empty cases above
 * @property {ForwardingSuggestion|null} mySuggestion - the CALLER's own advice, null when they have
 *   not advised and always null for the super admin, who advises on nothing. Always present as JSON
 *   `null`, never absent — both are falsy in JS, so a regression to "sometimes missing" would surface
 *   only as a wrongly pre-filled select (the myVote lesson, A.3.2.1 rule 2).
 */

/**
 * 'Assigned' precedes acknowledgement; 'Disputed' is a mock-only simplification
 * for the dispute path (Scaffold Spec §2 doesn't model backend re-assignment
 * after a dispute — TODO(contract) when B4/B5 define that flow).
 * @typedef {'Assigned' | 'Acknowledged' | 'Planned' | 'InProgress' | 'Done' | 'Resolved' | 'Blocked' | 'Reopened' | 'Disputed'} CaseStatus
 */

/**
 * @typedef {Object} Case
 * @property {string} id
 * @property {string} problemId
 * @property {string} officialId
 * @property {string|null} monitorOfficialId - copied from the Assignment; who's auto-notified if blocked
 * @property {CaseStatus} status
 * @property {string|null} acknowledgedAt
 * @property {string} deadline - ISO timestamp, first-response deadline D
 * @property {string} createdAt
 * @property {number} escalationLevel - visibility rung climbed from measured silence (0 = with the official only); backend-set
 * @property {string|null} [lastEscalatedAt] - when the worker last raised the escalation level
 * @property {Plan|null} [plan]
 * @property {ProgressUpdate[]} [updates]
 * @property {Evidence[]} [evidence]
 * @property {Obstacle[]} [obstacles]
 * @property {string|null} [disputeReason]
 * @property {boolean} [blockedOnHigherAuthority] - waiting on a obstacle a named authority judged real; NOT the official's failure
 */

/**
 * One week's task in a plan's checklist. week N = its position in the plan.
 * completed/completedAt are the only mutable fields — the official checks a week
 * off in public as it is finished (POST .../tasks/{id}/complete). Completing every
 * task changes no status: it is a milestone, not the path to Done (evidence-gated).
 * @typedef {Object} PlanTask
 * @property {string} id
 * @property {number} weekNumber
 * @property {string} task
 * @property {boolean} completed
 * @property {string|null} completedAt
 */

/**
 * @typedef {Object} Plan
 * @property {string} id
 * @property {string} caseId
 * @property {string} strategy
 * @property {number} timelineWeeks - derived from tasks.length, never a separate input
 * @property {string} obstacles
 * @property {string} suggestionResponse - must explicitly answer the community's top suggestion
 * @property {PlanTask[]} tasks - the week-by-week checklist (one task per week)
 * @property {string} [answeredSuggestionId] - the snapshot's id; empty when nothing was upvoted
 * @property {string} [answeredSuggestion] - the top suggestion's text AS IT STOOD when the plan was submitted.
 *   "Top" moves with live upvotes, so every public view pairs the response with THIS, never with today's top —
 *   otherwise an official appears to answer a question they were never asked (Scaffold §2).
 * @property {string} createdAt
 */

/**
 * @typedef {'progress' | 'obstacle'} ProgressUpdateKind
 */

/**
 * @typedef {Object} ProgressUpdate
 * @property {string} id
 * @property {string} caseId
 * @property {ProgressUpdateKind} kind
 * @property {string} text
 * @property {string} createdAt
 */

/**
 * The kind of obstacle, which decides which higher authority the obstacle is
 * forwarded to (Concept §8). Backend-owned routing; the UI only collects/displays it.
 * @typedef {'budget' | 'legal_authority' | 'higher_tier' | 'land_dispute' | 'inter_department' | 'technical'} ObstacleCategory
 */

/**
 * A community-proposed way to overcome a obstacle — the suggestion engine
 * re-pointed at the obstacle instead of the problem (Concept §8).
 * @typedef {Object} UnblockingPlan
 * @property {string} id
 * @property {string} obstacleId
 * @property {string} authorId
 * @property {string} text
 * @property {number} upvoteCount
 * @property {string} createdAt
 */

/**
 * The public's advisory read on whether an obstacle is genuine. Pressure only —
 * the named higher authority (or a moderator) adjudicates the scorecard
 * consequence, so this never sets status (Concept §8/§9).
 * @typedef {'real' | 'not_convinced'} ObstacleVoteChoice
 */

/**
 * @typedef {Object} Obstacle
 * @property {string} id
 * @property {string} caseId
 * @property {ObstacleCategory} category - the kind of obstacle; routes the forward
 * @property {string} whatBlocks
 * @property {string} whoUnblocks - the named higher authority notified
 * @property {string} proofTried
 * @property {number} realCount - advisory "obstacle is real" tally (display-only)
 * @property {number} notConvincedCount - advisory "not convinced" tally (display-only)
 * @property {UnblockingPlan[]} [unblockingPlans] - community ways to unblock it
 * @property {Adjudication} adjudication - the authority's binding verdict (sets the scorecard consequence, not the advisory vote)
 * @property {string|null} [adjudicatedAt] - when the obstacle was adjudicated
 * @property {string} createdAt
 * @property {string|null} [resolvedAt]
 */

/**
 * The named higher authority's (or a moderator's) binding verdict on a obstacle.
 * Unlike the advisory obstacle vote, this is what sets the scorecard consequence
 * (Concept §8/§9). 'confirmed' → responsibility sits up-ladder, the official is
 * protected; 'denied' → the case bounces back to the official.
 * @typedef {'pending' | 'confirmed' | 'denied'} Adjudication
 */

/**
 * @typedef {Object} Evidence
 * @property {string} id
 * @property {string} caseId
 * @property {string} beforeImageUrl
 * @property {string} afterImageUrl
 * @property {string} createdAt
 */

/**
 * @typedef {'solved' | 'not_solved'} ConfirmationOutcome
 */

/**
 * @typedef {Object} Confirmation
 * @property {string} id
 * @property {string} problemId
 * @property {string} residentId
 * @property {ConfirmationOutcome} outcome
 * @property {string} createdAt
 */

/**
 * One entry in the append-only audit log. GET /api/super/oversight (the super
 * admin's feed) returns exactly this shape — oversight is a projection of the
 * audit log, not a separate record, so it reuses this typedef rather than a twin.
 * @typedef {Object} AuditEntry
 * @property {string} id
 * @property {string} targetType
 * @property {string} targetId
 * @property {string} actor
 * @property {string} action
 * @property {string|null} [reason]
 * @property {string} createdAt
 */

/**
 * One row of the seat's PUBLIC decision record (GET /api/seat/activity).
 *
 * A resolved cousin of AuditEntry, not a twin of it, and the difference is the
 * point: an AuditEntry is read on a problem the reader already has open, so a raw
 * `actor` id is enough context. This one is seat-wide, so the server resolves the
 * ids into names and titles — otherwise the aggregate that makes a pattern visible
 * is a page of opaque strings.
 *
 * It carries the five PROBLEM actions only. `verified`, `claim_approved` and
 * `claim_rejected` are about people and never appear here; they stay on
 * GET /api/super/oversight. See lib/api/activity.js.
 *
 * @typedef {Object} ActivityEntry
 * @property {string} id
 * @property {string} action - approved | rejected | assigned | vote_opened | adjudicated
 * @property {string} actorId - an account id, a directory office id, or the literal 'system'
 * @property {string} [actorName] - absent when the id did not resolve; render actorId then
 * @property {string} targetType
 * @property {string} targetId
 * @property {string} [problemTitle] - present only for a PUBLICLY VISIBLE problem; absence is normal
 * @property {string} [reason]
 * @property {string} createdAt
 */

/**
 * @typedef {Object} OfficialStats
 * @property {string} officialId
 * @property {number} resolved
 * @property {number} pending
 * @property {number} blocked - blocked-on-higher-authority; NOT counted as this official's failure
 * @property {number} avgResponseDays
 */

/**
 * Seat-wide totals (GET /seat/overview). Every count is over publicly visible
 * problems only — counting unscreened ones would leak how many exist.
 * @typedef {Object} SeatOverview
 * @property {number} problems
 * @property {number} cases
 * @property {number} resolved
 * @property {number} pending
 * @property {number} blocked - the fair bucket; same rule as OfficialStats
 * @property {number} avgResponseDays
 */

/**
 * The plan as the public record carries it (GET /problems/{id}/progress). Note
 * there is no caseId here: it is the plan, projected, not the aggregate.
 * @typedef {Object} ProgressPlan
 * @property {string} id
 * @property {string} strategy
 * @property {number} timelineWeeks - derived from tasks.length
 * @property {string} obstacles
 * @property {string} suggestionResponse
 * @property {PlanTask[]} tasks - the week-by-week checklist, with each week's completion state (public)
 * @property {string} answeredSuggestion - the SNAPSHOT (see Plan.answeredSuggestion); '' when nothing was upvoted
 * @property {string} createdAt
 */

/**
 * A problem's public progress (GET /problems/{id}/progress).
 *
 * Deliberately narrower than Case: no monitorOfficialId, escalationLevel, or
 * disputeReason — those are the official's working internals, and publishing
 * them would disclose how far up the ladder a case has climbed on every problem
 * page.
 *
 * The endpoint resolves to `null` (a 200, never a 404) when there is no case:
 * an unknown problem, an unscreened one, an unassigned one, and a case no
 * official has opened yet are all indistinguishable on purpose, so the
 * difference cannot be used to enumerate what is awaiting screening.
 * @typedef {Object} Progress
 * @property {string} caseId
 * @property {string} problemId
 * @property {CaseStatus} status
 * @property {string} createdAt
 * @property {string} deadline
 * @property {boolean} acknowledged - silence is itself a fact, so an unacknowledged case is still reported
 * @property {string|null} acknowledgedAt
 * @property {ProgressPlan|null} plan - null until the official publishes one
 * @property {ProgressUpdate[]} updates
 * @property {Evidence[]} evidence
 * @property {Obstacle[]} obstacles - the "cause of not solving": obstacle history, persistent after the case leaves Blocked
 * @property {boolean} blockedOnHigherAuthority - amber + the fairness note, never red
 */

/**
 * One rung of the monitor ladder watching a case its official has gone silent on
 * (B19). Level 1 is the direct monitor, 2 the tier above, up to 3.
 *
 * `resolvedAt` is present-and-null while the official is still silent, never
 * absent — both are falsy in JS, so a field that came and went would surface
 * only as a wrongly offered note box.
 *
 * A resolved observation is KEPT, not deleted: that an official was silent for
 * eleven days is a fact about the public record.
 * @typedef {Object} Observation
 * @property {string} id
 * @property {number} level
 * @property {string} openedAt
 * @property {string|null} resolvedAt - null while the official is still silent
 * @property {ObservationNote[]} notes
 */

/**
 * What a monitor recorded doing about a silence — their one action. It is
 * appended, never overwritten, and also lands on the problem's public audit
 * trail.
 * @typedef {Object} ObservationNote
 * @property {string} id
 * @property {string} text
 * @property {string} createdAt
 */

/**
 * One row of GET /official/observations: a case below the caller on the
 * accountability ladder.
 *
 * Deliberately narrower than Case, for the same reason Progress is: no
 * disputeReason and no monitorOfficialId. A monitor watches the case; they do
 * not work it, and the API offers them no action but a note — silence raises
 * visibility, not responsibility (Concept §7).
 *
 * Rows arrive sorted by the backend, waiting ones first (A.5 rule 7). Link them
 * to /problems/{problemId}, NOT /official/cases/{caseId} — a monitor is not the
 * case's owner and the backend answers 403 there.
 * @typedef {Object} ObservedCase
 * @property {string} caseId
 * @property {string} problemId
 * @property {string} problemTitle
 * @property {string} officialId - the official the case is assigned to
 * @property {string} officialName
 * @property {CaseStatus} status
 * @property {string} deadline
 * @property {string|null} lastActivityAt - null when the official has never acted at all
 * @property {number} silentDays - days waiting past the deadline; 0 when on time or answered
 * @property {boolean} direct - true when the caller is this case's own monitor, watching since assignment
 * @property {Observation[]} observations
 */

/**
 * The eight in-app notification events — the "loop-closing set" (B21).
 *
 * They are the cases where a party has NO other surface that would tell them:
 * the reporter (approved / rejected / assigned / please confirm), the assigned
 * official (a case is yours / the resident reopened it / your obstacle was
 * judged) and the monitor (an obstacle was declared on a case you watch — the
 * one notification the design documents promise outright, Concept §8).
 *
 * There are deliberately NONE for admins, who already have three queues with
 * live counts on /admin, and none about PEOPLE: `verified`, `claim_approved`
 * and `claim_rejected` are out of scope here (A.3.5 rule 1, A.3.9).
 *
 * This list is the frontend's copy of the backend's Type whitelist, exactly as
 * ProblemStatus mirrors PublicStatuses(). There is no GET /notification-types,
 * and inventing one to avoid an eight-element union would be an A.1 violation
 * for no gain — the migration's CHECK constraint is what actually enforces it.
 * @typedef {'problem_approved' | 'problem_rejected' | 'problem_assigned' | 'confirmation_requested' | 'case_assigned' | 'case_reopened' | 'obstacle_declared' | 'obstacle_adjudicated'} NotificationType
 */

/**
 * One row of GET /api/me/notifications.
 *
 * `problemTitle` and `detail` are SNAPSHOTTED by the backend at write time, not
 * resolved on read — a notification records what was true when it was sent, and
 * a reporter may edit their own title afterwards. `problemTitle` may be `""`;
 * fall back to notification.untitled, as ObservedCase does.
 *
 * `detail` is TYPE-SPECIFIC and must only ever be read inside a branch on
 * `type`: the rejection ground enum (`problem_rejected`), the assigned
 * official's name (`problem_assigned`), an ISO deadline (`case_assigned`),
 * `'confirmed'`/`'denied'` (`obstacle_adjudicated`), the obstacle's free-text
 * whoUnblocks (`obstacle_declared`), and `""` for the rest.
 *
 * `readAt` is present-and-null while unread, NEVER absent — both are falsy in
 * JS, so a field that came and went would surface only as a wrongly-styled row.
 *
 * The list is self-derived from the token and takes no parameter, ever: the
 * problems are public, the authorship graph is not (A.3.2 rule 1).
 * @typedef {Object} Notification
 * @property {string} id
 * @property {NotificationType} type
 * @property {string} problemId
 * @property {string} problemTitle - snapshotted at write time; may be ""
 * @property {string} detail - type-specific; read only inside a switch on type
 * @property {string|null} readAt - null while unread, never absent
 * @property {string} createdAt
 */

/**
 * GET /api/me/notifications/unread-count — the bell's badge.
 *
 * A separate endpoint from the list because the bell mounts on EVERY page and
 * must not pull fifty rows to draw one number. The count leaks nothing: its
 * denominator is exactly the set the caller may read, which is what separates it
 * from the B10 SeatProblemCount regression, where an unfiltered count published
 * how many unscreened reports existed.
 * @typedef {Object} UnreadCount
 * @property {number} count
 */

export {}
