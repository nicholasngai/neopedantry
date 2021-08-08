package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
}

func main() {
	var err error

	var authToken string
	flag.StringVar(&authToken, "auth", "", "Authentication token for the Discord bot")
	flag.Parse()

	if authToken == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Construct session.
	s, err := discordgo.New("Bot " + authToken)
	if err != nil {
		log.Fatalln("Error creating discordgo instance:", err)
	}

	// Add message handler.
	s.AddHandler(handleMessage)

	// TODO Register intents.

	// Connect to Discord.
	err = s.Open()
	if err != nil {
		log.Fatalln("Error connecting to Discord:", err)
	}
	defer s.Close()
	log.Println("Bot started successfully")

	// Wait for interrupt.
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Terminating gracefully")
}
