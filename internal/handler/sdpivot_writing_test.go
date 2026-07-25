package handler

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestResolveSDPivotWritingTemplate(t *testing.T) {
	tests := map[string]string{
		"notice":          "notice",
		"technical":       "technical",
		"tech_doc":        "technical",
		"report":          "report",
		"training":        "training",
		"minutes":         "minutes",
		"meeting_minutes": "minutes",
		"proposal":        "proposal",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			template, ok := resolveSDPivotWritingTemplate(input)
			if !ok {
				t.Fatalf("expected category %q to resolve", input)
			}
			if template.ID != want {
				t.Fatalf("resolved category = %q, want %q", template.ID, want)
			}
		})
	}
	if _, ok := resolveSDPivotWritingTemplate("unknown"); ok {
		t.Fatal("expected unknown category to be rejected")
	}
}

func TestBuildSDPivotWritingPromptPriority(t *testing.T) {
	template, _ := resolveSDPivotWritingTemplate("proposal")
	prompt := buildSDPivotWritingPrompt(
		template,
		"编写迁移方案",
		"knowledge_base",
		false,
		[]sdPivotWritingExample{{DocumentID: "doc-1", Title: "既有方案", Tags: "迁移,数据库", Content: "既有章节风格"}},
		"增加审批章节",
		[]types.SDPivotDocumentChunk{{DocumentID: "doc-2", Content: "迁移窗口为两小时"}},
		nil,
	)

	sections := []string{"1. 系统内置模板", "2. 同标签文档自学习", "3. 用户自定义模板", "知识来源:"}
	last := -1
	for _, section := range sections {
		index := strings.Index(prompt, section)
		if index <= last {
			t.Fatalf("section %q missing or out of order in prompt:\n%s", section, prompt)
		}
		last = index
	}
	for _, expected := range []string{"既有章节风格", "增加审批章节", "迁移窗口为两小时", "内置模板和同标签文档风格"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func TestNormalizeWritingTags(t *testing.T) {
	got := normalizeWritingTags(" 迁移，数据库;迁移| 运维 \n")
	want := []string{"迁移", "数据库", "运维"}
	if len(got) != len(want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tags = %#v, want %#v", got, want)
		}
	}
}
