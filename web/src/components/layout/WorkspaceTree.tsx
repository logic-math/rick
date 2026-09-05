/**
 * 侧栏工作区导航（全部工作区平铺列表，v5）。
 *
 * 用户反馈（job_36）：
 *   - v2 全展开 → 内容爆炸难选；v3 选择器下拉 → 「复选框式切换」体验不佳；v4 平铺列表。
 *   - v5（本次）：①**拖拽排序**——工作区行 draggable（HTML5 DnD），拖到目标行即重排
 *     （本地即时更新 + PUT /api/workspaces/order 持久化 + 失败回滚 refresh）；
 *     dragOver 目标行 portal 绿边框高亮；②**注销**——行 hover 显示「注销」按钮 →
 *     确认弹窗（「注销后该工作区从 rick 移除，目录与 job 文件保留，可随时重新添加」）
 *     → DELETE /api/workspaces/{id}（只删注册表不碰目录）→ 刷新列表；当前路由在该
 *     工作区下则重定向到剩余第一个工作区（或空态）。
 *
 * 稳定引用兜底（job_36 黑屏教训）：selector 里 ?? [] 必须用模块级常量。
 */

import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useWorkspacesStore } from "../../stores/workspaces";
import { api } from "../../api/client";
import type { WorkspaceEntry } from "../../types";
import Dialog from "../common/Dialog";
// 复用全局「注册/浏览/创建」工作区卡片（Sessions.tsx 导出；本文件不构成循环——
// routes/Sessions 不依赖 components/layout/*）。
import { AddWorkspaceCard } from "../../routes/Sessions";
import WorkspaceListNode from "./WorkspaceListNode";

interface WorkspaceTreeProps {
  onNewSession: (wsId: string) => void;
  /** 导航点击后收起移动端 drawer（App 注入） */
  onNavigate?: () => void;
}

/** 拖拽中/拖放后的本地顺序（null = 用 store 顺序）；拖拽结束持久化成功后清空 */
interface DragState {
  /** 当前被拖拽的工作区 id */
  id: string | null;
  /** dragOver 目标行 index（高亮用） */
  overIndex: number | null;
}

export default function WorkspaceTree({ onNewSession, onNavigate }: WorkspaceTreeProps) {
  const workspaces = useWorkspacesStore((s) => s.list);
  const refresh = useWorkspacesStore((s) => s.refresh);
  const removeWs = useWorkspacesStore((s) => s.remove);
  const navigate = useNavigate();
  const location = useLocation();
  const [addOpen, setAddOpen] = useState(false);
  // 拖拽状态（本地顺序 + 进行中指示）
  const [localOrder, setLocalOrder] = useState<string[] | null>(null);
  const [drag, setDrag] = useState<DragState>({ id: null, overIndex: null });
  const [dragError, setDragError] = useState<string | null>(null);
  // 注销状态
  const [removeTarget, setRemoveTarget] = useState<WorkspaceEntry | null>(null);
  const [removeBusy, setRemoveBusy] = useState(false);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  // 展示列表：localOrder 非空（拖拽中/待持久化）时按其重排，否则用 store 顺序
  const displayList = useMemo(() => {
    if (!localOrder) return workspaces;
    const byId = new Map(workspaces.map((w) => [w.id, w]));
    return localOrder
      .map((id) => byId.get(id))
      .filter((w): w is WorkspaceEntry => !!w);
  }, [workspaces, localOrder]);

  const handleAdded = (id: string): void => {
    setAddOpen(false);
    void id;
  };

  // ---- 拖拽排序 ----

  const handleDragStart = (id: string): void => {
    setDrag({ id, overIndex: null });
  };

  const handleDragOver = (index: number): void => {
    setDrag((s) => (s.overIndex === index ? s : { ...s, overIndex: index }));
  };

  const handleDrop = async (targetIndex: number): Promise<void> => {
    const fromId = drag.id;
    const base = localOrder ?? workspaces.map((w) => w.id);
    const fromIndex = base.indexOf(fromId ?? "");
    if (fromId == null || fromIndex < 0 || fromIndex === targetIndex) {
      setDrag({ id: null, overIndex: null });
      return;
    }
    const arr = base.slice();
    const [moved] = arr.splice(fromIndex, 1);
    arr.splice(targetIndex, 0, moved);
    // 本地即时更新（拖拽响应优先）
    setLocalOrder(arr);
    setDrag({ id: null, overIndex: null });
    setDragError(null);
    try {
      await api.reorderWorkspaces(arr);
      // 持久化成功：以服务端顺序为准刷新并释放本地覆盖
      await refresh();
      setLocalOrder(null);
    } catch (e) {
      // 失败回滚到 store 顺序
      setLocalOrder(null);
      setDragError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleDragEnd = (): void => {
    setDrag({ id: null, overIndex: null });
  };

  // ---- 注销 ----

  const handleRemove = async (): Promise<void> => {
    if (!removeTarget) return;
    setRemoveBusy(true);
    setDragError(null);
    try {
      await removeWs(removeTarget.id);
      // 当前路由在该工作区下 → 重定向到剩余第一个工作区或空态
      if (location.pathname.startsWith(`/ws/${removeTarget.id}`)) {
        const rest = workspaces.filter((w) => w.id !== removeTarget.id);
        navigate(rest.length > 0 ? `/ws/${rest[0].id}/sessions` : "/");
      }
      setRemoveTarget(null);
    } catch (e) {
      setDragError(e instanceof Error ? e.message : String(e));
    } finally {
      setRemoveBusy(false);
    }
  };

  return (
    <div className="flex flex-col gap-0.5">
      {displayList.length === 0 ? (
        <p className="px-3 py-2 text-xs leading-relaxed text-ink-3">
          暂无工作区——先添加一个含 .rick 的项目目录
        </p>
      ) : (
        displayList.map((w, index) => (
          <WorkspaceRow
            key={w.id}
            workspace={w}
            isDragging={drag.id === w.id}
            isDropTarget={drag.overIndex === index}
            onDragStart={() => handleDragStart(w.id)}
            onDragOver={() => handleDragOver(index)}
            onDrop={() => void handleDrop(index)}
            onDragEnd={handleDragEnd}
            onRemove={() => setRemoveTarget(w)}
          >
            <WorkspaceListNode
              workspaceId={w.id}
              onNewSession={onNewSession}
              onNavigate={onNavigate}
            />
          </WorkspaceRow>
        ))
      )}

      {dragError && (
        <p className="px-3 py-1 text-[10px] text-danger" role="alert">
          ⚠ {dragError}
        </p>
      )}

      {/* 常驻「添加工作区」入口（有工作区时也能浏览/创建） */}
      <button
        type="button"
        onClick={() => setAddOpen(true)}
        className="mx-2 mt-1 flex items-center justify-center gap-1 rounded-md border border-dashed border-line px-2 py-1.5 text-xs text-ink-3 transition-colors hover:border-portal/40 hover:text-portal"
      >
        + 添加工作区
      </button>

      <Dialog open={addOpen} title="工作区" onClose={() => setAddOpen(false)}>
        <AddWorkspaceCard onAdded={handleAdded} />
      </Dialog>

      {/* 注销确认弹窗 */}
      <Dialog
        open={removeTarget !== null}
        title="注销工作区"
        onClose={() => setRemoveTarget(null)}
        contentClassName="w-full max-w-md"
      >
        {removeTarget && (
          <div className="flex flex-col gap-4">
            <p className="text-sm leading-relaxed text-ink-2">
              注销 <code className="rounded bg-white/5 px-1.5 py-0.5 text-ink">{removeTarget.name || removeTarget.path}</code>
              （{removeTarget.path}）？
            </p>
            <p className="text-xs leading-relaxed text-ink-3">
              注销后该工作区从 rick 移除，<b>目录与 job 文件保留</b>，可随时重新添加。
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setRemoveTarget(null)}
                className="rounded-md border border-line px-3 py-1.5 text-xs text-ink-2 hover:bg-white/5"
              >
                取消
              </button>
              <button
                type="button"
                onClick={() => void handleRemove()}
                disabled={removeBusy}
                className="rounded-md border border-danger/60 bg-danger/10 px-3 py-1.5 text-xs text-danger hover:bg-danger/20 disabled:opacity-40"
              >
                {removeBusy ? "注销中…" : "确认注销"}
              </button>
            </div>
          </div>
        )}
      </Dialog>
    </div>
  );
}

/**
 * WorkspaceRow：单行工作区拖拽容器 + hover 注销按钮。
 *
 * - draggable + HTML5 DnD（dragstart/dragover/drop/dragend）；移动端不启用
 *   （touch 手势与 DnD 冲突——桌面端语义，移动端仍可用行内展开/会话导航）。
 * - 拖拽中源行半透明、drop 目标行 portal 绿边框高亮。
 * - hover 显示右上角「注销」按钮（点击不触发行内展开——按钮在列表节点上方）。
 */
function WorkspaceRow({
  workspace,
  isDragging,
  isDropTarget,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
  onRemove,
  children,
}: {
  workspace: WorkspaceEntry;
  isDragging: boolean;
  isDropTarget: boolean;
  onDragStart: () => void;
  onDragOver: () => void;
  onDrop: () => void;
  onDragEnd: () => void;
  onRemove: () => void;
  children: React.ReactNode;
}) {
  return (
    <div
      draggable
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", workspace.id);
        e.dataTransfer.effectAllowed = "move";
        onDragStart();
      }}
      onDragOver={(e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        onDragOver();
      }}
      onDrop={(e) => {
        e.preventDefault();
        onDrop();
      }}
      onDragEnd={onDragEnd}
      className={`group relative rounded-md transition-shadow ${
        isDragging ? "opacity-50" : ""
      } ${isDropTarget ? "shadow-[0_0_0_1.5px] shadow-portal/60" : ""}`}
      title="拖拽排序（拖动到目标工作区位置）"
    >
      {children}
      {/* 注销按钮（hover 显示，不触发行内展开） */}
      <button
        type="button"
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          onRemove();
        }}
        aria-label={`注销 ${workspace.name || workspace.path}`}
        title="注销：从 rick 移除（目录保留）"
        className="absolute right-1.5 top-1/2 z-10 -translate-y-1/2 rounded-md border border-line bg-space-2/90 px-1.5 py-0.5 text-[10px] text-ink-3 opacity-0 transition-opacity hover:border-danger/50 hover:text-danger group-hover:opacity-100"
      >
        🗑 注销
      </button>
    </div>
  );
}
