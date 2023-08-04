package xmpp

import (
	"context"

	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"
)

type Config struct {
	Jid      string `yaml:"jid"`
	Password string `yaml:"password"`
}

func Run(ctx context.Context, cfg Config) {
	_ = xmpp.Config{
		Jid:        cfg.Jid,
		Credential: xmpp.Password(cfg.Password),
	}

	router := xmpp.NewRouter()
	router.HandleFunc("message", func(sender xmpp.Sender, packet stanza.Packet) {
		_, _ = packet.(stanza.Message)
	})
}
