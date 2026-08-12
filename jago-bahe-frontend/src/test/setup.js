// Test setup: jest-dom matchers + a synchronous i18n instance (Suspense off so
// components render immediately in tests) + automatic DOM cleanup between tests.
import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import bn from '../lib/i18n/bn.json'

// Mirrors lib/i18n/index.js: Bangla only, no fallback, and the same `lng: 'bn'`
// that makes {{count, number}} render Bengali numerals. Tests assert the real
// strings a user sees — including the digits — so this block must not drift from
// the app's init or the suite proves nothing.
i18n.use(initReactI18next).init({
  resources: { bn: { translation: bn } },
  lng: 'bn',
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
})

afterEach(() => cleanup())
