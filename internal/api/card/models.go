package card

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/konstantinfoerster/card-importer-go/internal/api/diff"
	"github.com/konstantinfoerster/card-importer-go/internal/fetch"
)

// Card A complete card including all faces (sides) and translations.
// The number of a card is unique per set.
type Card struct {
	ID          PrimaryID
	CardSetCode string
	Name        string
	Number      string
	Border      string // ENUM
	Rarity      string // ENUM
	Layout      string // ENUM
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

func (c *Card) Diff(other *Card) *diff.Changeset {
	changes := diff.New()

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

	switch {
	case c.Border != other.Border:
		changes.Add("Border", diff.Changes{
			From: c.Border,
			To:   other.Border,
		})
	case c.Rarity != other.Rarity:
		changes.Add("Rarity", diff.Changes{
			From: c.Rarity,
			To:   other.Rarity,
		})
	case c.CardSetCode != other.CardSetCode:
		changes.Add("CardSetCode", diff.Changes{
			From: c.CardSetCode,
			To:   other.CardSetCode,
		})

	case c.Layout != other.Layout:
		changes.Add("Layout", diff.Changes{
			From: c.Layout,
			To:   other.Layout,
		})
	}

	return changes
}

// Face The face data of a card.
type Face struct {
	ID                PrimaryID
	Name              string
	Text              string
	FlavorText        string
	TypeLine          string
	MultiverseID      int32
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

// isSame Compares the identities of two faces.
// Card 'Stitch in Time' from set SLD has the same name on both faces but a different flavor text.
func (f Face) isSame(other *Face) bool {
	return f.Name == other.Name && f.Text == other.Text && f.FlavorText == other.FlavorText
}

func (f Face) couldBeSame(other *Face) bool {
	return f.Name == other.Name
}

// Diff Compares the faces and returns all differences.
func (f Face) Diff(other *Face) *diff.Changeset {
	changes := diff.New()

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
	if f.Colors.String != other.Colors.String {
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
	if f.MultiverseID != other.MultiverseID {
		changes.Add("MultiverseID", diff.Changes{
			From: f.MultiverseID,
			To:   other.MultiverseID,
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

	return changes
}

// Translation The translation of the card. Does not include english (the default language).
type Translation struct {
	Name         string
	Text         string
	FlavorText   string
	TypeLine     string
	MultiverseID int32
	Lang         string
}

// Diff Compares the translations and returns all differences.
func (t Translation) Diff(other *Translation) *diff.Changeset {
	changes := diff.New()

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
	if t.MultiverseID != other.MultiverseID {
		changes.Add("MultiverseId", diff.Changes{
			From: t.MultiverseID,
			To:   other.MultiverseID,
		})
	}

	return changes
}

// CharacteristicType A type of card. Can be a Cardtype, Subtype or Superype.
// Cardtype: Creature, Artifact, Instant, Enchantment ... .
// Subtype: Archer, Shaman, Nomad, Nymph ... .
// Supertype: Basic, Host, Legendary, Ongoing, Snow, World.
type CharacteristicType struct {
	ID   PrimaryID
	Name string
}

func NewPrimaryID(id int64) PrimaryID {
	return PrimaryID{sql.NullInt64{Int64: id, Valid: true}}
}

type PrimaryID struct {
	sql.NullInt64
}

func (v *PrimaryID) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return json.Marshal(v.Int64)
	}

	return json.Marshal(nil)
}

func (v *PrimaryID) UnmarshalJSON(data []byte) error {
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

func (v PrimaryID) Get() int64 {
	return v.Int64
}

func NewColors(colors []string) Colors {
	var trimmed []string
	for _, c := range colors {
		trimmed = append(trimmed, strings.TrimSpace(c))
	}
	valid := len(trimmed) > 0
	colorsRow := strings.Join(trimmed, ",")

	return Colors{NullString: sql.NullString{String: colorsRow, Valid: valid}}
}

type Colors struct {
	sql.NullString
}

// UnmarshalJSON Unmarshal string into colors struct. Required for the card page query.
func (v *Colors) UnmarshalJSON(data []byte) error {
	// Unmarshalling into a pointer will let us detect null
	var x *string
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil && len(*x) > 0 {
		v.Valid = true
		v.String = *x
	} else {
		v.Valid = false
	}

	return nil
}

type Image struct {
	ID           PrimaryID
	Lang         string
	CardID       PrimaryID
	FaceID       PrimaryID
	ImagePath    string
	MimeType     string
	PHash        uint64
	PHashRotated uint64
}

func (img *Image) getFilePrefix() (string, error) {
	// check face id first since card id is always set
	if img.FaceID.Valid {
		return fmt.Sprintf("face-%d", img.FaceID.Get()), nil
	}
	if img.CardID.Valid {
		return fmt.Sprintf("card-%d", img.CardID.Get()), nil
	}

	return "", fmt.Errorf("failed to build file prefix, no valid id provided")
}

// BuildFilename Returns file name based on the image mime type.
func (img *Image) BuildFilename() (string, error) {
	prefix, err := img.getFilePrefix()
	if err != nil {
		return "", fmt.Errorf("can't build file name reason: %w", err)
	}

	return fetch.NewMimeType(img.MimeType).BuildFilename(prefix)
}
