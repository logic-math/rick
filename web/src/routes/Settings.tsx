/**
 * Settings 页：token 设置 / 服务器信息 / 自动压缩 / 模型兜底配置。
 *
 * - token：输入 + 保存 localStorage + 「测试连接」（调 /api/config 验证）
 * - 服务器信息：GET /api/config（version / rick_version / port / auth_required）
 */

import { useEffect, useState } from "react";
import { api, getStoredToken, setStoredToken, TOKEN_STORAGE_KEY } from "../api/client";
import { useUiStore } from "../stores/ui";
import type { ServerConfig } from "../types";
import {
  readStarfieldMode,
  writeStarfieldMode,
  STARFIELD_MODE_KEY,
  type StarfieldMode,
} from "../components/starfield/starfieldMode";
import Button from "../components/common/Button";
import ErrorBanner from "../components/common/ErrorBanner";
import {
  readHistoryCacheConfig,
  HISTORY_CACHE_CONFIG_KEY,
  type HistoryCacheConfig,
} from "../lib/sessionHistoryCache";

const FIELD =
  "w-full rounded-lg border border-line bg-space/60 px-3 py-2 text-sm text-ink placeholder:text-ink-3 focus:border-portal/60 focus:outline-none";

function Section({
  title,
  desc,
  children,
}: {
  title: string;
  desc?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3 rounded-xl border border-line bg-surface/50 p-5">
      <div className="flex flex-col gap-1">
        <h2 className="text-base font-semibold text-ink">{title}</h2>
        {desc && <p className="text-xs leading-relaxed text-ink-3">{desc}</p>}
      </div>
      {children}
    </section>
  );
}

export default function Settings() {
  const setAuthRequired = useUiStore((s) => s.setAuthRequired);

  // token
  const [token, setToken] = useState(() => getStoredToken());
  const [tokenSaved, setTokenSaved] = useState(false);
  // Bug #8 2C（human 2026-11-14）：自动压缩设置（上下文使用率超阈值自动 compact）
  const AUTO_COMPACT_KEY = "rick-web:auto-compact";
  const [autoCompact, setAutoCompact] = useState<{ enabled: boolean; threshold: number }>(() => {
    try {
      const raw = localStorage.getItem(AUTO_COMPACT_KEY);
      const v = raw ? JSON.parse(raw) : null;
      if (v && typeof v.enabled === "boolean") {
        return { enabled: v.enabled, threshold: typeof v.threshold === "number" ? v.threshold : 80 };
      }
    } catch {
      // localStorage 不可用——默认
    }
    return { enabled: true, threshold: 80 };
  });
  const saveAutoCompact = (next: { enabled: boolean; threshold: number }) => {
    setAutoCompact(next);
    try {
      localStorage.setItem(AUTO_COMPACT_KEY, JSON.stringify(next));
    } catch {
      // 静默
    }
  };
  // 会话历史缓存（human 2026-11-18 裁决：默认 7 天 / 64 会话，Settings 可调）
  const [historyCache, setHistoryCache] = useState<HistoryCacheConfig>(() => readHistoryCacheConfig());
  const saveHistoryCache = (next: HistoryCacheConfig) => {
    setHistoryCache(next);
    try {
      localStorage.setItem(HISTORY_CACHE_CONFIG_KEY, JSON.stringify(next));
    } catch {
      // 静默
    }
  };
  const [testing, setTesting] = useState(false);
  const [testOk, setTestOk] = useState<boolean | null>(null);
  const [testMsg, setTestMsg] = useState<string | null>(null);

  // 服务器信息
  const [server, setServer] = useState<ServerConfig | null>(null);
  const [serverErr, setServerErr] = useState<string | null>(null);

  // customize / reset
  // 模型兜底配置（Bug 5, human 2026-11-27）
  const [fallbackModels, setFallbackModels] = useState<string>(() => localStorage.getItem("rick-web:fallback-models") ?? "");
  const [fallbackRetryCount, setFallbackRetryCount] = useState<number>(() => Number(localStorage.getItem("rick-web:fallback-retry")) || 3);
  const [fallbackSaveMsg, setFallbackSaveMsg] = useState<string | null>(null);

  // Bug 5：保存模型兜底配置（前端 localStorage——服务端自动切换逻辑下轮迭代）
  function saveFallbackConfig(): void {
    localStorage.setItem("rick-web:fallback-models", fallbackModels);
    localStorage.setItem("rick-web:fallback-retry", String(fallbackRetryCount));
    setFallbackSaveMsg("已保存（前端配置）——服务端自动切换逻辑待下轮迭代，手动配置请在 ~/.rick/config.json 加 fallback_models 字段");
    setTimeout(() => setFallbackSaveMsg(null), 5000);
  }

  // 星空背景模式（job_43 KR4d）：auto=移动端静态/桌面动画（默认），animated=强制动画，static=强制静态
  const [starfieldMode, setStarfieldMode] = useState<StarfieldMode>(() => readStarfieldMode());

  function setMode(mode: StarfieldMode): void {
    setStarfieldMode(mode);
    writeStarfieldMode(mode); // localStorage + 同页签事件（背景立即切换，无需刷新）
  }

  useEffect(() => {
    api
      .getConfig()
      .then(setServer)
      .catch((e) => setServerErr(e instanceof Error ? e.message : String(e)));
  }, []);

  function saveToken(): void {
    setStoredToken(token.trim());
    setTokenSaved(true);
    setTestOk(null);
    // 已授权态复位（下次 401 会重新置位）
    setAuthRequired(false);
    setTimeout(() => setTokenSaved(false), 2000);
  }

  async function testConnection(): Promise<void> {
    setTesting(true);
    setTestOk(null);
    setTestMsg(null);
    try {
      // 先保存再测试（测试用当前输入框的值）
      setStoredToken(token.trim());
      const cfg = await api.getConfig();
      setTestOk(true);
      setTestMsg(`连接成功——rick v${cfg.rick_version}`);
      setServer(cfg);
      setServerErr(null);
      setAuthRequired(false);
    } catch (e) {
      setTestOk(false);
      setTestMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setTesting(false);
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-5">
      <h1 className="text-xl font-semibold text-ink">Settings</h1>

      {/* Token */}
      <Section
        title="访问令牌"
        desc={`Bearer token（存储于 localStorage "${TOKEN_STORAGE_KEY}"；SSE 走 query 参数）。token 在服务端 ~/.rick/config.json 的 web_token 字段（首次 rick web 启动自动生成并打印）。`}
      >
        <div className="flex flex-col gap-2 sm:flex-row">
          <input
            className={FIELD}
            type="password"
            placeholder="粘贴 rick web 启动时打印的 token"
            value={token}
            onChange={(e) => {
              setToken(e.target.value);
              setTokenSaved(false);
            }}
          />
          <div className="flex gap-2">
            <Button variant="primary" onClick={saveToken} disabled={!token.trim()}>
              {tokenSaved ? "已保存 ✓" : "保存"}
            </Button>
            <Button onClick={() => void testConnection()} loading={testing}>
              测试连接
            </Button>
          </div>
        </div>
        {testMsg && (
          <p className={`text-xs ${testOk ? "text-portal" : testOk === false ? "text-danger" : "text-ink-2"}`}>
            {testMsg}
          </p>
        )}
      </Section>

      {/* 星空背景（job_43 移动端性能） */}
      <Section
        title="星空背景"
        desc={`动态星空是 rick 的视觉身份，但全屏 canvas 动画是移动端发烫的主根因。默认「自动」：手机（<768px）静态一帧定格、桌面动画；也可强制全开/全关（存储于 localStorage "${STARFIELD_MODE_KEY}"，切换即时生效）。`}
      >
        <div className="grid grid-cols-3 gap-2">
          {([
            { value: "auto", label: "自动", hint: "手机静态·桌面动画" },
            { value: "animated", label: "动画", hint: "全平台强制动画" },
            { value: "static", label: "静态", hint: "全平台省电定格" },
          ] as const).map((opt) => (
            <button
              key={opt.value}
              type="button"
              aria-pressed={starfieldMode === opt.value}
              onClick={() => setMode(opt.value)}
              className={`flex flex-col gap-1 rounded-lg border px-3 py-2.5 text-left transition-colors ${
                starfieldMode === opt.value
                  ? "border-portal bg-portal-soft"
                  : "border-line bg-space/40 hover:border-portal/40"
              }`}
            >
              <span className="text-sm font-medium text-ink">{opt.label}</span>
              <span className="text-[11px] leading-tight text-ink-3">{opt.hint}</span>
            </button>
          ))}
        </div>
      </Section>

      {/* 服务器信息 */}
      {/* Bug #8 2C：自动压缩（上下文使用率超阈值自动 compact——会话页常驻使用率徽标） */}
      <Section title="自动压缩" desc="会话上下文使用率超过阈值时自动执行 compact（历史总结为摘要，保留最近消息；会话页头部常驻显示使用率）">
        <div className="flex flex-wrap items-center gap-3 text-sm">
          <label className="flex cursor-pointer items-center gap-2">
            <input
              type="checkbox"
              checked={autoCompact.enabled}
              onChange={(e) => saveAutoCompact({ ...autoCompact, enabled: e.target.checked })}
              className="h-4 w-4 accent-portal"
            />
            <span className="text-ink">启用自动压缩</span>
          </label>
          <label className="flex items-center gap-2 text-ink-2">
            阈值
            <input
              type="number"
              min={50}
              max={95}
              value={autoCompact.threshold}
              onChange={(e) => {
                const v = Number(e.target.value);
                if (Number.isFinite(v)) saveAutoCompact({ ...autoCompact, threshold: Math.min(95, Math.max(50, v)) });
              }}
              className="w-16 rounded border border-line bg-space/70 px-2 py-1 text-xs text-ink outline-none focus:border-portal/60"
            />
            %
          </label>
          <span className="text-xs text-ink-3">（默认 80%——使用率徽标 ≥80% 变警示色）</span>
        </div>
      </Section>

      {/* 会话历史缓存（多会话切换加速） */}
      <Section
        title="会话历史缓存"
        desc={`多会话切换加速：切回已打开的会话时立即渲染本地缓存，后台增量拉取新消息保持 timeline 一致。内存缓存（页面刷新后重建）；存储于 localStorage "${HISTORY_CACHE_CONFIG_KEY}"，修改即时生效。`}
      >
        <div className="flex flex-wrap items-center gap-3 text-sm">
          <label className="flex items-center gap-2 text-ink-2">
            缓存时长
            <input
              type="number"
              min={1}
              max={365}
              value={historyCache.ttlDays}
              onChange={(e) => {
                const v = Number(e.target.value);
                if (Number.isFinite(v))
                  saveHistoryCache({ ...historyCache, ttlDays: Math.min(365, Math.max(1, Math.floor(v))) });
              }}
              className="w-16 rounded border border-line bg-space/70 px-2 py-1 text-xs text-ink outline-none focus:border-portal/60"
            />
            天
          </label>
          <label className="flex items-center gap-2 text-ink-2">
            最多缓存
            <input
              type="number"
              min={1}
              max={512}
              value={historyCache.maxSessions}
              onChange={(e) => {
                const v = Number(e.target.value);
                if (Number.isFinite(v))
                  saveHistoryCache({ ...historyCache, maxSessions: Math.min(512, Math.max(1, Math.floor(v))) });
              }}
              className="w-16 rounded border border-line bg-space/70 px-2 py-1 text-xs text-ink outline-none focus:border-portal/60"
            />
            个会话
          </label>
          <span className="text-xs text-ink-3">（默认 7 天 / 64 会话，LRU 淘汰最旧）</span>
        </div>
      </Section>

      {/* 模型兜底配置（Bug 5, human 2026-11-27） */}
      <Section
        title="模型兜底配置"
        desc="当前模型连续失败 N 次后自动切换到兜底模型继续响应（额度用尽/服务商不可用/超时）。配置存储于服务端 ~/.rick/config.json 的 fallback_models 字段（格式：provider/model-id，逗号分隔）。需重启 rick web 生效。"
      >
        <div className="flex flex-col gap-3">
          <label className="flex flex-col gap-1.5">
            <span className="text-sm text-ink-2">兜底模型链（逗号分隔，按优先级）</span>
            <input
              className={FIELD}
              placeholder="catpaw/glm-5.3, selfdeploy/DeepSeek-V4.1-Flash-1017"
              value={fallbackModels}
              onChange={(e) => setFallbackModels(e.target.value)}
              onBlur={() => saveFallbackConfig()}
            />
            <span className="text-[11px] text-ink-3">
              示例：catpaw/glm-5.3, selfdeploy/DeepSeek-V4.1-Flash-1017 — 第一优先用 catpaw/glm-5.3，不可用切 selfdeploy
            </span>
          </label>
          <label className="flex items-center gap-2 text-ink-2">
            重试次数（连续失败 N 次后切换）
            <input
              type="number"
              min={1}
              max={10}
              value={fallbackRetryCount}
              onChange={(e) => {
                const v = Number(e.target.value);
                if (Number.isFinite(v)) setFallbackRetryCount(Math.min(10, Math.max(1, Math.floor(v))));
              }}
              className="w-16 rounded border border-line bg-space/70 px-2 py-1 text-xs text-ink outline-none focus:border-portal/60"
            />
          </label>
          {fallbackSaveMsg && <span className="text-xs text-portal">{fallbackSaveMsg}</span>}
        </div>
      </Section>

      <Section title="服务器" desc="GET /api/config">
        {serverErr ? (
          <ErrorBanner message={serverErr} />        ) : server ? (
          <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm sm:grid-cols-4">
            <div>
              <dt className="text-xs text-ink-3">API 版本</dt>
              <dd className="text-ink">{server.version}</dd>
            </div>
            <div>
              <dt className="text-xs text-ink-3">构建</dt>
              {/* Bug #7 版本探针：跨设备排查新旧 bundle（SW 缓存旧版时两设备 build 不同） */}
              <dd className="font-mono text-xs text-ink-2" title="服务端构建标识（/api/health 的 build_id）">
                {server.build_id ?? server.version ?? "?"}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-ink-3">rick 版本</dt>
              <dd className="text-ink">{server.rick_version}</dd>
            </div>
            <div>
              <dt className="text-xs text-ink-3">端口</dt>
              <dd className="text-ink">{server.port}</dd>
            </div>
            <div>
              <dt className="text-xs text-ink-3">鉴权</dt>
              <dd className="text-ink">{server.auth_required ? "开启" : "关闭"}</dd>
            </div>
          </dl>
        ) : (
          <p className="text-sm text-ink-3">加载中…</p>
        )}
      </Section>

      {/* 前端自迭代已移除（human 裁决 2026-11-28：不允许自迭代） */}
    </div>
  );
}
