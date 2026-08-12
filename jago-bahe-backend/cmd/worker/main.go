// Command worker runs background jobs on a schedule. B6 wires
// resolution/application/escalate_overdue here to scan for cases past their
// deadline (D) or blocker review window (R) and escalate up the monitor ladder.
// It shares the same internal/ code as the api.
//
// B19 gave that scan its read side: as well as raising a case's escalation level,
// it now OPENS and CLOSES the observations that put a copy of a silent case in
// front of the officials above the assignee, and closes them again when the
// official finally answers. That is the half a monitor can actually see, so a
// worker that is not running no longer merely fails to increment a number nobody
// reads — it leaves every observation page in the seat empty.
//
// This process is not optional: escalation exists only here, so an api running
// without a worker beside it will never escalate an overdue case — silently, since
// nothing in the api reports the absence. `make up` starts both; a no-Docker run
// must start both by hand.
//
// It once also ran problem/application/auto_approve_pending, which published
// problems left unscreened past the window A. That job died with the
// pre-publication gate: problems are public the moment they are filed, so there is
// nothing left for silence to bury.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jago-bahe-backend/config"
	identityapp "jago-bahe-backend/internal/identity/application"
	identitydomain "jago-bahe-backend/internal/identity/domain"
	identitypg "jago-bahe-backend/internal/identity/infrastructure/postgres"
	resolutionapp "jago-bahe-backend/internal/resolution/application"
	resolutionpg "jago-bahe-backend/internal/resolution/infrastructure/postgres"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	areapg "jago-bahe-backend/internal/shared/area/infrastructure/postgres"
	auditpg "jago-bahe-backend/internal/shared/audit/infrastructure/postgres"
	"jago-bahe-backend/pkg/logger"
	"jago-bahe-backend/pkg/postgres"
)

// ladderAdapter bridges the resolution context's Ladder port to the identity
// directory and the area tree, so the worker can resolve WHICH official an
// escalation should surface to.
//
// It is the worker's twin of cmd/api's resolutionOfficialAdapter, and both are
// three lines over identityapp.MonitorChain rather than two copies of the walk —
// the same shape identityapp.AdminUnion already has in both roots. A second
// implementation of "who monitors whom" is how the assignment's monitor and the
// escalation's observers would come to disagree.
type ladderAdapter struct {
	officials identitydomain.OfficialRepository
	areas     areadomain.Repository
}

func (a ladderAdapter) Chain(ctx context.Context, officialID string, maxRungs int) ([]string, error) {
	return identityapp.MonitorChain(ctx, a.officials, a.areas, officialID, maxRungs)
}

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// B6: the deterministic escalation scan. It shares the same internal/ code as
	// the api — the case repository, the audit trail, and the tunables D and R.
	//
	// B19 added the officials directory and the area tree, this process's first
	// dependency outside resolution: an escalation now has to land on a NAMED
	// official (the monitor one rung up, then the tier above), not merely bump an
	// integer, so the worker walks the same accountability ladder the api does. The
	// walk itself lives in identity — see identityapp.MonitorChain.
	caseRepo := resolutionpg.NewCaseRepository(pool)
	auditRepo := auditpg.NewAuditRepository(pool)
	officialRepo := identitypg.NewOfficialRepository(pool)
	areaRepo := areapg.NewAreaRepository(pool)
	escalate := resolutionapp.NewEscalateOverdue(
		caseRepo, auditRepo,
		ladderAdapter{officials: officialRepo, areas: areaRepo},
		cfg.ResponseDeadline, cfg.BlockerReviewWindow,
	)

	runScan := func() {
		now := time.Now().UTC()

		res, err := escalate.Run(ctx, now)
		if err != nil {
			log.Error("escalation scan failed", "err", err)
			return
		}
		if res.Escalated > 0 || res.ObservationsOpened > 0 || res.ObservationsClosed > 0 {
			log.Info("escalation scan",
				"escalated", res.Escalated,
				"observationsOpened", res.ObservationsOpened,
				"observationsClosed", res.ObservationsClosed)
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	log.Info("worker started", "interval", "1m")
	runScan() // scan once on boot so escalation isn't delayed a full interval
	for {
		select {
		case <-stop:
			log.Info("worker shutting down")
			return
		case <-ticker.C:
			runScan()
		}
	}
}
