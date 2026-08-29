/**
 * TokenGate：无 token 时的全屏锁页（首次体验）。
 *
 * 触发条件：ui store authRequired=true（ApiClient 401 信号）。
 * 表现：传送门旋涡 + 星空背景 + token 输入框（保存 localStorage 后解除）。
 * 保存后建议整页刷新（SSE/REST 全部以新 token 重建连接）。
 */

import { useState } from "react";
import { setStoredToken } from "../../api/client";
import { useUiStore } from "../../stores/ui";
import Button from "../common/Button";
import Portal from "../starfield/Portal";

const FIELD =
  "w-full rounded-lg border border-line bg-space/70 px-4 py-2.5 text-sm text-ink placeholder:text-ink-3 focus:border-portal/60 focus:outline-none";

export default function TokenGate() {
  const setAuthRequired = useUiStore((s) => s.setAuthRequired);
  const [token, setToken] = useState("");
  const [error, setError] = useState<string | null>(null);

  function handleSave(): void {
    const trimmed = token.trim();
    if (!trimmed) {
      setError("请输入 token");
      return;
    }
    setStoredToken(trimmed);
    // 整页刷新：SSE 与所有在途请求以新 token 重建
    window.location.reload();
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-space/95 backdrop-blur">
      <div className="flex w-[min(92vw,380px)] flex-col items-center gap-6 rounded-2xl border border-line bg-surface/60 p-8">
        <div className="relative flex flex-col items-center gap-3">
          <Portal size={64} loading />
          <h1 className="text-lg font-semibold tracking-wide text-ink">rick web</h1>
          <p className="text-center text-xs leading-relaxed text-ink-2">
            需要访问令牌——粘贴 <code className="rounded bg-white/5 px-1">rick web</code>{" "}
            启动时打印的 token
            <br />
            （或 ~/.rick/config.json 的 web_token 字段）
          </p>
        </div>

        <input
          className={FIELD}
          type="password"
          autoFocus
          placeholder="token"
          value={token}
          onChange={(e) => {
            setToken(e.target.value);
            setError(null);
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter") handleSave();
          }}
        />
        {error && <p className="text-xs text-danger">{error}</p>}

        <Button variant="primary" className="w-full" onClick={handleSave}>
          进入
        </Button>

        <button
          type="button"
          className="text-[10px] text-ink-3 hover:text-ink-2"
          onClick={() => setAuthRequired(false)}
        >
          稍后再说（继续以未授权态浏览静态壳）
        </button>
      </div>
    </div>
  );
}
