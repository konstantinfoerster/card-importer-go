package card

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/diff"
	"io"
	"strings"
)

var PartCard = "CARD"
var PartFace = "FACE"

// Card A complete card including all faces (sides) and translations.
// The number of a card is unique per set
type Card struct {
	Id          PrimaryId
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

func (c Card) Diff(other *Card) *diff.Changeset {
	changes := diff.NewChangeset()

	if c.Number != other.Number {
		changes.Add("Number", diff.Changes{
			From: c.Number,
			To:   other.Number,
		})
	}
	if c.Name != other.Name {
		changes.Add("Name", diff.Changes{
			From: c.Name,
			To:   other.Name,
		})
	}
	if c.Border != other.Border {
		changes.Add("Border", diff.Changes{
			From: c.Border,
			To:   other.Border,
		})
	} else if c.Rarity != other.Rarity {
		changes.Add("Rarity", diff.Changes{
			From: c.Rarity,
			To:   other.Rarity,
		})
	} else if c.CardSetCode != other.CardSetCode {
		changes.Add("CardSetCode", diff.Changes{
			From: c.CardSetCode,
			To:   other.CardSetCode,
		})
	} else if c.Layout != other.Layout {
		changes.Add("Layout", diff.Changes{
			From: c.Layout,
			To:   other.Layout,
		})
	}

	return &changes
}

// Face The face data of a card.
type Face struct {
	Id                PrimaryId
	Name              string
	Text              string
	FlavorText        string
	TypeLine          string
	MultiverseId      int32
	Artist            string
	ConvertedManaCost float64
	Colors            Colors
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

func (f Face) Diff(other *Face) *diff.Changeset {
	changes := diff.NewChangeset()

	if f.Name != other.Name {
		changes.Add("Name", diff.Changes{
			From: f.Name,
			To:   other.Name,
		})
	}
	if f.Text != other.Text {
		changes.Add("Text", diff.Changes{
			From: f.Text,
			To:   other.Text,
		})
	}
	if f.FlavorText != other.FlavorText {
		changes.Add("FlavorText", diff.Changes{
			From: f.FlavorText,
			To:   other.FlavorText,
		})
	}
	if f.TypeLine != other.TypeLine {
		changes.Add("TypeLine", diff.Changes{
			From: f.TypeLine,
			To:   other.TypeLine,
		})
	}
	if f.ConvertedManaCost != other.ConvertedManaCost {
		changes.Add("ConvertedManaCost", diff.Changes{
			From: f.ConvertedManaCost,
			To:   other.ConvertedManaCost,
		})
	}
	if !f.Colors.Equal(other.Colors) {
		changes.Add("Colors", diff.Changes{
			From: f.Colors,
			To:   other.Colors,
		})
	}
	if f.Artist != other.Artist {
		changes.Add("Artist", diff.Changes{
			From: f.Artist,
			To:   other.Artist,
		})
	}
	if f.HandModifier != other.HandModifier {
		changes.Add("HandModifier", diff.Changes{
			From: f.HandModifier,
			To:   other.HandModifier,
		})
	}
	if f.LifeModifier != other.LifeModifier {
		changes.Add("LifeModifier", diff.Changes{
			From: f.LifeModifier,
			To:   other.LifeModifier,
		})
	}
	if f.Loyalty != other.Loyalty {
		changes.Add("Loyalty", diff.Changes{
			From: f.Loyalty,
			To:   other.Loyalty,
		})
	}
	if f.ManaCost != other.ManaCost {
		changes.Add("ManaCost", diff.Changes{
			From: f.ManaCost,
			To:   other.ManaCost,
		})
	}
	if f.MultiverseId != other.MultiverseId {
		changes.Add("MultiverseId", diff.Changes{
			From: f.MultiverseId,
			To:   other.MultiverseId,
		})
	}
	if f.Power != other.Power {
		changes.Add("Power", diff.Changes{
			From: f.Power,
			To:   other.Power,
		})
	}
	if f.Toughness != other.Toughness {
		changes.Add("Toughness", diff.Changes{
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

func (t Translation) Diff(other *Translation) *diff.Changeset {
	changes := diff.NewChangeset()

	if t.Name != other.Name {
		changes.Add("Name", diff.Changes{
			From: t.Name,
			To:   other.Name,
		})
	}
	if t.Text != other.Text {
		changes.Add("Text", diff.Changes{
			From: t.Text,
			To:   other.Text,
		})
	}
	if t.FlavorText != other.FlavorText {
		changes.Add("FlavorText", diff.Changes{
			From: t.FlavorText,
			To:   other.FlavorText,
		})
	}
	if t.TypeLine != other.TypeLine {
		changes.Add("TypeLine", diff.Changes{
			From: t.TypeLine,
			To:   other.TypeLine,
		})
	}
	if t.MultiverseId != other.MultiverseId {
		changes.Add("MultiverseId", diff.Changes{
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
	Id   PrimaryId
	Name string
}

func NewPrimaryId(id int64) PrimaryId {
	return PrimaryId{sql.NullInt64{Int64: id, Valid: true}}
}

type PrimaryId struct {
	sql.NullInt64
}

func (v PrimaryId) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return json.Marshal(v.Int64)
	} else {
		return json.Marshal(nil)
	}
}

func (v *PrimaryId) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *int64
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		v.Valid = true
		v.Int64 = *x
	} else {
		v.Valid = false
	}
	return nil
}

func (v PrimaryId) Get() int64 {
	return v.Int64
}

func NewColors(colors []string) Colors {
	var trimmed []string
	for _, c := range colors {
		trimmed = append(trimmed, strings.TrimSpace(c))
	}
	valid := len(trimmed) > 0
	colorsRow := strings.Join(trimmed, ",")
	return Colors{NullString: sql.NullString{String: colorsRow, Valid: valid}, Array: colors}
}

type Colors struct {
	sql.NullString
	Array []string
}

func (v Colors) Equal(other Colors) bool {
	return v.String != other.String || len(v.Array) != len(other.Array)
}

func (v Colors) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return json.Marshal(v.String)
	} else {
		return json.Marshal(nil)
	}
}

func (v *Colors) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *string
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil && len(*x) > 0 {
		v.Valid = true
		v.String = *x
		v.Array = strings.Split(*x, ",")
	} else {
		v.Valid = false
	}
	return nil
}

type CardImage struct {
	Id        PrimaryId
	ImagePath string
	Lang      string
	CardId    PrimaryId
	FaceId    PrimaryId
	MimeType  string
	File      io.ReadCloser
}

func (img *CardImage) getFilePrefix() (string, error) {
	if img.CardId.Valid {
		return fmt.Sprintf("card-%d", img.CardId.Get()), nil
	}

	if img.FaceId.Valid {
		return fmt.Sprintf("face-%d", img.FaceId.Get()), nil
	}
	return "", fmt.Errorf("failed to build file prefix, no valid id provided")
}

func (img *CardImage) BuildFilename() (string, error) {
	prefix, err := img.getFilePrefix()
	if err != nil {
		return "", fmt.Errorf("can't build file name reason: %w", err)
	}
	ct := strings.Split(img.MimeType, ";")[0]
	switch ct {
	case "application/json":
		return prefix + ".json", nil
	case "application/zip":
		return prefix + ".zip", nil
	case "image/jpeg":
		return prefix + ".jpg", nil
	case "image/png":
		return prefix + ".png", nil
	default:
		return "", fmt.Errorf("unsupported content type %s", ct)
	}
}
