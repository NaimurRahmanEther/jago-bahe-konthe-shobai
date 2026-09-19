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
	"testing"
	"time"

	"jago-bahe-backend/config"
	"jago-bahe-backend/pkg/postgres"
)

// TestIntegration_NotificationLoop drives all eight loop-closing notifications
// (B21) through the whole HTTP stack against real Postgres: the router, the
// RequireAnyRole gate, the DTO mapping and the real repositories.
//
// It is a separate test from TestIntegration_ProblemSlice because it walks the
// case lifecycle end to end — approve, assign, acknowledge, plan, evidence, done,
// confirm, obstacle, adjudicate — and folding that into the screening slice would
// make one unreadable function out of two readable ones.
//
// Like its sibling it skips unless TEST_DATABASE_URL points at a migrated, seeded
// database. Never point that at the `jago` dev DB: this test writes and deletes.
func TestIntegration_NotificationLoop(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("set TEST_DATABASE_URL (a migrated, seeded DB) to run the integration test")
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Registered first so it runs LAST (t.Cleanup is LIFO) — the row cleanup below
	// needs the pool still open. The sibling test documents why a defer here leaks.
	t.Cleanup(func() { pool.Close() })

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
		req.RemoteAddr = "127.0.0.1:5556"
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}
	callList := func(method, path, token string) (int, []any) {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = "127.0.0.1:5556"
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var out []any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}

	t.Cleanup(func() {
		// notifications cascade from problems, so deleting the problems is enough for
		// them — but the audit entries have NO FK (A.3.3 constraint 3), so they are
		// removed by hand, exactly as delete_problem.go must.
		_, _ = pool.Exec(ctx, `DELETE FROM audit_entries WHERE target_id IN (
			SELECT id FROM problems WHERE title LIKE 'notif test %')`)
		_, _ = pool.Exec(ctx, `DELETE FROM problems WHERE title LIKE 'notif test %'`)
	})

	login := func(phone, password string) string {
		t.Helper()
		code, body := call(http.MethodPost, "/api/auth/login", "", map[string]any{"phone": phone, "password": password})
		if code != http.StatusOK {
			t.Fatalf("login %s: want 200, got %d (%v)", phone, code, body)
		}
		token, _ := body["token"].(string)
		if token == "" {
			t.Fatalf("login %s returned no token", phone)
		}
		return token
	}

	// notifications returns the caller's own list, and fails the test on any status
	// but 200 — every assertion below is about WHICH rows are there, so a silent
	// error status would otherwise read as "no notifications".
	notifications := func(token string) []any {
		t.Helper()
		code, list := callList(http.MethodGet, "/api/me/notifications", token)
		if code != http.StatusOK {
			t.Fatalf("list notifications: want 200, got %d", code)
		}
		return list
	}
	// findNotif locates one notification by (type, problemId). Both halves matter:
	// several problems are in flight at once, and one type can legitimately appear
	// for more than one of them.
	findNotif := func(list []any, typ, problemID string) map[string]any {
		for _, r := range list {
			row, ok := r.(map[string]any)
			if ok && row["type"] == typ && row["problemId"] == problemID {
				return row
			}
		}
		return nil
	}
	unreadCount := func(token string) int {
		t.Helper()
		code, body := call(http.MethodGet, "/api/me/notifications/unread-count", token, nil)
		if code != http.StatusOK {
			t.Fatalf("unread-count: want 200, got %d (%v)", code, body)
		}
		n, ok := body["count"].(float64)
		if !ok {
			t.Fatalf("unread-count body = %v, want {count: number}", body)
		}
		return int(n)
	}

	residentToken := login("01810000001", "resident123") // seeded, verified, union-dhamoirhat
	adminToken := login("01910000009", "admin123")       // the pourashava's admin
	officialToken := login("01710000001", "official123") // the mayor, off-mayor
	superToken := login("01710000010", "admin123")       // approves above-union claims

	// The MONITOR. The mayor's monitor is the upazila chairman (off-upz-chair), and
	// the migrations seed that OFFICE with no account against it — the directory
	// records elections, and registration never creates an office. So the account is
	// built here through the real B11 flow (register a claim, super admin approves an
	// above-union one), which is also the only way to get a JWT carrying
	// officialId=off-upz-chair — and that claim, not the account id, is what the
	// notification is addressed to.
	monitorPhone := fmt.Sprintf("0193%07d", rand.Intn(1e7))
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM audit_entries WHERE actor IN (SELECT id FROM accounts WHERE phone = $1)`, monitorPhone)
		_, _ = pool.Exec(ctx, `DELETE FROM official_claims WHERE account_id IN (SELECT id FROM accounts WHERE phone = $1)`, monitorPhone)
		_, _ = pool.Exec(ctx, `DELETE FROM accounts WHERE phone = $1`, monitorPhone)
	})
	code, monitorReg := call(http.MethodPost, "/api/auth/register/official", "", map[string]any{
		"name": "Notif Monitor", "phone": monitorPhone, "password": "monitor123",
		"nid": "1985010199999", "officialId": "off-upz-chair",
	})
	if code != http.StatusCreated {
		t.Fatalf("register the monitor official: want 201, got %d (%v)", code, monitorReg)
	}
	code, claims := callList(http.MethodGet, "/api/claims/pending", superToken)
	if code != http.StatusOK {
		t.Fatalf("pending claims: want 200, got %d", code)
	}
	var claimID string
	for _, c := range claims {
		if row, ok := c.(map[string]any); ok && row["officialId"] == "off-upz-chair" {
			claimID, _ = row["id"].(string)
		}
	}
	if claimID == "" {
		t.Fatalf("the upazila-chairman claim must be pending for the super admin; got %v", claims)
	}
	if code, b := call(http.MethodPost, "/api/claims/"+claimID+"/approve", superToken, nil); code != http.StatusOK {
		t.Fatalf("approve the monitor's claim: want 200, got %d (%v)", code, b)
	}
	monitorToken := login(monitorPhone, "monitor123")

	// A second RESIDENT — the same id space as the reporter, which is what makes the
	// privacy assertion below strong. An official would be a weaker witness: their
	// notifications are keyed on a directory office id, so a leak between two
	// accounts could hide behind the id spaces simply not matching.
	otherPhone := fmt.Sprintf("0194%07d", rand.Intn(1e7))
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM audit_entries WHERE actor IN (SELECT id FROM accounts WHERE phone = $1)`, otherPhone)
		_, _ = pool.Exec(ctx, `DELETE FROM accounts WHERE phone = $1`, otherPhone)
	})
	if code, b := call(http.MethodPost, "/api/auth/register", "", map[string]any{
		"name": "Notif Bystander", "phone": otherPhone, "password": "test1234",
		"nid": "1991010112345", "unionId": "union-dhamoirhat",
	}); code != http.StatusCreated {
		t.Fatalf("register the second resident: want 201, got %d (%v)", code, b)
	}
	otherResidentToken := login(otherPhone, "test1234")

	// The reporter's baseline. Notifications accumulate across runs for a shared
	// seeded account, so every assertion below is relative to this, never absolute.
	baseUnread := unreadCount(residentToken)

	// --- 1. report → approve: the reporter learns their report went public -------

	code, body := call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"title": "notif test broken drain", "description": "the drain by the school is blocked",
		"location":          map[string]any{"areaId": "pourashava-dhamoirhat", "address": "Ward 3"},
		"pointedOfficialId": "off-mayor",
	})
	if code != http.StatusCreated {
		t.Fatalf("report: want 201, got %d (%v)", code, body)
	}
	problemID, _ := body["id"].(string)

	// Nothing yet: filing is not an event anyone needs to be told about — the
	// reporter just did it, and admins have a screening queue with a count.
	if n := findNotif(notifications(residentToken), "problem_approved", problemID); n != nil {
		t.Fatal("a freshly reported problem must generate no notification")
	}

	if code, b := call(http.MethodPost, "/api/admin/problems/"+problemID+"/approve", adminToken, nil); code != http.StatusOK {
		t.Fatalf("approve: want 200, got %d (%v)", code, b)
	}

	approved := findNotif(notifications(residentToken), "problem_approved", problemID)
	if approved == nil {
		t.Fatalf("the reporter must be told their report was approved; got %v", notifications(residentToken))
	}
	// readAt is present-and-null while unread, NEVER absent: both are falsy in JS,
	// so a regression to "sometimes missing" would show up only as a wrongly-styled
	// row (A.5.2 at the field level).
	if v, present := approved["readAt"]; !present || v != nil {
		t.Errorf("readAt = %v (present=%v), want present and null on an unread row", v, present)
	}
	if approved["problemTitle"] != "notif test broken drain" {
		t.Errorf("problemTitle = %v, want the title snapshotted at write time", approved["problemTitle"])
	}
	if got := unreadCount(residentToken); got != baseUnread+1 {
		t.Errorf("unread = %d, want %d", got, baseUnread+1)
	}

	// A second resident is told nothing about someone else's report. This is the
	// privacy assertion the whole design turns on: the list is self-derived, and
	// there is no parameter that could widen it.
	if n := findNotif(notifications(otherResidentToken), "problem_approved", problemID); n != nil {
		t.Fatal("another account must never see a notification about this report")
	}

	// --- 2. assign: two parties, two id spaces ----------------------------------

	code, body = call(http.MethodPost, "/api/admin/problems/"+problemID+"/assign", adminToken, map[string]any{
		"officialId": "off-mayor", "priority": "normal",
	})
	if code != http.StatusCreated {
		t.Fatalf("assign: want 201, got %d (%v)", code, body)
	}

	assignedToReporter := findNotif(notifications(residentToken), "problem_assigned", problemID)
	if assignedToReporter == nil {
		t.Fatalf("the reporter must be told who their report went to; got %v", notifications(residentToken))
	}
	if assignedToReporter["detail"] == "" || assignedToReporter["detail"] == nil {
		t.Error("problem_assigned detail must carry the official's name — 'assigned' alone says nothing useful")
	}

	// The official is addressed by their DIRECTORY OFFICE id, which the JWT carries
	// alongside the account id. A resident's notification must not reach them, and
	// theirs must not reach the resident.
	officialList := notifications(officialToken)
	if findNotif(officialList, "case_assigned", problemID) == nil {
		t.Fatalf("the assigned official must be told; got %v", officialList)
	}
	if findNotif(officialList, "problem_assigned", problemID) != nil {
		t.Fatal("the reporter's notification leaked to the official — the two id spaces have crossed")
	}
	if findNotif(notifications(residentToken), "case_assigned", problemID) != nil {
		t.Fatal("the official's notification leaked to the reporter")
	}

	// --- 3. the lifecycle → done: the reporter is asked to confirm ---------------

	// GET /api/official/cases materializes the case lazily on first read.
	code, cases := callList(http.MethodGet, "/api/official/cases", officialToken)
	if code != http.StatusOK {
		t.Fatalf("official cases: want 200, got %d", code)
	}
	var caseID string
	for _, c := range cases {
		if row, ok := c.(map[string]any); ok && row["problemId"] == problemID {
			caseID, _ = row["id"].(string)
		}
	}
	if caseID == "" {
		t.Fatalf("the assigned case must appear on the official's dashboard; got %v", cases)
	}

	if code, b := call(http.MethodPost, "/api/official/cases/"+caseID+"/acknowledge", officialToken,
		map[string]any{"decision": "accept"}); code != http.StatusOK {
		t.Fatalf("acknowledge: want 200, got %d (%v)", code, b)
	}
	if code, b := call(http.MethodPost, "/api/official/cases/"+caseID+"/plan", officialToken, map[string]any{
		"strategy": "clear and re-line the drain", "tasks": []string{"survey", "clear", "re-line"},
		"suggestionResponse": "adopting the community's proposal",
	}); code != http.StatusOK {
		t.Fatalf("plan: want 200, got %d (%v)", code, b)
	}
	if code, b := call(http.MethodPost, "/api/official/cases/"+caseID+"/evidence", officialToken, map[string]any{
		"beforeImageUrl": "data:image/png;base64,before", "afterImageUrl": "data:image/png;base64,after",
	}); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("evidence: want 200/201, got %d (%v)", code, b)
	}
	if code, b := call(http.MethodPost, "/api/official/cases/"+caseID+"/updates", officialToken, map[string]any{
		"kind": "progress", "text": "drain cleared this week",
	}); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("update: want 200/201, got %d (%v)", code, b)
	}
	if code, b := call(http.MethodPost, "/api/official/cases/"+caseID+"/done", officialToken, nil); code != http.StatusOK {
		t.Fatalf("mark done: want 200, got %d (%v)", code, b)
	}

	// The highest-value notification in the platform: the reporter is the only party
	// who can move this to Resolved, and nothing else asks them to.
	if findNotif(notifications(residentToken), "confirmation_requested", problemID) == nil {
		t.Fatalf("the reporter must be asked to confirm; got %v", notifications(residentToken))
	}

	// --- 4. reopen: the official learns the work came back -----------------------

	if code, b := call(http.MethodPost, "/api/problems/"+problemID+"/confirm", residentToken,
		map[string]any{"outcome": "not_solved"}); code != http.StatusOK {
		t.Fatalf("confirm not_solved: want 200, got %d (%v)", code, b)
	}
	if findNotif(notifications(officialToken), "case_reopened", problemID) == nil {
		t.Fatalf("the official must be told their case was reopened; got %v", notifications(officialToken))
	}

	// --- 5. obstacle → adjudication ---------------------------------------------

	// Resume to InProgress: an obstacle is legal from Planned or InProgress, and a
	// Reopened case gets there with a progress note.
	if code, b := call(http.MethodPost, "/api/official/cases/"+caseID+"/updates", officialToken, map[string]any{
		"kind": "progress", "text": "resuming after the reopen",
	}); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("resume update: want 200/201, got %d (%v)", code, b)
	}
	code, obstacle := call(http.MethodPost, "/api/official/cases/"+caseID+"/report-obstacle", officialToken, map[string]any{
		"category": "budget", "whatBlocks": "no allocation this quarter",
		"whoUnblocks": "the upazila engineer", "proofTried": "wrote twice in June",
	})
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("report obstacle: want 200/201, got %d (%v)", code, obstacle)
	}

	// The MONITOR is told, not the free-text authority the official named — the
	// documented deviation (A.3.9 constraint 5). The mayor's monitor is the upazila
	// chairman, claimed above.
	declared := findNotif(notifications(monitorToken), "obstacle_declared", problemID)
	if declared == nil {
		t.Fatalf("the monitor must be told an obstacle was declared; got %v", notifications(monitorToken))
	}
	if declared["detail"] != "the upazila engineer" {
		t.Errorf("obstacle_declared detail = %v, want the free-text whoUnblocks carried as context", declared["detail"])
	}

	obstacleID := ""
	if code, o := call(http.MethodGet, "/api/problems/"+problemID+"/obstacle", "", nil); code == http.StatusOK {
		obstacleID, _ = o["id"].(string)
	}
	if obstacleID == "" {
		t.Fatalf("the public obstacle read must return the declared obstacle")
	}
	if code, b := call(http.MethodPost, "/api/admin/obstacles/"+obstacleID+"/adjudicate", adminToken,
		map[string]any{"decision": "confirm"}); code != http.StatusOK {
		t.Fatalf("adjudicate: want 200, got %d (%v)", code, b)
	}
	adjudicated := findNotif(notifications(officialToken), "obstacle_adjudicated", problemID)
	if adjudicated == nil {
		t.Fatalf("the official must be told the verdict on their obstacle; got %v", notifications(officialToken))
	}
	if adjudicated["detail"] != "confirmed" {
		t.Errorf("obstacle_adjudicated detail = %v, want 'confirmed'", adjudicated["detail"])
	}

	// --- 6. the gate and the no-oracle rule -------------------------------------

	// Anonymous is 401, NOT an empty list: "you have no messages" and "we do not
	// know who you are" are different claims and must not look the same.
	if code, _ := callList(http.MethodGet, "/api/me/notifications", ""); code != http.StatusUnauthorized {
		t.Fatalf("anonymous notification list: want 401, got %d", code)
	}
	if code, _ := call(http.MethodGet, "/api/me/notifications/unread-count", "", nil); code != http.StatusUnauthorized {
		t.Fatalf("anonymous unread-count: want 401, got %d", code)
	}
	// RequireAnyRole admits all four roles — an admin has their own list like anyone.
	if code, _ := callList(http.MethodGet, "/api/me/notifications", adminToken); code != http.StatusOK {
		t.Fatalf("an admin must reach their own notifications: got %d", code)
	}

	// Marking SOMEONE ELSE'S notification read is 404, never 403 — an id that exists
	// and belongs to another person must be indistinguishable from one that does
	// not, or the route enumerates notification ids.
	othersID, _ := approved["id"].(string)
	if code, _ := call(http.MethodPost, "/api/me/notifications/"+othersID+"/read", officialToken, nil); code != http.StatusNotFound {
		t.Fatalf("marking another account's notification read: want 404, got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/me/notifications/notif-does-not-exist/read", residentToken, nil); code != http.StatusNotFound {
		t.Fatalf("marking an unknown id read: want 404, got %d", code)
	}

	// --- 7. mark read -----------------------------------------------------------

	before := unreadCount(residentToken)
	if code, _ := call(http.MethodPost, "/api/me/notifications/"+othersID+"/read", residentToken, nil); code != http.StatusNoContent {
		t.Fatalf("mark read: want 204, got %d", code)
	}
	if got := unreadCount(residentToken); got != before-1 {
		t.Errorf("unread after marking one read = %d, want %d", got, before-1)
	}
	// Idempotent: a double-tap is a no-op that still succeeds, not a 404.
	if code, _ := call(http.MethodPost, "/api/me/notifications/"+othersID+"/read", residentToken, nil); code != http.StatusNoContent {
		t.Fatalf("re-marking read: want 204, got %d", code)
	}

	code, cleared := call(http.MethodPost, "/api/me/notifications/read-all", residentToken, nil)
	if code != http.StatusOK {
		t.Fatalf("read-all: want 200, got %d (%v)", code, cleared)
	}
	if got := unreadCount(residentToken); got != 0 {
		t.Errorf("unread after read-all = %d, want 0", got)
	}
	// The rows survive being read — marking read is not deleting.
	readRow := findNotif(notifications(residentToken), "problem_approved", problemID)
	if readRow == nil {
		t.Fatal("a read notification must still be listed")
	}
	if readRow["readAt"] == nil {
		t.Error("readAt must be a timestamp once read, not null")
	}

	// --- 8. the reporter's hard delete erases every party's notifications --------

	// A.3.3: a deleted report leaves no trace anywhere. notifications.problem_id's
	// ON DELETE CASCADE is what gives that for free, and this is the assertion that
	// the FK is actually doing it — dropping it would be silent otherwise.
	if code, _ := call(http.MethodDelete, "/api/problems/"+problemID, residentToken, nil); code != http.StatusNoContent {
		t.Fatalf("delete problem: want 204, got %d", code)
	}
	for _, tc := range []struct {
		who   string
		token string
	}{
		{"reporter", residentToken},
		{"official", officialToken},
		{"monitor", monitorToken},
	} {
		for _, r := range notifications(tc.token) {
			if row, ok := r.(map[string]any); ok && row["problemId"] == problemID {
				t.Errorf("%s still has a notification for the deleted problem: %v", tc.who, row)
			}
		}
	}
}

// TestIntegration_SecondDoneNotifiesAgain pins the ONE deliberate duplicate in the
// set, and the reason migration 000023 carries no unique index.
//
// The cycle is Done → not_solved → Reopened → resume → Done again. The second Done
// is a NEW claim of completion about work done since the first was rejected;
// suppressing it would leave the reporter holding a notification they may already
// have marked read, with no signal that the official had answered them. This is
// idx_case_observations_open's reasoning pointed the other way.
func TestIntegration_SecondDoneNotifiesAgain(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("set TEST_DATABASE_URL (a migrated, seeded DB) to run the integration test")
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	cfg := &config.Config{
		DatabaseURL: dbURL, JWTSecret: "integration-secret", JWTTTL: time.Hour,
		ValidityThreshold: 1,
		ResponseDeadline:  7 * 24 * time.Hour, BlockerReviewWindow: 72 * time.Hour,
	}
	router := newRouter(cfg, pool, slog.New(slog.NewTextHandler(io.Discard, nil)))

	call := func(method, path, token string, body any) (int, map[string]any) {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.RemoteAddr = "127.0.0.1:5557"
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}
	callList := func(method, path, token string) (int, []any) {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = "127.0.0.1:5557"
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var out []any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM audit_entries WHERE target_id IN (
			SELECT id FROM problems WHERE title LIKE 'notif redo test %')`)
		_, _ = pool.Exec(ctx, `DELETE FROM problems WHERE title LIKE 'notif redo test %'`)
	})

	login := func(phone, password string) string {
		t.Helper()
		code, body := call(http.MethodPost, "/api/auth/login", "", map[string]any{"phone": phone, "password": password})
		if code != http.StatusOK {
			t.Fatalf("login %s: want 200, got %d (%v)", phone, code, body)
		}
		token, _ := body["token"].(string)
		return token
	}
	residentToken := login("01810000001", "resident123")
	adminToken := login("01910000009", "admin123")
	officialToken := login("01710000001", "official123")

	countConfirmRequests := func(problemID string) int {
		code, list := callList(http.MethodGet, "/api/me/notifications", residentToken)
		if code != http.StatusOK {
			t.Fatalf("list notifications: want 200, got %d", code)
		}
		n := 0
		for _, r := range list {
			if row, ok := r.(map[string]any); ok && row["type"] == "confirmation_requested" && row["problemId"] == problemID {
				n++
			}
		}
		return n
	}

	code, body := call(http.MethodPost, "/api/problems", residentToken, map[string]any{
		"title": "notif redo test culvert", "description": "collapsed culvert",
		"location":          map[string]any{"areaId": "pourashava-dhamoirhat", "address": "Ward 4"},
		"pointedOfficialId": "off-mayor",
	})
	if code != http.StatusCreated {
		t.Fatalf("report: want 201, got %d (%v)", code, body)
	}
	problemID, _ := body["id"].(string)

	if code, _ := call(http.MethodPost, "/api/admin/problems/"+problemID+"/approve", adminToken, nil); code != http.StatusOK {
		t.Fatalf("approve: got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/admin/problems/"+problemID+"/assign", adminToken, map[string]any{
		"officialId": "off-mayor", "priority": "normal",
	}); code != http.StatusCreated {
		t.Fatalf("assign: got %d", code)
	}

	code, cases := callList(http.MethodGet, "/api/official/cases", officialToken)
	if code != http.StatusOK {
		t.Fatalf("official cases: got %d", code)
	}
	var caseID string
	for _, c := range cases {
		if row, ok := c.(map[string]any); ok && row["problemId"] == problemID {
			caseID, _ = row["id"].(string)
		}
	}
	if caseID == "" {
		t.Fatalf("case not materialized; got %v", cases)
	}

	markDone := func(step string) {
		t.Helper()
		if code, b := call(http.MethodPost, "/api/official/cases/"+caseID+"/done", officialToken, nil); code != http.StatusOK {
			t.Fatalf("%s mark done: want 200, got %d (%v)", step, code, b)
		}
	}

	// First pass: acknowledge → plan → evidence → done.
	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/acknowledge", officialToken,
		map[string]any{"decision": "accept"}); code != http.StatusOK {
		t.Fatalf("acknowledge: got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/plan", officialToken, map[string]any{
		"strategy": "rebuild the culvert", "tasks": []string{"survey", "build"},
		"suggestionResponse": "adopting the proposal",
	}); code != http.StatusOK {
		t.Fatalf("plan: got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/evidence", officialToken, map[string]any{
		"beforeImageUrl": "data:image/png;base64,b", "afterImageUrl": "data:image/png;base64,a",
	}); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("evidence: got %d", code)
	}
	// Planned → InProgress. Done is only legal from InProgress, and a progress note
	// is what starts the work — the lifecycle has no "begin" action of its own.
	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/updates", officialToken, map[string]any{
		"kind": "progress", "text": "headwall poured",
	}); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("start-work update: got %d", code)
	}
	markDone("first")

	if got := countConfirmRequests(problemID); got != 1 {
		t.Fatalf("confirmation requests after the first Done = %d, want 1", got)
	}

	// The resident rejects it, the official redoes the work, and marks it done again.
	if code, _ := call(http.MethodPost, "/api/problems/"+problemID+"/confirm", residentToken,
		map[string]any{"outcome": "not_solved"}); code != http.StatusOK {
		t.Fatalf("confirm not_solved: got %d", code)
	}
	if code, _ := call(http.MethodPost, "/api/official/cases/"+caseID+"/updates", officialToken, map[string]any{
		"kind": "progress", "text": "redoing the headwall",
	}); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("resume update: got %d", code)
	}
	markDone("second")

	// TWO, not one. If this ever reads 1, someone has added a uniqueness guard —
	// read A.3.9 constraint 6 before "fixing" this test.
	if got := countConfirmRequests(problemID); got != 2 {
		t.Errorf("confirmation requests after the second Done = %d, want 2 — a redo is a NEW claim of completion", got)
	}

	// Clean up so the row does not sit Done in the shared DB.
	_, _ = call(http.MethodDelete, "/api/problems/"+problemID, residentToken, nil)
}
