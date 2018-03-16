package trello

import (
	"fmt"
	"testing"
)

type fakeClient struct {
	// cardList maps board names to card lists
	cardList map[string][]Card
}

func (c *fakeClient) GetAllCards(board string) ([]Card, error) {
	cards, ok := c.cardList[board]
	if !ok {
		return nil, fmt.Errorf("No board found %q", board)
	}
	return cards, nil
}

func TestFilterTags(t *testing.T) {
	cl := &fakeClient{
		cardList: map[string][]Card{
			"myboard": {
				{Name: "foobar 1"},
				{Name: "foobar 2"},
				//{Name: "foobar 3"},
				{Name: "bazquux 1"},
			},
		},
	}

	filteredCards, err := FilterCards("foobar", cl)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(filteredCards) != 2 {
		t.Fatalf("expected 2 cards, got %#v", filteredCards)
	}
}
