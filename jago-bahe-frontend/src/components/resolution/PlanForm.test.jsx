import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import PlanForm from './PlanForm.jsx'

// The plan must answer the community's top suggestion; stub the suggestions hook
// so a top suggestion is present.
vi.mock('../../hooks/useSuggestions.js', () => ({
  useSuggestions: () => ({
    data: [{ id: 's1', isTop: true, text: 'Build a culvert', upvoteCount: 4 }],
  }),
}))

describe('PlanForm', () => {
  it('surfaces the top suggestion the plan must respond to', () => {
    render(<PlanForm problemId="p1" onSubmit={vi.fn()} isSubmitting={false} />)
    expect(screen.getByText('সবচেয়ে সমর্থিত')).toBeInTheDocument()
    expect(screen.getByText('Build a culvert')).toBeInTheDocument()
  })

  it('blocks submit and shows errors when required fields are empty', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()
    render(<PlanForm problemId="p1" onSubmit={onSubmit} isSubmitting={false} />)
    await user.click(screen.getByRole('button', { name: 'পরিকল্পনা জমা দিন' }))
    expect(onSubmit).not.toHaveBeenCalled()
    expect(screen.getByText('শীর্ষ প্রস্তাবের জবাব দেওয়া আবশ্যক')).toBeInTheDocument()
    expect(screen.getByText('এই তথ্যটি আবশ্যক')).toBeInTheDocument()
  })

  it('refuses a plan with no weekly task, even when the other fields are filled', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()
    render(<PlanForm problemId="p1" onSubmit={onSubmit} isSubmitting={false} />)

    await user.type(screen.getByLabelText(/শীর্ষ প্রস্তাবের জবাব/), 'Adopting the culvert idea')
    await user.type(screen.getByLabelText('কৌশল'), 'Repair in two phases')
    // the single week is left blank
    await user.click(screen.getByRole('button', { name: 'পরিকল্পনা জমা দিন' }))

    expect(onSubmit).not.toHaveBeenCalled()
    expect(screen.getByText('অন্তত একটি সপ্তাহের কাজ দিন')).toBeInTheDocument()
  })

  it('adds and removes week rows', async () => {
    const user = userEvent.setup()
    render(<PlanForm problemId="p1" onSubmit={vi.fn()} isSubmitting={false} />)

    // one week to start
    expect(screen.getAllByPlaceholderText('এই সপ্তাহের কাজ')).toHaveLength(1)
    await user.click(screen.getByRole('button', { name: 'আরেকটি সপ্তাহ যোগ করুন' }))
    expect(screen.getAllByPlaceholderText('এই সপ্তাহের কাজ')).toHaveLength(2)
    // the second row can be removed again (both rows show a remove control now)
    const removeButtons = screen.getAllByRole('button', { name: /সরান/ })
    await user.click(removeButtons[removeButtons.length - 1])
    expect(screen.getAllByPlaceholderText('এই সপ্তাহের কাজ')).toHaveLength(1)
  })

  describe('revise mode (restart)', () => {
    const initial = {
      strategy: 'The old plan',
      obstacles: 'anticipated silt',
      suggestionResponse: 'Adopting the culvert idea',
      tasks: [{ task: 'first week work' }],
    }

    it('prefills from the current plan and requires a restart reason', async () => {
      const onSubmit = vi.fn()
      const user = userEvent.setup()
      render(<PlanForm problemId="p1" onSubmit={onSubmit} isSubmitting={false} mode="revise" initial={initial} />)

      // The old plan is seeded so the official edits rather than retypes.
      expect(screen.getByDisplayValue('The old plan')).toBeInTheDocument()
      expect(screen.getByDisplayValue('first week work')).toBeInTheDocument()

      // Submitting with no reason is blocked — the reason is the public "what changed".
      await user.click(screen.getByRole('button', { name: 'সংশোধিত পরিকল্পনা জমা দিন' }))
      expect(onSubmit).not.toHaveBeenCalled()
      expect(screen.getByText('এই তথ্যটি আবশ্যক')).toBeInTheDocument()
    })

    it('submits the reason alongside the plan once given', async () => {
      const onSubmit = vi.fn().mockResolvedValue({})
      const user = userEvent.setup()
      render(<PlanForm problemId="p1" onSubmit={onSubmit} isSubmitting={false} mode="revise" initial={initial} />)

      await user.type(screen.getByLabelText(/কেন পুনরায় শুরু করছেন/), 'the pipe kept flooding')
      await user.click(screen.getByRole('button', { name: 'সংশোধিত পরিকল্পনা জমা দিন' }))

      expect(onSubmit).toHaveBeenCalledTimes(1)
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({ reason: 'the pipe kept flooding', tasks: ['first week work'] }),
      )
    })
  })

  it('submits the week-by-week checklist, dropping blank weeks', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()
    render(<PlanForm problemId="p1" onSubmit={onSubmit} isSubmitting={false} />)

    await user.type(screen.getByLabelText(/শীর্ষ প্রস্তাবের জবাব/), 'Adopting the culvert idea')
    await user.type(screen.getByLabelText('কৌশল'), 'Repair in two phases')

    const week1 = screen.getByPlaceholderText('এই সপ্তাহের কাজ')
    await user.type(week1, 'Clear the drain')
    await user.click(screen.getByRole('button', { name: 'আরেকটি সপ্তাহ যোগ করুন' }))
    const inputs = screen.getAllByPlaceholderText('এই সপ্তাহের কাজ')
    await user.type(inputs[1], 'Lay the pipe')
    // a third, left blank, must be dropped
    await user.click(screen.getByRole('button', { name: 'আরেকটি সপ্তাহ যোগ করুন' }))

    await user.click(screen.getByRole('button', { name: 'পরিকল্পনা জমা দিন' }))

    expect(onSubmit).toHaveBeenCalledTimes(1)
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        strategy: 'Repair in two phases',
        suggestionResponse: 'Adopting the culvert idea',
        tasks: ['Clear the drain', 'Lay the pipe'],
      }),
    )
  })
})
