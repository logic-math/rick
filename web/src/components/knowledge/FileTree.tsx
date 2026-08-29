/**
 * 通用文件树（KnowledgeBrowser 与 JobFiles 复用）。
 *
 * - 入参：扁平路径列表（"/"分隔，如 "domain/bugs.md"）→ 内部构建层级树
 * - 文件夹可折叠（默认展开）；文件点击回调 onSelect(path)
 * - selectedPath 高亮当前文件
 * - 深色 R&M 视觉：文件/文件夹用 lucide 风格内联 SVG 图标（不引图标库）
 */

import { useMemo, useState } from "react";

export interface FileTreeEntry {
  /** 完整相对路径（"/"分隔） */
  path: string;
  /** 字节数（可选——文件节点展示） */
  size?: number;
}

interface TreeNode {
  name: string;
  path: string;
  isDir: boolean;
  size?: number;
  children: TreeNode[];
}

/** 扁平路径 → 层级树（字典序：目录在前） */
export function buildTree(entries: FileTreeEntry[]): TreeNode[] {
  const root: TreeNode = { name: "", path: "", isDir: true, children: [] };
  for (const e of entries) {
    const parts = e.path.split("/").filter(Boolean);
    if (parts.length === 0) continue;
    let node = root;
    for (let i = 0; i < parts.length; i++) {
      const isLeaf = i === parts.length - 1;
      const seg = parts[i];
      const path = parts.slice(0, i + 1).join("/");
      let child = node.children.find((c) => c.name === seg && c.isDir === !isLeaf);
      if (!child) {
        child = {
          name: seg,
          path,
          isDir: !isLeaf,
          size: isLeaf ? e.size : undefined,
          children: [],
        };
        node.children.push(child);
      }
      node = child;
    }
  }
  const sortRec = (n: TreeNode): void => {
    n.children.sort((a, b) => {
      if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
      return a.name.localeCompare(b.name, undefined, { numeric: true });
    });
    n.children.forEach(sortRec);
  };
  sortRec(root);
  return root.children;
}

function humanSize(bytes?: number): string {
  if (bytes === undefined) return "";
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}K`;
  return `${(bytes / 1024 / 1024).toFixed(1)}M`;
}

function FolderIcon({ open }: { open: boolean }) {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" aria-hidden="true" className="shrink-0 text-ink-3">
      <path
        d={open ? "M1.5 3h4l1.5 2h7.5v8.5h-13V3z M2.5 5.5h11" : "M1.5 3h4l1.5 2h7.5v8.5h-13V3z"}
        fill={open ? "rgba(57,255,136,0.10)" : "none"}
        stroke="currentColor"
        strokeWidth="1.2"
      />
    </svg>
  );
}

function FileIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" aria-hidden="true" className="shrink-0 text-ink-3">
      <path
        d="M3.5 1.5h6l3 3v10h-9v-13z M9.5 1.5v3h3"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.2"
      />
    </svg>
  );
}

interface FileTreeProps {
  entries: FileTreeEntry[];
  selectedPath: string | null;
  onSelect: (path: string) => void;
  /** 树容器附加 class（高度/滚动由调用方控制） */
  className?: string;
}

function TreeLevel({
  nodes,
  depth,
  selectedPath,
  onSelect,
  defaultOpen,
}: {
  nodes: TreeNode[];
  depth: number;
  selectedPath: string | null;
  onSelect: (path: string) => void;
  defaultOpen: boolean;
}) {
  return (
    <ul className="min-w-0" role="tree">
      {nodes.map((node) =>
        node.isDir ? (
          <TreeFolder
            key={node.path}
            node={node}
            depth={depth}
            selectedPath={selectedPath}
            onSelect={onSelect}
            defaultOpen={defaultOpen}
          />
        ) : (
          <li key={node.path} role="treeitem" aria-selected={selectedPath === node.path}>
            <button
              type="button"
              className={`flex w-full items-center gap-1.5 rounded px-1.5 py-1 text-left text-xs transition-colors ${
                selectedPath === node.path
                  ? "bg-portal-soft text-portal"
                  : "text-ink-2 hover:bg-white/5 hover:text-ink"
              }`}
              style={{ paddingLeft: `${depth * 14 + 6}px` }}
              onClick={() => onSelect(node.path)}
              title={node.path}
            >
              <FileIcon />
              <span className="min-w-0 flex-1 truncate">{node.name}</span>
              {node.size !== undefined && (
                <span className="shrink-0 text-[10px] text-ink-3">{humanSize(node.size)}</span>
              )}
            </button>
          </li>
        ),
      )}
    </ul>
  );
}

function TreeFolder({
  node,
  depth,
  selectedPath,
  onSelect,
  defaultOpen,
}: {
  node: TreeNode;
  depth: number;
  selectedPath: string | null;
  onSelect: (path: string) => void;
  defaultOpen: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <li key={node.path} role="treeitem" aria-expanded={open}>
      <button
        type="button"
        className="flex w-full items-center gap-1.5 rounded px-1.5 py-1 text-left text-xs font-medium text-ink-2 transition-colors hover:bg-white/5 hover:text-ink"
        style={{ paddingLeft: `${depth * 14 + 6}px` }}
        onClick={() => setOpen((v) => !v)}
        aria-label={`${open ? "折叠" : "展开"} ${node.name}`}
      >
        <span aria-hidden="true" className={`shrink-0 text-[10px] text-ink-3 transition-transform ${open ? "rotate-90" : ""}`}>
          ▶
        </span>
        <FolderIcon open={open} />
        <span className="min-w-0 flex-1 truncate">{node.name}</span>
        <span className="shrink-0 text-[10px] text-ink-3">{node.children.length}</span>
      </button>
      {open && (
        <TreeLevel
          nodes={node.children}
          depth={depth + 1}
          selectedPath={selectedPath}
          onSelect={onSelect}
          defaultOpen={defaultOpen}
        />
      )}
    </li>
  );
}

export default function FileTree({ entries, selectedPath, onSelect, className = "" }: FileTreeProps) {
  const tree = useMemo(() => buildTree(entries), [entries]);
  if (tree.length === 0) {
    return <p className="px-2 py-4 text-xs text-ink-3">（空）</p>;
  }
  return (
    <div className={`min-w-0 overflow-x-auto ${className}`} role="tree">
      <TreeLevel nodes={tree} depth={0} selectedPath={selectedPath} onSelect={onSelect} defaultOpen />
    </div>
  );
}
