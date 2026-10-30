import { locale } from './locale';

const languages = ['en', 'zh', 'ja'] as const;
const keys = Object.keys(locale.en).sort();
for (const language of languages) {
  const actual = Object.keys(locale[language]).sort();
  if (actual.join('\0') !== keys.join('\0')) {
    throw new Error(`i18n key mismatch: ${language}`);
  }
}
console.log(`i18n keys consistent: ${keys.length} keys across ${languages.length} languages`);
