package memory

const maxConversationTurns = 20

// InsertTurn records a conversation turn.
func (s *Store) InsertTurn(sessionID, role, content string) error {
	return s.DB.Create(&ConversationTurn{
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	}).Error
}

// RecentTurns returns the last N conversation turns in chronological order.
func (s *Store) RecentTurns(sessionID string, limit int) ([]ConversationTurn, error) {
	var turns []ConversationTurn
	err := s.DB.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&turns).Error
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
		turns[i], turns[j] = turns[j], turns[i]
	}
	return turns, nil
}

// TrimOldTurns keeps only the most recent maxTurns for a session.
func (s *Store) TrimOldTurns(sessionID string, maxTurns int) error {
	var count int64
	if err := s.DB.Model(&ConversationTurn{}).
		Where("session_id = ?", sessionID).
		Count(&count).Error; err != nil {
		return err
	}
	if count <= int64(maxTurns) {
		return nil
	}

	var turns []ConversationTurn
	if err := s.DB.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(maxTurns).
		Find(&turns).Error; err != nil {
		return err
	}
	if len(turns) == 0 {
		return nil
	}

	oldestToKeep := turns[len(turns)-1].ID
	return s.DB.Where("session_id = ? AND id < ?", sessionID, oldestToKeep).
		Delete(&ConversationTurn{}).Error
}
