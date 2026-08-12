import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Register from './Register.jsx'
import * as areasApi from '../../lib/api/areas.js'

// There is no mock layer (A.5.5): the api module IS the boundary and this stubs it.
// The fixture below is test data, not a shipped fake — it can only describe this file.
vi.mock('../../lib/api/areas.js', () => ({ listAreas: vi.fn() }))

const register = vi.fn()
vi.mock('../../auth/useAuth.js', () => ({ useAuth: () => ({ register }) }))

// The seat as migrations/000018_seed_dhamoirhat.up.sql seeds it: a root with a
// null parentId, and the local units the form must offer. The pourashava is a
// union-level node (see the migration's header), so it belongs in this select.
const AREAS = [
  { id: 'seat-naogaon-2', name: 'নওগাঁ-২', level: 'seat', parentId: null },
  { id: 'upazila-dhamoirhat', name: 'ধামইরহাট উপজেলা', level: 'upazila', parentId: 'seat-naogaon-2' },
  { id: 'union-dhamoirhat', name: 'ধামইরহাট ইউনিয়ন', level: 'union', parentId: 'upazila-dhamoirhat' },
  { id: 'pourashava-dhamoirhat', name: 'ধামইরহাট পৌরসভা', level: 'union', parentId: 'upazila-dhamoirhat' },
]

// "নিবন্ধন করুন" is BOTH the heading and the submit button (auth.register.title
// and auth.register.submit), so submit is scoped by role — A.5.3 rule 4.
const submit = () => screen.getByRole('button', { name: 'নিবন্ধন করুন' })
const unionField = () => screen.getByLabelText('ইউনিয়ন / পৌরসভা')

function renderRegister() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Register />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

async function fillEverythingElse(user) {
  await user.type(screen.getByLabelText('নাম'), 'রহিমা খাতুন')
  await user.type(screen.getByLabelText('মোবাইল নম্বর'), '01810000001')
  await user.type(screen.getByLabelText('জাতীয় পরিচয়পত্র নম্বর'), '1234567890111')
  await user.type(screen.getByLabelText('পাসওয়ার্ড'), 'resident123')
  await user.type(screen.getByLabelText('পাসওয়ার্ড নিশ্চিত করুন'), 'resident123')
}

beforeEach(() => {
  register.mockReset()
  register.mockResolvedValue(undefined)
  areasApi.listAreas.mockReset()
  areasApi.listAreas.mockResolvedValue(AREAS)
})

describe('Register', () => {
  // The point of F16: a resident used to be asked to TYPE their union ("যেমন union-1"),
  // so the first screen of the pilot required knowing an internal id.
  it('offers the seat’s unions by name and submits the chosen id', async () => {
    const user = userEvent.setup()
    renderRegister()

    const union = await screen.findByRole('option', { name: 'ধামইরহাট ইউনিয়ন' })
    expect(union).toBeInTheDocument()

    await fillEverythingElse(user)
    await user.selectOptions(unionField(), 'union-dhamoirhat')
    await user.click(submit())

    expect(register).toHaveBeenCalledWith(expect.objectContaining({ unionId: 'union-dhamoirhat' }))
  })

  // Union level only: offering the upazila or the seat here would let someone
  // register into a level the backend's union scoping does not accept.
  it('offers no upazila or seat as a union', async () => {
    renderRegister()

    await screen.findByRole('option', { name: 'ধামইরহাট ইউনিয়ন' })
    expect(screen.queryByRole('option', { name: 'ধামইরহাট উপজেলা' })).not.toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'নওগাঁ-২' })).not.toBeInTheDocument()
  })

  // The pourashava is seeded at level 'union' precisely so a resident of the
  // municipality has something to register into. Drop it from this select and
  // they cannot make an account at all.
  it('offers the pourashava alongside the unions', async () => {
    const user = userEvent.setup()
    renderRegister()

    expect(await screen.findByRole('option', { name: 'ধামইরহাট পৌরসভা' })).toBeInTheDocument()

    await fillEverythingElse(user)
    await user.selectOptions(unionField(), 'pourashava-dhamoirhat')
    await user.click(submit())

    expect(register).toHaveBeenCalledWith(expect.objectContaining({ unionId: 'pourashava-dhamoirhat' }))
  })

  // The load-bearing one. Without the geography there is no union to submit; an
  // unguarded form would post unionId:'' and the backend would answer 400
  // invalid_union — a cause the registrant cannot see or act on.
  it('refuses to submit when the geography cannot be loaded, and offers a retry', async () => {
    areasApi.listAreas.mockRejectedValue({ status: 500, message: 'boom' })
    const user = userEvent.setup()
    renderRegister()

    expect(
      await screen.findByText('এলাকার তালিকা আনা যায়নি। নিবন্ধন করতে হলে আবার চেষ্টা করুন।'),
    ).toBeInTheDocument()
    expect(submit()).toBeDisabled()
    expect(screen.getByRole('button', { name: 'আবার চেষ্টা করুন' })).toBeInTheDocument()

    await fillEverythingElse(user)
    await user.click(submit())
    expect(register).not.toHaveBeenCalled()
  })

  // A real account was created with its name set to "01540787241", so the profile
  // greeted its owner by phone number. The name field only checked for emptiness.
  it.each([['01540787241'], ['০১৫৪০৭৮৭২৪১'], ['+880 1540-787241']])(
    'refuses %s as a name — that is a phone number in the wrong box',
    async (typed) => {
      const user = userEvent.setup()
      renderRegister()
      await screen.findByRole('option', { name: 'ধামইরহাট ইউনিয়ন' })

      await user.type(screen.getByLabelText('নাম'), typed)
      await user.type(screen.getByLabelText('মোবাইল নম্বর'), '01810000001')
      await user.type(screen.getByLabelText('জাতীয় পরিচয়পত্র নম্বর'), '1234567890111')
      await user.type(screen.getByLabelText('পাসওয়ার্ড'), 'resident123')
      await user.type(screen.getByLabelText('পাসওয়ার্ড নিশ্চিত করুন'), 'resident123')
      await user.selectOptions(unionField(), 'union-dhamoirhat')
      await user.click(submit())

      expect(await screen.findByText('আপনার নাম লিখুন — মোবাইল নম্বর নয়')).toBeInTheDocument()
      expect(register).not.toHaveBeenCalled()
    },
  )

  it('accepts a Bengali name', async () => {
    const user = userEvent.setup()
    renderRegister()
    await screen.findByRole('option', { name: 'ধামইরহাট ইউনিয়ন' })

    await fillEverythingElse(user)
    await user.selectOptions(unionField(), 'union-dhamoirhat')
    await user.click(submit())

    expect(register).toHaveBeenCalledWith(expect.objectContaining({ name: 'রহিমা খাতুন' }))
  })

  it('keeps the union required — an unchosen select is not a submitted one', async () => {
    const user = userEvent.setup()
    renderRegister()

    await screen.findByRole('option', { name: 'ধামইরহাট ইউনিয়ন' })
    await fillEverythingElse(user)
    await user.click(submit())

    expect(await screen.findByText('এই তথ্যটি আবশ্যক')).toBeInTheDocument()
    expect(register).not.toHaveBeenCalled()
  })
})
