# task17：matmul 模板格手术（225→6）

## 任务目标
J-i3-6：格子从 225 死格族裁到 6 个精选格（BLAS 族为中心——env_7 收尾报告的头号建议）。

## 关键结果
- `super/env_templates/matmul/feature_space.json`：2 轴 × 3 值 = 6 格，以领域知识精选（如 impl_strategy=[ctypes_blas, numpy_einsum, blocked_loops] × memory_layout=[standard, fused_transpose]——具体取值以 domain_knowledge.md 的性能分析为依据选择，确保每格是有潜力的方向而非死格）
- `super/env_templates/matmul/domain_knowledge.md`：对齐新格语义（每格一段实验指引；BLAS 天花板、einsum 优化路径、分块局部性）
- 自检脚本/README 若引用旧格名同步更新
- 验证：venv 中 test_correctness 对 reference/naive/假解三验依旧通过（gate5 复验）

# 写域
- super/env_templates/matmul/**（feature_space.json/domain_knowledge.md/如需 skills/self_check.py、README 引用）

# 依赖关系
- 无（L1 首层并行）

# 测试方法
venv 三验（参考实现过/假解拒/bench 出分）+ `python3 -c "import json; fs=json.load(open('super/env_templates/matmul/feature_space.json')); axes=fs['axes']; print(axes)"` 确认 2 轴且乘积=6

# 禁止事项
- 不动 super/superopt/**（框架代码）
