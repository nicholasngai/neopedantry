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

var commands = []*discordgo.ApplicationCommand{
	&secretHitlerCommand,
}

var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	secretHitlerName: handleSecretHitlerNewGame,
}

var handlerLists = [][]interface{}{
	secretHitlerHandlers,
}

func registerInteractions(s *discordgo.Session, force bool) (err error) {
	shouldRegister := false

	if force {
		shouldRegister = true
	} else {
		// Get set of current command names.
		currInteractions, err := s.ApplicationCommands(s.State.User.ID, "")
		if err != nil {
			log.Errorln("Error getting current commands:", err)
			return err
		}
		currNames := make(map[string]bool)
		for i := range currInteractions {
			currNames[currInteractions[i].Name] = true
		}
		log.Debugln("Current registered command names:", currNames)

		// Check if set matches current commands.
		for i := range commands {
			if _, ok := currNames[commands[i].Name]; !ok {
				shouldRegister = true
				break
			}
		}
	}

	// Register commands if needed.
	if shouldRegister {
		log.Println("Difference in command names found (or forced); re-registering commands")
		_, err = s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", commands)
		return err
	}

	return nil
}

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	// Handle commands.
	name := i.ApplicationCommandData().Name
	log.WithFields(logrus.Fields{
		"cmd": name,
	}).Debugln("Handling command")
	if handler, ok := commandHandlers[name]; ok {
		handler(s, i)
	} else {
		log.WithFields(logrus.Fields{
			"name": name,
		}).Warnln("Unknown command name")
	}
}

func main() {
	var err error

	var authToken string
	var forceRegisterInteractions bool
	var debug bool
	flag.StringVar(&authToken, "auth", "", "Authentication token for the Discord bot")
	flag.BoolVar(&forceRegisterInteractions, "forcereg", false, "Whether to force re-registration of commands")
	flag.BoolVar(&debug, "debug", false, "Debug log level")
	flag.Parse()

	if authToken == "" {
		flag.Usage()
		os.Exit(1)
	}

	if debug {
		log.Level = logrus.DebugLevel
	}

	// Construct session.
	s, err := discordgo.New("Bot " + authToken)
	if err != nil {
		log.Fatalln("Error creating discordgo instance:", err)
	}

	// Add command handler.
	s.AddHandler(handleCommand)

	// Add additional handlers.
	for _, handlers := range handlerLists {
		for _, handler := range handlers {
			s.AddHandler(handler)
		}
	}

	// TODO Register intents.

	// Connect to Discord.
	err = s.Open()
	if err != nil {
		log.Fatalln("Error connecting to Discord:", err)
	}
	defer s.Close()
	log.Println("Bot started successfully")

	// Register commands.
	err = registerInteractions(s, forceRegisterInteractions)
	if err != nil {
		log.Fatalln("Error registering commands")
	}

	// Wait for interrupt.
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Terminating gracefully")
}
