package handler

import "github.com/meow-pad/persian/frame/pnet/tcp/session"

type MessageHandler interface {
	HandleMessage(session session.Session, msg any) error
}
