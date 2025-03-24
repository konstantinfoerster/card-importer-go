package cards

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
)

type Service[T any] interface {
	Import(data T) error
	Count() (int, error)
}

type cardService struct {
	dao *PostgresCardDao
}

func NewCardService(dao *PostgresCardDao) Service[*Card] {
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
		var err error
		card, isNewCard, err = mergeCard(txDao, card)
		if err != nil {
			return err
		}

		return mergeCardFaces(txDao, card.Faces, card.ID.Int64, isNewCard)
	})
	if err != nil {
		return err
	}

	for _, f := range card.Faces {
		if !f.ID.Valid {
			return fmt.Errorf("expected face %s of card %s and set %s to be created", f.Name, card.Number, card.CardSetCode)
		}

		faceID := f.ID.Int64
		if err := mergeSubTypes(s.dao, f.Subtypes, faceID, isNewCard); err != nil {
			return err
		}
		if err := mergeSuperTypes(s.dao, f.Supertypes, faceID, isNewCard); err != nil {
			return err
		}
		if err := mergeCardTypes(s.dao, f.Cardtypes, faceID, isNewCard); err != nil {
			return err
		}

		err := s.dao.withTransaction(func(txDao *PostgresCardDao) error {
			return mergeFaceTranslations(txDao, f.Translations, faceID, isNewCard)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func mergeCard(txDao *PostgresCardDao, c *Card) (*Card, bool, error) {
	existingCard, err := txDao.FindUniqueCard(c.CardSetCode, c.Number)
	if err != nil {
		if !errors.Is(err, ErrEntryNotFound) {
			return nil, false, fmt.Errorf("failed to find card with code %s and number %s. %w", c.CardSetCode, c.Number, err)
		}
	}

	if existingCard == nil {
		if err := txDao.CreateCard(c); err != nil {
			log.Error().Err(err).Msgf("Failed to create card %#v", c)

			return nil, false, err
		}
		if e := log.Trace(); e.Enabled() {
			e.Msgf("Created card %s from set %s", c.Number, c.CardSetCode)
		}

		return c, true, nil
	}

	c.ID = existingCard.ID

	diff := existingCard.Diff(c)
	if diff.HasChanges() {
		log.Info().Msgf("Update card %s from set %s with changes %s", c.Name, c.CardSetCode, diff.String())
		if err := txDao.UpdateCard(c); err != nil {
			return nil, false, err
		}
	}

	return c, false, err
}

func mergeCardFaces(dao *PostgresCardDao, ff []*Face, cardID int64, isNewCard bool) error {
	var incomingFaces []*Face
	incomingFaces = append(incomingFaces, ff...)
	if !isNewCard {
		existingFaces, err := dao.FindAssignedFaces(cardID)
		if err != nil {
			return fmt.Errorf("failed to get assigned faces %w", err)
		}
		for _, existingFace := range existingFaces {
			if ok, pos := containsFace(incomingFaces, existingFace); ok {
				incomingFace := incomingFaces[pos]
				incomingFace.ID = existingFace.ID

				diff := existingFace.Diff(incomingFace)
				if diff.HasChanges() {
					log.Info().Msgf("Update face %s of card %v with changes %s", incomingFace.Name, cardID, diff.String())
					if err := dao.UpdateFace(incomingFace); err != nil {
						return err
					}
				}
				incomingFaces = removeFace(incomingFaces, pos)

				continue
			}
			faceID := existingFace.ID.Int64
			if err := dao.DeleteFace(faceID); err != nil {
				return err
			}
			log.Warn().Msgf("Deleted card face %v of card %v", existingFace.Name, cardID)
		}
	}

	for _, newFace := range incomingFaces {
		if err := dao.AddFace(cardID, newFace); err != nil {
			return err
		}
	}

	assigned, err := dao.FindAssignedFaces(cardID)
	if err != nil {
		return fmt.Errorf("failed to get assigned faces %w", err)
	}
	if len(assigned) != len(ff) {
		return fmt.Errorf("unexpected face count assigned to card %d, expected %d but found %d",
			cardID, len(ff), len(assigned))
	}

	return nil
}

func mergeSubTypes(dao *PostgresCardDao, tt []string, faceID int64, isNewCard bool) error {
	return dao.withTransaction(func(txDao *PostgresCardDao) error {
		return mergeTypes(txDao.subType, tt, faceID, isNewCard)
	})
}

func mergeSuperTypes(dao *PostgresCardDao, tt []string, faceID int64, isNewCard bool) error {
	return dao.withTransaction(func(txDao *PostgresCardDao) error {
		return mergeTypes(txDao.superType, tt, faceID, isNewCard)
	})
}

func mergeCardTypes(dao *PostgresCardDao, tt []string, faceID int64, isNewCard bool) error {
	return dao.withTransaction(func(txDao *PostgresCardDao) error {
		return mergeTypes(txDao.cardType, tt, faceID, isNewCard)
	})
}

func mergeTypes(dao TypeDao, tt []string, faceID int64, isNewCard bool) error {
	if isNewCard && len(tt) == 0 {
		// skip, nothing to create or delete
		return nil
	}

	var toCreate []string
	toCreate = append(toCreate, tt...)
	if !isNewCard {
		existingTypes, err := dao.FindAssignments(faceID)
		if err != nil {
			return fmt.Errorf("failed to get assigned types %w", err)
		}
		var toRemove []int64
		// remove all entries that are already assigned to the card
		for _, existingType := range existingTypes {
			if ok, pos := contains(toCreate, existingType.Name); ok {
				toCreate = remove(toCreate, pos)
			} else {
				toRemove = append(toRemove, existingType.ID.Int64)
			}
		}
		if len(toRemove) > 0 {
			if err := dao.DeleteAssignments(faceID, toRemove...); err != nil {
				return fmt.Errorf("failed to removed assigned types %w", err)
			}
		}
	}

	return assignTypes(dao, toCreate, faceID)
}

func assignTypes(dao TypeDao, toCreate []string, faceID int64) error {
	if len(toCreate) == 0 {
		return nil
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
			if entry, err = dao.Create(t); err != nil {
				return fmt.Errorf("failed to create type %s %w", t, err)
			}
		}

		if err := dao.AssignToFace(faceID, entry.ID.Int64); err != nil {
			return fmt.Errorf("failed to assign type %v to card %d %w", entry, faceID, err)
		}
	}

	return nil
}

func mergeFaceTranslations(dao *PostgresCardDao, tt []FaceTranslation, faceID int64, isNewCard bool) error {
	var toCreate []FaceTranslation
	toCreate = append(toCreate, tt...)
	if !isNewCard {
		existingTranslations, err := dao.FindTranslations(faceID)
		if err != nil {
			return fmt.Errorf("failed to get existing translations %w", err)
		}

		for _, existingTranslation := range existingTranslations {
			if ok, pos := containsFaceTranslation(tt, *existingTranslation); ok {
				toCreate = removeFaceTranslation(toCreate, *existingTranslation)

				translation := &tt[pos]

				diff := existingTranslation.Diff(translation)
				if diff.HasChanges() {
					log.Info().Msgf("Update translation for face %v and language %v with changes %s",
						faceID, existingTranslation.Lang, diff.String())
					if err := dao.UpdateTranslation(faceID, translation); err != nil {
						return err
					}
				}
			} else {
				if err := dao.DeleteTranslation(faceID, existingTranslation.Lang); err != nil {
					return err
				}
			}
		}
	}

	for _, t := range toCreate {
		t := t
		if err := dao.AddTranslation(faceID, &t); err != nil {
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

	for i, f := range arr {
		if f.couldBeSame(searchTerm) {
			return true, i
		}
	}

	return false, 0
}

func removeFaceTranslation(arr []FaceTranslation, toRemove FaceTranslation) []FaceTranslation {
	if ok, pos := containsFaceTranslation(arr, toRemove); ok {
		arr[pos] = arr[len(arr)-1]

		return arr[:len(arr)-1]
	}

	return arr
}

func containsFaceTranslation(arr []FaceTranslation, t FaceTranslation) (bool, int) {
	for i, e := range arr {
		if e.Lang == t.Lang {
			return true, i
		}
	}

	return false, 0
}
