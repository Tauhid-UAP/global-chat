package chat

import "time"

type WebSocketMessage struct {
	Type string `json:"Type"`
	Data interface{} `json:"Data"`
}

type UserData struct {
	ID string `json:"ID"`
	FullName string `json:"FullName"`
}

type MetaData struct {
	SentAt time.Time `json:SentAt`
}

type ChatMessageData struct {
	Message string `json:"Message"`
	User *UserData `json:"User"`
	Meta *MetaData `json:Meta`
}

type AttendanceData struct {
	User *UserData `json:"User"`
	Meta *MetaData `json:Meta`
}

func CreateUserData(id string, fullName string) *UserData {
	return &UserData {
		ID: id,
		FullName: fullName,
	}
}

func CreateMetaData(sentAt time.Time) *MetaData {
	return &MetaData {SentAt: sentAt}
}

func CreateChatMessageData(message string, userData *UserData, metaData *MetaData) *ChatMessageData {
	return &ChatMessageData {
		Message: message,
		User: userData,
		Meta: metaData,
	}
}

func CreateChatMessageDataFromRawParameters(id string, fullName string, message string, sentAt time.Time) *ChatMessageData {
	userData := CreateUserData(id, fullName)
	metaData := CreateMetaData(sentAt)
	return CreateChatMessageData(message, userData, metaData)
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

func CreateWebSocketMessageForChatMessageData(id string, fullName string, message string, sentAt time.Time) *WebSocketMessage {
	chatMessageData := CreateChatMessageDataFromRawParameters(id, fullName, message, sentAt)
	return CreateWebSocketMessageWithEventChatMessage(chatMessageData)
}

func CreateAttendanceData(userData *UserData, metaData *MetaData) *AttendanceData {
	return &AttendanceData {
		User: userData,
		Meta: metaData,
	}
}

func CreateAttendanceDataFromRawParameters(id string, fullName string, sentAt time.Time) *AttendanceData {
	userData := CreateUserData(id, fullName)
	metaData := CreateMetaData(sentAt)
	return CreateAttendanceData(userData, metaData)
}

func CreateWebSocketMessageForAttendance(eventType string, id string, fullName string, sentAt time.Time) *WebSocketMessage {
	attendanceData := CreateAttendanceDataFromRawParameters(id, fullName, sentAt)
	return CreateWebSocketMessage(eventType, attendanceData)
}

func CreateWebSocketMessageForUserJoining(id string, fullName string, joinedAt time.Time) *WebSocketMessage {
	return CreateWebSocketMessageForAttendance(EventUserJoined, id, fullName, joinedAt)
}

func CreateWebSocketMessageForUserLeaving(id string, fullName string, leftAt time.Time) *WebSocketMessage {
	return CreateWebSocketMessageForAttendance(EventUserLeft, id, fullName, leftAt)
}
