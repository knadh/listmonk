import os
import json
from glob import glob
from openai import OpenAI

client = OpenAI(
    # This is the default and can be omitted
    api_key=os.environ.get("OPENAI_API_KEY")
)

# Keys to translate. If this is empty, all keys are translated.
KEYS = []

DEFAULT_LANG = "en.json"
DIR = os.path.normpath(os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "../i18n"))
BASE = json.loads(open(os.path.join(DIR, DEFAULT_LANG), "r").read())


def translate(data, lang):
    completion = client.chat.completions.create(
        model="gpt-4.1-mini",
        messages=[
            {"role": "system", "content": "You are an i18n language pack translator for listmonk, a mailing list manager. Remember that context when translating."},
            {"role": "user",
                "content": "Translate the untranslated English strings in the following JSON language map to {}. Retain any technical terms or acronyms.".format(lang)},
            {"role": "user", "content": json.dumps(data)}
            # {"role": "user", "content": "Hello world good morning!"}
        ]
    )

    return json.loads(str(completion.choices[0].message.content))


# Go through every i18n file.
for f in glob(os.path.join(DIR, "*.json")):
    if os.path.basename(f) == DEFAULT_LANG:
        continue

    print(os.path.basename(f))

    data = json.loads(open(f, "r").read())

    # Diff the entire file or only given keys.
    if KEYS:
        diff = {k: BASE[k] for k in KEYS}
    else:
        diff = {k: v for k, v in data.items() if BASE.get(k) == v}

    new = translate(diff, data["_.name"])
    data.update(new)

    with open(f, "w") as o:
        o.write(json.dumps(data, sort_keys=True,
                indent=4, ensure_ascii=False) + "\n")
