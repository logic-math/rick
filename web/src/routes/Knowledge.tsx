/**
 * Knowledge 浏览页：工作区选择器 + domain/loops/skills 文件树与内容浏览。
 */

import { useEffect, useState } from "react";
import { useWorkspacesStore } from "../stores/workspaces";
import KnowledgeBrowser from "../components/knowledge/KnowledgeBrowser";
import EmptyState from "../components/common/EmptyState";
import { useIsMobile } from "./hooks";

export default function Knowledge() {
  const list = useWorkspacesStore((s) => s.list);
  const refresh = useWorkspacesStore((s) => s.refresh);
  const isMobile = useIsMobile();
  const [selected, setSelected] = useState<string | null>(null);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (!selected && list.length > 0) setSelected(list[0].id);
  }, [selected, list]);

  if (list.length === 0) {
    return (
      <div className="mx-auto w-full max-w-4xl">
        <EmptyState
          message="还没有注册工作区"
          hint="在 Sessions 页添加工作区后，这里浏览该项目的 domain/loops/skills 知识库"
        />
      </div>
    );
  }

  const FIELD =
    "rounded-lg border border-line bg-space/60 px-3 py-1.5 text-sm text-ink focus:border-portal/60 focus:outline-none";

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-5">
      <header className="flex flex-wrap items-center gap-3">
        <h1 className="text-xl font-semibold text-ink">Knowledge</h1>
        {isMobile ? (
          <select
            className={FIELD}
            value={selected ?? ""}
            onChange={(e) => setSelected(e.target.value)}
            aria-label="选择工作区"
          >
            {list.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name || w.path}
              </option>
            ))}
          </select>
        ) : (
          <div className="flex flex-wrap items-center gap-2">
            {list.map((w) => (
              <button
                key={w.id}
                type="button"
                onClick={() => setSelected(w.id)}
                className={`rounded-lg border px-3 py-1.5 text-sm transition-colors ${
                  selected === w.id
                    ? "border-portal/60 bg-portal-soft text-portal"
                    : "border-line bg-space/40 text-ink-2 hover:bg-white/5 hover:text-ink"
                }`}
              >
                {w.name || w.path}
              </button>
            ))}
          </div>
        )}
      </header>

      {selected ? (
        <KnowledgeBrowser workspaceId={selected} />
      ) : (
        <p className="py-6 text-sm text-ink-3">选择工作区…</p>
      )}
    </div>
  );
}
