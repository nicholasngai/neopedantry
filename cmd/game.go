package main

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

type gameState int

const (
	lobbyState gameState = iota
)

type game struct {
	state              gameState
	players            map[string]bool
	newGameInteraction *discordgo.Interaction
	lock               sync.RWMutex
}

func makeGame(i *discordgo.Interaction) *game {
	return &game{
		state:              lobbyState,
		newGameInteraction: i,
		players:            make(map[string]bool),
	}
}
