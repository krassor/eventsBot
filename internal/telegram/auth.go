package telegramBot

import (
	"slices"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (bot *Bot) isAdmin(msg *tgbotapi.Message) (bool, error) {
	result := slices.Contains(bot.cfg.BotConfig.Admins, msg.From.UserName)

	return result, nil
}
