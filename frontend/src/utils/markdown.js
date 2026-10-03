import { marked } from 'marked'
import DOMPurify from 'dompurify'

export function renderMarkdown(source) {
  return DOMPurify.sanitize(marked.parse(source || ''), {
    USE_PROFILES: { html: true }, FORBID_TAGS: ['style', 'form'], FORBID_ATTR: ['style']
  })
}
