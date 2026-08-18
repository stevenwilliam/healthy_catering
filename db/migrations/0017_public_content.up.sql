-- Editable public copy, in three languages, with Indonesian as the source.
--
-- Steven, 2026-08-18: the hero wording must be maintainable from the back
-- office, written in Indonesian, machine-translated to English and Chinese,
-- and overridable per language when the machine gets it wrong.
--
-- Why a table rather than more sys_parameters rows: this has state that a
-- key/value pair cannot carry — which locale is the SOURCE, whether a
-- translation was written by a human (and must therefore never be overwritten
-- by the translator), and whether it is STALE because the Indonesian text has
-- moved on since it was translated. The database enforces all three rather
-- than trusting the application (CLAUDE.md §4).
CREATE TABLE public_content (
  key         TEXT NOT NULL,
  locale      TEXT NOT NULL,
  value       TEXT NOT NULL DEFAULT '',
  -- TRUE = a human wrote this and the translator must leave it alone.
  is_override BOOLEAN NOT NULL DEFAULT FALSE,
  -- sha256 of the Indonesian text this translation was made from. When it no
  -- longer matches the current source, the translation is stale — which is
  -- what lets the admin screen show "the Indonesian changed since this was
  -- translated" instead of silently serving old copy.
  source_hash TEXT NOT NULL DEFAULT '',
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by  UUID REFERENCES app_user(id),

  PRIMARY KEY (key, locale),
  CONSTRAINT public_content_locale_ck CHECK (locale IN ('id', 'en', 'zh')),
  -- The source language is nobody's translation: it cannot be an override of
  -- itself and has no source to be stale against.
  CONSTRAINT public_content_source_ck
    CHECK (locale <> 'id' OR (is_override = FALSE AND source_hash = ''))
);

COMMENT ON TABLE public_content IS
  'Editable public copy. locale=id is the source; en and zh are derived by the translator unless is_override.';

CREATE TRIGGER public_content_touch BEFORE UPDATE ON public_content
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Seed with exactly what the catalogue renders today, so switching the page
-- over to this table is invisible until someone edits something. The English
-- and Chinese rows are marked as overrides: they are hand-written copy, not
-- machine output, and the translator must not overwrite them on the first
-- edit of the Indonesian.
INSERT INTO public_content (key, locale, value, is_override, source_hash) VALUES
 ('home.eyebrow', 'id', 'Katering sehat harian · Jakarta', FALSE, ''),
 ('home.h1',      'id', 'Makan sehat, setiap hari, tanpa repot.', FALSE, ''),
 ('home.lede',    'id', 'Makanan sehat harian diantar ke rumah atau kantor Anda di Jakarta. Pilih menu sesuai kebutuhan: Healthy, Weight Loss, High Protein dan lainnya.', FALSE, ''),
 ('home.cta',     'id', 'Lihat menu minggu ini', FALSE, '');

INSERT INTO public_content (key, locale, value, is_override, source_hash)
SELECT k.key, l.locale, l.value, TRUE,
       encode(sha256(convert_to(src.value, 'UTF8')), 'hex')
  FROM (VALUES
    ('home.eyebrow'), ('home.h1'), ('home.lede'), ('home.cta')
  ) AS k(key)
  JOIN LATERAL (VALUES
    ('home.eyebrow', 'en', 'Daily healthy catering · Jakarta'),
    ('home.eyebrow', 'zh', '每日健康餐 · 雅加达'),
    ('home.h1', 'en', 'Eat well, every day, without the hassle.'),
    ('home.h1', 'zh', '健康饮食，天天如此，轻松无忧。'),
    ('home.lede', 'en', 'Healthy daily meals delivered to your home or office in Jakarta. Choose the menu that fits: Healthy, Weight Loss, High Protein and more.'),
    ('home.lede', 'zh', '健康的每日餐点，配送到您在雅加达的家或办公室。按需选择菜单：Healthy、Weight Loss、High Protein 等。'),
    ('home.cta', 'en', 'See this week''s menu'),
    ('home.cta', 'zh', '查看本周菜单')
  ) AS l(key, locale, value) ON l.key = k.key
  JOIN public_content src ON src.key = k.key AND src.locale = 'id';
