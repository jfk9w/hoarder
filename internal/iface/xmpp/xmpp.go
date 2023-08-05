package xmpp

import (
	"context"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"

	"github.com/jfk9w/hoarder/internal/convo"
	"github.com/jfk9w/hoarder/internal/etl"
)

type Processor interface {
	Process(ctx context.Context, username string) error
}

type Config struct {
	Jid      string            `yaml:"jid" doc:"Jabber ID для бота."`
	Password string            `yaml:"password" doc:"Пароль для бота."`
	Users    map[string]string `yaml:"users" doc:"Маппинг JID в имя пользователя, используемое в загрузчиках."`
}

type Asker = convo.Asker[string, string]

type Builder struct {
	Config    Config      `validate:"required"`
	Processor Processor   `validate:"required"`
	Log       *zap.Logger `validate:"required"`
}

func (b Builder) Run() (context.CancelFunc, error) {
	config := &xmpp.Config{
		Jid:        b.Config.Jid,
		Credential: xmpp.Password(b.Config.Password),
	}

	router := xmpp.NewRouter()
	client, err := xmpp.NewClient(config, router, func(err error) {
		b.Log.Error("client error", zap.Error(err))
	})

	if err != nil {
		return nil, errors.Wrap(err, "create client")
	}

	ctx, cancel := context.WithCancel(context.Background())
	questions := convo.NewQuestions[string, string]()
	var wg sync.WaitGroup
	router.HandleFunc("message", func(sender xmpp.Sender, packet stanza.Packet) {
		wg.Add(1)
		defer wg.Done()

		msg, ok := packet.(stanza.Message)
		if !ok {
			return
		}

		var username string
		for prefix, user := range b.Config.Users {
			if strings.HasPrefix(msg.From, prefix) {
				username = user
				break
			}
		}

		if username == "" {
			return
		}

		log := b.Log.With(zap.String("username", username))
		log.Debug("received message", zap.String("text", msg.Body))

		var (
			ctx, cancel = context.WithCancel(ctx)
			recipient   = &recipient{
				sender: sender,
				from:   b.Config.Jid,
				to:     msg.From,
				log:    log,
			}
		)

		defer cancel()

		err := questions.Answer(ctx, msg.From, msg.Body)
		switch err {
		case nil:
			return

		case convo.ErrNoQuestions:
			var (
				ctx   = etl.WithRequestInputFunc(ctx, requestInputFunc(recipient, msg.From, questions))
				reply = "✅"
			)

			if err := b.Processor.Process(ctx, username); err != nil {
				var b strings.Builder
				for _, err := range multierr.Errors(err) {
					b.WriteString("❌ ")
					b.WriteString(err.Error())
					b.WriteRune('\n')
				}

				reply = strings.Trim(b.String(), "\n")
			}

			_ = recipient.send(reply)

		default:
			_ = recipient.send(err.Error())
		}
	})

	cm := xmpp.NewStreamManager(client, nil)
	go func() {
		if err := cm.Run(); err != nil {
			b.Log.Error("finished stream manager", zap.Error(err))
			return
		}

		b.Log.Debug("finished stream manager")
	}()

	b.Log.Debug("started")
	return func() {
		cancel()
		wg.Wait()
		cm.Stop()
		b.Log.Debug("stopped")
	}, nil
}

type recipient struct {
	sender   xmpp.Sender
	from, to string
	log      *zap.Logger
}

func (r *recipient) send(text string) error {
	log := r.log.With(zap.String("text", text))

	message := stanza.NewMessage(stanza.Attrs{
		From: r.from,
		To:   r.to,
	})

	message.Body = text

	if err := r.sender.Send(message); err != nil {
		log.Error("send", zap.Error(err))
		return errors.New("send")
	}

	log.Debug("sent message")
	return nil
}

func requestInputFunc(recipient *recipient, to string, asker convo.Asker[string, string]) etl.RequestInputFunc {
	return func(ctx context.Context, text string) (string, error) {
		return asker.Ask(ctx, to, func(ctx context.Context, key string) error {
			return recipient.send(text)
		})
	}
}
