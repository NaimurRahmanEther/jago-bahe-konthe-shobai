import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import bn from './bn.json'

// Bangla is the only language. No fallbackLng: with one language it could only
// mask a missing key, and a raw key on screen is the failure we want to see.
//
// Numbers render in Bengali numerals because `lng` is 'bn' and the strings that
// carry them ask for i18next's built-in formatter — `{{count, number}}` in
// bn.json, which is Intl.NumberFormat('bn') underneath. No custom hook: an
// `interpolation.format` without a named format is never called on i18next 26,
// so it would sit there looking correct and doing nothing.
i18n.use(initReactI18next).init({
  resources: { bn: { translation: bn } },
  lng: 'bn',
  interpolation: { escapeValue: false },
  react: { useSuspense: true },
})

export default i18n
