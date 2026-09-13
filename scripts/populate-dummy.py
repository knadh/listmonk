#!/usr/bin/env python3
"""Populates a listmonk database with random dummy data."""

import argparse
import json
import os
import random
import uuid
from datetime import datetime, timedelta, timezone
from urllib.parse import urlparse

import pg8000.dbapi

NUM_LISTS = 20
NUM_SUBSCRIBERS = 10000
NUM_CAMPAIGNS = 30
NUM_LINKS = 10
NUM_USERS = 10
NUM_ROLES = 3
NUM_CLICKS = 10000
NUM_VIEWS = 10000

# Number of campaigns and links that clicks/views are spread across.
SAMPLE_CAMPAIGNS = 10
SAMPLE_LINKS = 10

MAX_SUBSCRIPTIONS = 5
MAX_CAMPAIGN_LISTS = 3
MAX_TAGS = 3
MAX_ROLE_LISTS = 5
ACTIVITY_DAYS = 90

ADMIN_USER = "admin"
ADMIN_PASSWORD = "listmonk"

# Local mailhog (or equivalent) that outgoing e-mails are pointed at.
SMTP = {"name": "mailhog", "enabled": True, "host": "localhost", "port": 1025,
        "auth_protocol": "none", "username": "", "password": "", "hello_hostname": "",
        "max_conns": 10, "idle_timeout": "15s", "wait_timeout": "5s", "max_msg_retries": 2,
        "msg_retry_delay": "10ms", "tls_type": "none", "tls_skip_verify": False,
        "email_headers": [], "from_addresses": []}

# Per-list permissions that list roles are made of.
LIST_PERMS = ["list:get", "list:manage"]

VERBS = ["run", "jump", "build", "wander", "gather", "carve", "float", "sketch", "brew",
         "forge", "drift", "climb", "polish", "whistle", "scatter", "mend", "trace",
         "kindle", "ramble", "harvest"]

NOUNS = ["mango", "harbour", "lantern", "meadow", "compass", "river", "thunder", "orchard",
         "pebble", "canyon", "willow", "beacon", "anchor", "sparrow", "cobble", "marble",
         "prairie", "juniper", "cinder", "quarry"]

WORDS = VERBS + NOUNS

CAMPAIGN_STATUSES = ["draft", "finished", "cancelled", "scheduled"]

# (status, weight). Blocklisted is ~7% of subscribers.
SUBSCRIBER_STATUSES = [("enabled", 83), ("disabled", 10), ("blocklisted", 7)]


def sentence_name(n):
    words = random.sample(WORDS, n)
    return " ".join([words[0].capitalize()] + words[1:])


def camel_name():
    return "".join(w.capitalize() for w in random.sample(WORDS, 2))


def rand_tags():
    return random.sample(WORDS, random.randint(0, MAX_TAGS))


def sub_state():
    statuses, weights = zip(*SUBSCRIBER_STATUSES)
    return random.choices(statuses, weights=weights)[0]


def sub_status(optin):
    if optin == "double":
        return random.choice(["unconfirmed", "confirmed", "unsubscribed"])
    return random.choice(["unconfirmed", "unconfirmed", "unsubscribed"])


def all_perms():
    """Read the global permissions from the repo's permissions.json."""
    f = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "permissions.json")
    with open(f) as fp:
        return [p for group in json.load(fp) for p in group["permissions"]]


def rand_time(now):
    return now - timedelta(seconds=random.randint(0, ACTIVITY_DAYS * 86400))


def connect(dsn):
    u = urlparse(dsn)
    return pg8000.dbapi.connect(user=u.username or "listmonk", password=u.password,
                                host=u.hostname or "localhost", port=u.port or 5432,
                                database=u.path.lstrip("/") or "listmonk")


def insert(cur, sql, rows, fetch=False, chunk=1000):
    """Bulk INSERT rows in chunks. sql should end with a `{}` placeholder for VALUES."""
    out = []
    for i in range(0, len(rows), chunk):
        batch = rows[i:i + chunk]
        tpl = "(" + ",".join(["%s"] * len(batch[0])) + ")"
        cur.execute(sql.format(",".join([tpl] * len(batch))), [v for r in batch for v in r])
        if fetch:
            out += cur.fetchall()
    return out


def main():
    p = argparse.ArgumentParser(description="Populate a listmonk DB with dummy data.")
    p.add_argument("--db", default="postgres://listmonk:listmonk@localhost:5432/listmonk?sslmode=disable", help="Postgres DSN")
    args = p.parse_args()

    now = datetime.now(timezone.utc)
    db = connect(args.db)
    cur = db.cursor()

    cur.execute("UPDATE settings SET value=%s WHERE key='smtp'",
                [json.dumps([dict(SMTP, uuid=str(uuid.uuid4()))])])
    print(f"smtp -> {SMTP['host']}:{SMTP['port']}")

    cur.execute("SELECT id FROM templates WHERE type='campaign' ORDER BY id LIMIT 1")
    row = cur.fetchone()
    template_id = row[0] if row else None

    lists = [(str(uuid.uuid4()), sentence_name(3), random.choice(["public", "private"]),
              random.choice(["single", "double"]), rand_tags()) for _ in range(NUM_LISTS)]
    list_ids = insert(cur, """
        INSERT INTO lists (uuid, name, type, optin, tags) VALUES {} RETURNING id, optin, name
    """, lists, fetch=True)
    print(f"{len(list_ids)} lists")

    subs = [(str(uuid.uuid4()), f"{camel_name().lower()}{i}@example.com", camel_name(),
             sub_state()) for i in range(NUM_SUBSCRIBERS)]
    sub_ids = [r[0] for r in insert(cur, """
        INSERT INTO subscribers (uuid, email, name, status) VALUES {} RETURNING id
    """, subs, fetch=True)]
    print(f"{len(sub_ids)} subscribers")

    subscriptions = []
    for sid in sub_ids:
        for l in random.sample(list_ids, random.randint(1, MAX_SUBSCRIPTIONS)):
            subscriptions.append((sid, l[0], sub_status(l[1])))
    insert(cur, """
        INSERT INTO subscriber_lists (subscriber_id, list_id, status) VALUES {}
    """, subscriptions)
    print(f"{len(subscriptions)} subscriptions")

    perms = all_perms()

    # ID 1 is the primordial super admin role that has all permissions.
    cur.execute("""
        INSERT INTO roles (id, name, type, permissions) VALUES (1, 'Super Admin', 'user', %s)
    """, [perms])
    cur.execute("SELECT SETVAL('roles_id_seq', 1)")

    # Random user roles, each with a random subset of the global permissions.
    user_role_ids = [r[0] for r in insert(cur, """
        INSERT INTO roles (name, type, permissions) VALUES {} RETURNING id
    """, [(f"{sentence_name(2)} {i}", "user", random.sample(perms, random.randint(2, len(perms))))
          for i in range(NUM_ROLES)], fetch=True)]

    # Random list roles. A list role is a parent row with one child row per list
    # carrying that list's permissions.
    list_role_ids = [r[0] for r in insert(cur, """
        INSERT INTO roles (name, type) VALUES {} RETURNING id
    """, [(f"{sentence_name(2)} {i}", "list") for i in range(NUM_ROLES)], fetch=True)]

    role_lists = []
    for rid in list_role_ids:
        for l in random.sample(list_ids, random.randint(1, MAX_ROLE_LISTS)):
            role_lists.append((rid, l[0], "list",
                               random.sample(LIST_PERMS, random.randint(1, len(LIST_PERMS)))))
    insert(cur, """
        INSERT INTO roles (parent_id, list_id, type, permissions) VALUES {}
    """, role_lists)
    print(f"{len(user_role_ids)} user roles, {len(list_role_ids)} list roles")

    # `{}` is the row's ID expression, as the super admin user needs a fixed ID.
    user_sql = """
        INSERT INTO users (id, username, password_login, password, email, name, type,
            user_role_id, list_role_id, status)
        VALUES ({}, %s, TRUE, CRYPT(%s, GEN_SALT('bf')), %s, %s, 'user', %s, %s, 'enabled')
    """

    # ID 1 is the super admin user.
    cur.execute(user_sql.format(1), [ADMIN_USER, ADMIN_PASSWORD, f"{ADMIN_USER}@example.com",
                                     ADMIN_USER, 1, None])
    cur.execute("SELECT SETVAL('users_id_seq', 1)")

    for i in range(NUM_USERS):
        u = camel_name().lower() + str(i)
        cur.execute(user_sql.format("DEFAULT"),
                    [u, ADMIN_PASSWORD, f"{u}@example.com", camel_name(),
                     random.choice(user_role_ids), random.choice(list_role_ids)])
    print(f"{NUM_USERS + 1} users ('{ADMIN_USER}' / '{ADMIN_PASSWORD}')")

    camps = []
    for _ in range(NUM_CAMPAIGNS):
        status = random.choice(CAMPAIGN_STATUSES)
        name = sentence_name(4)
        camps.append((str(uuid.uuid4()), name, name, "noreply@example.com",
                      f"<p>{sentence_name(6)}</p>", "richtext", status,
                      now + timedelta(days=3650) if status == "scheduled" else None,
                      "email", template_id, rand_tags()))
    camp_ids = [r[0] for r in insert(cur, """
        INSERT INTO campaigns (uuid, name, subject, from_email, body, content_type, status,
            send_at, messenger, template_id, tags) VALUES {} RETURNING id
    """, camps, fetch=True)]
    print(f"{len(camp_ids)} campaigns")

    camp_lists = []
    for cid in camp_ids:
        for l in random.sample(list_ids, random.randint(1, MAX_CAMPAIGN_LISTS)):
            camp_lists.append((cid, l[0], l[2]))
    insert(cur, """
        INSERT INTO campaign_lists (campaign_id, list_id, list_name) VALUES {}
    """, camp_lists)

    links = [(str(uuid.uuid4()), f"https://example.com/{sentence_name(2).lower().replace(' ', '-')}-{i}")
             for i in range(NUM_LINKS)]
    link_ids = [r[0] for r in insert(cur, """
        INSERT INTO links (uuid, url) VALUES {} RETURNING id
    """, links, fetch=True)]

    click_camps = random.sample(camp_ids, min(SAMPLE_CAMPAIGNS, len(camp_ids)))
    click_links = random.sample(link_ids, min(SAMPLE_LINKS, len(link_ids)))

    clicks = [(random.choice(click_camps), random.choice(click_links),
               random.choice(sub_ids), rand_time(now)) for _ in range(NUM_CLICKS)]
    insert(cur, """
        INSERT INTO link_clicks (campaign_id, link_id, subscriber_id, created_at) VALUES {}
    """, clicks)
    print(f"{len(clicks)} link clicks")

    views = [(random.choice(click_camps), random.choice(sub_ids), rand_time(now))
             for _ in range(NUM_VIEWS)]
    insert(cur, """
        INSERT INTO campaign_views (campaign_id, subscriber_id, created_at) VALUES {}
    """, views)
    print(f"{len(views)} campaign views")

    db.commit()
    cur.close()
    db.close()


if __name__ == "__main__":
    main()
