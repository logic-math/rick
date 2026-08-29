/**
 * Markdown/日志渲染视图（react-markdown 同 chat 栈）。
 *
 * - markdown 后缀（.md/.markdown）：GFM + 代码高亮（rehype-highlight + 自带精简
 *   hljs 主题——common/hljs-theme.css，零外链）
 * - 其他（.log/.json/.txt/yaml…）：等宽 pre 块原样渲染（日志密度优先）
 * - XSS：react-markdown 默认不渲染原始 HTML（无 rehype-raw）——安全基线
 * - 超限（后端 400 file_too_large）：由调用方先行判断，本组件只管渲染
 */

import { memo, type ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import "../common/hljs-theme.css";

export function isMarkdownPath(path: string): boolean {
  return /\.(md|markdown)$/i.test(path);
}

interface MarkdownViewProps {
  path: string;
  content: string;
}

/** GFM markdown 渲染（knowledge/job md 文件） */
export const MarkdownView = memo(function MarkdownView({ content }: { content: string }) {
  return (
    <div className="markdown-body min-w-0 text-sm leading-relaxed text-ink">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        components={{
          h1: (props) => <h1 className="mb-3 mt-5 border-b border-line pb-1.5 text-lg font-semibold text-ink first:mt-0" {...props} />,
          h2: (props) => <h2 className="mb-2 mt-5 border-b border-line/60 pb-1 text-base font-semibold text-ink" {...props} />,
          h3: (props) => <h3 className="mb-2 mt-4 text-sm font-semibold text-ink" {...props} />,
          p: (props) => <p className="my-2 text-sm leading-relaxed" {...props} />,
          a: (props) => <a className="text-portal underline decoration-portal/40 underline-offset-2 hover:decoration-portal" {...props} />,
          ul: (props) => <ul className="my-2 list-disc space-y-1 pl-5 text-sm" {...props} />,
          ol: (props) => <ol className="my-2 list-decimal space-y-1 pl-5 text-sm" {...props} />,
          li: (props) => <li className="min-w-0 leading-relaxed" {...props} />,
          blockquote: (props) => (
            <blockquote className="my-2 border-l-2 border-nebula/60 bg-white/[0.02] px-3 py-1.5 text-ink-2" {...props} />
          ),
          code: ({ className, children, ...rest }) => {
            const isBlock = typeof className === "string" && className.includes("language-");
            if (isBlock) {
              return <code className={`${className} font-mono text-xs`} {...rest}>{children}</code>;
            }
            return (
              <code className="rounded bg-portal-soft px-1 py-0.5 font-mono text-[0.8em] text-portal" {...rest}>
                {children}
              </code>
            );
          },
          pre: (props) => (
            <pre className="my-3 overflow-x-auto rounded-lg border border-line bg-space p-3 font-mono text-xs leading-relaxed" {...props} />
          ),
          table: (props) => (
            <div className="my-3 overflow-x-auto">
              <table className="w-full border-collapse text-xs" {...props} />
            </div>
          ),
          th: (props) => (
            <th className="border-b border-line px-2.5 py-1.5 text-left font-semibold text-ink" {...props} />
          ),
          td: (props) => <td className="border-b border-line/40 px-2.5 py-1.5 align-top text-ink-2" {...props} />,
          hr: () => <hr className="my-4 border-line" />,
          img: (props) => (
            <img className="my-3 max-w-full rounded-lg border border-line" loading="lazy" {...props} />
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
});

/** 等宽原始文本（.log/.json/yaml——日志密度优先） */
export const RawTextView = memo(function RawTextView({ content }: { content: string }) {
  return (
    <pre className="overflow-x-auto whitespace-pre-wrap break-words rounded-lg border border-line bg-space p-3 font-mono text-xs leading-relaxed text-ink-2">
      {content}
    </pre>
  );
});

/** 按路径后缀分派（默认 markdown） */
export default function FileContentView({ path, content }: MarkdownViewProps): ReactNode {
  if (isMarkdownPath(path)) return <MarkdownView content={content} />;
  return <RawTextView content={content} />;
}
