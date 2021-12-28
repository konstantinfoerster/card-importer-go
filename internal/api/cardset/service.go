package cardset

import (
	"fmt"
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

func (s *setService) Count() (int, error) {
	return s.dao.Count()
}

// Import creates or updates the given card set
func (s *setService) Import(set *CardSet) error {
	if set == nil {
		// Skip nil set
		return nil
	}
	if set.Code == "" {
		return fmt.Errorf("field 'code' must not be empty")
	}
	if set.Type == "" {
		return fmt.Errorf("field 'type' must not be empty in set %s", set.Code)
	}

	// create block if required
	if set.Block.Block != "" {
		block, err := s.dao.FindBlockByName(set.Block.Block)
		if err != nil {
			return err
		}
		if block == nil {
			block, err = s.dao.CreateBlock(set.Block.Block)
			if err != nil {
				return err
			}
		}
		set.Block = *block
	}

	existingSet, err := s.dao.FindCardSetByCode(set.Code)
	if err != nil {
		return err
	}
	if existingSet == nil {
		if log.Trace().Enabled() {
			log.Trace().Msgf("Create set %s %s", set.Code, set.Name)
		}
		if err := s.dao.CreateCardSet(set); err != nil {
			return err
		}
	} else {
		changed := false

		if existingSet.Block.Id.Valid && existingSet.Block.notEquals(set.Block) {
			log.Info().Msgf("Block for set %s changed from %#v to %#v", set.Code, existingSet.Block, set.Block)
			changed = true
		}

		if set.Name != existingSet.Name {
			changed = true
		} else if set.Type != existingSet.Type {
			changed = true
		} else if !set.Released.Equal(existingSet.Released) {
			changed = true
		} else if set.TotalCount != existingSet.TotalCount {
			changed = true
		}

		if changed {
			log.Info().Msgf("Update set %s %s", set.Code, set.Name)
			if err := s.dao.UpdateCardSet(set); err != nil {
				return err
			}
		}
	}

	err = s.mergeTranslations(set.Translations, set.Code, existingSet == nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *setService) mergeTranslations(tt []Translation, setCode string, isNew bool) error {
	var toCreate []Translation
	toCreate = append(toCreate, tt...)
	if !isNew {
		existingTranslations, err := s.dao.FindTranslations(setCode)
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
				}
				if changed {
					log.Info().Msgf("Update translation for set %s and language %v from %v to %v", setCode, existing.Lang, existing.Name, newT.Name)
					if err := s.dao.UpdateTranslation(setCode, &newT); err != nil {
						return err
					}
				}
			} else {
				if err := s.dao.DeleteTranslation(setCode, existing.Lang); err != nil {
					return err
				}
			}
		}
	}

	for _, t := range toCreate {
		if err := s.dao.CreateTranslation(setCode, &t); err != nil {
			return err
		}
		continue
	}

	return nil
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
