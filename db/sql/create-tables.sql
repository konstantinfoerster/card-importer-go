-- generated always as identity -> prevents manual inserts of ids
CREATE TABLE public.lang
(
    lang CHAR(3) PRIMARY KEY NOT NULL CHECK (lang = lower(lang) AND lang <> '') -- <> == not equal
);

INSERT INTO public.lang
VALUES ('deu');
INSERT INTO public.lang
VALUES ('eng');

-- Border --
CREATE TYPE public.border AS ENUM (
    'WHITE', 'BLACK', 'SILVER', 'GOLD', 'BORDERLESS'
    );

-- Card set type --
CREATE TYPE public.card_set_type AS ENUM (
    'CORE',
    'EXPANSION',
    'REPRINT',
    'BOX',
    'UN',
    'FROM_THE_VAULT',
    'PREMIUM_DECK',
    'DUEL_DECK',
    'STARTER',
    'COMMANDER',
    'PLANECHASE',
    'ARCHENEMY',
    'PROMO',
    'VANGUARD',
    'MASTERS',
    'MEMORABILIA',
    'DRAFT_INNOVATION',
    'FUNNY',
    'MASTERPIECE',
    'TOKEN',
    'TREASURE_CHEST',
    'SPELLBOOK',
    'ARSENAL',
    'ALCHEMY'
    );

-- Layout --
CREATE TYPE public.layout AS ENUM (
    'NORMAL',
    'SPLIT',
    'FLIP',
    'TOKEN',
    'PLANE',
    'SCHEMA',
    'PHENOMENON',
    'LEVELER',
    'VANGUARD',
    'MELD',
    'AFTERMATH',
    'SAGA',
    'TRANSFORM',
    'ADVENTURE',
    'MODAL_DFC',
    'SCHEME',
    'PLANAR',
    'HOST',
    'AUGMENT',
    'CLASS',
    'REVERSIBLE_CARD'
    );

-- Rarity --
CREATE TYPE public.rarity AS ENUM (
    'COMMON',
    'UNCOMMON',
    'RARE',
    'MYTHIC',
    'SPECIAL',
    'BASIC_LAND',
    'BONUS'
    );

-- Sub Type --
CREATE TABLE public.sub_type
(
    id   INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    UNIQUE (name)
);

CREATE TABLE public.sub_type_translation
(
    id          INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    lang_lang   CHAR(3) REFERENCES public.lang (lang),
    sub_type_id INTEGER REFERENCES public.sub_type (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, sub_type_id)
);

-- Super Type --
CREATE TABLE public.super_type
(
    id   INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    UNIQUE (name)
);

CREATE TABLE public.super_type_translation
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    lang_lang     CHAR(3) REFERENCES public.lang (lang),
    super_type_id INTEGER REFERENCES public.super_type (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, super_type_id)
);

-- Card Type --
CREATE TABLE public.card_type
(
    id   INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    UNIQUE (name)
);

CREATE TABLE public.card_type_translation
(
    id           INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name         VARCHAR(100) NOT NULL CHECK ( name <> '' ),
    lang_lang    CHAR(3) REFERENCES public.lang (lang),
    card_type_id INTEGER REFERENCES public.card_type (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_type_id)
);

-- Card Block --
CREATE TABLE public.card_block
(
    id    INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    block VARCHAR(255) NOT NULL UNIQUE CHECK ( block <> '' )
);

CREATE TABLE public.card_block_translation
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    block         VARCHAR(255) NOT NULL CHECK ( block <> '' ),
    lang_lang     CHAR(3) REFERENCES public.lang (lang),
    card_block_id INTEGER REFERENCES public.card_block (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_block_id)
);

-- Card Set --
CREATE TABLE public.card_set
(
    code          VARCHAR(10) PRIMARY KEY NOT NULL CHECK ( code <> '' AND code = upper(code)),
    name          VARCHAR(255)            NOT NULL CHECK ( name <> '' ),
    type          card_set_type           NOT NULL, -- Enum
    released      DATE,
    total_count   INTEGER                 NOT NULL CHECK ( total_count >= 0 ),
    card_block_id INTEGER REFERENCES public.card_block (id),
    UNIQUE (code, card_block_id)
);

CREATE TABLE public.card_set_translation
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          VARCHAR(255) NOT NULL CHECK ( name <> '' ),
    lang_lang     CHAR(3) REFERENCES public.lang (lang),
    card_set_code VARCHAR(10) REFERENCES public.card_set (code) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_set_code)
);

-- Card --
CREATE TABLE public.card
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          VARCHAR(255) NOT NULL CHECK ( name <> '' ),
    number        VARCHAR(255) NOT NULL CHECK ( number <> '' ),
    rarity        rarity       NOT NULL, -- Enum
    border        border       NOT NULL, -- Enum
    layout        layout       NOT NULL, -- Enum
    card_set_code VARCHAR(10)  NOT NULL CHECK ( card_set_code <> '' AND card_set_code = upper(card_set_code) ),
    unique (card_set_code, number)
);


CREATE TABLE public.card_face
(
    id                  INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                VARCHAR(255)   NOT NULL CHECK ( name <> '' ),
    text                VARCHAR(800),
    flavor_text         VARCHAR(500),
    type_line           VARCHAR(255),
    converted_mana_cost NUMERIC(10, 2) NOT NULL CHECK ( converted_mana_cost >= 0 ),
    colors              VARCHAR(100),                                                -- List of Strings with ',' as separator
    artist              VARCHAR(100),
    hand_modifier       VARCHAR(10),                                                 -- only Vanguard cards
    life_modifier       VARCHAR(10),                                                 -- only Vanguard cards
    loyalty             VARCHAR(10),                                                 -- only planeswalker
    mana_cost           VARCHAR(255),
    multiverse_id       INTEGER CHECK (multiverse_id >= 0 OR multiverse_id IS NULL), -- id from gatherer.wizards.com, id per lang
    power               VARCHAR(255),
    toughness           VARCHAR(255),
    card_id             INTEGER REFERENCES public.card (id) ON DELETE CASCADE
);

CREATE TABLE public.card_translation
(
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          VARCHAR(255) NOT NULL CHECK ( name <> '' ),
    multiverse_id INTEGER CHECK (multiverse_id >= 0 OR multiverse_id IS NULL),
    text          VARCHAR(800),
    flavor_text   VARCHAR(500),
    type_line     VARCHAR(255),
    lang_lang     CHAR(3) REFERENCES public.lang (lang),
    face_id       INTEGER REFERENCES public.card_face (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, face_id)
);

CREATE TABLE public.face_super_type
(
    face_id INTEGER REFERENCES public.card_face (id),
    type_id INTEGER REFERENCES public.super_type (id),
    UNIQUE (face_id, type_id)
);

CREATE TABLE public.face_sub_type
(
    face_id INTEGER REFERENCES public.card_face (id),
    type_id INTEGER REFERENCES public.sub_type (id),
    UNIQUE (face_id, type_id)
);

CREATE TABLE public.face_card_type
(
    face_id INTEGER REFERENCES public.card_face (id),
    type_id INTEGER REFERENCES public.card_type (id),
    UNIQUE (face_id, type_id)
);
