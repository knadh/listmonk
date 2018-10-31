DROP TYPE IF EXISTS user_type CASCADE; CREATE TYPE user_type AS ENUM ('superadmin', 'user');
DROP TYPE IF EXISTS user_status CASCADE; CREATE TYPE user_status AS ENUM ('enabled', 'disabled');
DROP TYPE IF EXISTS list_type CASCADE; CREATE TYPE list_type AS ENUM ('public', 'private', 'temporary');
DROP TYPE IF EXISTS subscriber_status CASCADE; CREATE TYPE subscriber_status AS ENUM ('enabled', 'disabled', 'blacklisted');
DROP TYPE IF EXISTS subscription_status CASCADE; CREATE TYPE subscription_status AS ENUM ('unconfirmed', 'confirmed', 'unsubscribed');
DROP TYPE IF EXISTS campaign_status CASCADE; CREATE TYPE campaign_status AS ENUM ('draft', 'running', 'scheduled', 'paused', 'cancelled', 'finished');
DROP TYPE IF EXISTS content_type CASCADE; CREATE TYPE content_type AS ENUM ('richtext', 'html', 'plain');

-- users
DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
    id              SERIAL PRIMARY KEY,
    email           TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    password        TEXT NOT NULL,
    type            user_type NOT NULL,
    status          user_status NOT NULL,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_users_email; CREATE INDEX idx_users_email ON users(email);

-- subscribers
DROP TABLE IF EXISTS subscribers CASCADE;
CREATE TABLE subscribers (
    id              SERIAL PRIMARY KEY,
    uuid uuid       NOT NULL UNIQUE,
    email           TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    attribs         JSONB,
    status          subscriber_status NOT NULL,
    campaigns       INTEGER[],

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_subscribers_email; CREATE INDEX idx_subscribers_email ON subscribers(email);

-- lists
DROP TABLE IF EXISTS lists CASCADE;
CREATE TABLE lists (
    id              SERIAL PRIMARY KEY,
    uuid            uuid NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    type            list_type NOT NULL,
    tags            VARCHAR(100)[],

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_lists_uuid; CREATE INDEX idx_lists_uuid ON lists(uuid);

DROP TABLE IF EXISTS subscriber_lists CASCADE;
CREATE TABLE subscriber_lists (
    subscriber_id      INTEGER REFERENCES subscribers(id) ON DELETE CASCADE ON UPDATE CASCADE,
    list_id            INTEGER NULL REFERENCES lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    status             subscription_status NOT NULL DEFAULT 'unconfirmed',

    created_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    PRIMARY KEY(subscriber_id, list_id)
);


-- templates
DROP TABLE IF EXISTS templates CASCADE;
CREATE TABLE templates (
    id              SERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    body            TEXT NOT NULL,
    is_default      BOOLEAN NOT NULL DEFAULT false,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX ON templates (is_default) WHERE is_default = true;


-- campaigns
DROP TABLE IF EXISTS campaigns CASCADE;
CREATE TABLE campaigns (
    id               SERIAL PRIMARY KEY,
    uuid uuid        NOT NULL UNIQUE,
    name             TEXT NOT NULL,
    subject          TEXT NOT NULL,
    from_email       TEXT NOT NULL,
    body             TEXT NOT NULL,
    content_type     content_type NOT NULL DEFAULT 'richtext',
    send_at          TIMESTAMP WITH TIME ZONE,
    status           campaign_status NOT NULL DEFAULT 'draft',
    tags             VARCHAR(100)[],

    -- The ID of the messenger backend used to send this campaign. 
    messenger        TEXT NOT NULL,

    template_id      INTEGER REFERENCES templates(id) ON DELETE SET DEFAULT DEFAULT 1,

    -- The lists to which a campaign is sent can change at any point.
    -- They can be deleted, or they could be ephmeral. Hence, storing
    -- references to the lists table is not possible. The list names and
    -- their erstwhile IDs are stored in a JSON blob for posterity.
    lists            JSONB,

    -- Progress and stats.
    to_send            INT NOT NULL DEFAULT 0,
    sent               INT NOT NULL DEFAULT 0,
    max_subscriber_id  INT NOT NULL DEFAULT 0,
    last_subscriber_id INT NOT NULL DEFAULT 0,

    started_at       TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_campaigns_uuid; CREATE INDEX idx_campaigns_uuid ON campaigns(uuid);

DROP TABLE IF EXISTS campaign_lists CASCADE;
CREATE TABLE campaign_lists (
    campaign_id  INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Lists may be deleted, so list_id is nullable
    -- and a copy of the original list name is maintained here.
    list_id      INTEGER NULL REFERENCES lists(id) ON DELETE SET NULL ON UPDATE CASCADE,
    list_name    TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX ON campaign_lists (campaign_id, list_id);

DROP TABLE IF EXISTS campaign_views CASCADE;
CREATE TABLE campaign_views (
    campaign_id      INTEGER REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Subscribers may be deleted, but the link counts should remain.
    subscriber_id    INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL ON UPDATE CASCADE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- media
DROP TABLE IF EXISTS media CASCADE;
CREATE TABLE media (
    id               SERIAL PRIMARY KEY,
    uuid uuid        NOT NULL UNIQUE,
    filename         TEXT NOT NULL,
    thumb            TEXT NOT NULL,
    width            INT NOT NULL DEFAULT 0,
    height           INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- links
DROP TABLE IF EXISTS links CASCADE;
CREATE TABLE links (
    id               SERIAL PRIMARY KEY,
    uuid uuid        NOT NULL UNIQUE,
    url              TEXT NOT NULL UNIQUE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

DROP TABLE IF EXISTS link_clicks CASCADE;
CREATE TABLE link_clicks (
    campaign_id      INTEGER REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,
    link_id          INTEGER REFERENCES links(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Subscribers may be deleted, but the link counts should remain.
    subscriber_id    INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL ON UPDATE CASCADE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
