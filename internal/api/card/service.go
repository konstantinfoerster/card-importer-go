package card

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"reflect"
	"strings"
)

type Service interface {
	Import(card *Card) error
	Count() (int, error)
}

type cardService struct {
	dao *PostgresCardDao
}

func NewService(dao *PostgresCardDao) Service {
	return &cardService{
		dao: dao,
	}
}

func (s *cardService) Count() (int, error) {
	return s.dao.Count()
}

func (s *cardService) Import(card *Card) error {
	if card == nil {
		// Skip nil card
		return nil
	}
	if card.Name == "" {
		return fmt.Errorf("field 'name' must not be empty")
	}
	if card.CardSetCode == "" {
		return fmt.Errorf("field 'cardSetCode' must not be empty in card %s", card.Name)
	}
	if card.Number == "" {
		return fmt.Errorf("field 'number' must not be empty in card %s and set %s", card.Name, card.CardSetCode)
	}
	if card.Rarity == "" {
		return fmt.Errorf("field 'rarity' must not be empty in card %s and set %s", card.Name, card.CardSetCode)
	}
	if card.Border == "" {
		return fmt.Errorf("field 'border' must not be empty in card %s and set %s", card.Name, card.CardSetCode)
	}
	if card.Layout == "" {
		return fmt.Errorf("field 'layout' must not be empty in card %s and set %s", card.Name, card.CardSetCode)
	}
	existingCard, err := s.dao.FindUniqueCard(card.Name, card.CardSetCode, card.Number)
	if err != nil {
		return fmt.Errorf("failed to find card with name %s, code %s and number %s. %v", card.Name, card.CardSetCode, card.Number, err)
	}

	if existingCard == nil {
		if err := s.dao.CreateCard(card); err != nil {
			if err != nil && strings.Contains(err.Error(), "duplicate key") {
				log.Warn().Msgf("Card with name %s, set code %s and number %s already exists, skipping card", card.Name, card.CardSetCode, card.Number)
				return nil
			}
			log.Error().Msgf("Failed to create card %#v %v", card, err)
			return err
		}
		if log.Trace().Enabled() {
			log.Trace().Msgf("Created card %s %s", card.Name, card.CardSetCode)
		}
	} else {
		card.Id = existingCard.Id
		changed := false

		if card.Artist != existingCard.Artist {
			log.Info().Msgf("Update card.Artist from '%v' to '%v'", existingCard.Artist, card.Artist)
			changed = true
		} else if card.Border != existingCard.Border {
			log.Info().Msgf("Update card.Border from '%v' to '%v'", existingCard.Border, card.Border)
			changed = true
		} else if card.ConvertedManaCost != existingCard.ConvertedManaCost {
			log.Info().Msgf("Update card.ConvertedManaCost from '%v' to '%v'", existingCard.ConvertedManaCost, card.ConvertedManaCost)
			changed = true
		} else if !reflect.DeepEqual(card.Colors, existingCard.Colors) {
			if len(card.Colors) != 0 && len(existingCard.Colors) != 0 {
				log.Info().Msgf("Update card.Colors from '%v' to '%v'", existingCard.Colors, card.Colors)
				changed = true
			}
		} else if card.Text != existingCard.Text {
			log.Info().Msgf("Update card.Text from '%v' to '%v'", existingCard.Text, card.Text)
			changed = true
		} else if card.FlavorText != existingCard.FlavorText {
			log.Info().Msgf("Update card.FlavorText from '%v' to '%v'", existingCard.FlavorText, card.FlavorText)
			changed = true
		} else if card.Layout != existingCard.Layout {
			log.Info().Msgf("Update card.Layout from '%v' to '%v'", existingCard.Layout, card.Layout)
			changed = true
		} else if card.HandModifier != existingCard.HandModifier {
			log.Info().Msgf("Update card.HandModifier from '%v' to '%v'", existingCard.HandModifier, card.HandModifier)
			changed = true
		} else if card.LifeModifier != existingCard.LifeModifier {
			log.Info().Msgf("Update card.LifeModifier from '%v' to '%v'", existingCard.LifeModifier, card.LifeModifier)
			changed = true
		} else if card.Loyalty != existingCard.Loyalty {
			log.Info().Msgf("Update card.Loyalty from '%v' to '%v'", existingCard.Loyalty, card.Loyalty)
			changed = true
		} else if card.ManaCost != existingCard.ManaCost {
			log.Info().Msgf("Update card.ManaCost from '%v' to '%v'", existingCard.ManaCost, card.ManaCost)
			changed = true
		} else if card.MultiverseId != existingCard.MultiverseId {
			log.Info().Msgf("Update card.MultiverseId from '%v' to '%v'", existingCard.MultiverseId, card.MultiverseId)
			changed = true
		} else if card.Power != existingCard.Power {
			log.Info().Msgf("Update card.Power from '%v' to '%v'", existingCard.Power, card.Power)
			changed = true
		} else if card.Toughness != existingCard.Toughness {
			log.Info().Msgf("Update card.Toughness from '%v' to '%v'", existingCard.Toughness, card.Toughness)
			changed = true
		} else if card.Rarity != existingCard.Rarity {
			log.Info().Msgf("Update card.Rarity from '%v' to '%v'", existingCard.Rarity, card.Rarity)
			changed = true
		} else if card.Number != existingCard.Number {
			log.Info().Msgf("Update card.Number from '%v' to '%v'", existingCard.Number, card.Number)
			changed = true
		} else if card.FullType != existingCard.FullType {
			log.Info().Msgf("Update card.FullType from '%v' to '%v'", existingCard.FullType, card.FullType)
			changed = true
		}

		if changed {
			log.Info().Msgf("Update card %s %s", card.Name, card.CardSetCode)
			if err := s.dao.UpdateCard(card); err != nil {
				return err
			}
		}
	}

	cardId := card.Id.Int64

	err = s.mergeSubTypes(card.Subtypes, cardId, existingCard == nil)
	if err != nil {
		return err
	}
	err = s.mergeSuperTypes(card.Supertypes, cardId, existingCard == nil)
	if err != nil {
		return err
	}
	err = s.mergeCardTypes(card.Cardtypes, cardId, existingCard == nil)
	if err != nil {
		return err
	}
	return s.mergeTranslations(card.Translations, cardId, existingCard == nil)
}

func (s *cardService) mergeSubTypes(tt []string, cardId int64, isNew bool) error {
	if isNew && len(tt) == 0 {
		// skip, nothing to create or delete
		return nil
	}
	return s.dao.withTransaction(func(txDao *PostgresCardDao) error {
		return mergeTypes(txDao.subType, tt, cardId, isNew)
	})
}

func (s *cardService) mergeSuperTypes(tt []string, cardId int64, isNew bool) error {
	if isNew && len(tt) == 0 {
		// skip, nothing to create or delete
		return nil
	}
	return s.dao.withTransaction(func(txDao *PostgresCardDao) error {
		return mergeTypes(txDao.superType, tt, cardId, isNew)
	})
}

func (s *cardService) mergeCardTypes(tt []string, cardId int64, isNew bool) error {
	if isNew && len(tt) == 0 {
		// skip, nothing to create or delete
		return nil
	}
	return s.dao.withTransaction(func(txDao *PostgresCardDao) error {
		return mergeTypes(txDao.cardType, tt, cardId, isNew)
	})
}

func mergeTypes(dao TypeDao, tt []string, cardId int64, isNew bool) error {
	toCreate := tt
	if !isNew {
		assignedTypes, err := dao.FindAssignments(cardId)
		if err != nil {
			return fmt.Errorf("failed to get assigned types %v", err)
		}
		var toRemove []int64
		// remove all entries that are already assigned to the card
		for _, t := range assignedTypes {
			if ok, pos := contains(toCreate, t.Name); ok {
				toCreate = remove(toCreate, pos)
			} else {
				toRemove = append(toRemove, t.Id.Int64)
			}
		}
		if len(toRemove) > 0 {
			if err := dao.DeleteAssignments(cardId, toRemove...); err != nil {
				return fmt.Errorf("failed to removed assigned types %v", err)
			}
		}

		if len(toCreate) == 0 {
			return nil
		}
	}

	types, err := dao.Find(toCreate...)
	if err != nil {
		return fmt.Errorf("failed to find types %v %v", toCreate, err)
	}
	for _, t := range toCreate {
		var entry *CharacteristicType

		for _, existingType := range types {
			if t == existingType.Name {
				entry = existingType
			}
		}
		if entry == nil {
			entry, err = dao.Create(t)
			if err != nil {
				return err
			}
		}

		if err := dao.AssignToCard(cardId, entry.Id.Int64); err != nil {
			return fmt.Errorf("failed to assign type %v to card %d %v", entry, cardId, err)
		}
	}

	return nil
}

func (s *cardService) mergeTranslations(tt []Translation, cardId int64, isNew bool) error {
	var toCreate []Translation
	toCreate = append(toCreate, tt...)
	if !isNew {
		existingTranslations, err := s.dao.FindTranslations(cardId)
		if err != nil {
			return fmt.Errorf("failed to get existing translations %v", err)
		}

		for _, existing := range existingTranslations {
			if ok, pos := containsTranslation(tt, *existing); ok {
				toCreate = removeTranslation(toCreate, pos)

				newT := tt[pos]
				changed := false
				if existing.Name != newT.Name {
					log.Info().Msgf("Update translation.Name from '%v' to '%v'", existing.Name, newT.Name)
					changed = true
				} else if existing.Text != newT.Text {
					log.Info().Msgf("Update translation.Text from '%v' to '%v'", existing.Text, newT.Text)
					changed = true
				} else if existing.FlavorText != newT.FlavorText {
					log.Info().Msgf("Update translation.FlavorText from '%v' to '%v'", existing.FlavorText, newT.FlavorText)
					changed = true
				} else if existing.FullType != newT.FullType {
					log.Info().Msgf("Update translation.FullType from '%v' to '%v'", existing.FullType, newT.FullType)
					changed = true
				} else if existing.MultiverseId != newT.MultiverseId {
					log.Info().Msgf("Update translation.MultiverseId from '%v' to '%v'", existing.MultiverseId, newT.MultiverseId)
					changed = true
				}
				if changed {
					log.Info().Msgf("Update translation for card %v and language %v from %v to %v", cardId, existing.Lang, existing.Name, newT.Name)
					if err := s.dao.UpdateTranslation(cardId, &newT); err != nil {
						return err
					}
				}
			} else {
				if err := s.dao.DeleteTranslation(cardId, existing.Lang); err != nil {
					return err
				}
			}
		}
	}

	for _, t := range toCreate {
		if err := s.dao.CreateTranslation(cardId, &t); err != nil {
			return err
		}
		continue
	}

	return nil
}

func remove(arr []string, pos int) []string {
	arr[pos] = arr[len(arr)-1]
	return arr[:len(arr)-1]
}

func contains(s []string, term string) (bool, int) {
	for i, e := range s {
		if e == term {
			return true, i
		}
	}
	return false, 0
}

func removeTranslation(arr []Translation, pos int) []Translation {
	arr[pos] = arr[len(arr)-1]
	return arr[:len(arr)-1]
}

func containsTranslation(tt []Translation, t Translation) (bool, int) {
	for i, e := range tt {
		if e.Lang == t.Lang {
			return true, i
		}
	}
	return false, 0
}
