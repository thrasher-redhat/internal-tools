package trello

import (
	"fmt"
	"log"

	"github.com/adlio/trello"
)

type TrelloCredentials struct {
	Key   string `yaml:"key"`
	Token string `yaml:"token"`
}

// Client does the thing
type Client interface {
	// GetAllCards fetches all the cards for a board
	GetAllCards(board string) ([]*trello.Card, error)
}

// client implements client by doing HTTP
type client struct {
	client *trello.Client
}

func (c *client) GetAllCards(boardName string) ([]*trello.Card, error) {
	board, err := c.client.GetBoard(boardName, trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("no boardsssssss: %v", err)
	}
	lists, err := board.GetLists(trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("no listsssssssss: %v", err)
	}

	var allCards []*trello.Card
	for _, list := range lists {
		cards, err := list.GetCards(trello.Defaults())
		if err != nil {
			log.Printf("Cannot get cards: %v", err)
			continue
		}
		allCards = append(allCards, cards...)
	}
	return allCards, nil
}

func ClientForConfig(creds *TrelloCredentials) Client {
	cl := trello.NewClient(creds.Key, creds.Token)

	return &client{
		client: cl,
	}
}

func FilterCards(tag string, cl Client) ([]*trello.Card, error) {

}
