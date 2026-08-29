/**
 * gate 结果横幅（事件流里 gate 输出解析）。
 *
 * 数据源：会话事件缓冲中的文本内容（message_end 的 assistant 文本 / 工具结果
 * 文本）——含「✅ / ⛔ / gate / 门禁」关键词的行提取为横幅卡：
 * - ✅ 前缀 → 通过（传送门绿）
 * - ⛔ / 失败 / FAIL 前缀 → 未通过（红）
 * - 其余含 gate/门禁 → 信息（蓝灰）
 *
 * 只保留最近 MAX 条（监控视图密度优先——旧行不淹没新结论）。
 */

import { useMemo } from "react";
import type { SSEEnvelope, PiRpcEvent } from "../../types";

const MAX_BANNERS = 6;

const GATE_RE = /(✅|⛔|❌|gate|门禁|pipeline_gate|level_complete)/i;
const PASS_RE = /✅|通过|passed|all green/i;
// 失败判定需显式标记（“0 errors” 等成功语境不误判——不匹配裸 error 一词）
const FAIL_RE = /⛔|❌|失败|\bfailed\b/i;

/** 从单个 pi 事件提取文本行 */
function textLinesOf(event: PiRpcEvent): string[] {
  const lines: string[] = [];
  // assistant 消息文本
  const content = event.message?.content;
  if (typeof content === "string") {
    lines.push(...content.split("\n"));
  } else if (Array.isArray(content)) {
    for (const block of content) {
      if (block?.type === "text" && typeof block.text === "string") {
        lines.push(...block.text.split("\n"));
      }
    }
  }
  // 工具结果文本（result 可能是对象或字符串）
  if (event.result !== undefined && event.result !== null) {
    if (typeof event.result === "string") {
      lines.push(...event.result.split("\n"));
    } else if (typeof event.result === "object") {
      const r = event.result as Record<string, unknown>;
      for (const key of ["content", "text", "output", "stdout"]) {
        const v = r[key];
        if (typeof v === "string") lines.push(...v.split("\n"));
      }
    }
  }
  return lines;
}

export type GateLevel = "pass" | "fail" | "info";

export interface GateBanner {
  id: string;
  level: GateLevel;
  text: string;
}

/** 从会话事件缓冲解析 gate 横幅（导出供测试/复用） */
export function parseGateBanners(events: SSEEnvelope[], limit = MAX_BANNERS): GateBanner[] {
  const banners: GateBanner[] = [];
  for (const env of events) {
    if (env.type !== "session_event" || !env.session_id) continue;
    const data = env.data as { event?: PiRpcEvent } | undefined;
    const event = data?.event;
    if (!event) continue;
    for (const raw of textLinesOf(event)) {
      const line = raw.trim();
      if (line.length === 0 || line.length > 400) continue;
      if (!GATE_RE.test(line)) continue;
      // 过滤噪声：代码块围栏、纯命令行回显
      if (line.startsWith("```") || line.startsWith("$ ")) continue;
      const level: GateLevel = FAIL_RE.test(line)
        ? "fail"
        : PASS_RE.test(line)
          ? "pass"
          : "info";
      banners.push({ id: `${env.seq}-${banners.length}`, level, text: line });
    }
  }
  return banners.slice(-limit);
}

const LEVEL_STYLE: Record<GateLevel, string> = {
  pass: "border-portal/50 bg-portal-soft text-portal",
  fail: "border-danger/50 bg-danger/10 text-danger",
  info: "border-rick/40 bg-rick/10 text-rick",
};

const LEVEL_ICON: Record<GateLevel, string> = {
  pass: "✅",
  fail: "⛔",
  info: "◈",
};

interface GateResultProps {
  events: SSEEnvelope[];
}

export default function GateResult({ events }: GateResultProps) {
  const banners = useMemo(() => parseGateBanners(events), [events]);
  if (banners.length === 0) return null;
  return (
    <ul className="flex flex-col gap-1.5" aria-label="gate 结果">
      {banners.map((b) => (
        <li
          key={b.id}
          className={`flex items-start gap-2 rounded-lg border px-3 py-2 font-mono text-xs ${LEVEL_STYLE[b.level]}`}
          title={b.text}
        >
          <span aria-hidden="true" className="shrink-0">
            {LEVEL_ICON[b.level]}
          </span>
          <span className="min-w-0 flex-1 break-words">{b.text}</span>
        </li>
      ))}
    </ul>
  );
}
