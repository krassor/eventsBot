package telegramBot

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type sendFunction func(inputMsg *tgbotapi.Message, replyText string) error

func (bot *Bot) commandHandler(ctx context.Context, update *tgbotapi.Update, sendFunc sendFunction) error {
	op := "bot.commandHandle"
	// Extract the command from the Message.
	log := bot.log.With(
		slog.String("op", op),
	)

	msg := update.Message

	switch update.Message.Command() {
	// case "setsystemprompt":
	// 	replyText := ""
	// 	isAdmin, err := bot.isAdmin(update.Message)
	//
	// 	log.Debug("setsystemprompt",
	// 		slog.String("user name", update.Message.From.UserName),
	// 		slog.String("message", update.Message.Text),
	// 		slog.String("is admin", strconv.FormatBool(isAdmin)),
	// 	)
	//
	// 	if err != nil {
	// 		return fmt.Errorf("%s: %w", op, err)
	// 	}
	//
	// 	if isAdmin {
	//
	// 		prompt := strings.TrimPrefix(
	// 			update.Message.Text, "/setsystemprompt ")
	//
	//
	//
	// 		err = bot.cfg.Write()
	// 		if err != nil {
	// 			return fmt.Errorf("%s: %w", op, err)
	// 		}
	//
	// 		log.Debug(
	// 			"system prompt changed",
	// 			slog.String("user", update.Message.From.UserName),
	// 		)
	//
	// 		replyText = "👍 System role prompt changed 👍"
	// 		err := sendFunc(update.Message, replyText)
	// 		if err != nil {
	// 			return fmt.Errorf("%s: %w", op, err)
	// 		}
	// 	}

	case "setfile":
		//replyText := ""
		isAdmin, err := bot.isAdmin(update.Message)

		log.Debug("setfile",
			slog.String("user name", update.Message.From.UserName),
			slog.String("message", update.Message.Text),
			slog.String("is admin", strconv.FormatBool(isAdmin)),
		)

		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if !isAdmin {
			return fmt.Errorf("user is not admin")
		}

		err = bot.sendSurveyTypeMessage(update)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		// replyText = "Attach a file"
		// err = sendFunc(update.Message, replyText)
		// if err != nil {
		// 	return fmt.Errorf("%s: %w", op, err)
		// }
		// bot.UsersState[update.Message.From.ID] = UserState{AwaitingFile: true}

	/*case "settemplate":
	isAdmin, err := bot.isAdmin(update.Message)

	log.Debug("setpromptfile",
		slog.String("user name", update.Message.From.UserName),
		slog.String("message", update.Message.Text),
		slog.String("is admin", strconv.FormatBool(isAdmin)),
	)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if !isAdmin {
		return fmt.Errorf("user is not admin")
	}

	bot.sendSurveyTypeMessage(update)*/

	/*case "getsystemprompt":

	replyText := ""
	isAdmin, err := bot.isAdmin(update.Message)

	log.Debug("getsystemprompt",
		slog.String("user name", update.Message.From.UserName),
		slog.String("message", update.Message.Text),
		slog.String("is admin", strconv.FormatBool(isAdmin)),
	)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if isAdmin {

		replyText = bot.cfg.BotConfig.AI.SystemRolePrompt
		err := sendFunc(update.Message, replyText)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}
	*/

	case "setmodel":
		replyText := ""
		isAdmin, err := bot.isAdmin(update.Message)

		log.Debug("setmodel",
			slog.String("user name", update.Message.From.UserName),
			slog.String("message", update.Message.Text),
			slog.String("is admin", strconv.FormatBool(isAdmin)),
		)

		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if isAdmin {

			bot.cfg.BotConfig.AI.ModelName = strings.TrimPrefix(
				update.Message.Text, "/setmodel ")

			err = bot.cfg.Write()
			if err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}

			log.Debug(
				"system model changed",
				slog.String("model", bot.cfg.BotConfig.AI.ModelName),
			)

			replyText = "👍 Model changed 👍"
			err := sendFunc(update.Message, replyText)
			if err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}

	case "getmodel":

		replyText := ""
		isAdmin, err := bot.isAdmin(update.Message)

		log.Debug("getmodel",
			slog.String("user name", update.Message.From.UserName),
			slog.String("message", update.Message.Text),
			slog.String("is admin", strconv.FormatBool(isAdmin)),
		)

		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		if isAdmin {

			replyText = bot.cfg.BotConfig.AI.ModelName
			err := sendFunc(update.Message, replyText)
			if err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}

	case "start":
		replyText := fmt.Sprintf("Hi, %s! Send a command.", msg.From.UserName)
		err := sendFunc(update.Message, replyText)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

	default:
		replyText := "I don't know this command"
		err := bot.sendReplyMessage(msg, replyText)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}

func (bot *Bot) fileHandler(ctx context.Context, update *tgbotapi.Update, sendFunc sendFunction) error {
	op := "bot.fileHandler"
	// Extract the command from the Message.
	log := bot.log.With(
		slog.String("op", op),
	)

	replyText := ""
	isAdmin, err := bot.isAdmin(update.Message)
	log.Debug(
		"file handler",
		slog.String("user name", update.Message.From.UserName),
		slog.String("message", update.Message.Text),
		slog.String("is admin", strconv.FormatBool(isAdmin)),
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if !isAdmin {
		return fmt.Errorf("User dont have admin permission")
	}

	userState := bot.UsersState[update.Message.From.ID]
	log.Info(
		"User state",
		slog.String("file type", userState.FileType),
		slog.String("survey type", userState.SurveyType),
		slog.Bool("awaiting file", userState.AwaitingFile),
	)
	// Проверка ожидания файла в сообщении
	if !userState.AwaitingFile {
		replyText = "File not awaiting"
		err := sendFunc(update.Message, replyText)
		e := fmt.Errorf("File not awaiting")
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		return fmt.Errorf("%s: %w", op, e)
	}

	// Получение id файла
	fileID := update.Message.Document.FileID
	log.Info(
		"Received message with file",
		slog.String("user name", update.Message.From.UserName),
		slog.String("message", update.Message.Text),
		slog.String("file_id", fileID),
	)

	fileExt := strings.ToLower(filepath.Ext(update.Message.Document.FileName))
	filePath := ""
	fileName := ""
	isPromptFile := false
	isTmplFile := false

	switch userState.FileType {
	case "PROMPT":
		if fileExt != ".md" {
			replyText = "wrong file extension. PLease try again"
			err := fmt.Errorf("wrong file extention: %s", update.Message.Document.FileName)
			e := sendFunc(update.Message, replyText)
			if e != nil {
				return fmt.Errorf("%s: %w", op, e)
			}
			return fmt.Errorf("%s: %w", op, err)
		}
		isPromptFile = true
		fileName = bot.cfg.BotConfig.AI.PromptFileName
		log.Debug(
			"case PROMPT",
			slog.String("file ext", fileExt),
			slog.Bool("isPromptFile", isPromptFile),
			slog.String("fileName", fileName),
		)

	default:
		log.Error(
			"case default: unknown file type state",
		)
		return fmt.Errorf("unknown file type state")
	}

	switch userState.SurveyType {
	case "ADULT":
		if isPromptFile {
			filePath = bot.cfg.BotConfig.AI.PromptFilePath
		} else {
			log.Error(
				"case default: unknown survey type state",
			)
			return fmt.Errorf("unknown survey type state")
		}
		log.Debug(
			"case ADULT",
			slog.String("filePath", filePath),
		)

	case "SCHOOLCHILD":
		if isPromptFile {
			filePath = bot.cfg.BotConfig.AI.PromptFilePath
		} else {
			log.Error(
				"case default: unknown survey type state",
			)
			return fmt.Errorf("unknown file type state")
		}
		log.Debug(
			"case SCHOOLCHILD",
			slog.String("filePath", filePath),
		)
	}

	fullFilePath := filepath.Join(filePath, fileName)
	log.Debug(
		"Join filePath and fileName",
		slog.String("fullFilePath", fullFilePath),
	)

	//Получаем file_path
	fileURL, err := bot.tgbot.GetFileDirectURL(fileID)
	if err != nil {
		replyText = "Cannot download file. PLease try again"
		e := sendFunc(update.Message, replyText)
		if e != nil {
			return fmt.Errorf("%s: %w", op, e)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Debug(
		"Get file URL",
		slog.String("fileURL", fileURL),
	)

	// Делаем HTTP GET-запрос по URL
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Get(fileURL)
	//resp, err := http.Get(fileURL)
	if err != nil {
		replyText = "Cannot download file. PLease try again"
		e := sendFunc(update.Message, replyText)
		if e != nil {
			return fmt.Errorf("%s: %w", op, e)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	//Сохраняем файл на диск
	buf := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(buf)
	if err != nil {
		replyText = "Cannot download file. PLease try again"
		e := sendFunc(update.Message, replyText)
		if e != nil {
			return fmt.Errorf("%s: %w", op, e)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	err = os.WriteFile(fullFilePath, buf, 0775)
	if err != nil {
		replyText = "Cannot save file. PLease try again"
		e := sendFunc(update.Message, replyText)
		if e != nil {
			return fmt.Errorf("%s: %w", op, e)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info(
		"file saved",
		slog.String("user name", update.Message.From.UserName),
		slog.String("file_id", fileID),
		slog.String("file_path", fullFilePath),
	)

	//Перечитываем заново промт из файла для применения изменений
	if isPromptFile {
		err = bot.cfg.ReadPromptFromFile()
		if err != nil {
			replyText = "Prompt file saved. But config file not updated. PLease try again"
			e := sendFunc(update.Message, replyText)
			if e != nil {
				return fmt.Errorf("%s: %w", op, e)
			}
			return fmt.Errorf("%s: %w", op, err)
		}
		log.Info(
			"Prompt file saved. Config updated.",
			slog.String("user name", update.Message.From.UserName),
			slog.String("file_id", fileID),
			slog.String("file_path", fullFilePath),
		)
		replyText = "👍 Prompt file saved. Config updated 👍"
		err = sendFunc(update.Message, replyText)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

	} else if isTmplFile {
		log.Info(
			"Template file saved.",
			slog.String("user name", update.Message.From.UserName),
			slog.String("file_id", fileID),
			slog.String("file_path", fullFilePath),
		)
		replyText = "👍 Template file saved. Config updated 👍"
		err = sendFunc(update.Message, replyText)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	//сбрасываем состояния по этому пользователю, т.к. операция прошла успешно
	bot.UsersState[update.Message.From.ID] = UserState{
		AwaitingFile: false,
		FileType:     "",
		SurveyType:   "",
	}
	return nil
}

func (bot *Bot) sendSurveyTypeMessage(update *tgbotapi.Update) error {
	chatID := update.Message.Chat.ID

	text := "Выберите тип опроса:"

	// Создаём инлайн-кнопки
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Adult", "ADULT"),
			tgbotapi.NewInlineKeyboardButtonData("Schoolchild", "SCHOOLCHILD"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Cancel", "CANCEL"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = inlineKeyboard

	_, err := bot.tgbot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send survey type message: %w", err)
	}

	return nil
}

func (bot *Bot) handleCallbackQuery(update *tgbotapi.Update) {
	op := "bot.handleCallbackQuery"
	log := bot.log.With(
		slog.String("op", op),
	)

	if update.CallbackQuery == nil {
		log.Error(
			"callback query is nil",
		)
		return
	}

	callback := update.CallbackQuery
	data := callback.Data
	chatID := callback.Message.Chat.ID
	userID := callback.From.ID

	// Отправляем "секретный" ответ, чтобы скрыть часики у кнопки
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	callbackConfig.ShowAlert = false
	_, err := bot.tgbot.Request(callbackConfig)
	if err != nil {
		log.Error(
			"failed to send callback response",
			slog.String("error", err.Error()),
		)
	}

	var responseText string
	var editMsg tgbotapi.EditMessageTextConfig
	userState := bot.UsersState[userID]

	switch data {
	case "ADULT":
		responseText = "Вы выбрали: тип опроса Adult\\.\n\rВыберите тип файла:"
		userState.SurveyType = "ADULT"
		inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Prompt", "PROMPT"),
				tgbotapi.NewInlineKeyboardButtonData("Template", "TEMPLATE"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Cancel", "CANCEL"),
			),
		)
		editMsg = tgbotapi.NewEditMessageTextAndMarkup(chatID, callback.Message.MessageID, responseText, inlineKeyboard)

	case "SCHOOLCHILD":
		responseText = "Вы выбрали: тип опроса Schoolchild\\.\n\rВыберите тип файла:"
		userState.SurveyType = "SCHOOLCHILD"
		inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Prompt", "PROMPT"),
				tgbotapi.NewInlineKeyboardButtonData("Template", "TEMPLATE"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Cancel", "CANCEL"),
			),
		)
		editMsg = tgbotapi.NewEditMessageTextAndMarkup(chatID, callback.Message.MessageID, responseText, inlineKeyboard)

	case "PROMPT":
		responseText = "Вы выбрали: тип файла prompt\\.\n\rЗагрузите *\\.md* файл:"
		userState.FileType = "PROMPT"
		userState.AwaitingFile = true
		inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Cancel", "CANCEL"),
			),
		)
		editMsg = tgbotapi.NewEditMessageTextAndMarkup(chatID, callback.Message.MessageID, responseText, inlineKeyboard)

	case "TEMPLATE":
		responseText = "Вы выбрали: тип файла template\\.\n\rЗагрузите *\\.html* файл:"
		userState.FileType = "TEMPLATE"
		userState.AwaitingFile = true
		inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Cancel", "CANCEL"),
			),
		)
		editMsg = tgbotapi.NewEditMessageTextAndMarkup(chatID, callback.Message.MessageID, responseText, inlineKeyboard)

	case "CANCEL":
		responseText = "Команда отменена"
		userState.AwaitingFile = false
		userState.SurveyType = ""
		userState.FileType = ""
		editMsg = tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, responseText)

	default:
		responseText = "Неизвестный тип опроса/файла\\."
		userState.AwaitingFile = false
		userState.SurveyType = ""
		userState.FileType = ""
		editMsg = tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, responseText)
	}

	bot.UsersState[userID] = userState
	editMsg.ParseMode = tgbotapi.ModeMarkdownV2

	_, err = bot.tgbot.Send(editMsg)
	if err != nil {
		log.Error(
			"failed to send callback response",
			slog.String("error", err.Error()),
		)
	}

	// Или отправить новое сообщение:
	// msg := tgbotapi.NewMessage(chatID, responseText)
	// _, _ = bot.tgbot.Send(msg)
}
