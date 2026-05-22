package memory

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// mockChatModel returns predefined JSON for ExtractFacts per round.
type mockChatModel struct {
	responses []string
	callCount int
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.callCount >= len(m.responses) {
		return &schema.Message{Content: `{"facts":[]}`}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return &schema.Message{Content: resp}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("not implemented")
}

// ---------- 多轮对话集成测试：10 轮 ----------

func TestMultiRoundMemory_TenRounds(t *testing.T) {
	s := openTestStore(t)
	sessionID := "sess_abc"

	llm := &mockChatModel{
		responses: []string{
			// R1: 显式点名 → artist_opinion 高置信度
			`{"facts":[{"category":"artist_opinion","content":"周杰伦，好听","confidence":0.7}]}`,
			// R2: 再来一首 → 强化已有事实
			`{"facts":[{"category":"artist_opinion","content":"周杰伦，好听","confidence":0.3}]}`,
			// R3: 心情 → mood_pattern 仅此，不提取李宗盛
			`{"facts":[{"category":"mood_pattern","content":"疲惫时，舒缓，叙事","confidence":0.4}]}`,
			// R4: 显式点摇滚 → music_taste
			`{"facts":[{"category":"music_taste","content":"摇滚，不错","confidence":0.7}]}`,
			// R5: "换一首" → 结合 LastPlayed（崔健-一无所有）→ 负向，低置信度
			`{"facts":[{"category":"artist_opinion","content":"崔健，不好听","confidence":0.2}]}`,
			// R6: 闲聊中表达偏好 → 视同显式提取
			`{"facts":[{"category":"artist_opinion","content":"五月天，很棒","confidence":0.7}]}`,
			// R7: 闲聊查历史，无偏好表达 → 空
			`{"facts":[]}`,
			// R8: 生活场景 → mood_pattern（一次不算习惯）
			`{"facts":[{"category":"mood_pattern","content":"周末早晨，轻音乐","confidence":0.5}]}`,
			// R9: "再来一首" 不带条件 → 空
			`{"facts":[]}`,
			// R10: 再次点名周杰伦 → 继续强化
			`{"facts":[{"category":"artist_opinion","content":"周杰伦，好听","confidence":0.3}]}`,
		},
	}

	type round struct {
		prompt      string
		agentOutput string
		desc        string
	}
	rounds := []round{
		{"来首周杰伦", `[{"title":"七里香","artist":"周杰伦","segue":"来听一首周杰伦的经典"}]`, "显式点歌"},
		{"再来一首类似的", `[{"title":"晴天","artist":"周杰伦"}]`, "指代强化"},
		{"今天好累，放点歌吧", `[{"title":"山丘","artist":"李宗盛"}]`, "心情推断"},
		{"来点摇滚", `[{"title":"一无所有","artist":"崔健"}]`, "探索新风格"},
		{"换一首", `[{"title":"新长征路上的摇滚","artist":"崔健"}]`, "负向反馈"},
		{"我好喜欢五月天", "五月天确实是华语乐坛很棒的乐队！他们的歌充满青春回忆。", "闲聊偏好"},
		{"上次那个民谣歌手是谁来着", "您上次听的是老狼的《同桌的你》。", "回溯查询"},
		{"周末早上来点轻音乐", `[{"title":"天空之城","artist":"久石让"}]`, "场景习惯"},
		{"再来一首", `[{"title":"菊次郎的夏天","artist":"久石让"}]`, "无条件续播"},
		{"还是周杰伦吧", `[{"title":"稻香","artist":"周杰伦"}]`, "回归偏好"},
	}

	for i, rd := range rounds {
		roundNum := i + 1
		t.Logf("\n━━━ 第 %d 轮：%s ━━━", roundNum, rd.desc)
		t.Logf("用户: %s", rd.prompt)
		t.Logf("助手: %s", truncate(rd.agentOutput, 80))

		s.Postprocess(sessionID, rd.prompt, rd.agentOutput)

		if err := s.ExtractFacts(t.Context(), llm, rd.prompt, rd.agentOutput); err != nil {
			t.Fatalf("round %d ExtractFacts: %v", roundNum, err)
		}

		facts, _ := s.GetActiveFacts()
		t.Logf("--- 累积事实 (%d 条) ---", len(facts))
		for _, f := range facts {
			bar := strings.Repeat("█", int(f.Confidence*10))
			t.Logf("  [%s] %s  %.0f%% %s", f.Category, f.Content, f.Confidence*100, bar)
		}

		plays, _ := s.RecentPlays()
		t.Logf("--- 播放记录 (%d 条) ---", len(plays))
		for j, p := range plays {
			if j >= 5 {
				t.Logf("  ... 还有 %d 条", len(plays)-5)
				break
			}
			t.Logf("  %s - %s", p.SongArtist, p.SongTitle)
		}
	}

	// ────────── 终态断言 ──────────

	facts, _ := s.GetActiveFacts()

	// 周杰伦：R1(0.7) + R2(0.3→0.79) + R10(0.3→0.88)
	var jayFact *MemoryFact
	var maydayFact *MemoryFact
	var rockFact *MemoryFact
	var moodFact *MemoryFact
	for i := range facts {
		switch {
		case facts[i].Category == catArtistOpinion && strings.Contains(facts[i].Content, "周杰伦"):
			jayFact = &facts[i]
		case facts[i].Category == catArtistOpinion && strings.Contains(facts[i].Content, "五月天"):
			maydayFact = &facts[i]
		case facts[i].Category == catMusicTaste:
			rockFact = &facts[i]
		case facts[i].Category == catMoodPattern:
			moodFact = &facts[i]
		}
	}

	if jayFact == nil {
		t.Error("missing 周杰伦 artist_opinion")
	} else if jayFact.Confidence < 0.87 || jayFact.Confidence > 0.89 {
		t.Errorf("周杰伦 confidence = %.2f, want ~0.88 (0.7→0.79→0.88)", jayFact.Confidence)
	}

	if maydayFact == nil {
		t.Error("missing 五月天 artist_opinion")
	} else if maydayFact.Confidence != 0.7 {
		t.Errorf("五月天 confidence = %.2f, want 0.7", maydayFact.Confidence)
	}

	if rockFact == nil {
		t.Error("missing rock music_taste")
	} else if rockFact.Confidence != 0.7 {
		t.Errorf("rock confidence = %.2f, want 0.7", rockFact.Confidence)
	}

	if moodFact == nil {
		t.Error("missing mood_pattern")
	}


	// R6(闲聊) + R7(查询) should NOT create play records — 10 rounds but only 8 are music.
	plays, _ := s.RecentPlays()
	musicRounds := 0
	for _, rd := range rounds {
		if strings.HasPrefix(strings.TrimSpace(rd.agentOutput), "[") {
			musicRounds += len(strings.Split(rd.agentOutput, `"title"`)) - 1
		}
	}
	_ = musicRounds
	if len(plays) < 8 {
		t.Errorf("expected at least 8 plays (8 music rounds), got %d", len(plays))
	}

	// Conversation turns: 10 rounds × 2 = 20, just at TrimOldTurns limit (20).
	turns, _ := s.RecentTurns(sessionID, 30)
	if len(turns) < 18 {
		t.Errorf("expected ~20 turns, got %d", len(turns))
	}

	// Profile summary should surface top items.
	summary := s.GetProfileSummary()
	t.Logf("\n━━━ 最终画像 ━━━\n%s", summary)
	if !strings.Contains(summary, "周杰伦") {
		t.Error("summary missing 周杰伦")
	}
	if !strings.Contains(summary, "五月天") {
		t.Error("summary missing 五月天")
	}
	if !strings.Contains(summary, "摇滚") {
		t.Error("summary missing 摇滚")
	}
	// 负向低置信度 (<0.3) 不应该出现在画像中
	if strings.Contains(summary, "崔健") {
		t.Error("summary should NOT contain negative low-confidence fact about 崔健")
	}
}
