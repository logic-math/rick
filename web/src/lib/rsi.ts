/**
 * RSI（自进化）前端辅助：workspace 形态的**提示性**判断与文案。
 *
 * ⚠️ 这里的判断只是启发式，**权威判定在后端**（task24）：后端会校验目标工作区
 * ① 是 rick 源码树（含 cmd/rick 与 internal/web）② 含 .rick/loops/rick-rsi-loop.md
 * ③ **不是**生产仓库根（否则 RSI 会话会直接改生产源码）。
 *
 * 前端用它的唯一目的是：**别让用户白填一遍再吃 400** —— 给出醒目提示引导选 dev 工作区。
 * 因此这里只提示、不阻塞提交（真正的拒绝与中文原因由后端返回，前端原样展示）。
 */

/** loop 文件在工作区内的相对路径（与后端 task24 的解析路径一致） */
export const RSI_LOOP_REL_PATH = ".rick/loops/rick-rsi-loop.md";

/** 最小工作区形状（只用到 path/name，避免与 WorkspaceEntry 强耦合） */
export interface RsiWorkspaceLike {
  path: string;
  name?: string;
}

/** 路径末段（`/a/b/c` → `c`；处理末尾斜杠） */
function lastSegment(path: string): string {
  const trimmed = path.replace(/\/+$/, "");
  const i = trimmed.lastIndexOf("/");
  return i >= 0 ? trimmed.slice(i + 1) : trimmed;
}

/**
 * 是否像 `rick tools dev-web` 建的 dev 工作树。
 * 约定：dev 树默认落在**生产仓库祖父目录**下的 `rick-dev`（可被 RICK_DEV_TREE 覆盖），
 * 因此末段以 `-dev` 结尾是可靠信号；同时接受路径中任意一段以 `-dev` 结尾（如
 * `/workdir/x/rick-dev/tree`）。
 */
export function looksLikeDevTree(ws: RsiWorkspaceLike | null | undefined): boolean {
  if (!ws?.path) return false;
  return /(^|[/-])[^/]*-dev(\/|$)/i.test(ws.path) || /-dev$/i.test(lastSegment(ws.path));
}

/** 是否像 rick 源码树（末段或任意段含 rick） */
export function looksLikeRickSourceTree(ws: RsiWorkspaceLike | null | undefined): boolean {
  if (!ws?.path) return false;
  return /rick/i.test(ws.path) || /rick/i.test(ws.name ?? "");
}

export type RsiWorkspaceVerdict = "ok" | "not-dev" | "not-rick" | "none";

/** 三态判定（仅用于提示文案） */
export function rsiWorkspaceVerdict(ws: RsiWorkspaceLike | null | undefined): RsiWorkspaceVerdict {
  if (!ws?.path) return "none";
  if (looksLikeDevTree(ws)) return "ok";
  if (looksLikeRickSourceTree(ws)) return "not-dev";
  return "not-rick";
}

/** 无论哪种「不合适」都要出现的规范句（用户要求原文；后端还会再校验一次） */
const MUST_BE_DEV_SENTENCE =
  "RSI 会话必须指向 dev 工作区（后端会拒绝生产仓库根）。";

/** dev 工作区的建立方式（三种警示文案共用的引导句） */
const HOWTO =
  "请先用 rick tools dev-web init 建好 dev 工作区（*-dev 工作树），并把它注册为一个工作区再选它。";

/** 中文提示（tone 用于决定配色：ok=portal 绿 / 其余=morty 警示黄） */
export function rsiWorkspaceHint(ws: RsiWorkspaceLike | null | undefined): {
  tone: "portal" | "morty";
  text: string;
} {
  switch (rsiWorkspaceVerdict(ws)) {
    case "ok":
      return {
        tone: "portal",
        text: `✓ dev 工作区：${ws?.path} —— RSI 自进化会在这里隔离开发，生产不受影响`,
      };
    case "not-dev":
      return {
        tone: "morty",
        text: `⚠ ${ws?.path} 看起来不是 dev 工作区（名字不以 -dev 结尾）。${MUST_BE_DEV_SENTENCE}${HOWTO}`,
      };
    case "not-rick":
      return {
        tone: "morty",
        text:
          `⚠ ${ws?.path} 不像 rick 源码树。${MUST_BE_DEV_SENTENCE}` +
          `后端会校验 cmd/rick、internal/web 与 ${RSI_LOOP_REL_PATH} 是否存在。${HOWTO}`,
      };
    default:
      return {
        tone: "morty",
        text: `请选择 dev 工作区：RSI 自进化会加载 ${RSI_LOOP_REL_PATH} 并在隔离环境里改进 rick 自身。`,
      };
  }
}

/** rsi 类型的参数说明（弹窗内展示） */
export const RSI_PARAMS_EXPLAIN = [
  "无需填写参数：启动后由后端把 " + RSI_LOOP_REL_PATH + " **全文注入系统提示词**（method），会话即按 loop 执行。",
  "loop 流程：隔离 dev 开发（rick tools dev-web）→ 层门禁 → 人类确认 → rick tools release（含源码合并）→ 产出由 rick tools rsi_check 校验。",
  "RSI 会话不在生产仓库工作树上直接改代码；release 与重启的确认权始终在人类。",
] as const;
