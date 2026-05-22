package memory

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// openTestStore creates a Store backed by an in-memory SQLite DB.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	return s
}

// ---------- extractWords & wordOverlap ----------

func TestExtractWords_SplitsOnDelimiters(t *testing.T) {
	// delimReplacer replaces ，、。的了是很非常 with spaces, then Fields splits on whitespace.
	// Tokens < 2 runes are dropped.
	words := extractWords("喜欢，周杰伦、五月天")
	if !words["周杰伦"] {
		t.Errorf("expected 周杰伦 in %v", words)
	}
	if !words["五月天"] {
		t.Errorf("expected 五月天 in %v", words)
	}
	if !words["喜欢"] {
		t.Errorf("expected 喜欢 in %v", words)
	}
	// "喜欢，周杰伦" → "喜欢" "周杰伦" (，→space)
	// "、五月天" → "" "五月天" (、→space)
}

func TestExtractWords_DropsSingleRune(t *testing.T) {
	words := extractWords("我听歌")
	// No delimiters present → single token "我听歌" (3 runes) → kept.
	if !words["我听歌"] {
		t.Error("expected 我听歌 (3 runes, kept)")
	}

	words2 := extractWords("的")
	// "的" is a delimiter, replaced with space → empty string after Fields → no tokens.
	if len(words2) != 0 {
		t.Errorf("expected empty, got %v", words2)
	}

	words3 := extractWords("我听，歌")
	// "，" → space → "我听" (2 runes) + "歌" (1 rune, dropped)
	if !words3["我听"] {
		t.Error("expected 我听")
	}
	if words3["歌"] {
		t.Error("single rune 歌 should be dropped")
	}
}

func TestWordOverlap_SharedToken(t *testing.T) {
	// "喜欢，周杰伦" → tokens: {"喜欢": true, "周杰伦": true}
	a := extractWords("喜欢，周杰伦")
	// "周杰伦，不错" → tokens: {"周杰伦": true, "不错": true}
	overlap := wordOverlap(a, "周杰伦，不错")
	// Overlap: "周杰伦" present in both → 1/2 = 0.5
	if overlap != 0.5 {
		t.Errorf("expected overlap 0.5, got %.2f", overlap)
	}
}

func TestWordOverlap_TotalMatch(t *testing.T) {
	a := extractWords("喜欢，周杰伦")
	overlap := wordOverlap(a, "周杰伦 喜欢")
	// Both tokens match → 2/2 = 1.0
	if overlap != 1.0 {
		t.Errorf("expected overlap 1.0, got %.2f", overlap)
	}
}

func TestWordOverlap_NoMatch(t *testing.T) {
	a := extractWords("摇滚，音乐")
	overlap := wordOverlap(a, "周杰伦，歌手")
	if overlap != 0 {
		t.Errorf("expected 0 overlap, got %.2f", overlap)
	}
}

// ---------- AddFact / findSimilarFact / confidence merging ----------

func TestAddFact_New(t *testing.T) {
	s := openTestStore(t)
	err := s.AddFact(catArtistOpinion, "喜欢，周杰伦", 0.7, "extract")
	if err != nil {
		t.Fatalf("AddFact: %v", err)
	}

	facts, _ := s.GetActiveFacts()
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
	if facts[0].Confidence != 0.7 {
		t.Errorf("confidence = %.2f, want 0.7", facts[0].Confidence)
	}
}

func TestAddFact_MergeSimilar(t *testing.T) {
	s := openTestStore(t)
	// "喜欢，周杰伦" → tokens: {"喜欢": true, "周杰伦": true}
	_ = s.AddFact(catArtistOpinion, "喜欢，周杰伦", 0.7, "")
	// "周杰伦，很棒" → tokens: {"周杰伦": true, "很棒": true}
	// Overlap: 1/2 = 0.5 > threshold → merge
	_ = s.AddFact(catArtistOpinion, "周杰伦，很棒", 0.3, "")

	facts, _ := s.GetActiveFacts()
	if len(facts) != 1 {
		t.Fatalf("expected 1 merged fact, got %d", len(facts))
	}
	// 0.7 + 0.3*0.3 = 0.7 + 0.09 = 0.79
	if facts[0].Confidence < 0.78 || facts[0].Confidence > 0.80 {
		t.Errorf("merged confidence = %.2f, want ~0.79", facts[0].Confidence)
	}
}

func TestAddFact_MergeCapAtOne(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catArtistOpinion, "喜欢，周杰伦", 0.9, "")
	// "周杰伦" → single token → overlap 1.0 > 0.5 → merge
	_ = s.AddFact(catArtistOpinion, "周杰伦", 0.5, "")
	// 0.9 + 0.5*0.3 = 1.05, capped at 1.0

	facts, _ := s.GetActiveFacts()
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
	if facts[0].Confidence != 1.0 {
		t.Errorf("confidence = %.2f, want 1.0 (capped)", facts[0].Confidence)
	}
}

func TestAddFact_DifferentCategory_NoMerge(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catArtistOpinion, "喜欢，周杰伦", 0.7, "")
	_ = s.AddFact(catMusicTaste, "喜欢，周杰伦", 0.5, "")

	facts, _ := s.GetActiveFacts()
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts (different categories), got %d", len(facts))
	}
}

func TestAddFact_DifferentArtist_NoMerge(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catArtistOpinion, "喜欢，周杰伦", 0.7, "")
	_ = s.AddFact(catArtistOpinion, "喜欢，五月天", 0.5, "")

	facts, _ := s.GetActiveFacts()
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts (different artists), got %d", len(facts))
	}
}

// ---------- ExpireFact ----------

func TestExpireFact(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catMusicTaste, "喜欢，摇滚", 0.6, "")

	facts, _ := s.GetActiveFacts()
	if len(facts) != 1 {
		t.Fatalf("expected 1 active fact")
	}

	_ = s.ExpireFact(facts[0].ID)

	active, _ := s.GetActiveFacts()
	if len(active) != 0 {
		t.Errorf("expected 0 active facts after expire, got %d", len(active))
	}
}

// ---------- GetProfileSummary ----------

func TestGetProfileSummary_Empty(t *testing.T) {
	s := openTestStore(t)
	summary := s.GetProfileSummary()
	if summary != "" {
		t.Errorf("expected empty summary, got %q", summary)
	}
}

func TestGetProfileSummary_WithFacts(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catArtistOpinion, "用户喜欢，周杰伦，歌曲", 0.8, "")
	_ = s.AddFact(catMusicTaste, "喜欢，摇滚，音乐", 0.7, "")
	_ = s.AddFact(catMoodPattern, "疲惫时，听舒缓，歌曲", 0.4, "")

	summary := s.GetProfileSummary()
	if !strings.Contains(summary, "周杰伦") {
		t.Errorf("summary missing 周杰伦: %s", summary)
	}
	if !strings.Contains(summary, "摇滚") {
		t.Errorf("summary missing 摇滚: %s", summary)
	}
	if !strings.Contains(summary, "【用户画像】") {
		t.Errorf("summary missing header: %s", summary)
	}
}

func TestGetProfileSummary_FiltersLowConfidence(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catArtistOpinion, "微弱，信号", 0.2, "")

	summary := s.GetProfileSummary()
	if summary != "" {
		t.Errorf("expected empty (low confidence filtered), got %q", summary)
	}
}

func TestGetProfileSummary_TopFivePerCategory(t *testing.T) {
	s := openTestStore(t)
	// Add 7 artist_opinion facts with distinct tokens and varying confidence.
	// G (0.60) > F (0.55) > E (0.50) > D (0.45) > C (0.40) > B (0.35) > A (0.30)
	artists := []struct {
		content    string
		confidence float64
	}{
		{"喜欢A", 0.30}, {"喜欢B", 0.35}, {"喜欢C", 0.40},
		{"喜欢D", 0.45}, {"喜欢E", 0.50}, {"喜欢F", 0.55}, {"喜欢G", 0.60},
	}
	for _, a := range artists {
		_ = s.AddFact(catArtistOpinion, a.content, a.confidence, "")
	}

	summary := s.GetProfileSummary()
	// Top 5 by confidence: G(0.60), F(0.55), E(0.50), D(0.45), C(0.40)
	// A(0.30) and B(0.35) should NOT appear (below top 5)
	for _, expectPresent := range []string{"G", "F", "E", "D", "C"} {
		if !strings.Contains(summary, expectPresent) {
			t.Errorf("expected top-5 %s in summary", expectPresent)
		}
	}
	for _, expectAbsent := range []string{"A", "B"} {
		if strings.Contains(summary, expectAbsent) {
			t.Errorf("expected %s NOT in summary (below top 5)", expectAbsent)
		}
	}
}

// ---------- PlayRecord CRUD ----------

func TestInsertAndRecentPlays(t *testing.T) {
	s := openTestStore(t)
	_ = s.InsertPlay(&PlayRecord{
		SongTitle: "七里香", SongArtist: "周杰伦", UserPrompt: "来首周杰伦", SessionID: "s1",
	})
	_ = s.InsertPlay(&PlayRecord{
		SongTitle: "晴天", SongArtist: "周杰伦", UserPrompt: "再来一首", SessionID: "s1",
	})

	plays, err := s.RecentPlays()
	if err != nil {
		t.Fatalf("RecentPlays: %v", err)
	}
	if len(plays) != 2 {
		t.Fatalf("expected 2 plays, got %d", len(plays))
	}
	// Both songs should be present (order depends on timestamp precision, just check presence)
	foundQilixiang, foundQingtian := false, false
	for _, p := range plays {
		if p.SongTitle == "七里香" {
			foundQilixiang = true
		}
		if p.SongTitle == "晴天" {
			foundQingtian = true
		}
	}
	if !foundQilixiang || !foundQingtian {
		t.Errorf("expected both songs, got titles: %v", []string{plays[0].SongTitle, plays[1].SongTitle})
	}
}

func TestSearchPlays(t *testing.T) {
	s := openTestStore(t)
	_ = s.InsertPlay(&PlayRecord{
		SongTitle: "七里香", SongArtist: "周杰伦", UserPrompt: "来首周杰伦", SessionID: "s1",
	})
	_ = s.InsertPlay(&PlayRecord{
		SongTitle: "倔强", SongArtist: "五月天", UserPrompt: "来首五月天", SessionID: "s1",
	})

	results, err := s.SearchPlays("周杰伦", 30)
	if err != nil {
		t.Fatalf("SearchPlays: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].SongTitle != "七里香" {
		t.Errorf("got %s", results[0].SongTitle)
	}
}

// ---------- Conversation CRUD ----------

func TestInsertAndRecentTurns(t *testing.T) {
	s := openTestStore(t)
	_ = s.InsertTurn("s1", "user", "来首周杰伦")
	_ = s.InsertTurn("s1", "assistant", "播放了：周杰伦-七里香")

	turns, err := s.RecentTurns("s1", 10)
	if err != nil {
		t.Fatalf("RecentTurns: %v", err)
	}
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(turns))
	}
	if turns[0].Role != "user" {
		t.Error("first turn should be user")
	}
	if turns[1].Role != "assistant" {
		t.Error("second turn should be assistant")
	}
}

func TestRecentTurns_LimitRespected(t *testing.T) {
	s := openTestStore(t)
	for i := 0; i < 15; i++ {
		_ = s.InsertTurn("s1", "user", "msg")
	}

	turns, _ := s.RecentTurns("s1", 5)
	if len(turns) != 5 {
		t.Errorf("expected 5 turns, got %d", len(turns))
	}
}

func TestTrimOldTurns(t *testing.T) {
	s := openTestStore(t)
	for i := 0; i < 25; i++ {
		_ = s.InsertTurn("s1", "user", "msg")
	}

	_ = s.TrimOldTurns("s1", 20)

	turns, _ := s.RecentTurns("s1", 30)
	if len(turns) > 20 {
		t.Errorf("expected at most 20 turns after trim, got %d", len(turns))
	}
}

func TestTrimOldTurns_UnderLimit(t *testing.T) {
	s := openTestStore(t)
	for i := 0; i < 5; i++ {
		_ = s.InsertTurn("s1", "user", "msg")
	}

	err := s.TrimOldTurns("s1", 20)
	if err != nil {
		t.Fatalf("TrimOldTurns: %v", err)
	}

	turns, _ := s.RecentTurns("s1", 30)
	if len(turns) != 5 {
		t.Errorf("expected 5 turns (under limit, no trim), got %d", len(turns))
	}
}

// ---------- Preprocess ----------

func TestPreprocess_EmptyStore(t *testing.T) {
	s := openTestStore(t)
	vals := s.Preprocess("s1")

	if vals["UserProfile"] != "" {
		t.Errorf("empty UserProfile expected, got %q", vals["UserProfile"])
	}
	if vals["RecentPlays"] != "暂无播放记录" {
		t.Errorf("expected 暂无播放记录, got %q", vals["RecentPlays"])
	}
	if vals["ConversationContext"] != "" {
		t.Errorf("empty ConversationContext expected, got %q", vals["ConversationContext"])
	}
}

func TestPreprocess_WithData(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catArtistOpinion, "喜欢，周杰伦", 0.7, "")
	_ = s.InsertPlay(&PlayRecord{
		SongTitle: "七里香", SongArtist: "周杰伦", UserPrompt: "来首周杰伦", SessionID: "s1",
	})
	_ = s.InsertTurn("s1", "user", "来首周杰伦")
	_ = s.InsertTurn("s1", "assistant", "播放了：周杰伦-七里香")

	vals := s.Preprocess("s1")

	profile, _ := vals["UserProfile"].(string)
	if !strings.Contains(profile, "周杰伦") {
		t.Errorf("UserProfile missing 周杰伦: %s", profile)
	}

	plays, _ := vals["RecentPlays"].(string)
	if !strings.Contains(plays, "七里香") {
		t.Errorf("RecentPlays missing 七里香: %s", plays)
	}

	ctx, _ := vals["ConversationContext"].(string)
	if !strings.Contains(ctx, "来首周杰伦") {
		t.Errorf("ConversationContext missing user msg: %s", ctx)
	}
}

// ---------- Postprocess ----------

func TestPostprocess_MusicResponse(t *testing.T) {
	s := openTestStore(t)
	agentOutput := `[{"title":"七里香","artist":"周杰伦","segue":"来听周杰伦"},{"title":"晴天","artist":"周杰伦"}]`
	s.Postprocess("s1", "来首周杰伦", agentOutput)

	// Should have created 2 play records
	plays, _ := s.RecentPlays()
	if len(plays) != 2 {
		t.Fatalf("expected 2 plays, got %d", len(plays))
	}
	// Just check both songs exist (order may vary with same-ms timestamps in SQLite)
	titles := map[string]bool{}
	for _, p := range plays {
		titles[p.SongTitle] = true
	}
	if !titles["七里香"] || !titles["晴天"] {
		t.Errorf("expected both 七里香 and 晴天, got %v", titles)
	}

	// Should have created conversation turns (assistant + user)
	turns, _ := s.RecentTurns("s1", 10)
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(turns))
	}
	if turns[0].Role != "assistant" {
		t.Errorf("first turn should be assistant, got %s", turns[0].Role)
	}
	if !strings.Contains(turns[0].Content, "七里香") {
		t.Errorf("assistant turn missing song: %s", turns[0].Content)
	}
}

func TestPostprocess_ChatResponse(t *testing.T) {
	s := openTestStore(t)
	s.Postprocess("s1", "我喜欢周杰伦", "周杰伦确实很棒！他的音乐影响了整个华语乐坛。")

	plays, _ := s.RecentPlays()
	if len(plays) != 0 {
		t.Errorf("expected 0 plays for chat, got %d", len(plays))
	}

	turns, _ := s.RecentTurns("s1", 10)
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(turns))
	}
	if turns[0].Role != "assistant" {
		t.Error("first turn should be assistant")
	}
}

func TestPostprocess_EmptySession_Skipped(t *testing.T) {
	s := openTestStore(t)
	s.Postprocess("", "hello", `[{"title":"test","artist":"test"}]`)

	plays, _ := s.RecentPlays()
	if len(plays) != 0 {
		t.Errorf("empty session should skip, got %d plays", len(plays))
	}
}

// ---------- RecordPreferences (backward compat) ----------

func TestRecordPreferences(t *testing.T) {
	s := openTestStore(t)
	s.RecordPreferences("周杰伦", "流行", "")

	facts, _ := s.GetActiveFacts()
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(facts))
	}
}

// ---------- RecallMemory Tool ----------

func TestRecallMemoryTool(t *testing.T) {
	s := openTestStore(t)
	_ = s.InsertPlay(&PlayRecord{
		SongTitle: "七里香", SongArtist: "周杰伦", UserPrompt: "来首周杰伦", SessionID: "s1",
	})
	_ = s.InsertPlay(&PlayRecord{
		SongTitle: "晴天", SongArtist: "周杰伦", UserPrompt: "再来一首", SessionID: "s1",
	})
	_ = s.InsertPlay(&PlayRecord{
		SongTitle: "倔强", SongArtist: "五月天", UserPrompt: "五月天", SessionID: "s1",
	})

	tool := NewRecallMemoryTool(s)
	if tool == nil {
		t.Fatal("expected non-nil tool")
	}

	jsonStr, err := tool.InvokableRun(t.Context(), `{"keyword":"周杰伦","days":30}`)
	if err != nil {
		t.Fatalf("tool InvokableRun: %v", err)
	}
	if !strings.Contains(jsonStr, "七里香") {
		t.Errorf("result missing 七里香: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "倔强") {
		t.Errorf("result should not contain 倔强 (五月天): %s", jsonStr)
	}
}

// ---------- Open with file ----------

func TestOpen_FileDB(t *testing.T) {
	path := t.TempDir() + "/test.db"
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer os.Remove(path)

	_ = s.AddFact(catMusicTaste, "测试", 0.5, "")
	facts, _ := s.GetActiveFacts()
	if len(facts) != 1 {
		t.Errorf("expected 1 fact, got %d", len(facts))
	}
}

// ---------- truncate ----------

func TestTruncate_Short(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Errorf("got %q", result)
	}
}

// ---------- Negative feedback penalty ----------

func TestAddFact_NegativeSignal_PenalizesPositive(t *testing.T) {
	s := openTestStore(t)
	// Create a positive fact first.
	_ = s.AddFact(catArtistOpinion, "喜欢，崔健", 0.7, "")

	// Negative signal about same artist.
	_ = s.AddFact(catArtistOpinion, "不喜欢，崔健，风格", 0.2, "")

	facts, _ := s.GetActiveFacts()
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact (penalized but still >=0.3), got %d", len(facts))
	}
	// 0.7 - 0.2 = 0.5
	if facts[0].Confidence < 0.49 || facts[0].Confidence > 0.51 {
		t.Errorf("penalized confidence = %.2f, want 0.5", facts[0].Confidence)
	}
}

func TestAddFact_NegativeSignal_ExpiresBelowThreshold(t *testing.T) {
	s := openTestStore(t)
	// Create a weak positive fact.
	_ = s.AddFact(catArtistOpinion, "喜欢，崔健", 0.45, "")

	// Negative signal — should drop confidence to 0.25 → expire.
	_ = s.AddFact(catArtistOpinion, "不喜欢，崔健，风格", 0.2, "")

	facts, _ := s.GetActiveFacts()
	if len(facts) != 0 {
		t.Errorf("expected 0 active facts (expired due to penalty), got %d", len(facts))
	}
}

func TestAddFact_NegativeSignal_NoMatch_NoEffect(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catArtistOpinion, "喜欢，周杰伦", 0.7, "")

	// Negative signal about different artist — no overlap → no penalty.
	_ = s.AddFact(catArtistOpinion, "不喜欢，五月天", 0.2, "")

	facts, _ := s.GetActiveFacts()
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact (no match, no penalty), got %d", len(facts))
	}
	if facts[0].Confidence != 0.7 {
		t.Errorf("confidence unchanged = %.2f, want 0.7", facts[0].Confidence)
	}
}

// ---------- LLM output validation ----------

func TestValidateFact_ArtistOpinion_Valid(t *testing.T) {
	if !validateFact(catArtistOpinion, "喜欢，周杰伦，歌曲", "来首周杰伦", "") {
		t.Error("should pass: 周杰伦 appears in userPrompt")
	}
}

func TestValidateFact_ArtistOpinion_FromAgentContext(t *testing.T) {
	// User said "再来一首类似的", agent played 周杰伦
	if !validateFact(catArtistOpinion, "喜欢，周杰伦，歌曲", "再来一首类似的", "周杰伦 - 晴天") {
		t.Error("should pass: 周杰伦 appears in agent context")
	}
}

func TestValidateFact_ArtistOpinion_Invalid(t *testing.T) {
	if validateFact(catArtistOpinion, "喜欢，李宗盛，歌曲", "今天好累放点歌", "久石让 - 天空之城") {
		t.Error("should fail: 李宗盛 not in userPrompt or agent context")
	}
}

func TestValidateFact_MusicTaste_Valid(t *testing.T) {
	if !validateFact(catMusicTaste, "喜欢，摇滚，音乐", "来点摇滚", "") {
		t.Error("should pass: 摇滚 appears in userPrompt")
	}
}

func TestValidateFact_MoodPattern_AlwaysPass(t *testing.T) {
	if !validateFact(catMoodPattern, "疲惫时，听舒缓", "some unrelated text", "") {
		t.Error("mood_pattern should always pass")
	}
}

// ---------- Time decay ----------

func TestDecayFactor_Recent(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catArtistOpinion, "喜欢，周杰伦", 0.8, "")
	facts, _ := s.GetActiveFacts()
	d := s.decayFactor(facts[0])
	if d != 1.0 {
		t.Errorf("recent fact decay = %.2f, want 1.0", d)
	}
}

// ---------- ConsolidateFacts ----------

func TestConsolidateFacts_BelowThreshold(t *testing.T) {
	s := openTestStore(t)
	_ = s.AddFact(catArtistOpinion, "喜欢，周杰伦", 0.7, "")

	llm := &mockChatModel{responses: []string{`{"facts":[]}`}}
	err := s.ConsolidateFacts(t.Context(), llm)
	if err != nil {
		t.Fatalf("ConsolidateFacts: %v", err)
	}
	// LLM should not be called (facts <= 15), callCount stays 0.
	if llm.callCount != 0 {
		t.Errorf("expected 0 LLM calls (below threshold), got %d", llm.callCount)
	}
}

func TestConsolidateFacts_MergesDuplicates(t *testing.T) {
	s := openTestStore(t)
	// Add 16 facts to trigger consolidation.
	for i := 0; i < 16; i++ {
		_ = s.AddFact(catArtistOpinion, fmt.Sprintf("喜欢，歌手%d", i), 0.5, "")
	}

	// LLM returns only 5 facts — the rest should be expired.
	llm := &mockChatModel{
		responses: []string{
			`{"facts":[
				{"category":"artist_opinion","content":"喜欢，歌手0","confidence":0.6},
				{"category":"artist_opinion","content":"喜欢，歌手1","confidence":0.5},
				{"category":"artist_opinion","content":"合并后新事实","confidence":0.7}
			]}`,
		},
	}
	err := s.ConsolidateFacts(t.Context(), llm)
	if err != nil {
		t.Fatalf("ConsolidateFacts: %v", err)
	}
	if llm.callCount != 1 {
		t.Errorf("expected 1 LLM call, got %d", llm.callCount)
	}
	// Original fact "歌手0" should be kept (confidence updated to 0.6).
	// Original fact "歌手1" should be kept.
	// "合并后新事实" should be added.
	// Other 14 facts should be expired.
	facts, _ := s.GetActiveFacts()
	if len(facts) != 3 {
		t.Errorf("expected 3 facts after consolidation, got %d", len(facts))
	}
}

func TestConsolidateFacts_NilModel(t *testing.T) {
	s := openTestStore(t)
	err := s.ConsolidateFacts(t.Context(), nil)
	if err != nil {
		t.Errorf("nil model should return nil, got %v", err)
	}
}

func TestTruncate_Long(t *testing.T) {
	result := truncate("hello世界test", 5)
	if !strings.HasSuffix(result, "...") {
		t.Errorf("expected ... suffix, got %q", result)
	}
	r := []rune(result)
	if len(r) != 8 {
		t.Errorf("expected 8 runes, got %d in %q", len(r), result)
	}
}
