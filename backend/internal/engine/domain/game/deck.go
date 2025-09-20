package game

import (
	"math/rand"
	"time"

	"github.com/alexclewontin/riverboat/eval"
)

// Deck represents a deck of playing cards
type Deck struct {
	cards []Card
	index int
}

// NewDeck creates a new standard 52-card deck
func NewDeck() *Deck {
	deck := &Deck{
		cards: make([]Card, 52),
		index: 0,
	}

	// Create all 52 cards
	suits := []string{"♠", "♥", "♦", "♣"}
	ranks := []string{"2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}

	cardIndex := 0
	for _, suit := range suits {
		for i, rank := range ranks {
			deck.cards[cardIndex] = Card{
				Suit:  suit,
				Rank:  rank,
				Value: i + 2, // 2=2, 3=3, ..., T=10, J=11, Q=12, K=13, A=14
			}
			cardIndex++
		}
	}

	return deck
}

// Shuffle shuffles the deck using Fisher-Yates algorithm
func (d *Deck) Shuffle() {
	d.index = 0
	rand.Seed(time.Now().UnixNano())

	for i := len(d.cards) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	}
}

// Deal returns the next card from the deck
func (d *Deck) Deal() Card {
	if d.index >= len(d.cards) {
		// Deck is empty, reshuffle
		d.Shuffle()
	}

	card := d.cards[d.index]
	d.index++
	return card
}

// Reset resets the deck to the beginning
func (d *Deck) Reset() {
	d.index = 0
}

// RemainingCards returns the number of cards left in the deck
func (d *Deck) RemainingCards() int {
	return len(d.cards) - d.index
}

// ToRiverboatCard converts our Card to riverboat Card for hand evaluation
func ToRiverboatCard(card Card) eval.Card {
	// Convert our card format to riverboat's format
	// This is a mapping function between our domain model and the evaluation library

	var suit int
	switch card.Suit {
	case "♠":
		suit = 0
	case "♥":
		suit = 1
	case "♦":
		suit = 2
	case "♣":
		suit = 3
	default:
		suit = 0
	}

	var rank int
	switch card.Rank {
	case "2":
		rank = 0
	case "3":
		rank = 1
	case "4":
		rank = 2
	case "5":
		rank = 3
	case "6":
		rank = 4
	case "7":
		rank = 5
	case "8":
		rank = 6
	case "9":
		rank = 7
	case "T":
		rank = 8
	case "J":
		rank = 9
	case "Q":
		rank = 10
	case "K":
		rank = 11
	case "A":
		rank = 12
	default:
		rank = 0
	}

	// Create riverboat card using bit representation
	// Riverboat Card format: rank + suit*13
	return eval.Card(rank + suit*13)
}

// EvaluateHand evaluates the best 5-card hand from 7 cards (2 hole + 5 community)
func EvaluateHand(holeCards []Card, communityCards []Card) ([]Card, int, string) {
	if len(holeCards) != 2 || len(communityCards) != 5 {
		return nil, 0, "invalid"
	}

	// Convert to riverboat cards
	hole1 := ToRiverboatCard(holeCards[0])
	hole2 := ToRiverboatCard(holeCards[1])
	community1 := ToRiverboatCard(communityCards[0])
	community2 := ToRiverboatCard(communityCards[1])
	community3 := ToRiverboatCard(communityCards[2])
	community4 := ToRiverboatCard(communityCards[3])
	community5 := ToRiverboatCard(communityCards[4])

	// Use riverboat evaluation
	bestHand, score := eval.BestFiveOfSeven(hole1, hole2, community1, community2, community3, community4, community5)

	// Convert back to our Card format
	resultCards := make([]Card, 5)
	for i, rbCard := range bestHand {
		resultCards[i] = fromRiverboatCard(rbCard)
	}

	// Determine hand ranking name
	handRank := getHandRankName(score)

	return resultCards, score, handRank
}

// fromRiverboatCard converts riverboat Card back to our Card
func fromRiverboatCard(rbCard eval.Card) Card {
	suits := []string{"♠", "♥", "♦", "♣"}
	ranks := []string{"2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"}

	suit := int(rbCard) % 4
	rank := int(rbCard) / 4

	return Card{
		Suit:  suits[suit],
		Rank:  ranks[rank],
		Value: rank + 2,
	}
}

// getHandRankName converts score to hand ranking name
func getHandRankName(score int) string {
	// Riverboat uses lower scores for better hands
	// This is a simplified mapping - you may want to use riverboat's own classification
	if score <= 10 {
		return "Royal Flush"
	} else if score <= 166 {
		return "Straight Flush"
	} else if score <= 322 {
		return "Four of a Kind"
	} else if score <= 1599 {
		return "Full House"
	} else if score <= 1609 {
		return "Flush"
	} else if score <= 1619 {
		return "Straight"
	} else if score <= 2467 {
		return "Three of a Kind"
	} else if score <= 3325 {
		return "Two Pair"
	} else if score <= 6185 {
		return "One Pair"
	} else {
		return "High Card"
	}
}

