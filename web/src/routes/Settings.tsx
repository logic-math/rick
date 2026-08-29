/**
 * Settings 页：token 设置 / 服务器信息 / 前端自迭代（Customize / Reset）。
 *
 * - token：输入 + 保存 localStorage + 「测试连接」（调 /api/config 验证）
 * - Customize UI：POST /api/web/customize → {ok, scaffolded}；
 *   scaffolded=false 表示覆盖层已存在跳过；提示「在任意会话中让 agent 修改
 *   ~/.rick/web/src 后 npm run build——watcher 会推 frontend_reload 自动刷新」
 * - Reset to baseline：确认弹窗 → POST /api/web/reset（删覆盖层回 embed 基线）
 * - 服务器信息：GET /api/config（version / rick_version / port / auth_required）
 */

import { useEffect, useState } from "react";
import { api, getStoredToken, setStoredToken, TOKEN_STORAGE_KEY } from "../api/client";
import { useUiStore } from "../stores/ui";
import type { ServerConfig } from "../types";
import Button from "../components/common/Button";
import Dialog from "../components/common/Dialog";
import ErrorBanner from "../components/common/ErrorBanner";
import Portal from "../components/starfield/Portal";

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
  const [testing, setTesting] = useState(false);
  const [testOk, setTestOk] = useState<boolean | null>(null);
  const [testMsg, setTestMsg] = useState<string | null>(null);

  // 服务器信息
  const [server, setServer] = useState<ServerConfig | null>(null);
  const [serverErr, setServerErr] = useState<string | null>(null);

  // customize / reset
  const [customizing, setCustomizing] = useState(false);
  const [customizeMsg, setCustomizeMsg] = useState<string | null>(null);
  const [confirmReset, setConfirmReset] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [resetMsg, setResetMsg] = useState<string | null>(null);
  const [actionErr, setActionErr] = useState<string | null>(null);

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

  async function handleCustomize(): Promise<void> {
    setCustomizing(true);
    setCustomizeMsg(null);
    setActionErr(null);
    try {
      const result = await api.customize();
      setCustomizeMsg(
        result.scaffolded
          ? "基线源码已抽取到 ~/.rick/web/src——在任意 agent 会话中说「修改 rick web 前端：…」即可开始自迭代（agent 编辑后 npm run build，页面自动刷新）"
          : "覆盖层已存在（~/.rick/web/src）——直接在任意会话中让 agent 修改即可",
      );
    } catch (e) {
      setActionErr(e instanceof Error ? e.message : String(e));
    } finally {
      setCustomizing(false);
    }
  }

  async function handleReset(): Promise<void> {
    setResetting(true);
    setResetMsg(null);
    setActionErr(null);
    try {
      await api.reset();
      setConfirmReset(false);
      setResetMsg("已复位——删除 ~/.rick/web/{src,dist}，回到内嵌 baseline（页面即将自动刷新）");
      // 服务端 watcher 会推 frontend_reload；保险起见延迟刷新一次
      setTimeout(() => window.location.reload(), 1500);
    } catch (e) {
      setActionErr(e instanceof Error ? e.message : String(e));
    } finally {
      setResetting(false);
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

      {/* 服务器信息 */}
      <Section title="服务器" desc="GET /api/config">
        {serverErr ? (
          <ErrorBanner message={serverErr} />
        ) : server ? (
          <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm sm:grid-cols-4">
            <div>
              <dt className="text-xs text-ink-3">API 版本</dt>
              <dd className="text-ink">{server.version}</dd>
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

      {/* 前端自迭代 */}
      <Section
        title="前端自迭代"
        desc="通过对话改造 rick web 自身：抽取基线源码到可写层 → 在任意 agent 会话中让 agent 编辑 ~/.rick/web/src 并 npm run build → watcher 推送 frontend_reload 自动生效；改坏了随时复位回内嵌 baseline。"
      >
        <div className="flex flex-wrap gap-2">
          <Button variant="primary" onClick={() => void handleCustomize()} loading={customizing}>
            Customize UI
          </Button>
          <Button variant="danger" onClick={() => setConfirmReset(true)}>
            Reset to baseline
          </Button>
        </div>
        {customizeMsg && <p className="text-xs leading-relaxed text-portal">{customizeMsg}</p>}
        {resetMsg && <p className="text-xs leading-relaxed text-morty">{resetMsg}</p>}
        <ErrorBanner message={actionErr} onDismiss={() => setActionErr(null)} />
      </Section>

      <p className="flex items-center gap-2 text-xs text-ink-3">
        <Portal size={14} /> rick web · 对抗上下文熵增 · AICoding = Humans + Agents
      </p>

      {/* Reset 确认弹窗 */}
      <Dialog
        open={confirmReset}
        onClose={() => setConfirmReset(false)}
        title={<span className="flex items-center gap-2">⚠️ 复位前端自定义层</span>}
        contentClassName="w-[min(92vw,420px)]"
      >
        <div className="flex flex-col gap-4">
          <p className="text-sm leading-relaxed text-ink-2">
            将删除 <code className="rounded bg-white/5 px-1">~/.rick/web/</code> 下的 src 与 dist
            （工作区注册表与会话记录保留），前端回到内嵌 baseline。
            agent 的全部自迭代修改将丢失——确认？
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setConfirmReset(false)}>
              取消
            </Button>
            <Button variant="danger" onClick={() => void handleReset()} loading={resetting}>
              确认复位
            </Button>
          </div>
        </div>
      </Dialog>
    </div>
  );
}
