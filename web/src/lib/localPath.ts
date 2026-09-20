/**
 * 本地文件路径识别（聊天/知识库里的链接与行内代码）。
 *
 * 两个用途：
 * 1. **重定向语义**：http(s) 外链一律新标签打开——常驻对话页不能被导航走
 *    （用户实测：点了会话里的链接就离开了对话，回来要重新找到那个会话）。
 * 2. **本地 md 就地查阅**：聊天里指向本机文件的路径（`.rick/jobs/x/plan/t.md`、
 *    绝对路径、file:// URL）点击后在右侧阅读器里打开，跟 Dreams 页的
 *    markdown 视图复用同一组件。
 */

/** 可以就地打开的文件后缀（文本类；非文本文件不走阅读器） */
const READABLE_EXT = new Set([
  "md",
  "markdown",
  "txt",
  "log",
  "json",
  "yaml",
  "yml",
  "toml",
  "csv",
  "py",
  "go",
  "ts",
  "tsx",
  "js",
  "sh",
  "sql",
  "ini",
  "conf",
  "env",
]);

/** 路径里的最后一段后缀（小写，不含点） */
export function extOf(path: string): string {
  const clean = path.split(/[#?]/)[0];
  const base = clean.slice(clean.lastIndexOf("/") + 1);
  const dot = base.lastIndexOf(".");
  return dot <= 0 ? "" : base.slice(dot + 1).toLowerCase();
}

/** 是否可读文本文件（阅读器支持的后缀） */
export function isReadableFile(path: string): boolean {
  return READABLE_EXT.has(extOf(path));
}

export interface HrefInfo {
  kind: "external" | "local" | "anchor" | "other";
  /** local 时的原始路径（去 file:// 前缀） */
  path: string;
}

/** 判断链接目标：外链（新标签）/ 本地文件（右侧阅读器）/ 页内锚点 */
export function classifyHref(href: string | undefined): HrefInfo {
  const raw = (href ?? "").trim();
  if (!raw) return { kind: "other", path: "" };
  if (raw.startsWith("#")) return { kind: "anchor", path: "" };
  if (raw.startsWith("file://")) {
    let p = raw.slice("file://".length);
    try {
      p = decodeURIComponent(p);
    } catch {
      /* 保留原串 */
    }
    return { kind: "local", path: p };
  }
  // 有 scheme：http/https/mailto 外链；其余（javascript: 等）交给 urlTransform 拦
  const scheme = /^([a-z][a-z0-9+.-]*):/i.exec(raw);
  if (scheme) {
    const proto = scheme[1].toLowerCase();
    if (proto === "http" || proto === "https" || proto === "mailto") {
      return { kind: "external", path: raw };
    }
    return { kind: "other", path: raw };
  }
  // 无 scheme：绝对路径 / 相对路径 / 同源页面路由
  // 同源页面路由（/ws/x/jobs、/session/x、/settings、/）交给 SPA 路由处理
  if (/^\/(ws|session|settings)(\/|$)/.test(raw)) return { kind: "other", path: raw };
  if (raw.startsWith("/") || raw.startsWith("./") || raw.startsWith("../")) {
    return { kind: "local", path: raw };
  }
  // 裸相对路径：看起来像文件的才当本地文件（避免把 `foo/bar` 目录链接当文件）
  if (raw.includes("/") || isReadableFile(raw)) return { kind: "local", path: raw };
  return { kind: "other", path: raw };
}

/**
 * 行内代码里的路径是否该变成可点击的本地文件链接。
 * 只认「像路径 + 可读后缀」，避免把 `npm run build`、`main.go:42` 等都变成链接。
 * 支持 `path.md:12` 这种带行号的写法（行号不参与读取）。
 */
export function inlineCodePath(text: string): string | null {
  const t = text.trim();
  if (!t || t.length > 400) return null;
  if (/[\s<>"'`|]/.test(t)) return null; // 带空格/引号的不是路径
  // 去尾部标点（中文句子里的 `x.md`。 之类）
  const trimmed = t.replace(/[),.;:，。；：）]+$/, "");
  const noLine = trimmed.replace(/:\d+(-\d+)?$/, "");
  if (!noLine) return null;
  if (!isReadableFile(noLine)) return null;
  // 必须至少有一段路径结构，或本身就是文件名（README.md）
  if (!/^[~./]?[\w.@+-]+(\/[\w.@+-]+)*$/.test(noLine)) return null;
  return noLine;
}
