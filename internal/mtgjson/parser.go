package mtgjson

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
)

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

		isSetCodeFn, err := verifySetCodeStartFn()
		if err != nil {
			c <- &result{Result: nil, Err: err}
			return
		}

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

				if err := parseSet(dec, c, ctx); err != nil {
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

func parseSet(dec *json.Decoder, c chan<- *result, ctx context.Context) error {
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
				block = et.token.(string)
			}
		case "name":
			if et.next(dec) != nil {
				name = et.token.(string)
			}
		case "code":
			if et.next(dec) != nil {
				code = et.token.(string)
			}
		case "type":
			if et.next(dec) != nil {
				setType = et.token.(string)
			}
		case "totalSetSize":
			if et.next(dec) != nil {
				totalCount = et.token.(float64)
			}
		case "releaseDate":
			if et.next(dec) != nil {
				released = et.token.(string)
			}
		case "translations":
			if err := expectNext(json.Delim('{'), dec); err != nil {
				return err
			}
			for dec.More() {
				var lang string
				var translated string
				if et.next(dec) != nil {
					lang = et.token.(string)
				}
				if et.next(dec) != nil {
					translated = et.token.(string)
				}
				translations = append(translations, translation{Language: lang, Name: translated})
			}
			if err := expectNext(json.Delim('}'), dec); err != nil {
				return err
			}
		case "cards":
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

func verifySetCodeStartFn() (func(t json.Token) bool, error) {
	regex, err := regexp.Compile("^[A-Z0-9]{3,10}$")
	if err != nil {
		return nil, err
	}

	return func(t json.Token) bool {
		key, ok := t.(string)
		if ok && regex.MatchString(key) {
			return true
		}
		return false
	}, err
}
