package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w/hoarder/internal/common"
	"github.com/jfk9w/hoarder/internal/logs"
	"github.com/jfk9w/hoarder/internal/triggers"
	"github.com/mr-linch/go-tg"
	"github.com/mr-linch/go-tg/tgb"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

const (
	TriggerID = "telegram"

	startCommand = "start"
)

type Config struct {
	Token  string                            `yaml:"token" doc:"Токен бота."`
	Users  common.UserMap[string, tg.UserID] `yaml:"users" doc:"Маппинг пользователей в ID в Telegram."`
	Typing time.Duration                     `yaml:"typing,omitempty" doc:"Интервал для отправки действия \"печатает...\"." default:"4s"`
}

type TriggerParams struct {
	Clock  based.Clock  `validate:"required"`
	Config Config       `validate:"required"`
	Logger *slog.Logger `validate:"required"`
}

type Trigger struct {
	clock     based.Clock
	config    Config
	log       *slog.Logger
	users     map[tg.UserID]string
	questions common.Questions[string, string]
}

func NewTrigger(params TriggerParams) (*Trigger, error) {
	if err := based.Validate(params); err != nil {
		return nil, err
	}

	users, err := params.Config.Users.Reverse()
	if err != nil {
		return nil, errors.Wrap(err, "parse users")
	}

	return &Trigger{
		clock:     params.Clock,
		config:    params.Config,
		log:       params.Logger,
		users:     users,
		questions: common.NewQuestions[string, string](),
	}, nil
}

func (t *Trigger) ID() string {
	return TriggerID
}

func (t *Trigger) Run(ctx triggers.Context, jobs triggers.Jobs) {
	client := tg.New(t.config.Token)
	commands := []tg.BotCommand{
		{
			Command:     startCommand,
			Description: "Получение информации о чате и пользователе",
		},
	}

	if err := client.SetMyCommands(commands).DoVoid(ctx); err != nil {
		t.log.Error("failed to set my commands", logs.Error(err), slog.String("scope", "default"))
		return
	}

	for _, info := range jobs.Info() {
		commands = append(commands, tg.BotCommand{
			Command:     info.ID,
			Description: info.Description,
		})
	}

	for userID := range t.users {
		if err := client.SetMyCommands(commands).
			Scope(tg.BotCommandScopeChat{ChatID: tg.ChatID(userID)}).
			DoVoid(ctx); err != nil {
			t.log.Error("failed to set my commands", slog.String("scope", "chat"), slog.Any("user", userID), logs.Error(err))
			return
		}
	}

	router := tgb.NewRouter().
		Message(t.answer, t, tgb.Not(tgb.MessageEntity(tg.MessageEntityTypeBotCommand))).
		Message(t.start, tgb.Command(startCommand))

	for _, info := range jobs.Info() {
		jobID := info.ID
		router.Message(func(ctx context.Context, msg *tgb.MessageUpdate) error {
			return t.execute(ctx, msg, client, jobs, jobID)
		}, t, tgb.Command(jobID))
	}

	if err := tgb.NewPoller(withBoundContext(router), client,
		tgb.WithPollerAllowedUpdates(tg.UpdateTypeMessage)).
		Run(ctx); err != nil {
		t.log.Error("failed to start poller", logs.Error(err))
	}
}

func (t *Trigger) execute(ctx context.Context, msg *tgb.MessageUpdate, client *tg.Client, jobs triggers.Jobs, jobID string) error {
	userID, _ := t.getUserID(msg.From)
	typing := t.typing(ctx, client, msg.Chat.ID)
	jctx := triggers.ContextFrom(ctx).As(userID).Job().
		WithAskFn(func(ctx context.Context, text string) (string, error) {
			typing.Cancel()
			_ = typing.Join(ctx)
			return t.questions.Ask(ctx, userID, func(ctx context.Context, userID string) error {
				if err := msg.Answer(tg.HTML.Text(text)).DoVoid(ctx); err != nil {
					return err
				}

				typing = t.typing(ctx, client, msg.Chat.ID)
				return nil
			})
		})

	report := make([]string, 0)
	for _, result := range jobs.Run(jctx, t.clock.Now(), userID, []string{jobID}) {
		if err := result.Error; err != nil {
			for _, err := range multierr.Errors(err) {
				report = append(report, fmt.Sprintf("✘ %s: %s", result.JobID, err.Error()))
			}
		} else {
			report = append(report, fmt.Sprintf("✔ %s", result.JobID))
		}
	}

	return msg.Answer(tg.HTML.Text(report...)).DoVoid(ctx)
}

func (t *Trigger) start(ctx context.Context, msg *tgb.MessageUpdate) error {
	return msg.Answer(tg.HTML.Text(
		fmt.Sprintf("User ID: %d", msg.From.ID),
		fmt.Sprintf("Chat ID: %d", msg.Chat.ID),
	)).DoVoid(ctx)
}

func (t *Trigger) answer(ctx context.Context, msg *tgb.MessageUpdate) error {
	userID, _ := t.getUserID(msg.From)
	err := t.questions.Answer(ctx, userID, msg.Text)
	if err == common.ErrNoQuestions {
		return nil
	}

	return err
}

func (t *Trigger) typing(ctx context.Context, client *tg.Client, chatID tg.ChatID) based.Goroutine {
	return based.Go(ctx, func(ctx context.Context) {
		ticker := time.NewTicker(t.config.Typing)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = client.SendChatAction(chatID, tg.ChatActionTyping).DoVoid(ctx)
			}
		}
	})
}

func (t *Trigger) Allow(ctx context.Context, update *tgb.Update) (bool, error) {
	_, ok := t.getUserID(update.Message.From)
	return ok, nil
}

func (t *Trigger) getUserID(user *tg.User) (string, bool) {
	if userID, ok := t.users[user.ID]; ok {
		return userID, true
	}

	return "", false
}

func withBoundContext(handler tgb.Handler) tgb.Handler {
	return tgb.HandlerFunc(func(ctx context.Context, update *tgb.Update) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		return handler.Handle(ctx, update)
	})
}
