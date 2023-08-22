package xmpp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/based"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"

	"github.com/jfk9w/hoarder/internal/convo"
	"github.com/jfk9w/hoarder/internal/etl"
	"github.com/jfk9w/hoarder/internal/util"
)

const (
	presenceInterval = time.Minute
	stateInterval    = 5 * time.Second
)

type Pipelines interface {
	Run(ctx context.Context, log *slog.Logger, username string) error
}

type Config struct {
	Jid      string            `yaml:"jid" doc:"Jabber ID для бота."`
	Password string            `yaml:"password" doc:"Пароль для бота."`
	Users    map[string]string `yaml:"users" doc:"Маппинг JID в имя пользователя, используемое в пайплайнах."`
}

type Asker = convo.Asker[string, string]

type Builder struct {
	Config    Config       `validate:"required"`
	Pipelines Pipelines    `validate:"required"`
	Log       *slog.Logger `validate:"required"`
}

func (b Builder) Run(ctx context.Context) (*Handler, error) {
	if err := based.Validate.Struct(b); err != nil {
		return nil, err
	}

	users := make(map[string]user, len(b.Config.Users))
	for jid, name := range b.Config.Users {
		users[jid] = user{
			name: name,
			mu:   new(based.RWMutex),
		}
	}

	h := &Handler{
		config: xmpp.Config{
			Jid:        b.Config.Jid,
			Credential: xmpp.Password(b.Config.Password),
		},
		users:     users,
		questions: convo.NewQuestions[string, string](),
		pipelines: b.Pipelines,
		log:       b.Log,
	}

	h.ctx, h.cancel = context.WithCancel(ctx)

	if err := h.start(); err != nil {
		h.cancel()
		return nil, err
	}

	return h, nil
}

type user struct {
	name string
	mu   *based.RWMutex
}

type Handler struct {
	config    xmpp.Config
	users     map[string]user
	questions convo.Questions[string, string]
	pipelines Pipelines
	log       *slog.Logger

	ctx    context.Context
	cancel func()
	work   sync.WaitGroup

	cm *xmpp.StreamManager
}

func (h *Handler) start() error {
	router := xmpp.NewRouter()
	router.HandleFunc("message", h.handleMessage)

	client, err := xmpp.NewClient(&h.config, router, h.handleError)
	if err != nil {
		return errors.Wrap(err, "create client")
	}

	h.cm = xmpp.NewStreamManager(client, h.onConnect)
	go func() {
		if err := h.cm.Run(); err != nil {
			h.log.Error("finished stream manager", util.Error(err))
			return
		}

		h.log.Debug("finished stream manager")
	}()

	h.log.Debug("started")

	return nil
}

func (h *Handler) onConnect(sender xmpp.Sender) {
	h.work.Add(1)
	go func() {
		defer h.work.Done()
		ticker := time.NewTicker(presenceInterval)
		defer ticker.Stop()
		for {
			h.sendPresence(sender, stanza.PresenceShowChat)
			select {
			case <-h.ctx.Done():
				h.sendPresence(sender, stanza.PresenceShowXA)
				return
			case <-ticker.C:
				continue
			}
		}
	}()
}

func (h *Handler) handleError(err error) {
	h.log.Error("client error", util.Error(err))
}

func (h *Handler) handleMessage(sender xmpp.Sender, packet stanza.Packet) {
	select {
	case <-h.ctx.Done():
		return
	default:
	}

	h.work.Add(1)
	defer h.work.Done()

	msg, ok := packet.(stanza.Message)
	if !ok || msg.Body == "" {
		return
	}

	var user *user
	for jid, entry := range h.users {
		if strings.HasPrefix(msg.From, jid) {
			user = &entry
			break
		}
	}

	if user == nil {
		return
	}

	log := h.log.With(slog.String("username", user.name))
	log.Debug("received message", slog.String("text", msg.Body))

	ctx, cancel := context.WithCancel(h.ctx)
	defer cancel()

	err := h.questions.Answer(ctx, msg.From, msg.Body)
	switch {
	case err == nil:
		return

	case errors.Is(err, convo.ErrNoQuestions):
		h.displayComposing(ctx, sender, msg.From)
		ctx := etl.WithRequestInputFunc(ctx, h.requestInputFunc(sender, msg.From, user.mu))
		reply := "✅"
		if err := h.pipelines.Run(ctx, log, user.name); err != nil {
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

func (h *Handler) displayComposing(ctx context.Context, sender xmpp.Sender, to string) {
	h.work.Add(1)
	go func() {
		defer h.work.Done()
		ticker := time.NewTicker(stateInterval)
		defer ticker.Stop()
		for {
			h.sendState(sender, to, stanza.StateComposing{})
			select {
			case <-ctx.Done():
				h.sendState(sender, to, stanza.StateInactive{})
				return
			case <-ticker.C:
				continue
			}
		}
	}()
}

func (h *Handler) requestInputFunc(sender xmpp.Sender, to string, mu *based.RWMutex) etl.RequestInputFunc {
	return func(ctx context.Context, text string) (string, error) {
		ctx, cancel := mu.Lock(ctx)
		defer cancel()
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		return h.questions.Ask(ctx, to, func(ctx context.Context, key string) error {
			return h.sendMessage(sender, to, text)
		})
	}
}

func (h *Handler) sendPresence(sender xmpp.Sender, show stanza.PresenceShow) {
	log := h.log.With(slog.String("show", string(show)))
	for to := range h.users {
		log := log.With(slog.String("to", to))
		presence := stanza.NewPresence(stanza.Attrs{From: h.config.Jid, To: to})
		presence.Show = show

		if err := sender.Send(presence); err != nil {
			log.Error("failed to send presence", util.Error(err))
			continue
		}

		log.Debug("sent presence")
	}
}

func (h *Handler) sendState(sender xmpp.Sender, to string, state stanza.MsgExtension) {
	log := h.log.With(slog.String("to", to), slog.String("state", fmt.Sprintf("%T", state)))
	message := stanza.NewMessage(stanza.Attrs{From: h.config.Jid, To: to})
	message.Extensions = append(message.Extensions, state)

	if err := sender.Send(message); err != nil {
		h.log.Error("failed to send state", util.Error(err))
		return
	}

	log.Debug("sent state")
}

func (h *Handler) sendMessage(sender xmpp.Sender, to string, text string) error {
	log := h.log.With(slog.String("to", to), slog.String("text", text))
	message := stanza.NewMessage(stanza.Attrs{From: h.config.Jid, To: to})
	message.Body = text

	if err := sender.Send(message); err != nil {
		log.Error("failed to send message", util.Error(err))
		return errors.New("failed to send message")
	}

	log.Debug("sent message")
	return nil
}

func (h *Handler) Stop() {
	h.cancel()
	h.work.Wait()
	h.cm.Stop()
	h.log.Debug("stopped")
}
