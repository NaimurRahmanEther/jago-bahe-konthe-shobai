package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"jago-bahe-backend/config"
	assignmentapp "jago-bahe-backend/internal/assignment/application"
	assignmentdomain "jago-bahe-backend/internal/assignment/domain"
	assignmentpg "jago-bahe-backend/internal/assignment/infrastructure/postgres"
	assignmenthttp "jago-bahe-backend/internal/assignment/interfaces/http"
	identityapp "jago-bahe-backend/internal/identity/application"
	identitydomain "jago-bahe-backend/internal/identity/domain"
	identitypg "jago-bahe-backend/internal/identity/infrastructure/postgres"
	identityhttp "jago-bahe-backend/internal/identity/interfaces/http"
	problemapp "jago-bahe-backend/internal/problem/application"
	problemdomain "jago-bahe-backend/internal/problem/domain"
	problempg "jago-bahe-backend/internal/problem/infrastructure/postgres"
	problemhttp "jago-bahe-backend/internal/problem/interfaces/http"
	resolutionapp "jago-bahe-backend/internal/resolution/application"
	resolutiondomain "jago-bahe-backend/internal/resolution/domain"
	resolutionpg "jago-bahe-backend/internal/resolution/infrastructure/postgres"
	resolutionhttp "jago-bahe-backend/internal/resolution/interfaces/http"
	scorecardapp "jago-bahe-backend/internal/scorecard/application"
	scorecardpg "jago-bahe-backend/internal/scorecard/infrastructure/postgres"
	scorecardhttp "jago-bahe-backend/internal/scorecard/interfaces/http"
	areaapp "jago-bahe-backend/internal/shared/area/application"
	areapg "jago-bahe-backend/internal/shared/area/infrastructure/postgres"
	areahttp "jago-bahe-backend/internal/shared/area/interfaces/http"
	auditapp "jago-bahe-backend/internal/shared/audit/application"
	auditpg "jago-bahe-backend/internal/shared/audit/infrastructure/postgres"
	audithttp "jago-bahe-backend/internal/shared/audit/interfaces/http"
	notificationapp "jago-bahe-backend/internal/shared/notification/application"
	notificationpg "jago-bahe-backend/internal/shared/notification/infrastructure/postgres"
	notificationhttp "jago-bahe-backend/internal/shared/notification/interfaces/http"
	suggestionapp "jago-bahe-backend/internal/suggestion/application"
	suggestiondomain "jago-bahe-backend/internal/suggestion/domain"
	suggestionpg "jago-bahe-backend/internal/suggestion/infrastructure/postgres"
	suggestionhttp "jago-bahe-backend/internal/suggestion/interfaces/http"
	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
	"jago-bahe-backend/pkg/security"
)

// newRouter is the composition root: it constructs the concrete Postgres
// repositories, wires them into each context's handlers, mounts the routes, and
// wraps everything in the middleware chain. It is the single place concrete
// types are bound to interfaces — and, being pure (cfg + pool + log in, handler
// out), it is what the integration test drives end-to-end against a real DB.
func newRouter(cfg *config.Config, pool *pgxpool.Pool, log *slog.Logger) http.Handler {
	// Shared kernel repositories.
	areaRepo := areapg.NewAreaRepository(pool)
	auditRepo := auditpg.NewAuditRepository(pool)

	// B21: the in-app notification log. Like auditRepo it is threaded directly into
	// the use cases that write it — seven of them, across problem, resolution and
	// assignment — with no port and no adapter, because a shared-kernel package may
	// be imported by a context (A.4.1 forbids importing another CONTEXT). If you
	// find yourself writing a Notifier interface in some context's ports.go, check
	// that rule first: approve_problem.go has held an auditdomain.Repository
	// directly since B9 and this is the same shape.
	notificationRepo := notificationpg.NewNotificationRepository(pool)

	// Hoisted out of the assignment context below: the PROBLEM handler needs it
	// for its Assignments port (publishing which official a problem was actually
	// handed to), and it is constructed further down than that handler is built.
	// If you reorder this back down, the wiring stops compiling — do not "fix"
	// that by importing assignment from problem, which is the dependency arrow
	// the contexts are separated to prevent. Add the port's adapter instead.
	assignmentRepo := assignmentpg.NewAssignmentRepository(pool)

	// Area context (B13 — the seat's public geography). The repository already
	// existed for ResolveUnion; this only exposes its List over HTTP.
	areaHandler := areahttp.NewHandler(areaapp.NewListAreas(areaRepo))

	// Auth primitives.
	tokens := auth.NewManager(cfg.JWTSecret, cfg.JWTTTL)
	hasher := security.NewBcryptHasher()
	if cfg.Production {
		hasher = security.NewProductionBcryptHasher()
	}

	// Identity context (B1; B11 added official claims, resident verification, and
	// the super admin's oversight read).
	accountRepo := identitypg.NewAccountRepository(pool)
	officialRepo := identitypg.NewOfficialRepository(pool)
	claimRepo := identitypg.NewClaimRepository(pool)
	identitySvc := identitydomain.NewService()
	identityHandler := identityhttp.NewHandler(
		identityapp.NewRegisterResident(accountRepo, hasher, tokens, auditRepo),
		identityapp.NewRegisterOfficial(accountRepo, officialRepo, claimRepo, hasher, tokens, auditRepo),
		identityapp.NewLogin(accountRepo, identitySvc, hasher, tokens),
		identityapp.NewListOfficials(officialRepo),
		identityapp.NewGetOfficial(officialRepo),
		identityapp.NewReviewClaims(accountRepo, officialRepo, claimRepo, areaRepo, auditRepo),
		identityapp.NewVerifyResident(accountRepo, officialRepo, areaRepo, auditRepo),
		identityapp.NewOversight(auditRepo),
	)

	// Problem context (B2 — the reference slice). A report starts PendingApproval
	// and is published only when its union admin approves it (or rejected on a fixed
	// ground); pre-assignment takedown of an already-public report remains.
	idAdapter := identityAdapter{accounts: accountRepo, officials: officialRepo}
	problemRepo := problempg.NewProblemRepository(pool)
	problemSvc := problemdomain.NewService()
	problemScreening := problemdomain.NewScreening()
	problemAdmins := problemAdminAdapter{accounts: accountRepo, officials: officialRepo, areas: areaRepo}
	problemAssignments := problemAssignmentAdapter{assignments: assignmentRepo}
	problemHandler := problemhttp.NewHandler(
		problemapp.NewReportProblem(problemRepo, areaRepo, idAdapter, auditRepo),
		problemapp.NewListProblems(problemRepo),
		problemapp.NewListMyProblems(problemRepo),
		problemapp.NewGetProblem(problemRepo, areaRepo, problemAdmins, problemAssignments, auditRepo),
		problemapp.NewCastValidationVote(problemRepo, areaRepo, idAdapter, auditRepo, problemSvc, cfg.ValidityThreshold),
		problemapp.NewUpdateProblem(problemRepo, auditRepo),
		problemapp.NewWithdrawProblem(problemRepo, auditRepo),
		problemapp.NewDeleteProblem(problemRepo, auditRepo),
		problemapp.NewListMyVotes(problemRepo),
		problemapp.NewListAssignments(problemAssignments),
		problemapp.NewListModeration(problemRepo, areaRepo, problemAdmins),
		problemapp.NewApproveProblem(problemRepo, areaRepo, problemAdmins, auditRepo, notificationRepo, problemScreening),
		problemapp.NewRejectProblem(problemRepo, areaRepo, problemAdmins, auditRepo, notificationRepo, problemScreening),
		cfg.ValidityThreshold,
	)

	// Suggestion context (B3).
	suggestionRepo := suggestionpg.NewSuggestionRepository(pool)
	suggestionSvc := suggestiondomain.NewService()
	sugIdentity := suggestionIdentityAdapter{accounts: accountRepo}
	problemArea := problemAreaAdapter{problems: problemRepo}
	suggestionHandler := suggestionhttp.NewHandler(
		suggestionapp.NewListSuggestions(suggestionRepo, suggestionSvc),
		suggestionapp.NewProposeSuggestion(suggestionRepo, areaRepo, problemArea, sugIdentity, auditRepo),
		suggestionapp.NewUpvoteSuggestion(suggestionRepo, areaRepo, problemArea, sugIdentity, suggestionSvc),
		suggestionapp.NewListMyUpvotes(suggestionRepo),
	)

	// Assignment context (B4 — routing + admin decisions). assignmentRepo is
	// constructed at the top of this function — the problem handler above needs it.
	assignmentSvc := assignmentdomain.NewService()
	asgnProblems := assignmentProblemAdapter{problems: problemRepo}
	asgnOfficials := assignmentOfficialAdapter{officials: officialRepo, areas: areaRepo}
	asgnAdmins := assignmentAdminAdapter{accounts: accountRepo, officials: officialRepo, areas: areaRepo}
	assignmentHandler := assignmenthttp.NewHandler(
		assignmentapp.NewListQueue(asgnProblems, asgnOfficials, asgnAdmins, areaRepo, assignmentSvc),
		assignmentapp.NewAssignWithinUnion(assignmentRepo, asgnProblems, asgnOfficials, asgnAdmins, areaRepo, assignmentSvc, auditRepo, notificationRepo, cfg.ResponseDeadline),
		// B20: above-union forwarding is the super admin's decision, advised by the
		// union admins. This replaced the binding vote (quorum Q, window W) — see
		// CLAUDE.md A.3.8.
		assignmentapp.NewSuggestForwarding(assignmentRepo, asgnProblems, asgnOfficials, asgnAdmins, assignmentSvc, auditRepo),
		assignmentapp.NewListMyForwarding(assignmentRepo, asgnProblems, asgnOfficials, asgnAdmins, assignmentSvc),
		assignmentapp.NewListForwardingQueue(assignmentRepo, asgnProblems, asgnOfficials, assignmentSvc),
		assignmentapp.NewForwardProblem(assignmentRepo, asgnProblems, asgnOfficials, assignmentSvc, auditRepo, notificationRepo, cfg.ResponseDeadline),
		// V, echoed on each queue row for the admin's "X of V" display. It gates
		// nothing (A.3.1) — same reason the problem handler carries it.
		cfg.ValidityThreshold,
	)

	// Resolution context (B5 — the lifecycle + obstacle judgment).
	caseRepo := resolutionpg.NewCaseRepository(pool)
	resolutionSvc := resolutiondomain.NewService()
	resAssignments := resolutionAssignmentAdapter{assignments: assignmentRepo}
	resProblems := resolutionProblemAdapter{problems: problemRepo}
	resIdentity := resolutionIdentityAdapter{accounts: accountRepo}
	resSuggestions := resolutionSuggestionAdapter{suggestions: suggestionRepo, svc: suggestionSvc}
	// B19: the officials directory + the accountability ladder, for the monitor's
	// observation list. The ladder walk itself lives in identity (A.4.1).
	resOfficials := resolutionOfficialAdapter{officials: officialRepo, areas: areaRepo}
	caseHandler := resolutionhttp.NewCaseHandler(
		resolutionapp.NewListCases(caseRepo, resAssignments),
		resolutionapp.NewGetCase(caseRepo),
		resolutionapp.NewAcknowledgeCase(caseRepo, auditRepo),
		resolutionapp.NewSubmitPlan(caseRepo, resSuggestions, auditRepo),
		resolutionapp.NewRevisePlan(caseRepo, resSuggestions, resProblems, auditRepo),
		resolutionapp.NewPostUpdate(caseRepo, resProblems, auditRepo),
		resolutionapp.NewUploadEvidence(caseRepo, auditRepo),
		resolutionapp.NewMarkDone(caseRepo, resProblems, auditRepo, notificationRepo),
		resolutionapp.NewReportObstacle(caseRepo, resProblems, auditRepo, notificationRepo),
		resolutionapp.NewCompleteTask(caseRepo, auditRepo),
	)
	obstacleHandler := resolutionhttp.NewObstacleHandler(
		resolutionapp.NewGetObstacle(caseRepo, resolutionSvc),
		resolutionapp.NewVoteObstacle(caseRepo, areaRepo, resProblems, resIdentity, auditRepo),
		resolutionapp.NewProposeUnblockingPlan(caseRepo, areaRepo, resProblems, resIdentity, auditRepo),
		resolutionapp.NewUpvoteUnblockingPlan(caseRepo, areaRepo, resProblems, resIdentity),
	)
	confirmHandler := resolutionhttp.NewConfirmHandler(
		resolutionapp.NewConfirmResolution(caseRepo, resProblems, auditRepo, notificationRepo),
	)
	adjudicateHandler := resolutionhttp.NewAdjudicateHandler(
		resolutionapp.NewAdjudicateObstacle(caseRepo, resProblems, auditRepo, notificationRepo),
	)
	// Audit context — the seat's PUBLIC decision record. It reads the same audit
	// log the super admin's oversight feed does, over a narrower action set: the
	// five that are about problems, never the three that are about people (see
	// audit/application/list_activity.go for why that line is where it is).
	//
	// Constructed HERE, after accountRepo/officialRepo/problemRepo, because its
	// adapters need all three. The two adapters live in the composition root rather
	// than inside the audit context because `shared/audit` sits under every context
	// and must not import identity or problem (A.4.1) — if this ever fails to
	// compile, move the construction, never the import.
	activityHandler := audithttp.NewHandler(
		auditapp.NewListActivity(auditRepo),
		auditActorAdapter{accounts: accountRepo, officials: officialRepo},
		auditProblemAdapter{problems: problemRepo},
	)

	// B12: the public per-problem progress read. Needs only the case repo — the
	// point of the endpoint is that it never reads the problem row.
	progressHandler := resolutionhttp.NewProgressHandler(
		resolutionapp.NewGetProgress(caseRepo),
	)

	// B19: the monitor's observation list — the read side of escalation. The worker
	// opens the rows; this is where the official above the silent one reads them.
	observationHandler := resolutionhttp.NewObservationHandler(
		resolutionapp.NewListObservations(caseRepo, resProblems, resOfficials),
		resolutionapp.NewNoteObservation(caseRepo, auditRepo),
	)

	// B21: the caller's own in-app notifications. Four reads and two of the
	// smallest possible writes; it needs nothing but its own repository, because a
	// notification snapshots its title and detail at write time and so resolves
	// nothing at read time.
	notificationHandler := notificationhttp.NewHandler(
		notificationapp.NewListNotifications(notificationRepo),
		notificationapp.NewUnreadCount(notificationRepo),
		notificationapp.NewMarkRead(notificationRepo),
		notificationapp.NewMarkAllRead(notificationRepo),
	)

	// Scorecard context (B7 — the public accountability read model; B10 added the
	// official's public record and routed the seat overview).
	scorecardQuery := scorecardpg.NewScorecardQuery(pool)
	scorecardHandler := scorecardhttp.NewHandler(
		scorecardapp.NewOfficialScorecard(scorecardQuery),
		scorecardapp.NewOfficialRecord(scorecardQuery),
		scorecardapp.NewSeatOverview(scorecardQuery),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler(pool))
	identityHandler.Routes(mux, auth.RequireAdmin(), auth.RequireSuperAdmin(), auth.RequireClaimReviewer())
	problemHandler.Routes(mux, auth.RequirePublic(), auth.RequireAdmin())
	suggestionHandler.Routes(mux, auth.RequirePublic())
	assignmentHandler.Routes(mux, auth.RequireAdmin(), auth.RequireSuperAdmin())
	caseHandler.Routes(mux, auth.RequireOfficial())
	observationHandler.Routes(mux, auth.RequireOfficial())
	obstacleHandler.Routes(mux, auth.RequirePublic())
	confirmHandler.Routes(mux, auth.RequirePublic())
	adjudicateHandler.Routes(mux, auth.RequireAdmin())
	progressHandler.Routes(mux)
	scorecardHandler.Routes(mux)
	areaHandler.Routes(mux)
	activityHandler.Routes(mux)
	// The only route group behind RequireAnyRole: a notification is about the
	// CALLER rather than about their office, so all four roles need it and no
	// existing gate admits all four.
	notificationHandler.Routes(mux, auth.RequireAnyRole())

	return httpx.Chain(mux,
		httpx.RequestID(),
		httpx.Recoverer(log),
		httpx.Logger(log),
		httpx.CORS(cfg.AllowedOrigins),
		auth.Authenticate(tokens), // best-effort: populates claims for role gates
	)
}

// healthHandler reports service liveness and database reachability.
func healthHandler(pinger interface {
	Ping(context.Context) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbOK := true
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pinger.Ping(pingCtx); err != nil {
			dbOK = false
		}
		status := http.StatusOK
		if !dbOK {
			status = http.StatusServiceUnavailable
		}
		httpx.JSON(w, status, map[string]any{
			"status": "ok",
			"db":     dbOK,
		})
	}
}
