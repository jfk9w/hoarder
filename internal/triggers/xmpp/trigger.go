package xmpp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/based"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"

	"github.com/jfk9w/hoarder/internal/common"
	"github.com/jfk9w/hoarder/internal/jobs"
	"github.com/jfk9w/hoarder/internal/logs"
	"github.com/jfk9w/hoarder/internal/triggers"
)

const TriggerID = "xmpp"

type Config struct {
	Jid      string            `yaml:"jid" doc:"JID бота."`
	Password string            `yaml:"password" doc:"Пароль."`
	Users    map[string]string `yaml:"users" doc:"Маппинг JID в имя пользователя."`
	Presence time.Duration     `yaml:"presence,omitempty" doc:"Интервал для отправки присутствия." default:"1m"`
	State    time.Duration     `yaml:"state,omitempty" doc:"Интервал для отправки состояния (\"печатает\")." default:"5s"`
}

type TriggerParams struct {
	Clock  based.Clock `validate:"required"`
	Config Config      `validate:"required"`
}

type Trigger struct {
	clock  based.Clock
	config Config

	questions common.Questions[string, string]
	requests  sync.WaitGroup
	presence  based.Goroutine
	mu        sync.Mutex
}

func NewTrigger(params TriggerParams) (*Trigger, error) {
	if err := based.Validate(params); err != nil {
		return nil, err
	}

	return &Trigger{
		clock:     params.Clock,
		config:    params.Config,
		questions: common.NewQuestions[string, string](),
	}, nil
}

func (t *Trigger) ID() string {
	return TriggerID
}

func (t *Trigger) Run(ctx triggers.Context, jobs triggers.Jobs) {
	router := xmpp.NewRouter()
	router.HandleFunc("message", func(sender xmpp.Sender, packet stanza.Packet) {
		message, ok := packet.(stanza.Message)
		if !ok || message.Body == "" {
			return
		}

		isKnownJid := false
		for jid := range t.config.Users {
			if strings.HasPrefix(message.From, jid) {
				message.From = jid
				isKnownJid = true
				break
			}
		}

		if !isKnownJid {
			return
		}

		t.handleMessage(ctx, sender, message, jobs)
	})

	config := &xmpp.Config{
		Jid:        t.config.Jid,
		Credential: xmpp.Password(t.config.Password),
	}

	client, err := xmpp.NewClient(config, router, func(err error) {
		ctx.Error("client error", logs.Error(err))
	})

	if err != nil {
		ctx.Error("failed to create client", logs.Error(err))
		return
	}

	cm := xmpp.NewStreamManager(client, func(sender xmpp.Sender) {
		ticker := time.NewTicker(t.config.Presence)
		t.presence = based.Go(ctx, func(cctx context.Context) {
			ctx := triggers.ContextFrom(cctx)
			defer t.sendPresence(ctx, sender, stanza.PresenceShowXA)
			defer ticker.Stop()
			for {
				t.sendPresence(ctx, sender, stanza.PresenceShowChat)
				select {
				case <-ticker.C:
					continue
				case <-ctx.Done():
					return
				}
			}
		})
	})

	_ = based.Go(ctx, func(_ context.Context) {
		if err := cm.Run(); err != nil {
			ctx.Error("stream manager finished with error", logs.Error(err))
			return
		}

		ctx.Debug("stream manager finished OK")
	})

	defer cm.Stop()

	<-ctx.Done()
	t.requests.Wait()
}

func (t *Trigger) handleMessage(ctx triggers.Context, sender xmpp.Sender, message stanza.Message, jobs triggers.Jobs) {
	t.mu.Lock()
	select {
	case <-ctx.Done():
		t.mu.Unlock()
		_ = t.sendMessage(ctx, sender, message.From, "going down, try again later")
		return
	default:
		t.requests.Add(1)
		t.mu.Unlock()
		defer t.requests.Done()
	}

	userID := t.config.Users[message.From]
	ctx = ctx.As(userID)

	err := t.questions.Answer(ctx, userID, message.Body)
	switch {
	case err == nil:
		return

	case errors.Is(err, common.ErrNoQuestions):
		typing := t.startTyping(ctx, sender, message.From)

		askFn := t.askFn(sender, message.From)
		results := jobs.Run(ctx.Job().WithAskFn(askFn), t.clock.Now(), userID, strings.Fields(message.Body))

		var reply strings.Builder
		for _, result := range results {
			if result.Error == nil {
				reply.WriteString("✔ ")
				reply.WriteString(result.JobID)
				reply.WriteRune('\n')
			} else {
				for _, err := range multierr.Errors(result.Error) {
					reply.WriteString("✘ ")
					reply.WriteString(result.JobID)
					reply.WriteString(": ")
					reply.WriteString(err.Error())
					reply.WriteRune('\n')
				}
			}
		}

		typing.Cancel()
		_ = typing.Join(ctx)

		_ = t.sendMessage(ctx, sender, message.From, reply.String())

	default:
		_ = t.sendMessage(ctx, sender, message.From, err.Error())
	}
}

func (t *Trigger) askFn(sender xmpp.Sender, to string) jobs.AskFunc {
	userID := t.config.Users[to]
	return func(ctx context.Context, text string) (string, error) {
		return t.questions.Ask(ctx, userID, func(ctx context.Context, userID string) error {
			return t.sendMessage(ctx, sender, to, text)
		})
	}
}

func (t *Trigger) startTyping(ctx triggers.Context, sender xmpp.Sender, to string) based.Goroutine {
	ticker := time.NewTicker(t.config.State)
	return based.Go(ctx, func(cctx context.Context) {
		ctx := triggers.ContextFrom(cctx)
		defer t.sendState(ctx, sender, to, stanza.StateInactive{})
		defer ticker.Stop()

		for {
			t.sendState(ctx, sender, to, stanza.StateComposing{})

			select {
			case <-ticker.C:
				continue
			case <-ctx.Done():
				return
			}
		}
	})
}

func (t *Trigger) sendMessage(cctx context.Context, sender xmpp.Sender, to string, text string) error {
	message := stanza.NewMessage(stanza.Attrs{From: t.config.Jid, To: to})
	message.Body = text

	ctx := triggers.ContextFrom(cctx).With("to", to).With("text", text)
	if err := sender.Send(message); err != nil {
		ctx.Error("failed to send message", logs.Error(err))
		return errors.New("failed to send message")
	}

	ctx.Debug("sent message")
	return nil
}

func (t *Trigger) sendPresence(cctx context.Context, sender xmpp.Sender, show stanza.PresenceShow) {
	ctx := triggers.ContextFrom(cctx).With("show", show)
	for to := range t.config.Users {
		presence := stanza.NewPresence(stanza.Attrs{From: t.config.Jid, To: to})
		presence.Show = show

		ctx := ctx.With("to", to)
		if err := sender.Send(presence); err != nil {
			ctx.Error("failed to send presence", logs.Error(err))
			continue
		}

		ctx.Debug("sent presence")
	}
}

func (t *Trigger) sendState(ctx triggers.Context, sender xmpp.Sender, to string, state stanza.MsgExtension) {
	message := stanza.NewMessage(stanza.Attrs{From: t.config.Jid, To: to})
	message.Extensions = append(message.Extensions, state)

	ctx = ctx.With("to", to).With("state", fmt.Sprintf("%T", state))
	if err := sender.Send(message); err != nil {
		ctx.Error("failed to send state", logs.Error(err))
		return
	}

	ctx.Debug("sent state")
}
