package main

import (
	"context"
	"os"

	"github.com/Vladroon22/TG-Bot/internal/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func main() {
	logg := logrus.New()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("token"))
	if err != nil {
		logg.Fatalln(err)
	}

	bot.Debug = false

	telebot := telegram.NewBot(bot, logg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go logg.Fatalln("Bot failed: " + telebot.Run(ctx).Error())

}
