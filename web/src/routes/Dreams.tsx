/**
 * Dreams 视图（/ws/:wsId/dreams）——原 Knowledge 页改名（贴合 rick 概念：
 * dream 是跨 job 反思/知识演化机制，domain/loops/skills 是它的产物）。
 *
 * 单工作区：domain/loops/skills 知识库文件树浏览。
 * dream 日志（.rick/dream/dream_run_*_log.md）浏览待后端接口支持（占位说明）。
 */

import KnowledgeBrowser from "../components/knowledge/KnowledgeBrowser";

export default function Dreams({ workspaceId }: { workspaceId: string }) {
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-5">
      <header className="flex items-center gap-3">
        <h1 className="text-xl font-semibold text-ink">Dreams</h1>
        <span className="text-xs text-ink-3">
          知识库：domain / loops / skills（dream 产物）
        </span>
      </header>

      <KnowledgeBrowser workspaceId={workspaceId} />

      <div className="rounded-lg border border-dashed border-line bg-space/30 px-4 py-3 text-xs text-ink-3">
        💤 dream 运行日志（<code className="rounded bg-white/5 px-1">.rick/dream/dream_run_*_log.md</code>）浏览
        待后端接口支持——当前可在 Jobs 页查看 job 产物，或直接打开仓库目录。
      </div>
    </div>
  );
}
