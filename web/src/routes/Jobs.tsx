/**
 * Jobs 看板页：工作区选择器 + jobs 列表 / job 详情（文件浏览/任务状态）。
 *
 * 详情态用本地 state（onOpen/onBack）而非嵌套路由——JobsList/JobDetail 的
 * props 契约（onOpen/onBack）由 L3 组件定死，包装层薄封装即可。
 */

import { useEffect, useState } from "react";
import { useWorkspacesStore } from "../stores/workspaces";
import JobDetail from "../components/jobs/JobDetail";
import JobsList from "../components/jobs/JobsList";
import EmptyState from "../components/common/EmptyState";
import Spinner from "../components/common/Spinner";
import { useIsMobile } from "./hooks";

function WorkspacePicker({
  selected,
  onSelect,
}: {
  selected: string | null;
  onSelect: (id: string) => void;
}) {
  const list = useWorkspacesStore((s) => s.list);
  const refresh = useWorkspacesStore((s) => s.refresh);
  const isMobile = useIsMobile();

  useEffect(() => {
    void refresh();
  }, [refresh]);

  if (list.length === 0) return null;

  const FIELD =
    "rounded-lg border border-line bg-space/60 px-3 py-1.5 text-sm text-ink focus:border-portal/60 focus:outline-none";

  if (isMobile) {
    return (
      <select
        className={FIELD}
        value={selected ?? ""}
        onChange={(e) => onSelect(e.target.value)}
        aria-label="选择工作区"
      >
        {list.map((w) => (
          <option key={w.id} value={w.id}>
            {w.name || w.path}
          </option>
        ))}
      </select>
    );
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      {list.map((w) => (
        <button
          key={w.id}
          type="button"
          onClick={() => onSelect(w.id)}
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
  );
}

export default function Jobs() {
  const list = useWorkspacesStore((s) => s.list);
  const [selected, setSelected] = useState<string | null>(null);
  const [openJob, setOpenJob] = useState<string | null>(null);

  // 默认选中第一个工作区
  useEffect(() => {
    if (!selected && list.length > 0) setSelected(list[0].id);
  }, [selected, list]);

  if (list.length === 0) {
    return (
      <div className="mx-auto w-full max-w-4xl">
        <EmptyState
          message="还没有注册工作区"
          hint="在 Sessions 页添加含 .rick 目录的项目路径后，这里展示 jobs 看板"
        />
      </div>
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-5">
      <header className="flex flex-wrap items-center gap-3">
        <h1 className="text-xl font-semibold text-ink">Jobs</h1>
        <WorkspacePicker
          selected={selected}
          onSelect={(id) => {
            setSelected(id);
            setOpenJob(null);
          }}
        />
      </header>

      {!selected ? (
        <div className="flex items-center gap-2 py-6 text-sm text-ink-3">
          <Spinner size={16} /> 加载…
        </div>
      ) : openJob ? (
        <JobDetail
          workspaceId={selected}
          jobId={openJob}
          onBack={() => setOpenJob(null)}
        />
      ) : (
        <JobsList workspaceId={selected} onOpen={(jobId) => setOpenJob(jobId)} />
      )}
    </div>
  );
}
