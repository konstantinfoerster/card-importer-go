package mtgjson

type mtgjsonCardSet struct {
	Code         string        `json:"code"`
	Name         string        `json:"name"`
	Block        string        `json:"block"`
	Type         string        `json:"type"`
	TotalCount   float64       `json:"totalSetSize"`
	Released     string        `json:"releaseDate"`
	Translations []translation `json:"translations"`
}

type translation struct {
	Language string
	Name     string
}

type mtgjsonCard struct {
	Name                  string        `json:"name"`
	Code                  string        `json:"setCode"`
	Artist                string        `json:"artist"`
	Side                  string        `json:"side"`
	ConvertedManaCost     float64       `json:"convertedManaCost"`     // deprecated use manaValue instead
	FaceConvertedManaCost float64       `json:"faceConvertedManaCost"` // deprecated use faceManaValue instead
	FaceName              string        `json:"faceName"`
	FlavorText            string        `json:"flavorText"`
	Text                  string        `json:"text"`
	Hand                  string        `json:"hand"`
	Life                  string        `json:"life"`
	Loyalty               string        `json:"loyalty"`
	Layout                string        `json:"layout"`
	ManaCost              string        `json:"manaCost"`
	Number                string        `json:"number"`
	Power                 string        `json:"power"`
	Toughness             string        `json:"toughness"`
	Rarity                string        `json:"rarity"`
	Type                  string        `json:"type"`
	Identifiers           identifier    `json:"identifiers"`
	Colors                []string      `json:"colors"`
	ForeignData           []foreignData `json:"foreignData"`
	Cardtypes             []string      `json:"types"`
	Subtypes              []string      `json:"subtypes"`
	Supertypes            []string      `json:"supertypes"`
	CardParts             []string      `json:"cardParts"`
	Finishes              []string      `json:"finishes"`
	BorderColor           string        `json:"borderColor"`
	Alternative           bool          `json:"isAlternative"`
}

type foreignData struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Language     string `json:"language"`
	MultiverseID int32  `json:"multiverseId"`
	Text         string `json:"text"`
	FlavorText   string `json:"flavorText"`
}

type identifier struct {
	MultiverseID string `json:"multiverseId"`
}
