// 单源 markdown 渲染器. html:false —— 即使导入的容器塞 <script> 也不会渲染/执行 (安全,
// 无需额外 DOMPurify). linkify 自动识别裸 URL; breaks 让单换行也成 <br>.
import MarkdownIt from 'markdown-it'

const md = new MarkdownIt({ html: false, linkify: true, breaks: true })

export function renderMarkdown(src: string): string {
  return md.render(src ?? '')
}
