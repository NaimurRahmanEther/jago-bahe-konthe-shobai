import Navbar from './Navbar.jsx'

export default function AppShell({ children }) {
  return (
    <div className="flex min-h-svh flex-col bg-canvas text-ink">
      <Navbar />
      <main className="mx-auto w-full max-w-320 flex-1 px-4 py-6">{children}</main>
    </div>
  )
}
