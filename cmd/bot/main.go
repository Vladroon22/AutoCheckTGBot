package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Vladroon22/TG-Bot/internal/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	logg := logrus.New()

	if err := godotenv.Load(); err != nil {
		logg.Fatalln(err)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("token"))
	if err != nil {
		logg.Fatalln(err)
	}

	bot.Debug = false

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	telebot := telegram.NewBot(bot, logg)
	go func() {
		if err := telebot.Run(ctx); err != nil {
			logg.Infoln(err)
			return
		}
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGINT)
	<-exit

	go func() { telebot.StopUpdates() }()

	logg.Infoln("Gracefull shutdown")
}
