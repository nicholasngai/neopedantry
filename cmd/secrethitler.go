package main

import (
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

const secretHitlerName = "secret-hitler"

var secretHitlerCommand = discordgo.ApplicationCommand{
	Name:        secretHitlerName,
	Description: "Find and stop the secret Hitler!",
}

var secretHitlerHandlers = []interface{}{
	handleSecretHitlerJoin,
	handleSecretHitlerLeave,
}

var channelGameIds = make(map[string]string)
var games = make(map[string]*game)
var gamesLock sync.RWMutex

func getJoinString(players map[string]bool) string {
	joinStr := "A Secret Hitler game is starting. Join now!\nCurrent players:"
	for userId := range players {
		joinStr += " <@" + userId + ">"
	}
	return joinStr
}

func handleSecretHitlerNewGame(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var err error

	// Create game if nonexistent.
	var createdGame bool
	func() {
		gamesLock.Lock()
		defer gamesLock.Unlock()
		game := makeGame(i.Interaction)

		// Send join message.
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: getJoinString(game.players),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Join",
								Style:    discordgo.PrimaryButton,
								CustomID: secretHitlerName + "-join",
							},
							discordgo.Button{
								Label:    "Leave",
								Style:    discordgo.SecondaryButton,
								CustomID: secretHitlerName + "-leave",
							},
						},
					},
				},
			},
		})
		if err != nil {
			log.Errorln("Error sending Secret Hitler join message:", err)
			return
		}

		// Get sent join message.
		m, err := s.InteractionResponse(s.State.User.ID, i.Interaction)
		if err != nil {
			log.Errorln("Error getting Secret Hitler join message:", err)
			return
		}

		channelGameIds[i.ChannelID] = m.ID
		games[m.ID] = game
		createdGame = true

		log.WithFields(logrus.Fields{
			"game":   secretHitlerName,
			"gameId": m.ID,
		}).Debugln("New game created")
	}()

	// Send error if game already exists in this channel.
	if !createdGame {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "A game is already running in this channel!",
				Flags:   1 << 6, // User-only visibility.
			},
		})
		return
	}
}

func handleSecretHitlerJoin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var err error

	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	if i.MessageComponentData().CustomID != secretHitlerName+"-join" {
		return
	}

	// Acquire game.
	var game *game
	func() {
		gamesLock.RLock()
		defer gamesLock.RUnlock()
		game = games[channelGameIds[i.ChannelID]]
	}()

	if game == nil {
		log.WithFields(logrus.Fields{
			"game":   secretHitlerName,
			"user":   i.Member.User.ID,
			"gameId": i.Message.ID,
		}).Warnln("User attempted to join nonexistent game")
		return
	}

	// Atomically add user to player list and update message.
	var playerAdded bool
	func() {
		// Acquire lock.
		game.lock.Lock()
		defer game.lock.Unlock()

		// Probe players list and return if already present.
		if _, ok := game.players[i.Member.User.ID]; ok {
			return
		}

		// Add player to list.
		game.players[i.Member.User.ID] = true
		playerAdded = true

		// Update join message.
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: getJoinString(game.players),
			},
		})
		if err != nil {
			log.Errorln("Error updating join message:", err)
		}
	}()

	// Return if player not added.
	if !playerAdded {
		log.WithFields(logrus.Fields{
			"game":   secretHitlerName,
			"user":   i.Member.User.ID,
			"gameId": i.Message.ID,
		}).Debugln("User attempted to join game when already joined")

		// Send error and return if user is already in game.
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are already in the game!",
				Flags:   1 << 6, // User-only visibility.
			},
		})
		if err != nil {
			log.Errorln("Error sending already-joined message:", err)
		}
		return
	}

	log.WithFields(logrus.Fields{
		"game":   secretHitlerName,
		"user":   i.Member.User.ID,
		"gameId": i.Message.ID,
	}).Debugln("User joined game")
}

func handleSecretHitlerLeave(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var err error

	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	if i.MessageComponentData().CustomID != secretHitlerName+"-leave" {
		return
	}

	// Acquire game.
	var game *game
	func() {
		gamesLock.RLock()
		defer gamesLock.RUnlock()
		game = games[channelGameIds[i.ChannelID]]
	}()

	if game == nil {
		log.WithFields(logrus.Fields{
			"game":   secretHitlerName,
			"user":   i.Member.User.ID,
			"gameId": i.Message.ID,
		}).Warnln("User attempted to leave nonexistent game")
		return
	}

	// Atomically remove player.
	var playerRemoved bool
	func() {
		game.lock.Lock()
		defer game.lock.Unlock()

		// Probe players list and return if already present.
		if _, ok := game.players[i.Member.User.ID]; !ok {
			return
		}

		// Remove player from list.
		delete(game.players, i.Member.User.ID)
		playerRemoved = true

		// Update join message.
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: getJoinString(game.players),
			},
		})
		if err != nil {
			log.Errorln("Error updating join message:", err)
		}
	}()

	// Return if player not removed.
	if !playerRemoved {
		log.WithFields(logrus.Fields{
			"game":   secretHitlerName,
			"user":   i.Member.User.ID,
			"gameId": i.ChannelID,
		}).Debugln("Player not in game tried to leave")

		// Send error and return if user is not in game.
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are not in the game!",
				Flags:   1 << 6, // User-only visibility.
			},
		})
		if err != nil {
			log.Errorln("Error sending not-in-game message:", err)
		}
		return
	}

	log.WithFields(logrus.Fields{
		"game":   secretHitlerName,
		"user":   i.Member.User.ID,
		"gameId": i.Message.ID,
	}).Debugln("User left game")
}
