package xmpp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-playground/validator"
	"github.com/jfk9w-go/based"

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

var validate = &based.Lazy[*validator.Validate]{
	Fn: func(ctx context.Context) (*validator.Validate, error) {
		return validator.New(), nil
	},
}

func (b Builder) Run(ctx context.Context) (*Handler, error) {
	if validate, err := validate.Get(ctx); err != nil {
		return nil, err
	} else if err := validate.Struct(b); err != nil {
		return nil, err
	}

	config := &xmpp.Config{
		Jid:        b.Config.Jid,
		Credential: xmpp.Password(b.Config.Password),
	}

	router := xmpp.NewRouter()
	client, err := xmpp.NewClient(config, router, func(err error) { b.Log.Error("client error", zap.Error(err)) })
	if err != nil {
		return nil, errors.Wrap(err, "create client")
	}

	ctx, cancel := context.WithCancel(ctx)

	h := &Handler{
		jid:       b.Config.Jid,
		users:     b.Config.Users,
		questions: convo.NewQuestions[string, string](),
		processor: b.Processor,
		log:       b.Log,
		ctx:       ctx,
	}

	router.HandleFunc("message", h.handleMessage)

	cm := xmpp.NewStreamManager(client, func(sender xmpp.Sender) {
		b.sendPresence(sender, stanza.PresenceShowChat)
	})

	go func() {
		if err := cm.Run(); err != nil {
			b.Log.Error("finished stream manager", zap.Error(err))
			return
		}

		b.Log.Debug("finished stream manager")
	}()

	h.cancel = func() {
		b.sendPresence(client, stanza.PresenceShowDND)
		cancel()
		h.work.Wait()
		b.sendPresence(client, stanza.PresenceShowXA)
		cm.Stop()
	}

	b.Log.Debug("started")

	return h, nil
}

func (b Builder) sendPresence(sender xmpp.Sender, show stanza.PresenceShow) {
	log := b.Log.With(zap.String("show", string(show)))
	for to := range b.Config.Users {
		log := log.With(zap.String("to", to))
		presence := stanza.NewPresence(stanza.Attrs{From: b.Config.Jid, To: to})
		presence.Show = show

		if err := sender.Send(presence); err != nil {
			log.Error("failed to send presence", zap.Error(err))
			continue
		}

		log.Debug("sent presence")
	}
}

type Handler struct {
	jid       string
	users     map[string]string
	questions convo.Questions[string, string]
	processor Processor
	log       *zap.Logger
	ctx       context.Context
	cancel    func()
	stop      atomic.Bool
	work      sync.WaitGroup
}

func (h *Handler) handleMessage(sender xmpp.Sender, packet stanza.Packet) {
	if h.stop.Load() {
		return
	}

	h.work.Add(1)
	defer h.work.Done()

	msg, ok := packet.(stanza.Message)
	if !ok || msg.Body == "" {
		return
	}

	var username string
	for prefix, user := range h.users {
		if strings.HasPrefix(msg.From, prefix) {
			username = user
			break
		}
	}

	if username == "" {
		return
	}

	log := h.log.With(zap.String("username", username))
	log.Debug("received message", zap.String("text", msg.Body))

	ctx, cancel := context.WithCancel(h.ctx)
	defer cancel()

	err := h.questions.Answer(ctx, msg.From, msg.Body)
	switch err {
	case nil:
		return

	case convo.ErrNoQuestions:
		go func(ctx context.Context) {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				_ = h.sendState(sender, msg.From, stanza.StateComposing{})
				select {
				case <-ticker.C:
				case <-ctx.Done():
					_ = h.sendState(sender, msg.From, stanza.StateInactive{})
					return
				}
			}
		}(ctx)

		ctx := etl.WithRequestInputFunc(ctx, h.requestInputFunc(sender, msg.From))
		reply := "✅"

		if err := h.processor.Process(ctx, username); err != nil {
			var b strings.Builder
			for _, err := range multierr.Errors(err) {
				b.WriteString("❌ ")
				b.WriteString(err.Error())
				b.WriteRune('\n')
			}

			reply = strings.Trim(b.String(), "\n")
		}

		_ = h.sendMessage(sender, msg.From, reply)

	default:
		_ = h.sendMessage(sender, msg.From, err.Error())
	}
}

func (h *Handler) requestInputFunc(sender xmpp.Sender, to string) etl.RequestInputFunc {
	return func(ctx context.Context, text string) (string, error) {
		return h.questions.Ask(ctx, to, func(ctx context.Context, key string) error {
			return h.sendMessage(sender, to, text)
		})
	}
}

func (h *Handler) sendState(sender xmpp.Sender, to string, state stanza.MsgExtension) error {
	log := h.log.With(zap.String("to", to), zap.String("state", fmt.Sprintf("%T", state)))
	message := stanza.NewMessage(stanza.Attrs{From: h.jid, To: to})
	message.Extensions = append(message.Extensions, state)

	if err := sender.Send(message); err != nil {
		h.log.Error("failed to send state", zap.Error(err))
		return errors.New("failed to send state")
	}

	log.Debug("sent state")
	return nil
}

func (h *Handler) sendMessage(sender xmpp.Sender, to string, text string) error {
	log := h.log.With(zap.String("to", to), zap.String("text", text))
	message := stanza.NewMessage(stanza.Attrs{From: h.jid, To: to})
	message.Body = text

	if err := sender.Send(message); err != nil {
		log.Error("failed to send message", zap.Error(err))
		return errors.New("failed to send message")
	}

	log.Debug("sent message")
	return nil
}

func (h *Handler) Stop() {
	h.stop.Store(true)
	h.cancel()
	h.log.Debug("stopped")
}
