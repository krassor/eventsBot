package telegramBot

import (
	"context"
	"fmt"
	"strconv"
	"unicode/utf16"

	"app/main.go/internal/config"
	"app/main.go/internal/utils/logger/sl"

	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AIBotApi interface {
	ProcessMessage(ctx context.Context, userID int64, message string) (string, error)
}

// Repository определяет интерфейс для взаимодействия с хранилищем событий.
type Repository interface {
	UpdateEventStatus(ctx context.Context, eventID string, status string) error
}

type Bot struct {
	tgbot      *tgbotapi.BotAPI
	cfg        *config.Config
	repository Repository
	// AIBot           AIBotApi
	shutdownChannel chan struct{}
	ctx             context.Context
	cancel          context.CancelFunc
	log             *slog.Logger
	UsersState      map[int64]UserState
}

// UserState хранит состояние пользователя
// AwaitingFile - ожидание файла
// SurveyType - тип опроса: ADULT, SCHOOLCHILD
type UserState struct {
	AwaitingFile bool
	SurveyType   string
	FileType     string
}

func New(logger *slog.Logger, cfg *config.Config, repository Repository) *Bot {
	op := "telegramBot.New()"
	log := logger.With(
		slog.String("op", op),
	)

	bot, err := tgbotapi.NewBotAPI(cfg.BotConfig.TgbotApiToken)
	if err != nil {
		log.Error("error auth telegram bot", slog.String("error", err.Error()))
	}

	if cfg.Env == "local" {
		bot.Debug = true
	} else {
		bot.Debug = false
	}

	log.Info("Authorized on account", slog.String("UserName", bot.Self.UserName))

	ctx, cancel := context.WithCancel(context.Background())

	return &Bot{
		tgbot:           bot,
		cfg:             cfg,
		repository:      repository,
		shutdownChannel: make(chan struct{}),
		ctx:             ctx,
		cancel:          cancel,
		log:             log,
		UsersState:      make(map[int64]UserState),
	}
}

func (bot *Bot) Start(updateTimeout int) {
	op := "tgBot.Start()"
	log := bot.log.With(
		slog.String("op", op),
	)
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = updateTimeout

	_, err := bot.tgbot.MakeRequest("deleteWebhook", tgbotapi.Params{"drop_pending_updates": "false"})
	if err != nil {
		sl.Err(err)
	}

	updates := bot.tgbot.GetUpdatesChan(updateConfig)

	for update := range updates {
		log.Info("start update processing")

		go bot.processingMessages(&update)
	}
	log.Info("exiting update processing loop")

}

func (bot *Bot) processingMessages(update *tgbotapi.Update) {
	op := "tgBot.processingMessages()"
	log := bot.log.With(
		slog.String("op", op),
	)

	if update.Message != nil {
		log.Info(
			"Input message",
			slog.String("user id", strconv.FormatInt(update.Message.From.ID, 10)),
			slog.String("user name", update.Message.From.UserName),
			slog.String("first name", update.Message.From.FirstName),
			slog.String("last name", update.Message.From.LastName),
			slog.String("msg", update.Message.Text),
		)
	}
	if update.CallbackQuery != nil {
		log.Info(
			"Input callback",
			slog.String("user id", strconv.FormatInt(update.CallbackQuery.From.ID, 10)),
			slog.String("user name", update.CallbackQuery.From.UserName),
			slog.String("first name", update.CallbackQuery.From.FirstName),
			slog.String("last name", update.CallbackQuery.From.LastName),
		)
	}

	select {
	case <-bot.shutdownChannel:
		return
	case <-bot.ctx.Done():
		return
	default:
		switch {
		//Check if message is a command
		case (update.Message != nil && update.Message.IsCommand()):
			if err := bot.commandHandler(bot.ctx, update, bot.sendReplyMessage); err != nil {
				sl.Err(err)
			}
		case (update.Message != nil && update.Message.Document != nil):
			if err := bot.fileHandler(bot.ctx, update, bot.sendReplyMessage); err != nil {
				sl.Err(err)
			}
		case update.CallbackQuery != nil:
			bot.handleCallbackQuery(update)
		// // Проверяем, если сообщение адресовано самому боту
		// case update.Message.Chat.IsPrivate():
		// 	bot.defaultHandler(bot.ctx, update, bot.sendMessage)
		// // если сообщение адресовано каналу, в котором находится бот
		// case (update.Message.Chat.IsChannel() || update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup()) && bot.isBotMentioned(update):
		// 	bot.defaultHandler(bot.ctx, update, bot.sendReplyMessage)
		// // Проверяем, если сообщение является ответом на сообщение бота
		// case bot.isReplyToBotMessage(update):
		// 	bot.defaultHandler(bot.ctx, update, bot.sendReplyMessage)
		default:
			log.Info("unsupported message type",
				slog.String("message_type", "unknown"),
				slog.Any("entities", update.Message.Entities),
			)

		}

		return
	}
}

func (bot *Bot) isBotMentioned(update *tgbotapi.Update) bool {
	op := "tgBot.isBotMentioned()"
	log := bot.log.With(
		slog.String("op", op),
	)
	result := false
	msg := update.Message
	for _, entity := range msg.Entities {
		// Проверяем тип упоминания - если это упоминание, то
		// получаем само упоминание и обрабатываем его
		if entity.Type == "mention" {
			// Encode it into utf16
			utf16EncodedString := utf16.Encode([]rune(msg.Text))
			// Decode just the piece of string I need
			runeString := utf16.Decode(utf16EncodedString[entity.Offset+1 : entity.Offset+entity.Length])
			// Transform []rune into string
			mention := string(runeString)

			if mention == bot.tgbot.Self.UserName {
				log.Debug("mention found", slog.String("message", msg.Text))
				result = true
				break
			}

		}
	}
	return result
}

func (bot *Bot) isReplyToBotMessage(update *tgbotapi.Update) bool {
	result := false
	if update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.From.ID == bot.tgbot.Self.ID {
		result = true
	}
	return result
}

func (bot *Bot) sendReplyMessage(inputMsg *tgbotapi.Message, replyText string) error {
	replyMsg := tgbotapi.NewMessage(inputMsg.Chat.ID, "")
	replyMsg.ReplyToMessageID = inputMsg.MessageID

	chunks := splitTextIntoChunks(replyText, 4095)

	for _, chunk := range chunks {
		replyMsg.Text = chunk

		_, err := bot.tgbot.Send(replyMsg)
		if err != nil {
			return fmt.Errorf("tgbot.sendMessage: %w", err)
		}
	}
	return nil
}

func (bot *Bot) sendMessage(inputMsg *tgbotapi.Message, replyText string) error {
	replyMsg := tgbotapi.NewMessage(inputMsg.Chat.ID, "")

	chunks := splitTextIntoChunks(replyText, 4096)

	for _, chunk := range chunks {
		replyMsg.Text = chunk

		_, err := bot.tgbot.Send(replyMsg)
		if err != nil {
			return fmt.Errorf("tgbot.sendMessage: %w", err)
		}
	}
	return nil
}

func splitTextIntoChunks(text string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(text); i += chunkSize {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}
	return chunks
}

func (bot *Bot) Shutdown(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("exit tgBot: %w", ctx.Err())
		default:
			close(bot.shutdownChannel)
			bot.cancel()
			bot.tgbot.StopReceivingUpdates()
			return nil
		}
	}
}
