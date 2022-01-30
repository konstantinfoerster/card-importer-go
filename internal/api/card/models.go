package card

import (
	"database/sql"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api"
	"reflect"
)

// Card A complete card including all faces (sides) and translations.
// The number of a card is unique per set
type Card struct {
	Id          sql.NullInt64
	CardSetCode string
	Name        string
	Number      string
	Border      string
	Rarity      string
	Layout      string
	Faces       []*Face
}

func (c *Card) isValid() error {
	if c.CardSetCode == "" {
		return fmt.Errorf("field 'cardSetCode' must not be empty")
	}
	if c.Number == "" {
		return fmt.Errorf("field 'number' must not be empty in card %s and set %s", c.Number, c.CardSetCode)
	}
	if c.Name == "" {
		return fmt.Errorf("field 'name' must not be empty in card %s and set %s", c.Number, c.CardSetCode)
	}
	if c.Rarity == "" {
		return fmt.Errorf("field 'rarity' must not be empty in card %s and set %s", c.Number, c.CardSetCode)
	}
	if c.Border == "" {
		return fmt.Errorf("field 'border' must not be empty in card %s and set %s", c.Number, c.CardSetCode)
	}
	if c.Layout == "" {
		return fmt.Errorf("field 'layout' must not be empty in card %s and set %s", c.Number, c.CardSetCode)
	}
	if len(c.Faces) == 0 {
		return fmt.Errorf("must at least have one face in card %s and set %s", c.Number, c.CardSetCode)
	}

	for i, face := range c.Faces {
		if face.Name == "" {
			return fmt.Errorf("field 'face[%d].name' must not be empty in card %s and set %s", i, c.Number, c.CardSetCode)
		}
	}

	return nil
}

func (c Card) Diff(other *Card) *api.Changeset {
	changes := api.NewChangeset()

	if c.Number != other.Number {
		changes.Add("Number", api.Changes{
			From: c.Number,
			To:   other.Number,
		})
	}
	if c.Name != other.Name {
		changes.Add("Name", api.Changes{
			From: c.Name,
			To:   other.Name,
		})
	}
	if c.Border != other.Border {
		changes.Add("Border", api.Changes{
			From: c.Border,
			To:   other.Border,
		})
	} else if c.Rarity != other.Rarity {
		changes.Add("Rarity", api.Changes{
			From: c.Rarity,
			To:   other.Rarity,
		})
	} else if c.CardSetCode != other.CardSetCode {
		changes.Add("CardSetCode", api.Changes{
			From: c.CardSetCode,
			To:   other.CardSetCode,
		})
	} else if c.Layout != other.Layout {
		changes.Add("Layout", api.Changes{
			From: c.Layout,
			To:   other.Layout,
		})
	}

	return &changes
}

// Face The face data of a card.
type Face struct {
	Id                sql.NullInt64
	Name              string
	Text              string
	FlavorText        string
	TypeLine          string
	MultiverseId      int32
	Artist            string
	ConvertedManaCost float64
	Colors            []string
	HandModifier      string
	LifeModifier      string
	Loyalty           string
	ManaCost          string
	Power             string
	Toughness         string
	Cardtypes         []string // A list of all card types of the card
	Supertypes        []string // A list of card supertypes found before em-dash.
	Subtypes          []string // A list of card subtypes found after em-dash.
	Translations      []Translation
}

// card 'Stitch in Time' has the same name on both faces but a different flavor text
func (f Face) isSame(other *Face) bool {
	return f.Name == other.Name && f.Text == other.Text && f.FlavorText == other.FlavorText
}

func (f Face) Diff(other *Face) *api.Changeset {
	changes := api.NewChangeset()

	if f.Name != other.Name {
		changes.Add("Name", api.Changes{
			From: f.Name,
			To:   other.Name,
		})
	}
	if f.Text != other.Text {
		changes.Add("Text", api.Changes{
			From: f.Text,
			To:   other.Text,
		})
	}
	if f.FlavorText != other.FlavorText {
		changes.Add("FlavorText", api.Changes{
			From: f.FlavorText,
			To:   other.FlavorText,
		})
	}
	if f.TypeLine != other.TypeLine {
		changes.Add("TypeLine", api.Changes{
			From: f.TypeLine,
			To:   other.TypeLine,
		})
	}
	if f.ConvertedManaCost != other.ConvertedManaCost {
		changes.Add("ConvertedManaCost", api.Changes{
			From: f.ConvertedManaCost,
			To:   other.ConvertedManaCost,
		})
	}
	if !reflect.DeepEqual(f.Colors, other.Colors) {
		if len(f.Colors) != 0 && len(other.Colors) != 0 {
			changes.Add("Colors", api.Changes{
				From: f.Colors,
				To:   other.Colors,
			})
		}
	}
	if f.Artist != other.Artist {
		changes.Add("Artist", api.Changes{
			From: f.Artist,
			To:   other.Artist,
		})
	}
	if f.HandModifier != other.HandModifier {
		changes.Add("HandModifier", api.Changes{
			From: f.HandModifier,
			To:   other.HandModifier,
		})
	}
	if f.LifeModifier != other.LifeModifier {
		changes.Add("LifeModifier", api.Changes{
			From: f.LifeModifier,
			To:   other.LifeModifier,
		})
	}
	if f.Loyalty != other.Loyalty {
		changes.Add("Loyalty", api.Changes{
			From: f.Loyalty,
			To:   other.Loyalty,
		})
	}
	if f.ManaCost != other.ManaCost {
		changes.Add("ManaCost", api.Changes{
			From: f.ManaCost,
			To:   other.ManaCost,
		})
	}
	if f.MultiverseId != other.MultiverseId {
		changes.Add("MultiverseId", api.Changes{
			From: f.MultiverseId,
			To:   other.MultiverseId,
		})
	}
	if f.Power != other.Power {
		changes.Add("Power", api.Changes{
			From: f.Power,
			To:   other.Power,
		})
	}
	if f.Toughness != other.Toughness {
		changes.Add("Toughness", api.Changes{
			From: f.Toughness,
			To:   other.Toughness,
		})
	}

	return &changes
}

// Translation The translation of the card. Does not include english (the default language).
type Translation struct {
	Name         string
	Text         string
	FlavorText   string
	TypeLine     string
	MultiverseId int32
	Lang         string
}

func (t Translation) Diff(other *Translation) *api.Changeset {
	changes := api.NewChangeset()

	if t.Name != other.Name {
		changes.Add("Name", api.Changes{
			From: t.Name,
			To:   other.Name,
		})
	}
	if t.Text != other.Text {
		changes.Add("Text", api.Changes{
			From: t.Text,
			To:   other.Text,
		})
	}
	if t.FlavorText != other.FlavorText {
		changes.Add("FlavorText", api.Changes{
			From: t.FlavorText,
			To:   other.FlavorText,
		})
	}
	if t.TypeLine != other.TypeLine {
		changes.Add("TypeLine", api.Changes{
			From: t.TypeLine,
			To:   other.TypeLine,
		})
	}
	if t.MultiverseId != other.MultiverseId {
		changes.Add("MultiverseId", api.Changes{
			From: t.MultiverseId,
			To:   other.MultiverseId,
		})
	}
	return &changes
}

// CharacteristicType A type of a card. Can be a Cardtype, Subtype or Superype
// Cardtype: Creature, Artifact, Instant, Enchantment ...
// Subtype: Archer, Shaman, Nomad, Nymph ...
// Supertype: Basic, Host, Legendary, Ongoing, Snow, World
type CharacteristicType struct {
	Id   sql.NullInt64
	Name string
}
