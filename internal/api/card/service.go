package card

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
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
	if err := card.isValid(); err != nil {
		return fmt.Errorf("card is invalid %w", err)
	}

	var isNewCard bool
	err := s.dao.withTransaction(func(txDao *PostgresCardDao) error {
		existingCard, err := txDao.FindUniqueCard(card.CardSetCode, card.Number)
		if err != nil {
			return fmt.Errorf("failed to find card with code %s and number %s. %w", card.CardSetCode, card.Number, err)
		}

		if existingCard == nil {
			if err := txDao.CreateCard(card); err != nil {
				log.Error().Err(err).Msgf("Failed to create card %#v", card)
				return err
			}
			if log.Trace().Enabled() {
				log.Trace().Msgf("Created card %s from set %s", card.Number, card.CardSetCode)
			}
			isNewCard = true
		} else {
			card.Id = existingCard.Id

			diff := card.Diff(existingCard)
			if diff.HasChanges() {
				log.Info().Msgf("Update card %s from set %s with changes %#v", card.Name, card.CardSetCode, diff)
				if err := txDao.UpdateCard(card); err != nil {
					return err
				}
			}
		}

		return mergeCardFaces(txDao, card.Faces[:], card.Id.Int64, isNewCard)
	})
	if err != nil {
		return err
	}

	for _, f := range card.Faces {
		if !f.Id.Valid {
			return fmt.Errorf("expected face %s for card %s and set %s to be created", f.Name, card.Number, card.CardSetCode)
		}

		faceId := f.Id.Int64
		if err := mergeSubTypes(s.dao, f.Subtypes, faceId, isNewCard); err != nil {
			return err
		}
		if err := mergeSuperTypes(s.dao, f.Supertypes, faceId, isNewCard); err != nil {
			return err
		}
		if err := mergeCardTypes(s.dao, f.Cardtypes, faceId, isNewCard); err != nil {
			return err
		}

		if err := mergeFaceTranslations(s.dao, f.Translations, faceId, isNewCard); err != nil {
			return err
		}
	}
	return nil
}

func mergeFaceTranslations(dao *PostgresCardDao, tt []Translation, faceId int64, isNewCard bool) error {
	return dao.withTransaction(func(txDao *PostgresCardDao) error {
		return mergeTranslations(txDao, tt, faceId, isNewCard)
	})
}
func mergeCardFaces(dao *PostgresCardDao, ff []*Face, cardId int64, isNewCard bool) error {
	var toCreate []*Face
	toCreate = append(toCreate, ff...)
	if !isNewCard {
		assignedFaces, err := dao.FindAssignedFaces(cardId)
		if err != nil {
			return fmt.Errorf("failed to get assigned faces %w", err)
		}
		for _, assigned := range assignedFaces {
			if ok, pos := containsFace(toCreate, assigned); ok {
				faceToMerge := toCreate[pos]
				faceToMerge.Id = assigned.Id
				diff := faceToMerge.Diff(assigned)
				if diff.HasChanges() {
					log.Info().Msgf("Update face %s for card %v with changes %#v", faceToMerge.Name, cardId, diff)
					if err := dao.UpdateFace(faceToMerge); err != nil {
						return err
					}
				}
				toCreate = removeFace(toCreate, pos)
			} else {
				faceId := assigned.Id.Int64
				if err := dao.DeleteAllTranslation(faceId); err != nil {
					return err
				}
				if err := dao.DeleteFace(faceId); err != nil {
					return err
				}
				log.Info().Msgf("Deleted card face %v for card %v", assigned.Name, cardId)
			}
		}
	}

	for _, newFace := range toCreate {
		if err := dao.AddFace(cardId, newFace); err != nil {
			return err
		}
	}

	assigned, err := dao.FindAssignedFaces(cardId)
	if err != nil {
		return fmt.Errorf("failed to get assigned faces %w", err)
	}
	if len(assigned) != len(ff) {
		return fmt.Errorf("unexpected face count assigned to card %d, expected %d but found %d", cardId, len(ff), len(assigned))
	}
	return nil
}

func mergeSubTypes(dao *PostgresCardDao, tt []string, faceId int64, isNewCard bool) error {
	return withDuplicateKeyRetry(func() error {
		return dao.withTransaction(func(txDao *PostgresCardDao) error {
			return mergeTypes(txDao.subType, tt, faceId, isNewCard)
		})
	})
}

func mergeSuperTypes(dao *PostgresCardDao, tt []string, faceId int64, isNewCard bool) error {
	return withDuplicateKeyRetry(func() error {
		return dao.withTransaction(func(txDao *PostgresCardDao) error {
			return mergeTypes(txDao.superType, tt, faceId, isNewCard)
		})
	})
}

func mergeCardTypes(dao *PostgresCardDao, tt []string, faceId int64, isNewCard bool) error {
	return withDuplicateKeyRetry(func() error {
		return dao.withTransaction(func(txDao *PostgresCardDao) error {
			return mergeTypes(txDao.cardType, tt, faceId, isNewCard)
		})
	})
}

func withDuplicateKeyRetry(fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "duplicate key") {
		log.Warn().Msgf("retry error  after short sleep %v", err)
		time.Sleep(200 * time.Millisecond)
		return fn()
	}
	return err
}

func mergeTypes(dao TypeDao, tt []string, faceId int64, isNewCard bool) error {
	if isNewCard && len(tt) == 0 {
		// skip, nothing to create or delete
		return nil
	}

	var toCreate []string
	toCreate = append(toCreate, tt...)
	if !isNewCard {
		assignedTypes, err := dao.FindAssignments(faceId)
		if err != nil {
			return fmt.Errorf("failed to get assigned types %w", err)
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
			if err := dao.DeleteAssignments(faceId, toRemove...); err != nil {
				return fmt.Errorf("failed to removed assigned types %w", err)
			}
		}

		if len(toCreate) == 0 {
			return nil
		}
	}

	types, err := dao.Find(toCreate...)
	if err != nil {
		return fmt.Errorf("failed to find types %v %w", toCreate, err)
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
				return fmt.Errorf("failed to create type %s %w", t, err)
			}
		}

		if err := dao.AssignToFace(faceId, entry.Id.Int64); err != nil {
			return fmt.Errorf("failed to assign type %v to card %d %w", entry, faceId, err)
		}
	}

	return nil
}

func mergeTranslations(dao *PostgresCardDao, tt []Translation, faceId int64, isNewCard bool) error {
	var toCreate []Translation
	toCreate = append(toCreate, tt...)
	if !isNewCard {
		assignedTranslations, err := dao.FindTranslations(faceId)
		if err != nil {
			return fmt.Errorf("failed to get existing translations %w", err)
		}

		for _, assigned := range assignedTranslations {
			if ok, pos := containsTranslation(tt, *assigned); ok {
				toCreate = removeTranslation(toCreate, *assigned)

				translation := &tt[pos]

				diff := translation.Diff(assigned)
				if diff.HasChanges() {
					log.Info().Msgf("Update translation for face %v and language %v with changes %#v", faceId, assigned.Lang, diff)
					if err := dao.UpdateTranslation(faceId, translation); err != nil {
						return err
					}
				}
			} else {
				if err := dao.DeleteTranslation(faceId, assigned.Lang); err != nil {
					return err
				}
			}
		}
	}

	for _, t := range toCreate {
		if err := dao.AddTranslation(faceId, &t); err != nil {
			return err
		}
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

func removeFace(arr []*Face, pos int) []*Face {
	arr[pos] = arr[len(arr)-1]
	return arr[:len(arr)-1]
}

func containsFace(arr []*Face, searchTerm *Face) (bool, int) {
	for i, f := range arr {
		if f.isSame(searchTerm) {
			return true, i
		}
	}
	return false, 0
}

func removeTranslation(arr []Translation, toRemove Translation) []Translation {
	if ok, pos := containsTranslation(arr, toRemove); ok {
		arr[pos] = arr[len(arr)-1]
		return arr[:len(arr)-1]
	}
	return arr
}

func containsTranslation(arr []Translation, t Translation) (bool, int) {
	for i, e := range arr {
		if e.Lang == t.Lang {
			return true, i
		}
	}
	return false, 0
}
