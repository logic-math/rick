#!/usr/bin/env python3
import json
import sys
import os
import subprocess

def main():
    errors = []

    # Project root: tests/ -> doing/ -> job_5/ -> jobs/ -> .rick/ -> project root
    project_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))))
    template_file = os.path.join(project_root, 'internal', 'prompt', 'templates', 'learning.md')

    print(f"Project root: {project_root}", file=sys.stderr)
    print(f"Template file: {template_file}", file=sys.stderr)

    # Test 1: Check file exists
    if not os.path.exists(template_file):
        errors.append(f'learning.md template does not exist at {template_file}')
        result = {'pass': False, 'errors': errors}
        print(json.dumps(result))
        sys.exit(1)

    try:
        with open(template_file, 'r') as f:
            content = f.read()
    except Exception as e:
        errors.append(f'Failed to read learning.md: {str(e)}')
        result = {'pass': False, 'errors': errors}
        print(json.dumps(result))
        sys.exit(1)

    # Test 2: Check four output types directory structure (wiki/, skills/, OKR.md, SPEC.md)
    if 'wiki/' not in content:
        errors.append('learning.md missing wiki/ directory reference')
    if 'skills/' not in content:
        errors.append('learning.md missing skills/ directory reference')
    if 'OKR.md' not in content:
        errors.append('learning.md missing OKR.md reference')
    if 'SPEC.md' not in content:
        errors.append('learning.md missing SPEC.md reference')

    # Test 3: Check Wiki audience description (humans, system operation principles and control methods)
    wiki_human_indicators = ['人类', '给人类', '受众']
    if not any(indicator in content for indicator in wiki_human_indicators):
        errors.append('learning.md missing Wiki audience description (should mention 人类/受众)')

    wiki_content_indicators = ['系统运行原理', '控制方法', '运行原理']
    if not any(indicator in content for indicator in wiki_content_indicators):
        errors.append('learning.md missing Wiki content description (should mention 系统运行原理 or 控制方法)')

    # Test 4: Check Wiki writing requirements (overview/principles/control/examples + Mermaid diagrams)
    wiki_structure_indicators = ['概述', '工作原理', '如何控制', '示例']
    missing_wiki_structure = [ind for ind in wiki_structure_indicators if ind not in content]
    if missing_wiki_structure:
        errors.append(f'learning.md missing Wiki writing requirements: {missing_wiki_structure}')

    if 'mermaid' not in content.lower() and 'Mermaid' not in content:
        errors.append('learning.md missing Mermaid diagram requirement for Wiki')

    # Test 5: Check skills evolution four-step process
    skill_steps = ['定义目标', 'GitHub搜索', '组合评估', '实现决策']
    missing_skill_steps = [step for step in skill_steps if step not in content]
    if missing_skill_steps:
        errors.append(f'learning.md missing skill evolution steps: {missing_skill_steps}')

    # Test 6: Check OKR/SPEC update spec (complete version, top comment explaining changes)
    okr_spec_indicators = ['完整', '顶部注释', '变更']
    missing_okr_spec = [ind for ind in okr_spec_indicators if ind not in content]
    if missing_okr_spec:
        errors.append(f'learning.md missing OKR/SPEC update spec indicators: {missing_okr_spec}')

    # Test 7: Check no mention of old format OKR_UPDATE.md / SPEC_UPDATE.md
    if 'OKR_UPDATE.md' in content:
        errors.append('learning.md still mentions old format OKR_UPDATE.md (should be removed)')
    if 'SPEC_UPDATE.md' in content:
        errors.append('learning.md still mentions old format SPEC_UPDATE.md (should be removed)')

    # Test 8: Check AI agent five-step workflow
    # Looking for: analyze -> output -> check -> merge -> review loop
    workflow_indicators = ['分析', '产出', 'check', 'merge', '审查']
    missing_workflow = [ind for ind in workflow_indicators if ind not in content]
    if missing_workflow:
        errors.append(f'learning.md missing AI agent workflow indicators: {missing_workflow}')

    # Test 9: Check Step 5 is a loop (human rejects -> improvement -> modify -> re-diff -> confirm again)
    loop_indicators = ['循环', '拒绝', '改进', '修改']
    missing_loop = [ind for ind in loop_indicators if ind not in content]
    if len(missing_loop) > 2:
        errors.append(f'learning.md missing Step 5 loop description indicators: {missing_loop}')

    # Test 10: Run go build to verify embedded templates compile correctly
    try:
        result_build = subprocess.run(
            ['go', 'build', './...'],
            cwd=project_root,
            capture_output=True,
            text=True,
            timeout=60
        )
        if result_build.returncode != 0:
            errors.append(f'go build failed: {result_build.stderr}')
    except subprocess.TimeoutExpired:
        errors.append('go build timed out after 60 seconds')
    except Exception as e:
        errors.append(f'Failed to run go build: {str(e)}')

    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    print(json.dumps(result))
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
