package mtgjson

import (
	"context"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/cardset"
	dataset2 "github.com/konstantinfoerster/card-importer-go/internal/api/dataset"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"io"
	"strconv"
	"strings"
	"time"
)

var externalLangToLang = map[string]string{
	"German":  dataset2.SupportedLanguages[0],
	"English": dataset2.SupportedLanguages[1],
}

var doubleFaceCards = map[string]*card.Card{}

type dataset struct {
	setService  cardset.Service
	cardService card.Service
}

func NewImporter(setService cardset.Service, cardService card.Service) dataset2.Dataset {
	return &dataset{
		setService:  setService,
		cardService: cardService,
	}
}

func (imp *dataset) Import(r io.Reader) (*dataset2.Report, error) {
	errg, ctx := errgroup.WithContext(context.Background())

	for r := range parse(ctx, r) {
		r := r
		if r.Err != nil {
			return nil, r.Err
		}
		switch v := r.Result.(type) {
		case *mtgjsonCardSet:
			entry, err := mapToCardSet(v)
			if err != nil {
				return nil, err
			}
			if err := imp.setService.Import(entry); err != nil {
				return nil, err
			}
			log.Info().Msgf("Finished set %s", entry.Code)
		case *mtgjsonCard:
			entry, err := mapToCard(v)
			if err != nil {
				return nil, err
			}

			faceCount := expectedFaceCount(v)
			if faceCount > 1 {
				if collectFaces(faceCount, v, entry) {
					continue
				}
			}

			errg.Go(func() error {
				if err := imp.cardService.Import(entry); err != nil {
					return err
				}
				if log.Trace().Enabled() {
					log.Trace().Msgf("Finished card %s from set %s", entry.Number, entry.CardSetCode)
				}
				return nil
			})
		default:
			return nil, fmt.Errorf("found unknown result type %T\n", v)
		}
	}

	err := errg.Wait()
	if err != nil {
		return nil, err
	}

	if len(doubleFaceCards) != 0 {
		return nil, fmt.Errorf("found %d unprocessed double face cards %#v", len(doubleFaceCards), doubleFaceCards)
	}

	cardCount, err := imp.cardService.Count()
	if err != nil {
		return nil, err
	}
	setCount, err := imp.setService.Count()
	if err != nil {
		return nil, err
	}
	return &dataset2.Report{
		CardCount: cardCount,
		SetCount:  setCount,
	}, nil
}

func expectedFaceCount(v *mtgjsonCard) int {
	// meld cards have two sides but the back is only the first half of a card, so it does not count as a face
	if strings.ToUpper(v.Layout) == "MELD" {
		return 1
	}
	// card name contains all face names seperated by //
	return len(strings.Split(v.Name, "//"))
}

func collectFaces(faceCount int, v *mtgjsonCard, card *card.Card) bool {
	if faceCount > 1 {
		key := fmt.Sprintf("%s_%s", card.CardSetCode, v.Number)
		value, ok := doubleFaceCards[key]
		if !ok {
			doubleFaceCards[key] = card
			// continue collecting faces
			return true
		}

		card.Faces = append(card.Faces, value.Faces...)
		if faceCount != len(card.Faces) {
			doubleFaceCards[key] = card
			// continue collecting faces
			return true
		}
		delete(doubleFaceCards, key)
	}
	return false
}

func mapToCardSet(s *mtgjsonCardSet) (*cardset.CardSet, error) {
	released, err := time.Parse("2006-01-02", strings.TrimSpace(s.Released)) // ISO 8601 YYYY-MM-DD
	if err != nil {
		released = time.Time{}
	}

	var translations []cardset.Translation
	for _, t := range s.Translations {
		translation := cardset.Translation{
			Name: strings.TrimSpace(t.Name),
			Lang: externalLangToLang[strings.TrimSpace(t.Language)],
		}
		if translation.Name != "" && translation.Lang != "" {
			translations = append(translations, translation)
		}
	}
	set := &cardset.CardSet{
		Code:         strings.TrimSpace(s.Code),
		Name:         strings.TrimSpace(s.Name),
		TotalCount:   int(s.TotalCount),
		Released:     released,
		Block:        cardset.CardBlock{Block: strings.TrimSpace(s.Block)},
		Type:         strings.ToUpper(strings.TrimSpace(s.Type)),
		Translations: translations,
	}

	return set, nil
}

func mapToCard(c *mtgjsonCard) (*card.Card, error) {
	multiverseId, err := strToInt32(c.Identifiers.MultiverseId)
	if err != nil {
		return nil, fmt.Errorf("failed to convert 'MultiverseId' value %s into an int32. %w", c.Identifiers.MultiverseId, err)
	}

	var cardtypes []string
	for _, t := range c.Cardtypes {
		cardtypes = append(cardtypes, strings.TrimSpace(t))
	}
	var supertypes []string
	for _, t := range c.Supertypes {
		supertypes = append(supertypes, strings.TrimSpace(t))
	}
	var subtypes []string
	for _, t := range c.Subtypes {
		subtypes = append(subtypes, strings.TrimSpace(t))
	}

	var translations []card.Translation
	for _, fd := range c.ForeignData {
		lang := externalLangToLang[strings.TrimSpace(fd.Language)]
		t := card.Translation{
			Name:         strings.TrimSpace(fd.Name),
			Text:         strings.TrimSpace(fd.Text),
			FlavorText:   strings.TrimSpace(fd.FlavorText),
			TypeLine:     strings.TrimSpace(fd.Type),
			MultiverseId: fd.MultiverseId,
			Lang:         lang,
		}
		if t.Name != "" && t.Lang != "" {
			translations = append(translations, t)
		}
	}

	name := c.Name
	cmc := c.ConvertedManaCost
	if c.FaceName != "" {
		name = c.FaceName
		cmc = c.FaceConvertedManaCost
	}

	face := &card.Face{
		Name:              strings.TrimSpace(name),
		Artist:            strings.TrimSpace(c.Artist),
		ConvertedManaCost: cmc,
		Colors:            card.NewColors(c.Colors),
		Text:              strings.TrimSpace(c.Text),
		FlavorText:        strings.TrimSpace(c.FlavorText),
		HandModifier:      strings.TrimSpace(c.Hand),
		LifeModifier:      strings.TrimSpace(c.Life),
		Loyalty:           strings.TrimSpace(c.Loyalty),
		ManaCost:          strings.TrimSpace(c.ManaCost),
		Power:             strings.TrimSpace(c.Power),
		Toughness:         strings.TrimSpace(c.Toughness),
		MultiverseId:      multiverseId,
		TypeLine:          strings.TrimSpace(c.Type),
		Cardtypes:         cardtypes,
		Supertypes:        supertypes,
		Subtypes:          subtypes,
		Translations:      translations,
	}

	return &card.Card{
		Name:        strings.TrimSpace(c.Name),
		CardSetCode: strings.TrimSpace(c.Code),
		Number:      c.Number,
		Border:      strings.ToUpper(strings.TrimSpace(c.BorderColor)),
		Rarity:      strings.ToUpper(strings.TrimSpace(c.Rarity)),
		Layout:      strings.ToUpper(strings.TrimSpace(c.Layout)),
		Faces:       []*card.Face{face},
	}, nil
}

func strToInt32(in string) (int32, error) {
	s := strings.TrimSpace(in)
	if len(s) == 0 {
		return 0, nil
	}
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(i), nil
}
