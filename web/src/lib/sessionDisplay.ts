/**
 * 会话显示名派生（job_53 Bug #5 迭代5b human 反馈：会话页面也要显示任务名——
 * 侧栏（SessionBadge）与会话页（ChatView 头部）共用同一派生逻辑，单源不漂移）。
 *
 * 优先级：会话名（Bug #4 手动重命名）> job 任务名（job_name）> 输入主题
 * （params.requirement / topic 截断——历史会话的持久 title 是「类型+编号」
 * 格式，改为显示用户输入）> 持久 title > 类型·短 id。
 *
 * idTag：会话标识徽标（super: env_N / human-loop: loop_N）——params.id_tag
 * 显式传递（新会话）；历史会话从 title 正则提取。
 */
import type { SessionInfo } from "../types";

/** 截断长度（rune）——与后端 prepTitleFromInput 同口径（60） */
const TITLE_MAX = 60;

function truncate(s: string): string {
  const runes = [...s.trim()];
  return runes.length > TITLE_MAX ? runes.slice(0, TITLE_MAX).join("") : s.trim();
}

export interface SessionDisplay {
  title: string;
  idTag?: string;
}

export function deriveSessionDisplay(session: SessionInfo): SessionDisplay {
  const p = session.params as { requirement?: unknown; topic?: unknown; id_tag?: unknown } | undefined;
  // idTag：显式传递优先；历史会话从 title 正则提取
  const idTagParam = p?.id_tag;
  const idTag =
    typeof idTagParam === "string" && idTagParam
      ? idTagParam
      : session.type === "super"
        ? /\b(env_\d+)\b/.exec(session.title || "")?.[1]
        : session.type === "human-loop"
          ? /\b(loop_\d+)\b/.exec(session.title || "")?.[1]
          : undefined;

  // 输入主题（历史 title 是「类型+编号」格式时用它替代显示）
  const themeText =
    session.type === "super" && typeof p?.requirement === "string"
      ? truncate(p.requirement)
      : session.type === "human-loop" && typeof p?.topic === "string"
        ? truncate(p.topic)
        : "";

  const title =
    session.name?.trim() ||
    session.job_name?.trim() ||
    themeText ||
    session.title ||
    `${session.type} · ${session.id.slice(0, 8)}`;
  return { title, idTag };
}
