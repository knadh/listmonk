DROP TYPE IF EXISTS list_type CASCADE; CREATE TYPE list_type AS ENUM ('public', 'private', 'temporary');
DROP TYPE IF EXISTS list_optin CASCADE; CREATE TYPE list_optin AS ENUM ('single', 'double');
DROP TYPE IF EXISTS subscriber_status CASCADE; CREATE TYPE subscriber_status AS ENUM ('enabled', 'disabled', 'blacklisted');
DROP TYPE IF EXISTS subscription_status CASCADE; CREATE TYPE subscription_status AS ENUM ('unconfirmed', 'confirmed', 'unsubscribed');
DROP TYPE IF EXISTS campaign_status CASCADE; CREATE TYPE campaign_status AS ENUM ('draft', 'running', 'scheduled', 'paused', 'cancelled', 'finished');
DROP TYPE IF EXISTS content_type CASCADE; CREATE TYPE content_type AS ENUM ('richtext', 'html', 'plain');

-- subscribers
DROP TABLE IF EXISTS subscribers CASCADE;
CREATE TABLE subscribers (
    id              SERIAL PRIMARY KEY,
    uuid uuid       NOT NULL UNIQUE,
    email           TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    attribs         JSONB NOT NULL DEFAULT '{}',
    status          subscriber_status NOT NULL DEFAULT 'enabled',
    campaigns       INTEGER[],

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_subs_email; CREATE UNIQUE INDEX idx_subs_email ON subscribers(LOWER(email));
DROP INDEX IF EXISTS idx_subs_status; CREATE INDEX idx_subs_status ON subscribers(status);

-- lists
DROP TABLE IF EXISTS lists CASCADE;
CREATE TABLE lists (
    id              SERIAL PRIMARY KEY,
    uuid            uuid NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    type            list_type NOT NULL,
    optin           list_optin NOT NULL DEFAULT 'single',
    tags            VARCHAR(100)[],

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

DROP TABLE IF EXISTS subscriber_lists CASCADE;
CREATE TABLE subscriber_lists (
    subscriber_id      INTEGER REFERENCES subscribers(id) ON DELETE CASCADE ON UPDATE CASCADE,
    list_id            INTEGER NULL REFERENCES lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    status             subscription_status NOT NULL DEFAULT 'unconfirmed',

    created_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    PRIMARY KEY(subscriber_id, list_id)
);
DROP INDEX IF EXISTS idx_sub_lists_sub_id; CREATE INDEX idx_sub_lists_sub_id ON subscriber_lists(subscriber_id);
DROP INDEX IF EXISTS idx_sub_lists_list_id; CREATE INDEX idx_sub_lists_list_id ON subscriber_lists(list_id);
DROP INDEX IF EXISTS idx_sub_lists_status; CREATE INDEX idx_sub_lists_status ON subscriber_lists(status);

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

    -- Progress and stats.
    to_send            INT NOT NULL DEFAULT 0,
    sent               INT NOT NULL DEFAULT 0,
    max_subscriber_id  INT NOT NULL DEFAULT 0,
    last_subscriber_id INT NOT NULL DEFAULT 0,

    started_at       TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

DROP TABLE IF EXISTS campaign_lists CASCADE;
CREATE TABLE campaign_lists (
    campaign_id  INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Lists may be deleted, so list_id is nullable
    -- and a copy of the original list name is maintained here.
    list_id      INTEGER NULL REFERENCES lists(id) ON DELETE SET NULL ON UPDATE CASCADE,
    list_name    TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX ON campaign_lists (campaign_id, list_id);
DROP INDEX IF EXISTS idx_camp_lists_camp_id; CREATE INDEX idx_camp_lists_camp_id ON campaign_lists(campaign_id);
DROP INDEX IF EXISTS idx_camp_lists_list_id; CREATE INDEX idx_camp_lists_list_id ON campaign_lists(list_id);

DROP TABLE IF EXISTS campaign_views CASCADE;
CREATE TABLE campaign_views (
    campaign_id      INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,

    -- Subscribers may be deleted, but the view counts should remain.
    subscriber_id    INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL ON UPDATE CASCADE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
DROP INDEX IF EXISTS idx_views_camp_id; CREATE INDEX idx_views_camp_id ON campaign_views(campaign_id);
DROP INDEX IF EXISTS idx_views_subscriber_id; CREATE INDEX idx_views_subscriber_id ON campaign_views(subscriber_id);

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
DROP INDEX IF EXISTS idx_clicks_camp_id; CREATE INDEX idx_clicks_camp_id ON link_clicks(campaign_id);
DROP INDEX IF EXISTS idx_clicks_link_id; CREATE INDEX idx_clicks_link_id ON link_clicks(link_id);
DROP INDEX IF EXISTS idx_clicks_sub_id; CREATE INDEX idx_clicks_sub_id ON link_clicks(subscriber_id);
