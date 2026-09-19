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
    // 铺满主内容区：宽度用满（不再 max-w-5xl 居中窄栏——知识库是文件树+正文
    // 双栏浏览，宽屏下窄栏会留大片空白，用户实测「没铺满窗口，看起来奇怪」），
    // 高度用 h-full + min-h-0 形成完整 flex 高度链，让 KnowledgeBrowser 的
    // flex-1 双栏（左侧树/右侧正文）真正撑满可视区并可各自滚动。
    <div className="flex h-full min-h-0 w-full flex-col gap-4">
      <header className="flex shrink-0 items-center gap-3">
        <h1 className="text-xl font-semibold text-ink">Dreams</h1>
        <span className="text-xs text-ink-3">
          知识库：domain / loops / skills（dream 产物）
        </span>
      </header>

      <KnowledgeBrowser workspaceId={workspaceId} />

      <div className="shrink-0 rounded-lg border border-dashed border-line bg-space/30 px-4 py-3 text-xs text-ink-3">
        💤 dream 运行日志（<code className="rounded bg-white/5 px-1">.rick/dream/dream_run_*_log.md</code>）浏览
        待后端接口支持——当前可在 Jobs 页查看 job 产物，或直接打开仓库目录。
      </div>
    </div>
  );
}
