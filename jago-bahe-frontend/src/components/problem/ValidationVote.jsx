import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'
import ValidationCount from './ValidationCount.jsx'
import { useValidateProblem } from '../../hooks/useProblems.js'
import { useAuth } from '../../auth/useAuth.js'

// A report that has been taken down is not a question the community is still being
// asked. The backend would accept the vote — cast_validation_vote refuses only a
// problem that is not publicly visible — but offering "is this real?" on a report
// its own reporter withdrew is noise, and on a feed it is noise on every such row.
//
// Post-assignment voting is deliberately still offered: A.3.1.1 constraint 3 makes
// continued validation the signal that lets an early assignment be judged, so
// Assigned / InProgress / Blocked / Done / Resolved all keep the controls.
const CLOSED_TO_VOTES = ['Rejected', 'Withdrawn', 'PendingApproval']

// The backend's refusal codes (pkg/httpx.ErrorBody.code), mapped to the advice
// that actually helps. Switching on the code rather than the status matters:
// not_verified and not_area_resident are BOTH 403 and need opposite advice — one
// says "wait for your admin", the other says "this report is not in your union".
const ERROR_KEYS = {
  already_voted: 'problem.validate.already',
  not_verified: 'problem.validate.notVerified',
  not_area_resident: 'problem.validate.notAreaResident',
  not_found: 'problem.validate.notFound',
}

// Fallback for a refusal that arrives without a code — a proxy or gateway error
// never passes through httpx.Error, so it has a status and nothing else. Only 409
// is unambiguous enough to name: 403 could be either eligibility failure, and
// guessing between them would give confidently wrong advice.
const STATUS_KEYS = { 409: 'problem.validate.already' }

function errorKey(err) {
  return ERROR_KEYS[err?.code] ?? STATUS_KEYS[err?.status] ?? 'problem.validate.failed'
}

/**
 * The valid/invalid controls plus the public validation count.
 *
 * Detail page only. A compact variant of this lived on the feed card briefly and
 * was taken back out: "is this real?" is a question to answer after reading the
 * report, not from a title and an address. The feed shows the count alone
 * (ValidationCount) and links here to act on it.
 *
 * @param {{problem: import('../../lib/types/models.js').Problem}} props
 */
export default function ValidationVote({ problem }) {
  const { t } = useTranslation()
  const { role, user, updateUser } = useAuth()
  const validateProblem = useValidateProblem()

  // The just-cast vote, tagged with the problem it was cast on. Both halves matter:
  // the vote covers the gap between the mutation resolving and the refetched myVote
  // arriving, and the id keeps it from leaking onto the NEXT problem.
  //
  // This is the latching bug. It used to be a bare useState seeded from myVote once
  // per MOUNT, and ProblemDetail renders this component without a key, so the router
  // keeps it mounted across an :id change: voting on one report left the next
  // report's buttons disabled and thanked you for a vote you never cast. Deriving
  // from myVote alone is not enough — an untagged local vote latches the same way.
  const [justVoted, setJustVoted] = useState(null)
  const [error, setError] = useState('')

  const votedChoice = problem.myVote ?? (justVoted?.id === problem.id ? justVoted.vote : null)
  const isResident = role === 'resident'
  const isClosed = CLOSED_TO_VOTES.includes(problem.status)

  // Deliberately NOT gated on user.verified. The flag is captured at login and
  // there is no endpoint to re-read it, so an admin verifying a signed-in resident
  // leaves it stale-false — disabling the buttons on it would lock out exactly the
  // person who just became eligible. The backend is the authority (A.5.7): we
  // advise below, let them try, and heal the stale flag when a vote succeeds.
  const canVote = isResident && !isClosed
  const needsVerification = isResident && user?.verified === false

  async function castVote(vote) {
    if (votedChoice || validateProblem.isPending) return
    setError('')
    try {
      await validateProblem.mutateAsync({ id: problem.id, vote })
      // Only now. This used to be set before the request, so "ধন্যবাদ" rendered
      // whether or not the server accepted the vote — the UI thanking someone for
      // a civic act that did not happen.
      setJustVoted({ id: problem.id, vote })
      // The server accepted it, so this account IS verified whatever the stored
      // session claims. Correct it, or the advisory notice keeps nagging someone
      // who has just been verified.
      if (user?.verified === false) updateUser({ verified: true })
    } catch (err) {
      setError(t(errorKey(err)))
    }
  }

  return (
    <div className="rounded-card border border-hairline bg-surface p-4">
      <p className="font-medium text-ink">{t('problem.validate.prompt')}</p>

      {needsVerification && (
        <p className="mt-3 rounded-control bg-pendingapproval-bg p-3 text-meta text-pendingapproval-fg">
          {t('problem.validate.notVerified')}
        </p>
      )}

      {canVote && (
        <div className="mt-3 flex gap-3">
          <Button
            variant={votedChoice === 'valid' ? 'primary' : 'secondary'}
            className="min-h-13 flex-1 text-base"
            onClick={() => castVote('valid')}
            disabled={Boolean(votedChoice) || validateProblem.isPending}
          >
            {t('problem.validate.valid')}
          </Button>
          <Button
            variant={votedChoice === 'invalid' ? 'primary' : 'secondary'}
            className="min-h-13 flex-1 text-base"
            onClick={() => castVote('invalid')}
            disabled={Boolean(votedChoice) || validateProblem.isPending}
          >
            {t('problem.validate.invalid')}
          </Button>
        </div>
      )}

      {/* Every closed branch says WHY. This used to render only for non-residents,
          so a resident looking at a withdrawn or rejected report saw the prompt
          "is this problem real?" with no buttons and no explanation — which reads
          as the feature being broken rather than the question being over. */}
      {!canVote && (
        <p className="mt-3 text-meta text-muted">
          {isClosed ? t('problem.validate.closed') : t('problem.validate.residentOnly')}
        </p>
      )}

      <ValidationCount count={problem.validCount} className="mt-3" />

      {votedChoice && (
        <p className="mt-2 text-meta text-validated-fg">
          {problem.myVote
            ? `${t('problem.validate.yourVote')}: ${t(`problem.validate.${votedChoice}`)}`
            : t('problem.validate.thanks')}
        </p>
      )}
      {/* role=alert so a screen reader announces the refusal — and so tests can
          tell it apart from the standing advisory above, which shares its wording
          when an unverified resident is refused (A.5.3 rule 4: scope, never
          getAllByText[0]). */}
      {error && (
        <p role="alert" className="mt-2 text-meta text-reopened-fg">
          {error}
        </p>
      )}
    </div>
  )
}
