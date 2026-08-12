import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import LocationPicker from './LocationPicker.jsx'
import * as areasApi from '../../lib/api/areas.js'

vi.mock('../../lib/api/areas.js', () => ({ listAreas: vi.fn() }))

// The seat as migrations/000018_seed_dhamoirhat.up.sql seeds it, trimmed to what
// this file needs: two of Dhamoirhat's eight unions, the pourashava (which is a
// union-level node — see the migration's header), and the upazila above them, kept
// here to prove it is filtered out.
const AREAS = [
  { id: 'upazila-dhamoirhat', name: 'ধামইরহাট উপজেলা', level: 'upazila', parentId: 'seat-naogaon-2' },
  { id: 'union-dhamoirhat', name: 'ধামইরহাট ইউনিয়ন', level: 'union', parentId: 'upazila-dhamoirhat' },
  { id: 'union-agradigun', name: 'আগ্রাদ্বিগুন ইউনিয়ন', level: 'union', parentId: 'upazila-dhamoirhat' },
  { id: 'pourashava-dhamoirhat', name: 'ধামইরহাট পৌরসভা', level: 'union', parentId: 'upazila-dhamoirhat' },
]

function renderPicker() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <LocationPicker value={{ areaId: '', address: '' }} onChange={() => {}} />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  areasApi.listAreas.mockReset()
  areasApi.listAreas.mockResolvedValue(AREAS)
})

describe('LocationPicker', () => {
  it('renders the unions as buttons, and nothing above union level', async () => {
    renderPicker()

    expect(await screen.findByRole('button', { name: 'ধামইরহাট ইউনিয়ন' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'আগ্রাদ্বিগুন ইউনিয়ন' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'ধামইরহাট উপজেলা' })).not.toBeInTheDocument()
  })

  // The pourashava is seeded at level 'union', so it must appear beside the unions.
  // If it ever stops appearing, a resident of the municipality has nowhere to report.
  it('offers the pourashava alongside the unions', async () => {
    renderPicker()

    expect(await screen.findByRole('button', { name: 'ধামইরহাট পৌরসভা' })).toBeInTheDocument()
  })

  it('shows a placeholder, not area buttons, while the geography loads', () => {
    areasApi.listAreas.mockReturnValue(new Promise(() => {}))
    renderPicker()

    // Nothing is pressable before the seat's geography has actually arrived —
    // the alternative is a row that looks ready and answers to nothing.
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
    expect(screen.getByText('অবস্থান')).toBeInTheDocument()
  })

  it('shows the cause and a retry when the geography fails', async () => {
    areasApi.listAreas.mockRejectedValue({ status: 500, message: 'boom' })
    renderPicker()

    expect(await screen.findByText('এলাকার তালিকা আনা যায়নি।')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'আবার চেষ্টা করুন' })).toBeInTheDocument()
  })

  // The branch that would otherwise be a silent void: a label above an empty row,
  // with nothing to tell the reporter why there is nothing to press.
  it('says so when there are no areas, rather than rendering an empty row', async () => {
    areasApi.listAreas.mockResolvedValue([])
    renderPicker()

    expect(await screen.findByText('এলাকার তালিকা এখন পাওয়া যাচ্ছে না।')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })
})
