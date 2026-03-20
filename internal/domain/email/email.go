package email

import "context"

type Message struct {
	From    string
	To      string
	Subject string
	HTML    string
}

type Sender interface {
	Send(ctx context.Context, msg Message) error
}
