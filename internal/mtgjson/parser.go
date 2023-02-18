package mtgjson

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
)

var cardSetCodeRegex = regexp.MustCompile("^[A-Z0-9]{3,10}$")

type result struct {
	Result interface{}
	Err    error
}

func parse(ctx context.Context, r io.Reader) <-chan *result {
	c := make(chan *result)

	go func() {
		defer close(c)

		dec := json.NewDecoder(r)

		if err := expectNext(json.Delim('{'), dec); err != nil {
			c <- &result{Result: nil, Err: err}

			return
		}

		isSetCodeFn := verifySetCodeStartFn()

		for dec.More() {
			t, err := dec.Token()
			if err != nil {
				c <- &result{Result: nil, Err: err}

				return
			}

			if t != "data" {
				if err := skip(dec); err != nil {
					c <- &result{Result: nil, Err: err}

					return
				}

				continue
			}

			if err := expectNext(json.Delim('{'), dec); err != nil {
				c <- &result{Result: nil, Err: err}

				return
			}

			for dec.More() {
				t, err := dec.Token() // 10E
				if err != nil {
					c <- &result{Result: nil, Err: err}

					return
				}
				if !isSetCodeFn(t) {
					if err := skip(dec); err != nil {
						c <- &result{Result: nil, Err: err}
					}

					continue
				}

				if err := parseSet(ctx, dec, c); err != nil {
					if ctx.Err() != nil {
						return
					}
					c <- &result{Result: nil, Err: err}

					return
				}
			}
		}
	}()

	return c
}

func parseSet(ctx context.Context, dec *json.Decoder, c chan<- *result) error {
	if err := expectNext(json.Delim('{'), dec); err != nil {
		return err
	}

	var name string
	var code string
	var block string
	var setType string
	var totalCount float64
	var released string
	var translations []translation

	for dec.More() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		t, err := dec.Token()
		et := &errToken{token: t, err: err}
		switch t {
		case "block":
			if et.next(dec) != nil {
				var ok bool
				block, ok = et.token.(string)
				if !ok {
					return fmt.Errorf("field block is not a string but %T", block)
				}
			}
		case "name":
			if et.next(dec) != nil {
				var ok bool
				name, ok = et.token.(string)
				if !ok {
					return fmt.Errorf("field name is not a string but %T", name)
				}
			}
		case "code":
			if et.next(dec) != nil {
				var ok bool
				code, ok = et.token.(string)
				if !ok {
					return fmt.Errorf("field code is not a string but %T", code)
				}
			}
		case "type":
			if et.next(dec) != nil {
				var ok bool
				setType, ok = et.token.(string)
				if !ok {
					return fmt.Errorf("field setType is not a string but %T", setType)
				}
			}
		case "totalSetSize":
			if et.next(dec) != nil {
				var ok bool
				totalCount, ok = et.token.(float64)
				if !ok {
					return fmt.Errorf("field totalCount is not a float but %T", totalCount)
				}
			}
		case "releaseDate":
			if et.next(dec) != nil {
				var ok bool
				released, ok = et.token.(string)
				if !ok {
					return fmt.Errorf("field released is not a string but %T", released)
				}
			}
		case "translations":
			translations, err = parseTranslations(dec, et)
			if err != nil {
				return err
			}
		case "cards":
			if err := parseCard(ctx, c, dec); err != nil {
				return err
			}
		default:
			if err := skip(dec); err != nil {
				return err
			}
		}

		if et.err != nil {
			return et.err
		}
	}
	if code == "" {
		// most likely no set found
		return nil
	}

	c <- &result{Result: &mtgjsonCardSet{
		Code:         code,
		Name:         name,
		Block:        block,
		Type:         setType,
		TotalCount:   totalCount,
		Released:     released,
		Translations: translations,
	}, Err: nil}

	if err := expectNext(json.Delim('}'), dec); err != nil {
		return err
	}

	return nil
}

func parseCard(ctx context.Context, c chan<- *result, dec *json.Decoder) error {
	if err := expectNext(json.Delim('['), dec); err != nil {
		return err
	}

	for dec.More() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var card *mtgjsonCard
		err := dec.Decode(&card)
		if err != nil {
			return err
		}
		c <- &result{Result: card, Err: nil}
	}

	if err := expectNext(json.Delim(']'), dec); err != nil {
		return err
	}

	return nil
}

func parseTranslations(dec *json.Decoder, et *errToken) ([]translation, error) {
	if err := expectNext(json.Delim('{'), dec); err != nil {
		return nil, err
	}
	var translations []translation
	for dec.More() {
		var lang string
		var translated string
		if et.next(dec) != nil {
			var ok bool
			lang, ok = et.token.(string)
			if !ok {
				return nil, fmt.Errorf("field lang is not a string but %T", lang)
			}
		}
		if et.next(dec) != nil {
			var ok bool
			translated, ok = et.token.(string)
			if !ok {
				return nil, fmt.Errorf("field translated is not a string but %T", translated)
			}
		}
		translations = append(translations, translation{Language: lang, Name: translated})
	}

	if err := expectNext(json.Delim('}'), dec); err != nil {
		return nil, err
	}

	return translations, nil
}

type errToken struct {
	token json.Token
	err   error
}

func (et *errToken) next(dec *json.Decoder) json.Token {
	if et.err != nil {
		return nil
	}
	et.token, et.err = dec.Token()

	return et.token
}

func expectNext(expected json.Delim, dec *json.Decoder) error {
	t, err := dec.Token()
	if err != nil {
		return fmt.Errorf("failed to get next token %w", err)
	}

	if t != expected {
		return fmt.Errorf("expected token to be %v but found %v", expected, t)
	}

	return nil
}

func skip(dec *json.Decoder) error {
	n := 0
	for {
		t, err := dec.Token()
		if err != nil {
			return err
		}

		switch t {
		case json.Delim('['), json.Delim('{'):
			n++
		case json.Delim(']'), json.Delim('}'):
			n--
		}

		if n == 0 {
			return nil
		}
	}
}

func verifySetCodeStartFn() func(t json.Token) bool {
	return func(t json.Token) bool {
		key, ok := t.(string)
		if ok && cardSetCodeRegex.MatchString(key) {
			return true
		}

		return false
	}
}
