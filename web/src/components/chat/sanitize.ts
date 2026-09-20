/**
 * 服务端字符串清洗（dompurify）。
 *
 * React 渲染默认已做 HTML 转义（react-markdown 未启用 rehype-raw——原始 HTML
 * 以文本呈现不执行）；此处再过一道 dompurify（ALLOWED_TAGS=[]——剥掉一切
 * 标签/危险实体）作为双保险，用于服务端提供的 title/message/options 等直显文本。
 */

import DOMPurify from "dompurify";

/** 剥掉一切 HTML 标签/实体，仅留纯文本 */
export function sanitizePlainText(s: string | undefined | null): string {
  if (!s) return "";
  try {
    return DOMPurify.sanitize(s, {
      ALLOWED_TAGS: [],
      ALLOWED_ATTR: [],
      KEEP_CONTENT: true,
    });
  } catch {
    return s;
  }
}

/**
 * Markdown URL 白名单变换（react-markdown urlTransform）：
 * - 无 scheme 的相对/绝对路径：放行（本地文件链接交给 a 组件就地打开阅读器）
 * - http(s)/mailto/file：放行（外链新标签、file: 就地打开——点击语义见
 *   lib/localPath.ts 的 classifyHref）
 * - 其他协议（javascript:/data:/vbscript:…）：置 '#' 阻断
 *
 * 注意：旧实现只放行 `./` `/` `#` 前缀，裸相对路径（`plan/task1.md`）会走
 * new URL() 抛错分支 → 变成 '#'，**本地 md 链接全被吃掉了**。
 */
export function safeUrlTransform(url: string): string {
  if (!url) return url;
  if (url.startsWith("#")) return url;
  const scheme = /^([a-z][a-z0-9+.-]*):/i.exec(url);
  if (!scheme) return url; // 相对/绝对路径：放行
  const proto = scheme[1].toLowerCase();
  if (proto === "http" || proto === "https" || proto === "mailto" || proto === "file") {
    return url;
  }
  return "#";
}

/** 是否外链（非相对路径）——外链图片拦截用 */
export function isExternalUrl(url: string): boolean {
  if (!url) return false;
  if (url.startsWith("/") || url.startsWith("#") || url.startsWith("data:")) return false;
  try {
    const u = new URL(url);
    return u.origin !== window.location.origin;
  } catch {
    return false;
  }
}
