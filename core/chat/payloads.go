package chat

import "time"

type WebSocketMessage struct {
	Type string `json:"Type"`
	Data interface{} `json:"Data"`
}

type ChatMessageData struct {
	FullName string `json:"FullName"`
	Message string `json:"Message"`
	SentAt time.Time `json:"SentAt"`
}

type AttendanceData struct {
	FullName string `json:"FullName"`
	SentAt time.Time `json:"SentAt"`
}

func CreateChatMessageData(fullName string, message string, sentAt time.Time) *ChatMessageData {
	return &ChatMessageData {
		FullName: fullName,
		Message: message,
		SentAt: sentAt,
	}
}

func CreateWebSocketMessage(eventType string, data interface{}) *WebSocketMessage {
	return &WebSocketMessage {
		Type: eventType,
		Data: data,
	}
}

func CreateWebSocketMessageWithEventChatMessage(chatMessageData *ChatMessageData) *WebSocketMessage {
	return CreateWebSocketMessage(EventChatMessage, chatMessageData)
}

func CreateWebSocketMessageForChatMessageData(fullName string, message string, sentAt time.Time) *WebSocketMessage {
	chatMessageData := CreateChatMessageData(fullName, message, sentAt)
	return CreateWebSocketMessageWithEventChatMessage(chatMessageData)
}

func CreateAttendanceData(fullName string, sentAt time.Time) *AttendanceData {
	return &AttendanceData {
		FullName: fullName,
		SentAt: sentAt,
	}
}

func CreateWebSocketMessageForAttendance(eventType string, fullName string, sentAt time.Time) *WebSocketMessage {
	attendanceData := CreateAttendanceData(fullName, sentAt)
	return CreateWebSocketMessage(eventType, attendanceData)
}

func CreateWebSocketMessageForUserJoining(fullName string, joinedAt time.Time) *WebSocketMessage {
	return CreateWebSocketMessageForAttendance(EventUserJoined, fullName, joinedAt)
}

func CreateWebSocketMessageForUserLeaving(fullName string, leftAt time.Time) *WebSocketMessage {
	return CreateWebSocketMessageForAttendance(EventUserLeft, fullName, leftAt)
}
