/**
 * ModelSwitchDialog：会话模型/思考档位切换。
 *
 * - GET /api/sessions/{id}/models 加载可用模型列表（worker 不可用 → 409 降级提示）
 * - 模型 radio 选择 → POST /api/sessions/{id}/model {provider, model_id}
 * - thinking 档位下拉（off/low/medium/high…）→ POST /api/sessions/{id}/thinking {level}
 *
 * 降级策略：接口 409/网络错误 → 展示「会话不在运行」提示 + 关闭按钮，不崩溃。
 */

import { useEffect, useMemo, useState } from "react";
import { api } from "../../api/client";
import type { SessionModel } from "../../types";
import Dialog from "../common/Dialog";
import Button from "../common/Button";
import Spinner from "../common/Spinner";

const THINKING_LEVELS = ["off", "minimal", "low", "medium", "high", "xhigh", "max"];

interface ModelSwitchDialogProps {
  open: boolean;
  sessionId: string;
  currentModel: string | null;
  currentProvider: string | null;
  currentThinking: string | null;
  onClose: () => void;
  /** 切换成功后回调（父组件刷新展示） */
  onChanged?: (modelName: string | null) => void;
}

export default function ModelSwitchDialog({
  open,
  sessionId,
  currentModel,
  currentProvider,
  currentThinking,
  onClose,
  onChanged,
}: ModelSwitchDialogProps) {
  const [models, setModels] = useState<SessionModel[]>([]);
  const [current, setCurrent] = useState<SessionModel | null>(null);
  const [loading, setLoading] = useState(false);
  const [switching, setSwitching] = useState(false);
  const [thinking, setThinking] = useState<string>(currentThinking ?? "");
  const [error, setError] = useState<string | null>(null);
  const [offline, setOffline] = useState(false);

  // 选定模型（default：当前模型，找不到则第一项）
  const [selectedId, setSelectedId] = useState<string>("");

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    setOffline(false);
    setThinking(currentThinking ?? "");
    api
      .getSessionModels(sessionId)
      .then((res) => {
        if (cancelled) return;
        setModels(res.models ?? []);
        setCurrent(res.current ?? null);
        const curId = res.current?.id ?? currentModel ? undefined : res.models[0]?.id;
        // 优先当前模型 id；当前不在列表里则选第一项
        const preferred = res.models.find((m) => m.id === res.current?.id)?.id
          ?? res.models.find((m) => m.id === currentModel)?.id
          ?? curId
          ?? "";
        setSelectedId(preferred);
      })
      .catch((e) => {
        if (cancelled) return;
        const status = (e as { status?: number })?.status;
        if (status === 409) {
          setOffline(true);
          setModels([]);
        } else {
          setError(e instanceof Error ? e.message : String(e));
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, sessionId, currentThinking]);

  const selectedModel = useMemo(
    () => models.find((m) => m.id === selectedId) ?? null,
    [models, selectedId],
  );

  async function doSwitch(): Promise<void> {
    if (!selectedModel) return;
    setSwitching(true);
    setError(null);
    try {
      await api.setSessionModel(sessionId, selectedModel.provider, selectedModel.id);
      setCurrent(selectedModel);
      onChanged?.(selectedModel.name);
      // thinking 同步（若用户改了档位）
      if (thinking && thinking !== currentThinking) {
        await api.setSessionThinking(sessionId, thinking).catch(() => {
          /* thinking 失败不阻断模型切换——展示为警告 */
          setError("模型已切换，但思考档位设置失败");
        });
      }
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSwitching(false);
    }
  }

  return (
    <Dialog open={open} title="切换模型" onClose={onClose}>
      <div className="flex min-h-0 flex-col gap-3">
        <p className="text-xs text-ink-3">
          pi 支持的模型列表
          {currentProvider ? `（provider: ${currentProvider}）` : ""}——当前：{current?.name ?? currentModel ?? "—"}
        </p>

        {loading && (
          <div className="flex items-center gap-2 py-4 text-xs text-ink-3">
            <Spinner size={18} /> 加载模型列表…
          </div>
        )}

        {offline && (
          <div className="flex flex-col gap-2 rounded-lg border border-morty/40 bg-morty/10 p-3">
            <p className="text-xs text-morty">会话不在运行</p>
            <p className="text-xs leading-relaxed text-ink-3">
              模型切换需要该会话的 rpc worker 存活。当前会话的 worker 未连接
              （可能已关闭或本服务重启过）——请关闭会话后 Resume，或刷新页面重连。
            </p>
            <Button size="sm" variant="ghost" className="self-start" onClick={onClose}>
              知道了
            </Button>
          </div>
        )}

        {error && <p className="text-xs text-danger">{error}</p>}

        {!loading && !offline && (
          <>
            {models.length === 0 ? (
              <p className="py-3 text-xs text-ink-3">无可用模型（服务端返回空列表）</p>
            ) : (
              <div className="flex max-h-56 flex-col gap-1 overflow-y-auto pr-1">
                {models.map((m) => (
                  <label
                    key={m.id}
                    className={`flex cursor-pointer items-center gap-2 rounded-md border px-2.5 py-1.5 text-xs transition-colors ${
                      selectedId === m.id
                        ? "border-portal/60 bg-portal-soft text-ink"
                        : "border-line hover:bg-white/5"
                    }`}
                  >
                    <input
                      type="radio"
                      name="model"
                      className="accent-[var(--rm-portal)]"
                      checked={selectedId === m.id}
                      onChange={() => setSelectedId(m.id)}
                    />
                    <span className="min-w-0 flex-1 truncate">
                      {m.name}
                      <span className="ml-1 text-[10px] text-ink-3">{m.provider}</span>
                    </span>
                    {current?.id === m.id && (
                      <span className="shrink-0 text-[10px] text-portal">✓ 当前</span>
                    )}
                  </label>
                ))}
              </div>
            )}

            <div className="flex items-center gap-2 border-t border-line pt-2">
              <span className="w-24 shrink-0 text-[10px] text-ink-3">思考档位</span>
              <select
                value={thinking}
                onChange={(e) => setThinking(e.target.value)}
                className="flex-1 rounded-md border border-line bg-space/60 px-2 py-1.5 text-xs text-ink focus:border-portal/60 focus:outline-none"
              >
                {THINKING_LEVELS.map((lvl) => (
                  <option key={lvl} value={lvl}>
                    {lvl}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex justify-end gap-2 border-t border-line pt-2">
              <Button size="sm" variant="ghost" onClick={onClose} disabled={switching}>
                取消
              </Button>
              <Button
                size="sm"
                variant="primary"
                onClick={() => void doSwitch()}
                loading={switching}
                disabled={!selectedModel}
              >
                切换
              </Button>
            </div>
          </>
        )}
      </div>
    </Dialog>
  );
}
