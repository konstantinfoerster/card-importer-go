package mtgjson

import (
	"context"
	"fmt"
	"github.com/konstantinfoerster/card-importer-go/internal/api"
	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/konstantinfoerster/card-importer-go/internal/api/cardset"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"io"
	"strconv"
	"strings"
	"time"
)

var languages = [2]string{"deu", "eng"}

var externalLangToLang = map[string]string{
	"German":  languages[0],
	"English": languages[1],
}

type Importer struct {
	setService  cardset.Service
	cardService card.Service
}

func NewImporter(setService cardset.Service, cardService card.Service) api.Importer {
	return &Importer{
		setService:  setService,
		cardService: cardService,
	}
}

func (imp *Importer) Import(r io.Reader) (*api.Report, error) {
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
			errg.Go(func() error {
				if err := imp.cardService.Import(entry); err != nil {
					// retry
					if strings.Contains(err.Error(), "duplicate key") {
						log.Warn().Msgf("Retry card %s from set %s after short sleep. Reason: %v", entry.Name, entry.CardSetCode, err)
						time.Sleep(200 * time.Millisecond)
						if err := imp.cardService.Import(entry); err != nil {
							return err
						}
					} else {
						return err
					}
				}
				if log.Trace().Enabled() {
					log.Trace().Msgf("Finished card %s", entry.Name)
				}
				return nil
			})
		default:
			return nil, fmt.Errorf("found unknown result type %T\n", v)
		}
	}

	cardCount, err := imp.cardService.Count()
	if err != nil {
		return nil, err
	}
	setCount, err := imp.setService.Count()
	if err != nil {
		return nil, err
	}
	return &api.Report{
		CardCount: cardCount,
		SetCount:  setCount,
	}, errg.Wait()
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
	var colors []string
	for _, color := range c.Colors {
		colors = append(colors, strings.TrimSpace(color)) // TODO maybe to upper case?
	}

	hand, err := strToInt(c.Hand)
	if err != nil {
		return nil, fmt.Errorf("failed to convert 'Hand' value %s into an int. %v", c.Hand, err)
	}
	life, err := strToInt(c.Life)
	if err != nil {
		return nil, fmt.Errorf("failed to convert 'Life' value %s into an int. %v", c.Life, err)
	}
	multiverseId, err := strToInt64(c.Identifiers.MultiverseId)
	if err != nil {
		return nil, fmt.Errorf("failed to convert 'MultiverseId' value %s into an int64. %v", c.Identifiers.MultiverseId, err)
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
		translation := card.Translation{
			Name:         strings.TrimSpace(fd.Name),
			Text:         strings.TrimSpace(fd.Text),
			FlavorText:   strings.TrimSpace(fd.FlavorText),
			FullType:     strings.TrimSpace(fd.Type),
			MultiverseId: fd.MultiverseId,
			Lang:         lang,
		}
		if translation.Name != "" && translation.Lang != "" {
			translations = append(translations, translation)
		}
	}

	name := c.Name
	cmc := c.ConvertedManaCost
	// TODO Here we make one double face card into two cards. Is that really what we want
	if c.FaceName != "" {
		name = c.FaceName
		cmc = c.FaceConvertedManaCost
	}
	return &card.Card{
		CardSetCode:       strings.TrimSpace(c.Code),
		Name:              strings.TrimSpace(name),
		Artist:            strings.TrimSpace(c.Artist),
		Border:            strings.ToUpper(strings.TrimSpace(c.BorderColor)),
		ConvertedManaCost: cmc, // can this be an int?
		Colors:            colors,
		Text:              strings.TrimSpace(c.Text),
		FlavorText:        strings.TrimSpace(c.FlavorText),
		Layout:            strings.ToUpper(strings.TrimSpace(c.Layout)),
		HandModifier:      hand,
		LifeModifier:      life,
		Loyalty:           c.Loyalty,
		ManaCost:          strings.TrimSpace(c.ManaCost),
		Power:             strings.TrimSpace(c.Power),
		Toughness:         strings.TrimSpace(c.Toughness),
		Rarity:            strings.ToUpper(strings.TrimSpace(c.Rarity)),
		Number:            strings.TrimSpace(c.Number),
		MultiverseId:      multiverseId,
		FullType:          strings.TrimSpace(c.Type),
		Cardtypes:         cardtypes,
		Supertypes:        supertypes,
		Subtypes:          subtypes,
		Translations:      translations,
	}, nil
}

func strToInt(in string) (int, error) {
	s := strings.TrimSpace(in)
	if len(s) == 0 {
		return 0, nil
	}
	return strconv.Atoi(s)
}
func strToInt64(in string) (int64, error) {
	s := strings.TrimSpace(in)
	if len(s) == 0 {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}
