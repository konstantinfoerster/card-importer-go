package card

import "database/sql"

type Card struct {
	Id                sql.NullInt64
	MultiverseId      int64
	Name              string
	Artist            string
	Border            string
	ConvertedManaCost float64
	Colors            []string
	Text              string
	FlavorText        string
	Layout            string
	HandModifier      int
	LifeModifier      int
	Loyalty           string
	ManaCost          string
	Power             string
	Toughness         string
	Rarity            string
	Number            string
	FullType          string
	Cardtypes         []string // A list of all card types of the card
	Supertypes        []string // A list of card supertypes found before em-dash.
	Subtypes          []string // A list of card subtypes found after em-dash.
	CardSetCode       string
	Translations      []Translation
}

type Translation struct {
	Name         string
	Text         string
	FlavorText   string
	FullType     string
	MultiverseId int64
	Lang         string
}

// The types are in the end just lookup tables ?? Maybe it would make sense to inline them?
// Why not include it in the CardTranslation??
type CharacteristicType struct {
	Id   sql.NullInt64
	Name string
}
