// Trello package for managing the trello api and collecting data
package trello

import (
	"fmt"
	"log"
	"strings"

	"github.com/adlio/trello"
)

type TrelloCredentials struct {
	Key   string `yaml:"key"`
	Token string `yaml:"token"`
}

type Card struct {
	Name string
}

// Client does the thing
type Client interface {
	// GetAllCards fetches all the cards for a board
	GetAllCards(board string) ([]Card, error)
}

// client implements client by doing HTTP
type client struct {
	client *trello.Client
}

func (c *client) GetAllCards(boardName string) ([]Card, error) {
	board, err := c.client.GetBoard(boardName, trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("no boardsssssss: %v", err)
	}
	lists, err := board.GetLists(trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("no listsssssssss: %v", err)
	}

	var allCards []Card
	for _, list := range lists {
		cards, err := list.GetCards(trello.Defaults())
		if err != nil {
			log.Printf("Cannot get cards: %v", err)
			continue
		}
		for _, card := range cards {
			allCards = append(allCards, Card{Name: card.Name})
		}
	}
	return allCards, nil
}

func ClientForConfig(creds *TrelloCredentials) Client {
	cl := trello.NewClient(creds.Key, creds.Token)

	return &client{
		client: cl,
	}
}

func FilterCards(tag string, cl Client) ([]Card, error) {
	cards, err := cl.GetAllCards("myboard")
	if err != nil {
		return nil, err
	}
	var res []Card
	for _, card := range cards {
		if !strings.HasPrefix(card.Name, tag) {
			continue
		}
		res = append(res, card)
	}

	return res, nil
}
