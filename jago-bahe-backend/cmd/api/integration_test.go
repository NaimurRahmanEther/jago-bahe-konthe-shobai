package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"jago-bahe-backend/config"
	identitypg "jago-bahe-backend/internal/identity/infrastructure/postgres"
	resolutionapp "jago-bahe-backend/internal/resolution/application"
	resolutionpg "jago-bahe-backend/internal/resolution/infrastructure/postgres"
	areapg "jago-bahe-backend/internal/shared/area/infrastructure/postgres"
	auditpg "jago-bahe-backend/internal/shared/audit/infrastructure/postgres"
	"jago-bahe-backend/pkg/postgres"
)

// This integration test drives the whole HTTP stack — real router, middleware,
// auth gates, DTO mapping, and the real Postgres repositories — across a full
// vertical slice (register → login → report → validate → assign → read), plus the
// hardening guardrails (missing-field 400, unauthenticated 401) and the screening
// gate in full: a new report starts PendingApproval and is hidden from the feed
// and unvotable until its own union's admin approves it; a stranger and a
// foreign-union admin get 404; approval publishes it; rejection on a fixed ground
// makes it public as Rejected with its reason; and the rejectable window closes at
// assignment. It runs only when TEST_DATABASE_URL points at a migrated, seeded
// database; otherwise it skips, so `go test ./...` stays green without a database.
//
//   createdb jago_test
//   DATABASE_URL=postgres://postgres:pw@localhost:5432/jago_test?sslmode=disable go run ./cmd/migrate up
//   TEST_DATABASE_URL=postgres://postgres:pw@localhost:5432/jago_test?sslmode=disable go test ./cmd/api -run TestIntegration -v

func TestIntegration_ProblemSlice(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("set TEST_DATABASE_URL (a migrated, seeded DB) to run the integration test")
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Close via t.Cleanup, not defer: a deferred Close runs when this function
	// returns, which is BEFORE testing runs the t.Cleanup below, so the row cleanup
	// would delete against an already-closed pool and silently leak every 'integration
	// test %' row (each run leaving an assigned problem that later trips the live
	// suite). t.Cleanup is LIFO, and this one is registered first, so it runs last —
	// after the data cleanup, with the pool still open.
	t.Cleanup(func() { pool.Close() })

	// V=1 so a single seeded verified resident (acct-seed-res-1, union-dhamoirhat) can
	// carry a problem to Validated; every other tunable is a valid default. The gate
	// is a hard one — nothing auto-publishes, so a report stays PendingApproval until
	// the admin approves it in step 8.
	cfg := &config.Config{
		DatabaseURL: dbURL, JWTSecret: "integration-secret", JWTTTL: time.Hour,
		ValidityThreshold: 1,
		ResponseDeadline:  7 * 24 * time.Hour, BlockerReviewWindow: 72 * time.Hour,
	}
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(cfg, pool, quiet)

	call := func(method, path, token string, body any) (int, map[string]any) {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.RemoteAddr = "127.0.0.1:5555"
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}

	// callList is the array-returning twin of call (feeds and queues return a JSON
	// array, not an object).
	callList := func(method, path, token string) (int, []any) {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = "127.0.0.1:5555"
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var out []any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}

	phone := fmt.Sprintf("019%08d", rand.Intn(1e8))
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM validation_votes WHERE problem_id IN (
			SELECT id FROM problems WHERE title LIKE 'integration test %')`)
		_, _ = pool.Exec(ctx, `DELETE FROM audit_entries WHERE target_id IN (
			SELECT id FROM problems WHERE title LIKE 'integration test %')`)
		_, _ = pool.Exec(ctx, `DELETE FROM problems WHERE title LIKE 'integration test %'`)
		_, _ = pool.Exec(ctx, `DELETE FROM audit_entries WHERE actor IN (SELECT id FROM accounts WHERE phone = $1)`, phone)
		_, _ = pool.Exec(ctx, `DELETE FROM accounts WHERE phone = $1`, phone)
	})

	// 1) Register a fresh resident. Their token doubles as a second identity for
	//    B12's own-view scoping check below.
	code, body := call(http.MethodPost, "/api/auth/register", "", map[string]any{
		"name": "Integration Tester", "phone": phone, "password": "test1234",
		"nid": "1990010112345", "unionId": "union-dhamoirhat",
	})
	if code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d (%v)", code, body)
	}
	otherResidentToken, _ := body["token"].(string)
	if otherResidentToken == "" {
		t.Fatal("register returned no token")
	}

	// 2) Missing-field validation is rejected with a 400.
	if code, _ := call(http.MethodPost, "/api/auth/register", "", map[string]any{
		"name": "No Phone", "password": "test1234", "nid": "1", "unionId": "union-dhamoirhat",
	}); code != http.StatusBadRequest {
		t.Fatalf("register missing phone: want 400, got %d", code)
	}

	// 3) Login as the seeded verified resident (union-dhamoirhat) — the one who can validate.
	code, body = call(http.MethodPost, "/api/auth/login", "", map[string]any{
		"phone": "01810000001", "password": "resident123",
	})
	if code != http.StatusOK {
		t.Fatalf("login: want 200, got %d (%v)", code, body)
	}
	residentToken, _ := body["token"].(string)
	if residentToken == "" {
		t.Fatal("login returned no token")
	}

	// 4) Unauthenticated write is rejected by the role gate.
	if code, _ := call(http.MethodPost, "/api/problems", "", map[string]any{
		"title": "x", "description": "y",
		"location": map[string]any{"areaId": "union-dhamoirhat"}, "pointedOfficialId": "off-chair-dhamoirhat",
	}); code != http.StatusUnauthorized {
		t.Fatalf("anonymous report: want 401, got %d", code)
	}

	// 5) Report a problem (missing title first → 400, then the real one → 201).
	if code, _ := call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"description": "no title", "location": map[string]any{"areaId": "union-dhamoirhat"}, "pointedOfficialId": "off-chair-dhamoirhat",
	}); code != http.StatusBadRequest {
		t.Fatalf("report missing title: want 400, got %d", code)
	}
	code, body = call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"title": "integration test pothole", "description": "deep pothole on the union road",
		"location": map[string]any{"areaId": "union-dhamoirhat", "address": "Ward 1"}, "pointedOfficialId": "off-chair-dhamoirhat",
	})
	if code != http.StatusCreated {
		t.Fatalf("report: want 201, got %d (%v)", code, body)
	}
	problemID, _ := body["id"].(string)
	// A new problem starts PendingApproval: held invisible to the public until its
	// union's admin approves it. The gate keeps spam and defamation off the public
	// record until a human has seen them.
	if body["status"] != "PendingApproval" {
		t.Fatalf("new problem should be PendingApproval, got %v", body["status"])
	}

	// 6) It is hidden until approved: absent from the anonymous feed, and 404 to any
	//    stranger — the refusal is not-found, not forbidden, so no one can confirm a
	//    hidden report exists at this id.
	code, feed := callList(http.MethodGet, "/api/problems", "")
	if code != http.StatusOK {
		t.Fatalf("feed: want 200, got %d", code)
	}
	if containsID(feed, problemID) {
		t.Fatal("a pending problem must NOT appear in the anonymous feed until it is approved")
	}
	if code, _ := call(http.MethodGet, "/api/problems/"+problemID, "", nil); code != http.StatusNotFound {
		t.Fatalf("anonymous read of a pending problem: want 404, got %d", code)
	}
	if code, _ := call(http.MethodGet, "/api/problems/"+problemID+"/audit", "", nil); code != http.StatusNotFound {
		t.Fatalf("anonymous read of a pending problem's audit: want 404, got %d", code)
	}
	// Not votable yet: a resident cannot validate — or even learn of — a report
	// that has not cleared screening.
	if code, _ := call(http.MethodPost, "/api/problems/"+problemID+"/validate", residentToken, map[string]any{"vote": "valid"}); code != http.StatusNotFound {
		t.Fatalf("validating a pending problem: want 404, got %d", code)
	}
	// Its own reporter can always see it, though — otherwise the report would be
	// visible nowhere to the person who filed it.
	code, own := call(http.MethodGet, "/api/problems/"+problemID, residentToken, nil)
	if code != http.StatusOK {
		t.Fatalf("reporter reading own pending problem: want 200, got %d", code)
	}
	reporterID, _ := own["reporterId"].(string)
	// PendingApproval is not a publicly listable status: the feed refuses it.
	if code, _ := call(http.MethodGet, "/api/problems?status=PendingApproval", "", nil); code != http.StatusBadRequest {
		t.Fatalf("feed filtered to PendingApproval: want 400, got %d", code)
	}

	// B12: the reporter's own view lists what they filed, at every status — this is
	// where the author follows a pending report.
	code, mine := callList(http.MethodGet, "/api/me/problems", residentToken)
	if code != http.StatusOK {
		t.Fatalf("own view: want 200, got %d", code)
	}
	if !containsID(mine, problemID) {
		t.Fatal("the reporter's own pending problem must appear in /api/me/problems")
	}
	// Another resident's own view is theirs alone: no reporter parameter exists, so
	// there is no way to point this at someone else.
	if code, theirs := callList(http.MethodGet, "/api/me/problems", otherResidentToken); code != http.StatusOK || containsID(theirs, problemID) {
		t.Fatalf("a second resident must not see another's report in their own view: got %d / present=%v", code, containsID(theirs, problemID))
	}
	// Anonymous is a 401, never a silently empty list that reads as "you filed nothing".
	if code, _ := callList(http.MethodGet, "/api/me/problems", ""); code != http.StatusUnauthorized {
		t.Fatalf("anonymous own view: want 401, got %d", code)
	}
	// The feed ignores a reporter param — if one is ever added, this fails.
	if code, scoped := callList(http.MethodGet, "/api/problems?reporter="+reporterID, ""); code != http.StatusOK || len(scoped) != len(feed) {
		t.Fatalf("GET /api/problems must ignore ?reporter=: got %d rows, want the unfiltered %d", len(scoped), len(feed))
	}

	// B12: progress is null-shaped, never not-found. An unassigned problem and an
	// id that never existed must be indistinguishable.
	codeCaseless, caselessProgress := call(http.MethodGet, "/api/problems/"+problemID+"/progress", "", nil)
	codeUnknown, unknownProgress := call(http.MethodGet, "/api/problems/deadbeef/progress", "", nil)
	if codeCaseless != http.StatusOK || codeUnknown != http.StatusOK {
		t.Fatalf("progress: want 200/200, got %d/%d", codeCaseless, codeUnknown)
	}
	if len(caselessProgress) != 0 || len(unknownProgress) != 0 {
		t.Fatalf("progress for a caseless problem must be null: caseless=%v unknown=%v", caselessProgress, unknownProgress)
	}

	// A reporter may withdraw their own report even while it is pending — it is
	// theirs until an admin acts. Use a dedicated problem so problemID stays on its
	// approve→validate→assign path below.
	code, body = call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"title": "integration test withdraw me", "description": "filed by mistake",
		"location": map[string]any{"areaId": "union-dhamoirhat"}, "pointedOfficialId": "off-chair-dhamoirhat",
	})
	if code != http.StatusCreated {
		t.Fatalf("report to withdraw: want 201, got %d (%v)", code, body)
	}
	withdrawID, _ := body["id"].(string)
	if code, _ := call(http.MethodPost, "/api/problems/"+withdrawID+"/withdraw", otherResidentToken, map[string]any{}); code != http.StatusForbidden {
		t.Fatalf("withdraw by a non-reporter: want 403, got %d", code)
	}
	code, body = call(http.MethodPost, "/api/problems/"+withdrawID+"/withdraw", residentToken, map[string]any{"note": "duplicate report"})
	if code != http.StatusOK || body["status"] != "Withdrawn" {
		t.Fatalf("withdraw: want 200/Withdrawn, got %d/%v", code, body["status"])
	}
	// A withdrawn problem is public (it stays on the record as Withdrawn), even
	// though it never was while pending.
	if code, back := call(http.MethodGet, "/api/problems/"+withdrawID, "", nil); code != http.StatusOK || back["status"] != "Withdrawn" {
		t.Fatalf("a withdrawn problem stays publicly readable as Withdrawn: got %d/%v", code, back["status"])
	}
	if code, wfeed := callList(http.MethodGet, "/api/problems?status=Withdrawn", ""); code != http.StatusOK || !containsID(wfeed, withdrawID) {
		t.Fatalf("withdrawn problems must be listable publicly (got %d)", code)
	}
	if code, _ := call(http.MethodPost, "/api/problems/"+withdrawID+"/withdraw", residentToken, map[string]any{}); code != http.StatusConflict {
		t.Fatalf("re-withdrawing: want 409, got %d", code)
	}

	// 7) Screening is union-scoped: the Agradigun admin must not see or touch a
	//    pending report belonging to Dhamoirhat union.
	code, body = call(http.MethodPost, "/api/auth/login", "", map[string]any{"phone": "01910000001", "password": "admin123"})
	if code != http.StatusOK {
		t.Fatalf("dhamoirhat admin login: want 200, got %d (%v)", code, body)
	}
	adminToken, _ := body["token"].(string)
	adminUser, _ := body["user"].(map[string]any)
	adminID, _ := adminUser["id"].(string)

	code, body = call(http.MethodPost, "/api/auth/login", "", map[string]any{"phone": "01910000002", "password": "admin123"})
	if code != http.StatusOK {
		t.Fatalf("agradigun admin login: want 200, got %d (%v)", code, body)
	}
	otherAdminToken, _ := body["token"].(string)

	// A foreign-union admin cannot even read the pending report (404, like a
	// stranger), let alone approve or reject it.
	if code, _ := call(http.MethodGet, "/api/problems/"+problemID, otherAdminToken, nil); code != http.StatusNotFound {
		t.Fatalf("a foreign-union admin reading a pending report: want 404, got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+problemID+"/approve", otherAdminToken, nil); code != http.StatusForbidden {
		t.Fatalf("approve by another union's admin: want 403, got %d", code)
	}
	if code, q := callList(http.MethodGet, "/api/admin/moderation", otherAdminToken); code != http.StatusOK || containsID(q, problemID) {
		t.Fatalf("another union's screening queue must not list this problem (got %d)", code)
	}

	// 8) The right admin sees it awaiting screening and can read it in full.
	code, queue := callList(http.MethodGet, "/api/admin/moderation", adminToken)
	if code != http.StatusOK || !containsID(queue, problemID) {
		t.Fatalf("screening queue should list the pending problem, got %d/%v", code, queue)
	}
	if code, _ := call(http.MethodGet, "/api/problems/"+problemID, adminToken, nil); code != http.StatusOK {
		t.Fatalf("the own-union admin must be able to read the pending report: got %d", code)
	}

	// Approve it: it becomes Reported (public but unvalidated) and enters the feed.
	code, body = call(http.MethodPost, "/api/admin/problems/"+problemID+"/approve", adminToken, nil)
	if code != http.StatusOK || body["status"] != "Reported" {
		t.Fatalf("approve: want 200/Reported, got %d/%v", code, body["status"])
	}
	if code, f := callList(http.MethodGet, "/api/problems", ""); code != http.StatusOK || !containsID(f, problemID) {
		t.Fatalf("an approved problem must appear in the anonymous feed (got %d)", code)
	}
	if code, anon := call(http.MethodGet, "/api/problems/"+problemID, "", nil); code != http.StatusOK || anon["status"] != "Reported" {
		t.Fatalf("anonymous read of an approved problem: want 200/Reported, got %d/%v", code, anon["status"])
	}
	// It leaves the screening queue — the decision has been made.
	if code, q := callList(http.MethodGet, "/api/admin/moderation", adminToken); code != http.StatusOK || containsID(q, problemID) {
		t.Fatalf("an approved problem must leave the screening queue (got %d)", code)
	}

	// 8a) Hard delete: the reporter erases their own report outright. Unlike withdraw
	//     above, nothing survives — not the row, not the audit trail. Use a dedicated
	//     problem, approved so it is public and has a real audit trail to erase.
	code, body = call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"title": "integration test delete me", "description": "to be erased",
		"location": map[string]any{"areaId": "union-dhamoirhat"}, "pointedOfficialId": "off-chair-dhamoirhat",
	})
	if code != http.StatusCreated {
		t.Fatalf("report to delete: want 201, got %d (%v)", code, body)
	}
	deleteID, _ := body["id"].(string)
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+deleteID+"/approve", adminToken, nil); code != http.StatusOK {
		t.Fatalf("approve the problem to be deleted: want 200, got %d", code)
	}
	// It has an audit trail before the delete — otherwise the "trail is gone"
	// assertion below would pass against a problem that never had one.
	if code, trail := callList(http.MethodGet, "/api/problems/"+deleteID+"/audit", ""); code != http.StatusOK || len(trail) == 0 {
		t.Fatalf("the problem should carry an audit trail before deletion: got %d/%v", code, trail)
	}
	if code, _ := call(http.MethodDelete, "/api/problems/"+deleteID, otherResidentToken, nil); code != http.StatusForbidden {
		t.Fatalf("delete by a non-reporter: want 403, got %d", code)
	}
	if code, _ := call(http.MethodDelete, "/api/problems/"+deleteID, "", nil); code != http.StatusUnauthorized {
		t.Fatalf("anonymous delete: want 401, got %d", code)
	}
	// Still there after both refusals — a refused delete must not half-erase.
	if code, _ := call(http.MethodGet, "/api/problems/"+deleteID, "", nil); code != http.StatusOK {
		t.Fatalf("a refused delete must leave the problem intact: got %d", code)
	}
	if code, _ := call(http.MethodDelete, "/api/problems/"+deleteID, residentToken, nil); code != http.StatusNoContent {
		t.Fatalf("delete by the reporter: want 204, got %d", code)
	}
	if code, _ := call(http.MethodGet, "/api/problems/"+deleteID, residentToken, nil); code != http.StatusNotFound {
		t.Fatalf("a deleted problem must be gone even for its reporter: got %d", code)
	}
	// The audit trail went with it. This is the sharpest consequence of the hard
	// delete and the one most likely to be "fixed" by someone who thinks the log
	// should be append-only — it is, everywhere except here (CLAUDE.md A.3.3).
	if code, _ := call(http.MethodGet, "/api/problems/"+deleteID+"/audit", residentToken, nil); code != http.StatusNotFound {
		t.Fatalf("a deleted problem's audit trail must be gone too: got %d", code)
	}
	if code, f := callList(http.MethodGet, "/api/problems", ""); code != http.StatusOK || containsID(f, deleteID) {
		t.Fatalf("a deleted problem must leave the public feed (got %d)", code)
	}
	if code, m := callList(http.MethodGet, "/api/me/problems", residentToken); code != http.StatusOK || containsID(m, deleteID) {
		t.Fatalf("a deleted problem must leave its reporter's own view (got %d)", code)
	}
	if code, _ := call(http.MethodDelete, "/api/problems/"+deleteID, residentToken, nil); code != http.StatusNotFound {
		t.Fatalf("re-deleting: want 404, got %d", code)
	}

	// 8b) B17: an approved report enters the ASSIGNMENT queue at once, with zero
	//     validations, carrying the count the admin judges by. Before B17 the queue
	//     held Validated problems only, so the admin was blind between approving a
	//     report and its reaching V — the queue was binary, absent then present.
	code, queue = callList(http.MethodGet, "/api/admin/queue", adminToken)
	if code != http.StatusOK {
		t.Fatalf("admin queue: want 200, got %d", code)
	}
	row := findQueueItem(queue, problemID)
	if row == nil {
		t.Fatalf("an approved Reported problem must be in the assignment queue, got %v", queue)
	}
	if row["status"] != "Reported" {
		t.Fatalf("queue row status = %v, want Reported", row["status"])
	}
	// The fields the UI reads off a queue row. Every one was absent from the DTO
	// before B17 while the UI already read it, so `address` came back undefined and
	// crashed the row, and `routing` silently sent every above-union problem to the
	// union-assign screen. Assert their presence, not just the count.
	if row["address"] == nil || row["address"] == "" {
		t.Fatalf("queue row must carry the address the UI renders, got %v", row)
	}
	if row["routing"] != "union" {
		t.Fatalf("queue row routing = %v, want union (a union chairman is union-level)", row["routing"])
	}
	if row["validCount"] != float64(0) {
		t.Fatalf("queue row validCount = %v, want 0 — it is not yet validated", row["validCount"])
	}
	if row["validationThreshold"] == nil {
		t.Fatalf("queue row must echo V for the admin's X-of-V display, got %v", row)
	}

	// The queue is scoped to the acting admin's own union. It used to ignore the
	// caller entirely and hand every admin every union's problems; scoping lived
	// only at the assign step, so the list leaked what the action would refuse.
	if code, q := callList(http.MethodGet, "/api/admin/queue", otherAdminToken); code != http.StatusOK {
		t.Fatalf("other union's queue: want 200, got %d", code)
	} else if findQueueItem(q, problemID) != nil {
		t.Fatalf("another union's admin must not see this problem in their queue, got %v", q)
	}

	// 8c) B17's load-bearing case: forward a report to an official with ZERO
	//     validations. The community's votes are evidence the admin weighs, not a
	//     condition they wait on (A.3.1) — this returned 409 not_validated before
	//     B17. A separate problem, since problemID goes on to be validated below.
	code, unvalidated := call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"title": "integration test broken streetlight", "description": "dark since last week",
		"location": map[string]any{"areaId": "union-dhamoirhat", "address": "Ward 2"}, "pointedOfficialId": "off-chair-dhamoirhat",
	})
	if code != http.StatusCreated {
		t.Fatalf("report the zero-vote problem: want 201, got %d (%v)", code, unvalidated)
	}
	unvalidatedID, _ := unvalidated["id"].(string)
	// Unscreened, it can be forwarded to nobody: screening still precedes assignment.
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+unvalidatedID+"/assign", adminToken, map[string]any{
		"officialId": "off-chair-dhamoirhat", "priority": "normal",
	}); code != http.StatusConflict {
		t.Fatalf("assigning a PendingApproval problem: want 409 not_assignable, got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+unvalidatedID+"/approve", adminToken, nil); code != http.StatusOK {
		t.Fatalf("approve the zero-vote problem: got %d", code)
	}
	code, assigned := call(http.MethodPost, "/api/admin/problems/"+unvalidatedID+"/assign", adminToken, map[string]any{
		"officialId": "off-chair-dhamoirhat", "priority": "normal",
	})
	if code != http.StatusCreated {
		t.Fatalf("forwarding a Reported problem with zero votes: want 201, got %d (%v)", code, assigned)
	}
	if code, p := call(http.MethodGet, "/api/problems/"+unvalidatedID, "", nil); code != http.StatusOK || p["status"] != "Assigned" {
		t.Fatalf("the zero-vote problem should now be Assigned, got %d/%v", code, p["status"])
	}
	// And it leaves the queue — a case exists, so it is no longer forwardable.
	if code, q := callList(http.MethodGet, "/api/admin/queue", adminToken); code != http.StatusOK {
		t.Fatalf("admin queue after assigning: want 200, got %d", code)
	} else if findQueueItem(q, unvalidatedID) != nil {
		t.Fatalf("an assigned problem must leave the queue, got %v", q)
	}

	// The public feed names the official actually handling the report, for an
	// ANONYMOUS reader. This is composed by one batch query per page rather than a
	// per-row lookup; if it ever regresses to N+1 the field should come back off the
	// feed rather than the query being left in place.
	if code, feed := callList(http.MethodGet, "/api/problems", ""); code != http.StatusOK {
		t.Fatalf("public feed: want 200, got %d", code)
	} else {
		row := findRow(feed, unvalidatedID)
		if row == nil {
			t.Fatalf("the assigned problem must be in the public feed, got %v", feed)
		}
		if row["assignedOfficialId"] != "off-chair-dhamoirhat" {
			t.Errorf("feed assignedOfficialId = %v, want off-chair-dhamoirhat", row["assignedOfficialId"])
		}
	}

	// 8d) B20: an ABOVE-UNION report is the super admin's to forward, advised by the
	//     union admins. It must never appear on a union admin's assignment queue —
	//     AssignWithinUnion refuses it with wrong_route, and a queue that offers what
	//     the action refuses is the leak B17 fixed at the union boundary (A.3.8).
	code, aboveUnion := call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"title": "integration test upazila drain", "description": "floods the whole ward",
		"location":          map[string]any{"areaId": "union-dhamoirhat", "address": "Ward 3"},
		"pointedOfficialId": "off-upz-chair",
	})
	if code != http.StatusCreated {
		t.Fatalf("report the above-union problem: want 201, got %d (%v)", code, aboveUnion)
	}
	aboveUnionID, _ := aboveUnion["id"].(string)
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+aboveUnionID+"/approve", adminToken, nil); code != http.StatusOK {
		t.Fatalf("approve the above-union problem: got %d", code)
	}

	if code, q := callList(http.MethodGet, "/api/admin/queue", adminToken); code != http.StatusOK {
		t.Fatalf("admin queue: want 200, got %d", code)
	} else if findQueueItem(q, aboveUnionID) != nil {
		t.Fatalf("an above-union report must not be on the union admin's assign queue, got %v", q)
	}
	// A union admin may not assign it even by hitting the route directly.
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+aboveUnionID+"/assign", adminToken, map[string]any{
		"officialId": "off-upz-chair", "priority": "normal",
	}); code != http.StatusConflict {
		t.Fatalf("union admin assigning an above-union report: want 409 wrong_route, got %d", code)
	}

	// It reaches them on the advisory list instead, scoped by ADVICE SCOPE (the
	// upazila) rather than by the admin's own union — which is why the OTHER union's
	// admin sees it here though they saw nothing of it in the queue above.
	advisory := func(token string) map[string]any {
		code, rows := callList(http.MethodGet, "/api/admin/forwarding", token)
		if code != http.StatusOK {
			t.Fatalf("/api/admin/forwarding: want 200, got %d", code)
		}
		return findQueueItem(rows, aboveUnionID)
	}
	if advisory(adminToken) == nil {
		t.Fatal("the above-union report must appear on its own union admin's advisory list")
	}
	if advisory(otherAdminToken) == nil {
		t.Fatal("advice is scoped to the upazila, so another union's admin must see it too")
	}
	// mySuggestion is present-and-null before advising, never absent: both are falsy
	// in JS, so a field that came and went would surface only as a wrongly pre-filled
	// select (the myVote lesson, A.3.2.1 rule 2).
	if row := advisory(adminToken); row["mySuggestion"] != nil {
		t.Errorf("mySuggestion = %v, want null before advising", row["mySuggestion"])
	} else if _, present := row["mySuggestion"]; !present {
		t.Error("mySuggestion must be present-and-null, never absent")
	}

	// Two admins advise the same official — the tally converges.
	for _, tok := range []string{adminToken, otherAdminToken} {
		if code, s := call(http.MethodPost, "/api/admin/problems/"+aboveUnionID+"/suggest-forwarding", tok, map[string]any{
			"officialId": "off-upz-chair", "reason": "drainage is the upazila's budget",
		}); code != http.StatusCreated {
			t.Fatalf("suggest forwarding: want 201, got %d (%v)", code, s)
		}
	}
	advised := advisory(adminToken)
	if advised["topOfficialId"] != "off-upz-chair" || advised["topCount"] != float64(2) {
		t.Fatalf("tally = %v/%v, want off-upz-chair/2", advised["topOfficialId"], advised["topCount"])
	}
	if advised["mySuggestion"] == nil {
		t.Error("mySuggestion must carry this admin's own advice once given")
	}

	// Advice is REVISABLE — the deliberate difference from the ballot it replaced.
	// Moving one of the two makes it a TIE, and a tie has NO top: breaking it
	// arbitrarily would invent a consensus and make the reason requirement turn on a
	// coin flip (domain.TopSuggestion).
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+aboveUnionID+"/suggest-forwarding", otherAdminToken, map[string]any{
		"officialId": "off-upz-vice",
	}); code != http.StatusCreated {
		t.Fatalf("changing advice: want 201, got %d", code)
	}
	if row := advisory(adminToken); row["topOfficialId"] != "" || row["topCount"] != float64(0) {
		t.Fatalf("a tied tally must have no top, got %v/%v", row["topOfficialId"], row["topCount"])
	} else if len(row["suggestions"].([]any)) != 2 {
		t.Fatalf("changing advice must REPLACE the row, not add one: got %v", row["suggestions"])
	}

	// The gates are flat: an admin is refused the super admin's surface entirely.
	if code, _ := callList(http.MethodGet, "/api/super/queue", adminToken); code != http.StatusForbidden {
		t.Fatalf("admin reading the super admin's queue: want 403, got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/super/problems/"+aboveUnionID+"/forward", adminToken, map[string]any{
		"officialId": "off-upz-chair", "priority": "normal",
	}); code != http.StatusForbidden {
		t.Fatalf("admin forwarding: want 403, got %d", code)
	}
	// ...and a resident is refused the advisory one.
	if code, _ := callList(http.MethodGet, "/api/admin/forwarding", residentToken); code != http.StatusForbidden {
		t.Fatalf("resident reading the advisory list: want 403, got %d", code)
	}

	code, body = call(http.MethodPost, "/api/auth/login", "", map[string]any{"phone": "01710000010", "password": "admin123"})
	if code != http.StatusOK {
		t.Fatalf("super admin login: want 200, got %d (%v)", code, body)
	}
	superToken, _ := body["token"].(string)
	if code, q := callList(http.MethodGet, "/api/super/queue", superToken); code != http.StatusOK {
		t.Fatalf("super queue: want 200, got %d", code)
	} else if findQueueItem(q, aboveUnionID) == nil {
		t.Fatalf("the above-union report must await the super admin, got %v", q)
	}

	// The advisers are TIED and the reporter pointed at off-upz-chair, so forwarding
	// anywhere ELSE departs from the reporter and owes the public a reason.
	if code, _ := call(http.MethodPost, "/api/super/problems/"+aboveUnionID+"/forward", superToken, map[string]any{
		"officialId": "off-upz-vice", "priority": "normal",
	}); code != http.StatusBadRequest {
		t.Fatalf("forwarding against the reporter with no reason: want 400 reason_required, got %d", code)
	}
	// Forwarding to the official the REPORTER pointed at needs none.
	code, forwarded := call(http.MethodPost, "/api/super/problems/"+aboveUnionID+"/forward", superToken, map[string]any{
		"officialId": "off-upz-chair", "priority": "normal",
	})
	if code != http.StatusCreated {
		t.Fatalf("forwarding to the reporter's choice: want 201, got %d (%v)", code, forwarded)
	}
	if forwarded["officialId"] != "off-upz-chair" {
		t.Errorf("forwarded officialId = %v, want off-upz-chair", forwarded["officialId"])
	}
	if code, p := call(http.MethodGet, "/api/problems/"+aboveUnionID, "", nil); code != http.StatusOK || p["status"] != "Assigned" {
		t.Fatalf("the forwarded problem should be Assigned, got %d/%v", code, p["status"])
	}
	// It leaves both surfaces — a case exists now.
	if advisory(adminToken) != nil {
		t.Error("a forwarded report must leave the advisory list")
	}
	if code, q := callList(http.MethodGet, "/api/super/queue", superToken); code == http.StatusOK && findQueueItem(q, aboveUnionID) != nil {
		t.Error("a forwarded report must leave the super admin's queue")
	}
	// The decision is on the public trail, as `assigned` — nothing was overridden.
	if code, trail := callList(http.MethodGet, "/api/problems/"+aboveUnionID+"/audit", ""); code != http.StatusOK {
		t.Fatalf("audit: got %d", code)
	} else if !hasAction(trail, "forwarding_suggested") || !hasAction(trail, "assigned") {
		t.Errorf("the advice and the forward must both be on the public trail, got %v", trail)
	}

	// An unassigned problem carries NO assignment keys at all — omitempty drops the
	// zero view, so absent and "not assigned" are the same answer for the client.
	if code, p := call(http.MethodGet, "/api/problems/"+problemID, "", nil); code != http.StatusOK {
		t.Fatalf("get unassigned problem: got %d", code)
	} else if _, ok := p["assignedOfficialId"]; ok {
		t.Errorf("an unassigned problem must not carry assignedOfficialId, got %v", p["assignedOfficialId"])
	}

	// A reporter may now edit their own report while it is Reported and unvalidated.
	// A second resident cannot edit someone else's report (403, not the window state).
	if code, _ := call(http.MethodPatch, "/api/problems/"+problemID, otherResidentToken, map[string]any{
		"title": "integration test hijacked", "description": "not yours",
	}); code != http.StatusForbidden {
		t.Fatalf("edit by a non-reporter: want 403, got %d", code)
	}
	code, edited := call(http.MethodPatch, "/api/problems/"+problemID, residentToken, map[string]any{
		"title": "integration test pothole (edited)", "description": "now even deeper",
		"location":         map[string]any{"areaId": "union-dhamoirhat", "address": "Ward 1, near the school"},
		"proposedSolution": "fill it",
	})
	if code != http.StatusOK || edited["title"] != "integration test pothole (edited)" {
		t.Fatalf("reporter edit: want 200 with the new title, got %d/%v", code, edited["title"])
	}

	// 9) Validate with the verified resident — at V=1 it flips to Validated.
	code, body = call(http.MethodPost, "/api/problems/"+problemID+"/validate", residentToken, map[string]any{"vote": "valid"})
	if code != http.StatusOK {
		t.Fatalf("validate: want 200, got %d (%v)", code, body)
	}
	if body["status"] != "Validated" {
		t.Fatalf("problem should be Validated at V=1, got %v", body["status"])
	}
	// The vote response echoes the vote just cast. Without this a client that
	// trusts the response re-enables the button it just used.
	if body["myVote"] != "valid" {
		t.Fatalf("the validate response must echo myVote=valid, got %v", body["myVote"])
	}

	// 9a) myVote is the CALLER's own vote and nobody else's. It must survive a
	//     re-read for the voter, and be null for everyone else — a myVote that
	//     leaked another account's vote would be a privacy bug, not a display one.
	if code, mine := call(http.MethodGet, "/api/problems/"+problemID, residentToken, nil); code != http.StatusOK || mine["myVote"] != "valid" {
		t.Fatalf("the voter's own re-read must carry myVote=valid, got %d/%v", code, mine["myVote"])
	}
	assertNullMyVote := func(who string, body map[string]any) {
		t.Helper()
		v, present := body["myVote"]
		// Present-and-null, never absent: an absent key and a null are both falsy in
		// JavaScript, so a regression to "sometimes missing" would be invisible in
		// the UI and surface only as a wrongly-enabled button.
		if !present {
			t.Fatalf("%s: myVote must be present as JSON null, not omitted (%v)", who, body)
		}
		if v != nil {
			t.Fatalf("%s: myVote must be null, got %v", who, v)
		}
	}
	if code, other := call(http.MethodGet, "/api/problems/"+problemID, otherResidentToken, nil); code != http.StatusOK {
		t.Fatalf("second resident read: want 200, got %d", code)
	} else {
		assertNullMyVote("a second resident", other)
	}
	if code, anon := call(http.MethodGet, "/api/problems/"+problemID, "", nil); code != http.StatusOK {
		t.Fatalf("anonymous read: want 200, got %d", code)
	} else {
		assertNullMyVote("an anonymous reader", anon)
	}
	// The feed carries it too — that is what lets a row show a vote already cast.
	if code, feedRows := callList(http.MethodGet, "/api/problems", residentToken); code != http.StatusOK {
		t.Fatalf("feed as the voter: want 200, got %d", code)
	} else {
		row := findRow(feedRows, problemID)
		if row == nil || row["myVote"] != "valid" {
			t.Fatalf("the voter's feed row must carry myVote=valid, got %v", row)
		}
	}
	if code, feedRows := callList(http.MethodGet, "/api/problems", ""); code != http.StatusOK {
		t.Fatalf("anonymous feed: want 200, got %d", code)
	} else if row := findRow(feedRows, problemID); row != nil {
		assertNullMyVote("an anonymous feed row", row)
	}

	// Once endorsed (a valid vote landed), the reporter can no longer rewrite the
	// text residents validated — the edit window has closed.
	if code, _ := call(http.MethodPatch, "/api/problems/"+problemID, residentToken, map[string]any{
		"title": "integration test too late", "description": "already validated",
	}); code != http.StatusConflict {
		t.Fatalf("editing a validated report: want 409, got %d", code)
	}

	// 10) Read it back — status persisted and the audit trail recorded every step,
	//     including the admin's approval.
	code, body = call(http.MethodGet, "/api/problems/"+problemID, "", nil)
	if code != http.StatusOK || body["status"] != "Validated" {
		t.Fatalf("get problem: want 200/Validated, got %d/%v", code, body["status"])
	}
	audit, _ := body["audit"].([]any)
	if !hasAction(audit, "reported") || !hasAction(audit, "approved") || !hasAction(audit, "validated") {
		t.Fatalf("audit should contain reported + approved + validated, got %v", audit)
	}
	if !hasActor(audit, "approved", adminID) {
		t.Fatalf("the approval must be audited against the deciding admin, got %v", audit)
	}

	// 11) Rejection during screening: on a fixed ground, and public once decided.
	code, body = call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"title": "integration test spam", "description": "buy cheap watches",
		"location": map[string]any{"areaId": "union-dhamoirhat"}, "pointedOfficialId": "off-chair-dhamoirhat",
	})
	if code != http.StatusCreated {
		t.Fatalf("report spam: want 201, got %d (%v)", code, body)
	}
	spamID, _ := body["id"].(string)

	// Spam is hidden while pending — it never reaches the public feed at all,
	// because the admin catches it before publication.
	if code, spamFeed := callList(http.MethodGet, "/api/problems", ""); code != http.StatusOK || containsID(spamFeed, spamID) {
		t.Fatalf("a pending spam report must NOT be public before screening (got %d)", code)
	}

	// Merit is not a ground: only the four enum values are accepted.
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+spamID+"/reject", adminToken, map[string]any{
		"reason": "i do not think this is important",
	}); code != http.StatusBadRequest {
		t.Fatalf("reject on a made-up ground: want 400, got %d", code)
	}
	code, body = call(http.MethodPost, "/api/admin/problems/"+spamID+"/reject", adminToken, map[string]any{
		"reason": "spam", "note": "advertising",
	})
	if code != http.StatusOK || body["status"] != "Rejected" {
		t.Fatalf("reject: want 200/Rejected, got %d/%v", code, body["status"])
	}
	if body["rejectionReason"] != "spam" {
		t.Fatalf("a rejection must carry its ground, got %v", body["rejectionReason"])
	}
	// A rejected problem becomes publicly readable, with its reason — that is what
	// gives a wrongly-buried report a public witness.
	code, body = call(http.MethodGet, "/api/problems/"+spamID, "", nil)
	if code != http.StatusOK || body["status"] != "Rejected" || body["rejectionReason"] != "spam" {
		t.Fatalf("anonymous read of a rejected problem: want 200/Rejected/spam, got %d/%v/%v", code, body["status"], body["rejectionReason"])
	}
	if code, rejected := callList(http.MethodGet, "/api/problems?status=Rejected", ""); code != http.StatusOK || !containsID(rejected, spamID) {
		t.Fatalf("rejected problems must be listable publicly (got %d)", code)
	}
	// The rejection is attributed: a takedown leaves a named trace, never a silent
	// disappearance.
	audit, _ = body["audit"].([]any)
	if !hasActor(audit, "rejected", adminID) {
		t.Fatalf("a rejection must be audited against the deciding admin, got %v", audit)
	}
	// Rejecting twice is refused — the problem is out of the rejectable window now.
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+spamID+"/reject", adminToken, map[string]any{
		"reason": "spam",
	}); code != http.StatusConflict {
		t.Fatalf("re-rejecting: want 409, got %d", code)
	}
	// 12) The rejectable window closes at assignment: from there a case exists and an
	//     official is working in public, so removing the problem would erase their
	//     work. problemID is Validated (step 9) and still rejectable; drive it past
	//     assignment and the same call must be refused.
	code, body = call(http.MethodPost, "/api/admin/problems/"+problemID+"/assign", adminToken, map[string]any{
		"officialId": "off-chair-dhamoirhat", "priority": "normal",
	})
	if code != http.StatusCreated {
		t.Fatalf("assign: want 201, got %d (%v)", code, body)
	}
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+problemID+"/reject", adminToken, map[string]any{
		"reason": "spam",
	}); code != http.StatusConflict {
		t.Fatalf("rejecting an assigned problem: want 409, got %d", code)
	}

	// 8) Public scorecard for a seeded official responds with the read model.
	code, body = call(http.MethodGet, "/api/officials/off-chair-dhamoirhat/scorecard", "", nil)
	if code != http.StatusOK || body["officialId"] != "off-chair-dhamoirhat" {
		t.Fatalf("scorecard: want 200 for off-1, got %d (%v)", code, body)
	}

	// 13) B13: the seat's geography, anonymously. Register's union select reads
	//     this before the resident has an account, so a gate here would close the
	//     first screen of the pilot chain.
	code, areas := callList(http.MethodGet, "/api/areas", "")
	if code != http.StatusOK {
		t.Fatalf("anonymous areas: want 200, got %d", code)
	}
	if len(areas) == 0 {
		t.Fatal("areas must return the seeded geography, got none")
	}

	var unions int
	var seat map[string]any
	for _, a := range areas {
		m, ok := a.(map[string]any)
		if !ok {
			t.Fatalf("area row is not an object: %v", a)
		}
		switch m["level"] {
		case "union":
			unions++
		case "seat":
			seat = m
		}
	}
	// Register's select is a list of unions; without one the form cannot be filled.
	if unions < 1 {
		t.Fatalf("areas must include at least one union, got %d", unions)
	}
	if seat == nil {
		t.Fatal("areas must include the seat (the root)")
	}

	// The seat is the root, so its parentId must be JSON null. Checked in three
	// parts because all three failures look identical to a client doing
	// `if (parentId)`: an absent key, a null, and "" are equally falsy, and only
	// one of them is the contract (models.js says string|null).
	parent, present := seat["parentId"]
	if !present {
		t.Fatal("the seat's parentId key must be present, not omitted")
	}
	if parent != nil {
		t.Fatalf("the seat's parentId must be JSON null, got %#v", parent)
	}

	// 14) The seat's PUBLIC decision record. Anonymous: the people this exists to
	//     inform are residents, so a gate would defeat it.
	code, activity := callList(http.MethodGet, "/api/seat/activity", "")
	if code != http.StatusOK {
		t.Fatalf("anonymous activity: want 200, got %d", code)
	}

	var sawApproved, sawAssigned bool
	for _, row := range activity {
		m, ok := row.(map[string]any)
		if !ok {
			t.Fatalf("activity row is not an object: %v", row)
		}

		// THE privacy split, asserted against real data rather than only in the unit
		// test: publishing `verified` would build a public index of who is a verified
		// resident of which union (A.3.2 rule 1), and `claim_rejected` would publish
		// an accusation about an unproven identity claim. This flow verified a
		// resident above, so the entry exists in the log — it must not be here.
		switch m["action"] {
		case "verified", "claim_approved", "claim_rejected":
			t.Fatalf("the public feed leaked a person-action: %v", m["action"])
		case "approved":
			sawApproved = true
		case "assigned":
			sawAssigned = true
		}

		// A row has to say who acted and about what, or the seat-wide aggregate is a
		// page of opaque ids and reveals no pattern at all.
		if m["actorId"] == nil || m["actorId"] == "" {
			t.Fatalf("activity row has no actorId: %v", m)
		}
		if m["targetType"] == "problem" && m["problemTitle"] == nil {
			t.Fatalf("a public problem's row must carry its title: %v", m)
		}
	}

	// This flow approved and assigned a problem, so both must be on the record.
	if !sawApproved || !sawAssigned {
		t.Fatalf("activity must carry this flow's approve and assign (approved=%v assigned=%v)", sawApproved, sawAssigned)
	}

	// 15) The official's plan is a week-by-week checklist, and a week can be checked
	//     off in public. Driven end to end because the queue's own DTO/UI mismatch
	//     (Part D) proves a shape can rot untested until something reaches it.
	//     off-mayor is the only seeded official account; its admin is the pourashava
	//     admin, so a pourashava problem is what it can be assigned.
	code, body = call(http.MethodPost, "/api/auth/login", "", map[string]any{"phone": "01910000009", "password": "admin123"})
	if code != http.StatusOK {
		t.Fatalf("pourashava admin login: want 200, got %d (%v)", code, body)
	}
	pouraAdminToken, _ := body["token"].(string)

	code, body = call(http.MethodPost, "/api/auth/login", "", map[string]any{"phone": "01710000001", "password": "official123"})
	if code != http.StatusOK {
		t.Fatalf("official login: want 200, got %d (%v)", code, body)
	}
	officialToken, _ := body["token"].(string)

	code, body = call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"title": "integration test broken footpath", "description": "cracked and flooding",
		"location": map[string]any{"areaId": "pourashava-dhamoirhat", "address": "Ward 3"}, "pointedOfficialId": "off-mayor",
	})
	if code != http.StatusCreated {
		t.Fatalf("report the plan-walk problem: want 201, got %d (%v)", code, body)
	}
	taskProblemID, _ := body["id"].(string)

	if code, _ := call(http.MethodPost, "/api/admin/problems/"+taskProblemID+"/approve", pouraAdminToken, nil); code != http.StatusOK {
		t.Fatalf("approve the plan-walk problem as pourashava admin: got %d", code)
	}
	if code, a := call(http.MethodPost, "/api/admin/problems/"+taskProblemID+"/assign", pouraAdminToken, map[string]any{
		"officialId": "off-mayor", "priority": "normal",
	}); code != http.StatusCreated {
		t.Fatalf("assign to off-mayor: want 201, got %d (%v)", code, a)
	}

	// GET /official/cases materializes the case lazily; find the one for our problem.
	code, cases := callList(http.MethodGet, "/api/official/cases", officialToken)
	if code != http.StatusOK {
		t.Fatalf("official cases: want 200, got %d", code)
	}
	var caseID string
	for _, e := range cases {
		if m, ok := e.(map[string]any); ok && m["problemId"] == taskProblemID {
			caseID, _ = m["id"].(string)
		}
	}
	if caseID == "" {
		t.Fatalf("the assigned problem must materialize a case for off-mayor, got %v", cases)
	}

	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/acknowledge", officialToken, map[string]any{"decision": "accept"}); code != http.StatusOK {
		t.Fatalf("acknowledge: got %d", code)
	}

	// A plan with no weeks is refused — it is a checklist, and an empty one is not a plan.
	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/plan", officialToken, map[string]any{
		"strategy": "fix it", "tasks": []any{}, "suggestionResponse": "adopting",
	}); code != http.StatusBadRequest {
		t.Fatalf("a plan with no weekly tasks: want 400, got %d", code)
	}

	code, planned := call(http.MethodPost, "/api/official/cases/"+caseID+"/plan", officialToken, map[string]any{
		"strategy":           "phased repair",
		"tasks":              []any{"clear the drain inlet", "lay new culvert pipe", "backfill and repave"},
		"suggestionResponse": "adopting the community's plan",
	})
	if code != http.StatusOK {
		t.Fatalf("submit plan with weekly tasks: want 200, got %d (%v)", code, planned)
	}
	plan, _ := planned["plan"].(map[string]any)
	tasks, _ := plan["tasks"].([]any)
	if len(tasks) != 3 {
		t.Fatalf("plan must carry three weekly tasks, got %v", tasks)
	}
	// TimelineWeeks is derived from the checklist — the two can never disagree.
	if plan["timelineWeeks"] != float64(3) {
		t.Fatalf("timelineWeeks must equal the task count (3), got %v", plan["timelineWeeks"])
	}
	firstTask, _ := tasks[0].(map[string]any)
	firstTaskID, _ := firstTask["id"].(string)
	if firstTaskID == "" || firstTask["weekNumber"] != float64(1) || firstTask["completed"] != false {
		t.Fatalf("week 1 task malformed: %v", firstTask)
	}

	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/tasks/"+firstTaskID+"/complete", officialToken, nil); code != http.StatusOK {
		t.Fatalf("complete week 1: got %d", code)
	}
	// An unknown task id is a 404, never a silent success.
	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/tasks/task-nope/complete", officialToken, nil); code != http.StatusNotFound {
		t.Fatalf("completing an unknown task: want 404, got %d", code)
	}

	// The public progress route reflects the check — completion is public, and it is
	// the SAME evidence the /me view reads (one public account of a case, never two).
	code, progress := call(http.MethodGet, "/api/problems/"+taskProblemID+"/progress", "", nil)
	if code != http.StatusOK {
		t.Fatalf("public progress: want 200, got %d", code)
	}
	pplan, _ := progress["plan"].(map[string]any)
	ptasks, _ := pplan["tasks"].([]any)
	if len(ptasks) != 3 {
		t.Fatalf("progress must expose the three weekly tasks, got %v", ptasks)
	}
	if pt0, _ := ptasks[0].(map[string]any); pt0["completed"] != true {
		t.Fatalf("week 1 must read completed in public progress, got %v", pt0)
	}
	if pt1, _ := ptasks[1].(map[string]any); pt1["completed"] != false {
		t.Fatalf("week 2 must still read not-completed, got %v", pt1)
	}

	// --- the obstacle → resume → replan restart path (B18/F21) ---

	// Raise an obstacle: the case goes Blocked and the cause becomes public.
	if code, ob := call(http.MethodPost, "/api/official/cases/"+caseID+"/report-obstacle", officialToken, map[string]any{
		"category": "budget", "whatBlocks": "no allocation this quarter",
		"whoUnblocks": "upazila engineer", "proofTried": "wrote to the UNO twice",
	}); code != http.StatusOK {
		t.Fatalf("report obstacle: want 200, got %d (%v)", code, ob)
	}

	// The obstacle is on the public progress record — the "cause of not solving".
	code, blockedProgress := call(http.MethodGet, "/api/problems/"+taskProblemID+"/progress", "", nil)
	if code != http.StatusOK {
		t.Fatalf("progress after obstacle: want 200, got %d", code)
	}
	obstacles, _ := blockedProgress["obstacles"].([]any)
	if len(obstacles) != 1 {
		t.Fatalf("progress must expose the obstacle history, got %v", blockedProgress["obstacles"])
	}
	if ob0, _ := obstacles[0].(map[string]any); ob0["whatBlocks"] != "no allocation this quarter" {
		t.Fatalf("obstacle cause not on the public record: %v", ob0)
	}

	// Resume from Blocked with a progress note (Blocked → InProgress), the step that
	// precedes a replan of a blocked case.
	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/updates", officialToken, map[string]any{
		"kind": "progress", "text": "budget cleared, resuming",
	}); code != http.StatusOK {
		t.Fatalf("resume from blocked: got %d", code)
	}

	// A replan with no reason is refused — the reason is the public "what changed".
	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/replan", officialToken, map[string]any{
		"strategy": "new approach", "tasks": []any{"re-survey"}, "suggestionResponse": "adopting", "reason": "  ",
	}); code != http.StatusBadRequest {
		t.Fatalf("replan with no reason: want 400, got %d", code)
	}

	// Replan: a fresh plan restarts the case (still InProgress).
	code, revised := call(http.MethodPost, "/api/official/cases/"+caseID+"/replan", officialToken, map[string]any{
		"strategy": "switch to a box culvert", "tasks": []any{"re-survey", "install box culvert"},
		"suggestionResponse": "adopting the revised community idea", "reason": "the pipe approach kept flooding",
	})
	if code != http.StatusOK {
		t.Fatalf("replan: want 200, got %d (%v)", code, revised)
	}
	if revised["status"] != "InProgress" {
		t.Fatalf("replan status = %v, want InProgress", revised["status"])
	}

	// The public record shows the new plan, the restart on the timeline, and the
	// obstacle still (persistent cause), all at once.
	code, afterReplan := call(http.MethodGet, "/api/problems/"+taskProblemID+"/progress", "", nil)
	if code != http.StatusOK {
		t.Fatalf("progress after replan: want 200, got %d", code)
	}
	newPlan, _ := afterReplan["plan"].(map[string]any)
	if newPlan["strategy"] != "switch to a box culvert" {
		t.Fatalf("progress must show the revised plan, got %v", newPlan["strategy"])
	}
	updates, _ := afterReplan["updates"].([]any)
	var sawRevised bool
	for _, u := range updates {
		if m, ok := u.(map[string]any); ok {
			if txt, _ := m["text"].(string); strings.Contains(txt, "Plan revised") {
				sawRevised = true
			}
		}
	}
	if !sawRevised {
		t.Fatalf("the restart must appear on the public timeline, got %v", updates)
	}
	if obs, _ := afterReplan["obstacles"].([]any); len(obs) != 1 {
		t.Fatalf("the obstacle must persist on the record after the restart, got %v", afterReplan["obstacles"])
	}

	// --- B19: the observation ladder, against a real database ---
	//
	// The rows are opened by cmd/worker, not by any endpoint, so this drives the
	// scan directly. What only Postgres can prove is the OpenObservation insert:
	// its ON CONFLICT names a PARTIAL unique index's predicate, which no fake can
	// check and which silently opens duplicate rungs every minute if it is wrong.

	// Backdate the case past its deadline with no assignee activity since, so the
	// silence clock reads one full rung.
	if _, err := pool.Exec(ctx, `UPDATE cases SET deadline = now() - interval '1 day' WHERE id = $1`, caseID); err != nil {
		t.Fatalf("backdate the deadline: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM case_observations WHERE case_id = $1`, caseID); err != nil {
		t.Fatalf("clear observations: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM case_observations WHERE case_id = $1`, caseID)
	})

	caseRepo := resolutionpg.NewCaseRepository(pool)
	auditRepo := auditpg.NewAuditRepository(pool)
	ladder := resolutionOfficialAdapter{
		officials: identitypg.NewOfficialRepository(pool),
		areas:     areapg.NewAreaRepository(pool),
	}
	scan := resolutionapp.NewEscalateOverdue(caseRepo, auditRepo, ladder, cfg.ResponseDeadline, cfg.BlockerReviewWindow)

	// This case has activity (it was acknowledged, planned and replanned above), so
	// the SILENCE clock must read zero even though the deadline has passed — the
	// two-clock rule (A.3.7). An official who is working accumulates no observers.
	res, err := scan.Run(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("escalation scan: %v", err)
	}
	var openRungs int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM case_observations WHERE case_id = $1 AND resolved_at IS NULL`, caseID).
		Scan(&openRungs); err != nil {
		t.Fatalf("count observations: %v", err)
	}
	if openRungs != 0 {
		t.Fatalf("a case the official is actively working opened %d observations; silence is not lateness", openRungs)
	}

	// Now erase every trace of assignee activity and re-scan: the same case, now
	// genuinely silent, must put a rung in front of the monitor.
	if _, err := pool.Exec(ctx, `UPDATE cases SET acknowledged_at = NULL WHERE id = $1`, caseID); err != nil {
		t.Fatalf("clear acknowledgement: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE plans SET created_at = now() - interval '90 days' WHERE case_id = $1`, caseID); err != nil {
		t.Fatalf("age the plan: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE progress_updates SET created_at = now() - interval '90 days' WHERE case_id = $1`, caseID); err != nil {
		t.Fatalf("age the updates: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE obstacles SET created_at = now() - interval '90 days' WHERE case_id = $1`, caseID); err != nil {
		t.Fatalf("age the obstacles: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE evidence SET created_at = now() - interval '90 days' WHERE case_id = $1`, caseID); err != nil {
		t.Fatalf("age the evidence: %v", err)
	}
	if res, err = scan.Run(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("second escalation scan: %v", err)
	}
	if res.ObservationsOpened == 0 {
		t.Fatal("a silent case past its deadline must open a rung on the monitor ladder")
	}
	var monitorID string
	if err := pool.QueryRow(ctx,
		`SELECT observer_official_id FROM case_observations WHERE case_id = $1 AND level = 1`, caseID).
		Scan(&monitorID); err != nil {
		t.Fatalf("rung 1 must exist: %v", err)
	}
	if monitorID == "" {
		t.Fatal("rung 1 must name an official, not an empty id")
	}

	// Idempotence against the REAL partial unique index: a second scan at the same
	// instant must insert nothing. This is the assertion the fakes cannot make.
	before := openRungs
	if _, err := scan.Run(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("third escalation scan: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM case_observations WHERE case_id = $1 AND level = 1`, caseID).
		Scan(&openRungs); err != nil {
		t.Fatalf("recount observations: %v", err)
	}
	if openRungs != 1 {
		t.Fatalf("rung 1 exists %d times after two scans (was %d), want exactly 1", openRungs, before)
	}

	// The monitor's own list is caller-scoped and the silent official is not their
	// own monitor, so this case must not appear on the assignee's observation page.
	code, mine = callList(http.MethodGet, "/api/official/observations", officialToken)
	if code != http.StatusOK {
		t.Fatalf("GET observations: want 200, got %d", code)
	}
	for _, row := range mine {
		if m, ok := row.(map[string]any); ok && m["caseId"] == caseID {
			t.Fatal("an official must never appear on their own observation list")
		}
	}
	if code, _ := callList(http.MethodGet, "/api/official/observations", ""); code != http.StatusUnauthorized {
		t.Fatalf("anonymous observations: want 401, got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/official/observations/obs-nope/note", officialToken,
		map[string]any{"text": "x"}); code != http.StatusNotFound {
		t.Fatalf("note on an unknown observation: want 404, got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/official/observations/obs-nope/note", residentToken,
		map[string]any{"text": "x"}); code != http.StatusForbidden {
		t.Fatalf("note as a resident: want 403, got %d", code)
	}
}

func hasAction(audit []any, action string) bool {
	for _, e := range audit {
		if m, ok := e.(map[string]any); ok && m["action"] == action {
			return true
		}
	}
	return false
}

// containsID reports whether a JSON array of problem DTOs contains the given id.
// findQueueItem returns the assignment-queue row for a problem, or nil. The queue
// keys its rows on "problemId" rather than "id" — which is exactly the mismatch
// that made the UI read `item.id` as undefined — so containsID cannot be reused
// here, and a test that tried would silently find nothing and pass.
func findQueueItem(items []any, problemID string) map[string]any {
	for _, e := range items {
		if m, ok := e.(map[string]any); ok && m["problemId"] == problemID {
			return m
		}
	}
	return nil
}

// findRow returns a problem row from a feed by its id, or nil. The twin of
// findQueueItem, which keys on problemId because a queue row is an assignment
// view of a problem rather than the problem itself.
func findRow(items []any, id string) map[string]any {
	for _, e := range items {
		if m, ok := e.(map[string]any); ok && m["id"] == id {
			return m
		}
	}
	return nil
}

func containsID(items []any, id string) bool {
	for _, e := range items {
		if m, ok := e.(map[string]any); ok && m["id"] == id {
			return true
		}
	}
	return false
}

// hasActor reports whether the audit trail attributes the given action to the
// given actor.
func hasActor(audit []any, action, actor string) bool {
	for _, e := range audit {
		if m, ok := e.(map[string]any); ok && m["action"] == action && m["actor"] == actor {
			return true
		}
	}
	return false
}
