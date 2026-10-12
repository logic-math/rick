// humans.go —— humans 工作区页面的只读后端（浏览 human-loop 学习材料与进度）。
//
// 数据源：<ws>/.rick/draft/（与 handler.PrepareHumanLoopDirsIn 同源：rfc/、
// loops/loop_N/{mission.md,judgment.md,research/,lessons/,prompts/,session_id}）。
//
// 接口契约与 knowledge 节同构（api-contract.md）：
//
//	GET /api/workspaces/{ws}/humans/tree → 200 {"tree":[{"path","size"},...]}
//	     （扁平路径列表，"/" 分隔、draft 根相对，前端 FileTree 自行构建层级）
//	GET /api/workspaces/{ws}/humans/file?path=<draft 相对路径> → 200 {"path","content"}
//
// 安全语义（沿用 job_43 裁决：不做白名单根校验，输入什么路径就读什么路径）：
//   - 相对路径按 draft 根解析（绝对路径直接读——不引入更严限制，与
//     handleWorkspaceFile 一致）；
//   - 保留基本防御：NUL 拒绝、仅文件非目录、大小上限（MaxReadFileSize）。
package web

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunquan/rick/internal/workspace"
)

// humansDraftRoot resolves the human-loop artifact root for a workspace:
// <ws>/.rick/draft（字面量 draft 与 PrepareHumanLoopDirsIn 保持一致，
// 常量走 workspace.DraftDirName）。
func humansDraftRoot(wsPath string) string {
	return filepath.Join(wsPath, workspace.RickDirName, workspace.DraftDirName)
}

// HumansTree enumerates the human-loop files under <ws>/.rick/draft
// (recursively), returning draft-relative slash paths + sizes sorted by path.
// Mirrors KnowledgeTree: only whitelisted text extensions are listed,
// symlinks and unreadable entries are skipped, and a missing draft root is
// tolerated as an empty tree (a fresh workspace has no human-loop yet).
func HumansTree(wsPath string) ([]FileNode, error) {
	root := humansDraftRoot(wsPath)
	nodes := make([]FileNode, 0, 64)
	// Errors are swallowed per-entry: a tree listing must not fail because
	// the draft directory is missing or one subtree is unreadable.
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil // never traverse or list symlinks
		}
		if d.IsDir() {
			return nil
		}
		if !knowledgeExtAllowed(path) {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		nodes = append(nodes, FileNode{Path: filepath.ToSlash(rel), Size: info.Size()})
		return nil
	})
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Path < nodes[j].Path })
	return nodes, nil
}

// ReadHumansFile serves a single human-loop file. relPath is resolved against
// <ws>/.rick/draft（实现要点：不做白名单根校验——相对 draft 解析、绝对
// 路径直接读，沿用 job_43 裁决）。保留的基本防御：NUL 拒绝、仅文件非目录、
// MaxReadFileSize 上限。
func ReadHumansFile(wsPath, relPath string) (string, error) {
	abs, err := resolveWorkspaceFilePath(humansDraftRoot(wsPath), relPath)
	if err != nil {
		return "", err
	}
	return readFileBasic(abs)
}

// handleHumansTree serves GET /api/workspaces/{ws}/humans/tree.
func (deps Deps) handleHumansTree() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		nodes, err := HumansTree(ws.Path)
		if err != nil {
			writeError(w, err)
			return
		}
		if nodes == nil {
			nodes = []FileNode{}
		}
		writeJSON(w, http.StatusOK, treeResponse{Tree: nodes})
	}
}

// handleHumansFile serves GET /api/workspaces/{ws}/humans/file?path=<p> →
// 200 {"path","content"}（body 契约与 knowledge/jobs 文件读取一致）。
func (deps Deps) handleHumansFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, ok := deps.workspaceFromPath(w, r)
		if !ok {
			return
		}
		raw := r.URL.Query().Get("path")
		if strings.TrimSpace(raw) == "" {
			writeError(w, newWebError(http.StatusBadRequest, "invalid_params", "missing path"))
			return
		}
		content, err := ReadHumansFile(ws.Path, raw)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, fileResponse{Path: raw, Content: content})
	}
}
