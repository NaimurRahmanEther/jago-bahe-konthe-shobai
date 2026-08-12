import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useConfirmProblem, useProblem } from '../../hooks/useProblems.js'
import { useOfficials } from '../../hooks/useOfficials.js'
import { useProposeSuggestion, useSuggestions } from '../../hooks/useSuggestions.js'
import { useAuth } from '../../auth/useAuth.js'
import StatusBadge from '../../components/problem/StatusBadge.jsx'
import ValidationVote from '../../components/problem/ValidationVote.jsx'
import ReporterActions from '../../components/problem/ReporterActions.jsx'
import AssignedOfficial from '../../components/problem/AssignedOfficial.jsx'
import ProgressPanel from '../../components/resolution/ProgressPanel.jsx'
import { useProgress } from '../../hooks/useCases.js'
import { hasCase } from '../../lib/problemStatus.js'
import SuggestionList from '../../components/suggestion/SuggestionList.jsx'
import SuggestionForm from '../../components/suggestion/SuggestionForm.jsx'
import ConfirmBox from '../../components/resolution/ConfirmBox.jsx'
import ObstacleJudgment from '../../components/resolution/ObstacleJudgment.jsx'
import AuditTrail from '../../components/resolution/AuditTrail.jsx'
import Spinner from '../../components/ui/Spinner.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import ImageViewer from '../../components/ui/ImageViewer.jsx'
import Section from '../../components/ui/Section.jsx'

export default function ProblemDetail() {
  const { t } = useTranslation()
  const { id } = useParams()
  const { user } = useAuth()
  // The photo currently open full-screen, or null. One viewer serves the
  // reporter's photo and every before/after pair on the page.
  const [zoomed, setZoomed] = useState(null)
  const { data: problem, isLoading, isError, error, refetch } = useProblem(id)
  const { data: officials } = useOfficials()
  const { data: suggestions, isLoading: suggestionsLoading } = useSuggestions(id)
  const proposeSuggestion = useProposeSuggestion(id)
  const confirmProblem = useConfirmProblem(id)

  // The official's own account of the work, for ANYONE — the endpoint is ungated
  // by design ("a gate here would lock the public out of the public record"), and
  // until now the only screen that read it was /me, behind a login. That made an
  // official's published plan effectively private to the one resident who filed
  // the report, which is the opposite of what publishing it is for.
  //
  // enabled is gated on the status, not on auth: a problem with no case would
  // answer null harmlessly, but a request that can only ever return null is one
  // worth not making. Called unconditionally, before the early returns below —
  // problem is undefined while loading, and optional chaining keeps it disabled
  // until the status is known.
  const {
    data: progress,
    isLoading: progressLoading,
    isError: progressError,
    refetch: refetchProgress,
  } = useProgress(id, { enabled: hasCase(problem?.status) })

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  // A 404 is not a failure to retry — the report is deleted, or was never readable
  // by this caller (a pending report 404s to a stranger rather than 403ing, so the
  // gate cannot be used to confirm a hidden report exists). Offering Retry here
  // would loop on a request that can only ever fail. client.js normalizes every
  // rejection to { status, code, message }.
  if (error?.status === 404) {
    return (
      <div className="flex flex-col items-start gap-3 rounded-card border border-hairline bg-surface p-6 animate-fade-in">
        <p className="text-title font-semibold text-ink">{t('problem.detail.notFoundTitle')}</p>
        <p className="text-meta text-muted">{t('problem.detail.notFoundBody')}</p>
        <Link to="/problems" className="font-medium text-brand">
          {t('problem.detail.notFoundAction')}
        </Link>
      </div>
    )
  }

  if (isError || !problem) {
    return <ErrorState message={t('problem.detail.error')} onRetry={refetch} />
  }

  const official = officials?.find((o) => o.id === problem.pointedOfficialId)

  return (
    // gap-6 between sections (the AdminHome rhythm), gap-3 within one. The page was a
    // flat gap-4 column of 14 siblings, which is why four identically-bordered cards
    // read as one undifferentiated stack.
    <div className="flex w-full flex-col gap-6">
      <Link to="/problems" className="font-medium text-brand">
        {t('problem.detail.back')}
      </Link>

      {/* A rejection stays on the public record with its ground — a rejected
          report has a witness rather than simply disappearing. */}
      {problem.status === 'Rejected' && (
        <div className="rounded-card bg-rejected-bg p-4 text-sm text-rejected-fg animate-fade-in">
          <p className="font-semibold">{t('problem.detail.rejectedTitle')}</p>
          {problem.rejectionReason && (
            <p className="mt-1">
              {t('problem.detail.rejectedReason')}: {t(`problem.rejectionReason.${problem.rejectionReason}`)}
            </p>
          )}
        </div>
      )}

      {/* A pending report is not yet public — it is awaiting its union admin's
          screening. Only the reporter (and that admin) can reach this page at all,
          so the notice reassures the person who filed it that it is in the queue,
          not lost. */}
      {problem.status === 'PendingApproval' && (
        <div className="rounded-card bg-pendingapproval-bg p-4 text-sm text-pendingapproval-fg animate-fade-in">
          <p className="font-semibold">{t('problem.detail.pendingTitle')}</p>
          <p className="mt-1">{t('problem.detail.pendingBody')}</p>
        </div>
      )}

      {/* A withdrawal stays on the public record too — the reporter retracted it,
          it did not vanish. */}
      {problem.status === 'Withdrawn' && (
        <div className="rounded-card bg-withdrawn-bg p-4 text-sm text-withdrawn-fg animate-fade-in">
          <p className="font-semibold">{t('problem.detail.withdrawnTitle')}</p>
        </div>
      )}

      {/* THE REPORT — the page's subject, so it is unboxed. Everything below it is a
          named section; this is the thing they are all about. */}
      <header className="animate-fade-in">
        {/* The badge leads on its own line rather than sharing the top line with the
            title: at 30px the title would win that fight, and the status is the
            first thing a reader is looking for (Guideline §1). */}
        <StatusBadge status={problem.status} />
        <h1 className="mt-2 text-balance text-display font-semibold text-ink">{problem.title}</h1>
        <p className="mt-1.5 text-meta text-muted">
          {problem.location.address}
          {official &&
            ` · ${t('problem.detail.pointedTo', { name: official.name, tier: t(`problem.tier.${official.tier}`) })}`}
        </p>
        {/* Who is actually on it, in one line, before the reader is asked anything.
            The full card — with the override and its reason — lives in the authority
            section below, where the official's work is. Two placements, one
            component: the compact variant exists for exactly this. */}
        <AssignedOfficial problem={problem} compact />
        <p className="mt-3 max-w-[70ch] text-ink">{problem.description}</p>
        {/* A framed preview with a fixed aspect ratio, so the page does not jump
            as the photo decodes (A.6). The crop that frame costs is what the
            click undoes — the viewer shows it whole. */}
        {problem.imageUrl && (
          <button
            type="button"
            onClick={() => setZoomed({ src: problem.imageUrl, alt: t('problem.report.image') })}
            aria-label={t('problem.detail.viewImage')}
            className="mt-3 block w-full max-w-[48rem] cursor-zoom-in rounded-card"
          >
            <img
              src={problem.imageUrl}
              alt={t('problem.report.image')}
              loading="lazy"
              className="aspect-[4/3] w-full rounded-card border border-hairline object-cover"
            />
          </button>
        )}

        {/* The reporter's own controls: edit while unvalidated, withdraw before
            assignment. Renders nothing for anyone else, or once both windows close. */}
        <div className="mt-4">
          <ReporterActions problem={problem} />
        </div>
      </header>

      {/* 1. IS THIS REAL? — unboxed, because ValidationVote draws its own card. */}
      <Section title={t('problem.detail.validateTitle')} boxed={false}>
        <ValidationVote problem={problem} />
      </Section>

      {/* 2. WHAT SHOULD BE DONE — every proposal in one place: the reporter's own
          first, then the community's. They used to be far apart, the reporter's
          buried in the header at 14px muted, which made the person who found the
          problem the quietest voice in the conversation about fixing it. */}
      <Section
        title={t('problem.detail.suggestionsTitle')}
        meta={suggestions?.length ? t('problem.detail.suggestionsCount', { count: suggestions.length }) : undefined}
      >
        <div className="flex flex-col gap-4">
          {problem.proposedSolution && (
            <div>
              <p className="text-meta font-medium text-muted">{t('problem.detail.reporterProposal')}</p>
              <p className="mt-1 text-ink">{problem.proposedSolution}</p>
            </div>
          )}
          {suggestionsLoading ? (
            <Spinner />
          ) : (
            <SuggestionList problemId={id} suggestions={suggestions ?? []} />
          )}
          <SuggestionForm onSubmit={(text) => proposeSuggestion.mutateAsync(text)} />
        </div>
      </Section>

      {/* 3. WHAT THE AUTHORITY DID — and it comes AFTER the proposals, reversing the
          old order. The comment that used to sit here claimed the panel was above the
          suggestions "so the page reads in the order the accountability runs: the
          community proposed X, the official answered Y" — which is what this order
          actually produces. As written, the answer was read before the question.

          Rendered even when nothing has happened yet: on a public accountability page,
          "no official has taken this up" is itself a fact worth publishing, not an
          absence to hide. */}
      <Section title={t('problem.detail.authorityTitle')}>
        <div className="flex flex-col gap-4">
          <AssignedOfficial problem={problem} />

          {problem.status === 'Blocked' && <ObstacleJudgment problemId={id} />}

          {hasCase(problem.status) ? (
            <ProgressPanel
              progress={progress}
              isLoading={progressLoading}
              isError={progressError}
              onRetry={refetchProgress}
              onZoom={(src, alt) => setZoomed({ src, alt })}
            />
          ) : (
            <p className="text-muted">{t('problem.detail.noAuthorityAction')}</p>
          )}

          {/* B14 is unbuilt, so problemDTO.evidence is always empty and this never
              renders today; ProgressPanel serves the evidence that does arrive, via
              progressDTO. When B14 lands, DELETE THIS BLOCK rather than shipping both —
              two galleries of the same photos on one page, and §2 forbids two public
              views of one case disagreeing. */}
          {problem.evidence?.length > 0 && (
            <div>
              <h3 className="text-title font-semibold text-ink">{t('problem.detail.evidenceTitle')}</h3>
              <div className="mt-2 flex flex-col gap-3">
                {problem.evidence.map((ev) => (
                  <div key={ev.id} className="grid grid-cols-2 gap-3">
                    {[
                      { src: ev.beforeImageUrl, label: t('resolution.evidence.before') },
                      { src: ev.afterImageUrl, label: t('resolution.evidence.after') },
                    ].map((photo) => (
                      <figure key={photo.label} className="flex flex-col gap-1">
                        <button
                          type="button"
                          onClick={() => setZoomed({ src: photo.src, alt: photo.label })}
                          aria-label={t('problem.detail.viewImage')}
                          className="block w-full cursor-zoom-in rounded-control"
                        >
                          <img
                            src={photo.src}
                            alt={photo.label}
                            loading="lazy"
                            className="aspect-[4/3] w-full rounded-control border border-hairline object-cover"
                          />
                        </button>
                        <figcaption className="text-micro text-muted">{photo.label}</figcaption>
                      </figure>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Only the reporter closes their own report. Gating on the role instead
              would offer confirm on anyone's problem — the backend refuses it
              (ErrNotReporter), but a resident should never be shown an action that
              cannot succeed, and §2 says a Done case resolves only when THE REPORTING
              residents confirm. Ownership implies resident, so no role check. */}
          {problem.status === 'Done' && problem.reporterId === user?.id && (
            <ConfirmBox
              onConfirm={(outcome) => confirmProblem.mutateAsync(outcome)}
              isSubmitting={confirmProblem.isPending}
            />
          )}
        </div>
      </Section>

      {/* 4. THE RECORD — collapsed. Unboxed: <details> is its own affordance, and a
          card around a single summary line would be heavier than what it contains. */}
      <Section title={t('problem.detail.recordTitle')} boxed={false}>
        <AuditTrail entries={problem.audit ?? []} />
      </Section>

      <ImageViewer
        open={Boolean(zoomed)}
        src={zoomed?.src}
        alt={zoomed?.alt}
        onClose={() => setZoomed(null)}
      />
    </div>
  )
}
