package dto

// CreateDialogInput представляет тело запроса для создания диалога
// @Description Структура запроса для создания личного или группового чата
// @Description creator_id должен быть одним из участников
// @Description participant_ids - список user_id, включая creator
// @Description is_group = true означает групповой чат, иначе личный
// @Description title используется только для группового чата
// @Example {"creator_id":"123", "participant_ids":["123","456"], "title":"Рабочий чат", "is_group":true}
type CreateDialogInput struct {
	CreatorID      string   `json:"creator_id" binding:"required"`
	ParticipantIDs []string `json:"participant_ids" binding:"required"`
	Title          *string  `json:"title"` // nullable — только для групп
	IsGroup        bool     `json:"is_group"`
}

// SendMessageInput представляет тело запроса для отправки сообщения
// @Description Используется при отправке сообщения, включая вложения и цитирование
// @Example {"dialog_id":"abc123", "sender_id":"user123", "content":"Привет!", "attachment_ids":[], "reply_to_id":null, "forward_from":null}
type SendMessageInput struct {
	DialogID      string   `json:"dialog_id" binding:"required"`
	SenderID      string   `json:"sender_id" binding:"required"`
	Content       string   `json:"content"`
	ReplyToID     *string  `json:"reply_to_id"`
	ForwardFrom   *string  `json:"forward_from"`
	AttachmentIDs []string `json:"attachment_ids"`
}
