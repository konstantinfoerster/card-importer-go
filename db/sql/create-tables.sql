CREATE TABLE public.lang
(
    lang CHAR(3) PRIMARY KEY NOT NULL
);

INSERT INTO public.lang
VALUES ('deu');
INSERT INTO public.lang
VALUES ('eng');

-- Sub Type --
CREATE TABLE public.sub_type
(
    id   SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    UNIQUE (name)
);

CREATE TABLE public.sub_type_translation
(
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    lang_lang   CHAR(3) REFERENCES public.lang (lang),
    sub_type_id INTEGER REFERENCES public.sub_type (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, sub_type_id)
);

-- Super Type --
CREATE TABLE public.super_type
(
    id   SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    UNIQUE (name)
);

CREATE TABLE public.super_type_translation
(
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    lang_lang     CHAR(3) REFERENCES public.lang (lang),
    super_type_id INTEGER REFERENCES public.super_type (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, super_type_id)
);

-- Card Type --
CREATE TABLE public.card_type
(
    id   SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    UNIQUE (name)
);

CREATE TABLE public.card_type_translation
(
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    lang_lang    CHAR(3) REFERENCES public.lang (lang),
    card_type_id INTEGER REFERENCES public.card_type (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_type_id)
);

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
    'SPELLBOOK'
    );

-- Layout --
CREATE TYPE public.layout AS ENUM (
    'NORMAL',
    'SPLIT',
    'FLIP',
    'DOUBLE_FACED',
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


-- Card Block --
CREATE TABLE public.card_block
(
    id    SERIAL PRIMARY KEY,
    block VARCHAR(255) NOT NULL UNIQUE
);

CREATE TABLE public.card_block_translation
(
    id            SERIAL PRIMARY KEY,
    block         VARCHAR(255) NOT NULL,
    lang_lang     CHAR(3) REFERENCES public.lang (lang),
    card_block_id INTEGER REFERENCES public.card_block (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_block_id)
);

-- Card Set --
CREATE TABLE public.card_set
(
    code          VARCHAR(255) PRIMARY KEY NOT NULL,
    name          VARCHAR(255)             NOT NULL,
    type          card_set_type            NOT NULL, -- Enum
    released      DATE,
    total_count   INTEGER                  NOT NULL,
    card_block_id INTEGER REFERENCES public.card_block (id),
    UNIQUE (code, card_block_id)
);

CREATE TABLE public.card_set_translation
(
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    lang_lang     CHAR(3) REFERENCES public.lang (lang),
    card_set_code VARCHAR(255) REFERENCES public.card_set (code) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_set_code)
);

--cardmanager=# \d+ card_set
--                                            Table "public.card_set"
--   Column    |          Type          | Collation | Nullable | Default | Storage  | Stats target | Description
---------------+------------------------+-----------+----------+---------+----------+--------------+-------------
-- code        | character varying(255) |           | not null |         | extended |              |
-- name        | character varying(255) |           |          |         | extended |              |
-- released    | date                   |           |          |         | plain    |              |
-- total_count | integer                |           |          |         | plain    |              |
-- type        | character varying(255) |           |          |         | extended |              |
-- block_id    | bigint                 |           | not null |         | plain    |              |
--Indexes:
--    "card_set_pkey" PRIMARY KEY, btree (code)
--Foreign-key constraints:
--    "fkpdnrmfr1eaxra2iaifksrdxr2" FOREIGN KEY (block_id) REFERENCES card_block(id)


--- First iteration until here ---

-- Card --
CREATE TABLE public.card
(
    id                  SERIAL PRIMARY KEY,
    artist              VARCHAR(255)   NOT NULL,
    border              border         NOT NULL, -- Enum
    converted_mana_cost NUMERIC(10, 2) NOT NULL,
    colors              VARCHAR(255),            -- List of Strings with ',' as separator
    name                VARCHAR(255)   NOT NULL,
    text                VARCHAR(800),
    flavor_text         VARCHAR(500),
    layout              layout         NOT NULL, -- Enum
    hand_modifier       INTEGER        NOT NULL, -- only Vanguard cards
    life_modifier       INTEGER        NOT NULL, -- only Vanguard cards
    loyalty             VARCHAR(10),             -- only planeswalker
    mana_cost           VARCHAR(255),
    multiverse_id       BIGINT,                  -- id from gatherer.wizards.com, id per lang
    power               VARCHAR(255),
    toughness           VARCHAR(255),
    rarity              rarity         NOT NULL, -- Enum
    number              VARCHAR(255)   NOT NULL,
    full_type           VARCHAR(255),
    --   related_card_ids VARCHAR(255), -- List of Strings. Where do we get the value from ??
    card_set_code       VARCHAR(255)   NOT NULL,
    unique (name, card_set_code, number)
);

CREATE TABLE public.card_translation
(
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    multiverse_id BIGINT,
    text          VARCHAR(800),
    flavor_text   VARCHAR(500),
    full_type     VARCHAR(255),
    lang_lang     CHAR(3) REFERENCES public.lang (lang),
    card_id       INTEGER REFERENCES public.card (id) ON DELETE CASCADE,
    UNIQUE (lang_lang, card_id)
);

CREATE TABLE public.card_super_type
(
    card_id INTEGER REFERENCES public.card (id),
    type_id INTEGER REFERENCES public.super_type (id),
    UNIQUE (card_id, type_id)
);

CREATE TABLE public.card_sub_type
(
    card_id INTEGER REFERENCES public.card (id),
    type_id INTEGER REFERENCES public.sub_type (id),
    UNIQUE (card_id, type_id)
);

CREATE TABLE public.card_card_type
(
    card_id INTEGER REFERENCES public.card (id),
    type_id INTEGER REFERENCES public.card_type (id),
    UNIQUE (card_id, type_id)
);
-- CREATE TABLE public.external_lang (
--     id SERIAL PRIMARY KEY,
--     external_lang VARCHAR(255) NOT NULL,
--     lang_lang CHAR(3) REFERENCES public.lang(lang)
-- )