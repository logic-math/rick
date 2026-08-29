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
 * - 相对路径：放行（同源资源）
 * - http(s)/mailto：放行（链接 a 可外跳；图片由 img 组件单独拦截）
 * - 其他协议（javascript:/data:/vbscript:…）：置 '#' 阻断
 */
export function safeUrlTransform(url: string): string {
  if (!url) return url;
  // 相对/锚点/同源
  if (url.startsWith("/") || url.startsWith("#") || url.startsWith("./") || url.startsWith("../")) {
    return url;
  }
  try {
    const u = new URL(url);
    if (u.protocol === "http:" || u.protocol === "https:" || u.protocol === "mailto:") {
      return url;
    }
    return "#";
  } catch {
    return "#";
  }
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
