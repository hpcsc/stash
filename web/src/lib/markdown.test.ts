import { describe, expect, it } from 'vitest'
import { markdownToPlainText } from './markdown'

describe('markdownToPlainText', () => {
  it('strips heading and emphasis syntax', () => {
    expect(markdownToPlainText('## test\n\n**bold** and *italic*')).toBe('test\nbold and italic')
  })

  it('renders a link as its text', () => {
    expect(markdownToPlainText('[google](https://google.com)')).toBe('google')
  })

  it('turns an image into a placeholder', () => {
    expect(markdownToPlainText('before ![](photo.png) after')).toBe('before 📷 after')
  })

  it('keeps list items on separate lines', () => {
    expect(markdownToPlainText('- one\n- two')).toBe('one\ntwo')
  })
})
