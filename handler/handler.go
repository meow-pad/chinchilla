package handler

type MessageHandler interface {
	HandleMessage(msg any) error
}
