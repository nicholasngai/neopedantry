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

const pingName = "ping"

var interactions = []*discordgo.ApplicationCommand{
	{
		Name:        pingName,
		Description: "Ping!",
	},
}

var interactionHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	pingName: handlePing,
}

func registerInteractions(s *discordgo.Session, force bool) (err error) {
	shouldRegister := false

	if force {
		shouldRegister = true
	} else {
		// Get set of current interaction names.
		currInteractions, err := s.ApplicationCommands(s.State.User.ID, "")
		if err != nil {
			log.Errorln("Error getting current interactions:", err)
			return err
		}
		currNames := make(map[string]bool)
		for i := range currInteractions {
			currNames[currInteractions[i].Name] = true
		}
		log.Debugln("Current registered interaction names:", currNames)

		// Check if set matches current interactions.
		for i := range interactions {
			if _, ok := currNames[interactions[i].Name]; !ok {
				shouldRegister = true
				break
			}
		}
	}

	// Register commands if needed.
	if shouldRegister {
		log.Println("Difference in command names found (or forced); re-registering interactions")
		_, err = s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", interactions)
		return err
	}

	return nil
}

func handlePing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var err error

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Pong",
		},
	})
	if err != nil {
		log.Errorln("Error handling ping:", err)
	}
}

func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	log.WithFields(logrus.Fields{
		"name": name,
	}).Debugln("Handling command")
	if handler, ok := interactionHandlers[name]; ok {
		handler(s, i)
	} else {
		log.WithFields(logrus.Fields{
			"name": name,
		}).Warnln("Unknown interaction name")
	}
}

func main() {
	var err error

	var authToken string
	var forceRegisterInteractions bool
	var debug bool
	flag.StringVar(&authToken, "auth", "", "Authentication token for the Discord bot")
	flag.BoolVar(&forceRegisterInteractions, "forcereg", false, "Whether to force re-registration of interactions")
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

	// Add interaction handler.
	s.AddHandler(handleInteraction)

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
		log.Fatalln("Error registering interactions")
	}

	// Wait for interrupt.
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Terminating gracefully")
}
