DELETE FROM public_content WHERE key IN ('benefit.title', 'benefit.body');
ALTER TABLE public_content DROP COLUMN IF EXISTS is_html;
