package cardset

import (
	"errors"
	"fmt"

	"github.com/konstantinfoerster/card-importer-go/internal/api/card"
	"github.com/rs/zerolog/log"
)

type Service interface {
	Import(set *CardSet) error
	Count() (int, error)
}

type setService struct {
	dao *PostgresSetDao
}

func NewService(dao *PostgresSetDao) Service {
	return &setService{
		dao: dao,
	}
}

// Count Counts all card sets.
func (s *setService) Count() (int, error) {
	return s.dao.Count()
}

// Import Creates or updates the given card set.
func (s *setService) Import(set *CardSet) error {
	if set == nil {
		// Skip nil set
		return nil
	}
	if err := set.isValid(); err != nil {
		return fmt.Errorf("card set is invalid, %w", err)
	}
	// create block if required
	if set.Block.Block != "" {
		block, err := s.dao.FindBlockByName(set.Block.Block)
		if err != nil {
			if !errors.Is(err, card.ErrEntryNotFound) {
				return err
			}

			block, err = s.dao.CreateBlock(set.Block.Block)
			if err != nil {
				return err
			}
		}
		set.Block = *block
	}

	existingSet, err := s.dao.FindCardSetByCode(set.Code)
	if err != nil {
		if !errors.Is(err, card.ErrEntryNotFound) {
			return err
		}

		if e := log.Trace(); e.Enabled() {
			e.Msgf("Create set %s %s", set.Code, set.Name)
		}
		if err := s.dao.CreateCardSet(set); err != nil {
			return err
		}
	}

	if existingSet != nil {
		diff := existingSet.Diff(set)
		if diff.HasChanges() {
			log.Info().Msgf("Update set %s with changes %s", set.Code, diff.String())
			if err := s.dao.UpdateCardSet(set); err != nil {
				return err
			}
		}
	}

	return mergeTranslations(s.dao, set.Translations, set.Code, existingSet == nil)
}

func mergeTranslations(dao *PostgresSetDao, tt []Translation, setCode string, isNew bool) error {
	var toCreate []Translation
	toCreate = append(toCreate, tt...)

	if !isNew {
		existingTranslations, err := dao.FindTranslations(setCode)
		if err != nil {
			return fmt.Errorf("failed to get existing translations %w", err)
		}

		for _, existing := range existingTranslations {
			if ok, pos := containsTranslation(tt, *existing); ok {
				toCreate = removeTranslation(toCreate, *existing)

				newT := tt[pos]
				changed := false
				if existing.Name != newT.Name {
					log.Info().Msgf("Update translation.Name from '%v' to '%v'", existing.Name, newT.Name)
					changed = true
				}
				if changed {
					log.Info().Msgf("Update translation for set %s and language %v from %v to %v",
						setCode, existing.Lang, existing.Name, newT.Name)
					if err := dao.UpdateTranslation(setCode, &newT); err != nil {
						return err
					}
				}
			} else {
				if err := dao.DeleteTranslation(setCode, existing.Lang); err != nil {
					return err
				}
			}
		}
	}

	for _, t := range toCreate {
		t := t
		if err := dao.CreateTranslation(setCode, &t); err != nil {
			return err
		}

		continue
	}

	return nil
}

func removeTranslation(arr []Translation, toRemove Translation) []Translation {
	if ok, pos := containsTranslation(arr, toRemove); ok {
		arr[pos] = arr[len(arr)-1]

		return arr[:len(arr)-1]
	}

	return arr
}

func containsTranslation(tt []Translation, t Translation) (bool, int) {
	for i, e := range tt {
		if e.Lang == t.Lang {
			return true, i
		}
	}

	return false, 0
}
