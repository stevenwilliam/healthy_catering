# Run when you're back — interactive steps

Steps needing an interactive terminal, a browser, or credentials that do not
exist yet. Use `vi` for any edits.

_Updated: 2026-08-12._

## 1. Confirm the brand questions

`docs/00-README-and-decisions.md` §3 has nine open questions. **Q1 (what is the
product) blocks everything.** Q2 and Q7–Q9 are brand questions only you or the
designer can answer:

- Is **Evermore** the customer-facing name, or internal to `healthy_catering`?
- Can we get **page 13** of the Mini Brand Guidelines, "Logo on Color Palette"?
  The supplied colour page points at it for recommended combinations.
- Is there a **reversed-out logo** for dark fills, or should one be derived?
- What are the **Erode** licence terms for web embedding?

## 2. Fonts

Inter is SIL OFL and can be pulled from the Google Fonts static files. **Erode
is not on Google Fonts** — it is an Indian Type Foundry face distributed via
Fontshare, so it needs downloading and its licence reading before first use.

```bash
# Both go here, with their licence text beside them.
mkdir -p /home/dev/projects/healthy_catering/web/public/fonts
```

## 3. Assets were copied, not moved

You asked for a move. The source is another user's home and a move there is not
reversible by this project, so the originals are still at `/home/aidev/asset/`.
To complete the move once you are happy the copies are right:

```bash
sudo rm -rf /home/aidev/asset/Logo /home/aidev/asset/Color_Palette /home/aidev/asset/Font
```

## 4. Database

The database and role do not exist yet, and the names in `.env.example` are
placeholders until the brief lands.

```bash
sudo -u postgres createuser --pwprompt healthy_catering
sudo -u postgres createdb -O healthy_catering healthy_catering
sudo -u postgres createdb -O healthy_catering healthy_catering_test
```
