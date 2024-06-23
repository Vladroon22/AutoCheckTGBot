package telegram

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Vladroon22/TG-Bot/internal/encryption"
	stud "github.com/Vladroon22/TG-Bot/internal/students"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func MustToken() string {
	token := flag.String(
		"bot",
		"6714254546:AAHrhTeFzVwO54K4VwjY8-of8skLC7l4_zY",
		"access to telegram-bot",
	)
	flag.Parse()

	if *token == "" {
		log.Fatal("invalid token")
	}

	return *token
}

type Bot struct {
	bot      *tgbotapi.BotAPI
	logg     *logrus.Logger
	nums     int
	students map[int]int64
	mutex    sync.Mutex
	timeIn   time.Time
}

type TelegaApiResp struct {
	Ok bool `json:"ok"`
}

func NewBot(bot *tgbotapi.BotAPI, logger *logrus.Logger) *Bot {
	return &Bot{
		bot:      bot,
		logg:     logger,
		nums:     0,
		students: make(map[int]int64),
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

func (b *Bot) Run() error {
	b.logg.Infof("Bot connected: %s\n", b.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		chatID := update.Message.Chat.ID
		userID := update.Message.From.ID
		switch update.Message.Text {
		case "Регистрация":
			b.handleRegistration(chatID, userID, updates, key)
		case "Вход":
			b.handleEnter(chatID, userID, updates, key)
		}
	}
	return nil
}

func (b *Bot) handleInput(chatID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup, prompts ...string) ([]string, error) {
	var inputs []string
	for _, prompt := range prompts {
		b.MessageToUser(chatID, key, prompt)
		input, err := b.MessageToBot(chatID, up)
		if err != nil {
			b.logg.Errorln(err)
			return nil, err
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
}

func (b *Bot) handleRegistration(chatID, userID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) error {
	prompts := []string{"Введите вашу учебную группу", "Ваш Логин от ЛК", "Ваш пароль от ЛК"}
	inputs, err := b.handleInput(chatID, up, key, prompts...)
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		return err
	}
	enc_pass, err := encryption.Hashing(inputs[2])
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		return err
	}
	err = stud.Register(inputs[0], inputs[1], string(enc_pass))
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		return err
	}
	b.MessageToUser(chatID, key, "Данные успешно загружены")

	if err := b.IsSubOnChannel(chatID, userID, key); err != nil {
		b.MessageToUser(chatID, key, err.Error())
		return err
	}

	b.mutex.Lock()
	b.nums++
	b.students[b.nums] = userID
	b.mutex.Unlock()
	b.timeIn = time.Now()
	b.logg.Infof("%d - has been registered at %s\n", b.students[b.nums], b.timeIn.Format(time.DateTime))

	return nil
}

func (b *Bot) handleEnter(chatID int64, userID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) error {
	prompts := []string{"Ваша учебная группа", "Ваш Логин от ЛК", "Ваш пароль от ЛК"}
	inputs, err := b.handleInput(chatID, up, key, prompts...)
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		return err
	}

	st, err := stud.Enter(inputs[0], inputs[1], inputs[2])
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		return err
	}
	b.MessageToUser(chatID, key, "Вход выполнен!")

	if err := b.IsSubOnChannel(chatID, userID, key); err != nil {
		b.logg.Infof("%d - doesn't subcribed\n", b.students[b.nums])
		return err
	}

	b.MessageToUser(userID, key, "Выбирите свой статус")
	b.timeIn = time.Now()
	b.logg.Infof("%d - has been entered at %s\n", b.students[b.nums], b.timeIn.Format(time.DateTime))

	b.ChangeStatusOfStudent(st, chatID, userID, up, key)

	return nil
}

func (b *Bot) ChangeStatusOfStudent(st *stud.Student, chatID int64, userID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) {
	message := make(chan string, 10)
	exit := make(chan struct{})

	go func() {
		defer close(exit)
		for tag := range message {
			if tag == "Автопосещение Вкл" {
				b.mutex.Lock()
				st.ChangeStatus(true)
				b.mutex.Unlock()
				b.MessageToUser(chatID, key, "Поздравляем! Вы отметились на паре!")
			} else if tag == "Автопосещение Выкл" {
				b.mutex.Lock()
				st.ChangeStatus(false)
				b.mutex.Unlock()
				b.MessageToUser(chatID, key, "Вы ушли с Пары")
			} else {
				b.logg.Infoln("channel closed")
				return
			}
		}
	}()
	for {
		select {
		case <-exit:
			return
		default:
			tag, _ := b.MessageToBot(chatID, up)
			message <- tag
		}
	}

}

func (b *Bot) MessageToUser(chatID int64, key interface{}, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = key
	_, err := b.bot.Send(msg)
	if err != nil {
		b.logg.Errorln(err)
	}
}

func (b *Bot) MessageToBot(ChatID int64, updates tgbotapi.UpdatesChannel) (string, error) {
	for {
		select {
		case update := <-updates:
			if update.Message != nil && update.Message.Chat.ID == ChatID {
				return update.Message.Text, nil
			}
		case <-time.After(5 * time.Second):
			continue
		}
	}
}

func (b *Bot) IsSubOnChannel(chatID, userID int64, key tgbotapi.ReplyKeyboardMarkup) error {
	errChan := make(chan error)
	go func() {
		if ok, err := b.checkSub(userID); !ok {
			//b.MessageToUser(chatID, key, err.Error())
			channelLink := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Перейти в канал", "https://t.me/name_of_your_chanel"),
				),
			)
			b.MessageToUser(chatID, channelLink, "Чтобы получить возможность отмечаться надо подписаться")
			errChan <- err
			return
		} else {
			errChan <- nil
			return
		}
	}()
	err := <-errChan
	close(errChan)
	return err
}

func (b *Bot) checkSub(userID int64) (bool, error) {
	sub, err := GetReqToTelegram(userID)
	if err != nil {
		b.logg.Errorln(err)
		return false, err
	}

	if !sub {
		b.logg.Errorln(err)
		return false, errors.New("Вы-не-подписаны")
	}

	return true, nil
}

func GetReqToTelegram(userID int64) (bool, error) {
	URL := fmt.Sprintf("https://api.telegram.org/bot%s/getChatMember?chat_id=@%s&user_id=%d", "token_of_your_bot", "name_of_your_channel", userID)

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
