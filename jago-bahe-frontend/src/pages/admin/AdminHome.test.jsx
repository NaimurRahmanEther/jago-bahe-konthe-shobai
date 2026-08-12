import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import AdminHome from './AdminHome.jsx'
import Moderation from './Moderation.jsx'
import * as moderationApi from '../../lib/api/moderation.js'
import * as assignmentsApi from '../../lib/api/assignments.js'
import * as scorecardApi from '../../lib/api/scorecard.js'

// There is no mock layer any more (CLAUDE.md A.5.5): these stub the api modules,
// which are the boundary. Every export is listed — the hooks `import * as`, so an
// omitted name is `undefined` at call time rather than an import error.
vi.mock('../../lib/api/moderation.js', () => ({
  listModeration: vi.fn(),
  approveProblem: vi.fn(),
  rejectProblem: vi.fn(),
  REJECTION_REASONS: ['spam', 'abusive', 'duplicate', 'wrong_area'],
}))

vi.mock('../../lib/api/assignments.js', () => ({
  listQueue: vi.fn(),
  assignWithinUnion: vi.fn(),
  listMyForwarding: vi.fn(),
  suggestForwarding: vi.fn(),
  listForwardingQueue: vi.fn(),
  forwardProblem: vi.fn(),
}))

vi.mock('../../lib/api/scorecard.js', () => ({
  getOfficialScorecard: vi.fn(),
  getSeatOverview: vi.fn(),
}))

vi.mock('../../hooks/useOfficials.js', () => ({
  useOfficials: () => ({ data: [{ id: 'off-1', name: 'Karim Uddin', tier: 'ward_member', areaId: 'ward-1' }] }),
}))

// A report awaiting screening. It is not public and not votable until this admin
// approves it — the hard gate (CLAUDE.md A.3.1).
const MODERATABLE = [
  {
    id: 'prob-5',
    title: 'বাজারের ড্রেনটি বন্ধ হয়ে আছে',
    description: 'ওয়ার্ড ১ এর বাজারের পাশের ড্রেন বন্ধ থাকায় পানি জমে থাকছে।',
    location: { areaId: 'ward-1', address: 'বাজার সড়ক, ওয়ার্ড ১' },
    reporterId: 'res-1',
    pointedOfficialId: 'off-1',
    status: 'PendingApproval',
    validCount: 0,
    validationThreshold: 5,
    createdAt: '2026-07-14T09:00:00.000Z',
  },
]

// GET /api/admin/queue returns ONE list holding both stages; the page splits it.
// Note the shape: `problemId` (not `id`) and a flat `address` (not `location`) —
// getting this wrong is what crashed every queue row before B17, so the fixture
// mirrors the DTO exactly rather than a Problem.
const QUEUE = [
  {
    problemId: 'prob-2',
    title: 'রাস্তার বাতিগুলো জ্বলছে না',
    status: 'Reported',
    address: 'স্কুল রোড, ওয়ার্ড ২',
    pointedOfficialId: 'off-1',
    areaId: 'ward-2',
    routing: 'union',
    validCount: 3,
    validationThreshold: 5,
  },
  {
    problemId: 'prob-3',
    title: 'ইউনিয়ন সড়কের সেতুটি মেরামত প্রয়োজন',
    status: 'Validated',
    address: 'ইউনিয়ন সড়ক সেতু',
    pointedOfficialId: 'off-1',
    areaId: 'union-1',
    // Every row here is union-routed: since B20 an above-union report never reaches
    // this queue, so a 'vote' row in this fixture would describe a response the API
    // cannot produce (A.3.8).
    routing: 'union',
    validCount: 6,
    validationThreshold: 5,
  },
]

// GET /api/admin/forwarding — the above-union reports this admin may ADVISE on.
// A different endpoint and a different shape from the queue above, because it is a
// different question: not "whom do I assign this to" but "where do I think the
// super admin should send it".
const FORWARDING = [
  {
    problemId: 'prob-9',
    title: 'উপজেলা সড়কের সেতুটি ভেঙে পড়েছে',
    status: 'Validated',
    address: 'উপজেলা সড়ক সেতু',
    areaId: 'union-1',
    pointedOfficialId: 'off-1',
    scope: 'upazila',
    validCount: 6,
    validationThreshold: 5,
    suggestions: [],
    topOfficialId: '',
    topCount: 0,
    mySuggestion: null,
  },
]

const SEAT = { problems: 5, cases: 2, resolved: 1, pending: 1, blocked: 1, avgResponseDays: 2.5 }

function renderWithProviders(ui) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(moderationApi.listModeration).mockResolvedValue(MODERATABLE)
  vi.mocked(assignmentsApi.listQueue).mockResolvedValue(QUEUE)
  vi.mocked(assignmentsApi.listMyForwarding).mockResolvedValue(FORWARDING)
  vi.mocked(scorecardApi.getSeatOverview).mockResolvedValue(SEAT)
})

describe('AdminHome', () => {
  // /admin used to drop the admin straight into the bare assignment queue, with no
  // home and no sign that a moderation queue existed at all. B17 added the third
  // section: the window between approving a report and its reaching V, which the
  // admin previously could not see into at all.
  it('shows all four sections and the seat overview', async () => {
    renderWithProviders(<AdminHome />)
    expect(screen.getByText('অ্যাডমিন ড্যাশবোর্ড')).toBeInTheDocument()
    // Scoped to the heading role, with a regex that rides over the "(n)" count the
    // heading grows once data lands. Note that the screening heading is now
    // "অনুমোদনের অপেক্ষায়" — it used to read "যাচাইয়ের অপেক্ষায়", but "যাচাই" is
    // the community's validation, not the admin's screening, and that word now
    // belongs to the middle section (CLAUDE.md A.5.3 rule 4).
    expect(screen.getByRole('heading', { name: /অনুমোদনের অপেক্ষায়/ })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /যাচাই চলছে/ })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /নির্ধারণের অপেক্ষায়/ })).toBeInTheDocument()
    // The fourth section (B20) — advice on above-union reports, which this admin
    // does NOT decide. It is a different kind of row from the three above it.
    expect(screen.getByRole('heading', { name: /পরামর্শ দিন/ })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'আসনের সারসংক্ষেপ' })).toBeInTheDocument()

    // The seat stats arrive from the API, so wait for the region to populate.
    // Scoped to the named region, never getAllByText(...)[0]: "সমাধান হয়েছে" is
    // six different keys and "আটকে আছে" three, so an unscoped query here would
    // pass or fail on DOM order rather than on the tiles actually rendering.
    const overview = await screen.findByRole('region', { name: 'আসনের সারসংক্ষেপ' })
    expect(await within(overview).findByText('সমাধান হয়েছে')).toBeInTheDocument()
    expect(within(overview).getByText('আটকে আছে')).toBeInTheDocument()
  })

  // The B17 rule made visible: one response, split by status, so a report still
  // collecting votes sits in its own section rather than being invisible until it
  // crosses V. Each section is queried through its own named region because the
  // three headings rhyme with their own rows' status badges in Bangla.
  it('splits the one queue response into validating and assignment sections', async () => {
    renderWithProviders(<AdminHome />)

    const validating = await screen.findByRole('region', { name: /যাচাই চলছে/ })
    expect(await within(validating).findByText('রাস্তার বাতিগুলো জ্বলছে না')).toBeInTheDocument()
    expect(within(validating).queryByText(/ইউনিয়ন সড়কের সেতুটি/)).not.toBeInTheDocument()

    const assignment = await screen.findByRole('region', { name: /নির্ধারণের অপেক্ষায়/ })
    expect(await within(assignment).findByText(/ইউনিয়ন সড়কের সেতুটি/)).toBeInTheDocument()
    expect(within(assignment).queryByText('রাস্তার বাতিগুলো জ্বলছে না')).not.toBeInTheDocument()

    // One fetch feeds both sections — the whole reason they share a query key.
    expect(assignmentsApi.listQueue).toHaveBeenCalledTimes(1)
  })

  // B20: an above-union report is the super admin's to forward, so it arrives on a
  // separate endpoint and lands in the advice section — never in the assignment
  // queue, where the admin would be offered an action the backend refuses with
  // `wrong_route`. The two lists are also scoped differently (advice by upazila,
  // assignment by the admin's own union), so they are genuinely two fetches.
  it('puts above-union reports in the advice section, not the assignment queue', async () => {
    renderWithProviders(<AdminHome />)

    const advice = await screen.findByRole('region', { name: /পরামর্শ দিন/ })
    expect(await within(advice).findByText(/উপজেলা সড়কের সেতুটি/)).toBeInTheDocument()

    const assignment = await screen.findByRole('region', { name: /নির্ধারণের অপেক্ষায়/ })
    expect(within(assignment).queryByText(/উপজেলা সড়কের সেতুটি/)).not.toBeInTheDocument()

    // The row leads to the advise screen, not to an assign screen it cannot use.
    const link = within(advice).getByRole('link', { name: /উপজেলা সড়কের সেতুটি/ })
    expect(link).toHaveAttribute('href', '/admin/forwarding/prob-9')
  })

  // The count is the evidence the admin forwards a report on, so it has to be
  // legible: Bengali numerals, per A.5.3 rule 5. Asserting the absence of the
  // Latin digits is the half that catches a formatter silently doing nothing —
  // which is exactly how F18's first attempt passed the whole suite.
  it('shows each row’s validation progress in Bengali numerals', async () => {
    renderWithProviders(<AdminHome />)

    const validating = await screen.findByRole('region', { name: /যাচাই চলছে/ })
    const progress = await within(validating).findByText(/জন যাচাই করেছেন/)
    expect(progress).toHaveTextContent('৩ / ৫ জন যাচাই করেছেন')
    expect(progress.textContent).not.toMatch(/[0-9]/)
  })
})

describe('Moderation', () => {
  it('offers approve and reject on a report awaiting screening', async () => {
    renderWithProviders(<Moderation />)
    expect(await screen.findByText('বাজারের ড্রেনটি বন্ধ হয়ে আছে')).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'অনুমোদন করুন' }).length).toBeGreaterThan(0)
    expect(screen.getAllByRole('button', { name: 'প্রত্যাখ্যান' }).length).toBeGreaterThan(0)
  })
})
