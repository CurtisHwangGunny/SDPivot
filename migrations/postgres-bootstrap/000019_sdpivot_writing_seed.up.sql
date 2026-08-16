CREATE TABLE IF NOT EXISTS writing_category (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id   BIGINT NOT NULL,
    name        VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort        INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_writing_category_tenant_sort
    ON writing_category (tenant_id, sort, created_at);

CREATE TABLE IF NOT EXISTS writing_template (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    tenant_id   BIGINT NOT NULL,
    category_id VARCHAR(36) NOT NULL REFERENCES writing_category(id) ON DELETE RESTRICT,
    name        VARCHAR(100) NOT NULL,
    content     TEXT NOT NULL,
    is_builtin  BOOLEAN NOT NULL DEFAULT FALSE,
    sort        INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, category_id, name)
);

CREATE INDEX IF NOT EXISTS idx_writing_template_tenant_category_sort
    ON writing_template (tenant_id, category_id, sort, created_at);

ALTER TABLE writing_category ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_writing_category_tenant ON writing_category;
CREATE POLICY sdpivot_writing_category_tenant ON writing_category
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

ALTER TABLE writing_template ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sdpivot_writing_template_tenant ON writing_template;
CREATE POLICY sdpivot_writing_template_tenant ON writing_template
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

INSERT INTO writing_category (id, tenant_id, name, description, sort, created_at, updated_at)
VALUES
    ('a1000000-0000-4000-8000-000000000001', 1, '通知公告', '用于撰写各类通知、公告和公示材料。', 1, NOW(), NOW()),
    ('a1000000-0000-4000-8000-000000000002', 1, '技术文档', '用于撰写技术说明、操作手册和研发文档。', 2, NOW(), NOW()),
    ('a1000000-0000-4000-8000-000000000003', 1, '工作报告', '用于撰写周报、月报、总结和汇报材料。', 3, NOW(), NOW()),
    ('a1000000-0000-4000-8000-000000000004', 1, '培训材料', '用于撰写培训讲义、课程大纲和学习材料。', 4, NOW(), NOW()),
    ('a1000000-0000-4000-8000-000000000005', 1, '会议纪要', '用于整理会议议题、讨论结论和待办事项。', 5, NOW(), NOW()),
    ('a1000000-0000-4000-8000-000000000006', 1, '方案建议书', '用于撰写项目方案、实施建议和决策材料。', 6, NOW(), NOW())
ON CONFLICT (tenant_id, name) DO NOTHING;

INSERT INTO writing_template (id, tenant_id, category_id, name, content, is_builtin, sort, created_at, updated_at)
SELECT seed.id, 1, category.id, seed.name, seed.content, TRUE, seed.sort, NOW(), NOW()
FROM (VALUES
    ('b1000000-0000-4000-8000-000000000001', '通知公告', '通用通知', E'标题：关于【事项】的通知\n\n各相关单位/人员：\n【通知正文】\n\n一、时间与地点\n【具体安排】\n\n二、有关要求\n【相关要求】\n\n特此通知。', 1),
    ('b1000000-0000-4000-8000-000000000002', '技术文档', '技术说明文档', E'# 【系统或功能名称】\n\n## 1. 背景与目标\n【背景及目标】\n\n## 2. 技术方案\n【架构、流程和关键设计】\n\n## 3. 使用说明\n【操作步骤】\n\n## 4. 注意事项\n【限制、风险和常见问题】', 1),
    ('b1000000-0000-4000-8000-000000000003', '工作报告', '阶段工作报告', E'# 【时间范围】工作报告\n\n## 一、工作概述\n【总体情况】\n\n## 二、重点工作及成果\n【完成事项与成果】\n\n## 三、问题与风险\n【当前问题及应对措施】\n\n## 四、下一步计划\n【后续安排】', 1),
    ('b1000000-0000-4000-8000-000000000004', '培训材料', '培训课程大纲', E'# 【培训主题】\n\n## 一、培训目标\n【培训目标】\n\n## 二、培训对象与时间\n【对象、时间和地点】\n\n## 三、课程内容\n【课程模块及要点】\n\n## 四、考核与反馈\n【考核方式和反馈安排】', 1),
    ('b1000000-0000-4000-8000-000000000005', '会议纪要', '标准会议纪要', E'# 【会议名称】会议纪要\n\n会议时间：【时间】\n会议地点：【地点】\n参会人员：【人员】\n主持人：【主持人】\n记录人：【记录人】\n\n## 一、会议议题\n【议题列表】\n\n## 二、讨论与结论\n【讨论要点及结论】\n\n## 三、待办事项\n【责任人、任务和完成时限】', 1),
    ('b1000000-0000-4000-8000-000000000006', '方案建议书', '项目方案建议书', E'# 【项目名称】方案建议书\n\n## 一、项目背景\n【现状与需求】\n\n## 二、建设目标\n【总体目标与具体目标】\n\n## 三、实施方案\n【工作内容、技术路线和进度安排】\n\n## 四、资源与预算\n【人员、资源和预算】\n\n## 五、风险与保障措施\n【风险分析及保障措施】', 1)
) AS seed(id, category_name, name, content, sort)
JOIN writing_category category
    ON category.tenant_id = 1 AND category.name = seed.category_name
ON CONFLICT (tenant_id, category_id, name) DO NOTHING;
