/**
 * FileReaderContext：让聊天气泡里的本地文件链接「就地打开右侧阅读器」。
 *
 * 为什么用 context：Markdown 组件嵌在 MessageList → 多层 memo 组件里，逐层透传
 * 一个回调会污染每个 memo 的比较函数（历史消息本应零重渲）。用 context 只在
 * 「有没有打开能力」这一件事上订阅。
 *
 * job_43 增量 5 起 Provider 挂在 App 壳层（全局）：任意页面的消费方都能打开
 * 阅读器（顶栏工具箱输入路径 / 会话页消息链接点击共用同一面板）。
 *
 * 未挂 Provider（例如知识库页内嵌 markdown）时 hook 返回 null，链接退化为普通
 * 外链语义（新标签打开），行为仍是安全的。
 */

import { createContext, useContext } from "react";

export interface FileReaderApi {
  /** 打开一个本地文件（路径原样交给后端解析：相对工作区 / 绝对 / file://） */
  open: (path: string) => void;
  /** 当前正在阅读的路径（用于高亮/去重） */
  current: string | null;
}

const FileReaderContext = createContext<FileReaderApi | null>(null);

export const FileReaderProvider = FileReaderContext.Provider;

/** 可空读取（无 Provider 时返回 null） */
export function useFileReader(): FileReaderApi | null {
  return useContext(FileReaderContext);
}
