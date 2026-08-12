import { describe, it, expect } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import Section from './Section.jsx'

describe('Section', () => {
  // The reason this component exists. Pages used to title their cards with
  // `<p className="text-sm font-semibold">`, which is not a heading — so a page
  // built that way had no outline at all below its <h1>, and a screen-reader user
  // could not jump between its regions.
  it('titles the region with a real heading', () => {
    render(<Section title="কর্তৃপক্ষের পদক্ষেপ">সব ঠিক</Section>)
    expect(screen.getByRole('heading', { level: 2, name: 'কর্তৃপক্ষের পদক্ষেপ' })).toBeInTheDocument()
  })

  // Bangla section titles routinely repeat their own rows' status badges verbatim,
  // so the accessible name is what makes a region addressable at all — for assistive
  // tech and for the tests that have to scope their queries (A.5.3 rule 4).
  it('names the region by its heading so it can be scoped to', () => {
    render(<Section title="সম্পূর্ণ রেকর্ড">ভিতরের লেখা</Section>)
    const region = screen.getByRole('region', { name: 'সম্পূর্ণ রেকর্ড' })
    expect(within(region).getByText('ভিতরের লেখা')).toBeInTheDocument()
  })

  it('renders the meta string beside the title when given one', () => {
    render(
      <Section title="প্রস্তাবিত সমাধান" meta="৩টি">
        সব ঠিক
      </Section>,
    )
    // Scoped to the heading: the meta lives inside it, so an unscoped text query
    // would not prove where it landed.
    const heading = screen.getByRole('heading', { level: 2 })
    expect(within(heading).getByText('৩টি')).toBeInTheDocument()
  })

  it('omits the meta entirely when it is absent', () => {
    render(<Section title="যাচাই">সব ঠিক</Section>)
    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent('যাচাই')
  })

  it('renders an action slot beside the title', () => {
    render(
      <Section title="যাচাই" action={<a href="/problems">সব দেখুন</a>}>
        সব ঠিক
      </Section>,
    )
    expect(screen.getByRole('link', { name: 'সব দেখুন' })).toBeInTheDocument()
  })

  // boxed={false} is for content that draws its own container (ValidationVote's
  // card, AuditTrail's <details>) — double-boxing those was the visual noise the
  // whole redesign is removing.
  it('drops the card wrapper when boxed is false', () => {
    const { container } = render(
      <Section title="সম্পূর্ণ রেকর্ড" boxed={false}>
        <p>ভিতরের লেখা</p>
      </Section>,
    )
    expect(container.querySelector('.rounded-card')).toBeNull()
  })

  it('wraps content in a card by default', () => {
    const { container } = render(<Section title="যাচাই">ভিতরের লেখা</Section>)
    expect(container.querySelector('.rounded-card')).not.toBeNull()
  })
})
