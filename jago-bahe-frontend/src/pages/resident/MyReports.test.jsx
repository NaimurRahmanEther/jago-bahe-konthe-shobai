import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import MyReports from './MyReports.jsx'
import * as problemsApi from '../../lib/api/problems.js'
import * as resolutionApi from '../../lib/api/resolution.js'

// There is no mock layer any more (CLAUDE.md A.5.5): the api module IS the
// boundary, and these tests stub it. The fixture below is test data — a mock was
// shipped app code pretending to be a backend, which is how it came to lie about
// one (A.5.2); the same JSON living here can only ever describe this test.
//
// Every export must be listed: useProblems.js and useCases.js both `import * as`,
// so an omitted name is not an import error — it is `undefined` at call time.
vi.mock('../../lib/api/problems.js', () => ({
  listProblems: vi.fn(),
  getProblem: vi.fn(),
  reportProblem: vi.fn(),
  listMyProblems: vi.fn(),
  validateProblem: vi.fn(),
}))

vi.mock('../../lib/api/resolution.js', () => ({
  listCases: vi.fn(),
  getProgress: vi.fn(),
  getCase: vi.fn(),
  acknowledgeCase: vi.fn(),
  submitPlan: vi.fn(),
  postUpdate: vi.fn(),
  uploadEvidence: vi.fn(),
  markDone: vi.fn(),
  reportObstacle: vi.fn(),
  confirmProblem: vi.fn(),
}))

// The page reads the signed-in resident from auth; stub it so these tests are
// about the page, not the login flow.
const mockUser = { id: 'res-1', name: 'Rahima Khatun', role: 'resident' }
vi.mock('../../auth/useAuth.js', () => ({ useAuth: () => ({ user: mockUser, role: 'resident' }) }))
vi.mock('../../hooks/useOfficials.js', () => ({
  useOfficials: () => ({ data: [{ id: 'off-1', name: 'Karim Uddin', tier: 'ward_member', areaId: 'ward-1' }] }),
}))

// res-1's reports, spread across the lifecycle so every branch of the page has a
// row: Reported (approved, awaiting validation), Assigned-with-no-plan (the lazy
// -case branch), Blocked-with-a-plan, PendingApproval (awaiting the admin's
// screening — visible to its author here, nowhere else), and Rejected.
const MY_PROBLEMS = [
  {
    id: 'prob-1',
    title: 'রাস্তায় বড় গর্ত হয়ে গেছে',
    description: 'ওয়ার্ড ১ এর মূল সড়কে বৃষ্টির পর বড় গর্ত হয়েছে।',
    location: { areaId: 'ward-1', address: 'মূল সড়ক, ওয়ার্ড ১' },
    reporterId: 'res-1',
    pointedOfficialId: 'off-1',
    status: 'Reported',
    validCount: 2,
    validationThreshold: 5,
    createdAt: '2026-06-20T09:00:00.000Z',
  },
  {
    id: 'prob-2',
    title: 'পানির লাইনে লিকেজ',
    description: 'ওয়ার্ড ২ এর সরবরাহ লাইনে লিকেজের কারণে পানি অপচয় হচ্ছে।',
    location: { areaId: 'ward-2', address: 'স্কুল রোড, ওয়ার্ড ২' },
    reporterId: 'res-1',
    pointedOfficialId: 'off-1',
    status: 'Assigned',
    validCount: 5,
    validationThreshold: 5,
    createdAt: '2026-06-15T09:00:00.000Z',
  },
  {
    id: 'prob-4',
    title: 'কালভার্টটি ভেঙে পড়েছে',
    description: 'ওয়ার্ড ২ এর খাল পাড়ের কালভার্টটি ভেঙে পড়ায় চলাচল বন্ধ হয়ে গেছে।',
    location: { areaId: 'ward-2', address: 'খাল পাড়, ওয়ার্ড ২' },
    reporterId: 'res-1',
    pointedOfficialId: 'off-1',
    status: 'Blocked',
    validCount: 5,
    validationThreshold: 5,
    createdAt: '2026-06-01T09:00:00.000Z',
  },
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
  {
    id: 'prob-6',
    title: 'পরীক্ষামূলক পোস্ট',
    description: 'এটি একটি অপ্রাসঙ্গিক পরীক্ষামূলক পোস্ট ছিল।',
    location: { areaId: 'ward-2', address: 'ওয়ার্ড ২' },
    reporterId: 'res-1',
    pointedOfficialId: 'off-1',
    status: 'Rejected',
    rejectionReason: 'spam',
    reviewedBy: 'user-admin-1',
    reviewedAt: '2026-07-10T10:00:00.000Z',
    validCount: 0,
    validationThreshold: 5,
    createdAt: '2026-07-09T09:00:00.000Z',
  },
]

// case-1 (prob-2): assigned, never opened — the null-plan branch.
const PROGRESS_NO_PLAN = {
  caseId: 'case-1',
  problemId: 'prob-2',
  status: 'Assigned',
  createdAt: '2026-06-16T09:00:00.000Z',
  deadline: '2026-07-10T09:00:00.000Z',
  acknowledged: false,
  acknowledgedAt: null,
  plan: null,
  updates: [],
  evidence: [],
  blockedOnHigherAuthority: false,
}

// case-2 (prob-4): blocked, with a plan. answeredSuggestion is the SNAPSHOT the
// API returns — sug-4, the top when the plan was written — while sug-5 has since
// overtaken it on upvotes. blockedOnHigherAuthority is false because the seeded
// obstacle is declared but not adjudicated.
const PROGRESS_WITH_PLAN = {
  caseId: 'case-2',
  problemId: 'prob-4',
  status: 'Blocked',
  createdAt: '2026-06-05T09:00:00.000Z',
  deadline: '2026-06-20T09:00:00.000Z',
  acknowledged: true,
  acknowledgedAt: '2026-06-05T09:00:00.000Z',
  plan: {
    id: 'plan-case-2',
    strategy: 'অস্থায়ী সাঁকো বসিয়ে চলাচল চালু রাখা এবং স্থায়ী কালভার্ট নির্মাণের বরাদ্দের চেষ্টা।',
    timelineWeeks: 6,
    obstacles: 'বরাদ্দ প্রাপ্তি অনিশ্চিত।',
    suggestionResponse: 'সম্প্রদায়ের প্রস্তাব অনুযায়ী অস্থায়ী ব্যবস্থা নেওয়া হচ্ছে।',
    answeredSuggestion: 'আপাতত একটি অস্থায়ী সাঁকো বসানো হোক।',
    createdAt: '2026-06-06T09:00:00.000Z',
  },
  updates: [
    {
      id: 'update-case-2-seed-1',
      caseId: 'case-2',
      kind: 'obstacle',
      text: 'কালভার্ট মেরামতের জন্য বরাদ্দ নেই।',
      createdAt: '2026-06-18T09:00:00.000Z',
    },
  ],
  evidence: [],
  blockedOnHigherAuthority: false,
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MyReports />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockUser.id = 'res-1'
  mockUser.name = 'Rahima Khatun'
  vi.mocked(problemsApi.listMyProblems).mockResolvedValue(MY_PROBLEMS)
  vi.mocked(resolutionApi.getProgress).mockImplementation((problemId) => {
    if (problemId === 'prob-2') return Promise.resolve(PROGRESS_NO_PLAN)
    if (problemId === 'prob-4') return Promise.resolve(PROGRESS_WITH_PLAN)
    // Null, never a rejection — the backend's null-shaped rule (A.3.2.3).
    return Promise.resolve(null)
  })
})

describe('MyReports', () => {
  it('greets the resident and lists their own reports', async () => {
    renderPage()
    expect(screen.getByText('স্বাগতম, Rahima Khatun')).toBeInTheDocument()
    expect(await screen.findByText('রাস্তায় বড় গর্ত হয়ে গেছে')).toBeInTheDocument()
  })

  // A report awaiting the admin's screening is visible to its author here and
  // nowhere else — it is not on the public feed until approved. So /me is the one
  // place its reporter can follow it while it waits.
  it('shows the reporter their own pending report', async () => {
    renderPage()
    // Scope to the card's anchor, not `closest('div')`: the whole card is one
    // link, so the anchor is the row's real boundary. A div is whatever the
    // card's internal layout happens to nest today.
    const pendingRow = (await screen.findByText('বাজারের ড্রেনটি বন্ধ হয়ে আছে')).closest('a')
    expect(within(pendingRow).getByText('অনুমোদনের অপেক্ষায়')).toBeInTheDocument()
  })

  // A takedown is accountable, not a disappearance: the report stays on the
  // reporter's page carrying the ground it was removed on. This is what makes the
  // admin's one power over a report survivable — a wrongly-buried report has a
  // witness, starting with the person who filed it.
  it('shows a rejected report with the ground it was taken down on', async () => {
    renderPage()
    const rejectedRow = (await screen.findByText('পরীক্ষামূলক পোস্ট')).closest('a')
    expect(within(rejectedRow).getByText('প্রত্যাখ্যাত')).toBeInTheDocument()
    // The ground renders as "কারণ: স্প্যাম" across text nodes, so match the node.
    expect(screen.getByText(/স্প্যাম/)).toBeInTheDocument()
  })

  it('expands a planned case to the plan and the top-suggestion pairing', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('কালভার্টটি ভেঙে পড়েছে')

    // prob-4 (Blocked) has a plan; prob-2 (Assigned) does not.
    const toggles = screen.getAllByRole('button', { name: 'অগ্রগতি দেখুন' })
    await user.click(toggles[1])

    expect(await screen.findByText('কর্মকর্তার পরিকল্পনা')).toBeInTheDocument()
    // The SNAPSHOT the API returned — sug-4, the top *when the plan was written* —
    // not sug-5, which has more upvotes today. Re-ranking here would show the
    // official answering a question they were never asked.
    expect(await screen.findByText('আপাতত একটি অস্থায়ী সাঁকো বসানো হোক।')).toBeInTheDocument()
    expect(screen.queryByText('স্থায়ী কালভার্ট নির্মাণের বরাদ্দ চাওয়া হোক।')).not.toBeInTheDocument()
  })

  // The seeded obstacle is declared but not yet adjudicated, so the official gets
  // no protection from it: only an obstacle the named authority judged *real*
  // earns the fairness note. Declaring one must never be enough on its own, or
  // any official could stop their own clock by saying so.
  it('withholds the fairness note while a obstacle is only declared, not adjudicated', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('কালভার্টটি ভেঙে পড়েছে')

    const toggles = screen.getAllByRole('button', { name: 'অগ্রগতি দেখুন' })
    await user.click(toggles[1])

    await screen.findByText('কর্মকর্তার পরিকল্পনা')
    expect(screen.queryByText(/ব্যর্থতা হিসেবে গণ্য হবে না/)).not.toBeInTheDocument()
  })

  // A case with no plan is not an empty case. The panel used to bail out of the
  // whole component on a null plan, so an assigned-but-unplanned case rendered one
  // flat line and hid what WAS known — that the case exists, and whether the
  // official has opened it. Silence is itself a fact worth publishing.
  it('expands an assigned-but-unplanned case to a calm "no plan yet", not an error', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('পানির লাইনে লিকেজ')

    const toggles = screen.getAllByRole('button', { name: 'অগ্রগতি দেখুন' })
    await user.click(toggles[0]) // prob-2 — case-1 has plan: null
    expect(await screen.findByText('কর্মকর্তা এখনো কোনো পরিকল্পনা প্রকাশ করেননি।')).toBeInTheDocument()
    // The unacknowledged state still shows: this case has NOT been opened, and
    // that is the most informative thing about it.
    expect(screen.getByText('কর্মকর্তা এখনো দায়িত্ব বুঝে নেননি')).toBeInTheDocument()
    // Not an error, and not the never-assigned copy either.
    expect(screen.queryByText('কর্মকর্তা এখনো কাজটি শুরু করেননি।')).toBeNull()
  })

  it('invites a resident with no reports to file one', async () => {
    vi.mocked(problemsApi.listMyProblems).mockResolvedValue([])
    renderPage()
    expect(await screen.findByText('আপনি এখনো কোনো সমস্যা জানাননি')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'একটি সমস্যা জানান' })).toHaveAttribute('href', '/report')
  })

  // The page had no error-state test PRECISELY because the old mock never
  // rejected — so this branch went red the moment real HTTP started failing.
  // Stubbing the api module is what makes a refusal expressible at all (A.5 rule 4).
  it('offers a retry when the reports cannot be loaded', async () => {
    const user = userEvent.setup()
    vi.mocked(problemsApi.listMyProblems).mockRejectedValue(new Error('network down'))
    renderPage()

    expect(await screen.findByText('আপনার প্রতিবেদন লোড করা যায়নি।')).toBeInTheDocument()

    vi.mocked(problemsApi.listMyProblems).mockResolvedValue(MY_PROBLEMS)
    await user.click(screen.getByRole('button', { name: 'আবার চেষ্টা করুন' }))
    expect(await screen.findByText('রাস্তায় বড় গর্ত হয়ে গেছে')).toBeInTheDocument()
  })

  it('shows a skeleton while the reports are in flight', async () => {
    // Never resolves: the page stays in its loading branch for the assertion.
    vi.mocked(problemsApi.listMyProblems).mockReturnValue(new Promise(() => {}))
    renderPage()

    expect(screen.getByRole('status', { name: 'লোড হচ্ছে...' })).toBeInTheDocument()
    expect(screen.queryByText('আপনি এখনো কোনো সমস্যা জানাননি')).not.toBeInTheDocument()
  })
})

describe('MyReports date filter', () => {
  // A typed range rather than a preset chip: the presets are relative to "today",
  // so a fixture pinned to June/July 2026 would pass or fail on the calendar.
  // This asserts the filtering itself, which is the part that can break.
  it('narrows the list to the reports filed in the chosen span', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('রাস্তায় বড় গর্ত হয়ে গেছে')

    await user.type(screen.getByLabelText('শুরুর তারিখ'), '2026-07-01')
    await user.type(screen.getByLabelText('শেষ তারিখ'), '2026-07-31')

    // Filed in July: prob-5 (14 Jul) and prob-6 (9 Jul).
    expect(screen.getByText('বাজারের ড্রেনটি বন্ধ হয়ে আছে')).toBeInTheDocument()
    expect(screen.getByText('পরীক্ষামূলক পোস্ট')).toBeInTheDocument()
    // Filed in June: gone.
    expect(screen.queryByText('রাস্তায় বড় গর্ত হয়ে গেছে')).toBeNull()
    expect(screen.queryByText('কালভার্টটি ভেঙে পড়েছে')).toBeNull()
  })

  // The tiles are a summary of everything this resident ever filed. Making them
  // follow the filter would turn "মোট" into a second copy of the row count sitting
  // directly beneath it, and would quietly restate five reports as two.
  it('leaves the stat tiles counting all time', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('রাস্তায় বড় গর্ত হয়ে গেছে')

    await user.type(screen.getByLabelText('শুরুর তারিখ'), '2026-07-01')

    const tiles = await screen.findByRole('region', { name: 'সারসংক্ষেপ' })
    expect(within(tiles).getByText('৫')).toBeInTheDocument()
    expect(screen.getByText('২ / ৫ দেখানো হচ্ছে')).toBeInTheDocument()
  })

  // An empty span is a different fact from an empty account, and saying the
  // second when the first is true reads as a page that has lost the reports.
  it('says the span is empty rather than "you have never reported anything"', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('রাস্তায় বড় গর্ত হয়ে গেছে')

    await user.type(screen.getByLabelText('শুরুর তারিখ'), '2026-01-01')
    await user.type(screen.getByLabelText('শেষ তারিখ'), '2026-01-31')

    expect(await screen.findByText('এই সময়ের মধ্যে কিছু নেই')).toBeInTheDocument()
    expect(screen.queryByText('আপনি এখনো কোনো সমস্যা জানাননি')).toBeNull()
  })
})

describe('MyReports stats', () => {
  it('counts states without grading anyone', async () => {
    renderPage()
    await screen.findByText('রাস্তায় বড় গর্ত হয়ে গেছে')
    // Scoped to the tile region: "কাজ চলছে" is both a tile label and the
    // InProgress badge, and res-1's fixture will eventually seed one (A.5.3 rule
    // 4). Never getAllByText(...)[0] — that passes or fails on DOM order.
    const tiles = await screen.findByRole('region', { name: 'সারসংক্ষেপ' })
    expect(within(tiles).getByText('মোট')).toBeInTheDocument()
    // "অনুমোদনের অপেক্ষায়" is both this tile's label and the PendingApproval badge
    // on prob-5's row (A.5.3 rule 4), so scope to the region. getByText would throw
    // on the two matches otherwise.
    expect(within(tiles).getByText('অনুমোদনের অপেক্ষায়')).toBeInTheDocument()
    expect(within(tiles).getByText('কাজ চলছে')).toBeInTheDocument()
    expect(within(tiles).getByText('সমাধান হয়েছে')).toBeInTheDocument()
  })
})
