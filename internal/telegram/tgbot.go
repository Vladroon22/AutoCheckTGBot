package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Vladroon22/TG-Bot/internal/encryption"

	stud "github.com/Vladroon22/TG-Bot/internal/students"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type Bot struct {
	bot      *tgbotapi.BotAPI
	logg     *logrus.Logger
	students map[int]string
	mutex    sync.RWMutex
	timeIn   time.Time
}

type TelegaApiResp struct {
	Ok bool `json:"ok"`
}

func NewBot(bot *tgbotapi.BotAPI, logger *logrus.Logger) *Bot {
	return &Bot{
		bot:      bot,
		logg:     logger,
		students: make(map[int]string),
		timeIn:   time.Time{},
	}
}

var key = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Регистрация"),
		tgbotapi.NewKeyboardButton("Вход"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Автопосещение Вкл"),
		tgbotapi.NewKeyboardButton("Автопосещение Выкл"),
	),
)

func (b *Bot) Run(ctx context.Context) error {
	b.logg.Infof("Bot connected: %s\n", b.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.logg.Infoln("Bot is shutting down")
			return nil
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			chatID := update.Message.Chat.ID
			userID := update.Message.From.ID
			userName := update.Message.Chat.UserName
			switch update.Message.Text {
			case "Регистрация":
				if err := b.handleRegistration(userName, chatID, userID, updates, key); err != nil {
					b.logg.Errorln(err.Error())
				}
			case "Вход":
				if err := b.handleEnter(chatID, userID, updates, key); err != nil {
					b.logg.Errorln(err.Error())
				}
			}
		}
	}

}

func (b *Bot) handleInput(ctx context.Context, chatID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup, prompts ...string) ([]string, error) {
	go func() error {
		select {
		case <-ctx.Done():
			return errors.New("введите данные заново")
		default:
			return nil
		}
	}()
	var inputs []string
	for _, prompt := range prompts {
		b.MessageToUser(chatID, key, prompt)
		input, err := b.MessageToBot(ctx, chatID, up)
		if err != nil {
			b.logg.Errorln(err)
			return nil, err
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
}

func (b *Bot) handleRegistration(userName string, chatID, userID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	prompts := []string{"Введите вашу учебную группу", "Ваш Логин от ЛК", "Ваш пароль от ЛК"}
	inputs, err := b.handleInput(ctx, chatID, up, key, prompts...)
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, "Данные введены не корректно")
		return err
	}
	enc_pass, err := encryption.Hashing(inputs[2])
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, "Ошибка на сервере (password hashing)")
		return err
	}
	if err := stud.Register(ctx, inputs[0], inputs[1], string(enc_pass)); err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		return err
	}
	b.MessageToUser(chatID, key, "Данные успешно загружены")

	if err := b.IsSubOnChannel(ctx, chatID, userID, key); err != nil {
		b.MessageToUser(chatID, key, err.Error())
		return err
	}

	b.mutex.Lock()
	b.students[int(userID)] = userName
	b.mutex.Unlock()
	b.timeIn = time.Now()
	b.logg.Infof("%s - has been registered at %s\n", userName, b.timeIn.Format(time.DateTime))

	return nil
}

func (b *Bot) handleEnter(chatID int64, userID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	prompts := []string{"Ваша учебная группа", "Ваш Логин от ЛК", "Ваш пароль от ЛК"}
	inputs, err := b.handleInput(ctx, chatID, up, key, prompts...)
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, "Данные введенны не корректно")
		return err
	}

	st, err := stud.Enter(ctx, inputs[0], inputs[1], inputs[2])
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		return err
	}
	b.MessageToUser(chatID, key, "Вход выполнен!")

	if err := b.IsSubOnChannel(ctx, chatID, userID, key); err != nil {
		return err
	}

	b.ChangeStatusOfStudent(ctx, &st, chatID, userID, up, key)

	return nil
}

func (b *Bot) ChangeStatusOfStudent(c context.Context, st *stud.Student, chatID int64, userID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) {
	b.MessageToUser(userID, key, "Выбирите свой статус")
	b.timeIn = time.Now()
	b.mutex.RLock()
	user := b.students[int(userID)]
	b.mutex.RUnlock()
	b.logg.Infof("%s - has been entered at %s\n", user, b.timeIn.Format(time.DateTime))

	message := make(chan string, 1)
	exit := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case tag := <-message:
				if tag == "Автопосещение Вкл" {
					if err := st.ChangeStatus(true); err != nil {
						b.logg.Errorln(user, " - ", err)
						b.MessageToUser(chatID, key, err.Error())
						exit <- struct{}{}
						return
					}
					b.MessageToUser(chatID, key, "Поздравляем! Вы отметились на паре!")
					exit <- struct{}{}
					return
				} else if tag == "Автопосещение Выкл" {
					if err := st.ChangeStatus(false); err != nil {
						b.logg.Errorln(user, " - ", err)
						b.MessageToUser(chatID, key, err.Error())
						exit <- struct{}{}
						return
					}
					b.MessageToUser(chatID, key, "Вы ушли с Пары")
					exit <- struct{}{}
					return
				} else {
					b.logg.Infoln("signal channel closed")
					exit <- struct{}{}
					return
				}
			case <-c.Done():
				b.logg.Infoln("context done")
				exit <- struct{}{}
				return
			}
		}
	}()

	for {
		select {
		case <-c.Done():
			b.logg.Infoln("context done")
			return
		case <-exit:
			b.logg.Infoln("channel exit")
			return
		default:
			tag, _ := b.MessageToBot(c, chatID, up)
			message <- tag
			return
		}
	}
}

func (b *Bot) MessageToUser(chatID int64, key interface{}, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = key
	if _, err := b.bot.Send(msg); err != nil {
		b.logg.Errorln(err)
		return
	}
}

func (b *Bot) MessageToBot(c context.Context, ChatID int64, updates tgbotapi.UpdatesChannel) (string, error) {
	ctx, cancel := context.WithTimeout(c, time.Second*10)
	defer cancel()

	for {
		select {
		case update := <-updates:
			if update.Message != nil && update.Message.Chat.ID == ChatID {
				return update.Message.Text, nil
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func (b *Bot) IsSubOnChannel(c context.Context, chatID, userID int64, key tgbotapi.ReplyKeyboardMarkup) error {
	go func() error {
		select {
		case <-c.Done():
			return errors.New("введите данные заново")
		default:
			return nil
		}
	}()
	if ok, err := b.checkSub(userID); !ok {
		channelLink := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("Перейти в канал", "https://t.me/"+os.Getenv("channel")),
			),
		)
		b.MessageToUser(chatID, channelLink, "Чтобы получить возможность отмечаться надо подписаться")
		return err
	}
	return nil
}

func (b *Bot) checkSub(userID int64) (bool, error) {
	sub, err := GetReqToTelegram(userID)
	if err != nil {
		b.logg.Errorln(err)
		return false, err
	}

	if !sub {
		b.mutex.RLock()
		user := b.students[int(userID)]
		b.mutex.RUnlock()
		return false, errors.New(user + "не подписан")
	}

	return true, nil
}

func GetReqToTelegram(userID int64) (bool, error) {
	token := os.Getenv("token")
	nameChanel := os.Getenv("channel")
	URL := fmt.Sprintf("https://api.telegram.org/bot%s/getChatMember?chat_id=@%s&user_id=%d", token, nameChanel, userID)

	resp, err := http.Get(URL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var apiResp TelegaApiResp
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return false, err
	}

	return apiResp.Ok, nil
}
