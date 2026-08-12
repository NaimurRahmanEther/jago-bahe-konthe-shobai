import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/useAuth.js'
import { useWithdrawProblem, useDeleteProblem } from '../../hooks/useProblems.js'
import Button from '../ui/Button.jsx'
import Modal from '../ui/Modal.jsx'

// A report is editable only while it is still Reported and nobody has endorsed it
// (a valid vote landed) — after that, residents have validated the text as written.
// It is withdrawable any time before assignment; once assigned a case exists and an
// official is working in public. PendingApproval is in the withdraw window but not
// the edit one: an unscreened report is still wholly its reporter's, so they may
// retract it, while editing is about not rewriting text others have endorsed.
//
// These mirror the backend's Editable() and Withdrawable() in problem/domain/status.go
// and are UX only — the server enforces the real rule. Deleting has no predicate
// because it has no window (see below).
const canEditProblem = (p) => p.status === 'Reported' && p.validCount === 0
const canWithdrawProblem = (p) =>
  p.status === 'PendingApproval' || p.status === 'Reported' || p.status === 'Validated'

/**
 * The reporter's own Edit / Withdraw / Delete controls for a report. Renders
 * nothing for anyone who is not the reporter.
 *
 * @param {{problem: import('../../lib/types/models.js').Problem, className?: string, onDeleted?: () => void}} props
 */
export default function ReporterActions({ problem, className = '', onDeleted }) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const navigate = useNavigate()
  const withdraw = useWithdrawProblem(problem.id)
  const remove = useDeleteProblem(problem.id)
  // One modal at a time: null | 'withdraw' | 'delete'. Two booleans would let both
  // open at once, and these two actions must never be confusable.
  const [modal, setModal] = useState(null)
  const [note, setNote] = useState('')
  const [error, setError] = useState('')

  const isOwner = problem.reporterId === user?.id
  const editable = canEditProblem(problem)
  const withdrawable = canWithdrawProblem(problem)
  // Deleting has no window — the reporter may erase their report at any status,
  // including a resolved one. So the guard is ownership alone; it used to also bail
  // when both windows above had closed, which would now hide delete on exactly the
  // reports where it is the only action left.
  if (!isOwner) return null

  function close() {
    setModal(null)
    setError('')
  }

  async function onWithdraw() {
    setError('')
    try {
      await withdraw.mutateAsync(note.trim() || undefined)
      close()
    } catch {
      // The window can close between opening this page and confirming (someone
      // validated or an admin assigned it); the backend answers 409, and the honest
      // thing is to say so rather than pretend it worked.
      setError(t('problem.detail.withdrawFailed'))
    }
  }

  async function onDelete() {
    setError('')
    try {
      await remove.mutateAsync()
      close()
      // The problem is gone, so staying on its page would 404. onDeleted lets a
      // list row stay put (its query refetches without the row); on the detail page
      // there is nowhere to stay.
      if (onDeleted) onDeleted()
      else navigate('/me')
    } catch {
      setError(t('problem.detail.deleteFailed'))
    }
  }

  return (
    <div className={`flex flex-wrap gap-2 ${className}`}>
      {editable && (
        <Link
          to={`/problems/${problem.id}/edit`}
          className="inline-flex min-h-11 items-center justify-center rounded-control border border-hairline bg-surface px-4 font-medium text-ink hover:bg-canvas"
        >
          {t('problem.detail.edit')}
        </Link>
      )}
      {withdrawable && (
        <Button variant="secondary" onClick={() => setModal('withdraw')}>
          {t('problem.detail.withdraw')}
        </Button>
      )}
      <Button variant="secondary" onClick={() => setModal('delete')}>
        {t('problem.detail.delete')}
      </Button>

      <Modal open={modal === 'withdraw'} onClose={close} title={t('problem.detail.withdrawConfirmTitle')}>
        <p className="text-ink">{t('problem.detail.withdrawConfirmBody')}</p>
        <div className="mt-4 flex flex-col gap-1">
          <label htmlFor="withdraw-note" className="font-medium text-ink">
            {t('problem.detail.withdrawNote')} <span className="font-normal text-muted">({t('problem.report.optional')})</span>
          </label>
          <textarea
            id="withdraw-note"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={3}
            className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
          />
        </div>
        {error && <p className="mt-2 text-meta text-reopened-fg">{error}</p>}
        <div className="mt-4 flex gap-3">
          <Button variant="secondary" className="flex-1" onClick={close} disabled={withdraw.isPending}>
            {t('common.cancel')}
          </Button>
          <Button className="flex-1 bg-rejected-dot hover:bg-rejected-fg" onClick={onWithdraw} disabled={withdraw.isPending}>
            {t('problem.detail.withdrawConfirm')}
          </Button>
        </div>
      </Modal>

      {/* Deleting is irreversible and takes other people's work with it, so this
          modal spells out the cost rather than asking a bare "are you sure?". The
          withdraw alternative is offered here because this is the moment a reporter
          is most likely to want it and least likely to know it exists. */}
      <Modal open={modal === 'delete'} onClose={close} title={t('problem.detail.deleteConfirmTitle')}>
        <p className="text-ink">{t('problem.detail.deleteConfirmBody')}</p>
        <p className="mt-3 rounded-control bg-rejected-bg px-3 py-2 text-meta text-rejected-fg">
          {t('problem.detail.deleteWarning')}
        </p>
        {withdrawable && (
          <p className="mt-3 text-meta text-muted">{t('problem.detail.deleteWithdrawHint')}</p>
        )}
        {error && <p className="mt-2 text-meta text-reopened-fg">{error}</p>}
        <div className="mt-4 flex gap-3">
          <Button variant="secondary" className="flex-1" onClick={close} disabled={remove.isPending}>
            {t('common.cancel')}
          </Button>
          <Button className="flex-1 bg-rejected-dot hover:bg-rejected-fg" onClick={onDelete} disabled={remove.isPending}>
            {t('problem.detail.deleteConfirm')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}
