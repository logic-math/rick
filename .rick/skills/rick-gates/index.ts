// rick-gates — rick doing 的确定性 hook 扩展（v4.4.2：测试收敛到层门禁）。
//
// 工具：
//   - level_complete：层门禁检查点——跑 human 确认的 gate{N}.py（模块级集成
//     测试，层验收唯一标准）→ 绿 → git add -A 单次 commit → tasks.json 批量写
//   - pipeline_gate：流水线结构门禁（分层 DAG / 写域互斥 / gate 存在性）
// task 级无专门测试脚本（worker 按 # 测试方法 自测，过程性）；会话结束后
// rick 侧 helper.py 门禁兜底（zombie/缺 hash 校验）。
//
// 部署：internal/env/customizations.go 把 <repo>/.rick/skills/rick-gates/ 复制到
// ~/.rick/pi/agent/extensions/rick-gates/（本 index.ts + helper.py + run_tests.py）。
// pi 自动发现 ~/.rick/pi/agent/extensions/*/index.ts。

import * as fs from "node:fs";
import * as path from "node:path";
import { execFileSync } from "node:child_process";
import { Type } from "@earendil-works/pi-ai";
import { defineTool, type ExtensionAPI } from "@earendil-works/pi-coding-agent";

function git(repoRoot: string, ...args: string[]): string {
	return execFileSync("git", ["-C", repoRoot, ...args], {
		encoding: "utf-8",
		maxBuffer: 64 * 1024 * 1024,
	});
}

function findGitRoot(start: string): string {
	let dir = path.resolve(start);
	for (;;) {
		if (fs.existsSync(path.join(dir, ".git"))) return dir;
		const parent = path.dirname(dir);
		if (parent === dir) return path.resolve(start);
		dir = parent;
	}
}

interface LevelCompleteParams {
	level_tasks: string[];
	doing_dir: string;
	gate_cmd?: string;
	summary: string;
}

const levelCompleteTool = defineTool({
	name: "level_complete",
	label: "Level Complete (deterministic layer gate + commit)",
	description:
		"声明一个 DAG 层的全部 task 完成，触发层门禁检查点（v4.4.2 测试收敛到层门禁）：hook 运行该层的**门禁程序**（gate_cmd 必填——" +
		"plan 阶段 human 确认的 gate{N}.py，携带模块级集成测试，是层验收唯一标准）。门禁 exit 0 才继续；" +
		"然后 git add -A + 单次 commit（feat(layer): <task 列表>: <summary>），并把该层所有 task 的 status=success + commit_hash 批量写回 tasks.json。" +
		"门禁失败 → 拒绝提交并输出失败详情（parent 只需重派失败 task 的 impl-worker 后重试）。" +
		"impl-worker 不应调用任何提交工具——层内并行时无 git 操作，提交权集中在检查点。",
	parameters: Type.Object({
		level_tasks: Type.Array(Type.String(), { description: "本层全部 task id，如 [\"task2\", \"task4\"]" }),
		doing_dir: Type.String({ description: "该 job 的 doing 目录绝对路径" }),
		gate_cmd: Type.String({ description: "门禁命令（human 确认的层门禁，如 python3 <plan>/gates/gate1.py）——必填" }),
		summary: Type.String({ description: "一句话层摘要（写进 commit message）" }),
	}),
	async execute(_id, params: LevelCompleteParams) {
		const { level_tasks, doing_dir, summary } = params;
		if (!level_tasks || level_tasks.length === 0) {
			return { content: [{ type: "text", text: "❌ level_tasks 为空" }], isError: true };
		}
		if (!params.gate_cmd) {
			return { content: [{ type: "text", text: "❌ gate_cmd 必填（human 确认的层门禁，如 python3 <plan>/gates/gate{N}.py）——测试已收敛到层门禁，无逐 task 缺省模式" }], isError: true };
		}
		const repoRoot = findGitRoot(doing_dir);

		// 1. 层门禁（确定性）：human 确认的 gate{N}.py（模块集成测试）
		const outputs: string[] = [];
		try {
			const out = execFileSync("bash", ["-c", params.gate_cmd], {
				cwd: repoRoot,
				encoding: "utf-8",
				maxBuffer: 64 * 1024 * 1024,
				timeout: 900_000,
			});
			outputs.push(`门禁 ✅：${params.gate_cmd}\n${out.trim().slice(-1200)}`);
		} catch (err: unknown) {
			const e = err as { stdout?: string; stderr?: string; killed?: boolean };
			const out = `${e.stdout ?? ""}\n${e.stderr ?? ""}`.trim();
			return {
				content: [{ type: "text", text: `❌ 层门禁未通过（human 确认的验收标准），拒绝提交。门禁：${params.gate_cmd}${e.killed ? "（超时 15min）" : ""}\n${out.slice(-3000)}` }],
				isError: true,
			};
		}
		// 2. 全绿 → 单次确定性 commit（层粒度）
		git(repoRoot, "add", "-A");
		let commitHash = "";
		const taskList = level_tasks.join("+");
		try {
			git(repoRoot, "-c", "user.name=rick", "-c", "user.email=rick@local",
				"commit", "-m", `feat(layer): ${taskList}: ${summary.replace(/\n+/g, " ").slice(0, 100)}`);
		} catch (err: unknown) {
			const e = err as { stdout?: string; stderr?: string };
			const out = `${e.stdout ?? ""}${e.stderr ?? ""}`;
			if (!out.includes("nothing to commit") && !out.includes("no changes added to commit")) {
				return { content: [{ type: "text", text: `❌ git commit 失败：${out.slice(-2000)}` }], isError: true };
			}
			try {
				commitHash = git(repoRoot, "log", "-1", "--grep", `feat(layer): ${taskList}`, "--format=%H").trim();
			} catch {
				commitHash = "";
			}
		}
		if (!commitHash) {
			commitHash = git(repoRoot, "rev-parse", "HEAD").trim();
		}

		// 3. 批量写回 tasks.json（层内全部 task：success + 同一 commit_hash）
		try {
			updateTasksJSONLevel(doing_dir, level_tasks, commitHash);
		} catch (err: unknown) {
			return { content: [{ type: "text", text: `❌ 更新 tasks.json 失败：${String(err)}` }], isError: true };
		}

		return {
			content: [{
				type: "text",
				text: `✅ 层检查点完成（${level_tasks.length} 个 task）：测试全绿 + commit ${commitHash.slice(0, 8)} + tasks.json 批量更新\n${outputs.join("\n")}`,
			}],
		};
	},
});

function updateTasksJSONLevel(doingDir: string, taskIDs: string[], commitHash: string): void {
	const tasksPath = path.join(doingDir, "tasks.json");
	const raw = fs.readFileSync(tasksPath, "utf-8");
	const data = JSON.parse(raw);
	if (!Array.isArray(data.tasks)) throw new Error(`tasks.json missing tasks array: ${tasksPath}`);
	const now = new Date().toISOString();
	const idSet = new Set(taskIDs);
	for (const task of data.tasks) {
		if (!task || !idSet.has(task.task_id)) continue;
		if (task.status === "success" && task.commit_hash) {
			throw new Error(`task ${task.task_id} already success (${String(task.commit_hash).slice(0, 8)}) — duplicate level_complete`);
		}
		task.status = "success";
		task.commit_hash = commitHash;
		task.attempts = (task.attempts ?? 0) + 1;
		if (!task.created_at) task.created_at = now;
		task.updated_at = now;
	}
	data.updated_at = now;
	const tmp = `${tasksPath}.tmp`;
	fs.writeFileSync(tmp, JSON.stringify(data, null, 2), "utf-8");
	fs.renameSync(tmp, tasksPath);
}

interface PipelineGateParams {
	doing_dir: string;
	plan_dir: string;
}

// ---- grilling 确定性门禁（v4.4.15：追问流程的产出物校验）----

interface GrillingGateParams {
	grilling_dir: string; // {{grilling_workdir}}：design-tree.md 与 research 简报所在目录
}

function countLayers(tree: string): number {
	const m = tree.match(/^(#+\s*(第\s*[0-9一二三四五六七八九十]+\s*层|Layer\s*\d+|L\d+))/gim);
	return m ? m.length : 0;
}

function hasExplicitAllResolved(tree: string): boolean {
	return /全部消解|无判断节点|fully resolved|no judgment nodes/i.test(tree);
}

function countQuestionsAsked(tree: string): number {
	// 判断节点被上呈的痕迹：推荐答案标记 / 问题编号 / 追问记录
	const m = tree.match(/推荐[：:]|Q\d+|问题\s*\d+|❓|判断节点/g);
	return m ? m.length : 0;
}

const grillingGateTool = defineTool({
	name: "grilling_gate",
	label: "Grilling Gate (deterministic process validation)",
	description:
		"grilling 追问流程的确定性门禁（在声明 grilling 完成后、进入实现流水线设计前调用）：校验流程产出物是否齐备——" +
		"① design-tree.md 存在且含根层 OKR（O/目标 + KR）；② 树有多层结构（层数 ≥1，以「第 N 层/Layer N/LN」标题计）；" +
		"③ 每层要么有 research-L{N}.md 简报、要么在树中显式记录「本层全部消解，无判断节点」；" +
		"④ 存在对 human 的提问痕迹（推荐答案/问题编号/判断节点记录）或显式全消解声明。" +
		"任一缺失 → isError 并给出补齐指引（缺什么补什么后重跑本门禁）。通过 → 才允许进入实现流水线设计。",
	parameters: Type.Object({
		grilling_dir: Type.String({ description: "grilling 工作目录绝对路径（design-tree.md 与 research-L{N}.md 所在目录，如 <plan>/grilling 或 <doing>/grilling）" }),
	}),
	async execute(_id, params: GrillingGateParams) {
		const { grilling_dir } = params;
		const errors: string[] = [];
		const treePath = path.join(grilling_dir, "design-tree.md");

		// ① design-tree.md 存在
		if (!fs.existsSync(treePath)) {
			return {
				content: [{ type: "text", text: `⛔ grilling 门禁失败：设计树缺失（${treePath} 不存在）——grilling 的第一动作就是建立设计树根层（O + KR 集）并落盘。补齐后重跑本门禁。` }],
				isError: true,
			};
		}
		const tree = fs.readFileSync(treePath, "utf-8");

		// ② 根层 OKR（O + KR 痕迹）——O 后跟任意括号/冒号（全角半角均可）
		const hasO = /(^|\n|\s)O\s*[（(:：]/.test(tree) || /(^|\n)#{1,3}\s*(O\b|目标|Objective)/im.test(tree);
		const hasKR = /KR\s*[\d（(：:]|关键结果/.test(tree);
		if (!hasO || !hasKR) {
			errors.push("设计树缺根层 OKR 结构（未见 O/目标 与 KR/关键结果标记）——根层必须是具体的 OKR");
		}

		// ③ 层结构与每层消解证据
		const layers = countLayers(tree);
		if (layers < 1) {
			errors.push("设计树无分层结构（未见「第 N 层/Layer N/LN」标题）——逐层下钻是必须的流程");
		}
		for (let n = 1; n <= layers; n++) {
			const research = path.join(grilling_dir, `research-L${n}.md`);
			const researchR2 = path.join(grilling_dir, `research-L${n}-r2.md`);
			if (!fs.existsSync(research) && !fs.existsSync(researchR2)) {
				// 无简报：树中须有该层显式全消解声明
				const layerHeader = new RegExp(`第\\s*[${n}]\\s*层|Layer\\s*${n}|L${n}\\b`, "i");
				if (!hasExplicitAllResolved(tree)) {
					errors.push(`第 ${n} 层无 research-L${n}.md 简报，且树中无「本层全部消解」显式声明——重量级问题必须派 research，全消解必须显式记录`);
				}
			}
		}

		// ④ 提问痕迹
		const questions = countQuestionsAsked(tree);
		const allResolved = hasExplicitAllResolved(tree);
		if (questions === 0 && !allResolved) {
			errors.push("无对 human 的提问痕迹（推荐答案/问题编号/判断节点记录均缺失）——判断节点必须上呈 human（可附推荐答案，禁自行拍板）；若真无判断节点，须在树中显式记录「本层全部消解」");
		}

		if (errors.length > 0) {
			return {
				content: [{ type: "text", text: `⛔ grilling 门禁失败（${errors.length} 项）：\n\n${errors.map((e) => `- ${e}`).join("\n")}\n\n补齐后重跑 grilling_gate。` }],
				isError: true,
			};
		}
		return {
			content: [{ type: "text", text: `✅ grilling 门禁通过：设计树 ${layers} 层（根层 OKR 完整），research 简报齐备或显式全消解，提问/消解证据在案。可进入实现流水线设计。` }],
		};
	},
});

// ---- 流水线结构门禁（v4.4.1：确定性校验收口到 hook，Go 侧保持薄）----

interface GateTask {
	id: string;
	deps: string[];
	writeDomain: string[];
}

function parseDepsSection(md: string): string[] {
	const section = extractSection(md, "# 依赖关系");
	if (!section) return [];
	const out: string[] = [];
	for (const tok of section.split(/[，,\n]/)) {
		const t = tok.trim();
		if (!t || /^(无|无依赖|none|null|nil|n\/a|na|-)$/i.test(t.replace(/^[（(]+|[)）]+$/g, ""))) continue;
		out.push(t);
	}
	return out;
}

function parseWriteDomainSection(md: string): string[] {
	const section = extractSection(md, "# 写域");
	if (!section) return [];
	const out: string[] = [];
	for (let line of section.split("\n")) {
		const t = line.trim();
		if (!t || t.startsWith("[") || t.startsWith("#")) continue;
		out.push(t.replace(/^- /, ""));
	}
	return out.filter((s) => s.length > 0);
}

function extractSection(md: string, heading: string): string | undefined {
	const lines = md.split("\n");
	let found = false;
	let level = 0;
	const out: string[] = [];
	for (const line of lines) {
		const t = line.trim();
		if (t === heading) {
			found = true;
			level = (t.match(/^#+/) ?? ["#"])[0].length;
			continue;
		}
		if (!found) continue;
		if (t.startsWith("#") && (t.match(/^#+/) ?? ["#"])[0].length <= level) break;
		out.push(line);
	}
	while (out.length && out[0].trim() === "") out.shift();
	while (out.length && out[out.length - 1].trim() === "") out.pop();
	return out.join("\n");
}

function writeDomainConflict(a: string, b: string): boolean {
	a = a.trim();
	b = b.trim();
	if (!a || !b) return false;
	if (a === b) return true;
	if (a.endsWith("/") && b.startsWith(a)) return true;
	if (b.endsWith("/") && a.startsWith(b)) return true;
	return false;
}

async function readDir(p: string): Promise<string[]> {
	try {
		return await fs.promises.readdir(p);
	} catch {
		return [];
	}
}

const pipelineGateTool = defineTool({
	name: "pipeline_gate",
	label: "Pipeline Gate (structural validation)",
	description:
		"流水线结构门禁（执行任何派发之前的确定性校验）：①读取 tasks.json（或 plan/task*.md）取 pending task，Kahn 分层并检测环与依赖引用存在性（依赖必须在 pending 集或已 success）；" +
		"②同层多 task 必须全员声明 # 写域 且两两不相交（相等/目录前缀包含=冲突；单 task 层豁免）；" +
		"③每层 plan/gates/gate{N}.py（N=层序号）必须存在。任一失败 → isError 并给出修正指引；全部通过 → 输出分层结构清单。",
	parameters: Type.Object({
		doing_dir: Type.String({ description: "该 job 的 doing 目录绝对路径（读 tasks.json）" }),
		plan_dir: Type.String({ description: "该 job 的 plan 目录绝对路径（读 task*.md 与 gates/）" }),
	}),
	async execute(_id, params: PipelineGateParams) {
		const { doing_dir, plan_dir } = params;
		const errors: string[] = [];

		// 1. 收集 pending tasks + satisfied 集
		const tasks: GateTask[] = [];
		const satisfied = new Set<string>();
		const tasksJSON = path.join(doing_dir, "tasks.json");
		try {
			const data = JSON.parse(fs.readFileSync(tasksJSON, "utf-8"));
			for (const t of data.tasks ?? []) {
				if (t.status === "success") {
					satisfied.add(t.task_id);
					continue;
				}
				const md = readTaskMD(plan_dir, t.task_id);
				tasks.push({
					id: t.task_id,
					deps: t.dependencies ?? [],
					writeDomain: md ? parseWriteDomainSection(md) : [],
				});
			}
		} catch {
			// 无 tasks.json：回退扫描 plan/task*.md（全部 pending）
			for (const name of (await readDir(plan_dir)).sort()) {
				const m = /^task.+\\.md$/.exec(name);
				if (!m) continue;
				const id = name.replace(/\\.md$/, "");
				const md = readTaskMD(plan_dir, id);
				if (!md) continue;
				tasks.push({ id, deps: parseDepsSection(md), writeDomain: parseWriteDomainSection(md) });
			}
		}
		if (tasks.length === 0) {
			return { content: [{ type: "text", text: "⚠️ 无 pending task（全部已完成或 plan 为空）——无需派发。" }] };
		}

		// 2. 依赖引用存在性
		const known = new Set(tasks.map((t) => t.id));
		for (const t of tasks) {
			for (const d of t.deps) {
				if (!known.has(d) && !satisfied.has(d)) {
					errors.push(`task ${t.id} 依赖不存在的 "${d}"（plan/task*.md 的 # 依赖关系 引用错误）`);
				}
			}
		}

		// 3. Kahn 分层 + 环检测
		const inDeg = new Map<string, number>();
		const dependents = new Map<string, string[]>();
		for (const t of tasks) {
			for (const d of t.deps) {
				if (satisfied.has(d)) continue;
				inDeg.set(t.id, (inDeg.get(t.id) ?? 0) + 1);
				dependents.set(d, [...(dependents.get(d) ?? []), t.id]);
			}
		}
		let level = tasks.filter((t) => (inDeg.get(t.id) ?? 0) === 0).map((t) => t.id).sort();
		const levels: string[][] = [];
		while (level.length > 0) {
			levels.push(level);
			const next: string[] = [];
			for (const cur of level) {
				for (const dep of dependents.get(cur) ?? []) {
					const n = (inDeg.get(dep) ?? 1) - 1;
					inDeg.set(dep, n);
					if (n === 0) next.push(dep);
				}
			}
			level = next.sort();
		}
		if (levels.reduce((a, l) => a + l.length, 0) !== tasks.length) {
			errors.push("cycle detected：依赖关系存在环");
		}

		// 4. 同层写域互斥（多 task 层全员声明 + 两两不相交）
		const domainOf = new Map(tasks.map((t) => [t.id, t.writeDomain]));
		for (const lv of levels) {
			if (lv.length <= 1) continue;
			for (const id of lv) {
				if ((domainOf.get(id) ?? []).length === 0) {
					errors.push(`同层 ${id} 未声明 # 写域（同层多 task 并行的前提）——补 plan/${id}.md 的 # 写域 节`);
				}
			}
			for (let i = 0; i < lv.length; i++) {
				for (let j = i + 1; j < lv.length; j++) {
					for (const a of domainOf.get(lv[i]) ?? []) {
						for (const b of domainOf.get(lv[j]) ?? []) {
							if (writeDomainConflict(a, b)) {
								errors.push(`同层写域冲突：${lv[i]} 的 "${a}" 与 ${lv[j]} 的 "${b}" 重叠——补依赖分层或收敛写域`);
							}
						}
					}
				}
			}
		}

		// 5. gate{N}.py 存在性
		for (let n = 1; n <= levels.length; n++) {
			const gate = path.join(plan_dir, "gates", `gate${n}.py`);
			if (!fs.existsSync(gate)) {
				errors.push(`第 ${n} 层门禁程序缺失：${gate}（plan 阶段应为每层产出 human 确认的 gate{N}.py）`);
			}
		}

		if (errors.length > 0) {
			return {
				content: [{ type: "text", text: `⛔ 流水线结构门禁失败（${errors.length} 项）：\n\n${errors.map((e) => `- ${e}`).join("\n")}\n\n修正 plan 后重新调用 pipeline_gate。` }],
				isError: true,
			};
		}

		const structure = levels.map((lv, i) => `第 ${i + 1} 层：${lv.join(", ")}`).join("\n");
		return {
			content: [{ type: "text", text: `✅ 流水线结构门禁通过（${tasks.length} task / ${levels.length} 层）：\n${structure}\n\n同层写域两两不相交；每层 gate 就位。可开始步骤 ①。` }],
		};
	},
});

function readTaskMD(planDir: string, id: string): string | undefined {
	try {
		return fs.readFileSync(path.join(planDir, `${id}.md`), "utf-8");
	} catch {
		return undefined;
	}
}

export default async function rickGates(pi: ExtensionAPI): Promise<void> {
	pi.registerTool(levelCompleteTool);
	pi.registerTool(pipelineGateTool);
	pi.registerTool(grillingGateTool);
}
