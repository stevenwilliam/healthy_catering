-- The Benefit section, and rich text for the copy that needs it.
--
-- Steven, 2026-08-18: a Benefit section after the price list, editable from the
-- back office with a WYSIWYG editor, with initial wording.
--
-- is_html marks the rows whose value is markup rather than a sentence. It is a
-- column rather than a naming convention because it decides two things that
-- must not be guessed: whether the back office shows a rich-text editor, and
-- whether the template renders the value unescaped. A key that is HTML in one
-- place and plain in another is how a stored <script> reaches a page.
ALTER TABLE public_content
  ADD COLUMN is_html BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN public_content.is_html IS
  'Value is sanitised HTML from the WYSIWYG editor. Sanitised on write AND on render (internal/platform/richtext).';

-- Heading: plain text.
INSERT INTO public_content (key, locale, value, is_override, source_hash, is_html) VALUES
 ('benefit.title', 'id', 'Kenapa Evermore', FALSE, '', FALSE)
ON CONFLICT (key, locale) DO NOTHING;

-- Body: HTML.
INSERT INTO public_content (key, locale, value, is_override, source_hash, is_html) VALUES
 ('benefit.body', 'id', '<ul><li><strong>Menu disusun bersama ahli gizi</strong>, bukan sekadar makanan rendah kalori.</li><li><strong>Dimasak pagi hari, diantar hari itu juga</strong> dari dapur terdekat dengan Anda.</li><li><strong>Kalori dan protein tercantum</strong> di setiap porsi, jadi Anda tahu persis apa yang dimakan.</li><li><strong>Pilih kategori sesuai tujuan</strong> — menjaga berat, menambah massa, atau tinggi protein.</li><li><strong>Tanpa kontrak panjang.</strong> Ambil paket harian, dan pesan sesuai kebutuhan.</li></ul>', FALSE, '', TRUE)
ON CONFLICT (key, locale) DO NOTHING;

-- English and Chinese seeded as hand-written overrides, so the translator does
-- not overwrite them the first time the Indonesian is edited.
INSERT INTO public_content (key, locale, value, is_override, source_hash, is_html)
SELECT v.key, v.locale, v.value, TRUE,
       encode(sha256(convert_to(src.value, 'UTF8')), 'hex'), v.is_html
  FROM (VALUES
    ('benefit.title', 'en', 'Why Evermore', FALSE),
    ('benefit.title', 'zh', '为什么选择 Evermore', FALSE),
    ('benefit.body',  'en', '<ul><li><strong>Menus planned with nutritionists</strong>, not just low-calorie food.</li><li><strong>Cooked in the morning, delivered the same day</strong> from the kitchen nearest you.</li><li><strong>Calories and protein on every portion</strong>, so you know exactly what you are eating.</li><li><strong>Pick the category that fits your goal</strong> — maintain, gain, or high protein.</li><li><strong>No long contract.</strong> Take a daily package and order as you need it.</li></ul>', TRUE),
    ('benefit.body',  'zh', '<ul><li><strong>菜单由营养师共同制定</strong>，而不只是低热量餐。</li><li><strong>清晨烹制，当天送达</strong>，由离您最近的厨房配送。</li><li><strong>每份标明热量与蛋白质</strong>，您清楚知道自己吃了什么。</li><li><strong>按目标选择菜单类别</strong>——保持体重、增重或高蛋白。</li><li><strong>无需长期合约。</strong>选择按天套餐，按需订购。</li></ul>', TRUE)
  ) AS v(key, locale, value, is_html)
  JOIN public_content src ON src.key = v.key AND src.locale = 'id'
ON CONFLICT (key, locale) DO NOTHING;
