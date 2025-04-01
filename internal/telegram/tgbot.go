package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Vladroon22/TG-Bot/internal/database"
	"github.com/Vladroon22/TG-Bot/internal/entity"
	"github.com/Vladroon22/TG-Bot/internal/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
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
		mutex:    sync.RWMutex{},
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

func (b *Bot) StopUpdates() {
	b.bot.StopReceivingUpdates()
}

func (b *Bot) Run(ctx context.Context) error {
	b.logg.Infoln("Bot connected:", b.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)
	//wg := sync.WaitGroup{}
	//workerPool := make(chan struct{}, 10)
	errCh := make(chan string, 10)

	for {
		select {
		case <-ctx.Done():
			//wg.Wait()
			return errors.New("Bot has shutted down")
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			//workerPool <- struct{}{}
			//wg.Add(1)
			//go func(update tgbotapi.Update) {
			//	defer wg.Done()
			//	defer func() { <-workerPool }()

			chatID := update.Message.Chat.ID
			userID := update.Message.From.ID
			userName := update.Message.Chat.UserName

			switch update.Message.Text {
			case "Регистрация":
				if err := b.handleRegistration(userName, chatID, userID, updates, key); err != nil {
					b.timeIn = time.Now()
					errCh <- err.Error() + " - " + b.timeIn.Format(time.DateTime)
				}
			case "Вход":
				if err := b.handleEnter(userName, chatID, userID, updates, key); err != nil {
					b.timeIn = time.Now()
					errCh <- err.Error() + " - " + b.timeIn.Format(time.DateTime)
				}
			}
		//	}(update)
		case err := <-errCh:
			b.logg.Errorln(err)
		}
	}

}

func (b *Bot) handleInput(ctx context.Context, chatID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) ([]string, error) {
	b.MessageToUser(chatID, key, "Введите вашу учебную группу (пробел) Ваш Логин от ЛК (пробел) Ваш пароль от ЛК")
	input, err := b.MessageToBot(ctx, chatID, up)
	if err != nil {
		b.logg.Errorln(err)
		return nil, errors.New("прозошла ошибка. введите данные еще раз")
	}

	inputs := strings.Split(input, " ")

	if len(inputs) != 3 {
		b.logg.Errorln("Некорректное количество данных: ", len(inputs))
		return nil, errors.New("пожалуйста, введите ровно три значения через точку: учебная группа(пробел)логин(пробел)пароль")
	}

	return inputs, nil
}

func (b *Bot) handleRegistration(userName string, chatID, userID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputs, err := b.handleInput(ctx, chatID, up, key)
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		return err
	}
	groupname := inputs[0]
	login := inputs[1]
	password := inputs[2]

	enc_pass, err := utils.Hashing(password)
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, "Ошибка на сервере (password hashing)")
		return err
	}

	if err := b.AddNewStudent(ctx, groupname, login, string(enc_pass)); err != nil {
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

func (b *Bot) handleEnter(userName string, chatID int64, userID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inputs, err := b.handleInput(ctx, chatID, up, key)
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		return err
	}

	b.mutex.Lock()
	if _, ok := b.students[int(userID)]; !ok {
		b.students[int(userID)] = userName
	}
	b.mutex.Unlock()

	st, err := b.AuthStudent(ctx, inputs[0], inputs[1], inputs[2])
	if err != nil {
		b.logg.Errorln(err)
		b.MessageToUser(chatID, key, err.Error())
		b.MessageToUser(chatID, key, "Попробуйте еще раз")
		return err
	}
	b.MessageToUser(chatID, key, "Вход выполнен!")

	if err := b.IsSubOnChannel(ctx, chatID, userID, key); err != nil {
		return err
	}

	if err := b.ChangeStatusOfStudent(ctx, st, chatID, userID, up, key); err != nil {
		b.MessageToUser(chatID, key, err.Error())
		return err
	}

	return nil
}

func (b *Bot) ChangeStatusOfStudent(c context.Context, st *entity.Student, chatID int64, userID int64, up tgbotapi.UpdatesChannel, key tgbotapi.ReplyKeyboardMarkup) error {
	b.MessageToUser(userID, key, "Выберите свой статус")

	b.timeIn = time.Now()
	b.mutex.RLock()
	user := b.students[int(userID)]
	b.mutex.RUnlock()
	b.logg.Infof("%s - has been entered at %s\n", user, b.timeIn.Format(time.DateTime))

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	tag, _ := b.MessageToBot(c, chatID, up)

	switch tag {
	case "Автопосещение Вкл":
		tm := b.timeIn
		if err := b.statusChange(ctx, st, chatID, key, true, "Поздравляем! Вы отметились на паре!"); err != nil {
			b.logg.Errorln(err.Error(), "for", user, tm.Format(time.DateTime))
			return err
		}
		b.logg.Infoln("success tagging for", user, tm.Format(time.DateTime))
		return nil
	case "Автопосещение Выкл":
		tm := b.timeIn
		if err := b.statusChange(ctx, st, chatID, key, false, "Вы ушли с Пары"); err != nil {
			b.logg.Errorln(err.Error(), "for", user, tm.Format(time.DateTime))
			return err
		}
		b.logg.Infoln("success tagging for", user, tm.Format(time.DateTime))
		return nil
	default:
		b.logg.Infoln("unknown choice -", tag)
		b.MessageToUser(chatID, key, "Неизвестная команда -> авторизуйтесь заново")
		return nil
	}
}

func (b *Bot) statusChange(c context.Context, st *entity.Student, chatID int64, key tgbotapi.ReplyKeyboardMarkup, status bool, statusMsg string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if err := database.UpdateStudentSub(c, st, status); err != nil {
		return err
	}

	b.MessageToUser(chatID, key, statusMsg)
	return nil
}

func (b *Bot) MessageToUser(chatID int64, key any, text string) {
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

	errCh := make(chan error, 1)
	defer close(errCh)

	go func() {
		select {
		case <-ctx.Done():
			errCh <- errors.New("введите данные заново")
		default:
			return
		}
	}()

	for {
		select {
		case update := <-updates:
			if update.Message != nil && update.Message.Chat.ID == ChatID {
				return update.Message.Text, nil
			}
		case <-ctx.Done():
			err := <-errCh
			return "", err
		}
	}
}

func (b *Bot) IsSubOnChannel(ctx context.Context, chatID, userID int64, key tgbotapi.ReplyKeyboardMarkup) error {
	if ok, err := b.checkSub(ctx, userID); !ok {
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

func (b *Bot) checkSub(ctx context.Context, userID int64) (bool, error) {
	sub, err := getReqToTelegram(ctx, userID)
	if err != nil {
		b.logg.Errorln(err)
		return false, err
	}

	if !sub {
		b.mutex.RLock()
		user := b.students[int(userID)]
		b.mutex.RUnlock()
		return false, errors.New(user + " не подписан")
	}

	return true, nil
}

func (b *Bot) AddNewStudent(c context.Context, s ...string) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	student := entity.Student{
		ID:           bson.NewObjectID().String(),
		GroupName:    s[0],
		Login:        s[1],
		Password:     s[2],
		Subscription: false,
	}

	if err := database.Insert(ctx, &student); err != nil {
		return err
	}

	return nil
}

func (b *Bot) AuthStudent(c context.Context, s ...string) (*entity.Student, error) {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	student := entity.Student{
		GroupName:    s[0],
		Login:        s[1],
		Password:     s[2],
		Subscription: false,
	}

	id, err := database.CheckEquillity(ctx, &student)
	if err != nil {
		return nil, err
	}
	student.ID = id

	return &student, nil
}

func getReqToTelegram(c context.Context, userID int64) (bool, error) {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	errCh := make(chan error, 1)
	defer close(errCh)

	go func() {
		select {
		case <-ctx.Done():
			errCh <- errors.New("проверка подписки отменена -> авторизуйтесь заново")
		default:
			return
		}
	}()

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
