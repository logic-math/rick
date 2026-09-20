/**
 * LocalAwareLink / LocalAwareCode：markdown 里的「重定向语义」链接。
 *
 * 规则（用户要求：不要在对话页里被导航走）：
 * - **外链**（http/https/mailto）→ `target="_blank" rel="noopener noreferrer"`，
 *   新标签打开，常驻对话页原地不动。
 * - **本地文件**（相对路径 / 绝对路径 / file://）→ 不做浏览器导航，调用
 *   FileReaderContext.open()，在右侧富文本阅读器里打开（与 Dreams 页同一渲染）。
 *   没有 Provider 时退化为「复制路径」按钮——宁可不可点，也不把当前视图导航走。
 * - **同源页面路由**（/ws/…、/session/…）→ 正常 SPA 导航（这是应用内跳转）。
 * - **页内锚点** → 默认行为。
 *
 * 行内代码里的路径（`plan/task1.md`）也复用同一判定——agent 输出里绝大多数
 * 文件路径其实写在反引号里，不是 markdown 链接（用户实测：链接根本点不到）。
 */

import type { ReactNode } from "react";
import { classifyHref, inlineCodePath } from "../../lib/localPath";
import { useFileReader } from "../files/FileReaderContext";

/** 本地文件链接的统一样式（与普通链接一致，加虚线提示可就地打开） */
const LINK_CLASS =
  "text-portal underline decoration-portal/40 underline-offset-2 hover:decoration-portal";

export function LocalAwareLink({
  href,
  children,
  className,
}: {
  href?: string;
  children?: ReactNode;
  className?: string;
}) {
  const reader = useFileReader();
  const info = classifyHref(href);
  const cls = className ?? LINK_CLASS;

  if (info.kind === "local") {
    const path = info.path;
    if (reader) {
      return (
        <button
          type="button"
          onClick={() => reader.open(path)}
          title={`在右侧阅读器打开：${path}`}
          className={`${cls} cursor-pointer text-left decoration-dotted`}
        >
          {children}
        </button>
      );
    }
    // 无阅读器可用（理论上只有知识库/Jobs 之外的嵌入式视图）→ 只复制路径，
    // **绝不**让浏览器导航走（用户要求：任何时候都不破坏当前常驻视图）。
    return (
      <button
        type="button"
        onClick={() => void navigator.clipboard?.writeText(path).catch(() => {})}
        title={`复制路径：${path}`}
        className={`${cls} cursor-pointer decoration-dotted`}
      >
        {children}
      </button>
    );
  }

  // 外链/其他：重定向语义——新标签，不破坏当前对话
  if (info.kind === "external") {
    return (
      <a
        href={info.path}
        target="_blank"
        rel="noopener noreferrer"
        title={`新标签打开：${info.path}`}
        className={cls}
      >
        {children}
      </a>
    );
  }

  return (
    <a href={href} className={cls}>
      {children}
    </a>
  );
}

/**
 * 行内代码：看起来是本地可读文件路径时变成可点击链接，否则原样渲染。
 */
export function LocalAwareCode({
  text,
  className,
}: {
  text: string;
  className: string;
}) {
  const reader = useFileReader();
  const path = inlineCodePath(text);
  if (!path || !reader) {
    return <code className={className}>{text}</code>;
  }
  return (
    <button
      type="button"
      onClick={() => reader.open(path)}
      title={`在右侧阅读器打开：${path}`}
      className={`${className} cursor-pointer decoration-dotted underline decoration-portal/40 hover:decoration-portal`}
    >
      {text}
    </button>
  );
}
