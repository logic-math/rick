/**
 * 聊天 Markdown 渲染器（react-markdown 配置封装）。
 *
 * - remark-gfm：表格/任务列表/删除线
 * - rehype-highlight：代码高亮（atom-one-dark 配色）
 * - 未启用 rehype-raw：原始 HTML 以文本呈现（XSS 防线一）
 * - urlTransform 白名单（XSS 防线二）
 * - img 组件拦截外链图片（隐私 + 离线友好；XSS 防线三）
 * - a 组件：外链新标签打开（重定向语义，不破坏常驻对话）；本地文件路径 →
 *   右侧富文本阅读器（见 files/LocalAwareLink.tsx）
 */

import { memo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import { isExternalUrl, safeUrlTransform } from "./sanitize";
import { LocalAwareCode, LocalAwareLink } from "../files/LocalAwareLink";
import "highlight.js/styles/atom-one-dark.css";

interface MarkdownProps {
  children: string;
  className?: string;
}

/** 外链图片占位（不加载外部资源） */
function Img({ src, alt }: { src?: string; alt?: string }) {
  if (!src || isExternalUrl(src)) {
    return (
      <span className="my-1 inline-flex items-center gap-1 rounded border border-line bg-space px-2 py-0.5 text-xs text-ink-3">
        🖼 {alt || "外部图片已拦截"}
      </span>
    );
  }
  // 同源图片放行
  return <img src={src} alt={alt ?? ""} className="max-w-full rounded-md" loading="lazy" />;
}

/** Markdown 正文排版（prose 风格自实现——不引 typography 插件，保持轻量） */
const Markdown = memo(function Markdown({ children, className }: MarkdownProps) {
  return (
    <div
      className={`rick-md min-w-0 break-words text-sm leading-relaxed text-ink [&_a]:text-portal [&_a]:underline [&_a]:decoration-portal/40 hover:[&_a]:decoration-portal [&_blockquote]:border-l-2 [&_blockquote]:border-line [&_blockquote]:pl-3 [&_blockquote]:text-ink-2 [&_code]:rounded [&_code]:bg-space [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[0.85em] [&_h1]:mb-2 [&_h1]:mt-3 [&_h1]:text-base [&_h1]:font-semibold [&_h2]:mb-2 [&_h2]:mt-3 [&_h2]:text-[15px] [&_h2]:font-semibold [&_h3]:mb-1 [&_h3]:mt-2 [&_h3]:text-sm [&_h3]:font-semibold [&_hr]:my-3 [&_hr]:border-line [&_li]:my-0.5 [&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_p]:my-2 [&_pre]:my-2 [&_pre]:overflow-x-auto [&_pre]:rounded-md [&_pre]:bg-space [&_pre]:p-3 [&_pre]:font-mono [&_pre]:text-xs [&_strong]:font-semibold [&_table]:my-2 [&_table]:w-full [&_table]:border-collapse [&_td]:border [&_td]:border-line [&_td]:px-2 [&_td]:py-1 [&_th]:border [&_th]:border-line [&_th]:bg-space-2 [&_th]:px-2 [&_th]:py-1 [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5 ${className ?? ""}`}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        urlTransform={safeUrlTransform}
        components={{
          img: ({ src, alt }) => <Img src={typeof src === "string" ? src : undefined} alt={alt} />,
          // 重定向语义：外链新标签（不把常驻对话导航走）；本地文件 → 右侧阅读器
          a: ({ href, children }) => <LocalAwareLink href={typeof href === "string" ? href : undefined}>{children}</LocalAwareLink>,
          // 行内代码里的文件路径也可点击（agent 输出里的路径大多是反引号形式）
          code: ({ className, children, ...rest }) => {
            const isBlock = typeof className === "string" && className.includes("language-");
            if (isBlock) {
              return (
                <code className={className} {...rest}>
                  {children}
                </code>
              );
            }
            const text = String(children ?? "");
            return (
              <LocalAwareCode
                text={text}
                className="rounded bg-space px-1 py-0.5 font-mono text-[0.85em] text-portal"
              />
            );
          },
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
});

export default Markdown;
