/**
 * Humans 视图（/ws/:wsId/humans）——human-loop 学习材料与进度。
 *
 * 单工作区：.rick/draft/ 下学习材料（loops/loop_N 的 mission/research/lessons/
 * judgment + rfc 历史）文件树浏览，供 human 复习。
 * human 明确拒绝一切额外装饰性 UI 面板——本页只有纯文件树 + 内容查看。
 */

import HumansBrowser from "../components/humans/HumansBrowser";

export default function Humans({ workspaceId }: { workspaceId: string }) {
  return (
    // 铺满主内容区（与 Dreams 一致）：h-full + min-h-0 形成完整 flex 高度链，
    // 让 HumansBrowser 的 flex-1 双栏分别滚动、撑满可视区。
    <div className="flex h-full min-h-0 w-full flex-col gap-4">
      <header className="flex shrink-0 items-center gap-3">
        <h1 className="text-xl font-semibold text-ink">Humans</h1>
        <span className="text-xs text-ink-3">human-loop 学习材料与进度</span>
      </header>

      <HumansBrowser workspaceId={workspaceId} />
    </div>
  );
}
