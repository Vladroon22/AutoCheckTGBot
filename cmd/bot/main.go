package main

import (
	"github.com/Vladroon22/TG-Bot/internal/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func main() {
	logg := logrus.New()

	bot, err := tgbotapi.NewBotAPI(telegram.MustToken())
	if err != nil {
		logg.Fatalln(err)
	}

	bot.Debug = false

	telebot := telegram.NewBot(bot, logg)
	go logg.Fatalln(telebot.Run())
}
